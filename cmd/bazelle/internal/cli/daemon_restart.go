package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonRestartFlags struct {
	force  bool
	socket string
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Long: `Restart the bazelle daemon.

This is equivalent to running 'bazelle daemon stop' followed by
'bazelle daemon start'.

Examples:
  bazelle daemon restart         # Restart the daemon
  bazelle daemon restart --force # Force restart if graceful stop fails`,
	RunE: runDaemonRestart,
}

func init() {
	daemonRestartCmd.Flags().BoolVar(&daemonRestartFlags.force, "force", false,
		"Force kill if graceful shutdown fails")
	daemonRestartCmd.Flags().StringVar(&daemonRestartFlags.socket, "socket", "",
		"Custom socket path")

	daemonCmd.AddCommand(daemonRestartCmd)
}

func runDaemonRestart(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := getRestartDaemonPaths()
	if err != nil {
		return err
	}

	// Check if daemon is running
	status := daemon.GetStatus(paths)

	if status.Running {
		fmt.Printf("Stopping daemon (PID: %d)...\n", status.PID)

		// Try graceful shutdown
		if err := tryGracefulShutdown(paths); err == nil {
			if waitForExit(status.PID, 5*time.Second) {
				fmt.Println("Daemon stopped")
			} else if daemonRestartFlags.force {
				fmt.Println("Graceful shutdown timed out, forcing...")
				_ = daemon.KillProcess(status.PID)
				waitForExit(status.PID, 2*time.Second)
			} else {
				return fmt.Errorf("graceful shutdown timed out (use --force to kill)")
			}
		} else if daemonRestartFlags.force {
			// Try SIGTERM
			_ = daemon.StopProcess(status.PID)
			if !waitForExit(status.PID, 2*time.Second) {
				_ = daemon.KillProcess(status.PID)
				waitForExit(status.PID, 2*time.Second)
			}
		}
	} else if status.Stale {
		fmt.Println("Cleaning up stale files...")
		_ = paths.Cleanup()
	}

	// Small delay before starting
	time.Sleep(200 * time.Millisecond)

	// Start daemon
	fmt.Println("Starting daemon...")

	// Set flags for start command
	daemonStartFlags.socket = daemonRestartFlags.socket
	daemonStartFlags.foreground = false

	return runDaemonStart(cmd, args)
}

// getRestartDaemonPaths returns the daemon paths based on flags or defaults.
func getRestartDaemonPaths() (*daemon.Paths, error) {
	if daemonRestartFlags.socket != "" {
		socketDir := filepath.Dir(daemonRestartFlags.socket)
		return &daemon.Paths{
			Dir:    socketDir,
			Socket: daemonRestartFlags.socket,
			PID:    daemonRestartFlags.socket + ".pid",
			Log:    daemonRestartFlags.socket + ".log",
		}, nil
	}

	return daemon.DefaultPaths()
}
