package cli

import (
	"github.com/spf13/cobra"
)

// daemonCmd is the parent command for daemon operations.
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the bazelle daemon",
	Long: `Manage the bazelle background daemon process.

The daemon runs in the background and provides file watching and BUILD
file generation services. Multiple clients (CLI, VS Code, etc.) can
connect to a single daemon instance.

Commands:
  start   - Start the daemon process
  stop    - Stop the running daemon
  status  - Show daemon status
  restart - Restart the daemon

Examples:
  bazelle daemon start              # Start daemon in background
  bazelle daemon start --foreground # Run daemon in foreground (for debugging)
  bazelle daemon status             # Check if daemon is running
  bazelle daemon stop               # Stop the daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
