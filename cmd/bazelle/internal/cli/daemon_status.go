package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonStatusFlags struct {
	jsonOutput bool
	socket     string
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long: `Show the status of the bazelle daemon.

Displays whether the daemon is running, its PID, socket path,
uptime, and watch status.

Examples:
  bazelle daemon status        # Show status as text
  bazelle daemon status --json # Show status as JSON`,
	RunE: runDaemonStatus,
}

func init() {
	daemonStatusCmd.Flags().BoolVar(&daemonStatusFlags.jsonOutput, "json", false,
		"Output as JSON")
	daemonStatusCmd.Flags().StringVar(&daemonStatusFlags.socket, "socket", "",
		"Custom socket path")

	daemonCmd.AddCommand(daemonStatusCmd)
}

// DaemonStatusOutput is the JSON output format for daemon status.
type DaemonStatusOutput struct {
	Running         bool     `json:"running"`
	PID             int      `json:"pid,omitempty"`
	SocketPath      string   `json:"socket_path"`
	Version         string   `json:"version,omitempty"`
	Uptime          string   `json:"uptime,omitempty"`
	StartTime       string   `json:"start_time,omitempty"`
	Watching        bool     `json:"watching"`
	WatchPaths      []string `json:"watch_paths,omitempty"`
	WatchLanguages  []string `json:"watch_languages,omitempty"`
	ConnectedClients int     `json:"connected_clients,omitempty"`
	Error           string   `json:"error,omitempty"`
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	// Get paths
	paths, err := getStatusDaemonPaths()
	if err != nil {
		return err
	}

	// Check basic status from PID file
	status := daemon.GetStatus(paths)

	output := DaemonStatusOutput{
		Running:    status.Running,
		PID:        status.PID,
		SocketPath: paths.Socket,
	}

	// If running, get detailed info from daemon
	if status.Running {
		if err := enrichStatusFromDaemon(paths, &output); err != nil {
			output.Error = err.Error()
		}
	} else if status.Stale {
		output.Error = "stale PID file (daemon crashed)"
	}

	if daemonStatusFlags.jsonOutput {
		return outputDaemonStatusJSON(output)
	}

	return outputDaemonStatusText(output, status)
}

// enrichStatusFromDaemon connects to the daemon to get detailed status.
func enrichStatusFromDaemon(paths *daemon.Paths, output *DaemonStatusOutput) error {
	client, err := daemon.Connect(paths.Socket)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Get ping info (version, uptime)
	ping, err := client.Ping()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	output.Version = ping.Version
	output.Uptime = ping.Uptime
	output.StartTime = ping.StartTime

	// Get watch status
	watchStatus, err := client.WatchStatus()
	if err != nil {
		return fmt.Errorf("watch status failed: %w", err)
	}
	output.Watching = watchStatus.Watching
	output.WatchPaths = watchStatus.Paths
	output.WatchLanguages = watchStatus.Languages

	return nil
}

// outputDaemonStatusJSON outputs status as JSON.
func outputDaemonStatusJSON(output DaemonStatusOutput) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputDaemonStatusText outputs status as human-readable text.
func outputDaemonStatusText(output DaemonStatusOutput, status *daemon.DaemonStatus) error {
	if !output.Running {
		fmt.Println("Daemon: not running")
		if status.Stale {
			fmt.Printf("  (stale PID file found for PID %d)\n", status.PID)
			fmt.Println("  Run 'bazelle daemon start' to start the daemon")
		}
		return nil
	}

	fmt.Printf("Daemon: running (PID: %d)\n", output.PID)
	fmt.Printf("Socket: %s\n", output.SocketPath)

	if output.Version != "" {
		fmt.Printf("Version: %s\n", output.Version)
	}

	if output.Uptime != "" {
		fmt.Printf("Uptime: %s\n", formatUptime(output.Uptime))
	}

	if output.Watching {
		fmt.Println("Watching: yes")
		if len(output.WatchPaths) > 0 {
			fmt.Println("  Paths:")
			for _, p := range output.WatchPaths {
				fmt.Printf("    - %s\n", p)
			}
		}
		if len(output.WatchLanguages) > 0 {
			fmt.Println("  Languages:")
			for _, l := range output.WatchLanguages {
				fmt.Printf("    - %s\n", l)
			}
		}
	} else {
		fmt.Println("Watching: no")
	}

	if output.Error != "" {
		fmt.Printf("Warning: %s\n", output.Error)
	}

	return nil
}

// formatUptime formats the uptime string for display.
func formatUptime(uptime string) string {
	// Try to parse and format nicely
	d, err := time.ParseDuration(uptime)
	if err != nil {
		return uptime
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// getStatusDaemonPaths returns the daemon paths based on flags or defaults.
func getStatusDaemonPaths() (*daemon.Paths, error) {
	if daemonStatusFlags.socket != "" {
		socketDir := filepath.Dir(daemonStatusFlags.socket)
		return &daemon.Paths{
			Dir:    socketDir,
			Socket: daemonStatusFlags.socket,
			PID:    daemonStatusFlags.socket + ".pid",
			Log:    daemonStatusFlags.socket + ".log",
		}, nil
	}

	return daemon.DefaultPaths()
}
