package treesitter

import (
	"fmt"
	"os"
	"strings"
)

// BackendType identifies a specific tree-sitter backend implementation.
type BackendType string

const (
	// BackendAuto automatically selects the best available backend.
	// It tries CGO first (production-ready, broad language support),
	// then falls back to wazero (CGO-free, limited language support).
	BackendAuto BackendType = "auto"

	// BackendCGO uses the CGO-based backend (smacker/go-tree-sitter).
	// This is the production-ready backend with broad language support.
	BackendCGO BackendType = "cgo"

	// BackendWazero uses the WASM/wazero backend (malivvan/tree-sitter).
	// This is experimental and has limited language support (C, C++ only).
	BackendWazero BackendType = "wazero"
)

// EnvVarBackend is the environment variable used to select the backend.
const EnvVarBackend = "BAZELLE_TREESITTER_BACKEND"

// NewBackend creates a backend of the specified type.
// For BackendAuto, it tries CGO first, then falls back to wazero.
func NewBackend(typ BackendType) (Backend, error) {
	switch typ {
	case BackendCGO:
		return NewCGOBackend()
	case BackendWazero:
		return NewWazeroBackend()
	case BackendAuto:
		// Try CGO first as it's production-ready with broad language support
		if b, err := NewCGOBackend(); err == nil {
			return b, nil
		}
		// Fall back to wazero for CGO-free environments
		return NewWazeroBackend()
	default:
		return nil, fmt.Errorf("unknown backend type: %s", typ)
	}
}

// NewBackendFromEnv creates a backend based on the BAZELLE_TREESITTER_BACKEND
// environment variable. If the variable is not set or empty, it defaults to
// BackendAuto.
//
// Valid values are:
//   - "auto" (default): Try CGO first, fall back to wazero
//   - "cgo": Use the CGO backend
//   - "wazero": Use the wazero/WASM backend
func NewBackendFromEnv() (Backend, error) {
	envVal := strings.TrimSpace(os.Getenv(EnvVarBackend))
	if envVal == "" {
		return NewBackend(BackendAuto)
	}

	typ := BackendType(strings.ToLower(envVal))
	switch typ {
	case BackendAuto, BackendCGO, BackendWazero:
		return NewBackend(typ)
	default:
		return nil, fmt.Errorf("invalid %s value %q: must be one of auto, cgo, wazero", EnvVarBackend, envVal)
	}
}

// MustNewBackend is like NewBackend but panics on error.
// This is useful for initialization code that cannot handle errors.
func MustNewBackend(typ BackendType) Backend {
	b, err := NewBackend(typ)
	if err != nil {
		panic(fmt.Sprintf("failed to create tree-sitter backend %s: %v", typ, err))
	}
	return b
}

// MustNewBackendFromEnv is like NewBackendFromEnv but panics on error.
func MustNewBackendFromEnv() Backend {
	b, err := NewBackendFromEnv()
	if err != nil {
		panic(fmt.Sprintf("failed to create tree-sitter backend from env: %v", err))
	}
	return b
}

// AvailableBackends returns a list of backend types that can be created
// successfully in the current environment.
func AvailableBackends() []BackendType {
	var available []BackendType

	if b, err := NewCGOBackend(); err == nil {
		_ = b.Close()
		available = append(available, BackendCGO)
	}

	if b, err := NewWazeroBackend(); err == nil {
		_ = b.Close()
		available = append(available, BackendWazero)
	}

	return available
}

// BackendInfo provides information about a backend type.
type BackendInfo struct {
	// Type is the backend type identifier.
	Type BackendType

	// Name is the human-readable name.
	Name string

	// Description provides details about the backend.
	Description string

	// IsExperimental indicates if the backend is production-ready.
	IsExperimental bool

	// SupportedLanguages lists languages the backend can parse.
	SupportedLanguages []Language
}

// GetBackendInfo returns information about a backend type without creating it.
func GetBackendInfo(typ BackendType) BackendInfo {
	switch typ {
	case BackendCGO:
		return BackendInfo{
			Type:           BackendCGO,
			Name:           "CGO",
			Description:    "Production-ready backend using smacker/go-tree-sitter with CGO bindings",
			IsExperimental: false,
			SupportedLanguages: []Language{
				Go, Java, Kotlin, Scala, Rust, Python, JavaScript, TypeScript, TSX,
				Groovy, C, Cpp, CSharp, Ruby, PHP, Swift, Bash, HTML, CSS, SQL,
				YAML, TOML, Markdown, Protobuf, HCL, Dockerfile, Lua, Elixir,
				Elm, OCaml, Svelte, Cue,
			},
		}
	case BackendWazero:
		return BackendInfo{
			Type:           BackendWazero,
			Name:           "Wazero",
			Description:    "Experimental CGO-free backend using malivvan/tree-sitter with WASM/wazero",
			IsExperimental: true,
			SupportedLanguages: []Language{
				C, Cpp,
			},
		}
	default:
		return BackendInfo{
			Type:        typ,
			Name:        string(typ),
			Description: "Unknown backend type",
		}
	}
}
