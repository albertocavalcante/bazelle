package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPipToModule(t *testing.T) {
	tests := []struct {
		pipName string
		want    string
	}{
		{"scikit-learn", "sklearn"},
		{"Pillow", "PIL"},
		{"PyYAML", "yaml"},
		{"beautifulsoup4", "bs4"},
		{"requests", "requests"},
		{"flask", "flask"},
		{"some-unknown-pkg", "some_unknown_pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.pipName, func(t *testing.T) {
			got := PipToModule(tt.pipName)
			if got != tt.want {
				t.Errorf("PipToModule(%q) = %q, want %q", tt.pipName, got, tt.want)
			}
		})
	}
}

func TestModuleToPip(t *testing.T) {
	tests := []struct {
		moduleName string
		want       string
	}{
		{"sklearn", "scikit-learn"},
		{"PIL", "pillow"},
		{"yaml", "pyyaml"},
		{"bs4", "beautifulsoup4"},
		{"requests", "requests"},
		{"some_module", "some-module"},
	}

	for _, tt := range tests {
		t.Run(tt.moduleName, func(t *testing.T) {
			got := ModuleToPip(tt.moduleName)
			if got != tt.want {
				t.Errorf("ModuleToPip(%q) = %q, want %q", tt.moduleName, got, tt.want)
			}
		})
	}
}

func TestRequirementsParser(t *testing.T) {
	// Create a temporary requirements.txt
	content := `# This is a comment
requests==2.28.0
flask>=2.0.0
numpy
scikit-learn[full]==1.0.0
pandas>=1.3,<2.0
python-dateutil
-r base.txt
# Another comment
aiohttp
`
	tmpDir := t.TempDir()
	reqFile := filepath.Join(tmpDir, "requirements.txt")
	err := os.WriteFile(reqFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewRequirementsParser()
	deps, err := parser.ParseFile(reqFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	expectedDeps := []struct {
		name       string
		version    string
		moduleName string
		extras     []string
	}{
		{"requests", "==2.28.0", "requests", nil},
		{"flask", ">=2.0.0", "flask", nil},
		{"numpy", "", "numpy", nil},
		{"scikit-learn", "==1.0.0", "sklearn", []string{"full"}},
		{"pandas", ">=1.3,<2.0", "pandas", nil},
		{"python-dateutil", "", "dateutil", nil},
		{"aiohttp", "", "aiohttp", nil},
	}

	if len(deps) != len(expectedDeps) {
		t.Fatalf("ParseFile() returned %d deps, want %d", len(deps), len(expectedDeps))
	}

	for i, expected := range expectedDeps {
		if deps[i].Name != expected.name {
			t.Errorf("deps[%d].Name = %q, want %q", i, deps[i].Name, expected.name)
		}
		if deps[i].Version != expected.version {
			t.Errorf("deps[%d].Version = %q, want %q", i, deps[i].Version, expected.version)
		}
		if deps[i].ModuleName != expected.moduleName {
			t.Errorf("deps[%d].ModuleName = %q, want %q", i, deps[i].ModuleName, expected.moduleName)
		}
		if len(deps[i].Extras) != len(expected.extras) {
			t.Errorf("deps[%d].Extras = %v, want %v", i, deps[i].Extras, expected.extras)
		}
	}
}

func TestPipConfigGetPipLabel(t *testing.T) {
	pc := &PipConfig{
		PipRepository: "pip",
		Dependencies: []PipDependency{
			{Name: "requests", ModuleName: "requests"},
			{Name: "scikit-learn", ModuleName: "sklearn"},
			{Name: "beautifulsoup4", ModuleName: "bs4"},
		},
	}

	tests := []struct {
		moduleName string
		want       string
	}{
		{"requests", "@pip//requests"},
		{"sklearn", "@pip//scikit_learn"},
		{"bs4", "@pip//beautifulsoup4"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.moduleName, func(t *testing.T) {
			got := pc.GetPipLabel(tt.moduleName)
			if got != tt.want {
				t.Errorf("GetPipLabel(%q) = %q, want %q", tt.moduleName, got, tt.want)
			}
		})
	}
}

func TestParseLineWithExtras(t *testing.T) {
	parser := NewRequirementsParser()

	tests := []struct {
		line       string
		wantName   string
		wantExtras []string
	}{
		{"requests[security]", "requests", []string{"security"}},
		{"celery[redis,sqs]>=4.0", "celery", []string{"redis", "sqs"}},
		{"black", "black", nil},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			dep := parser.parseLine(tt.line)
			if dep.Name != tt.wantName {
				t.Errorf("parseLine(%q).Name = %q, want %q", tt.line, dep.Name, tt.wantName)
			}
			if len(dep.Extras) != len(tt.wantExtras) {
				t.Errorf("parseLine(%q).Extras = %v, want %v", tt.line, dep.Extras, tt.wantExtras)
			}
		})
	}
}
