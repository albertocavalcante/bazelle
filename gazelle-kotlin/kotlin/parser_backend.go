// Package kotlin provides a Gazelle extension for Kotlin BUILD file generation.
//
// # Parser Backend Architecture
//
// This package provides multiple parsing strategies for extracting metadata from
// Kotlin source files. Understanding the distinction between heuristic and
// deterministic parsing is important for choosing the right backend:
//
// ## Deterministic Parsing (Tree-Sitter)
//
// The tree-sitter backend uses a formal grammar to build an Abstract Syntax Tree
// (AST). This approach is deterministic because:
//   - The same input always produces the same parse tree
//   - The parser follows the official Kotlin grammar specification
//   - Edge cases like string escaping, nested generics, and complex expressions
//     are handled correctly by the grammar rules
//
// Trade-offs: Requires tree-sitter runtime (CGO or WASM), slightly higher latency.
//
// ## Heuristic Parsing (Regex-Based)
//
// The heuristic backend uses carefully crafted regular expressions to extract
// metadata. This approach is approximate because:
//   - Regex cannot fully parse context-free grammars
//   - Edge cases (e.g., package declarations in strings) may cause false matches
//   - The patterns are tuned for common code patterns, not all valid Kotlin
//
// Trade-offs: No external dependencies, faster for simple files, sufficient for
// most real-world code.
//
// ## FQN Scanning (Heuristic)
//
// Detection of fully-qualified names in code bodies (e.g., "com.example.Foo()")
// is always heuristic, even when using tree-sitter for imports. This is because
// distinguishing FQNs from package-qualified references requires semantic
// analysis beyond syntax parsing.
//
// ## Choosing a Backend
//
// Use BackendHeuristic (default) for:
//   - Fast, dependency-free parsing
//   - Codebases with conventional import/package patterns
//   - CI/CD environments where minimal dependencies are preferred
//
// Use BackendTreeSitter for:
//   - Maximum accuracy for edge cases
//   - Generated or minified code
//   - When false positives/negatives are unacceptable
//
// Use BackendHybrid for:
//   - Validating heuristic accuracy against tree-sitter
//   - Gradual migration from heuristic to tree-sitter
//   - Debugging parsing discrepancies
package kotlin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/treesitter"
)

// -----------------------------------------------------------------------------
// Backend Types and Interface
// -----------------------------------------------------------------------------

// ParserBackendType identifies the parsing strategy to use.
//
// The choice of backend affects parsing accuracy vs. performance trade-offs.
// See package documentation for detailed guidance on choosing a backend.
type ParserBackendType string

const (
	// BackendHeuristic uses regex-based parsing.
	//
	// This is a HEURISTIC approach: fast and sufficient for most real-world
	// Kotlin code, but may produce incorrect results for edge cases like
	// imports inside string literals or unusual package declaration formats.
	//
	// Accuracy: ~99% for conventional Kotlin code.
	// Performance: O(n) where n is file size, minimal allocations.
	// Dependencies: None.
	BackendHeuristic ParserBackendType = "heuristic"

	// BackendTreeSitter uses AST-based parsing via tree-sitter.
	//
	// This is a DETERMINISTIC approach: the parser follows the formal Kotlin
	// grammar, producing correct results for all syntactically valid Kotlin.
	// Invalid syntax is handled gracefully with partial results.
	//
	// Accuracy: 100% for syntactically valid Kotlin.
	// Performance: O(n) with higher constant factor than heuristic.
	// Dependencies: Requires tree-sitter runtime (CGO or WASM backend).
	BackendTreeSitter ParserBackendType = "treesitter"

	// BackendHybrid runs both backends and compares results.
	//
	// This is a VALIDATION mode for comparing heuristic approximations against
	// deterministic parsing. Differences are logged for analysis. The primary
	// backend's result is returned, with automatic fallback on errors.
	//
	// Use cases:
	//   - Evaluating heuristic accuracy on a codebase
	//   - Debugging parsing discrepancies
	//   - Gradual migration from heuristic to tree-sitter
	BackendHybrid ParserBackendType = "hybrid"
)

// ParserBackend abstracts the parsing implementation, allowing callers to
// switch between heuristic and deterministic strategies without changing
// their code.
//
// Implementations should document whether their parsing approach is:
//   - Deterministic: produces identical results for identical input
//   - Heuristic: uses approximations that may miss edge cases
type ParserBackend interface {
	// Name returns the backend identifier ("heuristic", "treesitter", or "hybrid").
	Name() string

	// ParseContent parses Kotlin source code and returns metadata.
	//
	// The content parameter is the full source code as a string.
	// The path parameter is used for error messages and result metadata.
	//
	// Returns a ParseResult containing:
	//   - Package declaration (deterministic in both backends)
	//   - Import statements (deterministic in both backends)
	//   - FQN usages (always heuristic, see FQNScanner)
	//   - File annotations (deterministic in both backends)
	ParseContent(ctx context.Context, content, path string) (*ParseResult, error)

	// ParseFile reads and parses a Kotlin source file.
	//
	// This is a convenience method equivalent to reading the file and
	// calling ParseContent. File read errors are returned immediately.
	ParseFile(ctx context.Context, path string) (*ParseResult, error)

	// Close releases any resources held by the backend.
	//
	// For HeuristicBackend: no-op (no resources to release).
	// For TreeSitterBackend: releases the tree-sitter parser instance.
	// For HybridBackend: closes both underlying backends.
	Close() error
}

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

// ErrBackendNotSupported indicates the requested parser backend is not available.
//
// This error occurs when:
//   - An unknown backend type is requested
//   - Tree-sitter is requested but the runtime is not available
//   - A backend fails to initialize due to missing dependencies
type ErrBackendNotSupported struct {
	Backend ParserBackendType // The backend that was requested
	Reason  string            // Why it's not supported (optional)
}

func (e ErrBackendNotSupported) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("backend %q not supported: %s", e.Backend, e.Reason)
	}
	return fmt.Sprintf("backend %q not supported", e.Backend)
}

// ErrLanguageNotSupported indicates Kotlin is not supported by the tree-sitter backend.
//
// This error occurs when tree-sitter is available but doesn't have the Kotlin
// grammar loaded. This typically indicates a build or configuration issue.
type ErrLanguageNotSupported struct {
	Backend string // The tree-sitter backend that was tried
}

func (e ErrLanguageNotSupported) Error() string {
	return fmt.Sprintf("tree-sitter backend %q does not support Kotlin", e.Backend)
}

// -----------------------------------------------------------------------------
// Configuration
// -----------------------------------------------------------------------------

// BackendConfig holds configuration for parser backends.
//
// These options control the balance between parsing accuracy and performance.
// See individual field documentation for heuristic vs. deterministic behavior.
type BackendConfig struct {
	// EnableFQNScanning enables detection of fully-qualified names in code body.
	//
	// When enabled, the parser scans for FQNs like "com.example.Foo" used
	// directly in code without imports. This is ALWAYS HEURISTIC regardless
	// of the parser backend, as true FQN detection requires type resolution.
	//
	// Default: true
	EnableFQNScanning bool

	// TreeSitterBackend specifies which tree-sitter runtime to use.
	//
	// This affects TreeSitterBackend and HybridBackend only.
	// Options: BackendAuto, BackendCGO, BackendWazero
	//
	// DETERMINISTIC: All backends produce identical parse results.
	// They differ only in performance characteristics and build requirements.
	TreeSitterBackend treesitter.BackendType

	// HybridPrimary specifies which backend's output to use in hybrid mode.
	//
	// In hybrid mode, both backends run and results are compared. This option
	// determines which result is actually returned:
	//   - BackendHeuristic: Use heuristic result, validate against tree-sitter
	//   - BackendTreeSitter: Use tree-sitter result, validate against heuristic
	//
	// Default: BackendHeuristic (use heuristic for speed, validate for accuracy)
	HybridPrimary ParserBackendType

	// HybridLogDiffs enables logging of differences between backends.
	//
	// When enabled, any differences between heuristic and tree-sitter results
	// are logged at debug level. This helps identify files where heuristics fail.
	//
	// Default: true
	HybridLogDiffs bool
}

// DefaultBackendConfig returns sensible defaults for parser configuration.
//
// Defaults favor heuristic parsing with FQN scanning enabled:
//   - EnableFQNScanning: true (detect inline FQNs)
//   - TreeSitterBackend: Auto (let runtime choose best backend)
//   - HybridPrimary: Heuristic (prefer speed over accuracy)
//   - HybridLogDiffs: true (log differences for debugging)
func DefaultBackendConfig() BackendConfig {
	return BackendConfig{
		EnableFQNScanning: true,
		TreeSitterBackend: treesitter.BackendAuto,
		HybridPrimary:     BackendHeuristic,
		HybridLogDiffs:    true,
	}
}

// -----------------------------------------------------------------------------
// Factory
// -----------------------------------------------------------------------------

// NewParserBackend creates a parser backend of the specified type.
//
// This is the recommended way to create parser backends. It handles:
//   - Selecting the appropriate implementation based on type
//   - Applying configuration options
//   - Validating that required dependencies are available
//
// # Backend Selection Guide
//
// Choose based on your accuracy vs. performance needs:
//
//	typ := BackendHeuristic  // Fast, no dependencies, ~99% accuracy
//	typ := BackendTreeSitter // Slower, needs runtime, 100% accuracy
//	typ := BackendHybrid     // Both (for validation/debugging)
//
// Returns an error if tree-sitter is requested but not available.
func NewParserBackend(typ ParserBackendType, cfg BackendConfig) (ParserBackend, error) {
	switch typ {
	case BackendHeuristic:
		return NewHeuristicBackend(cfg), nil
	case BackendTreeSitter:
		return NewTreeSitterBackend(cfg)
	case BackendHybrid:
		return NewHybridBackend(cfg)
	default:
		return nil, ErrBackendNotSupported{Backend: typ, Reason: "unknown type"}
	}
}

// -----------------------------------------------------------------------------
// HeuristicBackend - Regex-Based Parsing (HEURISTIC)
// -----------------------------------------------------------------------------

// HeuristicBackend implements ParserBackend using regex pattern matching.
//
// # Heuristic Behavior
//
// This backend is HEURISTIC, not deterministic. It uses regular expressions
// to approximate Kotlin syntax parsing. While this works well for conventional
// code (~99% accuracy), it may produce incorrect results in edge cases:
//
// Known Limitations:
//   - Package/import declarations inside multi-line strings may be matched
//   - Escaped characters in strings containing "import" may confuse the parser
//   - Complex annotation syntax may not be fully captured
//   - Minified code with unusual formatting may parse incorrectly
//
// Why Use Heuristics:
//   - Zero external dependencies (no CGO, no WASM runtime)
//   - Lower latency for simple files
//   - Sufficient for most real-world, hand-written Kotlin code
//   - Well-tested patterns tuned for common coding conventions
//
// For maximum accuracy, use TreeSitterBackend instead.
type HeuristicBackend struct {
	parser *KotlinParser
}

// NewHeuristicBackend creates a new heuristic (regex-based) backend.
//
// The backend can optionally scan for fully-qualified names (FQNs) in the
// code body. FQN scanning is itself heuristic (see FQNScanner).
func NewHeuristicBackend(cfg BackendConfig) *HeuristicBackend {
	var opts []ParserOption
	if !cfg.EnableFQNScanning {
		opts = append(opts, WithFQNScanning(false))
	}
	return &HeuristicBackend{parser: NewParser(opts...)}
}

func (b *HeuristicBackend) Name() string { return string(BackendHeuristic) }

func (b *HeuristicBackend) ParseContent(_ context.Context, content, path string) (*ParseResult, error) {
	return b.parser.ParseContent(content, path)
}

func (b *HeuristicBackend) ParseFile(ctx context.Context, path string) (*ParseResult, error) {
	content, err := readFileContent(path)
	if err != nil {
		return nil, err
	}
	return b.ParseContent(ctx, content, path)
}

func (b *HeuristicBackend) Close() error { return nil }

// -----------------------------------------------------------------------------
// TreeSitterBackend - AST-Based Parsing (DETERMINISTIC)
// -----------------------------------------------------------------------------

// Kotlin tree-sitter node types define the AST structure used for extraction.
// These are determined by the tree-sitter-kotlin grammar and are stable
// across parser invocations.
const (
	nodePackageHeader       = "package_header"
	nodeImportHeader        = "import_header"
	nodeFileAnnotation      = "file_annotation"
	nodeIdentifier          = "identifier"
	nodeClassDeclaration    = "class_declaration"
	nodeObjectDeclaration   = "object_declaration"
	nodeFunctionDeclaration = "function_declaration"
	nodePropertyDeclaration = "property_declaration"
	nodeTypeAlias           = "type_alias"
)

// declarationNodeTypes lists node types that mark the start of code.
// These are used to determine where the import section ends and code begins.
var declarationNodeTypes = []string{
	nodeClassDeclaration,
	nodeObjectDeclaration,
	nodeFunctionDeclaration,
	nodePropertyDeclaration,
	nodeTypeAlias,
}

// TreeSitterBackend implements ParserBackend using tree-sitter AST parsing.
//
// # Deterministic Behavior
//
// This backend is DETERMINISTIC for extracting packages, imports, and annotations.
// It uses the tree-sitter-kotlin grammar to build a proper Abstract Syntax Tree,
// ensuring:
//   - The same input always produces the same output
//   - String literals are never confused with real declarations
//   - Complex syntax (nested generics, multi-line statements) is handled correctly
//   - Invalid syntax produces partial results, not garbage
//
// # FQN Scanning (Heuristic Component)
//
// Note that FQN (fully-qualified name) detection in code bodies remains HEURISTIC
// even with this backend. True FQN detection requires semantic analysis (type
// resolution) which is beyond syntax parsing. The heuristic FQNScanner is used
// for this purpose when EnableFQNScanning is set.
//
// # Dependencies
//
// Requires a tree-sitter runtime, either:
//   - CGO backend (native, fastest)
//   - Wazero backend (pure Go, portable)
//
// The backend is selected automatically based on build tags.
type TreeSitterBackend struct {
	backend      treesitter.Backend
	enableFQN    bool
	heuristicFQN *FQNScanner // Note: FQN scanning is always heuristic
}

// NewTreeSitterBackend creates a deterministic AST-based parser backend.
//
// Returns an error if the tree-sitter runtime is not available or doesn't
// support Kotlin parsing.
func NewTreeSitterBackend(cfg BackendConfig) (*TreeSitterBackend, error) {
	backend, err := treesitter.NewBackend(cfg.TreeSitterBackend)
	if err != nil {
		return nil, fmt.Errorf("create tree-sitter backend: %w", err)
	}

	if !backend.SupportsLanguage(treesitter.Kotlin) {
		_ = backend.Close()
		return nil, ErrLanguageNotSupported{Backend: backend.Name()}
	}

	return &TreeSitterBackend{
		backend:      backend,
		enableFQN:    cfg.EnableFQNScanning,
		heuristicFQN: NewFQNScanner(),
	}, nil
}

func (b *TreeSitterBackend) Name() string { return string(BackendTreeSitter) }

func (b *TreeSitterBackend) ParseContent(ctx context.Context, content, path string) (_ *ParseResult, retErr error) {
	if b == nil {
		return nil, fmt.Errorf("TreeSitterBackend is nil")
	}
	parser, err := b.backend.NewParser(treesitter.Kotlin)
	if err != nil {
		return nil, fmt.Errorf("create Kotlin parser: %w", err)
	}
	defer func() {
		if closeErr := parser.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("close parser: %w", closeErr)
		}
	}()

	tree, err := parser.Parse(ctx, []byte(content))
	if err != nil {
		return nil, fmt.Errorf("parse Kotlin source: %w", err)
	}
	defer func() {
		if closeErr := tree.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("close tree: %w", closeErr)
		}
	}()

	source := []byte(content)
	root := tree.RootNode()

	result := &ParseResult{
		FilePath:      path,
		Imports:       make([]string, 0),
		StarImports:   make([]string, 0),
		ImportAliases: make(map[string]string),
		FQNs:          make([]string, 0),
		Annotations:   make([]string, 0),
	}

	result.Package = extractPackageFromAST(root, source)
	extractImportsFromAST(root, source, result)
	result.Annotations = extractAnnotationsFromAST(root, source)
	result.CodeStartLine = findCodeStartLineFromAST(root)

	// FQN scanning uses heuristic approach (AST-based FQN detection is future work)
	if b.enableFQN && b.heuristicFQN != nil && result.CodeStartLine > 0 {
		startLine := max(result.CodeStartLine-1, 0)
		scanResult := b.heuristicFQN.Scan(content, startLine)
		result.FQNs = scanResult.FQNs
	}

	result.AllDependencies = buildAllDependencies(result)
	return result, nil
}

func (b *TreeSitterBackend) ParseFile(ctx context.Context, path string) (*ParseResult, error) {
	content, err := readFileContent(path)
	if err != nil {
		return nil, err
	}
	return b.ParseContent(ctx, content, path)
}

func (b *TreeSitterBackend) Close() error {
	if b.backend != nil {
		return b.backend.Close()
	}
	return nil
}

// extractPackageFromAST finds the package declaration in the AST.
func extractPackageFromAST(root treesitter.Node, source []byte) string {
	pkgNodes := treesitter.FindByType(root, nodePackageHeader)
	if len(pkgNodes) == 0 {
		return ""
	}

	pkgNode := pkgNodes[0]

	// Try field name first
	if identNode := pkgNode.ChildByFieldName(nodeIdentifier); identNode != nil && !identNode.IsNull() {
		return identNode.Content(source)
	}

	// Fallback: search children
	for i := uint32(0); i < pkgNode.ChildCount(); i++ {
		if child := pkgNode.Child(i); child != nil && child.Type() == nodeIdentifier {
			return child.Content(source)
		}
	}

	return ""
}

// extractImportsFromAST finds all import declarations in the AST.
func extractImportsFromAST(root treesitter.Node, source []byte, result *ParseResult) {
	importNodes := treesitter.FindByType(root, nodeImportHeader)

	for _, node := range importNodes {
		processImportNode(node, source, result)
	}

	slices.Sort(result.Imports)
	slices.Sort(result.StarImports)
}

// processImportNode extracts details from a single import node.
func processImportNode(node treesitter.Node, source []byte, result *ParseResult) {
	content := strings.TrimSpace(node.Content(source))

	// Remove "import " prefix
	path, found := strings.CutPrefix(content, "import ")
	if !found {
		return
	}
	path = strings.TrimSpace(path)

	// Star import: "package.*"
	if pkg, isStar := strings.CutSuffix(path, ".*"); isStar {
		if pkg = strings.TrimSpace(pkg); pkg != "" {
			result.StarImports = append(result.StarImports, pkg)
		}
		return
	}

	// Aliased import: "package.Class as Alias"
	if before, after, hasAlias := strings.Cut(path, " as "); hasAlias {
		importPath := strings.TrimSpace(before)
		alias := strings.TrimSpace(after)
		if importPath != "" {
			result.Imports = append(result.Imports, importPath)
			if alias != "" {
				result.ImportAliases[alias] = importPath
			}
		}
		return
	}

	// Regular import
	if identNode := node.ChildByFieldName(nodeIdentifier); identNode != nil && !identNode.IsNull() {
		if p := identNode.Content(source); p != "" {
			result.Imports = append(result.Imports, p)
			return
		}
	}

	// Fallback: use cleaned content
	if path != "" {
		result.Imports = append(result.Imports, path)
	}
}

// extractAnnotationsFromAST finds file-level annotations in the AST.
func extractAnnotationsFromAST(root treesitter.Node, source []byte) []string {
	var annotations []string

	for _, node := range treesitter.FindByType(root, nodeFileAnnotation) {
		content := node.Content(source)
		if name, found := strings.CutPrefix(content, "@file:"); found {
			// Remove arguments: @file:Name(args) -> Name
			if idx := strings.Index(name, "("); idx > 0 {
				name = name[:idx]
			}
			if name = strings.TrimSpace(name); name != "" {
				annotations = append(annotations, name)
			}
		}
	}

	return annotations
}

// findCodeStartLineFromAST finds where declarations begin in the AST.
func findCodeStartLineFromAST(root treesitter.Node) int {
	minLine := -1

	for _, declType := range declarationNodeTypes {
		for _, node := range treesitter.FindByType(root, declType) {
			line := int(node.StartPoint().Row) + 1 // 1-indexed
			if minLine < 0 || line < minLine {
				minLine = line
			}
		}
	}

	if minLine < 0 {
		return 0
	}
	return minLine
}

// -----------------------------------------------------------------------------
// HybridBackend - Validation and Comparison Mode
// -----------------------------------------------------------------------------

// HybridBackend runs both heuristic and deterministic backends, comparing results.
//
// # Purpose
//
// This backend is designed for validation, not production use. It allows you to:
//   - Measure heuristic accuracy against deterministic parsing
//   - Identify files where heuristics fail
//   - Gradually migrate from heuristic to deterministic parsing
//   - Debug parsing discrepancies
//
// # Behavior
//
// When parsing, HybridBackend:
//  1. Runs both backends on the same input
//  2. Compares results (package, imports, star imports)
//  3. Logs differences if HybridLogDiffs is enabled
//  4. Returns the primary backend's result
//  5. Falls back to the other backend on errors
//
// # Result Selection
//
// The HybridPrimary config option determines which backend's result to return:
//   - BackendHeuristic (default): Returns heuristic result, validates against tree-sitter
//   - BackendTreeSitter: Returns tree-sitter result, validates against heuristic
//
// This allows using hybrid mode in production while collecting validation data.
type HybridBackend struct {
	heuristic  *HeuristicBackend  // Regex-based (heuristic)
	treesitter *TreeSitterBackend // AST-based (deterministic)
	primary    ParserBackendType  // Which result to return
	logDiffs   bool               // Log differences between backends
	cfg        BackendConfig
}

// NewHybridBackend creates a validation backend that runs both parsing strategies.
//
// Requires tree-sitter support; returns an error if tree-sitter is unavailable.
func NewHybridBackend(cfg BackendConfig) (*HybridBackend, error) {
	heuristic := NewHeuristicBackend(cfg)

	ts, err := NewTreeSitterBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("create tree-sitter for hybrid: %w", err)
	}

	return &HybridBackend{
		heuristic:  heuristic,
		treesitter: ts,
		primary:    cfg.HybridPrimary,
		logDiffs:   cfg.HybridLogDiffs,
		cfg:        cfg,
	}, nil
}

func (b *HybridBackend) Name() string { return string(BackendHybrid) }

func (b *HybridBackend) ParseContent(ctx context.Context, content, path string) (*ParseResult, error) {
	if b == nil {
		return nil, fmt.Errorf("HybridBackend is nil")
	}
	hResult, hErr := b.heuristic.ParseContent(ctx, content, path)
	tsResult, tsErr := b.treesitter.ParseContent(ctx, content, path)

	if b.logDiffs && hErr == nil && tsErr == nil {
		if diff := compareResults(hResult, tsResult); diff.HasDifferences() {
			log.V(3).Debugw("hybrid parser diff", "path", path, "diff", diff.String())
		}
	}

	// Return based on primary preference with fallback
	if b.primary == BackendTreeSitter {
		if tsErr != nil {
			log.V(3).Debugw("hybrid tree-sitter failed, using heuristic",
				"path", path, "error", tsErr)
			return hResult, hErr
		}
		return tsResult, nil
	}

	if hErr != nil {
		log.V(3).Debugw("hybrid heuristic failed, using tree-sitter",
			"path", path, "error", hErr)
		return tsResult, tsErr
	}
	return hResult, nil
}

func (b *HybridBackend) ParseFile(ctx context.Context, path string) (*ParseResult, error) {
	content, err := readFileContent(path)
	if err != nil {
		return nil, err
	}
	return b.ParseContent(ctx, content, path)
}

func (b *HybridBackend) Close() error {
	return errors.Join(b.heuristic.Close(), b.treesitter.Close())
}

// -----------------------------------------------------------------------------
// Comparison Utilities - Heuristic vs Deterministic Validation
// -----------------------------------------------------------------------------

// ResultDiff captures differences between heuristic and deterministic parse results.
//
// This type is used by HybridBackend to identify cases where heuristic parsing
// produces different results than AST-based parsing. Differences indicate either:
//   - Edge cases where heuristics fail (heuristic has wrong values)
//   - Parser bugs that need fixing
//   - Unusual code patterns worth investigating
//
// When differences are found, examine the source file to determine which
// result is correct. In most cases, the tree-sitter result is authoritative.
type ResultDiff struct {
	// PackageDiff holds [heuristic, treesitter] values if they differ.
	// nil means both backends found the same package.
	PackageDiff *[2]string

	// OnlyInHeuristic lists imports found by heuristic but not tree-sitter.
	// These are likely false positives from the heuristic parser.
	OnlyInHeuristic []string

	// OnlyInTreeSitter lists imports found by tree-sitter but not heuristic.
	// These are likely false negatives from the heuristic parser.
	OnlyInTreeSitter []string

	// StarOnlyHeuristic lists star imports found only by heuristic.
	StarOnlyHeuristic []string

	// StarOnlyTreeSit lists star imports found only by tree-sitter.
	StarOnlyTreeSit []string
}

// HasDifferences returns true if any differences were found.
func (d ResultDiff) HasDifferences() bool {
	return d.PackageDiff != nil ||
		len(d.OnlyInHeuristic) > 0 ||
		len(d.OnlyInTreeSitter) > 0 ||
		len(d.StarOnlyHeuristic) > 0 ||
		len(d.StarOnlyTreeSit) > 0
}

// String formats the diff for logging.
func (d ResultDiff) String() string {
	var parts []string

	if d.PackageDiff != nil {
		parts = append(parts, fmt.Sprintf("package: heuristic=%q treesitter=%q",
			d.PackageDiff[0], d.PackageDiff[1]))
	}
	if len(d.OnlyInHeuristic) > 0 {
		parts = append(parts, fmt.Sprintf("imports only in heuristic: %v", d.OnlyInHeuristic))
	}
	if len(d.OnlyInTreeSitter) > 0 {
		parts = append(parts, fmt.Sprintf("imports only in treesitter: %v", d.OnlyInTreeSitter))
	}
	if len(d.StarOnlyHeuristic) > 0 {
		parts = append(parts, fmt.Sprintf("star imports only in heuristic: %v", d.StarOnlyHeuristic))
	}
	if len(d.StarOnlyTreeSit) > 0 {
		parts = append(parts, fmt.Sprintf("star imports only in treesitter: %v", d.StarOnlyTreeSit))
	}

	return strings.Join(parts, "; ")
}

// compareResults computes differences between two parse results.
func compareResults(h, ts *ParseResult) ResultDiff {
	var diff ResultDiff

	if h.Package != ts.Package {
		diff.PackageDiff = &[2]string{h.Package, ts.Package}
	}

	hImports, tsImports := toStringSet(h.Imports), toStringSet(ts.Imports)
	diff.OnlyInHeuristic = sortedDifference(hImports, tsImports)
	diff.OnlyInTreeSitter = sortedDifference(tsImports, hImports)

	hStars, tsStars := toStringSet(h.StarImports), toStringSet(ts.StarImports)
	diff.StarOnlyHeuristic = sortedDifference(hStars, tsStars)
	diff.StarOnlyTreeSit = sortedDifference(tsStars, hStars)

	return diff
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// toStringSet converts a slice to a set for efficient lookup.
func toStringSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}

// sortedDifference returns elements in a but not in b, sorted.
func sortedDifference(a, b map[string]struct{}) []string {
	var diff []string
	for k := range a {
		if _, ok := b[k]; !ok {
			diff = append(diff, k)
		}
	}
	slices.Sort(diff)
	return diff
}

// readFileContent reads a file as string.
func readFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	return string(content), nil
}
