package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// ErrNotConnected is returned when trying to use a disconnected client.
var ErrNotConnected = errors.New("not connected to daemon")

// ErrDaemonNotRunning is returned when the daemon is not running.
var ErrDaemonNotRunning = errors.New("daemon not running")

// Client is a client for connecting to the daemon.
type Client struct {
	conn      net.Conn
	encoder   *json.Encoder
	decoder   *json.Decoder
	encoderMu sync.Mutex
	decoderMu sync.Mutex
	idGen     IDGenerator

	// Event handling
	eventCh   chan *Notification
	eventOnce sync.Once
	closeCh   chan struct{}
}

// Connect connects to the daemon at the given socket path.
func Connect(socketPath string) (*Client, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		if errors.Is(err, net.ErrClosed) || isConnectionRefused(err) {
			return nil, ErrDaemonNotRunning
		}
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}

	return &Client{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(bufio.NewReader(conn)),
		closeCh: make(chan struct{}),
	}, nil
}

// ConnectDefault connects to the daemon at the default socket path.
func ConnectDefault() (*Client, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}
	return Connect(paths.Socket)
}

// isConnectionRefused checks if the error is a connection refused error.
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	// Check for both ECONNREFUSED and ENOENT (socket file doesn't exist)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	return false
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	c.eventOnce.Do(func() {
		close(c.closeCh)
	})
	return c.conn.Close()
}

// call sends a request and waits for a response.
func (c *Client) call(method string, params any, result any) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	id := c.idGen.Next()
	req, err := NewRequest(id, method, params)
	if err != nil {
		return err
	}

	// Send request
	c.encoderMu.Lock()
	err = c.encoder.Encode(req)
	c.encoderMu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	c.decoderMu.Lock()
	var resp Response
	err = c.decoder.Decode(&resp)
	c.decoderMu.Unlock()
	if err != nil {
		if err == io.EOF {
			return ErrNotConnected
		}
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error
	if resp.Error != nil {
		return resp.Error
	}

	// Unmarshal result
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// Ping sends a ping request to the daemon.
func (c *Client) Ping() (*PingResult, error) {
	var result PingResult
	if err := c.call(MethodPing, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Shutdown sends a shutdown request to the daemon.
func (c *Client) Shutdown() (*ShutdownResult, error) {
	var result ShutdownResult
	if err := c.call(MethodShutdown, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WatchStart starts watching the given paths.
func (c *Client) WatchStart(params *WatchStartParams) (*WatchStartResult, error) {
	var result WatchStartResult
	if err := c.call(MethodWatchStart, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WatchStop stops watching.
func (c *Client) WatchStop() (*WatchStopResult, error) {
	var result WatchStopResult
	if err := c.call(MethodWatchStop, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// WatchStatus returns the current watch status.
func (c *Client) WatchStatus() (*WatchStatusResult, error) {
	var result WatchStatusResult
	if err := c.call(MethodWatchStatus, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateRun triggers a manual update.
func (c *Client) UpdateRun(params *UpdateRunParams) (*UpdateRunResult, error) {
	var result UpdateRunResult
	if err := c.call(MethodUpdateRun, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StatusGet returns the staleness status.
func (c *Client) StatusGet() (*StatusGetResult, error) {
	var result StatusGetResult
	if err := c.call(MethodStatusGet, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubscribeEvents starts receiving event notifications.
// Returns a channel that receives notifications.
// The channel is closed when the connection is closed.
func (c *Client) SubscribeEvents() (<-chan *Notification, error) {
	// Create event channel if not already created
	c.eventOnce.Do(func() {
		c.eventCh = make(chan *Notification, 100)
		go c.readEvents()
	})

	// Note: In a real implementation, we would send a subscribe request.
	// For simplicity, the server sends events to all clients.

	return c.eventCh, nil
}

// readEvents reads notifications from the server.
func (c *Client) readEvents() {
	defer close(c.eventCh)

	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		c.decoderMu.Lock()
		var notif Notification
		err := c.decoder.Decode(&notif)
		c.decoderMu.Unlock()

		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}

		// Only process notifications (no ID)
		if notif.Method != "" {
			select {
			case c.eventCh <- &notif:
			default:
				// Drop if channel full
			}
		}
	}
}

// IsDaemonRunning checks if the daemon is running using the default paths.
func IsDaemonRunning() bool {
	paths, err := DefaultPaths()
	if err != nil {
		return false
	}
	return IsDaemonRunningAt(paths)
}

// IsDaemonRunningAt checks if the daemon is running at the given paths.
func IsDaemonRunningAt(paths *Paths) bool {
	status := GetStatus(paths)
	return status.Running
}

// GetDaemonInfo retrieves information about the running daemon.
func GetDaemonInfo() (*DaemonInfo, error) {
	client, err := ConnectDefault()
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	ping, err := client.Ping()
	if err != nil {
		return nil, err
	}

	status, err := client.WatchStatus()
	if err != nil {
		return nil, err
	}

	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}
	pidStatus := GetStatus(paths)

	return &DaemonInfo{
		PID:        pidStatus.PID,
		SocketPath: paths.Socket,
		Version:    ping.Version,
		Watching:   status.Watching,
		WatchPaths: status.Paths,
	}, nil
}
