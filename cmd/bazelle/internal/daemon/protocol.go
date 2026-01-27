// Package daemon implements the bazelle daemon server and client.
package daemon

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
)

// JSON-RPC 2.0 version string.
const JSONRPCVersion = "2.0"

// JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"` // nil for notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// NewRequest creates a new JSON-RPC request.
func NewRequest(id int64, method string, params any) (*Request, error) {
	req := &Request{
		JSONRPC: JSONRPCVersion,
		ID:      &id,
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = data
	}

	return req, nil
}

// NewNotification creates a new JSON-RPC notification.
func NewNotification(method string, params any) (*Notification, error) {
	notif := &Notification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		notif.Params = data
	}

	return notif, nil
}

// NewResponse creates a successful JSON-RPC response.
func NewResponse(id int64, result any) (*Response, error) {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      &id,
	}

	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resp.Result = data
	} else {
		// JSON-RPC requires result to be present on success (can be null)
		resp.Result = json.RawMessage("null")
	}

	return resp, nil
}

// NewErrorResponse creates an error JSON-RPC response.
func NewErrorResponse(id *int64, code int, message string, data any) *Response {
	resp := &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}

	if data != nil {
		if d, err := json.Marshal(data); err == nil {
			resp.Error.Data = d
		}
	}

	return resp
}

// Standard RPC methods.
const (
	MethodPing        = "ping"
	MethodShutdown    = "shutdown"
	MethodWatchStart  = "watch/start"
	MethodWatchStop   = "watch/stop"
	MethodWatchStatus = "watch/status"
	MethodWatchEvent  = "watch/event" // notification from server to client
	MethodUpdateRun   = "update/run"
	MethodStatusGet   = "status/get"
)

// PingResult is the response to a ping request.
type PingResult struct {
	Pong      bool   `json:"pong"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	StartTime string `json:"start_time"`
}

// ShutdownResult is the response to a shutdown request.
type ShutdownResult struct {
	Message string `json:"message"`
}

// WatchStartParams are the parameters for watch/start.
type WatchStartParams struct {
	Paths     []string `json:"paths,omitempty"`
	Languages []string `json:"languages,omitempty"`
	Debounce  int      `json:"debounce,omitempty"` // milliseconds
}

// WatchStartResult is the response to watch/start.
type WatchStartResult struct {
	Status    string   `json:"status"`
	Paths     []string `json:"paths"`
	Languages []string `json:"languages,omitempty"`
}

// WatchStopResult is the response to watch/stop.
type WatchStopResult struct {
	Status string `json:"status"`
}

// WatchStatusResult is the response to watch/status.
type WatchStatusResult struct {
	Watching   bool     `json:"watching"`
	Paths      []string `json:"paths,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	FileCount  int      `json:"file_count,omitempty"`
	UpdateTime string   `json:"update_time,omitempty"` // time of last update
}

// WatchEventParams are the parameters for watch/event notifications.
type WatchEventParams struct {
	Type        string   `json:"type"` // "change", "update", "error"
	Directories []string `json:"directories,omitempty"`
	Files       []string `json:"files,omitempty"`
	Message     string   `json:"message,omitempty"`
	Timestamp   string   `json:"timestamp"`
}

// UpdateRunParams are the parameters for update/run.
type UpdateRunParams struct {
	Paths       []string `json:"paths,omitempty"`
	Incremental bool     `json:"incremental,omitempty"`
}

// UpdateRunResult is the response to update/run.
type UpdateRunResult struct {
	Status      string   `json:"status"`
	UpdatedDirs []string `json:"updated_dirs,omitempty"`
	Duration    string   `json:"duration,omitempty"`
}

// StatusGetResult is the response to status/get.
type StatusGetResult struct {
	Stale     bool     `json:"stale"`
	StaleDirs []string `json:"stale_dirs,omitempty"`
}

// IDGenerator generates unique request IDs.
type IDGenerator struct {
	counter atomic.Int64
}

// Next returns the next unique ID.
func (g *IDGenerator) Next() int64 {
	return g.counter.Add(1)
}

// DaemonInfo contains information about the running daemon.
type DaemonInfo struct {
	PID         int       `json:"pid"`
	SocketPath  string    `json:"socket_path"`
	StartTime   time.Time `json:"start_time"`
	Version     string    `json:"version"`
	Watching    bool      `json:"watching"`
	WatchPaths  []string  `json:"watch_paths,omitempty"`
	ClientCount int       `json:"client_count"`
}
