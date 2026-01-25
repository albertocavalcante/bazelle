// Package incremental provides incremental BUILD file generation support.
package incremental

// Entry represents a single file's metadata and content hash.
type Entry struct {
	Path    string `json:"path"`
	Hash    string `json:"hash"`     // xxHash64 hex
	ModTime int64  `json:"mtime_ns"` // UnixNano
	Size    int64  `json:"size"`
}
