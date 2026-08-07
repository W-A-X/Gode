/*
extproc manages the Node.js extension host process lifecycle.

The extension host is the only part of VS Code that remains in TypeScript/Node.js.
Go launches it as a child process and connects via IPC (persistent protocol).

Key entry points in the TS build:
  - out/vs/workbench/api/node/extensionHostProcess.js (the bootstrap)
  - Set env VSCODE_EXTHOST_IPC_HOOK=<pipeName> so the host connects to Go
*/

package extproc

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/microsoft/gode/internal/ipc"
)

const (
	extHostScript = "vs/workbench/api/node/extensionHostProcess.js"
	readyTimeout  = 60 * time.Second
)

// Config configures the extension host process launch.
type Config struct {
	NodeBin          string // path to node binary
	BuildOutputDir   string // path to out/ directory (compiled TS)
	UserDataDir      string
	ExtensionsDir    string
	LogDir           string
	Commit           string
	Version          string
	AppName          string
	AppLanguage      string
}

// Manager manages the extension host process and IPC connection.
type Manager struct {
	cfg   Config
	mu    sync.Mutex
	cmd   *exec.Cmd
	conn  net.Conn
	done  chan struct{}
	ready chan struct{}
	err   error
}

// NewManager creates a new extension host manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:   cfg,
		done:  make(chan struct{}),
		ready: make(chan struct{}),
	}
}

// Start launches the extension host and establishes IPC.
// Returns the IPC connection once the handshake is complete.
func (m *Manager) Start(initData *ipc.ExtensionHostInitData) (<-chan struct{}, error) {
	// Create named pipe / socket for IPC
	pipeName, cleanup, err := m.createIPCServer()
	if err != nil {
		return nil, fmt.Errorf("create IPC server: %w", err)
	}

	// Set up environment for the extension host
	env := os.Environ()
	env = append(env, fmt.Sprintf("VSCODE_EXTHOST_IPC_HOOK=%s", pipeName))
	env = append(env, fmt.Sprintf("VSCODE_DEV=1"))
	env = append(env, fmt.Sprintf("VSCODE_IPC_HOOK=%s", pipeName))
	env = append(env, fmt.Sprintf("VSCODE_COMMIT=%s", m.cfg.Commit))
	env = append(env, fmt.Sprintf("PATH=%s:%s", m.cfg.BuildOutputDir, os.Getenv("PATH")))

	nodeScript := filepath.Join(m.cfg.BuildOutputDir, extHostScript)
	if _, err := os.Stat(nodeScript); err != nil {
		cleanup()
		return nil, fmt.Errorf("extension host script not found at %s\n"+
			"Hint: compile the TS extension host first:\n"+
			"  cd <repo-root> && npm run gulp compile  # or: npm run build-fast\n"+
			"Then re-run gode", nodeScript)
	}

	// Spawn Node extension host process
	cmd := exec.Command(m.cfg.NodeBin, nodeScript,
		"--type=extension",
		"--extensionDevelopmentPath="+m.cfg.ExtensionsDir,
		"--extensions-dir="+m.cfg.ExtensionsDir,
		"--user-data-dir="+m.cfg.UserDataDir,
	)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()

	log.Printf("starting extension host: %s %s", m.cfg.NodeBin, nodeScript)

	if err := cmd.Start(); err != nil {
		cleanup()
		return nil, fmt.Errorf("start extension host: %w", err)
	}

	// Wait for connection from extension host
	go m.waitForConnection(initData, cleanup)

	return m.ready, nil
}

// createIPCServer creates a named pipe (Windows) or Unix domain socket (Unix).
func (m *Manager) createIPCServer() (string, func(), error) {
	tmpDir := os.TempDir()
	pipePath := filepath.Join(tmpDir, fmt.Sprintf("gode-ext-%d.sock", time.Now().UnixNano()))

	ln, err := net.Listen("unix", pipePath)
	if err != nil {
		return "", nil, fmt.Errorf("listen on %s: %w", pipePath, err)
	}

	cleanup := func() {
		ln.Close()
		os.Remove(pipePath)
	}

	// Accept connection in background, store it once connected
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("IPC accept error: %v", err)
			return
		}
		m.mu.Lock()
		m.conn = conn
		m.mu.Unlock()
	}()

	return pipePath, cleanup, nil
}

func (m *Manager) waitForConnection(initData *ipc.ExtensionHostInitData, cleanup func()) {
	defer close(m.done)

	// Wait up to 60s for the connection
	for i := 0; i < 600; i++ {
		m.mu.Lock()
		conn := m.conn
		m.mu.Unlock()
		if conn != nil {
			m.handleConnection(conn, initData, cleanup)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	m.mu.Lock()
	m.err = fmt.Errorf("extension host connection timeout")
	m.mu.Unlock()
	cleanup()
	close(m.ready)
}

func (m *Manager) handleConnection(conn net.Conn, initData *ipc.ExtensionHostInitData, cleanup func()) {
	log.Println("extension host connected, performing handshake")

	handler := &rpcHandler{}
	connection := ipc.NewConnection(conn, handler)

	// Start the handshake
	if err := connection.SendInitData(initData); err != nil {
		m.mu.Lock()
		m.err = fmt.Errorf("send init data: %w", err)
		m.mu.Unlock()
		cleanup()
		close(m.ready)
		return
	}

	// Read messages until Ready is received, then send Initialized
	go func() {
		// Wait for Ready message, then send Initialized
		// In a full implementation, we'd parse the full protocol here
		time.Sleep(100 * time.Millisecond) // give ext host time to process
		if err := connection.SendInitialized(); err != nil {
			log.Printf("error sending initialized: %v", err)
		}
	}()

	// Start message loop
	if err := connection.HandleConnection(); err != nil {
		log.Printf("connection error: %v", err)
	}

	cleanup()
}

// Stop terminates the extension host process.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill extension host: %w", err)
		}
	}

	if m.conn != nil {
		m.conn.Close()
	}

	return nil
}

// Err returns any error that occurred during startup or operation.
func (m *Manager) Err() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Done returns a channel that's closed when the extension host exits.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// rpcHandler handles RPC messages from the extension host.
type rpcHandler struct {
	mu      sync.Mutex
	pending map[uint32]chan []byte
	nextID  uint32
}

func (h *rpcHandler) HandleRPC(msg *ipc.ProtocolMessage) {
	log.Printf("received RPC: id=%d ack=%d len=%d", msg.ID, msg.Ack, len(msg.Data))
	// In a full implementation, this would dispatch to channel handlers
	// based on the RPC protocol defined in extHost.protocol.ts
}

// Ensure io is referenced (for potential future use)
var _ io.Writer = (*dummyWriter)(nil)

type dummyWriter struct{}

func (*dummyWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
