package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nathfavour/beaverish/pkg/logger"
)

type IPCMessage struct {
	Type      string          `json:"type"`      // "log", "event", "command", "response"
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type SingleInstance struct {
	socketPath string
	listener   net.Listener
	clients    map[net.Conn]bool
	mu         sync.Mutex
	isServer   bool
}

func SocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	return filepath.Join(runtimeDir, "beaverish.sock")
}

// AcquireOrConnect tries to become the primary server daemon.
// If another instance already owns the socket, it returns (instance, false, nil),
// indicating this process should run as a follower/client streaming from or querying the primary.
func AcquireOrConnect(sockPath string) (*SingleInstance, bool, error) {
	if sockPath == "" {
		sockPath = SocketPath()
	}

	// Try dialing existing socket
	conn, err := net.DialTimeout("unix", sockPath, 200*time.Millisecond)
	if err == nil {
		// Server already running!
		_ = conn.Close()
		return &SingleInstance{
			socketPath: sockPath,
			isServer:   false,
		}, false, nil
	}

	// Socket file might be stale if server crashed
	_ = os.Remove(sockPath)

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to listen on socket %s: %w", sockPath, err)
	}

	si := &SingleInstance{
		socketPath: sockPath,
		listener:   l,
		clients:    make(map[net.Conn]bool),
		isServer:   true,
	}

	return si, true, nil
}

func (si *SingleInstance) IsServer() bool {
	return si.isServer
}

func (si *SingleInstance) StartBroadcaster(ctx context.Context) {
	if !si.isServer || si.listener == nil {
		return
	}

	go func() {
		for {
			conn, err := si.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					return
				}
			}

			si.mu.Lock()
			si.clients[conn] = true
			si.mu.Unlock()

			go func(c net.Conn) {
				defer func() {
					si.mu.Lock()
					delete(si.clients, c)
					si.mu.Unlock()
					_ = c.Close()
				}()

				// Keep alive reading or waiting
				buf := make([]byte, 1024)
				for {
					_, err := c.Read(buf)
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()
}

func (si *SingleInstance) Broadcast(msgType string, data interface{}) {
	if !si.isServer {
		return
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := IPCMessage{
		Type:      msgType,
		Payload:   raw,
		Timestamp: time.Now(),
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return
	}
	bytes = append(bytes, '\n')

	si.mu.Lock()
	defer si.mu.Unlock()

	for c := range si.clients {
		_ = c.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		if _, err := c.Write(bytes); err != nil {
			_ = c.Close()
			delete(si.clients, c)
		}
	}
}

func (si *SingleInstance) Close() {
	if si.isServer && si.listener != nil {
		_ = si.listener.Close()
		_ = os.Remove(si.socketPath)
	}
}

// AttachFollower connects to the primary daemon and pipes events/logs live to stderr/stdout
func AttachFollower(ctx context.Context, sockPath string) error {
	if sockPath == "" {
		sockPath = SocketPath()
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return fmt.Errorf("failed to attach to primary beaverish daemon: %w", err)
	}
	defer conn.Close()

	logger.Infof("Attached to primary Beaverish daemon via %s. Streaming live telemetry...", sockPath)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()
		var msg IPCMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			var strPayload string
			if err := json.Unmarshal(msg.Payload, &strPayload); err == nil {
				fmt.Fprintln(os.Stderr, strPayload)
			} else {
				fmt.Fprintln(os.Stderr, string(msg.Payload))
			}
		} else {
			fmt.Fprintln(os.Stderr, line)
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}
