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

	// Watch state
	watchMu      sync.RWMutex
	watcher      *watch.Watcher
	watchCancel  context.CancelFunc
	watchPaths   []string
	watchLangs   []string
	lastUpdate   time.Time
	watching     bool
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

	switch req.Method {
	case MethodPing:
		return h.handlePing(req)
	case MethodShutdown:
		return h.handleShutdown(req)
	case MethodWatchStart:
		return h.handleWatchStart(client, req)
	case MethodWatchStop:
		return h.handleWatchStop(req)
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

	// If already watching, return current status
	if h.watching {
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

	cfg := watch.Config{
		Root:            watchPath,
		Languages:       h.languages,
		LangFilter:      params.Languages,
		Debounce:        debounce,
		Verbose:         false,
		NoColor:         true,
		JSON:            false,
		GazelleDefaults: h.defaults,
	}

	watcher, err := watch.New(cfg)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to start watcher", err.Error())
	}

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	h.watcher = watcher
	h.watchCancel = cancel
	h.watchPaths = paths
	h.watchLangs = params.Languages
	h.watching = true

	go h.runWatcher(ctx, watcher)

	// Subscribe client to events
	client.Subscribe()

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

// runWatcher runs the watcher and broadcasts events.
func (h *Handler) runWatcher(ctx context.Context, w *watch.Watcher) {
	logger := log.Component("daemon")

	// Run the watcher (this blocks)
	if err := w.Run(ctx); err != nil {
		logger.Warnw("watcher stopped with error", "error", err)
	}

	h.watchMu.Lock()
	h.watching = false
	h.watcher = nil
	h.watchCancel = nil
	h.watchMu.Unlock()

	logger.Infow("watcher stopped")
}

// handleWatchStop handles the watch/stop request.
func (h *Handler) handleWatchStop(req *Request) *Response {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()

	if !h.watching {
		result := WatchStopResult{
			Status: "not_watching",
		}
		resp, _ := NewResponse(*req.ID, result)
		return resp
	}

	// Cancel the watcher
	if h.watchCancel != nil {
		h.watchCancel()
	}
	if h.watcher != nil {
		_ = h.watcher.Close()
	}

	h.watching = false
	h.watcher = nil
	h.watchCancel = nil

	result := WatchStopResult{
		Status: "stopped",
	}

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
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
	var params UpdateRunParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return NewErrorResponse(req.ID, ErrCodeInvalidParams, "Invalid params", err.Error())
		}
	}

	// TODO: Implement manual update trigger
	// For now, return a not implemented error
	return NewErrorResponse(req.ID, ErrCodeInternalError, "Not implemented", nil)
}

// handleStatusGet handles the status/get request.
func (h *Handler) handleStatusGet(req *Request) *Response {
	// TODO: Implement status/get using incremental tracker
	// For now, return empty status
	result := StatusGetResult{
		Stale:     false,
		StaleDirs: nil,
	}

	resp, err := NewResponse(*req.ID, result)
	if err != nil {
		return NewErrorResponse(req.ID, ErrCodeInternalError, "Failed to create response", nil)
	}
	return resp
}

// Stop stops the handler and any running watcher.
func (h *Handler) Stop() {
	h.watchMu.Lock()
	defer h.watchMu.Unlock()

	if h.watchCancel != nil {
		h.watchCancel()
	}
	if h.watcher != nil {
		_ = h.watcher.Close()
	}

	h.watching = false
	h.watcher = nil
	h.watchCancel = nil
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
