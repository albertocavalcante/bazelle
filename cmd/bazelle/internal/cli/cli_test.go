package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestNoFlagConflicts verifies that all subcommands can be initialized
// without flag shorthand conflicts. This catches issues like multiple
// commands defining the same shorthand (e.g., -v for both --verbosity
// and --verbose).
func TestNoFlagConflicts(t *testing.T) {
	// Get the root command which has all subcommands registered
	root := RootCmd()

	// Verify root command exists
	if root == nil {
		t.Fatal("RootCmd() returned nil")
	}

	// Get all subcommands
	subcommands := root.Commands()
	if len(subcommands) == 0 {
		t.Fatal("expected at least one subcommand")
	}

	// Test that we can parse flags for each subcommand without panic
	// This exercises the flag merging that happens when persistent flags
	// are combined with local flags
	for _, cmd := range subcommands {
		t.Run(cmd.Name(), func(t *testing.T) {
			// This will panic if there are flag conflicts
			// We catch the panic and fail the test with a descriptive message
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("flag conflict in %q command: %v", cmd.Name(), r)
				}
			}()

			// Trigger flag parsing by getting all flags
			// This merges persistent flags from parent with local flags
			_ = cmd.Flags()

			// Also check inherited flags which forces the merge
			_ = cmd.InheritedFlags()
		})
	}
}

// TestGlobalVerbosityFlag verifies the global -v flag exists and is properly configured.
func TestGlobalVerbosityFlag(t *testing.T) {
	root := RootCmd()

	// Check that -v is defined as a persistent flag on root
	vFlag := root.PersistentFlags().Lookup("verbosity")
	if vFlag == nil {
		t.Fatal("expected persistent 'verbosity' flag on root command")
	}

	if vFlag.Shorthand != "v" {
		t.Errorf("expected verbosity flag shorthand to be 'v', got %q", vFlag.Shorthand)
	}
}

// TestSubcommandsExist verifies expected subcommands are registered.
func TestSubcommandsExist(t *testing.T) {
	root := RootCmd()

	expectedCmds := []string{"version", "update", "fix", "watch", "status"}

	for _, name := range expectedCmds {
		found := false
		for _, cmd := range root.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}

// TestVerboseFlagNoShorthand verifies that subcommand --verbose flags
// don't have a -v shorthand (which would conflict with root's -v).
func TestVerboseFlagNoShorthand(t *testing.T) {
	root := RootCmd()

	// Commands that have a --verbose flag
	cmdsWithVerbose := []string{"update", "fix", "watch", "status"}

	for _, cmdName := range cmdsWithVerbose {
		t.Run(cmdName, func(t *testing.T) {
			var cmd *cobra.Command
			for _, c := range root.Commands() {
				if c.Name() == cmdName {
					cmd = c
					break
				}
			}

			if cmd == nil {
				t.Skipf("command %q not found", cmdName)
				return
			}

			verboseFlag := cmd.Flags().Lookup("verbose")
			if verboseFlag == nil {
				t.Skipf("command %q has no verbose flag", cmdName)
				return
			}

			if verboseFlag.Shorthand != "" {
				t.Errorf("command %q verbose flag should not have shorthand, got %q",
					cmdName, verboseFlag.Shorthand)
			}
		})
	}
}
