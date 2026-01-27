//go:build integration

package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// shortTempDirIntegration creates a short temp directory for Unix socket tests.
// Unix sockets have a path length limit (~104 chars on macOS).
func shortTempDirIntegration(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "di")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// Integration tests that test the full client-server interaction.
// Run with: go test -tags=integration ./cmd/bazelle/internal/daemon/...

func TestIntegration_ClientServerPing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "integration-test-1.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	// Wait for server to be ready
	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	// Connect client
	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { client.Close() })

	// Ping
	result, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if !result.Pong {
		t.Error("Pong should be true")
	}
	if result.Version != "integration-test-1.0" {
		t.Errorf("Version = %q, want %q", result.Version, "integration-test-1.0")
	}

	// Shutdown
	cancel()
	select {
	case err := <-serverErr:
		if err != nil {
			t.Logf("Server shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server shutdown timed out")
	}
}

func TestIntegration_ClientServerShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx := context.Background()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Request shutdown via RPC
	result, err := client.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if result.Message == "" {
		t.Error("Shutdown message should not be empty")
	}

	// Wait for server to shut down
	select {
	case err := <-serverErr:
		if err != nil {
			t.Logf("Server shutdown: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Server did not shut down after RPC shutdown request")
	}

	// Verify files are cleaned up
	if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after shutdown")
	}
	if _, err := os.Stat(paths.PID); !os.IsNotExist(err) {
		t.Error("PID file should be removed after shutdown")
	}
}

func TestIntegration_MultipleClients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	const numClients = 10
	const requestsPerClient = 5

	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	errorCount := atomic.Int32{}

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client, err := Connect(paths.Socket)
			if err != nil {
				t.Logf("Client %d: Connect error: %v", clientID, err)
				errorCount.Add(1)
				return
			}
			defer client.Close()

			for j := 0; j < requestsPerClient; j++ {
				result, err := client.Ping()
				if err != nil {
					t.Logf("Client %d request %d: Ping error: %v", clientID, j, err)
					errorCount.Add(1)
					continue
				}
				if result.Pong {
					successCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalExpected := int32(numClients * requestsPerClient)
	if successCount.Load() < totalExpected/2 {
		t.Errorf("Only %d/%d requests succeeded", successCount.Load(), totalExpected)
	}

	cancel()
	<-serverErr
}

func TestIntegration_ClientReconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	// Connect, use, disconnect, reconnect multiple times
	for i := 0; i < 5; i++ {
		client, err := Connect(paths.Socket)
		if err != nil {
			t.Fatalf("Iteration %d: Connect error: %v", i, err)
		}

		result, err := client.Ping()
		if err != nil {
			t.Fatalf("Iteration %d: Ping error: %v", i, err)
		}

		if !result.Pong {
			t.Errorf("Iteration %d: Pong should be true", i)
		}

		client.Close()
	}

	cancel()
	<-serverErr
}

func TestIntegration_ServerRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	// Start server 1
	server1 := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx1, cancel1 := context.WithCancel(context.Background())
	serverErr1 := make(chan error, 1)
	go func() {
		serverErr1 <- server1.Start(ctx1)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server 1 did not start in time")
	}

	// Use server 1
	client1, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect to server 1 error: %v", err)
	}
	_, err = client1.Ping()
	if err != nil {
		t.Fatalf("Ping server 1 error: %v", err)
	}
	client1.Close()

	// Shutdown server 1
	cancel1()
	<-serverErr1

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Start server 2
	server2 := NewServer(ServerConfig{
		Paths:   paths,
		Version: "2.0.0",
	})

	ctx2, cancel2 := context.WithCancel(context.Background())
	t.Cleanup(cancel2)

	serverErr2 := make(chan error, 1)
	go func() {
		serverErr2 <- server2.Start(ctx2)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server 2 did not start in time")
	}

	// Use server 2
	client2, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect to server 2 error: %v", err)
	}
	defer client2.Close()

	result, err := client2.Ping()
	if err != nil {
		t.Fatalf("Ping server 2 error: %v", err)
	}

	if result.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", result.Version, "2.0.0")
	}

	cancel2()
	<-serverErr2
}

func TestIntegration_StaleSocketCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	// Create stale socket file
	if err := os.WriteFile(paths.Socket, []byte("stale"), 0600); err != nil {
		t.Fatalf("Failed to create stale socket: %v", err)
	}

	// Create stale PID file with non-existent PID
	if err := os.WriteFile(paths.PID, []byte("999999999"), 0600); err != nil {
		t.Fatalf("Failed to create stale PID: %v", err)
	}

	// Server should clean up stale files and start successfully
	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start after stale cleanup")
	}

	// Verify we can connect
	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	result, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping error: %v", err)
	}

	if !result.Pong {
		t.Error("Pong should be true")
	}

	cancel()
	<-serverErr
}

func TestIntegration_WatchStatusRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// Check initial status
	status, err := client.WatchStatus()
	if err != nil {
		t.Fatalf("WatchStatus error: %v", err)
	}

	if status.Watching {
		t.Error("Should not be watching initially")
	}

	cancel()
	<-serverErr
}

func TestIntegration_RapidConnectDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	// Rapidly connect and disconnect
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("unix", paths.Socket, time.Second)
		if err != nil {
			t.Logf("Connection %d failed: %v", i, err)
			continue
		}
		conn.Close()
	}

	// Server should still be responsive
	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Final connect error: %v", err)
	}
	defer client.Close()

	_, err = client.Ping()
	if err != nil {
		t.Fatalf("Final ping error: %v", err)
	}

	cancel()
	<-serverErr
}

func TestIntegration_ClientTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	// Connect with a short deadline
	conn, err := net.DialTimeout("unix", paths.Socket, time.Second)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer conn.Close()

	// Set a very short read deadline
	conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))

	// Try to read (should timeout)
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err == nil {
		t.Error("Expected timeout error")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Logf("Error type: %T, error: %v", err, err)
	}

	cancel()
	<-serverErr
}

func TestIntegration_LargeRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// Create a large params payload
	largePaths := make([]string, 1000)
	for i := range largePaths {
		largePaths[i] = "/some/very/long/path/that/goes/on/and/on/" + string(rune('a'+i%26))
	}

	// This should either succeed or return a reasonable error
	_, err = client.WatchStart(&WatchStartParams{
		Paths:    largePaths,
		Debounce: 500,
	})
	// We expect this might fail (watcher creation may fail without actual paths)
	// but it should not panic or hang
	t.Logf("Large request result: %v", err)

	cancel()
	<-serverErr
}

func TestIntegration_GracefulShutdownWithPendingRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	// Start multiple clients making requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := Connect(paths.Socket)
			if err != nil {
				return
			}
			defer client.Close()

			for j := 0; j < 10; j++ {
				_, _ = client.Ping()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Trigger shutdown while requests are in flight
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for server shutdown
	select {
	case <-serverErr:
		// Good
	case <-time.After(10 * time.Second):
		t.Fatal("Server did not shut down in time")
	}

	wg.Wait()
}

// waitForSocket waits for a Unix socket to become available.
func waitForSocket(t *testing.T, socketPath string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}

	return false
}

func TestIntegration_AllRPCMethods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	client, err := Connect(paths.Socket)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	t.Run("Ping", func(t *testing.T) {
		result, err := client.Ping()
		if err != nil {
			t.Errorf("Ping error: %v", err)
			return
		}
		if !result.Pong {
			t.Error("Pong should be true")
		}
	})

	t.Run("WatchStatus", func(t *testing.T) {
		result, err := client.WatchStatus()
		if err != nil {
			t.Errorf("WatchStatus error: %v", err)
			return
		}
		if result.Watching {
			t.Error("Should not be watching initially")
		}
	})

	t.Run("WatchStop", func(t *testing.T) {
		result, err := client.WatchStop()
		if err != nil {
			t.Errorf("WatchStop error: %v", err)
			return
		}
		if result.Status != "not_watching" {
			t.Errorf("Status = %q, want %q", result.Status, "not_watching")
		}
	})

	t.Run("StatusGet", func(t *testing.T) {
		result, err := client.StatusGet()
		if err != nil {
			t.Errorf("StatusGet error: %v", err)
			return
		}
		// Default is not stale
		if result.Stale {
			t.Log("StatusGet returned stale=true")
		}
	})

	cancel()
	<-serverErr
}

func TestIntegration_ErrorResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := shortTempDirIntegration(t)
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "daemon.sock"),
		PID:    filepath.Join(tmpDir, "daemon.pid"),
		Log:    filepath.Join(tmpDir, "daemon.log"),
	}

	server := NewServer(ServerConfig{
		Paths:   paths,
		Version: "1.0.0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start(ctx)
	}()

	if !waitForSocket(t, paths.Socket, 5*time.Second) {
		t.Fatal("Server did not start in time")
	}

	conn, err := net.DialTimeout("unix", paths.Socket, time.Second)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer conn.Close()

	// Send request for unknown method
	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      ptr(int64(1)),
		Method:  "unknown/method",
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	decoder := json.NewDecoder(conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error response")
	}
	if resp.Error != nil && resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Error code = %d, want %d", resp.Error.Code, ErrCodeMethodNotFound)
	}

	cancel()
	<-serverErr
}
