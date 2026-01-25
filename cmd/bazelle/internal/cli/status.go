package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/incremental"
	"github.com/bazelbuild/bazel-gazelle/runner"
	"github.com/spf13/cobra"
)

var statusFlags struct {
	verbose bool
	json    bool
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which directories have stale BUILD files",
	Long: `Shows the status of BUILD files in the workspace.

Compares the current state of source files against the last 'bazelle update'
to identify directories that need BUILD file regeneration.

The --verbose flag shows individual file changes (new, modified, deleted).
The --json flag outputs the result as JSON for scripting.`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVarP(&statusFlags.verbose, "verbose", "v", false,
		"Show individual file changes")
	statusCmd.Flags().BoolVar(&statusFlags.json, "json", false,
		"Output as JSON")

	rootCmd.AddCommand(statusCmd)
}

// StatusOutput is the JSON output format for bazelle status.
type StatusOutput struct {
	Stale         bool     `json:"stale"`
	StaleDirs     []string `json:"stale_dirs"`
	NewFiles      []string `json:"new_files,omitempty"`
	ModifiedFiles []string `json:"modified_files,omitempty"`
	DeletedFiles  []string `json:"deleted_files,omitempty"`
	Error         string   `json:"error,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	wd, err := runner.GetDefaultWorkspaceDirectory()
	if err != nil {
		return err
	}

	ctx := context.Background()
	tracker := incremental.NewTracker(wd, updateFlags.languages)

	// Check if state file exists
	if !tracker.HasState() {
		if statusFlags.json {
			output := StatusOutput{
				Stale:     true,
				StaleDirs: []string{"."},
				Error:     "no state found",
			}
			return outputJSON(output)
		}
		fmt.Println("No state found. Run 'bazelle update' to create initial state.")
		return nil
	}

	// Get status
	cs, err := tracker.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect staleness: %w", err)
	}

	// Output result
	if statusFlags.json {
		output := StatusOutput{
			Stale:         !cs.IsEmpty(),
			StaleDirs:     cs.AffectedDirs(),
			NewFiles:      cs.Added,
			ModifiedFiles: cs.Modified,
			DeletedFiles:  cs.Deleted,
		}
		return outputJSON(output)
	}

	// Text output
	if cs.IsEmpty() {
		fmt.Println("BUILD files are up to date")
		return nil
	}

	staleDirs := cs.AffectedDirs()
	fmt.Printf("Stale directories (%d):\n", len(staleDirs))
	for _, dir := range staleDirs {
		fmt.Printf("  %s\n", dir)
	}

	if statusFlags.verbose {
		if len(cs.Added) > 0 {
			fmt.Printf("\nNew files (%d):\n", len(cs.Added))
			for _, f := range cs.Added {
				fmt.Printf("  + %s\n", f)
			}
		}

		if len(cs.Modified) > 0 {
			fmt.Printf("\nModified files (%d):\n", len(cs.Modified))
			for _, f := range cs.Modified {
				fmt.Printf("  ~ %s\n", f)
			}
		}

		if len(cs.Deleted) > 0 {
			fmt.Printf("\nDeleted files (%d):\n", len(cs.Deleted))
			for _, f := range cs.Deleted {
				fmt.Printf("  - %s\n", f)
			}
		}
	}

	fmt.Println("\nRun 'bazelle update --incremental' to update stale directories")
	return nil
}

func outputJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
