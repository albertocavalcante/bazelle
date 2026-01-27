package groovy

import (
	"bufio"
	"strings"
)

// IsGroovyStdlib checks if an import is part of the Groovy/Java standard library.
//
// This function returns true for packages that are:
//   - Part of the Groovy standard library (groovy.*)
//   - Part of the Java standard library (java.*, javax.*)
//   - Part of the Groovy runtime (org.codehaus.groovy.*)
//
// These packages are always available on the classpath and don't need
// explicit dependency declarations.
func IsGroovyStdlib(imp string) bool {
	stdlibPrefixes := getGroovyStdlibPrefixes()

	// Extract the top-level package prefix
	parts := strings.Split(imp, ".")
	if len(parts) == 0 {
		return false
	}

	prefix := parts[0]
	if stdlibPrefixes[prefix] {
		return true
	}

	// Check for org.codehaus.groovy specifically
	if len(parts) >= 3 && parts[0] == "org" && parts[1] == "codehaus" && parts[2] == "groovy" {
		return true
	}

	return false
}

// IsGroovyBuiltinType checks if a type name is a built-in Groovy/Java type.
//
// Built-in types are automatically available without imports and include:
//   - Primitive wrappers (String, Integer, Boolean, etc.)
//   - Common interfaces (List, Map, Set, etc.)
//   - Groovy types (Closure, GString, etc.)
func IsGroovyBuiltinType(typeName string) bool {
	builtinTypes := getGroovyBuiltinTypes()
	return builtinTypes[typeName]
}

// GetGroovyStdlibPrefixesList returns a list of stdlib prefixes for debugging.
func GetGroovyStdlibPrefixesList() []string {
	stdlibPrefixes := getGroovyStdlibPrefixes()
	result := make([]string, 0, len(stdlibPrefixes))
	for prefix := range stdlibPrefixes {
		result = append(result, prefix)
	}
	return result
}

// IsSpockImport checks if an import is from the Spock testing framework.
func IsSpockImport(imp string) bool {
	return strings.HasPrefix(imp, "spock.") ||
		strings.HasPrefix(imp, "org.spockframework.")
}

// IsTestImport checks if an import is from a common testing library.
func IsTestImport(imp string) bool {
	testPrefixes := []string{
		"spock.",
		"org.spockframework.",
		"org.junit.",
		"junit.",
		"org.mockito.",
		"org.assertj.",
		"org.hamcrest.",
		"geb.",
	}

	for _, prefix := range testPrefixes {
		if strings.HasPrefix(imp, prefix) {
			return true
		}
	}
	return false
}

// parseEmbeddedList parses a newline-separated list, ignoring comments and empty lines.
func parseEmbeddedList(data string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}
	}
	return result
}
