package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/watch"
	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/bazelbuild/bazel-gazelle/language"
)

// Handler handles RPC method calls.
type Handler struct {
	server    *Server
	languages []language.Language
	defaults  []string // gazelle defaults

	// newWatcher creates file watchers; nil means watch.New (tests replace it).
	newWatcher func(watch.Config) (fileWatcher, error)

	// Watch state. One watcher is shared by all subscribed connections and
	// stops when the last one unsubscribes or disconnects.
	watchMu     sync.RWMutex
	watcher     fileWatcher
	watchCancel context.CancelFunc
	watchGen    uint64 // identifies the current watch session
	watchPaths  []string
	watchLangs  []string
	subscribers map[*ClientConn]struct{}
	lastUpdate  time.Time
	watching    bool
}

// fileWatcher is the subset of *watch.Watcher the handler uses.
type fileWatcher interface {
	Run(ctx context.Context) error
	Close() error
}

// HandlerConfig configures the RPC handler.
type HandlerConfig struct {
	Languages       []language.Language
	GazelleDefaults []string
}

// NewHandler creates a new RPC handler.
func NewHandler(server *Server) *Handler {
	return &Handler{
		server: server,
	}
}

// NewHandlerWithConfig creates a new RPC handler with configuration.
func NewHandlerWithConfig(server *Server, cfg HandlerConfig) *Handler {
	return &Handler{
		server:    server,
		languages: cfg.Languages,
		defaults:  cfg.GazelleDefaults,
	}
}

// SetLanguages sets the language extensions to use.
func (h *Handler) SetLanguages(langs []language.Language) {
	h.languages = langs
}

// SetDefaults sets the gazelle defaults.
func (h *Handler) SetDefaults(defaults []string) {
	h.defaults = defaults
}

// HandleRequest dispatches a request to the appropriate handler.
func (h *Handler) HandleRequest(client *ClientConn, req *Request) *Response {
	logger := log.Component("daemon")
	logger.Debugw("handling request", "method", req.Method, "id", req.ID)

	if req.ID == nil {
		// JSON-RPC notification: process it, but never reply.
		withID := *req
		withID.ID = new(int64)
		h.dispatch(client, &withID)
		return nil
	}
	return h.dispatch(client, req)
}

// dispatch routes a request (with a non-nil ID) to its method handler.
func (h *Handler) dispatch(client *ClientConn, req *Request) *Response {
	switch req.Method {
	case MethodPing:
		return h.handlePing(req)
	case MethodShutdown:
		return h.handleShutdown(req)
	case MethodWatchStart:
		return h.handleWatchStart(client, req)
	case MethodWatchStop:
		return h.handleWatchStop(client, req)
	case MethodWatchStatus:
		return h.handleWatchStatus(req)
	case MethodUpdateRun:
		return h.handleUpdateRun(req)
	case MethodStatusGet:
		return h.handleStatusGet(req)
	default:
		return NewErrorResponse(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// handlePing handles the ping request.
func (h *Handler) handlePing(req *Request) *Response {
	result := PingResult{
		Pong:      true,
		Version:   h.server.version,
		Uptime:    h.server.Uptime().String(),
		StartTime: h.server.startTime.Format(time.RFC3339),
	}

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
}

// handleShutdown handles the shutdown request.
func (h *Handler) handleShutdown(req *Request) *Response {
	// Send response first, then shutdown
	result := ShutdownResult{
		Message: "daemon shutting down",
	}

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}

	// Schedule shutdown after response is sent
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time to send response
		h.server.RequestShutdown()
	}()

	return resp
}

// handleWatchStart handles the watch/start request.
func (h *Handler) handleWatchStart(client *ClientConn, req *Request) *Response {
	var params WatchStartParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return NewErrorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
		}
	}

	h.watchMu.Lock()
	defer h.watchMu.Unlock()

	// If already watching, share the existing watcher
	if h.watching {
		h.subscribeLocked(client)
		result := WatchStartResult{
			Status:    "already_watching",
			Paths:     h.watchPaths,
			Languages: h.watchLangs,
		}
		resp, _ := NewResponse(*req.ID, result)
		return resp
	}

	// Use provided paths or default to current directory
	paths := params.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// For now, we only support watching a single path (the first one)
	watchPath := paths[0]

	// Create watcher config
	debounce := params.Debounce
	if debounce <= 0 {
		debounce = 500
	}

	gen := h.watchGen + 1
	cfg := watch.Config{
		Root:            watchPath,
		Languages:       h.languages,
		LangFilter:      params.Languages,
		Debounce:        debounce,
		Verbose:         false,
		NoColor:         true,
		JSON:            false,
		GazelleDefaults: h.defaults,
		OnChange: func(dirs []string) {
			h.emit(gen, "change", dirs, nil, "")
		},
		OnUpdate: func(dirs, buildFiles []string) {
			h.emit(gen, "update", dirs, buildFiles, "BUILD files updated")
		},
		OnError: func(err error) {
			h.emit(gen, "error", nil, nil, err.Error())
		},
	}

	newWatcher := h.newWatcher
	if newWatcher == nil {
		newWatcher = func(cfg watch.Config) (fileWatcher, error) { return watch.New(cfg) }
	}
	watcher, err := newWatcher(cfg)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to start watcher", err.Error())
	}

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	h.watchGen = gen
	h.watcher = watcher
	h.watchCancel = cancel
	h.watchPaths = paths
	h.watchLangs = params.Languages
	h.watching = true
	h.subscribeLocked(client)

	go h.runWatcher(ctx, watcher, gen)

	result := WatchStartResult{
		Status:    "watching",
		Paths:     paths,
		Languages: params.Languages,
	}

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
}

// runWatcher runs the watcher and, if it ends while still the current
// session (nobody stopped it), tells the subscribers it is gone.
func (h *Handler) runWatcher(ctx context.Context, w fileWatcher, gen uint64) {
	logger := log.Component("daemon")

	// Run the watcher (this blocks)
	runErr := w.Run(ctx)

	h.watchMu.Lock()
	if !h.isCurrentLocked(gen) {
		// Stopped on purpose, possibly already replaced by a newer watcher.
		h.watchMu.Unlock()
		logger.Infow("watcher stopped")
		return
	}
	subs := h.subscribers
	h.stopLocked()
	h.watchMu.Unlock()

	var events []WatchEventParams
	if runErr != nil {
		logger.Warnw("watcher stopped with error", "error", runErr)
		events = append(events, WatchEventParams{Type: "error", Message: fmt.Sprintf("watcher stopped: %v", runErr)})
	} else {
		logger.Warnw("watcher stopped unexpectedly")
	}
	events = append(events, WatchEventParams{Type: "stopped", Message: "watcher stopped; call watch/start to resume"})
	h.notify(subs, events)
}

// handleWatchStop releases the caller's watch subscription. The shared
// watcher stops only when no subscribers remain.
func (h *Handler) handleWatchStop(client *ClientConn, req *Request) *Response {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()

	status := "not_watching"
	if h.watching {
		status = "not_subscribed"
		if h.unsubscribeLocked(client) {
			status = "stopped"
			if h.watching {
				status = "unsubscribed" // other connections still use the watcher
			}
		}
	}

	resp, err := NewResponse(*req.ID, WatchStopResult{Status: status})
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
}

// ClientDisconnected releases any watch subscription held by client.
func (h *Handler) ClientDisconnected(client *ClientConn) {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()
	h.unsubscribeLocked(client)
}

// subscribeLocked adds client to the current watch session.
func (h *Handler) subscribeLocked(client *ClientConn) {
	if h.subscribers == nil {
		h.subscribers = make(map[*ClientConn]struct{})
	}
	h.subscribers[client] = struct{}{}
	client.Subscribe()
}

// unsubscribeLocked removes client from the watch session, stopping the
// watcher if it was the last subscriber. It reports whether client was
// subscribed.
func (h *Handler) unsubscribeLocked(client *ClientConn) bool {
	if _, ok := h.subscribers[client]; !ok {
		return false
	}
	delete(h.subscribers, client)
	client.Unsubscribe()
	if len(h.subscribers) == 0 && h.watching {
		h.stopLocked()
	}
	return true
}

// stopLocked stops the current watcher and drops all subscriptions.
func (h *Handler) stopLocked() {
	if h.watchCancel != nil {
		h.watchCancel()
	}
	if h.watcher != nil {
		_ = h.watcher.Close()
	}
	for c := range h.subscribers {
		c.Unsubscribe()
	}

	h.watching = false
	h.watcher = nil
	h.watchCancel = nil
	h.watchPaths = nil
	h.watchLangs = nil
	h.subscribers = nil
}

// isCurrentLocked reports whether gen is the running watch session.
func (h *Handler) isCurrentLocked(gen uint64) bool {
	return h.watching && h.watchGen == gen
}

// emit broadcasts a watcher event unless its session has been superseded.
func (h *Handler) emit(gen uint64, eventType string, dirs, files []string, message string) {
	h.watchMu.Lock()
	current := h.isCurrentLocked(gen)
	if current && eventType == "update" {
		h.lastUpdate = time.Now()
	}
	h.watchMu.Unlock()

	if current {
		h.BroadcastEvent(eventType, dirs, files, message)
	}
}

// notify sends events directly to subs (already unsubscribed). Each client
// gets all of them back to back, so a request it sends in reaction (handled
// under the same encoder lock) cannot slip in between.
func (h *Handler) notify(subs map[*ClientConn]struct{}, events []WatchEventParams) {
	var notifs []*Notification
	for _, ev := range events {
		ev.Timestamp = time.Now().Format(time.RFC3339)
		if n, err := NewNotification(MethodWatchEvent, ev); err == nil {
			notifs = append(notifs, n)
		}
	}
	for c := range subs {
		c.encoderMu.Lock()
		for _, n := range notifs {
			if c.sendLocked(n) != nil {
				break
			}
		}
		c.encoderMu.Unlock()
	}
}

// handleWatchStatus handles the watch/status request.
func (h *Handler) handleWatchStatus(req *Request) *Response {
	result := h.GetWatchStatus()

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
}

// GetWatchStatus returns the current watch status.
func (h *Handler) GetWatchStatus() *WatchStatusResult {
	h.watchMu.RLock()
	defer h.watchMu.RUnlock()

	result := &WatchStatusResult{
		Watching:  h.watching,
		Paths:     h.watchPaths,
		Languages: h.watchLangs,
	}

	if !h.lastUpdate.IsZero() {
		result.UpdateTime = h.lastUpdate.Format(time.RFC3339)
	}

	return result
}

// handleUpdateRun handles the update/run request.
func (h *Handler) handleUpdateRun(req *Request) *Response {
	// TODO: Implement manual update trigger (parse UpdateRunParams then)
	return notImplemented(req)
}

// handleStatusGet handles the status/get request.
func (h *Handler) handleStatusGet(req *Request) *Response {
	// TODO: Implement status/get using incremental tracker
	return notImplemented(req)
}

// notImplemented answers a known but unimplemented method. JSON-RPC has no
// dedicated code, so it uses Method not found rather than a fake result.
func notImplemented(req *Request) *Response {
	return NewErrorResponse(req.ID, ErrCodeMethodNotFound, "Method not implemented: "+req.Method, nil)
}

// Stop stops the handler and any running watcher.
func (h *Handler) Stop() {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()
	h.stopLocked()
}

// BroadcastEvent broadcasts a watch event to all subscribed clients.
func (h *Handler) BroadcastEvent(eventType string, dirs []string, files []string, message string) {
	if h.server == nil {
		return
	}

	notif, err := NewNotification(MethodWatchEvent, WatchEventParams{
		Type:        eventType,
		Directories: dirs,
		Files:       files,
		Message:     message,
		Timestamp:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	h.server.Broadcast(notif)
}
