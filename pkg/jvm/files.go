package jvm

import (
	"os"
	"path/filepath"
	"strings"
)

// FindSourceFiles finds all source files with the given extensions under a subdirectory.
// The returned paths are relative to baseDir.
func FindSourceFiles(baseDir, subDir string, extensions []string) []string {
	dir := filepath.Join(baseDir, subDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	extSet := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		extSet[ext] = true
	}

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			if extSet[ext] {
				relPath, _ := filepath.Rel(baseDir, path)
				files = append(files, relPath)
			}
		}
		return nil
	})

	return files
}

// FindLanguageFiles finds all source files for a language under a subdirectory.
func FindLanguageFiles(baseDir, subDir string, lang Language) []string {
	return FindSourceFiles(baseDir, subDir, lang.FileExtensions())
}

// FindMainSources finds main source files using the standard directory layout.
func FindMainSources(baseDir string, lang Language) []string {
	return FindLanguageFiles(baseDir, lang.MainSourceDir(), lang)
}

// FindTestSources finds test source files using the standard directory layout.
func FindTestSources(baseDir string, lang Language) []string {
	return FindLanguageFiles(baseDir, lang.TestSourceDir(), lang)
}

// IsSourceDir checks if a directory contains source files for the given language.
func IsSourceDir(dir string, lang Language) bool {
	return strings.Contains(dir, lang.MainSourceDir()) ||
		strings.Contains(dir, lang.TestSourceDir())
}

// IsTestDir checks if a directory is a test directory.
func IsTestDir(dir string) bool {
	return strings.Contains(dir, filepath.Join("src", "test"))
}

// IsMainDir checks if a directory is a main source directory.
func IsMainDir(dir string) bool {
	return strings.Contains(dir, filepath.Join("src", "main"))
}

// HasSourceFiles checks if any source files exist in the given directory.
func HasSourceFiles(baseDir, subDir string, lang Language) bool {
	return len(FindLanguageFiles(baseDir, subDir, lang)) > 0
}
