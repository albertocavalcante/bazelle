package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/spf13/cobra"
)

// ============================================================================
// Update Command Tests
// ============================================================================

func TestUpdateCmd_FlagDefaults(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("update command not found")
	}

	tests := []struct {
		name         string
		flagName     string
		wantDefault  string
		wantShortcut string
	}{
		{
			name:         "check flag defaults to false",
			flagName:     "check",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "verbose flag defaults to false",
			flagName:     "verbose",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "incremental flag defaults to false",
			flagName:     "incremental",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "force flag defaults to false",
			flagName:     "force",
			wantDefault:  "false",
			wantShortcut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on update command", tt.flagName)
			}

			if flag.DefValue != tt.wantDefault {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.wantDefault)
			}

			if flag.Shorthand != tt.wantShortcut {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.wantShortcut)
			}
		})
	}
}

func TestUpdateCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("update command not found")
	}

	if cmd.Use != "update [path...]" {
		t.Errorf("update command Use = %q, want %q", cmd.Use, "update [path...]")
	}

	if cmd.Short != "Update BUILD files" {
		t.Errorf("update command Short = %q, want %q", cmd.Short, "Update BUILD files")
	}
}

func TestUpdateCmd_AllowsUnknownFlags(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("update command not found")
	}

	// Verify that unknown flags are whitelisted (for passthrough to gazelle)
	if !cmd.FParseErrWhitelist.UnknownFlags {
		t.Error("update command should allow unknown flags for gazelle passthrough")
	}
}

func TestUpdateCmd_LanguagesFlag(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("update command not found")
	}

	flag := cmd.Flags().Lookup("languages")
	if flag == nil {
		t.Fatal("languages flag not found on update command")
	}

	// Languages flag should accept slice values
	if flag.Value.Type() != "stringSlice" {
		t.Errorf("languages flag type = %q, want %q", flag.Value.Type(), "stringSlice")
	}
}

// ============================================================================
// Fix Command Tests
// ============================================================================

func TestFixCmd_FlagDefaults(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "fix" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("fix command not found")
	}

	tests := []struct {
		name         string
		flagName     string
		wantDefault  string
		wantShortcut string
	}{
		{
			name:         "check flag defaults to false",
			flagName:     "check",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "dry-run flag defaults to false",
			flagName:     "dry-run",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "verbose flag defaults to false",
			flagName:     "verbose",
			wantDefault:  "false",
			wantShortcut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on fix command", tt.flagName)
			}

			if flag.DefValue != tt.wantDefault {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.wantDefault)
			}

			if flag.Shorthand != tt.wantShortcut {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.wantShortcut)
			}
		})
	}
}

func TestFixCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "fix" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("fix command not found")
	}

	if cmd.Use != "fix [path...]" {
		t.Errorf("fix command Use = %q, want %q", cmd.Use, "fix [path...]")
	}

	expectedShort := "Fix BUILD files (may make breaking changes)"
	if cmd.Short != expectedShort {
		t.Errorf("fix command Short = %q, want %q", cmd.Short, expectedShort)
	}
}

func TestFixCmd_AllowsUnknownFlags(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "fix" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("fix command not found")
	}

	// Verify that unknown flags are whitelisted (for passthrough to gazelle)
	if !cmd.FParseErrWhitelist.UnknownFlags {
		t.Error("fix command should allow unknown flags for gazelle passthrough")
	}
}

// ============================================================================
// Status Command Tests
// ============================================================================

func TestStatusCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "status" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("status command not found")
	}

	if cmd.Use != "status" {
		t.Errorf("status command Use = %q, want %q", cmd.Use, "status")
	}

	expectedShort := "Show which directories have stale BUILD files"
	if cmd.Short != expectedShort {
		t.Errorf("status command Short = %q, want %q", cmd.Short, expectedShort)
	}
}

func TestStatusCmd_FlagDefaults(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "status" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("status command not found")
	}

	tests := []struct {
		name         string
		flagName     string
		wantDefault  string
		wantShortcut string
	}{
		{
			name:         "verbose flag defaults to false",
			flagName:     "verbose",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "json flag defaults to false",
			flagName:     "json",
			wantDefault:  "false",
			wantShortcut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on status command", tt.flagName)
			}

			if flag.DefValue != tt.wantDefault {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.wantDefault)
			}

			if flag.Shorthand != tt.wantShortcut {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.wantShortcut)
			}
		})
	}
}

func TestStatusCmd_LongDescription(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "status" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("status command not found")
	}

	// Verify long description contains key information
	if cmd.Long == "" {
		t.Error("status command should have a long description")
	}
}

// ============================================================================
// Gazelle Command Tests
// ============================================================================

func TestGazelleCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "gazelle" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("gazelle command not found")
	}

	if cmd.Use != "gazelle [args...]" {
		t.Errorf("gazelle command Use = %q, want %q", cmd.Use, "gazelle [args...]")
	}

	expectedShort := "Run gazelle directly (raw passthrough)"
	if cmd.Short != expectedShort {
		t.Errorf("gazelle command Short = %q, want %q", cmd.Short, expectedShort)
	}
}

func TestGazelleCmd_DisablesFlagParsing(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "gazelle" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("gazelle command not found")
	}

	// Verify that flag parsing is disabled (all args go to gazelle)
	if !cmd.DisableFlagParsing {
		t.Error("gazelle command should have DisableFlagParsing=true for raw passthrough")
	}
}

func TestGazelleCmd_LongDescription(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "gazelle" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("gazelle command not found")
	}

	// Verify long description mentions it's a raw passthrough
	if cmd.Long == "" {
		t.Error("gazelle command should have a long description")
	}
}

// ============================================================================
// Watch Command Tests
// ============================================================================

func TestWatchCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "watch" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("watch command not found")
	}

	if cmd.Use != "watch [path]" {
		t.Errorf("watch command Use = %q, want %q", cmd.Use, "watch [path]")
	}

	expectedShort := "Watch for source file changes and auto-update BUILD files"
	if cmd.Short != expectedShort {
		t.Errorf("watch command Short = %q, want %q", cmd.Short, expectedShort)
	}
}

func TestWatchCmd_FlagDefaults(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "watch" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("watch command not found")
	}

	tests := []struct {
		name         string
		flagName     string
		wantDefault  string
		wantShortcut string
	}{
		{
			name:         "debounce flag defaults to 500",
			flagName:     "debounce",
			wantDefault:  "500",
			wantShortcut: "",
		},
		{
			name:         "verbose flag defaults to false",
			flagName:     "verbose",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "json flag defaults to false",
			flagName:     "json",
			wantDefault:  "false",
			wantShortcut: "",
		},
		{
			name:         "no-color flag defaults to false",
			flagName:     "no-color",
			wantDefault:  "false",
			wantShortcut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on watch command", tt.flagName)
			}

			if flag.DefValue != tt.wantDefault {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.wantDefault)
			}

			if flag.Shorthand != tt.wantShortcut {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.wantShortcut)
			}
		})
	}
}

func TestWatchCmd_LanguagesFlag(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "watch" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("watch command not found")
	}

	flag := cmd.Flags().Lookup("languages")
	if flag == nil {
		t.Fatal("languages flag not found on watch command")
	}

	// Languages flag should accept slice values
	if flag.Value.Type() != "stringSlice" {
		t.Errorf("languages flag type = %q, want %q", flag.Value.Type(), "stringSlice")
	}
}

// ============================================================================
// Version Command Tests
// ============================================================================

func TestVersionCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	var cmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "version" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("version command not found")
	}

	if cmd.Use != "version" {
		t.Errorf("version command Use = %q, want %q", cmd.Use, "version")
	}

	expectedShort := "Print version information"
	if cmd.Short != expectedShort {
		t.Errorf("version command Short = %q, want %q", cmd.Short, expectedShort)
	}
}

// ============================================================================
// Root Command Tests
// ============================================================================

func TestRootCmd_UseAndShort(t *testing.T) {
	root := RootCmd()

	if root.Use != "bazelle" {
		t.Errorf("root command Use = %q, want %q", root.Use, "bazelle")
	}

	expectedShort := "Polyglot BUILD file generator"
	if root.Short != expectedShort {
		t.Errorf("root command Short = %q, want %q", root.Short, expectedShort)
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	root := RootCmd()

	tests := []struct {
		name         string
		flagName     string
		wantDefault  string
		wantShortcut string
	}{
		{
			name:         "verbosity flag defaults to 1",
			flagName:     "verbosity",
			wantDefault:  "1",
			wantShortcut: "v",
		},
		{
			name:         "log-format flag defaults to text",
			flagName:     "log-format",
			wantDefault:  "text",
			wantShortcut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := root.PersistentFlags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on root command", tt.flagName)
			}

			if flag.DefValue != tt.wantDefault {
				t.Errorf("flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.wantDefault)
			}

			if flag.Shorthand != tt.wantShortcut {
				t.Errorf("flag %q shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.wantShortcut)
			}
		})
	}
}

// ============================================================================
// GazelleDefaults Tests
// ============================================================================

func TestGazelleDefaults(t *testing.T) {
	// Verify the gazelle defaults are set as expected
	expectedDefaults := []string{
		"-go_naming_convention=import",
		"-go_naming_convention_external=import",
	}

	if len(GazelleDefaults) != len(expectedDefaults) {
		t.Errorf("GazelleDefaults length = %d, want %d", len(GazelleDefaults), len(expectedDefaults))
	}

	for i, expected := range expectedDefaults {
		if i >= len(GazelleDefaults) {
			break
		}
		if GazelleDefaults[i] != expected {
			t.Errorf("GazelleDefaults[%d] = %q, want %q", i, GazelleDefaults[i], expected)
		}
	}
}

// ============================================================================
// StatusOutput Tests
// ============================================================================

func TestStatusOutput_Structure(t *testing.T) {
	// Test that StatusOutput can be created with expected fields
	output := StatusOutput{
		Stale:         true,
		StaleDirs:     []string{"dir1", "dir2"},
		NewFiles:      []string{"new.go"},
		ModifiedFiles: []string{"mod.go"},
		DeletedFiles:  []string{"del.go"},
		Error:         "test error",
	}

	if !output.Stale {
		t.Error("expected Stale to be true")
	}

	if len(output.StaleDirs) != 2 {
		t.Errorf("StaleDirs length = %d, want 2", len(output.StaleDirs))
	}

	if len(output.NewFiles) != 1 {
		t.Errorf("NewFiles length = %d, want 1", len(output.NewFiles))
	}

	if len(output.ModifiedFiles) != 1 {
		t.Errorf("ModifiedFiles length = %d, want 1", len(output.ModifiedFiles))
	}

	if len(output.DeletedFiles) != 1 {
		t.Errorf("DeletedFiles length = %d, want 1", len(output.DeletedFiles))
	}

	if output.Error != "test error" {
		t.Errorf("Error = %q, want %q", output.Error, "test error")
	}
}

func TestStatusOutput_EmptyState(t *testing.T) {
	// Test that StatusOutput works with empty/default values
	output := StatusOutput{}

	if output.Stale {
		t.Error("expected Stale to be false by default")
	}

	if output.StaleDirs != nil {
		t.Error("expected StaleDirs to be nil by default")
	}

	if output.Error != "" {
		t.Error("expected Error to be empty by default")
	}
}

// ============================================================================
// Command Count Tests
// ============================================================================

func TestExpectedSubcommandCount(t *testing.T) {
	root := RootCmd()

	// Count all subcommands
	subcommands := root.Commands()

	// We expect: update, fix, watch, status, gazelle, version, init
	expectedCommands := []string{"update", "fix", "watch", "status", "gazelle", "version", "init"}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

// ============================================================================
// Helper function to get a specific command by name
// ============================================================================

func getCommand(name string) *cobra.Command {
	root := RootCmd()
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestGetCommand_Helper(t *testing.T) {
	// Test the helper function works correctly
	cmd := getCommand("update")
	if cmd == nil {
		t.Fatal("getCommand(\"update\") returned nil")
	}

	if cmd.Name() != "update" {
		t.Errorf("getCommand returned wrong command: %q", cmd.Name())
	}

	// Test with non-existent command
	cmd = getCommand("nonexistent")
	if cmd != nil {
		t.Error("getCommand should return nil for non-existent command")
	}
}

// ============================================================================
// SetLanguages Tests
// ============================================================================

func TestSetLanguages(t *testing.T) {
	// Save original languages
	originalLangs := languages

	// Test setting languages
	mockLangs := []language.Language{}
	SetLanguages(mockLangs)

	if languages == nil {
		t.Error("SetLanguages should set the languages variable")
	}

	// Restore original languages
	languages = originalLangs
}

func TestSetLanguages_Nil(t *testing.T) {
	// Save original languages
	originalLangs := languages

	// Test setting nil
	SetLanguages(nil)

	if languages != nil {
		t.Error("SetLanguages(nil) should set languages to nil")
	}

	// Restore original languages
	languages = originalLangs
}

// ============================================================================
// outputJSON Tests
// ============================================================================

func TestOutputJSON(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	// Create a test StatusOutput
	output := StatusOutput{
		Stale:     true,
		StaleDirs: []string{"pkg/foo", "pkg/bar"},
	}

	// Call outputJSON
	err = outputJSON(output)

	// Close writer and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputJSON() error = %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Verify output is valid JSON
	var decoded StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Errorf("outputJSON produced invalid JSON: %v", err)
	}

	if !decoded.Stale {
		t.Error("decoded Stale should be true")
	}

	if len(decoded.StaleDirs) != 2 {
		t.Errorf("decoded StaleDirs length = %d, want 2", len(decoded.StaleDirs))
	}
}

func TestOutputJSON_EmptyOutput(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	// Create an empty StatusOutput
	output := StatusOutput{}

	// Call outputJSON
	err = outputJSON(output)

	// Close writer and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("outputJSON() error = %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Verify output is valid JSON
	var decoded StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Errorf("outputJSON produced invalid JSON: %v", err)
	}

	if decoded.Stale {
		t.Error("decoded Stale should be false for empty output")
	}
}

// ============================================================================
// StatusOutput JSON Encoding Tests
// ============================================================================

func TestStatusOutput_JSONTags(t *testing.T) {
	output := StatusOutput{
		Stale:         true,
		StaleDirs:     []string{"dir1"},
		NewFiles:      []string{"new.go"},
		ModifiedFiles: []string{"mod.go"},
		DeletedFiles:  []string{"del.go"},
		Error:         "test",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify JSON field names
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	expectedKeys := []string{"stale", "stale_dirs", "new_files", "modified_files", "deleted_files", "error"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("expected JSON key %q not found", key)
		}
	}
}

func TestStatusOutput_JSONOmitEmpty(t *testing.T) {
	output := StatusOutput{
		Stale:     false,
		StaleDirs: nil,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Verify omitempty fields are omitted
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// These fields should be omitted due to omitempty
	omitEmptyFields := []string{"new_files", "modified_files", "deleted_files", "error"}
	for _, key := range omitEmptyFields {
		if _, ok := m[key]; ok {
			t.Errorf("expected JSON key %q to be omitted (omitempty)", key)
		}
	}
}

// ============================================================================
// Command Long Description Tests
// ============================================================================

func TestUpdateCmd_LongDescription(t *testing.T) {
	cmd := getCommand("update")
	if cmd == nil {
		t.Fatal("update command not found")
	}

	// Verify long description contains key information
	if cmd.Long == "" {
		t.Error("update command should have a long description")
	}

	// Check for key content in the long description
	expectedContent := []string{"--check", "--incremental", "--force", "gazelle"}
	for _, content := range expectedContent {
		if !contains(cmd.Long, content) {
			t.Errorf("update command long description should mention %q", content)
		}
	}
}

func TestFixCmd_LongDescription(t *testing.T) {
	cmd := getCommand("fix")
	if cmd == nil {
		t.Fatal("fix command not found")
	}

	// Verify long description contains key information
	if cmd.Long == "" {
		t.Error("fix command should have a long description")
	}

	// Check for key content in the long description
	expectedContent := []string{"--dry-run", "breaking changes", "gazelle"}
	for _, content := range expectedContent {
		if !contains(cmd.Long, content) {
			t.Errorf("fix command long description should mention %q", content)
		}
	}
}

func TestWatchCmd_LongDescription(t *testing.T) {
	cmd := getCommand("watch")
	if cmd == nil {
		t.Fatal("watch command not found")
	}

	// Verify long description contains key information
	if cmd.Long == "" {
		t.Error("watch command should have a long description")
	}

	// Check for key content in the long description
	expectedContent := []string{"Watches", "BUILD files", "Ctrl+C"}
	for _, content := range expectedContent {
		if !contains(cmd.Long, content) {
			t.Errorf("watch command long description should mention %q", content)
		}
	}
}

// ============================================================================
// Command Has RunE Tests
// ============================================================================

func TestCommands_HaveRunE(t *testing.T) {
	commands := []string{"update", "fix", "watch", "status", "gazelle"}

	for _, cmdName := range commands {
		t.Run(cmdName, func(t *testing.T) {
			cmd := getCommand(cmdName)
			if cmd == nil {
				t.Fatalf("command %q not found", cmdName)
			}

			if cmd.RunE == nil {
				t.Errorf("command %q should have RunE defined", cmdName)
			}
		})
	}
}

// ============================================================================
// Flag Usage Tests
// ============================================================================

func TestUpdateCmd_FlagUsage(t *testing.T) {
	cmd := getCommand("update")
	if cmd == nil {
		t.Fatal("update command not found")
	}

	tests := []struct {
		flagName     string
		expectedDesc string
	}{
		{"check", "Check if BUILD files are up to date"},
		{"verbose", "Show detailed output"},
		{"incremental", "Only update directories with changed source files"},
		{"force", "Force full update, ignoring cached state"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if !contains(flag.Usage, tt.expectedDesc) {
				t.Errorf("flag %q usage = %q, want to contain %q", tt.flagName, flag.Usage, tt.expectedDesc)
			}
		})
	}
}

func TestFixCmd_FlagUsage(t *testing.T) {
	cmd := getCommand("fix")
	if cmd == nil {
		t.Fatal("fix command not found")
	}

	tests := []struct {
		flagName     string
		expectedDesc string
	}{
		{"check", "Check if BUILD files need fixing"},
		{"dry-run", "Show what would change without applying"},
		{"verbose", "Show detailed output"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if !contains(flag.Usage, tt.expectedDesc) {
				t.Errorf("flag %q usage = %q, want to contain %q", tt.flagName, flag.Usage, tt.expectedDesc)
			}
		})
	}
}

func TestWatchCmd_FlagUsage(t *testing.T) {
	cmd := getCommand("watch")
	if cmd == nil {
		t.Fatal("watch command not found")
	}

	tests := []struct {
		flagName     string
		expectedDesc string
	}{
		{"debounce", "Debounce window in milliseconds"},
		{"verbose", "Show file-level changes"},
		{"json", "Stream JSON events"},
		{"no-color", "Disable colored output"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if !contains(flag.Usage, tt.expectedDesc) {
				t.Errorf("flag %q usage = %q, want to contain %q", tt.flagName, flag.Usage, tt.expectedDesc)
			}
		})
	}
}

func TestStatusCmd_FlagUsage(t *testing.T) {
	cmd := getCommand("status")
	if cmd == nil {
		t.Fatal("status command not found")
	}

	tests := []struct {
		flagName     string
		expectedDesc string
	}{
		{"verbose", "Show individual file changes"},
		{"json", "Output as JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found", tt.flagName)
			}

			if !contains(flag.Usage, tt.expectedDesc) {
				t.Errorf("flag %q usage = %q, want to contain %q", tt.flagName, flag.Usage, tt.expectedDesc)
			}
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
