// Package cli implements the bazelle command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/bazelbuild/bazel-gazelle/language"
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
