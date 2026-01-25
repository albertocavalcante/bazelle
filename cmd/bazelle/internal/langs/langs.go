// Package langs provides shared language configuration for bazelle.
package langs

// Extensions maps language names to their file extensions.
// This is the single source of truth for file extension filtering
// across scanner and watcher components.
var Extensions = map[string][]string{
	"go":     {".go"},
	"kotlin": {".kt", ".kts"},
	"java":   {".java"},
	"python": {".py"},
	"proto":  {".proto"},
	"groovy": {".groovy"},
	"scala":  {".scala", ".sc"},
	"cc":     {".cc", ".cpp", ".cxx", ".c", ".h", ".hpp", ".hxx"},
	"rust":   {".rs"},
}

// IgnoredDirs contains directory prefixes to skip during scanning/watching.
// Directories starting with any of these prefixes will be excluded.
var IgnoredDirs = []string{
	"bazel-",      // Bazel output directories
	".",           // Hidden directories
	"node_modules", // Node.js dependencies
	"__pycache__", // Python cache
	"vendor",      // Go vendor, other vendored deps
	"target",      // Rust/Maven target
	"build",       // Gradle/generic build output
	"out",         // Generic output
	"dist",        // Distribution output
	".bazelle",    // Bazelle state directory
}

// ExtensionSet returns a set of all extensions for the given languages.
// If languages is nil or empty, returns all known extensions.
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
