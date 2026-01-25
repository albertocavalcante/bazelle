package incremental

import (
	"path/filepath"
	"slices"
)

// ChangeSet represents the differences between two indexes.
type ChangeSet struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Deleted  []string `json:"deleted"`
}

// NewChangeSet creates an empty ChangeSet.
func NewChangeSet() *ChangeSet {
	return &ChangeSet{
		Added:    []string{},
		Modified: []string{},
		Deleted:  []string{},
	}
}

// IsEmpty returns true if there are no changes.
func (cs *ChangeSet) IsEmpty() bool {
	if cs == nil {
		return true
	}
	return len(cs.Added) == 0 && len(cs.Modified) == 0 && len(cs.Deleted) == 0
}

// TotalChanges returns the total number of changed files.
func (cs *ChangeSet) TotalChanges() int {
	if cs == nil {
		return 0
	}
	return len(cs.Added) + len(cs.Modified) + len(cs.Deleted)
}

// AffectedDirs returns sorted unique directories containing changes.
func (cs *ChangeSet) AffectedDirs() []string {
	if cs == nil {
		return nil
	}

	dirs := make(map[string]struct{})

	for _, path := range cs.Added {
		dirs[filepath.Dir(path)] = struct{}{}
	}
	for _, path := range cs.Modified {
		dirs[filepath.Dir(path)] = struct{}{}
	}
	for _, path := range cs.Deleted {
		dirs[filepath.Dir(path)] = struct{}{}
	}

	result := make([]string, 0, len(dirs))
	for dir := range dirs {
		result = append(result, dir)
	}
	slices.Sort(result)
	return result
}

// AsTargets converts affected directories to Bazel target patterns.
func (cs *ChangeSet) AsTargets() []string {
	dirs := cs.AffectedDirs()
	targets := make([]string, len(dirs))
	for i, dir := range dirs {
		if dir == "." {
			targets[i] = "//:all"
		} else {
			targets[i] = "//" + dir + ":all"
		}
	}
	return targets
}

// sort sorts all slices for deterministic output.
func (cs *ChangeSet) sort() {
	if cs == nil {
		return
	}
	slices.Sort(cs.Added)
	slices.Sort(cs.Modified)
	slices.Sort(cs.Deleted)
}
