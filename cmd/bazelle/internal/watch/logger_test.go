package watch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_Ready(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf})

	logger.Ready(100, []string{"go", "kotlin"}, "/path/to/workspace")

	output := buf.String()
	if !strings.Contains(output, "100 files") {
		t.Errorf("expected file count in output: %s", output)
	}
	if !strings.Contains(output, "/path/to/workspace") {
		t.Errorf("expected path in output: %s", output)
	}
	if !strings.Contains(output, "go") || !strings.Contains(output, "kotlin") {
		t.Errorf("expected languages in output: %s", output)
	}
	if !strings.Contains(output, "ready") {
		t.Errorf("expected 'ready' in output: %s", output)
	}
}

func TestLogger_Ready_NoLanguages(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf})

	logger.Ready(50, nil, "/workspace")

	output := buf.String()
	if !strings.Contains(output, "50 files") {
		t.Errorf("expected file count in output: %s", output)
	}
}

func TestLogger_FileChanged_Verbose(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, Verbose: true, NoColor: true})

	logger.FileChanged("src/main.go", ChangeAdded)

	output := buf.String()
	if !strings.Contains(output, "+") {
		t.Errorf("expected '+' indicator: %s", output)
	}
	if !strings.Contains(output, "src/main.go") {
		t.Errorf("expected path in output: %s", output)
	}
}

func TestLogger_FileChanged_NotVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, Verbose: false})

	logger.FileChanged("src/main.go", ChangeAdded)

	output := buf.String()
	if output != "" {
		t.Errorf("expected no output when not verbose, got: %s", output)
	}
}

func TestLogger_Updating(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf})

	logger.Updating([]string{"src"})

	output := buf.String()
	if !strings.Contains(output, "//src:all") {
		t.Errorf("expected target in output: %s", output)
	}
}

func TestLogger_Updating_Multiple(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf})

	logger.Updating([]string{"src", "lib", "pkg"})

	output := buf.String()
	if !strings.Contains(output, "3 directories") {
		t.Errorf("expected directory count in output: %s", output)
	}
}

func TestLogger_Updated(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, NoColor: true})

	logger.Updated("src/BUILD.bazel")

	output := buf.String()
	if !strings.Contains(output, "src/BUILD.bazel") {
		t.Errorf("expected build file in output: %s", output)
	}
	if !strings.Contains(output, "updated") {
		t.Errorf("expected 'updated' in output: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, NoColor: true})

	logger.Error(errTest{msg: "test error"})

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("expected error message in output: %s", output)
	}
	if !strings.Contains(output, "error") {
		t.Errorf("expected 'error' in output: %s", output)
	}
}

type errTest struct {
	msg string
}

func (e errTest) Error() string { return e.msg }

func TestLogger_Shutdown(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf})

	// Simulate some updates and errors
	logger.Updated("a/BUILD.bazel")
	logger.Updated("b/BUILD.bazel")
	logger.Error(errTest{msg: "oops"})

	logger.Shutdown()

	output := buf.String()
	if !strings.Contains(output, "2 updates") {
		t.Errorf("expected update count in output: %s", output)
	}
	if !strings.Contains(output, "1 errors") {
		t.Errorf("expected error count in output: %s", output)
	}
}

func TestLogger_Stats(t *testing.T) {
	logger := NewLogger(LoggerConfig{})

	stats := logger.Stats()
	if stats.UpdateCount != 0 {
		t.Errorf("expected 0 updates, got %d", stats.UpdateCount)
	}

	logger.Updated("test")
	logger.Updated("test2")
	logger.Error(errTest{})

	stats = logger.Stats()
	if stats.UpdateCount != 2 {
		t.Errorf("expected 2 updates, got %d", stats.UpdateCount)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", stats.ErrorCount)
	}
}

func TestLogger_JSON_Ready(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, JSON: true})

	logger.Ready(100, []string{"go"}, "/workspace")

	var event map[string]any
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event["event"] != "ready" {
		t.Errorf("expected event=ready, got %v", event["event"])
	}
	if event["files"].(float64) != 100 {
		t.Errorf("expected files=100, got %v", event["files"])
	}
}

func TestLogger_JSON_FileChanged(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, JSON: true, Verbose: true})

	logger.FileChanged("src/main.go", ChangeModified)

	var event map[string]any
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event["event"] != "file_changed" {
		t.Errorf("expected event=file_changed, got %v", event["event"])
	}
	if event["change"] != "~" {
		t.Errorf("expected change=~, got %v", event["change"])
	}
}

func TestLogger_JSON_Updated(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, JSON: true})

	logger.Updated("src/BUILD.bazel")

	var event map[string]any
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event["event"] != "updated" {
		t.Errorf("expected event=updated, got %v", event["event"])
	}
	if event["build_file"] != "src/BUILD.bazel" {
		t.Errorf("expected build_file=src/BUILD.bazel, got %v", event["build_file"])
	}
}

func TestLogger_JSON_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Writer: &buf, JSON: true})

	logger.Error(errTest{msg: "something failed"})

	var event map[string]any
	if err := json.Unmarshal(buf.Bytes(), &event); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if event["event"] != "error" {
		t.Errorf("expected event=error, got %v", event["event"])
	}
	if event["error"] != "something failed" {
		t.Errorf("expected error message, got %v", event["error"])
	}
}
