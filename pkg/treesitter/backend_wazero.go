package treesitter

import (
	"context"
	"fmt"
	"sync"

	sitter "github.com/malivvan/tree-sitter"
)

// wazeroBackend implements Backend using the WASM/wazero-based malivvan/tree-sitter library.
// This backend does not require CGO, making it suitable for environments where CGO is
// not available or desirable. However, it currently only supports C and C++ languages.
type wazeroBackend struct {
	mu        sync.RWMutex
	ctx       context.Context
	ts        sitter.TreeSitter
	closed    bool
	languages map[Language]sitter.Language
}

// NewWazeroBackend creates a new WASM/wazero-based tree-sitter backend.
// This backend uses malivvan/tree-sitter which runs tree-sitter in WASM via wazero,
// eliminating the need for CGO. Currently only C and C++ are supported.
func NewWazeroBackend() (Backend, error) {
	ctx := context.Background()
	ts, err := sitter.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tree-sitter wazero runtime: %w", err)
	}

	b := &wazeroBackend{
		ctx:       ctx,
		ts:        ts,
		languages: make(map[Language]sitter.Language),
	}

	// Pre-load supported languages
	if langC, err := ts.LanguageC(ctx); err == nil {
		b.languages[C] = langC
	}
	if langCpp, err := ts.LanguageCpp(ctx); err == nil {
		b.languages[Cpp] = langCpp
	}

	return b, nil
}

func (b *wazeroBackend) Name() string {
	return "wazero"
}

func (b *wazeroBackend) IsExperimental() bool {
	return true
}

func (b *wazeroBackend) SupportedLanguages() []Language {
	b.mu.RLock()
	defer b.mu.RUnlock()

	langs := make([]Language, 0, len(b.languages))
	for lang := range b.languages {
		langs = append(langs, lang)
	}
	return langs
}

func (b *wazeroBackend) SupportsLanguage(lang Language) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.languages[lang]
	return ok
}

func (b *wazeroBackend) NewParser(lang Language) (Parser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, ErrBackendClosed{Backend: b.Name()}
	}

	sitterLang, ok := b.languages[lang]
	if !ok {
		return nil, ErrLanguageNotSupported{Language: lang, Backend: b.Name()}
	}

	parser, err := b.ts.NewParser(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	if err := parser.SetLanguage(b.ctx, sitterLang); err != nil {
		_ = parser.Close(b.ctx)
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	return &wazeroParser{
		ctx:    b.ctx,
		parser: parser,
		lang:   lang,
	}, nil
}

func (b *wazeroBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	// The TreeSitter instance will be garbage collected
	return nil
}

// wazeroParser implements Parser using the wazero backend.
type wazeroParser struct {
	mu     sync.RWMutex
	ctx    context.Context
	parser sitter.Parser
	lang   Language
	closed bool
}

func (p *wazeroParser) Language() Language {
	return p.lang
}

func (p *wazeroParser) Parse(ctx context.Context, source []byte) (Tree, error) {
	p.mu.RLock()
	closed := p.closed
	parser := p.parser
	p.mu.RUnlock()

	if closed {
		return nil, ErrParserClosed{}
	}

	tree, err := parser.ParseString(ctx, string(source))
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return &wazeroTree{
		ctx:    ctx,
		tree:   tree,
		source: source,
	}, nil
}

func (p *wazeroParser) ParseString(ctx context.Context, source string) (Tree, error) {
	return p.Parse(ctx, []byte(source))
}

func (p *wazeroParser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.parser.Close(p.ctx)
}

// wazeroTree implements Tree using the wazero backend.
type wazeroTree struct {
	ctx    context.Context
	tree   sitter.Tree
	source []byte
}

func (t *wazeroTree) RootNode() Node {
	node, err := t.tree.RootNode(t.ctx)
	if err != nil {
		return &wazeroNode{ctx: t.ctx, isNull: true}
	}
	return &wazeroNode{ctx: t.ctx, node: node}
}

func (t *wazeroTree) Source() []byte {
	return t.source
}

func (t *wazeroTree) HasError() bool {
	// Walk the tree to check for errors since wazero doesn't provide
	// a direct HasError method on the tree
	root := t.RootNode()
	return HasErrors(root)
}

func (t *wazeroTree) Close() error {
	// wazero tree doesn't have explicit close
	return nil
}

// wazeroNode implements Node using the wazero backend.
type wazeroNode struct {
	ctx    context.Context
	node   sitter.Node
	isNull bool
}

func (n *wazeroNode) Type() string {
	if n.isNull {
		return ""
	}
	kind, err := n.node.Kind(n.ctx)
	if err != nil {
		return ""
	}
	return kind
}

func (n *wazeroNode) StartByte() uint32 {
	if n.isNull {
		return 0
	}
	start, err := n.node.StartByte(n.ctx)
	if err != nil {
		return 0
	}
	return uint32(start)
}

func (n *wazeroNode) EndByte() uint32 {
	if n.isNull {
		return 0
	}
	end, err := n.node.EndByte(n.ctx)
	if err != nil {
		return 0
	}
	return uint32(end)
}

func (n *wazeroNode) StartPoint() Point {
	// wazero backend doesn't expose point information directly
	// Return zero point as a fallback
	return Point{}
}

func (n *wazeroNode) EndPoint() Point {
	// wazero backend doesn't expose point information directly
	// Return zero point as a fallback
	return Point{}
}

func (n *wazeroNode) Content(source []byte) string {
	if n.isNull {
		return ""
	}
	start := n.StartByte()
	end := n.EndByte()
	if start > uint32(len(source)) || end > uint32(len(source)) || start > end {
		return ""
	}
	return string(source[start:end])
}

func (n *wazeroNode) ChildCount() uint32 {
	if n.isNull {
		return 0
	}
	count, err := n.node.ChildCount(n.ctx)
	if err != nil {
		return 0
	}
	return uint32(count)
}

func (n *wazeroNode) Child(index uint32) Node {
	if n.isNull {
		return nil
	}
	// Bounds check to ensure consistent behavior across backends
	if index >= n.ChildCount() {
		return nil
	}
	child, err := n.node.Child(n.ctx, uint64(index))
	if err != nil {
		return nil
	}
	return &wazeroNode{ctx: n.ctx, node: child}
}

func (n *wazeroNode) NamedChildCount() uint32 {
	if n.isNull {
		return 0
	}
	count, err := n.node.NamedChildCount(n.ctx)
	if err != nil {
		return 0
	}
	return uint32(count)
}

func (n *wazeroNode) NamedChild(index uint32) Node {
	if n.isNull {
		return nil
	}
	// Bounds check to ensure consistent behavior across backends
	if index >= n.NamedChildCount() {
		return nil
	}
	child, err := n.node.NamedChild(n.ctx, uint64(index))
	if err != nil {
		return nil
	}
	return &wazeroNode{ctx: n.ctx, node: child}
}

func (n *wazeroNode) ChildByFieldName(name string) Node {
	// wazero backend doesn't support field name lookup
	// Iterate through children to find by field name is not efficient
	// Return nil as this feature is not supported
	return nil
}

func (n *wazeroNode) Parent() Node {
	// wazero backend doesn't expose parent navigation
	return nil
}

func (n *wazeroNode) NextSibling() Node {
	// wazero backend doesn't expose sibling navigation
	return nil
}

func (n *wazeroNode) PrevSibling() Node {
	// wazero backend doesn't expose sibling navigation
	return nil
}

func (n *wazeroNode) NextNamedSibling() Node {
	// wazero backend doesn't expose sibling navigation
	return nil
}

func (n *wazeroNode) PrevNamedSibling() Node {
	// wazero backend doesn't expose sibling navigation
	return nil
}

func (n *wazeroNode) IsNamed() bool {
	// wazero backend doesn't expose this directly
	// Named nodes typically have non-empty types that don't start with _
	nodeType := n.Type()
	return nodeType != "" && nodeType[0] != '_'
}

func (n *wazeroNode) IsError() bool {
	if n.isNull {
		return false
	}
	isErr, err := n.node.IsError(n.ctx)
	if err != nil {
		return false
	}
	return isErr
}

func (n *wazeroNode) IsMissing() bool {
	// wazero backend doesn't expose this
	return false
}

func (n *wazeroNode) IsNull() bool {
	return n.isNull
}

func (n *wazeroNode) String() string {
	if n.isNull {
		return "(null)"
	}
	str, err := n.node.String(n.ctx)
	if err != nil {
		return "(error)"
	}
	return str
}

// wazeroTreeCursor is not implemented for the wazero backend
// as it has limited traversal capabilities.
// Users should use direct node traversal methods instead.
