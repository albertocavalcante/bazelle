package incremental

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// stateDir is the directory name for bazelle state files.
	stateDir = ".bazelle"

	// stateFile is the name of the state file.
	stateFile = "state.json"
)

// Store defines the interface for index persistence.
type Store interface {
	Load() (*Index, error)
	Save(idx *Index) error
	Exists() bool
	Clear() error
}

// JSONStore implements Store using JSON files.
type JSONStore struct {
	dir  string
	path string
}

// NewJSONStore creates a store at the given directory.
// Creates .bazelle/state.json within the directory.
func NewJSONStore(workspaceRoot string) *JSONStore {
	dir := filepath.Join(workspaceRoot, stateDir)
	path := filepath.Join(dir, stateFile)
	return &JSONStore{
		dir:  dir,
		path: path,
	}
}

// Load reads the index from disk. If the state file doesn't exist, returns an empty index.
func (s *JSONStore) Load() (*Index, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return NewIndex(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Check version compatibility
	if idx.Version > IndexVersion {
		return nil, fmt.Errorf("state file version %d is newer than supported version %d", idx.Version, IndexVersion)
	}

	// Ensure entries map is initialized
	if idx.Entries == nil {
		idx.Entries = make(map[string]*Entry)
	}

	return &idx, nil
}

// Save writes the index to disk atomically.
func (s *JSONStore) Save(idx *Index) error {
	if idx == nil {
		return fmt.Errorf("cannot save nil index")
	}

	// Ensure state directory exists
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Update timestamp and version
	idx.UpdatedAt = time.Now()
	idx.Version = IndexVersion

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write to temp file first for atomic update
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Rename temp file to actual file (atomic on POSIX)
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// Exists returns true if the state file exists.
func (s *JSONStore) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Clear removes the state file and directory.
func (s *JSONStore) Clear() error {
	return os.RemoveAll(s.dir)
}
