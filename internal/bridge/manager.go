package bridge

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bridgepb "github.com/adam-stokes/orcai/proto/bridgepb"
)

// Manager spawns and manages provider adapter subprocesses.
type Manager struct {
	cwd       string
	socketDir string
	adapters  []*adapterEntry
	cancel    context.CancelFunc
}

type adapterEntry struct {
	name     string
	proc     *exec.Cmd
	conn     *grpc.ClientConn
	client   bridgepb.ProviderBridgeClient
	descResp *bridgepb.DescribeResponse
}

// New creates a manager for the given working directory.
func New(cwd string) *Manager {
	return &Manager{cwd: cwd}
}

// Start spawns all provider adapters and waits for them to be ready.
// The adapters run under an independent background context so that
// cancelling the caller's ctx (e.g. a startup timeout) does not kill them.
// Call Stop to shut them down gracefully.
func (m *Manager) Start(_ context.Context) error {
	dir, err := os.MkdirTemp("", "stok-bridge-*")
	if err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	m.socketDir = dir

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	mgrCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	// adapterDef maps the adapter name to the CLI binary it requires.
	type adapterDef struct{ name, bin string }
	adapterDefs := []adapterDef{
		{"claude", "claude"},
		{"gemini", "gemini"},
		{"copilot", "copilot"},
	}

	for _, def := range adapterDefs {
		if _, err := exec.LookPath(def.bin); err != nil {
			continue // CLI not installed; skip this adapter
		}
		name := def.name
		sockPath := filepath.Join(dir, name+".sock")
		proc := exec.CommandContext(mgrCtx, self, "bridge", name,
			"--socket", sockPath,
			"--cwd", m.cwd,
		)
		proc.Stderr = os.Stderr
		if err := proc.Start(); err != nil {
			cancel()
			return fmt.Errorf("start %s adapter: %w", name, err)
		}

		conn, err := dialWithRetry(sockPath, 2*time.Second)
		if err != nil {
			proc.Process.Kill()
			cancel()
			return fmt.Errorf("dial %s adapter: %w", name, err)
		}

		client := bridgepb.NewProviderBridgeClient(conn)
		desc, err := client.Describe(mgrCtx, &bridgepb.DescribeRequest{})
		if err != nil {
			conn.Close()
			proc.Process.Kill()
			cancel()
			return fmt.Errorf("describe %s adapter: %w", name, err)
		}

		m.adapters = append(m.adapters, &adapterEntry{
			name:     name,
			proc:     proc,
			conn:     conn,
			client:   client,
			descResp: desc,
		})
	}

	return nil
}

// Stop gracefully shuts down all adapters.
func (m *Manager) Stop(ctx context.Context) {
	for _, a := range m.adapters {
		// Best-effort graceful shutdown.
		ctx2, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		a.client.Shutdown(ctx2, &bridgepb.ShutdownRequest{})
		cancel()
		a.conn.Close()
		a.proc.Process.Kill()
	}
	if m.cancel != nil {
		m.cancel()
	}
	if m.socketDir != "" {
		os.RemoveAll(m.socketDir)
	}
}

// Client returns the gRPC client for the named adapter (e.g. "claude").
// Returns nil if the adapter is not running.
func (m *Manager) Client(name string) bridgepb.ProviderBridgeClient {
	for _, a := range m.adapters {
		if a.name == name {
			return a.client
		}
	}
	return nil
}

// Capabilities returns all capabilities from all running adapters.
func (m *Manager) Capabilities() []*bridgepb.Capability {
	var all []*bridgepb.Capability
	for _, a := range m.adapters {
		all = append(all, a.descResp.Capabilities...)
	}
	return all
}

// dialWithRetry attempts to dial a Unix socket, retrying until timeout.
func dialWithRetry(socketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			// Verify socket is accepting connections before creating gRPC client.
			c, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
			if err == nil {
				c.Close()
				conn, err := grpc.NewClient("unix://"+socketPath,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				)
				if err == nil {
					return conn, nil
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for socket %s", socketPath)
}

