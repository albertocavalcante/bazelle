//go:build cgo

package treesitter

import (
	"context"
	"fmt"
	"slices"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	// Language grammars - import all supported languages
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	// NOTE: markdown has tree_sitter/ include path issues in Bazel, excluded
	// markdownblock "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/ocaml"
	// NOTE: php has tree_sitter/ include path issues in Bazel, excluded
	// "github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	// NOTE: sql has tree_sitter/ include path issues in Bazel, excluded
	// "github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

// cgoBackend implements Backend using the CGO-based smacker/go-tree-sitter library.
type cgoBackend struct {
	mu     sync.RWMutex
	closed bool
}

// NewCGOBackend creates a new CGO-based tree-sitter backend.
// This backend uses smacker/go-tree-sitter which requires CGO.
func NewCGOBackend() (Backend, error) {
	return &cgoBackend{}, nil
}

func (b *cgoBackend) Name() string {
	return "cgo"
}

func (b *cgoBackend) IsExperimental() bool {
	return false
}

func (b *cgoBackend) SupportedLanguages() []Language {
	// NOTE: PHP, SQL, and Markdown are excluded due to tree_sitter/ include path issues in Bazel
	return []Language{
		Go, Java, Kotlin, Scala, Rust, Python, JavaScript, TypeScript, TSX,
		Groovy, C, Cpp, CSharp, Ruby, Swift, Bash, HTML, CSS,
		YAML, TOML, Protobuf, HCL, Dockerfile, Lua, Elixir,
		Elm, OCaml, Svelte, Cue,
	}
}

func (b *cgoBackend) SupportsLanguage(lang Language) bool {
	return slices.Contains(b.SupportedLanguages(), lang)
}

func (b *cgoBackend) NewParser(lang Language) (Parser, error) {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return nil, ErrBackendClosed{Backend: b.Name()}
	}

	sitterLang, err := b.getSitterLanguage(lang)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(sitterLang)

	return &cgoParser{
		parser: parser,
		lang:   lang,
	}, nil
}

func (b *cgoBackend) getSitterLanguage(lang Language) (*sitter.Language, error) {
	switch lang {
	case Go:
		return golang.GetLanguage(), nil
	case Java:
		return java.GetLanguage(), nil
	case Kotlin:
		return kotlin.GetLanguage(), nil
	case Scala:
		return scala.GetLanguage(), nil
	case Rust:
		return rust.GetLanguage(), nil
	case Python:
		return python.GetLanguage(), nil
	case JavaScript:
		return javascript.GetLanguage(), nil
	case TypeScript:
		return typescript.GetLanguage(), nil
	case TSX:
		return tsx.GetLanguage(), nil
	case Groovy:
		return groovy.GetLanguage(), nil
	case C:
		return c.GetLanguage(), nil
	case Cpp:
		return cpp.GetLanguage(), nil
	case CSharp:
		return csharp.GetLanguage(), nil
	case Ruby:
		return ruby.GetLanguage(), nil
	case PHP:
		// PHP has tree_sitter/ include path issues in Bazel
		return nil, ErrLanguageNotSupported{Language: lang, Backend: b.Name()}
	case Swift:
		return swift.GetLanguage(), nil
	case Bash:
		return bash.GetLanguage(), nil
	case HTML:
		return html.GetLanguage(), nil
	case CSS:
		return css.GetLanguage(), nil
	case SQL:
		// SQL has tree_sitter/ include path issues in Bazel
		return nil, ErrLanguageNotSupported{Language: lang, Backend: b.Name()}
	case YAML:
		return yaml.GetLanguage(), nil
	case TOML:
		return toml.GetLanguage(), nil
	case Markdown:
		// Markdown has tree_sitter/ include path issues in Bazel
		return nil, ErrLanguageNotSupported{Language: lang, Backend: b.Name()}
	case Protobuf:
		return protobuf.GetLanguage(), nil
	case HCL:
		return hcl.GetLanguage(), nil
	case Dockerfile:
		return dockerfile.GetLanguage(), nil
	case Lua:
		return lua.GetLanguage(), nil
	case Elixir:
		return elixir.GetLanguage(), nil
	case Elm:
		return elm.GetLanguage(), nil
	case OCaml:
		return ocaml.GetLanguage(), nil
	case Svelte:
		return svelte.GetLanguage(), nil
	case Cue:
		return cue.GetLanguage(), nil
	default:
		return nil, ErrLanguageNotSupported{Language: lang, Backend: b.Name()}
	}
}

func (b *cgoBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

// cgoParser implements Parser using the CGO backend.
type cgoParser struct {
	mu     sync.RWMutex
	parser *sitter.Parser
	lang   Language
	closed bool
}

func (p *cgoParser) Language() Language {
	return p.lang
}

func (p *cgoParser) Parse(ctx context.Context, source []byte) (Tree, error) {
	p.mu.RLock()
	closed := p.closed
	parser := p.parser
	p.mu.RUnlock()

	if closed {
		return nil, ErrParserClosed{}
	}

	tree, err := parser.ParseCtx(ctx, nil, source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &cgoTree{
		tree:   tree,
		source: source,
	}, nil
}

func (p *cgoParser) ParseString(ctx context.Context, source string) (Tree, error) {
	return p.Parse(ctx, []byte(source))
}

func (p *cgoParser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	p.parser.Close()
	return nil
}

// cgoTree implements Tree using the CGO backend.
type cgoTree struct {
	tree   *sitter.Tree
	source []byte
}

func (t *cgoTree) RootNode() Node {
	return &cgoNode{node: t.tree.RootNode()}
}

func (t *cgoTree) Source() []byte {
	return t.source
}

func (t *cgoTree) HasError() bool {
	root := t.tree.RootNode()
	if root == nil {
		return false
	}
	return root.HasError()
}

func (t *cgoTree) Close() error {
	t.tree.Close()
	return nil
}

// cgoNode implements Node using the CGO backend.
type cgoNode struct {
	node *sitter.Node
}

func (n *cgoNode) Type() string {
	if n.node == nil {
		return ""
	}
	return n.node.Type()
}

func (n *cgoNode) StartByte() uint32 {
	if n.node == nil {
		return 0
	}
	return n.node.StartByte()
}

func (n *cgoNode) EndByte() uint32 {
	if n.node == nil {
		return 0
	}
	return n.node.EndByte()
}

func (n *cgoNode) StartPoint() Point {
	if n.node == nil {
		return Point{}
	}
	p := n.node.StartPoint()
	return Point{Row: p.Row, Column: p.Column}
}

func (n *cgoNode) EndPoint() Point {
	if n.node == nil {
		return Point{}
	}
	p := n.node.EndPoint()
	return Point{Row: p.Row, Column: p.Column}
}

func (n *cgoNode) Content(source []byte) string {
	if n.node == nil {
		return ""
	}
	return n.node.Content(source)
}

func (n *cgoNode) ChildCount() uint32 {
	if n.node == nil {
		return 0
	}
	return n.node.ChildCount()
}

func (n *cgoNode) Child(index uint32) Node {
	if n.node == nil {
		return nil
	}
	child := n.node.Child(int(index))
	if child == nil {
		return nil
	}
	return &cgoNode{node: child}
}

func (n *cgoNode) NamedChildCount() uint32 {
	if n.node == nil {
		return 0
	}
	return n.node.NamedChildCount()
}

func (n *cgoNode) NamedChild(index uint32) Node {
	if n.node == nil {
		return nil
	}
	child := n.node.NamedChild(int(index))
	if child == nil {
		return nil
	}
	return &cgoNode{node: child}
}

func (n *cgoNode) ChildByFieldName(name string) Node {
	if n.node == nil {
		return nil
	}
	child := n.node.ChildByFieldName(name)
	if child == nil {
		return nil
	}
	return &cgoNode{node: child}
}

func (n *cgoNode) Parent() Node {
	if n.node == nil {
		return nil
	}
	parent := n.node.Parent()
	if parent == nil {
		return nil
	}
	return &cgoNode{node: parent}
}

func (n *cgoNode) NextSibling() Node {
	if n.node == nil {
		return nil
	}
	sibling := n.node.NextSibling()
	if sibling == nil {
		return nil
	}
	return &cgoNode{node: sibling}
}

func (n *cgoNode) PrevSibling() Node {
	if n.node == nil {
		return nil
	}
	sibling := n.node.PrevSibling()
	if sibling == nil {
		return nil
	}
	return &cgoNode{node: sibling}
}

func (n *cgoNode) NextNamedSibling() Node {
	if n.node == nil {
		return nil
	}
	sibling := n.node.NextNamedSibling()
	if sibling == nil {
		return nil
	}
	return &cgoNode{node: sibling}
}

func (n *cgoNode) PrevNamedSibling() Node {
	if n.node == nil {
		return nil
	}
	sibling := n.node.PrevNamedSibling()
	if sibling == nil {
		return nil
	}
	return &cgoNode{node: sibling}
}

func (n *cgoNode) IsNamed() bool {
	if n.node == nil {
		return false
	}
	return n.node.IsNamed()
}

func (n *cgoNode) IsError() bool {
	if n.node == nil {
		return false
	}
	return n.node.IsError()
}

func (n *cgoNode) IsMissing() bool {
	if n.node == nil {
		return false
	}
	return n.node.IsMissing()
}

func (n *cgoNode) IsNull() bool {
	return n.node == nil || n.node.IsNull()
}

func (n *cgoNode) String() string {
	if n.node == nil {
		return "(null)"
	}
	return n.node.String()
}

// cgoTreeCursor implements TreeCursor using the CGO backend.
type cgoTreeCursor struct {
	cursor *sitter.TreeCursor
}

// NewTreeCursor creates a new tree cursor starting at the given node.
// This is only available with the CGO backend.
func NewTreeCursor(node Node) TreeCursor {
	cgoNode, ok := node.(*cgoNode)
	if !ok {
		return nil
	}
	return &cgoTreeCursor{
		cursor: sitter.NewTreeCursor(cgoNode.node),
	}
}

func (c *cgoTreeCursor) Reset(node Node) {
	cgoNode, ok := node.(*cgoNode)
	if !ok {
		return
	}
	c.cursor.Reset(cgoNode.node)
}

func (c *cgoTreeCursor) CurrentNode() Node {
	node := c.cursor.CurrentNode()
	if node == nil {
		return nil
	}
	return &cgoNode{node: node}
}

func (c *cgoTreeCursor) CurrentFieldName() string {
	return c.cursor.CurrentFieldName()
}

func (c *cgoTreeCursor) GotoParent() bool {
	return c.cursor.GoToParent()
}

func (c *cgoTreeCursor) GotoFirstChild() bool {
	return c.cursor.GoToFirstChild()
}

func (c *cgoTreeCursor) GotoNextSibling() bool {
	return c.cursor.GoToNextSibling()
}

func (c *cgoTreeCursor) Close() {
	c.cursor.Close()
}
