package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bazelbuild/bazel-gazelle/runner"
)

var updateFlags struct {
	check     bool
	languages []string
	verbose   bool
}

var updateCmd = &cobra.Command{
	Use:   "update [path...]",
	Short: "Update BUILD files",
	Long: `Updates BUILD files by running gazelle update.

The --check flag can be used in CI to verify BUILD files are up to date
without making changes.`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateFlags.check, "check", false,
		"Check if BUILD files are up to date (exit 1 if changes needed)")
	updateCmd.Flags().StringSliceVar(&updateFlags.languages, "languages", nil,
		"Only run specific language extensions (comma-separated)")
	updateCmd.Flags().BoolVarP(&updateFlags.verbose, "verbose", "v", false,
		"Show detailed output")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	wd, err := runner.GetDefaultWorkspaceDirectory()
	if err != nil {
		return err
	}

	// Build gazelle arguments
	gazelleArgs := []string{"update"}

	if updateFlags.check {
		gazelleArgs = append(gazelleArgs, "-mode=diff")
	}

	// Add path arguments
	if len(args) > 0 {
		gazelleArgs = append(gazelleArgs, args...)
	}

	if updateFlags.check {
		return runUpdateCheck(wd, gazelleArgs)
	}

	// Normal update: run gazelle
	return runner.Run(languages, wd, gazelleArgs...)
}

func runUpdateCheck(wd string, args []string) error {
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
		// There are changes needed
		if updateFlags.verbose {
			fmt.Fprintln(os.Stderr, "BUILD files need updating:")
			fmt.Fprintln(os.Stderr, strings.TrimSpace(string(output)))
		} else {
			// Count files that would change
			lines := strings.Split(string(output), "\n")
			fileCount := 0
			for _, line := range lines {
				if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
					fileCount++
				}
			}
			fileCount /= 2 // Each file has --- and +++
			if fileCount > 0 {
				fmt.Fprintf(os.Stderr, "BUILD files need updating (%d file(s) would change)\n", fileCount)
			} else {
				fmt.Fprintln(os.Stderr, "BUILD files need updating")
			}
		}
		fmt.Fprintln(os.Stderr, "Run 'bazelle update' to apply changes")
		os.Exit(1)
	}

	if runErr != nil {
		return fmt.Errorf("gazelle failed: %w", runErr)
	}

	fmt.Println("BUILD files are up to date")
	return nil
}
