package jvm

import "testing"

func TestDeriveTargetName(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		repoRoot string
		want     string
	}{
		{
			name:     "normal directory",
			dir:      "/path/to/mylib",
			repoRoot: "/path/to/repo",
			want:     "mylib",
		},
		{
			name:     "dot directory uses repo root",
			dir:      ".",
			repoRoot: "/path/to/myrepo",
			want:     "myrepo",
		},
		{
			name:     "empty directory uses repo root",
			dir:      "",
			repoRoot: "/path/to/myrepo",
			want:     "myrepo",
		},
		{
			name:     "nested directory",
			dir:      "/path/to/deep/nested/module",
			repoRoot: "/path/to/repo",
			want:     "module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveTargetName(tt.dir, tt.repoRoot)
			if got != tt.want {
				t.Errorf("DeriveTargetName(%q, %q) = %q, want %q",
					tt.dir, tt.repoRoot, got, tt.want)
			}
		})
	}
}

func TestDeriveTestTargetName(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		repoRoot string
		want     string
	}{
		{
			name:     "normal directory",
			dir:      "/path/to/mylib",
			repoRoot: "/path/to/repo",
			want:     "mylib_test",
		},
		{
			name:     "dot directory uses repo root",
			dir:      ".",
			repoRoot: "/path/to/myrepo",
			want:     "myrepo_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveTestTargetName(tt.dir, tt.repoRoot)
			if got != tt.want {
				t.Errorf("DeriveTestTargetName(%q, %q) = %q, want %q",
					tt.dir, tt.repoRoot, got, tt.want)
			}
		})
	}
}

func TestDeriveLibraryLabel(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		repoRoot string
		want     string
	}{
		{
			name:     "normal directory",
			dir:      "/path/to/mylib",
			repoRoot: "/path/to/repo",
			want:     ":mylib",
		},
		{
			name:     "dot directory uses repo root",
			dir:      ".",
			repoRoot: "/path/to/myrepo",
			want:     ":myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveLibraryLabel(tt.dir, tt.repoRoot)
			if got != tt.want {
				t.Errorf("DeriveLibraryLabel(%q, %q) = %q, want %q",
					tt.dir, tt.repoRoot, got, tt.want)
			}
		})
	}
}
