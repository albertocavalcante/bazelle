// Package cli implements the bazelle command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/spf13/cobra"
)

// Version information (set via ldflags)
var (
	Version   = "dev"
	GitCommit = "unknown"
)

// languages holds the list of language extensions to use with gazelle
var languages []language.Language

// SetLanguages sets the language extensions to use with gazelle
func SetLanguages(langs []language.Language) {
	languages = langs
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bazelle",
	Short: "Polyglot BUILD file generator",
	Long: `Bazelle is a polyglot BUILD file generator that wraps Gazelle
with support for multiple languages: Go, Kotlin, Python, Proto, and C++.

Use 'bazelle gazelle' for direct access to the underlying gazelle binary.`,
	// Default behavior: show help
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bazelle %s (%s)\n", Version, GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// GazelleDefaults are opinionated defaults prepended to gazelle args.
//
// Why these defaults?
// -------------------
// Gazelle historically used "go_default_library" as the naming convention for Go targets,
// which results in verbose deps like "@gazelle//language:go_default_library". The modern
// "import" convention uses cleaner names like "@gazelle//language" (target name matches
// the last segment of the import path).
//
// For external dependencies, gazelle defaults to "go_default_library" for backwards
// compatibility with older repos. We override this to use "import" consistently.
//
// These defaults are applied to all gazelle invocations through bazelle (update, fix).
// Users can still override per-directory via BUILD file directives:
//   # gazelle:go_naming_convention go_default_library
//
// TODO(albertocavalcante): Make these defaults more declarative:
//   - Allow configuration via .bazelle.yaml or similar config file
//   - Support opt-out flags like --no-defaults or --legacy-naming
//   - Consider environment variables for CI/CD customization
//   - Document the defaults prominently in --help output
//
var GazelleDefaults = []string{
	"-go_naming_convention=import",          // Modern naming (not go_default_library)
	"-go_naming_convention_external=import", // Same for external deps
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// RootCmd returns the root command for testing.
func RootCmd() *cobra.Command {
	return rootCmd
}
