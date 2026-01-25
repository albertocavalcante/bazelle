package treesitter

import (
	"context"
	"os"
	"testing"
)

// testBackend runs tests against a specific backend.
func testBackend(t *testing.T, backend Backend, lang Language, source string, expectedRootType string) {
	t.Helper()

	t.Logf("Testing backend %s with language %s", backend.Name(), lang)

	// Test backend info
	t.Run("BackendInfo", func(t *testing.T) {
		if backend.Name() == "" {
			t.Error("backend name should not be empty")
		}
		langs := backend.SupportedLanguages()
		if len(langs) == 0 {
			t.Error("backend should support at least one language")
		}
	})

	// Test language support
	t.Run("SupportsLanguage", func(t *testing.T) {
		if !backend.SupportsLanguage(lang) {
			t.Errorf("backend %s should support language %s", backend.Name(), lang)
		}
	})

	// Test parser creation
	parser, err := backend.NewParser(lang)
	if err != nil {
		t.Fatalf("NewParser(%s) failed: %v", lang, err)
	}
	defer parser.Close()

	if parser.Language() != lang {
		t.Errorf("parser.Language() = %s, want %s", parser.Language(), lang)
	}

	// Test parsing
	t.Run("Parse", func(t *testing.T) {
		ctx := context.Background()
		tree, err := parser.Parse(ctx, []byte(source))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		defer tree.Close()

		if tree.Source() == nil {
			t.Error("tree.Source() should not be nil")
		}
		if string(tree.Source()) != source {
			t.Errorf("tree.Source() = %q, want %q", string(tree.Source()), source)
		}

		root := tree.RootNode()
		if root == nil {
			t.Fatal("root node should not be nil")
		}
		if root.IsNull() {
			t.Fatal("root node should not be null")
		}

		rootType := root.Type()
		if rootType != expectedRootType {
			t.Errorf("root.Type() = %q, want %q", rootType, expectedRootType)
		}

		// Test node content
		if root.Content(tree.Source()) == "" {
			t.Error("root.Content() should not be empty")
		}

		// Test child access
		if root.ChildCount() == 0 {
			t.Error("root should have children")
		}

		child := root.Child(0)
		if child == nil {
			t.Error("first child should not be nil")
		}
	})

	// Test ParseString
	t.Run("ParseString", func(t *testing.T) {
		ctx := context.Background()
		tree, err := parser.ParseString(ctx, source)
		if err != nil {
			t.Fatalf("ParseString failed: %v", err)
		}
		defer tree.Close()

		root := tree.RootNode()
		if root.Type() != expectedRootType {
			t.Errorf("root.Type() = %q, want %q", root.Type(), expectedRootType)
		}
	})
}

func TestCGOBackend(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	if backend.Name() != "cgo" {
		t.Errorf("backend.Name() = %q, want %q", backend.Name(), "cgo")
	}
	if backend.IsExperimental() {
		t.Error("CGO backend should not be experimental")
	}

	// Test multiple languages
	testCases := []struct {
		lang     Language
		source   string
		rootType string
	}{
		{
			lang:     Go,
			source:   "package main\n\nfunc main() {}\n",
			rootType: "source_file",
		},
		{
			lang:     JavaScript,
			source:   "const x = 1;\n",
			rootType: "program",
		},
		{
			lang:     Python,
			source:   "def hello():\n    print('hello')\n",
			rootType: "module",
		},
		{
			lang:     Java,
			source:   "public class Hello { public static void main(String[] args) {} }",
			rootType: "program",
		},
		{
			lang:     C,
			source:   "int main() { return 0; }",
			rootType: "translation_unit",
		},
		{
			lang:     Rust,
			source:   "fn main() {}",
			rootType: "source_file",
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.lang), func(t *testing.T) {
			if !backend.SupportsLanguage(tc.lang) {
				t.Skipf("language %s not supported", tc.lang)
			}
			testBackend(t, backend, tc.lang, tc.source, tc.rootType)
		})
	}
}

func TestWazeroBackend(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}
	defer backend.Close()

	if backend.Name() != "wazero" {
		t.Errorf("backend.Name() = %q, want %q", backend.Name(), "wazero")
	}
	if !backend.IsExperimental() {
		t.Error("wazero backend should be experimental")
	}

	// Test C language
	testCases := []struct {
		lang     Language
		source   string
		rootType string
	}{
		{
			lang:     C,
			source:   "int main() { return 0; }",
			rootType: "translation_unit",
		},
		{
			lang:     Cpp,
			source:   "int main() { return 0; }",
			rootType: "translation_unit",
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.lang), func(t *testing.T) {
			if !backend.SupportsLanguage(tc.lang) {
				t.Skipf("language %s not supported", tc.lang)
			}
			testBackend(t, backend, tc.lang, tc.source, tc.rootType)
		})
	}
}

func TestWazeroBackendUnsupportedLanguage(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}
	defer backend.Close()

	// Go is not supported by wazero backend
	if backend.SupportsLanguage(Go) {
		t.Error("wazero backend should not support Go")
	}

	_, err = backend.NewParser(Go)
	if err == nil {
		t.Error("expected error for unsupported language")
	}

	var langErr ErrLanguageNotSupported
	if _, ok := err.(ErrLanguageNotSupported); !ok {
		t.Errorf("expected ErrLanguageNotSupported, got %T: %v", err, err)
	} else {
		langErr = err.(ErrLanguageNotSupported)
		if langErr.Language != Go {
			t.Errorf("langErr.Language = %s, want %s", langErr.Language, Go)
		}
		if langErr.Backend != "wazero" {
			t.Errorf("langErr.Backend = %s, want %s", langErr.Backend, "wazero")
		}
	}
}

func TestNewBackend(t *testing.T) {
	testCases := []struct {
		typ      BackendType
		wantName string
		wantErr  bool
	}{
		{BackendCGO, "cgo", false},
		{BackendWazero, "wazero", false},
		{BackendAuto, "", false}, // Name depends on what's available
		{"invalid", "", true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.typ), func(t *testing.T) {
			backend, err := NewBackend(tc.typ)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				if tc.typ == BackendCGO {
					// CGO might not be available
					t.Skipf("CGO backend not available: %v", err)
				}
				t.Fatalf("NewBackend(%s) failed: %v", tc.typ, err)
			}
			defer backend.Close()

			if tc.wantName != "" && backend.Name() != tc.wantName {
				t.Errorf("backend.Name() = %q, want %q", backend.Name(), tc.wantName)
			}
		})
	}
}

func TestNewBackendFromEnv(t *testing.T) {
	// Save original env var
	origVal := os.Getenv(EnvVarBackend)
	defer os.Setenv(EnvVarBackend, origVal)

	testCases := []struct {
		envVal   string
		wantName string
		wantErr  bool
	}{
		{"", "", false},        // Default to auto
		{"auto", "", false},    // Explicit auto
		{"cgo", "cgo", false},  // CGO might not be available
		{"wazero", "wazero", false},
		{"CGO", "cgo", false},  // Case insensitive
		{"WAZERO", "wazero", false},
		{"invalid", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.envVal, func(t *testing.T) {
			os.Setenv(EnvVarBackend, tc.envVal)

			backend, err := NewBackendFromEnv()
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				if tc.envVal == "cgo" || tc.envVal == "CGO" {
					t.Skipf("CGO backend not available: %v", err)
				}
				t.Fatalf("NewBackendFromEnv() failed: %v", err)
			}
			defer backend.Close()

			if tc.wantName != "" && backend.Name() != tc.wantName {
				t.Errorf("backend.Name() = %q, want %q", backend.Name(), tc.wantName)
			}
		})
	}
}

func TestBackendInfo(t *testing.T) {
	cgoInfo := GetBackendInfo(BackendCGO)
	if cgoInfo.Name != "CGO" {
		t.Errorf("cgoInfo.Name = %q, want %q", cgoInfo.Name, "CGO")
	}
	if cgoInfo.IsExperimental {
		t.Error("CGO backend should not be experimental")
	}
	if len(cgoInfo.SupportedLanguages) == 0 {
		t.Error("CGO backend should support languages")
	}

	wazeroInfo := GetBackendInfo(BackendWazero)
	if wazeroInfo.Name != "Wazero" {
		t.Errorf("wazeroInfo.Name = %q, want %q", wazeroInfo.Name, "Wazero")
	}
	if !wazeroInfo.IsExperimental {
		t.Error("Wazero backend should be experimental")
	}
	if len(wazeroInfo.SupportedLanguages) == 0 {
		t.Error("Wazero backend should support some languages")
	}
}

func TestParserClose(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(C)
	if err != nil {
		t.Fatalf("NewParser(C) failed: %v", err)
	}

	// Close the parser
	if err := parser.Close(); err != nil {
		t.Errorf("parser.Close() failed: %v", err)
	}

	// Parsing should fail after close
	ctx := context.Background()
	_, err = parser.Parse(ctx, []byte("int x;"))
	if err == nil {
		t.Error("expected error when parsing with closed parser")
	}
}

func TestBackendClose(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}

	// Close the backend
	if err := backend.Close(); err != nil {
		t.Errorf("backend.Close() failed: %v", err)
	}

	// Creating a parser should fail after close
	_, err = backend.NewParser(C)
	if err == nil {
		t.Error("expected error when creating parser with closed backend")
	}
}

func TestNodeTraversal(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Go)
	if err != nil {
		t.Fatalf("NewParser(Go) failed: %v", err)
	}
	defer parser.Close()

	source := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte(source))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	// Test various node methods
	t.Run("NodeMethods", func(t *testing.T) {
		if root.IsNull() {
			t.Error("root should not be null")
		}
		if root.IsError() {
			t.Error("root should not be an error node")
		}
		if root.IsMissing() {
			t.Error("root should not be missing")
		}
		if !root.IsNamed() {
			t.Error("root should be named")
		}
		if root.Parent() != nil {
			t.Error("root should not have a parent")
		}
	})

	// Test child traversal
	t.Run("ChildTraversal", func(t *testing.T) {
		count := root.ChildCount()
		if count == 0 {
			t.Error("root should have children")
		}

		namedCount := root.NamedChildCount()
		if namedCount == 0 {
			t.Error("root should have named children")
		}

		for i := uint32(0); i < count; i++ {
			child := root.Child(i)
			if child == nil {
				t.Errorf("child %d should not be nil", i)
			}
		}

		for i := uint32(0); i < namedCount; i++ {
			child := root.NamedChild(i)
			if child == nil {
				t.Errorf("named child %d should not be nil", i)
			}
		}
	})

	// Test sibling traversal
	t.Run("SiblingTraversal", func(t *testing.T) {
		firstChild := root.NamedChild(0)
		if firstChild == nil {
			t.Fatal("first named child should not be nil")
		}

		sibling := firstChild.NextNamedSibling()
		if sibling == nil {
			t.Error("first child should have a next named sibling")
		}

		if sibling != nil {
			prevSibling := sibling.PrevNamedSibling()
			if prevSibling == nil {
				t.Error("second child should have a previous named sibling")
			}
		}
	})

	// Test content extraction
	t.Run("Content", func(t *testing.T) {
		content := root.Content(tree.Source())
		if content != source {
			t.Errorf("root content mismatch")
		}
	})

	// Test S-expression output
	t.Run("String", func(t *testing.T) {
		str := root.String()
		if str == "" {
			t.Error("String() should return non-empty S-expression")
		}
		if str == "(null)" || str == "(error)" {
			t.Errorf("String() returned unexpected value: %s", str)
		}
	})
}

func TestTreeCursor(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Go)
	if err != nil {
		t.Fatalf("NewParser(Go) failed: %v", err)
	}
	defer parser.Close()

	source := `package main

func main() {}
`
	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte(source))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	cursor := NewTreeCursor(root)
	if cursor == nil {
		t.Skip("TreeCursor not available")
	}
	defer cursor.Close()

	// Test cursor traversal
	node := cursor.CurrentNode()
	if node == nil {
		t.Fatal("current node should not be nil")
	}
	if node.Type() != "source_file" {
		t.Errorf("current node type = %q, want %q", node.Type(), "source_file")
	}

	// Go to first child
	if !cursor.GotoFirstChild() {
		t.Error("should be able to go to first child")
	}

	// Go to next sibling
	if cursor.GotoNextSibling() {
		// Good, there's a sibling
	}

	// Go back to parent
	if !cursor.GotoParent() {
		t.Error("should be able to go to parent")
	}

	if cursor.CurrentNode().Type() != "source_file" {
		t.Error("should be back at root")
	}
}

func TestAllLanguages(t *testing.T) {
	langs := AllLanguages()
	if len(langs) == 0 {
		t.Error("AllLanguages() should return non-empty list")
	}

	// Check that basic languages are included
	basicLangs := []Language{Go, Java, Python, JavaScript, C, Rust}
	for _, lang := range basicLangs {
		found := false
		for _, l := range langs {
			if l == lang {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("language %s should be in AllLanguages()", lang)
		}
	}
}

func TestErrorTypes(t *testing.T) {
	// Test ErrLanguageNotSupported
	err := ErrLanguageNotSupported{Language: Go, Backend: "test"}
	if err.Error() != "language go is not supported by backend test" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Test ErrBackendClosed
	err2 := ErrBackendClosed{Backend: "test"}
	if err2.Error() != "backend test has been closed" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}

	// Test ErrParserClosed
	err3 := ErrParserClosed{}
	if err3.Error() != "parser has been closed" {
		t.Errorf("unexpected error message: %s", err3.Error())
	}
}

func TestEmptyInput(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(C)
	if err != nil {
		t.Fatalf("NewParser(C) failed: %v", err)
	}
	defer parser.Close()

	ctx := context.Background()

	// Test empty input
	tree, err := parser.Parse(ctx, []byte(""))
	if err != nil {
		t.Fatalf("Parse with empty input failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("root should not be nil even for empty input")
	}

	// Empty C file should produce a translation_unit with no children
	if root.Type() != "translation_unit" {
		t.Errorf("root.Type() = %q, want %q", root.Type(), "translation_unit")
	}
}

func TestSyntaxErrors(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Go)
	if err != nil {
		t.Fatalf("NewParser(Go) failed: %v", err)
	}
	defer parser.Close()

	ctx := context.Background()

	// Invalid Go code with syntax errors
	source := `package main

func main( {
	// missing closing paren
}
`
	tree, err := parser.Parse(ctx, []byte(source))
	if err != nil {
		t.Fatalf("Parse failed (syntax errors should be captured in tree): %v", err)
	}
	defer tree.Close()

	// Tree should have errors
	if !tree.HasError() {
		t.Error("tree.HasError() should return true for invalid syntax")
	}

	// Also test the helper function
	if !HasErrors(tree.RootNode()) {
		t.Error("HasErrors() should return true for invalid syntax")
	}
}

func TestOutOfBoundsChildAccess(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(C)
	if err != nil {
		t.Fatalf("NewParser(C) failed: %v", err)
	}
	defer parser.Close()

	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte("int x;"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	// Access child at invalid index
	invalidChild := root.Child(9999)
	if invalidChild != nil {
		t.Error("Child(9999) should return nil for out of bounds access")
	}

	// Access named child at invalid index
	invalidNamedChild := root.NamedChild(9999)
	if invalidNamedChild != nil {
		t.Error("NamedChild(9999) should return nil for out of bounds access")
	}
}

func TestHelperFunctions(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Go)
	if err != nil {
		t.Fatalf("NewParser(Go) failed: %v", err)
	}
	defer parser.Close()

	source := `package main

import (
	"fmt"
	"os"
)

func hello() {}
func world() {}
`
	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte(source))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	// Test Children
	t.Run("Children", func(t *testing.T) {
		children := Children(root)
		if len(children) == 0 {
			t.Error("Children() should return non-empty slice")
		}
	})

	// Test NamedChildren
	t.Run("NamedChildren", func(t *testing.T) {
		namedChildren := NamedChildren(root)
		if len(namedChildren) == 0 {
			t.Error("NamedChildren() should return non-empty slice")
		}
	})

	// Test ChildrenByType
	t.Run("ChildrenByType", func(t *testing.T) {
		funcDecls := ChildrenByType(root, "function_declaration")
		if len(funcDecls) != 2 {
			t.Errorf("ChildrenByType for function_declaration: got %d, want 2", len(funcDecls))
		}
	})

	// Test FindByType
	t.Run("FindByType", func(t *testing.T) {
		identifiers := FindByType(root, "identifier")
		if len(identifiers) == 0 {
			t.Error("FindByType should find identifiers")
		}
	})

	// Test FindFirst
	t.Run("FindFirst", func(t *testing.T) {
		found := FindFirst(root, func(n Node) bool {
			return n.Type() == "import_declaration"
		})
		if found == nil {
			t.Error("FindFirst should find import_declaration")
		}
	})

	// Test Walk
	t.Run("Walk", func(t *testing.T) {
		count := 0
		Walk(root, func(n Node) bool {
			count++
			return true
		})
		if count == 0 {
			t.Error("Walk should visit nodes")
		}
	})

	// Test Walk with early termination
	t.Run("WalkEarlyTermination", func(t *testing.T) {
		count := 0
		completed := Walk(root, func(n Node) bool {
			count++
			return count < 3 // stop after 3 nodes
		})
		if completed {
			t.Error("Walk should return false when terminated early")
		}
		if count < 3 {
			t.Errorf("Walk should have visited at least 3 nodes, got %d", count)
		}
	})

	// Test helpers with nil node
	t.Run("NilNodeHelpers", func(t *testing.T) {
		if Children(nil) != nil {
			t.Error("Children(nil) should return nil")
		}
		if NamedChildren(nil) != nil {
			t.Error("NamedChildren(nil) should return nil")
		}
		if ChildrenByType(nil, "test") != nil {
			t.Error("ChildrenByType(nil, ...) should return nil")
		}
		if FindFirst(nil, func(n Node) bool { return true }) != nil {
			t.Error("FindFirst(nil, ...) should return nil")
		}
		if HasErrors(nil) {
			t.Error("HasErrors(nil) should return false")
		}
	})
}

func TestAvailableBackends(t *testing.T) {
	backends := AvailableBackends()
	if len(backends) == 0 {
		t.Error("AvailableBackends() should return at least one backend")
	}

	// Wazero should always be available
	hasWazero := false
	for _, b := range backends {
		if b == BackendWazero {
			hasWazero = true
			break
		}
	}
	if !hasWazero {
		t.Error("Wazero backend should always be available")
	}
}

func TestMustNewBackendPanic(t *testing.T) {
	// Test that MustNewBackend panics on invalid backend
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNewBackend should panic on invalid backend type")
		}
	}()
	MustNewBackend("invalid")
}

func TestDoubleClose(t *testing.T) {
	backend, err := NewWazeroBackend()
	if err != nil {
		t.Fatalf("NewWazeroBackend failed: %v", err)
	}

	parser, err := backend.NewParser(C)
	if err != nil {
		t.Fatalf("NewParser(C) failed: %v", err)
	}

	ctx := context.Background()
	tree, err := parser.Parse(ctx, []byte("int x;"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Close everything twice - should not panic
	if err := tree.Close(); err != nil {
		t.Errorf("First tree.Close() failed: %v", err)
	}
	if err := tree.Close(); err != nil {
		t.Errorf("Second tree.Close() failed: %v", err)
	}

	if err := parser.Close(); err != nil {
		t.Errorf("First parser.Close() failed: %v", err)
	}
	if err := parser.Close(); err != nil {
		t.Errorf("Second parser.Close() failed: %v", err)
	}

	if err := backend.Close(); err != nil {
		t.Errorf("First backend.Close() failed: %v", err)
	}
	if err := backend.Close(); err != nil {
		t.Errorf("Second backend.Close() failed: %v", err)
	}
}

// TestRealWorldImportExtraction demonstrates extracting imports from Go code,
// which is a common use case for gazelle extensions.
func TestRealWorldImportExtraction(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Go)
	if err != nil {
		t.Fatalf("NewParser(Go) failed: %v", err)
	}
	defer parser.Close()

	source := `package main

import (
	"context"
	"fmt"
	"os"

	"github.com/example/foo"
	bar "github.com/example/bar"
)

func main() {
	ctx := context.Background()
	fmt.Println(ctx, foo.X, bar.Y, os.Args)
}
`
	ctx := context.Background()
	tree, err := parser.ParseString(ctx, source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	sourceBytes := tree.Source()

	// Find all import specs
	var imports []string
	importSpecs := FindByType(root, "import_spec")
	for _, spec := range importSpecs {
		// The path is in the "path" field or is a string child
		pathNode := spec.ChildByFieldName("path")
		if pathNode != nil {
			// Remove quotes from the import path
			path := pathNode.Content(sourceBytes)
			if len(path) >= 2 {
				path = path[1 : len(path)-1] // strip quotes
			}
			imports = append(imports, path)
		}
	}

	expectedImports := []string{
		"context",
		"fmt",
		"os",
		"github.com/example/foo",
		"github.com/example/bar",
	}

	if len(imports) != len(expectedImports) {
		t.Errorf("found %d imports, want %d: %v", len(imports), len(expectedImports), imports)
	}

	for i, expected := range expectedImports {
		if i >= len(imports) {
			break
		}
		if imports[i] != expected {
			t.Errorf("import[%d] = %q, want %q", i, imports[i], expected)
		}
	}
}

// TestRealWorldJavaImports demonstrates extracting imports from Java code.
func TestRealWorldJavaImports(t *testing.T) {
	backend, err := NewCGOBackend()
	if err != nil {
		t.Skipf("CGO backend not available: %v", err)
	}
	defer backend.Close()

	parser, err := backend.NewParser(Java)
	if err != nil {
		t.Fatalf("NewParser(Java) failed: %v", err)
	}
	defer parser.Close()

	source := `package com.example;

import java.util.List;
import java.util.Map;
import com.google.common.collect.ImmutableList;
import static org.junit.Assert.assertEquals;

public class Example {
    public static void main(String[] args) {
        List<String> items = ImmutableList.of("a", "b");
    }
}
`
	ctx := context.Background()
	tree, err := parser.ParseString(ctx, source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	sourceBytes := tree.Source()

	// Find all import declarations
	var imports []string
	importDecls := FindByType(root, "import_declaration")
	for _, decl := range importDecls {
		// Get the full import text minus 'import' keyword and semicolon
		content := decl.Content(sourceBytes)
		imports = append(imports, content)
	}

	if len(imports) != 4 {
		t.Errorf("found %d imports, want 4: %v", len(imports), imports)
	}

	// Verify we can distinguish static imports
	hasStaticImport := false
	for _, imp := range imports {
		if len(imp) > 7 && imp[7:13] == "static" {
			hasStaticImport = true
			break
		}
	}
	if !hasStaticImport {
		t.Error("should have found a static import")
	}
}

// BenchmarkParsing benchmarks parsing performance for both backends.
func BenchmarkParsing(b *testing.B) {
	source := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}
`)

	cgoBackend, err := NewCGOBackend()
	if err == nil {
		b.Run("CGO", func(b *testing.B) {
			parser, _ := cgoBackend.NewParser(Go)
			defer parser.Close()
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree, _ := parser.Parse(ctx, source)
				tree.Close()
			}
		})
		cgoBackend.Close()
	}
}
