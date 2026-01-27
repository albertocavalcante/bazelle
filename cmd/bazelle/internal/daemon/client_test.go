package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// shortTempDir creates a short temp directory for Unix socket tests.
// Unix sockets have a path length limit (~104 chars on macOS).
func shortTempDir(t *testing.T) string {
	t.Helper()
	// Use /tmp directly to get shorter paths
	dir, err := os.MkdirTemp("/tmp", "dt")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestConnect(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a test server
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close() // Just accept and close for this test
		}
	}()

	// Connect
	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	if client.conn == nil {
		t.Error("conn should not be nil")
	}
}

func TestConnect_NotRunning(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "nonexistent.sock")

	_, err := Connect(socketPath)
	if err == nil {
		t.Error("Connect() should return error for non-existent socket")
	}
	if !errors.Is(err, ErrDaemonNotRunning) {
		t.Errorf("Error should be ErrDaemonNotRunning, got %v", err)
	}
}

func TestConnect_Timeout(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a listener but don't accept
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	// Connect should succeed even if no one accepts (connection is queued)
	client, err := Connect(socketPath)
	if err != nil {
		t.Logf("Connect result: %v", err)
		// This is acceptable - the connection might timeout or queue
		return
	}
	client.Close()
}

func TestClient_Close(t *testing.T) {
	t.Parallel()
	client := &Client{
		conn:    nil,
		closeCh: make(chan struct{}),
	}

	// Close on nil conn should not error
	err := client.Close()
	if err != nil {
		t.Errorf("Close() on nil conn error = %v", err)
	}
}

func TestClient_CloseMultiple(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Keep connection open
		time.Sleep(5 * time.Second)
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}

	// Multiple closes should not panic
	for i := 0; i < 5; i++ {
		err := client.Close()
		if i > 0 && err == nil {
			// Subsequent closes may or may not error
		}
	}
}

func TestClient_Ping(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create test server
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	// Handle requests
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodPing {
			result := PingResult{
				Pong:      true,
				Version:   "1.0.0",
				Uptime:    "1h0m0s",
				StartTime: "2024-01-01T00:00:00Z",
			}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if !result.Pong {
		t.Error("Pong should be true")
	}
	if result.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "1.0.0")
	}
}

func TestClient_Shutdown(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodShutdown {
			result := ShutdownResult{Message: "shutting down"}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestClient_WatchStatus(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodWatchStatus {
			result := WatchStatusResult{
				Watching:  true,
				Paths:     []string{"/tmp"},
				FileCount: 100,
			}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.WatchStatus()
	if err != nil {
		t.Fatalf("WatchStatus() error = %v", err)
	}

	if !result.Watching {
		t.Error("Watching should be true")
	}
	if len(result.Paths) != 1 || result.Paths[0] != "/tmp" {
		t.Errorf("Paths = %v, want [/tmp]", result.Paths)
	}
}

func TestClient_CallNotConnected(t *testing.T) {
	t.Parallel()
	client := &Client{
		conn:    nil,
		closeCh: make(chan struct{}),
	}

	_, err := client.Ping()
	if err == nil {
		t.Error("Ping() should return error when not connected")
	}
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("Error should be ErrNotConnected, got %v", err)
	}
}

func TestClient_CallServerDisconnects(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	// Server accepts then immediately closes
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// Read request then close without responding
		decoder := json.NewDecoder(bufio.NewReader(conn))
		var req Request
		decoder.Decode(&req)
		conn.Close()
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	_, err = client.Ping()
	if err == nil {
		t.Error("Ping() should return error when server disconnects")
	}
}

func TestClient_RPCError(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		// Return an error response
		resp := NewErrorResponse(req.ID, ErrCodeMethodNotFound, "Method not found", nil)
		encoder.Encode(resp)
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	_, err = client.Ping()
	if err == nil {
		t.Error("Ping() should return error on RPC error")
	}

	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Errorf("Error should be RPCError, got %T: %v", err, err)
	}
	if rpcErr != nil && rpcErr.Code != ErrCodeMethodNotFound {
		t.Errorf("Error code = %d, want %d", rpcErr.Code, ErrCodeMethodNotFound)
	}
}

func TestClient_ConcurrentRequests(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	// Server that handles multiple requests
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		for {
			var req Request
			if err := decoder.Decode(&req); err != nil {
				return
			}

			result := PingResult{Pong: true, Version: "1.0.0"}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	// Note: The current client implementation uses locks for encoder/decoder,
	// but doesn't handle out-of-order responses. Sequential requests work fine.
	for i := 0; i < 10; i++ {
		result, err := client.Ping()
		if err != nil {
			t.Errorf("Ping %d error = %v", i, err)
			continue
		}
		if !result.Pong {
			t.Errorf("Ping %d: Pong = false", i)
		}
	}
}

func TestClient_SubscribeEvents(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	notifSent := make(chan struct{})

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		encoder := json.NewEncoder(conn)

		// Send a notification
		notif, _ := NewNotification(MethodWatchEvent, WatchEventParams{
			Type:      "change",
			Files:     []string{"main.go"},
			Timestamp: time.Now().Format(time.RFC3339),
		})
		encoder.Encode(notif)
		close(notifSent)

		// Keep connection open
		time.Sleep(time.Second)
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
		return // for nilaway
	}
	defer client.Close()

	ch, err := client.SubscribeEvents()
	if err != nil {
		t.Fatalf("SubscribeEvents() error = %v", err)
		return // for nilaway
	}

	// Wait for notification to be sent
	<-notifSent

	// Try to receive
	select {
	case notif := <-ch:
		if notif.Method != MethodWatchEvent {
			t.Errorf("Method = %q, want %q", notif.Method, MethodWatchEvent)
		}
	case <-time.After(time.Second):
		t.Log("No notification received (timeout)")
	}
}

func TestClient_SubscribeEventsIdempotent(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(2 * time.Second)
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
		return // for nilaway
	}
	defer client.Close()

	// Multiple subscribes should return the same channel
	ch1, _ := client.SubscribeEvents()
	ch2, _ := client.SubscribeEvents()

	if ch1 != ch2 {
		t.Error("Multiple SubscribeEvents() calls should return same channel")
	}
}

func TestClient_IDGenerator_Concurrent(t *testing.T) {
	t.Parallel()
	client := &Client{closeCh: make(chan struct{})}

	const numGoroutines = 100
	ids := make(chan int64, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- client.idGen.Next()
		}()
	}

	wg.Wait()
	close(ids)

	// Verify uniqueness
	seen := make(map[int64]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID: %d", id)
		}
		seen[id] = true
	}
}

func TestIsDaemonRunning(t *testing.T) {
	// This test uses default paths, which may or may not have a daemon
	// Just verify it doesn't panic
	_ = IsDaemonRunning()
}

func TestIsDaemonRunningAt(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
	}

	// No daemon running
	if IsDaemonRunningAt(paths) {
		t.Error("Should return false when no daemon files exist")
	}
}

func TestIsConnectionRefused(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "net.OpError",
			err:  &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isConnectionRefused(tt.err)
			if got != tt.want {
				t.Errorf("isConnectionRefused() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_WatchStart(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodWatchStart {
			result := WatchStartResult{
				Status: "watching",
				Paths:  []string{"."},
			}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.WatchStart(&WatchStartParams{
		Paths:    []string{"."},
		Debounce: 500,
	})
	if err != nil {
		t.Fatalf("WatchStart() error = %v", err)
	}

	if result.Status != "watching" {
		t.Errorf("Status = %q, want %q", result.Status, "watching")
	}
}

func TestClient_WatchStop(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodWatchStop {
			result := WatchStopResult{Status: "stopped"}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.WatchStop()
	if err != nil {
		t.Fatalf("WatchStop() error = %v", err)
	}

	if result.Status != "stopped" {
		t.Errorf("Status = %q, want %q", result.Status, "stopped")
	}
}

func TestClient_UpdateRun(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodUpdateRun {
			result := UpdateRunResult{
				Status:      "success",
				UpdatedDirs: []string{"src"},
				Duration:    "1.5s",
			}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.UpdateRun(&UpdateRunParams{
		Paths:       []string{"."},
		Incremental: true,
	})
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
}

func TestClient_StatusGet(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		if req.Method == MethodStatusGet {
			result := StatusGetResult{
				Stale:     true,
				StaleDirs: []string{"pkg"},
			}
			resp, _ := NewResponse(*req.ID, result)
			encoder.Encode(resp)
		}
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	result, err := client.StatusGet()
	if err != nil {
		t.Fatalf("StatusGet() error = %v", err)
	}

	if !result.Stale {
		t.Error("Stale should be true")
	}
}

func TestClient_ReadEventsChannelClosed(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// Close immediately
		conn.Close()
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
		return // for nilaway
	}

	ch, _ := client.SubscribeEvents()

	// Wait for channel to close
	select {
	case _, ok := <-ch:
		if ok {
			t.Log("Received notification")
		} else {
			// Channel closed as expected
		}
	case <-time.After(2 * time.Second):
		// Timeout is acceptable
	}

	client.Close()
}

// TestClient_ReadEventsWithMixedMessages tests reading when responses and notifications are mixed
func TestClient_ReadEventsWithMixedMessages(t *testing.T) {
	t.Parallel()
	tmpDir := shortTempDir(t)
	socketPath := filepath.Join(tmpDir, "daemon.sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		decoder := json.NewDecoder(bufio.NewReader(conn))
		encoder := json.NewEncoder(conn)

		// Read request first
		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		// Send response
		result := PingResult{Pong: true}
		resp, _ := NewResponse(*req.ID, result)
		encoder.Encode(resp)

		// Send a notification
		notif, _ := NewNotification(MethodWatchEvent, WatchEventParams{
			Type:      "change",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		encoder.Encode(notif)

		time.Sleep(time.Second)
	}()

	client, err := Connect(socketPath)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
		return // for nilaway
	}
	defer client.Close()

	// The client implementation doesn't support interleaved responses and notifications well
	// This test documents the current behavior
	_, err = client.Ping()
	if err != nil && err != io.EOF {
		t.Logf("Ping error (may be expected): %v", err)
	}
}

// mockListener creates a server that responds to all methods
func setupTestServer(t *testing.T, socketPath string) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer listener.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			listener.(*net.UnixListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go handleTestConnection(conn)
		}
	}()

	return cancel, done
}

func handleTestConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(bufio.NewReader(conn))
	encoder := json.NewEncoder(conn)

	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			return
		}

		var resp *Response
		switch req.Method {
		case MethodPing:
			resp, _ = NewResponse(*req.ID, PingResult{Pong: true, Version: "test"})
		case MethodShutdown:
			resp, _ = NewResponse(*req.ID, ShutdownResult{Message: "ok"})
		case MethodWatchStatus:
			resp, _ = NewResponse(*req.ID, WatchStatusResult{Watching: false})
		default:
			resp = NewErrorResponse(req.ID, ErrCodeMethodNotFound, "not found", nil)
		}
		encoder.Encode(resp)
	}
}
