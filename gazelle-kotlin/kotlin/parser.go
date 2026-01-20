package kotlin

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// KotlinParser parses Kotlin source files to extract metadata.
type KotlinParser struct {
	packageRegex *regexp.Regexp
	importRegex  *regexp.Regexp
}

// ParseResult contains the parsed metadata from a Kotlin file.
type ParseResult struct {
	// Package is the package declaration (e.g., "com.example.myapp").
	Package string

	// Imports is a list of import statements.
	Imports []string

	// FilePath is the path to the parsed file.
	FilePath string
}

// NewParser creates a new KotlinParser.
func NewParser() *KotlinParser {
	return &KotlinParser{
		// Match: package com.example.myapp
		packageRegex: regexp.MustCompile(`^\s*package\s+([\w.]+)`),
		// Match: import com.example.SomeClass
		// Also match: import com.example.SomeClass as Alias
		importRegex: regexp.MustCompile(`^\s*import\s+([\w.]+(?:\.\*)?)`),
	}
}

// ParseFile parses a Kotlin source file and returns metadata.
func (p *KotlinParser) ParseFile(path string) (*ParseResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := &ParseResult{
		FilePath: path,
		Imports:  make([]string, 0),
	}

	scanner := bufio.NewScanner(file)
	inMultilineComment := false

	for scanner.Scan() {
		line := scanner.Text()

		// Skip multi-line comments
		if strings.Contains(line, "/*") {
			inMultilineComment = true
		}
		if strings.Contains(line, "*/") {
			inMultilineComment = false
			continue
		}
		if inMultilineComment {
			continue
		}

		// Skip single-line comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Try to match package declaration
		if result.Package == "" {
			if matches := p.packageRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Package = matches[1]
			}
		}

		// Try to match import statements
		if matches := p.importRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Imports = append(result.Imports, matches[1])
		}

		// Stop parsing after we've passed the imports section
		// (imports must come before class/function declarations)
		if strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "object ") ||
			strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "fun ") ||
			strings.HasPrefix(trimmed, "val ") ||
			strings.HasPrefix(trimmed, "var ") ||
			strings.HasPrefix(trimmed, "annotation ") ||
			strings.HasPrefix(trimmed, "enum ") ||
			strings.HasPrefix(trimmed, "sealed ") ||
			strings.HasPrefix(trimmed, "data ") ||
			strings.HasPrefix(trimmed, "inline ") ||
			strings.HasPrefix(trimmed, "suspend ") ||
			strings.HasPrefix(trimmed, "private ") ||
			strings.HasPrefix(trimmed, "internal ") ||
			strings.HasPrefix(trimmed, "public ") ||
			strings.HasPrefix(trimmed, "protected ") ||
			strings.HasPrefix(trimmed, "abstract ") ||
			strings.HasPrefix(trimmed, "open ") {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ParseFiles parses multiple Kotlin files and returns their metadata.
func (p *KotlinParser) ParseFiles(paths []string) ([]*ParseResult, error) {
	results := make([]*ParseResult, 0, len(paths))
	for _, path := range paths {
		result, err := p.ParseFile(path)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// GetPackages returns unique packages from parse results.
func GetPackages(results []*ParseResult) []string {
	pkgSet := make(map[string]bool)
	for _, r := range results {
		if r.Package != "" {
			pkgSet[r.Package] = true
		}
	}

	packages := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		packages = append(packages, pkg)
	}
	return packages
}
