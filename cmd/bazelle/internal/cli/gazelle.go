package cli

import (
	"github.com/spf13/cobra"
	"github.com/bazelbuild/bazel-gazelle/runner"
)

var gazelleCmd = &cobra.Command{
	Use:                "gazelle [args...]",
	Short:              "Run gazelle directly",
	Long:               `Passes all arguments directly to gazelle with language extensions.`,
	DisableFlagParsing: true, // Pass all flags to gazelle
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := runner.GetDefaultWorkspaceDirectory()
		if err != nil {
			return err
		}
		return runner.Run(languages, wd, args...)
	},
}

func init() {
	rootCmd.AddCommand(gazelleCmd)
}
