package incremental

import (
	"time"
)

// IndexVersion is the current version of the index format.
const IndexVersion = 1

// Index represents a snapshot of files in the workspace.
type Index struct {
	Version   int               `json:"version"`
	UpdatedAt time.Time         `json:"updated_at"`
	Entries   map[string]*Entry `json:"entries"`
}

// NewIndex creates an empty index.
func NewIndex() *Index {
	return &Index{
		Version:   IndexVersion,
		UpdatedAt: time.Now(),
		Entries:   make(map[string]*Entry),
	}
}

// Add adds or updates an entry.
func (idx *Index) Add(e *Entry) {
	if idx == nil || e == nil {
		return
	}
	if idx.Entries == nil {
		idx.Entries = make(map[string]*Entry)
	}
	idx.Entries[e.Path] = e
}

// Get retrieves an entry by path.
func (idx *Index) Get(path string) (*Entry, bool) {
	if idx == nil || idx.Entries == nil {
		return nil, false
	}
	e, ok := idx.Entries[path]
	return e, ok
}

// Diff compares this index against another, returning changes.
// The receiver (idx) is the "old" state, other is the "new" state.
func (idx *Index) Diff(other *Index) *ChangeSet {
	cs := NewChangeSet()

	if idx == nil && other == nil {
		return cs
	}

	oldEntries := make(map[string]*Entry)
	newEntries := make(map[string]*Entry)

	if idx != nil && idx.Entries != nil {
		oldEntries = idx.Entries
	}
	if other != nil && other.Entries != nil {
		newEntries = other.Entries
	}

	// Check for new and modified files
	for path, newEntry := range newEntries {
		oldEntry, exists := oldEntries[path]
		if !exists {
			cs.Added = append(cs.Added, path)
			continue
		}

		// Fast path: if mtime and size unchanged, skip hash comparison
		if oldEntry.ModTime == newEntry.ModTime && oldEntry.Size == newEntry.Size {
			continue
		}

		// Hash changed means content changed
		if oldEntry.Hash != newEntry.Hash {
			cs.Modified = append(cs.Modified, path)
		}
	}

	// Check for deleted files
	for path := range oldEntries {
		if _, exists := newEntries[path]; !exists {
			cs.Deleted = append(cs.Deleted, path)
		}
	}

	cs.sort()
	return cs
}
