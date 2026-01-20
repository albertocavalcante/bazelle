package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/runner"
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

Use --dry-run to preview changes without applying them.`,
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVar(&fixFlags.check, "check", false,
		"Check if BUILD files need fixing (exit 1 if changes needed)")
	fixCmd.Flags().BoolVar(&fixFlags.dryRun, "dry-run", false,
		"Show what would change without applying")
	fixCmd.Flags().BoolVarP(&fixFlags.verbose, "verbose", "v", false,
		"Show detailed output")

	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	r := runner.New()

	// Build gazelle arguments
	gazelleArgs := []string{"fix"}

	if fixFlags.check || fixFlags.dryRun {
		gazelleArgs = append(gazelleArgs, "-mode=diff")
	}

	// Add path arguments
	if len(args) > 0 {
		gazelleArgs = append(gazelleArgs, args...)
	}

	if fixFlags.check {
		return runFixCheck(r, gazelleArgs)
	}

	if fixFlags.dryRun {
		return runFixDryRun(r, gazelleArgs)
	}

	// Normal fix: exec gazelle (replaces process)
	return r.Exec(gazelleArgs)
}

func runFixCheck(r *runner.Runner, args []string) error {
	output, err := r.RunWithOutput(args)

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

	if err != nil {
		return fmt.Errorf("gazelle failed: %w", err)
	}

	fmt.Println("BUILD files are up to date")
	return nil
}

func runFixDryRun(r *runner.Runner, args []string) error {
	output, err := r.RunWithOutput(args)
	if err != nil {
		return fmt.Errorf("gazelle failed: %w", err)
	}

	if len(output) > 0 {
		fmt.Print(string(output))
	} else {
		fmt.Println("No changes needed")
	}
	return nil
}
