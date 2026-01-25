package incremental

import (
	"context"
	"fmt"
	"path/filepath"
)

// Tracker provides high-level incremental update tracking.
type Tracker struct {
	store   Store
	scanner *Scanner
	root    string
}

// NewTracker creates a tracker for the given workspace.
func NewTracker(workspaceRoot string, languages []string) *Tracker {
	store := NewJSONStore(workspaceRoot)
	scanner := NewScanner(ScanConfig{
		Root:      workspaceRoot,
		Languages: languages,
	})

	return &Tracker{
		store:   store,
		scanner: scanner,
		root:    workspaceRoot,
	}
}

// Status checks for changes without modifying state.
// Returns a ChangeSet describing what has changed since the last Refresh.
func (t *Tracker) Status(ctx context.Context) (*ChangeSet, error) {
	// Load stored index
	oldIdx, err := t.store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Scan current state - use fast scan first
	fastIdx, err := t.scanner.ScanFast(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	// Compute changes, hashing only files that might have changed
	cs := t.computeChangesWithLazyHash(ctx, oldIdx, fastIdx)
	return cs, nil
}

// computeChangesWithLazyHash computes changes, only hashing files when needed.
func (t *Tracker) computeChangesWithLazyHash(ctx context.Context, oldIdx, fastIdx *Index) *ChangeSet {
	cs := NewChangeSet()

	oldEntries := make(map[string]*Entry)
	newEntries := make(map[string]*Entry)

	if oldIdx != nil && oldIdx.Entries != nil {
		oldEntries = oldIdx.Entries
	}
	if fastIdx != nil && fastIdx.Entries != nil {
		newEntries = fastIdx.Entries
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

		// Need to hash to check if content actually changed
		hash, err := HashFile(filepath.Join(t.root, path))
		if err != nil {
			// If we can't hash, assume modified
			cs.Modified = append(cs.Modified, path)
			continue
		}

		if oldEntry.Hash != hash {
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

// Refresh updates the stored index from current disk state.
func (t *Tracker) Refresh(ctx context.Context) error {
	// Scan current state with full hashing
	idx, err := t.scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan workspace: %w", err)
	}

	// Save to store
	if err := t.store.Save(idx); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// HasState returns true if a previous state exists.
func (t *Tracker) HasState() bool {
	return t.store.Exists()
}

// TrackedFileCount returns the number of files in the current stored index.
// Returns 0 if no state exists or on error.
func (t *Tracker) TrackedFileCount() int {
	idx, err := t.store.Load()
	if err != nil {
		return 0
	}
	if idx == nil || idx.Entries == nil {
		return 0
	}
	return len(idx.Entries)
}
