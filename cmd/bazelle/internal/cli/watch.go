package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/watch"
	"github.com/bazelbuild/bazel-gazelle/runner"
	"github.com/spf13/cobra"
)

var watchFlags struct {
	debounce  int
	languages []string
	verbose   bool
	json      bool
	noColor   bool
}

var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Watch for source file changes and auto-update BUILD files",
	Long: `Watches the workspace for source file changes and automatically
updates BUILD files when changes are detected.

This provides a seamless development experience where BUILD files
stay in sync with your code without manual intervention.

Example output:

  $ bazelle watch

  bazelle: watching 1,247 files in /path/to/workspace
  bazelle: languages: go, kotlin, java
  bazelle: ready

  [14:32:15] ~ src/auth/login.kt
  [14:32:15] updating //src/auth:all...
  [14:32:16] ✓ src/auth/BUILD.bazel updated

Press Ctrl+C to stop watching.`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().IntVar(&watchFlags.debounce, "debounce", 500,
		"Debounce window in milliseconds")
	watchCmd.Flags().StringSliceVar(&watchFlags.languages, "languages", nil,
		"Only watch specific languages (comma-separated)")
	watchCmd.Flags().BoolVarP(&watchFlags.verbose, "verbose", "v", false,
		"Show file-level changes")
	watchCmd.Flags().BoolVar(&watchFlags.json, "json", false,
		"Stream JSON events (for tooling integration)")
	watchCmd.Flags().BoolVar(&watchFlags.noColor, "no-color", false,
		"Disable colored output")

	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	// Get workspace directory
	wd, err := runner.GetDefaultWorkspaceDirectory()
	if err != nil {
		return fmt.Errorf("failed to determine workspace directory: %w", err)
	}

	// If path argument provided, validate and use it
	if len(args) > 0 {
		wd = args[0]
		info, err := os.Stat(wd)
		if err != nil {
			return fmt.Errorf("invalid path %s: %w", wd, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path must be a directory: %s", wd)
		}
	}

	// Setup signal handling for graceful shutdown
	// Include SIGHUP to handle terminal hangup
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	// Create watcher
	w, err := watch.New(watch.Config{
		Root:            wd,
		Languages:       languages,
		LangFilter:      watchFlags.languages,
		Debounce:        watchFlags.debounce,
		Verbose:         watchFlags.verbose,
		NoColor:         watchFlags.noColor,
		JSON:            watchFlags.json,
		GazelleDefaults: GazelleDefaults,
	})
	if err != nil {
		return err
	}
	defer func() { _ = w.Close() }()

	// Run watch loop
	return w.Run(ctx)
}
