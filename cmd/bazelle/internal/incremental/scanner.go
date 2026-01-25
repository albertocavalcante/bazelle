package incremental

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/langs"
)

// ScanConfig configures the scanner.
type ScanConfig struct {
	Root       string
	Languages  []string // nil = all languages
	IgnoreDirs []string // Additional dirs to ignore
}

// Scanner builds an Index by walking the filesystem.
type Scanner struct {
	root       string
	ignoreDirs []string
	extensions map[string]bool
}

// NewScanner creates a scanner with the given config.
func NewScanner(cfg ScanConfig) *Scanner {
	// Build extension filter from shared config
	extensions := langs.ExtensionSet(cfg.Languages)

	// Combine default and custom ignored dirs
	ignoreDirs := make([]string, len(langs.IgnoredDirs))
	copy(ignoreDirs, langs.IgnoredDirs)
	ignoreDirs = append(ignoreDirs, cfg.IgnoreDirs...)

	return &Scanner{
		root:       cfg.Root,
		ignoreDirs: ignoreDirs,
		extensions: extensions,
	}
}

// Scan walks the filesystem and builds an Index.
func (s *Scanner) Scan(ctx context.Context) (*Index, error) {
	idx := NewIndex()

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			name := d.Name()
			for _, prefix := range s.ignoreDirs {
				if strings.HasPrefix(name, prefix) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file has a tracked extension
		ext := filepath.Ext(path)
		if !s.extensions[ext] {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Compute relative path
		relPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}

		// Get file hash
		hash, err := HashFile(path)
		if err != nil {
			return err
		}

		// Create entry
		entry := &Entry{
			Path:    relPath,
			Hash:    hash,
			ModTime: info.ModTime().UnixNano(),
			Size:    info.Size(),
		}

		idx.Add(entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return idx, nil
}

// ScanFast performs a fast scan that only checks mtime/size without hashing.
// This is useful for quickly detecting if a full scan is needed.
func (s *Scanner) ScanFast(ctx context.Context) (*Index, error) {
	idx := NewIndex()

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			name := d.Name()
			for _, prefix := range s.ignoreDirs {
				if strings.HasPrefix(name, prefix) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file has a tracked extension
		ext := filepath.Ext(path)
		if !s.extensions[ext] {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Compute relative path
		relPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}

		// Create entry without hash
		entry := &Entry{
			Path:    relPath,
			ModTime: info.ModTime().UnixNano(),
			Size:    info.Size(),
		}

		idx.Add(entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return idx, nil
}
