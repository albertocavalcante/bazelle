package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/incremental"
	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/bazelbuild/bazel-gazelle/runner"
	"github.com/spf13/cobra"
)

var updateFlags struct {
	check       bool
	languages   []string
	verbose     bool
	incremental bool
	force       bool
}

var updateCmd = &cobra.Command{
	Use:   "update [path...]",
	Short: "Update BUILD files",
	Long: `Updates BUILD files by running gazelle update.

The --check flag can be used in CI to verify BUILD files are up to date
without making changes.

The --incremental flag enables incremental mode, which only updates
directories that have changed since the last update. This can be
significantly faster for large codebases.

The --force flag forces a full update, ignoring any cached state.

Additional gazelle flags (like -bzlmod, -go_prefix) are passed through.`,
	RunE:                  runUpdate,
	FParseErrWhitelist:    cobra.FParseErrWhitelist{UnknownFlags: true},
	DisableFlagsInUseLine: true,
}

func init() {
	updateCmd.Flags().BoolVar(&updateFlags.check, "check", false,
		"Check if BUILD files are up to date (exit 1 if changes needed)")
	updateCmd.Flags().StringSliceVar(&updateFlags.languages, "languages", nil,
		"Only run specific language extensions (comma-separated)")
	updateCmd.Flags().BoolVarP(&updateFlags.verbose, "verbose", "v", false,
		"Show detailed output")
	updateCmd.Flags().BoolVar(&updateFlags.incremental, "incremental", false,
		"Only update directories with changed source files")
	updateCmd.Flags().BoolVar(&updateFlags.force, "force", false,
		"Force full update, ignoring cached state")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	start := time.Now()
	wd, err := runner.GetDefaultWorkspaceDirectory()
	if err != nil {
		return err
	}

	log.V(2).Infow("starting update",
		"dir", wd,
		"incremental", updateFlags.incremental,
		"check", updateFlags.check)

	// Build gazelle arguments: "update" + defaults + mode + passthrough args
	// Note: gazelle expects command first, then flags
	gazelleArgs := []string{"update"}
	gazelleArgs = append(gazelleArgs, GazelleDefaults...)

	if updateFlags.check {
		gazelleArgs = append(gazelleArgs, "-mode=diff")
	}

	// Add passthrough args (unknown flags + paths)
	if len(args) > 0 {
		gazelleArgs = append(gazelleArgs, args...)
	}

	if updateFlags.check {
		return runUpdateCheck(wd, gazelleArgs)
	}

	// Handle incremental mode
	if updateFlags.incremental && !updateFlags.force {
		return runIncrementalUpdate(wd, args)
	}

	// Normal update: run gazelle
	if err := runner.Run(languages, wd, gazelleArgs...); err != nil {
		return err
	}

	// Update state after successful run
	if err := updateStateAfterRun(wd); err != nil {
		return err
	}

	log.V(2).Infow("update complete", "duration", time.Since(start))
	return nil
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

func runIncrementalUpdate(wd string, passthroughArgs []string) error {
	ctx := context.Background()
	tracker := incremental.NewTracker(wd, updateFlags.languages)

	// Check if state exists
	if !tracker.HasState() {
		if updateFlags.verbose {
			fmt.Println("No state found, running full update...")
		}
		return runFullUpdate(wd, passthroughArgs)
	}

	// Get status
	cs, err := tracker.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect staleness: %w", err)
	}

	// Check if there are stale directories
	if cs.IsEmpty() {
		fmt.Println("BUILD files are up to date")
		return nil
	}

	staleDirs := cs.AffectedDirs()

	// Print stale directories
	if updateFlags.verbose {
		fmt.Printf("Found %d stale directories:\n", len(staleDirs))
		for _, dir := range staleDirs {
			fmt.Printf("  %s\n", dir)
		}
		fmt.Println()
	}

	// Build gazelle arguments with stale directories as targets
	gazelleArgs := []string{"update"}
	gazelleArgs = append(gazelleArgs, GazelleDefaults...)
	gazelleArgs = append(gazelleArgs, passthroughArgs...)

	// Add stale directories as targets
	targets := cs.AsTargets()
	gazelleArgs = append(gazelleArgs, targets...)

	// Run gazelle on stale directories
	fmt.Printf("Updating %d directories...\n", len(staleDirs))
	if err := runner.Run(languages, wd, gazelleArgs...); err != nil {
		return fmt.Errorf("gazelle failed: %w", err)
	}

	// Update state after successful run
	return updateStateAfterRun(wd)
}

func runFullUpdate(wd string, passthroughArgs []string) error {
	// Build gazelle arguments
	gazelleArgs := []string{"update"}
	gazelleArgs = append(gazelleArgs, GazelleDefaults...)
	gazelleArgs = append(gazelleArgs, passthroughArgs...)

	// Run gazelle
	if err := runner.Run(languages, wd, gazelleArgs...); err != nil {
		return fmt.Errorf("gazelle failed: %w", err)
	}

	// Update state after successful run
	return updateStateAfterRun(wd)
}

func updateStateAfterRun(wd string) error {
	ctx := context.Background()
	tracker := incremental.NewTracker(wd, updateFlags.languages)

	// Refresh state from current disk state
	if err := tracker.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	if updateFlags.verbose {
		fmt.Printf("State saved (%d files tracked)\n", tracker.TrackedFileCount())
	}

	return nil
}
