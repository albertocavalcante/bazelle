package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonStopFlags struct {
	force  bool
	socket string
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Long: `Stop the bazelle daemon process.

By default, sends a graceful shutdown request via the socket.
If the daemon doesn't respond within 5 seconds, use --force to
send SIGKILL.

Examples:
  bazelle daemon stop         # Graceful shutdown
  bazelle daemon stop --force # Force kill if graceful fails`,
	RunE: runDaemonStop,
}

func init() {
	daemonStopCmd.Flags().BoolVar(&daemonStopFlags.force, "force", false,
		"Force kill if graceful shutdown fails")
	daemonStopCmd.Flags().StringVar(&daemonStopFlags.socket, "socket", "",
		"Custom socket path")

	daemonCmd.AddCommand(daemonStopCmd)
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := getStopDaemonPaths()
	if err != nil {
		return err
	}

	// Check if daemon is running
	status := daemon.GetStatus(paths)

	if status.Stale {
		// Clean up stale files
		fmt.Println("Daemon not running (cleaning up stale files)")
		_ = paths.Cleanup()
		return nil
	}

	if !status.Running {
		fmt.Println("Daemon not running")
		return nil
	}

	fmt.Printf("Stopping daemon (PID: %d)...\n", status.PID)

	// Try graceful shutdown via RPC
	if err := tryGracefulShutdown(paths); err == nil {
		// Wait for process to exit
		if waitForExit(status.PID, 5*time.Second) {
			fmt.Println("Daemon stopped")
			return nil
		}
	}

	// Graceful shutdown failed or timed out
	if !daemonStopFlags.force {
		fmt.Println("Graceful shutdown timed out. Use --force to kill.")
		return fmt.Errorf("shutdown timed out")
	}

	// Force kill
	fmt.Println("Forcing shutdown...")
	if err := daemon.KillProcess(status.PID); err != nil {
		// Process might have exited between checks
		if !daemon.IsProcessRunning(status.PID) {
			fmt.Println("Daemon stopped")
			_ = paths.Cleanup()
			return nil
		}
		return fmt.Errorf("failed to kill daemon: %w", err)
	}

	// Wait for process to exit after SIGKILL
	if waitForExit(status.PID, 2*time.Second) {
		fmt.Println("Daemon stopped (forced)")
		_ = paths.Cleanup()
		return nil
	}

	return fmt.Errorf("failed to stop daemon")
}

// tryGracefulShutdown attempts to stop the daemon via RPC.
func tryGracefulShutdown(paths *daemon.Paths) error {
	client, err := daemon.Connect(paths.Socket)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	_, err = client.Shutdown()
	return err
}

// waitForExit waits for a process to exit.
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !daemon.IsProcessRunning(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// getStopDaemonPaths returns the daemon paths based on flags or defaults.
func getStopDaemonPaths() (*daemon.Paths, error) {
	if daemonStopFlags.socket != "" {
		socketDir := filepath.Dir(daemonStopFlags.socket)
		return &daemon.Paths{
			Dir:    socketDir,
			Socket: daemonStopFlags.socket,
			PID:    daemonStopFlags.socket + ".pid",
			Log:    daemonStopFlags.socket + ".log",
		}, nil
	}

	return daemon.DefaultPaths()
}
