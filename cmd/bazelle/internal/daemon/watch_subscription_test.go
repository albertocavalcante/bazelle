package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/watch"
)

// fakeWatcher stands in for watch.Watcher. Run blocks until release
// receives a value (the error to return); it deliberately ignores ctx so
// tests control exactly when a watcher goroutine finishes.
type fakeWatcher struct {
	cfg     watch.Config
	release chan error
	closed  atomic.Bool
}

func (f *fakeWatcher) Run(context.Context) error { return <-f.release }

func (f *fakeWatcher) Close() error {
	f.closed.Store(true)
	return nil
}

// newFakeHandler returns a handler whose watchers are fakes, published on
// the returned channel in creation order.
func newFakeHandler(server *Server) (*Handler, <-chan *fakeWatcher) {
	created := make(chan *fakeWatcher, 8)
	h := NewHandler(server)
	h.newWatcher = func(cfg watch.Config) (fileWatcher, error) {
		f := &fakeWatcher{cfg: cfg, release: make(chan error, 1)}
		created <- f
		return f, nil
	}
	return h, created
}

// pipePeer is a server-side ClientConn whose outgoing messages are decoded
// on the other end of an in-memory pipe.
type pipePeer struct {
	conn *ClientConn
	msgs chan Notification
}

func newPipePeer(t *testing.T, server *Server) *pipePeer {
	t.Helper()
	srv, cli := net.Pipe()
	p := &pipePeer{
		conn: &ClientConn{conn: srv, server: server, encoder: json.NewEncoder(srv)},
		msgs: make(chan Notification, 32),
	}
	go func() {
		dec := json.NewDecoder(cli)
		for {
			var n Notification
			if err := dec.Decode(&n); err != nil {
				return
			}
			if n.Method != "" { // skip responses
				p.msgs <- n
			}
		}
	}()
	server.clientsMu.Lock()
	server.clients[p.conn] = struct{}{}
	server.clientsMu.Unlock()
	t.Cleanup(func() {
		p.conn.Close()
		_ = cli.Close()
	})
	return p
}

func (p *pipePeer) nextEvent(t *testing.T) WatchEventParams {
	t.Helper()
	select {
	case n := <-p.msgs:
		if n.Method != MethodWatchEvent {
			t.Fatalf("method = %q, want %q", n.Method, MethodWatchEvent)
		}
		var ev WatchEventParams
		if err := json.Unmarshal(n.Params, &ev); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch/event")
		return WatchEventParams{}
	}
}

func (p *pipePeer) expectNoEvent(t *testing.T) {
	t.Helper()
	select {
	case n := <-p.msgs:
		t.Fatalf("unexpected notification: %s %s", n.Method, n.Params)
	case <-time.After(100 * time.Millisecond):
	}
}

func newTestServer() *Server {
	return &Server{clients: make(map[*ClientConn]struct{}), startTime: time.Now()}
}

func call(t *testing.T, h *Handler, c *ClientConn, method string) *Response {
	t.Helper()
	resp := h.HandleRequest(c, &Request{JSONRPC: JSONRPCVersion, ID: ptr(int64(1)), Method: method})
	if resp.Error != nil {
		t.Fatalf("%s: unexpected error %v", method, resp.Error)
	}
	return resp
}

func statusOf(t *testing.T, resp *Response) string {
	t.Helper()
	var r struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp.Result, &r); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return r.Status
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Bug 1: watcher events must reach subscribed clients.
func TestHandler_WatcherEventsAreBroadcast(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	h, created := newFakeHandler(server)
	peer := newPipePeer(t, server)

	call(t, h, peer.conn, MethodWatchStart)
	w := <-created
	t.Cleanup(func() { w.release <- nil })

	w.cfg.OnChange([]string{"pkg"})
	if ev := peer.nextEvent(t); ev.Type != "change" || len(ev.Directories) != 1 || ev.Directories[0] != "pkg" {
		t.Errorf("change event = %+v", ev)
	}

	w.cfg.OnUpdate([]string{"pkg"}, []string{"pkg/BUILD.bazel"})
	if ev := peer.nextEvent(t); ev.Type != "update" || len(ev.Files) != 1 || ev.Files[0] != "pkg/BUILD.bazel" {
		t.Errorf("update event = %+v", ev)
	}
	if h.GetWatchStatus().UpdateTime == "" {
		t.Error("update_time not recorded after an update")
	}

	w.cfg.OnError(errors.New("gazelle failed"))
	if ev := peer.nextEvent(t); ev.Type != "error" || ev.Message != "gazelle failed" {
		t.Errorf("error event = %+v", ev)
	}
}

// Bug 2: a watcher that dies on its own must tell its subscribers.
func TestHandler_WatcherFailureNotifiesSubscribers(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	h, created := newFakeHandler(server)
	peer := newPipePeer(t, server)

	call(t, h, peer.conn, MethodWatchStart)
	w := <-created
	w.release <- errors.New("inotify limit")

	if ev := peer.nextEvent(t); ev.Type != "error" || !strings.Contains(ev.Message, "inotify limit") {
		t.Errorf("first event = %+v, want error carrying the cause", ev)
	}
	if ev := peer.nextEvent(t); ev.Type != "stopped" {
		t.Errorf("second event = %+v, want stopped", ev)
	}
	if st := h.GetWatchStatus(); st.Watching || len(st.Paths) != 0 {
		t.Errorf("status after failure = %+v, want idle with no paths", st)
	}

	// The failed session must not keep the client subscribed.
	h.BroadcastEvent("change", nil, nil, "")
	peer.expectNoEvent(t)
}

// Bug 3: the watcher is shared and refcounted per connection.
func TestHandler_WatchRefcountedPerConnection(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	h, created := newFakeHandler(server)
	a := newPipePeer(t, server)
	b := newPipePeer(t, server)

	if got := statusOf(t, call(t, h, a.conn, MethodWatchStart)); got != "watching" {
		t.Fatalf("a start = %q", got)
	}
	w := <-created
	t.Cleanup(func() { w.release <- nil })
	if got := statusOf(t, call(t, h, b.conn, MethodWatchStart)); got != "already_watching" {
		t.Fatalf("b start = %q", got)
	}

	// a stops: b still needs the watcher, and only b gets events.
	if got := statusOf(t, call(t, h, a.conn, MethodWatchStop)); got != "unsubscribed" {
		t.Errorf("a stop = %q, want unsubscribed", got)
	}
	if !h.GetWatchStatus().Watching || w.closed.Load() {
		t.Fatal("watcher stopped while b is still subscribed")
	}
	w.cfg.OnChange([]string{"x"})
	if ev := b.nextEvent(t); ev.Type != "change" {
		t.Errorf("b event = %+v", ev)
	}
	a.expectNoEvent(t)

	// A connection that never subscribed cannot stop someone else's watch.
	c := newPipePeer(t, server)
	if got := statusOf(t, call(t, h, c.conn, MethodWatchStop)); got != "not_subscribed" {
		t.Errorf("c stop = %q, want not_subscribed", got)
	}

	// b disconnects: last subscriber gone, the watcher must stop.
	h.ClientDisconnected(b.conn)
	if h.GetWatchStatus().Watching || !w.closed.Load() {
		t.Error("watcher still running after the last subscriber disconnected")
	}
}

// Bug 3 (server wiring): dropping the socket releases the subscription.
func TestServer_DisconnectStopsWatch(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDirServer(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}
	h, created := newFakeHandler(nil)
	server := NewServer(ServerConfig{Paths: paths, Version: "test", Handler: h})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- server.Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-errCh; err != nil {
			t.Errorf("server.Start: %v", err)
		}
	})

	var client *Client
	waitFor(t, "daemon socket", func() bool {
		var err error
		client, err = Connect(paths.Socket)
		return err == nil
	})
	if client == nil {
		t.Fatal("Connect succeeded without returning a client")
	}
	if _, err := client.WatchStart(&WatchStartParams{}); err != nil {
		t.Fatalf("WatchStart: %v", err)
	}
	w := <-created
	t.Cleanup(func() { w.release <- nil })

	_ = client.Close()
	waitFor(t, "watch to stop after disconnect", func() bool {
		return !h.GetWatchStatus().Watching && w.closed.Load()
	})
}

// Bug 4: a stale watcher goroutine must not tear down its successor.
func TestHandler_StaleWatcherDoesNotOrphanSuccessor(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	h, created := newFakeHandler(server)
	peer := newPipePeer(t, server)

	call(t, h, peer.conn, MethodWatchStart)
	old := <-created
	call(t, h, peer.conn, MethodWatchStop)
	call(t, h, peer.conn, MethodWatchStart)
	current := <-created
	t.Cleanup(func() { current.release <- nil })

	// The old Run returns only now, after the restart.
	old.release <- errors.New("late failure")
	time.Sleep(100 * time.Millisecond)

	if !h.GetWatchStatus().Watching {
		t.Fatal("old watcher goroutine cleared the new watcher's state")
	}
	peer.expectNoEvent(t)
	old.cfg.OnChange([]string{"stale"})
	peer.expectNoEvent(t)

	if got := statusOf(t, call(t, h, peer.conn, MethodWatchStop)); got != "stopped" {
		t.Errorf("stop = %q", got)
	}
	if !current.closed.Load() {
		t.Error("current watcher was orphaned: stop did not close it")
	}
}

// A peer that stops reading must not block broadcasts forever.
func TestClientConn_SendTimesOutOnStuckPeer(t *testing.T) {
	t.Parallel()
	srv, cli := net.Pipe() // nobody reads cli
	t.Cleanup(func() { _ = cli.Close() })
	cc := &ClientConn{
		conn:    srv,
		server:  &Server{writeTimeout: 50 * time.Millisecond},
		encoder: json.NewEncoder(srv),
	}

	done := make(chan error, 1)
	go func() { done <- cc.Send(map[string]string{"x": "y"}) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Send to a stuck peer succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send blocked on a peer that never reads")
	}
	if err := cc.Send("again"); !errors.Is(err, net.ErrClosed) {
		t.Errorf("connection not closed after a failed write: %v", err)
	}
}

// The watch/start reply must reach the client before any event from the
// session it started, even when the watcher fails at once.
func TestServer_StartReplyPrecedesWatcherEvents(t *testing.T) {
	t.Parallel()
	for i := 0; i < 50; i++ {
		server := newTestServer()
		h := NewHandler(server)
		h.newWatcher = func(cfg watch.Config) (fileWatcher, error) {
			f := &fakeWatcher{cfg: cfg, release: make(chan error, 1)}
			f.release <- errors.New("fails immediately")
			return f, nil
		}
		server.handler = h

		srv, cli := net.Pipe()
		cc := &ClientConn{conn: srv, server: server, encoder: json.NewEncoder(srv), decoder: json.NewDecoder(srv)}
		server.clients[cc] = struct{}{}
		go server.handleClient(cc)

		if err := json.NewEncoder(cli).Encode(Request{JSONRPC: JSONRPCVersion, ID: ptr(int64(7)), Method: MethodWatchStart}); err != nil {
			t.Fatalf("send request: %v", err)
		}
		_ = cli.SetReadDeadline(time.Now().Add(2 * time.Second))
		dec := json.NewDecoder(cli)
		var first struct {
			ID     *int64 `json:"id"`
			Method string `json:"method"`
		}
		if err := dec.Decode(&first); err != nil {
			t.Fatalf("read first message: %v", err)
		}
		if first.ID == nil || *first.ID != 7 {
			t.Fatalf("iteration %d: first message is %q, want the watch/start reply", i, first.Method)
		}
		_ = cli.Close()
	}
}

// json.Decoder errors are sticky: after a malformed line the server must
// reply once with a parse error and drop the connection, not loop forever.
func TestServer_MalformedRequestClosesConnection(t *testing.T) {
	t.Parallel()
	server := newTestServer()
	server.writeTimeout = 200 * time.Millisecond
	h := NewHandler(server)
	server.handler = h

	srv, cli := net.Pipe()
	t.Cleanup(func() { _ = cli.Close() })
	cc := &ClientConn{conn: srv, server: server, encoder: json.NewEncoder(srv), decoder: json.NewDecoder(srv)}
	server.clients[cc] = struct{}{}
	h.subscribeLocked(cc)
	h.watching = true // pretend cc holds a watch subscription
	done := make(chan struct{})
	go func() { server.handleClient(cc); close(done) }()

	go func() { _, _ = cli.Write([]byte("{not json\n")) }()
	_ = cli.SetReadDeadline(time.Now().Add(2 * time.Second))
	dec := json.NewDecoder(cli)
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("read parse error reply: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != ErrCodeParseError || resp.ID != nil {
		t.Fatalf("first reply = %+v, want a parse error with id null", resp)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err == nil {
		t.Fatalf("server kept talking after a parse error: %s", extra)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleClient still running after a malformed request")
	}
	h.watchMu.RLock()
	defer h.watchMu.RUnlock()
	if len(h.subscribers) != 0 || h.watching {
		t.Error("disconnect cleanup did not release the subscription")
	}
}
