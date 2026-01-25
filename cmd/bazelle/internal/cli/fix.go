package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bazelbuild/bazel-gazelle/runner"
)

var fixFlags struct {
	check   bool
	dryRun  bool
	verbose bool
}

var fixCmd = &cobra.Command{
	Use:   "fix [path...]",
	Short: "Fix BUILD files (may make breaking changes)",
	Long: `Fixes BUILD files by running gazelle fix.

Unlike 'update', fix may make potentially breaking changes such as
deleting obsolete rules or renaming existing rules.

Use --dry-run to preview changes without applying them.

Additional gazelle flags (like -bzlmod, -go_prefix) are passed through.`,
	RunE:                       runFix,
	FParseErrWhitelist:         cobra.FParseErrWhitelist{UnknownFlags: true},
	DisableFlagsInUseLine:      true,
}

func init() {
	fixCmd.Flags().BoolVar(&fixFlags.check, "check", false,
		"Check if BUILD files need fixing (exit 1 if changes needed)")
	fixCmd.Flags().BoolVar(&fixFlags.dryRun, "dry-run", false,
		"Show what would change without applying")
	fixCmd.Flags().BoolVar(&fixFlags.verbose, "verbose", false,
		"Show detailed output")

	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	wd, err := runner.GetDefaultWorkspaceDirectory()
	if err != nil {
		return err
	}

	// Build gazelle arguments: "fix" + defaults + mode + passthrough args
	// Note: gazelle expects command first, then flags
	gazelleArgs := []string{"fix"}
	gazelleArgs = append(gazelleArgs, GazelleDefaults...)

	if fixFlags.check || fixFlags.dryRun {
		gazelleArgs = append(gazelleArgs, "-mode=diff")
	}

	// Add passthrough args (unknown flags + paths)
	if len(args) > 0 {
		gazelleArgs = append(gazelleArgs, args...)
	}

	if fixFlags.check {
		return runFixCheck(wd, gazelleArgs)
	}

	if fixFlags.dryRun {
		return runFixDryRun(wd, gazelleArgs)
	}

	// Normal fix: run gazelle
	return runner.Run(languages, wd, gazelleArgs...)
}

func runFixCheck(wd string, args []string) error {
	// Capture output by redirecting stdout/stderr
	var buf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	os.Stdout = w
	os.Stderr = w

	// Run gazelle
	runErr := runner.Run(languages, wd, args...)

	// Restore stdout/stderr
	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	_, _ = buf.ReadFrom(r)
	output := buf.Bytes()

	if len(output) > 0 {
		if fixFlags.verbose {
			fmt.Fprintln(os.Stderr, "BUILD files need fixing:")
			fmt.Fprintln(os.Stderr, strings.TrimSpace(string(output)))
		} else {
			fmt.Fprintln(os.Stderr, "BUILD files need fixing")
		}
		fmt.Fprintln(os.Stderr, "Run 'bazelle fix' to apply changes")
		os.Exit(1)
	}

	if runErr != nil {
		return fmt.Errorf("gazelle failed: %w", runErr)
	}

	fmt.Println("BUILD files are up to date")
	return nil
}

func runFixDryRun(wd string, args []string) error {
	// Capture output by redirecting stdout/stderr
	var buf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	os.Stdout = w
	os.Stderr = w

	// Run gazelle
	runErr := runner.Run(languages, wd, args...)

	// Restore stdout/stderr
	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	_, _ = buf.ReadFrom(r)
	output := buf.Bytes()

	if runErr != nil {
		return fmt.Errorf("gazelle failed: %w", runErr)
	}

	if len(output) > 0 {
		fmt.Print(string(output))
	} else {
		fmt.Println("No changes needed")
	}
	return nil
}
