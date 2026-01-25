// Package treesitter provides a pluggable abstraction layer for tree-sitter
// parsing backends. It supports both CGO-based (smacker/go-tree-sitter) and
// WASM/wazero-based (malivvan/tree-sitter) implementations, allowing users
// to choose the appropriate backend for their deployment needs.
//
// The CGO backend provides production-ready parsing with broad language support,
// while the wazero backend enables CGO-free deployments at the cost of limited
// language support (currently only C and C++).
//
// # Quick Start
//
// Create a backend and parse some code:
//
//	backend, err := treesitter.NewBackendFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer backend.Close()
//
//	parser, err := backend.NewParser(treesitter.Go)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer parser.Close()
//
//	tree, err := parser.ParseString(context.Background(), `package main`)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tree.Close()
//
//	root := tree.RootNode()
//	fmt.Println(root.Type()) // Output: source_file
//
// # Backend Selection
//
// The backend can be selected via environment variable:
//
//	export BAZELLE_TREESITTER_BACKEND=cgo    # Use CGO backend (default when available)
//	export BAZELLE_TREESITTER_BACKEND=wazero # Use WASM/wazero backend
//	export BAZELLE_TREESITTER_BACKEND=auto   # Auto-select (CGO first, then wazero)
//
// # Thread Safety
//
// Backends are safe for concurrent use. Parsers should not be used concurrently
// from multiple goroutines. Trees and Nodes are safe to read concurrently but
// should not be modified. For concurrent parsing, create multiple parsers from
// the same backend.
package treesitter

import "context"

// Language represents a programming language grammar that can be parsed.
type Language string

const (
	// Go represents the Go programming language.
	Go Language = "go"

	// Java represents the Java programming language.
	Java Language = "java"

	// Kotlin represents the Kotlin programming language.
	Kotlin Language = "kotlin"

	// Scala represents the Scala programming language.
	Scala Language = "scala"

	// Rust represents the Rust programming language.
	Rust Language = "rust"

	// Python represents the Python programming language.
	Python Language = "python"

	// JavaScript represents the JavaScript programming language.
	JavaScript Language = "javascript"

	// TypeScript represents the TypeScript programming language.
	TypeScript Language = "typescript"

	// TSX represents TypeScript with JSX syntax.
	TSX Language = "tsx"

	// Groovy represents the Groovy programming language.
	Groovy Language = "groovy"

	// C represents the C programming language.
	C Language = "c"

	// Cpp represents the C++ programming language.
	Cpp Language = "cpp"

	// CSharp represents the C# programming language.
	CSharp Language = "csharp"

	// Ruby represents the Ruby programming language.
	Ruby Language = "ruby"

	// PHP represents the PHP programming language.
	PHP Language = "php"

	// Swift represents the Swift programming language.
	Swift Language = "swift"

	// Bash represents the Bash shell scripting language.
	Bash Language = "bash"

	// HTML represents HTML markup.
	HTML Language = "html"

	// CSS represents CSS stylesheets.
	CSS Language = "css"

	// SQL represents SQL query language.
	SQL Language = "sql"

	// YAML represents YAML data format.
	YAML Language = "yaml"

	// TOML represents TOML configuration format.
	TOML Language = "toml"

	// JSON represents JSON data format.
	JSON Language = "json"

	// Markdown represents Markdown text format.
	Markdown Language = "markdown"

	// Protobuf represents Protocol Buffers definition language.
	Protobuf Language = "protobuf"

	// HCL represents HashiCorp Configuration Language.
	HCL Language = "hcl"

	// Dockerfile represents Dockerfile syntax.
	Dockerfile Language = "dockerfile"

	// Lua represents the Lua programming language.
	Lua Language = "lua"

	// Elixir represents the Elixir programming language.
	Elixir Language = "elixir"

	// Elm represents the Elm programming language.
	Elm Language = "elm"

	// OCaml represents the OCaml programming language.
	OCaml Language = "ocaml"

	// Svelte represents Svelte component syntax.
	Svelte Language = "svelte"

	// Cue represents the CUE configuration language.
	Cue Language = "cue"
)

// AllLanguages returns a slice of all defined Language constants.
func AllLanguages() []Language {
	return []Language{
		Go, Java, Kotlin, Scala, Rust, Python, JavaScript, TypeScript, TSX,
		Groovy, C, Cpp, CSharp, Ruby, PHP, Swift, Bash, HTML, CSS, SQL,
		YAML, TOML, JSON, Markdown, Protobuf, HCL, Dockerfile, Lua, Elixir,
		Elm, OCaml, Svelte, Cue,
	}
}

// Backend abstracts the tree-sitter implementation, allowing different
// backends (CGO, WASM/wazero) to be used interchangeably.
type Backend interface {
	// Name returns the backend identifier ("cgo" or "wazero").
	Name() string

	// IsExperimental returns true for backends not yet production-ready.
	// The wazero backend is considered experimental due to limited language
	// support and the pre-release status of the underlying library.
	IsExperimental() bool

	// SupportedLanguages returns the list of languages this backend can parse.
	SupportedLanguages() []Language

	// SupportsLanguage checks if the backend can parse the given language.
	SupportsLanguage(lang Language) bool

	// NewParser creates a parser configured for the given language.
	// Returns an error if the language is not supported by this backend.
	NewParser(lang Language) (Parser, error)

	// Close releases any resources held by the backend.
	// After Close is called, the backend should not be used.
	Close() error
}

// Parser parses source code into a concrete syntax tree (CST).
type Parser interface {
	// Language returns the language this parser is configured for.
	Language() Language

	// Parse parses the given source code and returns the syntax tree.
	// The context can be used for cancellation of long-running parses.
	Parse(ctx context.Context, source []byte) (Tree, error)

	// ParseString is a convenience method that parses a string.
	ParseString(ctx context.Context, source string) (Tree, error)

	// Close releases any resources held by the parser.
	Close() error
}

// Tree represents a parsed syntax tree.
type Tree interface {
	// RootNode returns the root node of the syntax tree.
	RootNode() Node

	// Source returns the original source code that was parsed.
	Source() []byte

	// HasError returns true if the tree contains any syntax errors.
	// This is useful for quick validation without traversing the entire tree.
	HasError() bool

	// Close releases any resources held by the tree.
	Close() error
}

// Node represents a node in the syntax tree.
type Node interface {
	// Type returns the grammar type of this node (e.g., "import_statement",
	// "function_definition", "identifier").
	Type() string

	// StartByte returns the byte offset where this node starts in the source.
	StartByte() uint32

	// EndByte returns the byte offset where this node ends in the source.
	EndByte() uint32

	// StartPoint returns the (row, column) position where this node starts.
	StartPoint() Point

	// EndPoint returns the (row, column) position where this node ends.
	EndPoint() Point

	// Content extracts the source text for this node using the provided source.
	// This is a convenience method equivalent to source[StartByte():EndByte()].
	Content(source []byte) string

	// ChildCount returns the total number of children (including anonymous nodes).
	ChildCount() uint32

	// Child returns the child at the given index.
	// Returns nil if the index is out of bounds.
	Child(index uint32) Node

	// NamedChildCount returns the number of named children (excluding anonymous nodes).
	// Named children correspond to named rules in the grammar.
	NamedChildCount() uint32

	// NamedChild returns the named child at the given index.
	// Returns nil if the index is out of bounds.
	NamedChild(index uint32) Node

	// ChildByFieldName returns the child with the given field name.
	// Returns nil if no child has this field name.
	ChildByFieldName(name string) Node

	// Parent returns the parent node, or nil if this is the root.
	Parent() Node

	// NextSibling returns the next sibling node, or nil if none.
	NextSibling() Node

	// PrevSibling returns the previous sibling node, or nil if none.
	PrevSibling() Node

	// NextNamedSibling returns the next named sibling, or nil if none.
	NextNamedSibling() Node

	// PrevNamedSibling returns the previous named sibling, or nil if none.
	PrevNamedSibling() Node

	// IsNamed returns true if this is a named node (not anonymous).
	IsNamed() bool

	// IsError returns true if this node represents a syntax error.
	IsError() bool

	// IsMissing returns true if this node was inserted by error recovery.
	IsMissing() bool

	// IsNull returns true if this is a null/missing node reference.
	IsNull() bool

	// String returns a S-expression representation of the subtree.
	String() string
}

// Point represents a position in source code as (row, column).
// Both row and column are 0-indexed.
type Point struct {
	Row    uint32
	Column uint32
}

// TreeCursor provides efficient traversal of a syntax tree.
// It's more memory-efficient than recursively accessing children.
type TreeCursor interface {
	// Reset resets the cursor to start at the given node.
	Reset(node Node)

	// CurrentNode returns the node at the current cursor position.
	CurrentNode() Node

	// CurrentFieldName returns the field name of the current node, if any.
	CurrentFieldName() string

	// GotoParent moves to the parent of the current node.
	// Returns false if already at the root.
	GotoParent() bool

	// GotoFirstChild moves to the first child of the current node.
	// Returns false if the current node has no children.
	GotoFirstChild() bool

	// GotoNextSibling moves to the next sibling of the current node.
	// Returns false if there are no more siblings.
	GotoNextSibling() bool

	// Close releases any resources held by the cursor.
	Close()
}

// ErrLanguageNotSupported is returned when attempting to parse a language
// that is not supported by the current backend.
type ErrLanguageNotSupported struct {
	Language Language
	Backend  string
}

func (e ErrLanguageNotSupported) Error() string {
	return "language " + string(e.Language) + " is not supported by backend " + e.Backend
}

// ErrBackendClosed is returned when attempting to use a backend after Close.
type ErrBackendClosed struct {
	Backend string
}

func (e ErrBackendClosed) Error() string {
	return "backend " + e.Backend + " has been closed"
}

// ErrParserClosed is returned when attempting to use a parser after Close.
type ErrParserClosed struct{}

func (e ErrParserClosed) Error() string {
	return "parser has been closed"
}

// Children returns a slice of all children of the given node.
// This is a convenience function that collects all children into a slice.
// For large nodes, consider using Child() with an index for better performance.
func Children(n Node) []Node {
	if n == nil || n.IsNull() {
		return nil
	}
	count := n.ChildCount()
	if count == 0 {
		return nil
	}
	children := make([]Node, 0, count)
	for i := uint32(0); i < count; i++ {
		if child := n.Child(i); child != nil {
			children = append(children, child)
		}
	}
	return children
}

// NamedChildren returns a slice of all named children of the given node.
// Named children are nodes that correspond to named rules in the grammar,
// excluding anonymous tokens like punctuation.
func NamedChildren(n Node) []Node {
	if n == nil || n.IsNull() {
		return nil
	}
	count := n.NamedChildCount()
	if count == 0 {
		return nil
	}
	children := make([]Node, 0, count)
	for i := uint32(0); i < count; i++ {
		if child := n.NamedChild(i); child != nil {
			children = append(children, child)
		}
	}
	return children
}

// ChildrenByType returns all children of the given node that match the specified type.
// This is useful for finding all nodes of a specific kind (e.g., all "import_declaration" nodes).
func ChildrenByType(n Node, nodeType string) []Node {
	if n == nil || n.IsNull() {
		return nil
	}
	var matches []Node
	count := n.ChildCount()
	for i := uint32(0); i < count; i++ {
		if child := n.Child(i); child != nil && child.Type() == nodeType {
			matches = append(matches, child)
		}
	}
	return matches
}

// FindFirst performs a depth-first search and returns the first node matching the predicate.
// Returns nil if no matching node is found.
func FindFirst(n Node, predicate func(Node) bool) Node {
	if n == nil || n.IsNull() {
		return nil
	}
	if predicate(n) {
		return n
	}
	count := n.ChildCount()
	for i := uint32(0); i < count; i++ {
		if child := n.Child(i); child != nil {
			if found := FindFirst(child, predicate); found != nil {
				return found
			}
		}
	}
	return nil
}

// FindAll performs a depth-first search and returns all nodes matching the predicate.
func FindAll(n Node, predicate func(Node) bool) []Node {
	var results []Node
	Walk(n, func(node Node) bool {
		if predicate(node) {
			results = append(results, node)
		}
		return true // continue walking
	})
	return results
}

// Walk traverses the tree in depth-first order, calling the visitor function for each node.
// The visitor returns true to continue walking, false to stop.
// Walk returns true if the entire tree was traversed, false if stopped early.
func Walk(n Node, visitor func(Node) bool) bool {
	if n == nil || n.IsNull() {
		return true
	}
	if !visitor(n) {
		return false
	}
	count := n.ChildCount()
	for i := uint32(0); i < count; i++ {
		if child := n.Child(i); child != nil {
			if !Walk(child, visitor) {
				return false
			}
		}
	}
	return true
}

// FindByType performs a depth-first search and returns all nodes of the given type.
// This is a convenience wrapper around FindAll.
func FindByType(n Node, nodeType string) []Node {
	return FindAll(n, func(node Node) bool {
		return node.Type() == nodeType
	})
}

// HasErrors walks the tree and returns true if any error nodes are found.
// For better performance, prefer using Tree.HasError() if available.
func HasErrors(n Node) bool {
	if n == nil || n.IsNull() {
		return false
	}
	found := false
	Walk(n, func(node Node) bool {
		if node.IsError() || node.IsMissing() {
			found = true
			return false // stop walking
		}
		return true
	})
	return found
}
