package python

import (
	"bufio"
	_ "embed"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
)

//go:embed stdlib_modules.txt
var stdlibModulesData string

// stdlibModules is the set of Python standard library module names.
// Initialized lazily on first access.
var (
	stdlibModules     map[string]bool
	stdlibModulesOnce sync.Once
)

// initStdlibModules initializes the stdlib modules set from the embedded data.
func initStdlibModules() {
	stdlibModules = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(stdlibModulesData))
	for scanner.Scan() {
		module := strings.TrimSpace(scanner.Text())
		if module != "" && !strings.HasPrefix(module, "#") {
			stdlibModules[module] = true
		}
	}
}

// IsStdlib checks if a module name is part of the Python standard library.
func IsStdlib(module string) bool {
	stdlibModulesOnce.Do(initStdlibModules)

	// Handle dotted module names - check the top-level module
	if idx := strings.Index(module, "."); idx > 0 {
		module = module[:idx]
	}

	return stdlibModules[module]
}

// GetStdlibModules returns a copy of all stdlib module names.
func GetStdlibModules() []string {
	stdlibModulesOnce.Do(initStdlibModules)

	return slices.Sorted(maps.Keys(stdlibModules))
}

// LoadCustomStdlibModules loads stdlib modules from a custom file.
// This can be used to extend or replace the default list.
func LoadCustomStdlibModules(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// Initialize default modules first if not already done
	stdlibModulesOnce.Do(initStdlibModules)

	// Ensure stdlibModules is initialized before writing
	if stdlibModules == nil {
		stdlibModules = make(map[string]bool)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		module := strings.TrimSpace(scanner.Text())
		if module != "" && !strings.HasPrefix(module, "#") {
			stdlibModules[module] = true
		}
	}

	return scanner.Err()
}

// StdlibModuleCount returns the number of stdlib modules.
func StdlibModuleCount() int {
	stdlibModulesOnce.Do(initStdlibModules)
	return len(stdlibModules)
}
