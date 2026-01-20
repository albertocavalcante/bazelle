package cli

import (
	"github.com/spf13/cobra"
	"github.com/bazelbuild/bazel-gazelle/runner"
)

var gazelleCmd = &cobra.Command{
	Use:   "gazelle [args...]",
	Short: "Run gazelle directly (raw passthrough)",
	Long: `Passes all arguments directly to gazelle with language extensions.

This is a raw passthrough - no opinionated defaults are added.
Use 'bazelle update' or 'bazelle fix' for enhanced commands with defaults.`,
	DisableFlagParsing: true, // Pass all flags to gazelle
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := runner.GetDefaultWorkspaceDirectory()
		if err != nil {
			return err
		}
		// Raw passthrough - no defaults (user has full control)
		return runner.Run(languages, wd, args...)
	},
}

func init() {
	rootCmd.AddCommand(gazelleCmd)
}
