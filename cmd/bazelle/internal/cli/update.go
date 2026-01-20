package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/runner"
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
	r := runner.New()

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
		return runUpdateCheck(r, gazelleArgs)
	}

	// Normal update: exec gazelle (replaces process)
	return r.Exec(gazelleArgs)
}

func runUpdateCheck(r *runner.Runner, args []string) error {
	output, err := r.RunWithOutput(args)

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

	if err != nil {
		return fmt.Errorf("gazelle failed: %w", err)
	}

	fmt.Println("BUILD files are up to date")
	return nil
}
