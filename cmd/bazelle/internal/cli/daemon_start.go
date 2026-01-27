package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/daemon"
	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/spf13/cobra"
)

var daemonStartFlags struct {
	foreground bool
	socket     string
	logFile    string
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon process",
	Long: `Start the bazelle daemon process.

By default, the daemon runs in the background. Use --foreground to run
in the foreground for debugging.

The daemon listens on a Unix socket for client connections. Multiple
clients can connect simultaneously.

Examples:
  bazelle daemon start              # Start in background
  bazelle daemon start --foreground # Run in foreground (Ctrl+C to stop)
  bazelle daemon start --socket /custom/path.sock`,
	RunE: runDaemonStart,
}

func init() {
	daemonStartCmd.Flags().BoolVar(&daemonStartFlags.foreground, "foreground", false,
		"Run in foreground (don't daemonize)")
	daemonStartCmd.Flags().StringVar(&daemonStartFlags.socket, "socket", "",
		"Custom socket path (default: ~/.bazelle/daemon.sock)")
	daemonStartCmd.Flags().StringVar(&daemonStartFlags.logFile, "log", "",
		"Log file path (default: ~/.bazelle/daemon.log)")

	daemonCmd.AddCommand(daemonStartCmd)
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := getDaemonPaths()
	if err != nil {
		return err
	}

	// Check if daemon is already running
	status := daemon.GetStatus(paths)
	if status.Running {
		fmt.Printf("Daemon already running (PID: %d)\n", status.PID)
		return nil
	}

	// Clean up stale files if needed
	if status.Stale {
		if _, err := daemon.CleanupStale(paths); err != nil {
			log.Warn("failed to clean up stale files", "error", err)
		}
	}

	if daemonStartFlags.foreground {
		return runDaemonForeground(paths)
	}

	return runDaemonBackground(paths)
}

// runDaemonForeground runs the daemon in the foreground.
func runDaemonForeground(paths *daemon.Paths) error {
	fmt.Printf("Starting daemon in foreground (PID: %d)\n", os.Getpid())
	fmt.Printf("Socket: %s\n", paths.Socket)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create server
	handler := daemon.NewHandlerWithConfig(nil, daemon.HandlerConfig{
		Languages:       languages,
		GazelleDefaults: GazelleDefaults,
	})

	server := daemon.NewServer(daemon.ServerConfig{
		Paths:   paths,
		Version: Version,
		Handler: handler,
	})

	// Run server (blocks until shutdown)
	ctx := context.Background()
	return server.Start(ctx)
}

// runDaemonBackground starts the daemon in a background process.
func runDaemonBackground(paths *daemon.Paths) error {
	// Get the path to the current executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command args for foreground mode
	args := []string{"daemon", "start", "--foreground"}
	if daemonStartFlags.socket != "" {
		args = append(args, "--socket", daemonStartFlags.socket)
	}
	if daemonStartFlags.logFile != "" {
		args = append(args, "--log", daemonStartFlags.logFile)
	}

	// Ensure daemon directory exists
	if err := paths.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create daemon directory: %w", err)
	}

	// Open log file for daemon output
	logPath := paths.Log
	if daemonStartFlags.logFile != "" {
		logPath = daemonStartFlags.logFile
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create background process
	cmd := exec.Command(executable, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	// Detach from parent process
	cmd.SysProcAttr = daemonSysProcAttr()

	// Start the daemon
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Close log file in parent (child keeps its own handle)
	_ = logFile.Close()

	// Wait briefly for daemon to start and write PID file
	time.Sleep(500 * time.Millisecond)

	// Verify daemon started successfully
	status := daemon.GetStatus(paths)
	if !status.Running {
		// Try to read any error from log
		return fmt.Errorf("daemon failed to start (check %s for details)", logPath)
	}

	fmt.Printf("Daemon started (PID: %d)\n", status.PID)
	fmt.Printf("Socket: %s\n", paths.Socket)
	fmt.Printf("Log: %s\n", logPath)

	return nil
}

// getDaemonPaths returns the daemon paths based on flags or defaults.
func getDaemonPaths() (*daemon.Paths, error) {
	if daemonStartFlags.socket != "" {
		// Custom socket path - derive other paths from it
		socketDir := filepath.Dir(daemonStartFlags.socket)
		return &daemon.Paths{
			Dir:    socketDir,
			Socket: daemonStartFlags.socket,
			PID:    daemonStartFlags.socket + ".pid",
			Log:    daemonStartFlags.socket + ".log",
		}, nil
	}

	return daemon.DefaultPaths()
}
