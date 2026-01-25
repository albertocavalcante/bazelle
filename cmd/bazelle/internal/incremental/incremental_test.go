package incremental

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHashBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty",
			input: []byte{},
			want:  "ef46db3751d8e999", // xxHash64 of empty input
		},
		{
			name:  "hello",
			input: []byte("hello"),
			want:  "26c7827d889f6da3", // xxHash64 of "hello"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashBytes(tt.input)
			if got != tt.want {
				t.Errorf("HashBytes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("file content for hashing")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := HashFile(testFile)
	if err != nil {
		t.Fatalf("HashFile() error = %v", err)
	}

	// Verify it matches HashBytes of the same content
	expected := HashBytes(content)
	if hash != expected {
		t.Errorf("HashFile() = %q, want %q", hash, expected)
	}
}

func TestHashFileNotFound(t *testing.T) {
	_, err := HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("HashFile() expected error for nonexistent file")
	}
}

func TestNewIndex(t *testing.T) {
	idx := NewIndex()
	if idx == nil {
		t.Fatal("NewIndex() returned nil")
	}
	if idx.Version != IndexVersion {
		t.Errorf("Version = %d, want %d", idx.Version, IndexVersion)
	}
	if idx.Entries == nil {
		t.Error("Entries should not be nil")
	}
}

func TestIndexAddGet(t *testing.T) {
	idx := NewIndex()

	entry := &Entry{
		Path:    "test.go",
		Hash:    "abc123",
		ModTime: 1234567890,
		Size:    100,
	}

	idx.Add(entry)

	got, ok := idx.Get("test.go")
	if !ok {
		t.Fatal("Get() should find entry")
	}
	if got.Hash != "abc123" {
		t.Errorf("Hash = %q, want %q", got.Hash, "abc123")
	}

	_, ok = idx.Get("nonexistent.go")
	if ok {
		t.Error("Get() should not find nonexistent entry")
	}
}

func TestIndexAddNilSafety(t *testing.T) {
	var idx *Index
	idx.Add(&Entry{Path: "test.go"}) // Should not panic

	idx = NewIndex()
	idx.Add(nil) // Should not panic
}

func TestIndexGetNilSafety(t *testing.T) {
	var idx *Index
	_, ok := idx.Get("test.go")
	if ok {
		t.Error("Get() on nil index should return false")
	}
}

func TestIndexDiff(t *testing.T) {
	old := NewIndex()
	old.Add(&Entry{Path: "unchanged.go", Hash: "hash1", ModTime: 1000, Size: 10})
	old.Add(&Entry{Path: "modified.go", Hash: "hash2", ModTime: 1000, Size: 20})
	old.Add(&Entry{Path: "deleted.go", Hash: "hash3", ModTime: 1000, Size: 30})

	new := NewIndex()
	new.Add(&Entry{Path: "unchanged.go", Hash: "hash1", ModTime: 1000, Size: 10})
	new.Add(&Entry{Path: "modified.go", Hash: "hash2-changed", ModTime: 2000, Size: 25})
	new.Add(&Entry{Path: "added.go", Hash: "hash4", ModTime: 1000, Size: 40})

	cs := old.Diff(new)

	if len(cs.Added) != 1 || cs.Added[0] != "added.go" {
		t.Errorf("Added = %v, want [added.go]", cs.Added)
	}
	if len(cs.Modified) != 1 || cs.Modified[0] != "modified.go" {
		t.Errorf("Modified = %v, want [modified.go]", cs.Modified)
	}
	if len(cs.Deleted) != 1 || cs.Deleted[0] != "deleted.go" {
		t.Errorf("Deleted = %v, want [deleted.go]", cs.Deleted)
	}
}

func TestIndexDiffNilSafety(t *testing.T) {
	var idx *Index
	cs := idx.Diff(nil)
	if !cs.IsEmpty() {
		t.Error("Diff of nil indexes should be empty")
	}

	idx = NewIndex()
	cs = idx.Diff(nil)
	if !cs.IsEmpty() {
		t.Error("Diff with nil other should be empty")
	}
}

func TestChangeSetIsEmpty(t *testing.T) {
	cs := NewChangeSet()
	if !cs.IsEmpty() {
		t.Error("New ChangeSet should be empty")
	}

	cs.Added = []string{"file.go"}
	if cs.IsEmpty() {
		t.Error("ChangeSet with added files should not be empty")
	}
}

func TestChangeSetIsEmptyNilSafety(t *testing.T) {
	var cs *ChangeSet
	if !cs.IsEmpty() {
		t.Error("nil ChangeSet should be empty")
	}
}

func TestChangeSetTotalChanges(t *testing.T) {
	cs := NewChangeSet()
	if cs.TotalChanges() != 0 {
		t.Error("New ChangeSet should have 0 changes")
	}

	cs.Added = []string{"a.go"}
	cs.Modified = []string{"b.go", "c.go"}
	cs.Deleted = []string{"d.go"}
	if cs.TotalChanges() != 4 {
		t.Errorf("TotalChanges() = %d, want 4", cs.TotalChanges())
	}
}

func TestChangeSetTotalChangesNilSafety(t *testing.T) {
	var cs *ChangeSet
	if cs.TotalChanges() != 0 {
		t.Error("nil ChangeSet should have 0 changes")
	}
}

func TestChangeSetAffectedDirs(t *testing.T) {
	cs := &ChangeSet{
		Added:    []string{"src/main.go"},
		Modified: []string{"src/util.go", "lib/app.go"},
		Deleted:  []string{"pkg/old.go"},
	}

	dirs := cs.AffectedDirs()
	if len(dirs) != 3 {
		t.Errorf("AffectedDirs() = %v, want 3 dirs", dirs)
	}

	// Should be sorted
	expected := []string{"lib", "pkg", "src"}
	for i, dir := range dirs {
		if dir != expected[i] {
			t.Errorf("AffectedDirs()[%d] = %q, want %q", i, dir, expected[i])
		}
	}
}

func TestChangeSetAffectedDirsNilSafety(t *testing.T) {
	var cs *ChangeSet
	dirs := cs.AffectedDirs()
	if dirs != nil {
		t.Error("nil ChangeSet should return nil dirs")
	}
}

func TestChangeSetAsTargets(t *testing.T) {
	tests := []struct {
		name string
		cs   *ChangeSet
		want []string
	}{
		{
			name: "root dir",
			cs:   &ChangeSet{Added: []string{"main.go"}},
			want: []string{"//:all"},
		},
		{
			name: "single dir",
			cs:   &ChangeSet{Added: []string{"src/main.go"}},
			want: []string{"//src:all"},
		},
		{
			name: "multiple dirs",
			cs: &ChangeSet{
				Added:    []string{"src/main.go"},
				Modified: []string{"lib/app.go"},
				Deleted:  []string{"pkg/util/helper.go"},
			},
			want: []string{"//lib:all", "//pkg/util:all", "//src:all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cs.AsTargets()
			if len(got) != len(tt.want) {
				t.Errorf("AsTargets() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("AsTargets()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewScanner(t *testing.T) {
	scanner := NewScanner(ScanConfig{Root: "/tmp/workspace"})
	if scanner.root != "/tmp/workspace" {
		t.Errorf("scanner.root = %q, want %q", scanner.root, "/tmp/workspace")
	}
	// Should have extensions from all languages
	if !scanner.extensions[".go"] {
		t.Error("scanner should include .go extension")
	}
	if !scanner.extensions[".kt"] {
		t.Error("scanner should include .kt extension")
	}
}

func TestNewScannerWithLanguages(t *testing.T) {
	scanner := NewScanner(ScanConfig{
		Root:      "/tmp/workspace",
		Languages: []string{"go", "kotlin"},
	})

	if !scanner.extensions[".go"] {
		t.Error("scanner should include .go extension")
	}
	if !scanner.extensions[".kt"] {
		t.Error("scanner should include .kt extension")
	}
	if scanner.extensions[".java"] {
		t.Error("scanner should not include .java extension")
	}
}

func TestScan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []struct {
		path    string
		content string
	}{
		{"src/main.go", "package main"},
		{"src/util/helper.go", "package util"},
		{"lib/app.kt", "class App"},
		{"README.md", "# Readme"}, // Should be ignored
	}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(f.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(ScanConfig{
		Root:      tmpDir,
		Languages: []string{"go", "kotlin"},
	})
	idx, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Should find Go and Kotlin files
	if _, ok := idx.Get("src/main.go"); !ok {
		t.Error("Scan() should find src/main.go")
	}
	if _, ok := idx.Get("src/util/helper.go"); !ok {
		t.Error("Scan() should find src/util/helper.go")
	}
	if _, ok := idx.Get("lib/app.kt"); !ok {
		t.Error("Scan() should find lib/app.kt")
	}

	// Should not find README.md
	if _, ok := idx.Get("README.md"); ok {
		t.Error("Scan() should not find README.md")
	}
}

func TestScanSkipsIgnoredDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in ignored directories
	files := []struct {
		path    string
		content string
	}{
		{"src/main.go", "package main"},
		{".hidden/secret.go", "package hidden"},
		{"bazel-out/gen.go", "package gen"},
		{"node_modules/dep.go", "package dep"},
	}

	for _, f := range files {
		fullPath := filepath.Join(tmpDir, f.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(f.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(ScanConfig{
		Root:      tmpDir,
		Languages: []string{"go"},
	})
	idx, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// Should only find src/main.go
	if len(idx.Entries) != 1 {
		t.Errorf("Scan() found %d files, want 1", len(idx.Entries))
	}
	if _, ok := idx.Get("src/main.go"); !ok {
		t.Error("Scan() should find src/main.go")
	}
}

func TestScanContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(ScanConfig{Root: tmpDir})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := scanner.Scan(ctx)
	if err == nil {
		t.Error("Scan() should return error when context is cancelled")
	}
}

func TestJSONStoreLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStore(tmpDir)

	// Load should return empty index when file doesn't exist
	idx, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if idx == nil {
		t.Fatal("Load() should not return nil")
	}
	if idx.Version != IndexVersion {
		t.Errorf("Version = %d, want %d", idx.Version, IndexVersion)
	}

	// Add some data
	idx.Add(&Entry{
		Path:    "test.go",
		Hash:    "abc123",
		ModTime: 1234567890,
		Size:    100,
	})

	// Save
	if err := store.Save(idx); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, ".bazelle", "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file should exist after Save()")
	}

	// Create new store and load
	store2 := NewJSONStore(tmpDir)
	idx2, err := store2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify data
	entry, ok := idx2.Get("test.go")
	if !ok {
		t.Error("loaded index should contain test.go")
	} else if entry.Hash != "abc123" {
		t.Errorf("entry.Hash = %q, want %q", entry.Hash, "abc123")
	}
}

func TestJSONStoreExists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStore(tmpDir)

	if store.Exists() {
		t.Error("Exists() should return false when no state file")
	}

	// Create state file
	idx := NewIndex()
	if err := store.Save(idx); err != nil {
		t.Fatal(err)
	}

	if !store.Exists() {
		t.Error("Exists() should return true after Save()")
	}
}

func TestJSONStoreClear(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStore(tmpDir)

	// Create state file
	idx := NewIndex()
	if err := store.Save(idx); err != nil {
		t.Fatal(err)
	}

	if !store.Exists() {
		t.Fatal("state file should exist")
	}

	// Clear
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if store.Exists() {
		t.Error("Exists() should return false after Clear()")
	}
}

func TestTrackerStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial files
	files := map[string]string{
		"unchanged.go": "package unchanged",
		"modified.go":  "package modified",
		"deleted.go":   "package deleted",
	}
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create tracker and refresh to save initial state
	tracker := NewTracker(tmpDir, []string{"go"})
	ctx := context.Background()

	if err := tracker.Refresh(ctx); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Now make changes:
	// 1. Add a new file
	if err := os.WriteFile(filepath.Join(tmpDir, "new.go"), []byte("package new"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 2. Modify a file
	if err := os.WriteFile(filepath.Join(tmpDir, "modified.go"), []byte("package modified // changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 3. Delete a file
	if err := os.Remove(filepath.Join(tmpDir, "deleted.go")); err != nil {
		t.Fatal(err)
	}

	// Check status
	cs, err := tracker.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	// Verify results
	if cs.IsEmpty() {
		t.Error("ChangeSet should not be empty")
	}

	if len(cs.Added) != 1 || cs.Added[0] != "new.go" {
		t.Errorf("Added = %v, want [new.go]", cs.Added)
	}

	if len(cs.Modified) != 1 || cs.Modified[0] != "modified.go" {
		t.Errorf("Modified = %v, want [modified.go]", cs.Modified)
	}

	if len(cs.Deleted) != 1 || cs.Deleted[0] != "deleted.go" {
		t.Errorf("Deleted = %v, want [deleted.go]", cs.Deleted)
	}
}

func TestTrackerStatusNoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	tracker := NewTracker(tmpDir, []string{"go"})
	ctx := context.Background()

	if err := tracker.Refresh(ctx); err != nil {
		t.Fatal(err)
	}

	// Check status without changes
	cs, err := tracker.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !cs.IsEmpty() {
		t.Error("ChangeSet should be empty when no changes")
	}
}

func TestTrackerHasState(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(tmpDir, nil)
	ctx := context.Background()

	if tracker.HasState() {
		t.Error("HasState() should return false before Refresh()")
	}

	if err := tracker.Refresh(ctx); err != nil {
		t.Fatal(err)
	}

	if !tracker.HasState() {
		t.Error("HasState() should return true after Refresh()")
	}
}

func TestTrackerTrackedFileCount(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("package main"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tracker := NewTracker(tmpDir, []string{"go"})
	ctx := context.Background()

	if tracker.TrackedFileCount() != 0 {
		t.Error("TrackedFileCount() should return 0 before Refresh()")
	}

	if err := tracker.Refresh(ctx); err != nil {
		t.Fatal(err)
	}

	if tracker.TrackedFileCount() != 3 {
		t.Errorf("TrackedFileCount() = %d, want 3", tracker.TrackedFileCount())
	}
}
