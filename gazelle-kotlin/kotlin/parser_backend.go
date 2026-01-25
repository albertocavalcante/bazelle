// Package kotlin provides a Gazelle extension for Kotlin BUILD file generation.
package kotlin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/treesitter"
)

// -----------------------------------------------------------------------------
// Backend Types and Interface
// -----------------------------------------------------------------------------

// ParserBackendType identifies the parsing strategy to use.
type ParserBackendType string

const (
	// BackendHeuristic uses regex-based parsing (current implementation).
	// Fast and well-tested but may miss edge cases.
	BackendHeuristic ParserBackendType = "heuristic"

	// BackendTreeSitter uses tree-sitter AST parsing.
	// More accurate but requires CGO or wazero backend.
	BackendTreeSitter ParserBackendType = "treesitter"

	// BackendHybrid runs both backends and compares results.
	// Useful for evaluation and gradual migration.
	BackendHybrid ParserBackendType = "hybrid"
)

// ParserBackend abstracts the parsing implementation.
type ParserBackend interface {
	// Name returns the backend identifier.
	Name() string

	// ParseContent parses Kotlin source code and returns metadata.
	ParseContent(ctx context.Context, content, path string) (*ParseResult, error)

	// ParseFile parses a Kotlin source file and returns metadata.
	ParseFile(ctx context.Context, path string) (*ParseResult, error)

	// Close releases any resources held by the backend.
	Close() error
}

// -----------------------------------------------------------------------------
// Errors
// -----------------------------------------------------------------------------

// ErrBackendNotSupported indicates the requested backend is not available.
type ErrBackendNotSupported struct {
	Backend ParserBackendType
	Reason  string
}

func (e ErrBackendNotSupported) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("backend %q not supported: %s", e.Backend, e.Reason)
	}
	return fmt.Sprintf("backend %q not supported", e.Backend)
}

// ErrLanguageNotSupported indicates Kotlin is not supported by the tree-sitter backend.
type ErrLanguageNotSupported struct {
	Backend string
}

func (e ErrLanguageNotSupported) Error() string {
	return fmt.Sprintf("tree-sitter backend %q does not support Kotlin", e.Backend)
}

// -----------------------------------------------------------------------------
// Configuration
// -----------------------------------------------------------------------------

// BackendConfig holds configuration for parser backends.
type BackendConfig struct {
	// EnableFQNScanning enables detection of fully-qualified names in code body.
	EnableFQNScanning bool

	// TreeSitterBackend specifies which tree-sitter backend to use.
	TreeSitterBackend treesitter.BackendType

	// HybridPrimary specifies which backend's output to use in hybrid mode.
	HybridPrimary ParserBackendType

	// HybridLogDiffs enables logging of differences between backends.
	HybridLogDiffs bool
}

// DefaultBackendConfig returns sensible defaults.
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
// HeuristicBackend - regex-based parsing
// -----------------------------------------------------------------------------

// HeuristicBackend wraps the existing regex-based parser.
type HeuristicBackend struct {
	parser *KotlinParser
}

// NewHeuristicBackend creates a heuristic (regex) backend.
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
// TreeSitterBackend - AST-based parsing
// -----------------------------------------------------------------------------

// Kotlin tree-sitter node types.
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
var declarationNodeTypes = []string{
	nodeClassDeclaration,
	nodeObjectDeclaration,
	nodeFunctionDeclaration,
	nodePropertyDeclaration,
	nodeTypeAlias,
}

// TreeSitterBackend uses tree-sitter for accurate AST-based parsing.
type TreeSitterBackend struct {
	backend      treesitter.Backend
	enableFQN    bool
	heuristicFQN *FQNScanner
}

// NewTreeSitterBackend creates a tree-sitter based backend.
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

	sort.Strings(result.Imports)
	sort.Strings(result.StarImports)
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
// HybridBackend - comparison mode
// -----------------------------------------------------------------------------

// HybridBackend runs both backends and compares results.
type HybridBackend struct {
	heuristic  *HeuristicBackend
	treesitter *TreeSitterBackend
	primary    ParserBackendType
	logDiffs   bool
	cfg        BackendConfig
}

// NewHybridBackend creates a hybrid backend for comparison.
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
			log.V(3).Debug("hybrid parser diff", "path", path, "diff", diff.String())
		}
	}

	// Return based on primary preference with fallback
	if b.primary == BackendTreeSitter {
		if tsErr != nil {
			log.V(3).Debug("hybrid tree-sitter failed, using heuristic",
				"path", path, "error", tsErr)
			return hResult, hErr
		}
		return tsResult, nil
	}

	if hErr != nil {
		log.V(3).Debug("hybrid heuristic failed, using tree-sitter",
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
// Comparison utilities
// -----------------------------------------------------------------------------

// ResultDiff captures differences between two parse results.
type ResultDiff struct {
	PackageDiff       *[2]string // [heuristic, treesitter] if different
	OnlyInHeuristic   []string   // imports only in heuristic
	OnlyInTreeSitter  []string   // imports only in tree-sitter
	StarOnlyHeuristic []string   // star imports only in heuristic
	StarOnlyTreeSit   []string   // star imports only in tree-sitter
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
	sort.Strings(diff)
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
