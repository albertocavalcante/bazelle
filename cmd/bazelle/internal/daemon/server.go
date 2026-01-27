package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/albertocavalcante/bazelle/internal/log"
)

// Server is the daemon server that listens on a Unix socket.
type Server struct {
	paths     *Paths
	listener  net.Listener
	handler   *Handler
	startTime time.Time
	version   string

	// Client management
	clients   map[*ClientConn]struct{}
	clientsMu sync.RWMutex

	// Shutdown management
	shutdown    chan struct{}
	shutdownMu  sync.Mutex
	isShutdown  bool
	wg          sync.WaitGroup
	shutdownErr error
}

// ClientConn represents a connected client.
type ClientConn struct {
	conn       net.Conn
	server     *Server
	encoder    *json.Encoder
	decoder    *json.Decoder
	encoderMu  sync.Mutex
	subscribed bool // whether client wants watch events
	closed     bool
	closeMu    sync.Mutex
}

// ServerConfig configures the daemon server.
type ServerConfig struct {
	Paths   *Paths
	Version string
	Handler *Handler
}

// NewServer creates a new daemon server.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		paths:     cfg.Paths,
		version:   cfg.Version,
		clients:   make(map[*ClientConn]struct{}),
		shutdown:  make(chan struct{}),
		startTime: time.Now(),
	}

	// Create handler with reference to server
	if cfg.Handler != nil {
		s.handler = cfg.Handler
		s.handler.server = s
	} else {
		s.handler = NewHandler(s)
	}

	return s
}

// Start starts the server and listens for connections.
// This method blocks until the server is shut down.
func (s *Server) Start(ctx context.Context) error {
	logger := log.Component("daemon")

	// Clean up stale files from previous runs
	if _, err := CleanupStale(s.paths); err != nil {
		logger.Warnw("failed to clean up stale files", "error", err)
	}

	// Ensure daemon directory exists
	if err := s.paths.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create daemon directory: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.paths.Socket)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions (readable/writable by owner only)
	if err := os.Chmod(s.paths.Socket, 0600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Write PID file
	if err := s.paths.WritePID(); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	logger.Infow("daemon started",
		"pid", os.Getpid(),
		"socket", s.paths.Socket,
		"version", s.version)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Accept loop in goroutine
	s.wg.Add(1)
	go s.acceptLoop()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		logger.Infow("context cancelled, shutting down")
	case sig := <-sigCh:
		logger.Infow("received signal, shutting down", "signal", sig)
	case <-s.shutdown:
		logger.Infow("shutdown requested via RPC")
	}

	// Perform graceful shutdown
	return s.Shutdown()
}

// acceptLoop accepts new client connections.
func (s *Server) acceptLoop() {
	defer s.wg.Done()
	logger := log.Component("daemon")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			s.shutdownMu.Lock()
			isShutdown := s.isShutdown
			s.shutdownMu.Unlock()

			if isShutdown {
				return
			}

			// Log error but continue accepting
			if !errors.Is(err, net.ErrClosed) {
				logger.Warnw("accept error", "error", err)
			}
			continue
		}

		// Create client connection
		client := &ClientConn{
			conn:    conn,
			server:  s,
			encoder: json.NewEncoder(conn),
			decoder: json.NewDecoder(bufio.NewReader(conn)),
		}

		// Register client
		s.clientsMu.Lock()
		s.clients[client] = struct{}{}
		clientCount := len(s.clients)
		s.clientsMu.Unlock()

		logger.Debugw("client connected", "client_count", clientCount)

		// Handle client in goroutine
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleClient(client)
		}()
	}
}

// handleClient processes requests from a single client.
func (s *Server) handleClient(client *ClientConn) {
	logger := log.Component("daemon")
	defer func() {
		client.Close()
		s.clientsMu.Lock()
		delete(s.clients, client)
		clientCount := len(s.clients)
		s.clientsMu.Unlock()
		logger.Debugw("client disconnected", "client_count", clientCount)
	}()

	for {
		var req Request
		if err := client.decoder.Decode(&req); err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				return // Client disconnected
			}
			// Parse error - send error response
			logger.Debugw("failed to decode request", "error", err)
			resp := NewErrorResponse(nil, ErrCodeParseError, "Parse error", nil)
			if err := client.Send(resp); err != nil {
				logger.Debugw("failed to send error response", "error", err)
			}
			continue
		}

		// Validate JSON-RPC version
		if req.JSONRPC != JSONRPCVersion {
			resp := NewErrorResponse(req.ID, ErrCodeInvalidRequest, "Invalid Request: unsupported JSON-RPC version", nil)
			if err := client.Send(resp); err != nil {
				logger.Debugw("failed to send error response", "error", err)
			}
			continue
		}

		// Handle the request
		resp := s.handler.HandleRequest(client, &req)
		if resp != nil {
			if err := client.Send(resp); err != nil {
				logger.Debugw("failed to send response", "error", err)
				return
			}
		}
	}
}

// Shutdown performs a graceful shutdown of the server.
func (s *Server) Shutdown() error {
	s.shutdownMu.Lock()
	if s.isShutdown {
		s.shutdownMu.Unlock()
		return s.shutdownErr
	}
	s.isShutdown = true
	s.shutdownMu.Unlock()

	logger := log.Component("daemon")
	logger.Infow("shutting down daemon")

	// Stop accepting new connections
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			logger.Warnw("failed to close listener", "error", err)
		}
	}

	// Notify all clients of shutdown
	s.notifyShutdown()

	// Stop handler (stops watcher)
	if s.handler != nil {
		s.handler.Stop()
	}

	// Close all client connections
	s.clientsMu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clientsMu.Unlock()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		logger.Warnw("shutdown timed out waiting for goroutines")
	}

	// Cleanup files
	if err := s.paths.Cleanup(); err != nil {
		logger.Warnw("failed to cleanup daemon files", "error", err)
		s.shutdownErr = err
	}

	logger.Infow("daemon stopped")
	return s.shutdownErr
}

// RequestShutdown requests the server to shut down.
func (s *Server) RequestShutdown() {
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	if !s.isShutdown {
		close(s.shutdown)
	}
}

// notifyShutdown sends a shutdown notification to all subscribed clients.
func (s *Server) notifyShutdown() {
	notif, err := NewNotification(MethodWatchEvent, WatchEventParams{
		Type:      "shutdown",
		Message:   "daemon is shutting down",
		Timestamp: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	s.Broadcast(notif)
}

// Broadcast sends a notification to all subscribed clients.
func (s *Server) Broadcast(notif *Notification) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for client := range s.clients {
		if client.subscribed {
			_ = client.Send(notif)
		}
	}
}

// GetInfo returns information about the running daemon.
func (s *Server) GetInfo() *DaemonInfo {
	s.clientsMu.RLock()
	clientCount := len(s.clients)
	s.clientsMu.RUnlock()

	info := &DaemonInfo{
		PID:         os.Getpid(),
		SocketPath:  s.paths.Socket,
		StartTime:   s.startTime,
		Version:     s.version,
		ClientCount: clientCount,
	}

	if s.handler != nil {
		status := s.handler.GetWatchStatus()
		info.Watching = status.Watching
		info.WatchPaths = status.Paths
	}

	return info
}

// Uptime returns how long the server has been running.
func (s *Server) Uptime() time.Duration {
	return time.Since(s.startTime)
}

// Send sends a message to the client (thread-safe).
func (c *ClientConn) Send(msg any) error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return net.ErrClosed
	}
	c.closeMu.Unlock()

	c.encoderMu.Lock()
	defer c.encoderMu.Unlock()
	return c.encoder.Encode(msg)
}

// Close closes the client connection.
func (c *ClientConn) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return
	}
	c.closed = true
	_ = c.conn.Close()
}

// Subscribe enables event notifications for this client.
func (c *ClientConn) Subscribe() {
	c.subscribed = true
}

// Unsubscribe disables event notifications for this client.
func (c *ClientConn) Unsubscribe() {
	c.subscribed = false
}
