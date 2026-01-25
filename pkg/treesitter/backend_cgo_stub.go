//go:build !cgo

package treesitter

import "fmt"

// ErrCGODisabled is returned when the CGO backend is requested but CGO is disabled.
var ErrCGODisabled = fmt.Errorf("CGO backend is not available: build with CGO_ENABLED=1 or use the wazero backend instead (set %s=wazero)", EnvVarBackend)

// NewCGOBackend returns an error when CGO is not available.
// The CGO backend requires CGO to be enabled at build time.
// To use this backend, rebuild with CGO_ENABLED=1.
// Alternatively, use the wazero backend which does not require CGO.
func NewCGOBackend() (Backend, error) {
	return nil, ErrCGODisabled
}

// NewTreeCursor returns nil when CGO is not available.
// TreeCursor requires the CGO backend.
func NewTreeCursor(node Node) TreeCursor {
	return nil
}
