package groovy

import (
	"testing"
)

func TestParsePackage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "simple package",
			content: "package com.example\n\nclass Foo {}",
			want:    "com.example",
		},
		{
			name:    "package with subpackages",
			content: "package com.example.app.service\n\nclass Service {}",
			want:    "com.example.app.service",
		},
		{
			name:    "no package",
			content: "class Foo {}",
			want:    "",
		},
		{
			name:    "package with leading whitespace",
			content: "  package com.example\n\nclass Foo {}",
			want:    "com.example",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseContent(tt.content, "test.groovy")
			if err != nil {
				t.Fatalf("ParseContent() error = %v", err)
			}
			if result.Package != tt.want {
				t.Errorf("Package = %q, want %q", result.Package, tt.want)
			}
		})
	}
}

func TestParseImports(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantImports []string
	}{
		{
			name:        "single import",
			content:     "package com.example\n\nimport com.other.Foo\n\nclass Bar {}",
			wantImports: []string{"com.other.Foo"},
		},
		{
			name: "multiple imports",
			content: `package com.example

import com.other.Foo
import com.another.Bar
import org.lib.Baz

class Test {}`,
			wantImports: []string{"com.other.Foo", "com.another.Bar", "org.lib.Baz"},
		},
		{
			name:        "no imports",
			content:     "package com.example\n\nclass Foo {}",
			wantImports: []string{},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseContent(tt.content, "test.groovy")
			if err != nil {
				t.Fatalf("ParseContent() error = %v", err)
			}
			if len(result.Imports) != len(tt.wantImports) {
				t.Errorf("Imports count = %d, want %d", len(result.Imports), len(tt.wantImports))
				return
			}
			for i, imp := range tt.wantImports {
				if result.Imports[i] != imp {
					t.Errorf("Imports[%d] = %q, want %q", i, result.Imports[i], imp)
				}
			}
		})
	}
}

func TestParseStarImports(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantStarImports []string
	}{
		{
			name:            "single star import",
			content:         "package com.example\n\nimport com.other.*\n\nclass Bar {}",
			wantStarImports: []string{"com.other"},
		},
		{
			name: "mixed imports",
			content: `package com.example

import com.other.*
import com.specific.Foo
import org.lib.*

class Test {}`,
			wantStarImports: []string{"com.other", "org.lib"},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseContent(tt.content, "test.groovy")
			if err != nil {
				t.Fatalf("ParseContent() error = %v", err)
			}
			if len(result.StarImports) != len(tt.wantStarImports) {
				t.Errorf("StarImports count = %d, want %d", len(result.StarImports), len(tt.wantStarImports))
				return
			}
			for i, imp := range tt.wantStarImports {
				if result.StarImports[i] != imp {
					t.Errorf("StarImports[%d] = %q, want %q", i, result.StarImports[i], imp)
				}
			}
		})
	}
}

func TestParseStaticImports(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		wantStaticImports []string
	}{
		{
			name:              "static import",
			content:           "package com.example\n\nimport static com.other.Util.helper\n\nclass Bar {}",
			wantStaticImports: []string{"com.other.Util.helper"},
		},
		{
			name: "multiple static imports",
			content: `package com.example

import static com.other.Util.helper
import static org.junit.Assert.assertEquals

class Test {}`,
			wantStaticImports: []string{"com.other.Util.helper", "org.junit.Assert.assertEquals"},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseContent(tt.content, "test.groovy")
			if err != nil {
				t.Fatalf("ParseContent() error = %v", err)
			}
			if len(result.StaticImports) != len(tt.wantStaticImports) {
				t.Errorf("StaticImports count = %d, want %d", len(result.StaticImports), len(tt.wantStaticImports))
				return
			}
			for i, imp := range tt.wantStaticImports {
				if result.StaticImports[i] != imp {
					t.Errorf("StaticImports[%d] = %q, want %q", i, result.StaticImports[i], imp)
				}
			}
		})
	}
}

func TestParseGrabAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantGrab []GrabDependency
	}{
		{
			name:    "short form grab",
			content: "@Grab('org.codehaus.groovy:groovy-all:3.0.9')\nclass Foo {}",
			wantGrab: []GrabDependency{
				{Group: "org.codehaus.groovy", Module: "groovy-all", Version: "3.0.9"},
			},
		},
		{
			name:    "double quotes grab",
			content: `@Grab("com.google.guava:guava:31.0-jre")\nclass Foo {}`,
			wantGrab: []GrabDependency{
				{Group: "com.google.guava", Module: "guava", Version: "31.0-jre"},
			},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseContent(tt.content, "test.groovy")
			if err != nil {
				t.Fatalf("ParseContent() error = %v", err)
			}
			if len(result.GrabDeps) != len(tt.wantGrab) {
				t.Errorf("GrabDeps count = %d, want %d", len(result.GrabDeps), len(tt.wantGrab))
				return
			}
			for i, grab := range tt.wantGrab {
				if result.GrabDeps[i].Group != grab.Group {
					t.Errorf("GrabDeps[%d].Group = %q, want %q", i, result.GrabDeps[i].Group, grab.Group)
				}
				if result.GrabDeps[i].Module != grab.Module {
					t.Errorf("GrabDeps[%d].Module = %q, want %q", i, result.GrabDeps[i].Module, grab.Module)
				}
				if result.GrabDeps[i].Version != grab.Version {
					t.Errorf("GrabDeps[%d].Version = %q, want %q", i, result.GrabDeps[i].Version, grab.Version)
				}
			}
		})
	}
}

func TestStripComments(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		inBlock bool
		want    string
		outBlock bool
	}{
		{
			name:     "no comments",
			line:     "def x = 1",
			inBlock:  false,
			want:     "def x = 1",
			outBlock: false,
		},
		{
			name:     "line comment",
			line:     "def x = 1 // comment",
			inBlock:  false,
			want:     "def x = 1 ",
			outBlock: false,
		},
		{
			name:     "block comment start",
			line:     "def x = /* comment */ 1",
			inBlock:  false,
			want:     "def x =  1",
			outBlock: false,
		},
		{
			name:     "in block comment",
			line:     "comment content */",
			inBlock:  true,
			want:     "",
			outBlock: false,
		},
		{
			name:     "string with //",
			line:     `def url = "https://example.com"`,
			inBlock:  false,
			want:     `def url = "https://example.com"`,
			outBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotBlock := stripComments(tt.line, tt.inBlock)
			if got != tt.want {
				t.Errorf("stripComments() got = %q, want %q", got, tt.want)
			}
			if gotBlock != tt.outBlock {
				t.Errorf("stripComments() gotBlock = %v, want %v", gotBlock, tt.outBlock)
			}
		})
	}
}

func TestIsGroovyStdlib(t *testing.T) {
	tests := []struct {
		imp  string
		want bool
	}{
		{"groovy.lang.Closure", true},
		{"java.util.List", true},
		{"javax.swing.JFrame", true},
		{"org.codehaus.groovy.ast.ClassNode", true},
		{"com.example.Foo", false},
		{"org.junit.Test", false},
		{"spock.lang.Specification", false},
	}

	for _, tt := range tests {
		t.Run(tt.imp, func(t *testing.T) {
			got := IsGroovyStdlib(tt.imp)
			if got != tt.want {
				t.Errorf("IsGroovyStdlib(%q) = %v, want %v", tt.imp, got, tt.want)
			}
		})
	}
}

func TestIsGroovyBuiltinType(t *testing.T) {
	tests := []struct {
		typeName string
		want     bool
	}{
		{"String", true},
		{"Integer", true},
		{"List", true},
		{"Map", true},
		{"Closure", true},
		{"GString", true},
		{"MyClass", false},
		{"CustomType", false},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := IsGroovyBuiltinType(tt.typeName)
			if got != tt.want {
				t.Errorf("IsGroovyBuiltinType(%q) = %v, want %v", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestExtractPackageFromFQN(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{"com.example.Foo", "com.example"},
		{"com.example.sub.Bar", "com.example.sub"},
		{"Foo", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fqn, func(t *testing.T) {
			got := ExtractPackageFromFQN(tt.fqn)
			if got != tt.want {
				t.Errorf("ExtractPackageFromFQN(%q) = %q, want %q", tt.fqn, got, tt.want)
			}
		})
	}
}

func TestExtractClassFromFQN(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{"com.example.Foo", "Foo"},
		{"com.example.sub.Bar", "Bar"},
		{"Foo", "Foo"},
		{"com.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fqn, func(t *testing.T) {
			got := ExtractClassFromFQN(tt.fqn)
			if got != tt.want {
				t.Errorf("ExtractClassFromFQN(%q) = %q, want %q", tt.fqn, got, tt.want)
			}
		})
	}
}
