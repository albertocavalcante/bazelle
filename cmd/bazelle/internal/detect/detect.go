// Package detect provides language detection for bazel projects.
//
// # Detection Algorithm
//
// Language detection is DETERMINISTIC: given the same directory contents,
// it always produces the same list of detected languages. The algorithm:
//
//  1. Walk the directory tree, skipping ignored directories
//  2. For each file, check if its extension matches a known language
//  3. Return the deduplicated, sorted list of detected languages
//
// # Extension Mapping
//
// This package uses langs.Extensions as the source of truth for mapping
// file extensions to language names. See the langs package for the
// complete list of supported extensions.
//
// # Ignored Directories
//
// Certain directories are skipped during detection to avoid false positives
// from build outputs, vendored code, and generated files. See langs.IgnoredDirs.
package detect

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/langs"
)

// Languages detects programming languages used in the given directory.
//
// This function is DETERMINISTIC: given the same directory contents,
// it always returns the same sorted list of language identifiers.
//
// The detection is based purely on file extensions, not file contents.
// A language is detected if at least one file with a matching extension exists.
//
// Extension mappings are defined in langs.Extensions (single source of truth).
//
// Returns a sorted slice of language identifiers (e.g., ["go", "kotlin", "proto"]).
func Languages(root string) ([]string, error) {
	found := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories (using shared list from langs package)
		if d.IsDir() {
			name := d.Name()
			for _, prefix := range langs.IgnoredDirs {
				if strings.HasPrefix(name, prefix) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check file extension against known languages
		ext := filepath.Ext(path)
		if ext == "" {
			return nil
		}

		// Use langs.Extensions as single source of truth
		for lang, exts := range langs.Extensions {
			if slices.Contains(exts, ext) {
				found[lang] = true
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to sorted slice
	result := make([]string, 0, len(found))
	for lang := range found {
		result = append(result, lang)
	}
	slices.Sort(result)

	return result, nil
}

// HasLanguage checks if a specific language is detected in the directory.
func HasLanguage(root, lang string) (bool, error) {
	langs, err := Languages(root)
	if err != nil {
		return false, err
	}
	return slices.Contains(langs, lang), nil
}
