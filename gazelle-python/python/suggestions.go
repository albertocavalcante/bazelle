package python

import (
	"fmt"
	"strings"
	"sync"
)

// UnresolvedImport represents an import that couldn't be resolved.
type UnresolvedImport struct {
	// Module is the Python module name that couldn't be resolved.
	Module string

	// File is the file where the import was found.
	File string

	// SuggestedPipPackage is the likely pip package name, if known.
	SuggestedPipPackage string
}

// SuggestionTracker tracks unresolved imports and provides suggestions.
//
// This is useful for helping users identify missing pip dependencies
// or local packages that need BUILD rules.
type SuggestionTracker struct {
	mu          sync.Mutex
	unresolved  []UnresolvedImport
	seenModules map[string]bool
}

// NewSuggestionTracker creates a new SuggestionTracker.
func NewSuggestionTracker() *SuggestionTracker {
	return &SuggestionTracker{
		seenModules: make(map[string]bool),
	}
}

// Track records an unresolved import.
func (t *SuggestionTracker) Track(module, file string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Deduplicate by module
	if t.seenModules[module] {
		return
	}
	t.seenModules[module] = true

	// Try to suggest a pip package
	suggested := suggestPipPackage(module)

	t.unresolved = append(t.unresolved, UnresolvedImport{
		Module:              module,
		File:                file,
		SuggestedPipPackage: suggested,
	})
}

// GetUnresolved returns all unresolved imports.
func (t *SuggestionTracker) GetUnresolved() []UnresolvedImport {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Return a copy
	result := make([]UnresolvedImport, len(t.unresolved))
	copy(result, t.unresolved)
	return result
}

// GetSuggestions returns a formatted list of suggestions for requirements.txt.
func (t *SuggestionTracker) GetSuggestions() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var suggestions []string
	seen := make(map[string]bool)

	for _, u := range t.unresolved {
		if u.SuggestedPipPackage != "" && !seen[u.SuggestedPipPackage] {
			seen[u.SuggestedPipPackage] = true
			suggestions = append(suggestions, u.SuggestedPipPackage)
		}
	}

	return suggestions
}

// FormatSuggestions returns a human-readable summary of suggestions.
func (t *SuggestionTracker) FormatSuggestions() string {
	suggestions := t.GetSuggestions()
	if len(suggestions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Suggested additions to requirements.txt:\n")
	for _, pkg := range suggestions {
		sb.WriteString(fmt.Sprintf("%s\n", pkg))
	}
	return sb.String()
}

// HasUnresolved returns true if there are any unresolved imports.
func (t *SuggestionTracker) HasUnresolved() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.unresolved) > 0
}

// Clear removes all tracked imports.
func (t *SuggestionTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.unresolved = nil
	t.seenModules = make(map[string]bool)
}

// suggestPipPackage attempts to suggest a pip package name for a module.
func suggestPipPackage(module string) string {
	// First, check known mappings (reverse lookup)
	mappings := getPipToModuleMappings()
	for pipName, moduleName := range mappings {
		if moduleName == module {
			return pipName
		}
	}

	// For submodules, try the top-level module
	if idx := strings.Index(module, "."); idx > 0 {
		topLevel := module[:idx]
		for pipName, moduleName := range mappings {
			if moduleName == topLevel {
				return pipName
			}
		}
	}

	// Common heuristic: module names with underscores often have hyphens in pip
	return strings.ReplaceAll(module, "_", "-")
}

// WellKnownPackages contains suggestions for commonly used but differently named packages.
var WellKnownPackages = map[string]string{
	"cv2":                "opencv-python",
	"sklearn":            "scikit-learn",
	"PIL":                "Pillow",
	"yaml":               "PyYAML",
	"bs4":                "beautifulsoup4",
	"dateutil":           "python-dateutil",
	"dotenv":             "python-dotenv",
	"google.cloud":       "google-cloud-core",
	"google.auth":        "google-auth",
	"googleapiclient":    "google-api-python-client",
	"tensorflow":         "tensorflow",
	"torch":              "torch",
	"torchvision":        "torchvision",
	"transformers":       "transformers",
	"numpy":              "numpy",
	"pandas":             "pandas",
	"scipy":              "scipy",
	"matplotlib":         "matplotlib",
	"seaborn":            "seaborn",
	"plotly":             "plotly",
	"flask":              "Flask",
	"fastapi":            "fastapi",
	"django":             "Django",
	"sqlalchemy":         "SQLAlchemy",
	"requests":           "requests",
	"httpx":              "httpx",
	"aiohttp":            "aiohttp",
	"boto3":              "boto3",
	"botocore":           "botocore",
	"celery":             "celery",
	"redis":              "redis",
	"pymongo":            "pymongo",
	"psycopg2":           "psycopg2-binary",
	"pytest":             "pytest",
	"unittest":           "",                  // stdlib
	"click":              "click",
	"typer":              "typer",
	"pydantic":           "pydantic",
	"attrs":              "attrs",
	"mypy":               "mypy",
	"black":              "black",
	"ruff":               "ruff",
	"isort":              "isort",
}

// SuggestFromWellKnown attempts to suggest a pip package from well-known packages.
func SuggestFromWellKnown(module string) string {
	// Direct match
	if pkg, ok := WellKnownPackages[module]; ok {
		return pkg
	}

	// Try top-level module
	if idx := strings.Index(module, "."); idx > 0 {
		topLevel := module[:idx]
		if pkg, ok := WellKnownPackages[topLevel]; ok {
			return pkg
		}
	}

	return ""
}
