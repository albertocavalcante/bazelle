// Package detect provides language detection for bazel projects.
package detect

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// Language extensions mapping
var langExtensions = map[string][]string{
	"go":     {".go"},
	"kotlin": {".kt", ".kts"},
	"java":   {".java"},
	"python": {".py"},
	"proto":  {".proto"},
	"groovy": {".groovy"},
	"cc":     {".cc", ".cpp", ".cxx", ".c", ".h", ".hpp", ".hxx"},
}

// ignoredDirs contains directory prefixes to skip during scanning.
var ignoredDirs = []string{
	"bazel-",
	".",
	"node_modules",
	"__pycache__",
	"vendor",
}

// Languages detects programming languages used in the given directory.
// It returns a sorted slice of language identifiers.
func Languages(root string) ([]string, error) {
	found := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			name := d.Name()
			for _, prefix := range ignoredDirs {
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

		for lang, exts := range langExtensions {
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
