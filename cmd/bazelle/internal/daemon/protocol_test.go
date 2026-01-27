package daemon

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestJSONRPCVersion(t *testing.T) {
	t.Parallel()
	if JSONRPCVersion != "2.0" {
		t.Errorf("JSONRPCVersion = %q, want %q", JSONRPCVersion, "2.0")
	}
}

func TestErrorCodes(t *testing.T) {
	t.Parallel()
	// Verify standard JSON-RPC error codes
	tests := []struct {
		name string
		code int
		want int
	}{
		{"ParseError", ErrCodeParseError, -32700},
		{"InvalidRequest", ErrCodeInvalidRequest, -32600},
		{"MethodNotFound", ErrCodeMethodNotFound, -32601},
		{"InvalidParams", ErrCodeInvalidParams, -32602},
		{"InternalError", ErrCodeInternalError, -32603},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

func TestRPCError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *RPCError
		wantStr string
	}{
		{
			name:    "basic error",
			err:     &RPCError{Code: -32600, Message: "Invalid Request"},
			wantStr: "RPC error -32600: Invalid Request",
		},
		{
			name:    "with data",
			err:     &RPCError{Code: -32603, Message: "Internal error", Data: json.RawMessage(`"details"`)},
			wantStr: "RPC error -32603: Internal error",
		},
		{
			name:    "empty message",
			err:     &RPCError{Code: 0, Message: ""},
			wantStr: "RPC error 0: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantStr {
				t.Errorf("Error() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestRPCError_ImplementsError(t *testing.T) {
	t.Parallel()
	var err error = &RPCError{Code: -32600, Message: "test"}
	if err == nil {
		t.Error("RPCError should implement error interface")
	}
}

func TestNewRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int64
		method     string
		params     any
		wantErr    bool
		wantMethod string
	}{
		{
			name:       "simple request",
			id:         1,
			method:     "ping",
			params:     nil,
			wantErr:    false,
			wantMethod: "ping",
		},
		{
			name:       "with struct params",
			id:         42,
			method:     "watch/start",
			params:     WatchStartParams{Paths: []string{"/tmp"}, Debounce: 500},
			wantErr:    false,
			wantMethod: "watch/start",
		},
		{
			name:       "with map params",
			id:         100,
			method:     "custom",
			params:     map[string]string{"key": "value"},
			wantErr:    false,
			wantMethod: "custom",
		},
		{
			name:    "unmarshalable params",
			id:      1,
			method:  "test",
			params:  make(chan int), // channels can't be marshaled
			wantErr: true,
		},
		{
			name:       "zero id",
			id:         0,
			method:     "test",
			params:     nil,
			wantErr:    false,
			wantMethod: "test",
		},
		{
			name:       "negative id",
			id:         -1,
			method:     "test",
			params:     nil,
			wantErr:    false,
			wantMethod: "test",
		},
		{
			name:       "max int64 id",
			id:         math.MaxInt64,
			method:     "test",
			params:     nil,
			wantErr:    false,
			wantMethod: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := NewRequest(tt.id, tt.method, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if req.JSONRPC != JSONRPCVersion {
				t.Errorf("JSONRPC = %q, want %q", req.JSONRPC, JSONRPCVersion)
			}
			if req.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", req.Method, tt.wantMethod)
			}
			if req.ID == nil || *req.ID != tt.id {
				t.Errorf("ID = %v, want %d", req.ID, tt.id)
			}
		})
	}
}

func TestNewNotification(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		method     string
		params     any
		wantErr    bool
		wantMethod string
	}{
		{
			name:       "simple notification",
			method:     "watch/event",
			params:     nil,
			wantMethod: "watch/event",
		},
		{
			name:   "with params",
			method: "watch/event",
			params: WatchEventParams{
				Type:      "change",
				Files:     []string{"main.go"},
				Timestamp: "2024-01-01T00:00:00Z",
			},
			wantMethod: "watch/event",
		},
		{
			name:    "unmarshalable params",
			method:  "test",
			params:  func() {}, // functions can't be marshaled
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			notif, err := NewNotification(tt.method, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNotification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if notif.JSONRPC != JSONRPCVersion {
				t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, JSONRPCVersion)
			}
			if notif.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", notif.Method, tt.wantMethod)
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		id         int64
		result     any
		wantErr    bool
		wantResult string
	}{
		{
			name:       "nil result",
			id:         1,
			result:     nil,
			wantResult: "null",
		},
		{
			name:       "struct result",
			id:         2,
			result:     PingResult{Pong: true, Version: "1.0"},
			wantResult: `{"pong":true,"version":"1.0","uptime":"","start_time":""}`,
		},
		{
			name:       "string result",
			id:         3,
			result:     "success",
			wantResult: `"success"`,
		},
		{
			name:    "unmarshalable result",
			id:      1,
			result:  make(chan int),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := NewResponse(tt.id, tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if resp.JSONRPC != JSONRPCVersion {
				t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
			}
			if resp.ID == nil || *resp.ID != tt.id {
				t.Errorf("ID = %v, want %d", resp.ID, tt.id)
			}
			if resp.Error != nil {
				t.Errorf("Error should be nil, got %v", resp.Error)
			}
			if string(resp.Result) != tt.wantResult {
				t.Errorf("Result = %s, want %s", resp.Result, tt.wantResult)
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	t.Parallel()
	id := int64(42)
	tests := []struct {
		name        string
		id          *int64
		code        int
		message     string
		data        any
		wantCode    int
		wantMessage string
		wantData    bool
	}{
		{
			name:        "basic error",
			id:          &id,
			code:        ErrCodeMethodNotFound,
			message:     "Method not found",
			data:        nil,
			wantCode:    ErrCodeMethodNotFound,
			wantMessage: "Method not found",
			wantData:    false,
		},
		{
			name:        "nil id",
			id:          nil,
			code:        ErrCodeParseError,
			message:     "Parse error",
			data:        nil,
			wantCode:    ErrCodeParseError,
			wantMessage: "Parse error",
			wantData:    false,
		},
		{
			name:        "with string data",
			id:          &id,
			code:        ErrCodeInternalError,
			message:     "Internal error",
			data:        "additional details",
			wantCode:    ErrCodeInternalError,
			wantMessage: "Internal error",
			wantData:    true,
		},
		{
			name:        "with struct data",
			id:          &id,
			code:        ErrCodeInvalidParams,
			message:     "Invalid params",
			data:        map[string]string{"field": "path"},
			wantCode:    ErrCodeInvalidParams,
			wantMessage: "Invalid params",
			wantData:    true,
		},
		{
			name:        "unmarshalable data is ignored",
			id:          &id,
			code:        ErrCodeInternalError,
			message:     "test",
			data:        make(chan int),
			wantCode:    ErrCodeInternalError,
			wantMessage: "test",
			wantData:    false, // unmarshalable data is silently ignored
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := NewErrorResponse(tt.id, tt.code, tt.message, tt.data)
			if resp.JSONRPC != JSONRPCVersion {
				t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
			}
			if (resp.ID == nil) != (tt.id == nil) {
				t.Errorf("ID nil mismatch")
			}
			if resp.Result != nil {
				t.Errorf("Result should be nil for error response")
			}
			if resp.Error == nil {
				t.Fatal("Error should not be nil")
			}
			if resp.Error.Code != tt.wantCode {
				t.Errorf("Error.Code = %d, want %d", resp.Error.Code, tt.wantCode)
			}
			if resp.Error.Message != tt.wantMessage {
				t.Errorf("Error.Message = %q, want %q", resp.Error.Message, tt.wantMessage)
			}
			if (resp.Error.Data != nil) != tt.wantData {
				t.Errorf("Error.Data presence = %v, want %v", resp.Error.Data != nil, tt.wantData)
			}
		})
	}
}

func TestRequestJSONRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		req    *Request
		params any
	}{
		{
			name: "request with params",
			req: &Request{
				JSONRPC: JSONRPCVersion,
				ID:      ptr(int64(1)),
				Method:  "watch/start",
				Params:  json.RawMessage(`{"paths":["/tmp"]}`),
			},
		},
		{
			name: "request without params",
			req: &Request{
				JSONRPC: JSONRPCVersion,
				ID:      ptr(int64(2)),
				Method:  "ping",
			},
		},
		{
			name: "notification (nil ID)",
			req: &Request{
				JSONRPC: JSONRPCVersion,
				ID:      nil,
				Method:  "notify",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got Request
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.JSONRPC != tt.req.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, tt.req.JSONRPC)
			}
			if got.Method != tt.req.Method {
				t.Errorf("Method = %q, want %q", got.Method, tt.req.Method)
			}
			if (got.ID == nil) != (tt.req.ID == nil) {
				t.Errorf("ID nil mismatch")
			}
			if got.ID != nil && tt.req.ID != nil && *got.ID != *tt.req.ID {
				t.Errorf("ID = %d, want %d", *got.ID, *tt.req.ID)
			}
		})
	}
}

func TestResponseJSONRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		resp *Response
	}{
		{
			name: "success response",
			resp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      ptr(int64(1)),
				Result:  json.RawMessage(`{"pong":true}`),
			},
		},
		{
			name: "error response",
			resp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      ptr(int64(2)),
				Error:   &RPCError{Code: -32600, Message: "Invalid Request"},
			},
		},
		{
			name: "null id error response",
			resp: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got Response
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.JSONRPC != tt.resp.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, tt.resp.JSONRPC)
			}
		})
	}
}

func TestMalformedJSONParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "not json",
			input:   "not json at all",
			wantErr: true,
		},
		{
			name:    "incomplete json",
			input:   `{"jsonrpc": "2.0"`,
			wantErr: true,
		},
		{
			name:    "array instead of object",
			input:   `[1, 2, 3]`,
			wantErr: true, // Go's JSON unmarshaler rejects array for struct
		},
		{
			name:    "null",
			input:   "null",
			wantErr: false,
		},
		{
			name:    "wrong type for id",
			input:   `{"jsonrpc":"2.0","id":"string","method":"test"}`,
			wantErr: true, // id should be int64
		},
		{
			name:    "float id",
			input:   `{"jsonrpc":"2.0","id":1.5,"method":"test"}`,
			wantErr: true, // id should be int64, not float
		},
		{
			name:    "very large id",
			input:   `{"jsonrpc":"2.0","id":9999999999999999999999,"method":"test"}`,
			wantErr: true, // overflow
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req Request
			err := json.Unmarshal([]byte(tt.input), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIDGenerator(t *testing.T) {
	t.Parallel()
	gen := &IDGenerator{}

	// First ID should be 1
	if id := gen.Next(); id != 1 {
		t.Errorf("First ID = %d, want 1", id)
	}

	// IDs should be sequential
	for i := int64(2); i <= 10; i++ {
		if id := gen.Next(); id != i {
			t.Errorf("ID = %d, want %d", id, i)
		}
	}
}

func TestIDGenerator_Concurrent(t *testing.T) {
	t.Parallel()
	gen := &IDGenerator{}
	const numGoroutines = 100
	const idsPerGoroutine = 100

	ids := make(chan int64, numGoroutines*idsPerGoroutine)
	done := make(chan struct{})

	// Start multiple goroutines generating IDs
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < idsPerGoroutine; j++ {
				ids <- gen.Next()
			}
		}()
	}

	// Collect all IDs
	go func() {
		collected := make([]int64, 0, numGoroutines*idsPerGoroutine)
		for i := 0; i < numGoroutines*idsPerGoroutine; i++ {
			collected = append(collected, <-ids)
		}

		// Verify all IDs are unique
		seen := make(map[int64]bool)
		for _, id := range collected {
			if seen[id] {
				t.Errorf("Duplicate ID: %d", id)
			}
			seen[id] = true
		}

		// Verify we got the expected range
		if len(seen) != numGoroutines*idsPerGoroutine {
			t.Errorf("Got %d unique IDs, want %d", len(seen), numGoroutines*idsPerGoroutine)
		}

		close(done)
	}()

	<-done
}

func TestMethodConstants(t *testing.T) {
	t.Parallel()
	// Verify method constants are defined as expected
	methods := []string{
		MethodPing,
		MethodShutdown,
		MethodWatchStart,
		MethodWatchStop,
		MethodWatchStatus,
		MethodWatchEvent,
		MethodUpdateRun,
		MethodStatusGet,
	}

	seen := make(map[string]bool)
	for _, m := range methods {
		if m == "" {
			t.Error("Empty method constant")
		}
		if seen[m] {
			t.Errorf("Duplicate method: %q", m)
		}
		seen[m] = true
	}
}

func TestParamStructsSerialization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value any
	}{
		{"PingResult", PingResult{Pong: true, Version: "1.0", Uptime: "1h0m0s", StartTime: "2024-01-01T00:00:00Z"}},
		{"ShutdownResult", ShutdownResult{Message: "shutting down"}},
		{"WatchStartParams", WatchStartParams{Paths: []string{"/a", "/b"}, Languages: []string{"go"}, Debounce: 500}},
		{"WatchStartResult", WatchStartResult{Status: "watching", Paths: []string{"/a"}}},
		{"WatchStopResult", WatchStopResult{Status: "stopped"}},
		{"WatchStatusResult", WatchStatusResult{Watching: true, Paths: []string{"."}, FileCount: 100}},
		{"WatchEventParams", WatchEventParams{Type: "change", Files: []string{"main.go"}}},
		{"UpdateRunParams", UpdateRunParams{Paths: []string{"."}, Incremental: true}},
		{"UpdateRunResult", UpdateRunResult{Status: "success", UpdatedDirs: []string{"src"}}},
		{"StatusGetResult", StatusGetResult{Stale: true, StaleDirs: []string{"pkg"}}},
		{"DaemonInfo", DaemonInfo{PID: 1234, SocketPath: "/tmp/daemon.sock", Version: "1.0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("Marshal() returned empty data")
			}
		})
	}
}

func TestWatchStartParams_Defaults(t *testing.T) {
	t.Parallel()
	// Test that empty params deserialize correctly
	input := `{}`
	var params WatchStartParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if params.Paths != nil && len(params.Paths) != 0 {
		t.Errorf("Paths should be nil or empty, got %v", params.Paths)
	}
	if params.Debounce != 0 {
		t.Errorf("Debounce should be 0 (default), got %d", params.Debounce)
	}
}

func TestUnicodeInParams(t *testing.T) {
	t.Parallel()
	params := WatchStartParams{
		Paths: []string{"/tmp/\u4e2d\u6587", "/tmp/\u0440\u0443\u0441\u0441\u043a\u0438\u0439"},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got WatchStartParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !slices.Equal(params.Paths, got.Paths) {
		t.Errorf("Unicode paths not preserved: got %v, want %v", got.Paths, params.Paths)
	}
}

func TestSpecialCharactersInPaths(t *testing.T) {
	t.Parallel()
	specialPaths := []string{
		"/tmp/path with spaces",
		"/tmp/path\twith\ttabs",
		"/tmp/path\"with\"quotes",
		"/tmp/path\\with\\backslash",
		"/tmp/path\nwith\nnewlines",
	}
	params := WatchStartParams{Paths: specialPaths}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got WatchStartParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !slices.Equal(params.Paths, got.Paths) {
		t.Errorf("Special character paths not preserved")
	}
}

func TestEmptyResponse(t *testing.T) {
	t.Parallel()
	// Test response with null result
	resp, err := NewResponse(1, nil)
	if err != nil {
		t.Fatalf("NewResponse() error = %v", err)
	}
	if string(resp.Result) != "null" {
		t.Errorf("Result = %s, want null", resp.Result)
	}
}

func TestRPCError_AsError(t *testing.T) {
	t.Parallel()
	rpcErr := &RPCError{Code: -32600, Message: "test error"}
	var err error = rpcErr

	// Test errors.As
	var target *RPCError
	if !errors.As(err, &target) {
		t.Error("errors.As should match RPCError")
	}
	if target.Code != -32600 {
		t.Errorf("Code = %d, want -32600", target.Code)
	}
}

// ptr is a helper function to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}

// FuzzRequest tests request parsing with random inputs.
func FuzzRequest(f *testing.F) {
	// Add seed corpus
	f.Add(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	f.Add(`{"jsonrpc":"2.0","id":null,"method":"test"}`)
	f.Add(`{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}`)
	f.Add(`{}`)
	f.Add(`[]`)
	f.Add(`null`)
	f.Add(`"string"`)
	f.Add(`123`)

	f.Fuzz(func(t *testing.T, input string) {
		var req Request
		// Should not panic
		_ = json.Unmarshal([]byte(input), &req)

		// If it parsed, try to marshal it back
		if req.Method != "" {
			_, _ = json.Marshal(req)
		}
	})
}

// FuzzResponse tests response parsing with random inputs.
func FuzzResponse(f *testing.F) {
	f.Add(`{"jsonrpc":"2.0","id":1,"result":null}`)
	f.Add(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"test"}}`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, input string) {
		var resp Response
		// Should not panic
		_ = json.Unmarshal([]byte(input), &resp)
	})
}

// FuzzNewRequest tests NewRequest with various inputs.
func FuzzNewRequest(f *testing.F) {
	f.Add(int64(1), "ping", "")
	f.Add(int64(0), "", "params")
	f.Add(int64(-1), "method/with/slashes", `{"nested":{"value":true}}`)

	f.Fuzz(func(t *testing.T, id int64, method string, paramsStr string) {
		var params any
		if paramsStr != "" {
			// Try to use as JSON
			var m map[string]any
			if err := json.Unmarshal([]byte(paramsStr), &m); err == nil {
				params = m
			} else {
				params = paramsStr
			}
		}

		req, err := NewRequest(id, method, params)
		if err == nil && req != nil {
			// Verify it can be serialized
			_, _ = json.Marshal(req)
		}
	})
}

func TestLargeParams(t *testing.T) {
	t.Parallel()
	// Test with a large number of paths
	paths := make([]string, 10000)
	for i := range paths {
		paths[i] = strings.Repeat("a", 100)
	}

	params := WatchStartParams{Paths: paths}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got WatchStartParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(got.Paths) != len(paths) {
		t.Errorf("Got %d paths, want %d", len(got.Paths), len(paths))
	}
}
