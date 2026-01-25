package jvm

import "path/filepath"

// DeriveTargetName derives a target name from the directory.
// If the directory is "." or empty, it uses the repo root name.
func DeriveTargetName(dir, repoRoot string) string {
	name := filepath.Base(dir)
	if name == "." || name == "" {
		name = filepath.Base(repoRoot)
	}
	return name
}

// DeriveTestTargetName derives a test target name from the directory.
// It appends "_test" to the base target name.
func DeriveTestTargetName(dir, repoRoot string) string {
	return DeriveTargetName(dir, repoRoot) + "_test"
}

// DeriveLibraryLabel returns a label reference to the library target in the same package.
func DeriveLibraryLabel(dir, repoRoot string) string {
	return ":" + DeriveTargetName(dir, repoRoot)
}
