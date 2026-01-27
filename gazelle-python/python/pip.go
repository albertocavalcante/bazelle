package python

import (
	"bufio"
	_ "embed"
	"os"
	"regexp"
	"strings"
	"sync"
)

//go:embed pip_known_mappings.txt
var pipKnownMappingsData string

// Lazily initialized pip mappings
var (
	pipToModuleMappings     map[string]string
	pipToModuleMappingsOnce sync.Once
)

// PipDependency represents a dependency from requirements.txt.
type PipDependency struct {
	// Name is the pip package name (e.g., "requests", "scikit-learn").
	Name string

	// Version is the version specifier (e.g., "==1.2.3", ">=2.0", "").
	Version string

	// Extras is the list of extras (e.g., ["security"] for requests[security]).
	Extras []string

	// ModuleName is the Python module name if different from pip name.
	ModuleName string
}

// RequirementsParser parses requirements.txt files.
type RequirementsParser struct {
	// Regex patterns for parsing requirements
	requirementRegex *regexp.Regexp
}

// NewRequirementsParser creates a new requirements parser.
func NewRequirementsParser() *RequirementsParser {
	return &RequirementsParser{
		// Matches: package, package==version, package>=version, package[extras]>=version
		// Captures: name, extras (optional), version specifier (optional)
		requirementRegex: regexp.MustCompile(`^([a-zA-Z0-9][-a-zA-Z0-9._]*)(?:\[([^\]]+)\])?(.*)$`),
	}
}

// ParseFile parses a requirements.txt file.
func (p *RequirementsParser) ParseFile(path string) ([]PipDependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var deps []PipDependency
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip -r, -e, --index-url, etc.
		if strings.HasPrefix(line, "-") {
			continue
		}

		// Skip URLs (git+, http://, etc.)
		if strings.Contains(line, "://") || strings.HasPrefix(line, "git+") {
			continue
		}

		dep := p.parseLine(line)
		if dep.Name != "" {
			deps = append(deps, dep)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return deps, nil
}

// parseLine parses a single requirement line.
func (p *RequirementsParser) parseLine(line string) PipDependency {
	// Remove inline comments
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}

	// Remove environment markers (e.g., ; python_version >= "3.7")
	if idx := strings.Index(line, ";"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}

	matches := p.requirementRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return PipDependency{}
	}

	dep := PipDependency{
		Name:       strings.ToLower(matches[1]),
		ModuleName: PipToModule(matches[1]),
	}

	// Parse extras if present
	if len(matches) > 2 && matches[2] != "" {
		extras := strings.Split(matches[2], ",")
		for i, extra := range extras {
			extras[i] = strings.TrimSpace(extra)
		}
		dep.Extras = extras
	}

	// Parse version specifier if present
	if len(matches) > 3 && matches[3] != "" {
		dep.Version = strings.TrimSpace(matches[3])
	}

	return dep
}

// initPipToModuleMappings initializes the pip-to-module mappings from embedded data.
func initPipToModuleMappings() {
	pipToModuleMappings = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(pipKnownMappingsData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			pipName := strings.TrimSpace(parts[0])
			moduleName := strings.TrimSpace(parts[1])
			pipToModuleMappings[strings.ToLower(pipName)] = moduleName
		}
	}
}

// getPipToModuleMappings returns the pip-to-module mappings, initializing lazily.
func getPipToModuleMappings() map[string]string {
	pipToModuleMappingsOnce.Do(initPipToModuleMappings)
	return pipToModuleMappings
}

// PipToModule converts a pip package name to its Python module name.
//
// Many pip packages have different names than their Python modules:
//   - scikit-learn -> sklearn
//   - Pillow -> PIL
//   - PyYAML -> yaml
//
// If no mapping is found, returns the normalized pip name (lowercase, hyphens to underscores).
func PipToModule(pipName string) string {
	mappings := getPipToModuleMappings()

	// Check for explicit mapping
	normalized := strings.ToLower(pipName)
	if moduleName, ok := mappings[normalized]; ok {
		return moduleName
	}

	// Default: convert hyphens to underscores
	return strings.ReplaceAll(normalized, "-", "_")
}

// ModuleToPip converts a Python module name to its likely pip package name.
//
// This is a best-effort reverse mapping. Not all modules have pip packages,
// and some pip packages provide multiple modules.
func ModuleToPip(moduleName string) string {
	mappings := getPipToModuleMappings()

	// Search for reverse mapping
	for pipName, modName := range mappings {
		if modName == moduleName {
			return pipName
		}
	}

	// Default: convert underscores to hyphens
	return strings.ReplaceAll(moduleName, "_", "-")
}

// IsPipPackage checks if a module name corresponds to a known pip package.
func IsPipPackage(moduleName string) bool {
	// If it's in our mappings, it's definitely a pip package
	mappings := getPipToModuleMappings()
	for _, modName := range mappings {
		if modName == moduleName {
			return true
		}
	}

	// Otherwise, we can't be sure
	return false
}

// PipConfig holds pip-specific configuration.
type PipConfig struct {
	// RequirementsFile is the path to requirements.txt.
	RequirementsFile string

	// PipRepository is the name of the pip repository rule (e.g., "pip", "pypi").
	PipRepository string

	// Dependencies is the parsed list of pip dependencies.
	Dependencies []PipDependency
}

// NewPipConfig creates a new PipConfig with default values.
func NewPipConfig() *PipConfig {
	return &PipConfig{
		RequirementsFile: "requirements.txt",
		PipRepository:    "pip",
	}
}

// LoadDependencies loads and parses the requirements file.
func (c *PipConfig) LoadDependencies(repoRoot string) error {
	if c.RequirementsFile == "" {
		return nil
	}

	parser := NewRequirementsParser()
	deps, err := parser.ParseFile(c.RequirementsFile)
	if err != nil {
		// Requirements file doesn't exist or couldn't be parsed
		// This is not necessarily an error - project might not use pip
		return nil
	}

	c.Dependencies = deps
	return nil
}

// GetPipLabel returns the Bazel label for a pip package.
func (c *PipConfig) GetPipLabel(moduleName string) string {
	pipName := ModuleToPip(moduleName)

	// Check if this module is in our dependencies
	for _, dep := range c.Dependencies {
		if dep.ModuleName == moduleName || dep.Name == pipName {
			// Return label in the format @pip//package_name
			return "@" + c.PipRepository + "//" + strings.ReplaceAll(dep.Name, "-", "_")
		}
	}

	return ""
}

// IsKnownPipDependency checks if a module is a known pip dependency.
func (c *PipConfig) IsKnownPipDependency(moduleName string) bool {
	pipName := ModuleToPip(moduleName)

	for _, dep := range c.Dependencies {
		if dep.ModuleName == moduleName || dep.Name == pipName {
			return true
		}
	}

	return false
}
