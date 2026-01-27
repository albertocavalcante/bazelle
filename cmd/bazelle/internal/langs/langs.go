// Package langs provides shared language configuration for bazelle.
//
// # Single Source of Truth
//
// This package defines the DETERMINISTIC mapping between language names and
// file extensions used across bazelle. All components that need to identify
// source files by extension should use this package rather than defining
// their own mappings.
//
// The mappings are deterministic: given a language name, you always get the
// same set of extensions. There is no heuristic or guessing involved.
//
// # Usage
//
// Components should use ExtensionSet() to get a set of extensions for filtering:
//
//	exts := langs.ExtensionSet([]string{"kotlin", "java"})
//	if exts[filepath.Ext(file)] {
//	    // file is a Kotlin or Java source file
//	}
//
// # Adding New Languages
//
// To add support for a new language:
//  1. Add an entry to the Extensions map below
//  2. Ensure the extension list is complete for the language
//  3. Update any language-specific gazelle extensions
package langs

// Extensions maps language names to their file extensions.
//
// This is the SINGLE SOURCE OF TRUTH for file extension filtering
// across scanner and watcher components. Other packages (detect, jvm)
// should reference this map rather than duplicating it.
//
// DETERMINISTIC: The same language always maps to the same extensions.
var Extensions = map[string][]string{
	"go":     {".go"},
	"kotlin": {".kt", ".kts"},
	"java":   {".java"},
	"python": {".py"},
	"proto":  {".proto"},
	"groovy": {".groovy", ".gvy", ".gy", ".gsh"},
	"scala":  {".scala", ".sc"},
	"cc":     {".cc", ".cpp", ".cxx", ".c", ".h", ".hpp", ".hxx"},
	"rust":   {".rs"},
}

// IgnoredDirs contains directory prefixes to skip during scanning/watching.
//
// These patterns are DETERMINISTIC: any directory starting with one of
// these prefixes will be excluded. This is used to skip build outputs,
// vendored dependencies, and other non-source directories.
//
// Note: Prefix matching means "bazel-" matches "bazel-out", "bazel-bin", etc.
var IgnoredDirs = []string{
	"bazel-",       // Bazel output directories
	".",            // Hidden directories
	"node_modules", // Node.js dependencies
	"__pycache__",  // Python cache
	"vendor",       // Go vendor, other vendored deps
	"target",       // Rust/Maven target
	"build",        // Gradle/generic build output
	"out",          // Generic output
	"dist",         // Distribution output
	".bazelle",     // Bazelle state directory
}

// ExtensionSet returns a set of all extensions for the given languages.
//
// If languages is nil or empty, returns all known extensions.
//
// This is a DETERMINISTIC function: given the same language list, it always
// returns the same extension set.
func ExtensionSet(languages []string) map[string]bool {
	extensions := make(map[string]bool)

	if len(languages) == 0 {
		// Use all known extensions
		for _, exts := range Extensions {
			for _, ext := range exts {
				extensions[ext] = true
			}
		}
	} else {
		// Use only specified languages
		for _, lang := range languages {
			if exts, ok := Extensions[lang]; ok {
				for _, ext := range exts {
					extensions[ext] = true
				}
			}
		}
	}

	return extensions
}

// IgnoreDirSet returns a set of ignored directory prefixes,
// combining defaults with any additional patterns.
func IgnoreDirSet(additional []string) map[string]bool {
	dirs := make(map[string]bool)
	for _, dir := range IgnoredDirs {
		dirs[dir] = true
	}
	for _, dir := range additional {
		dirs[dir] = true
	}
	return dirs
}
