/*
Connection manages the VS Code extension host IPC connection.

Implements the handshake from extensionHostProcess.ts:
  1. Go sends IExtensionHostInitData as a Regular message (first message)
  2. Node ext host sends MessageType.Ready (Control, 1 byte: 0x02)
  3. Go sends MessageType.Initialized (Control, 1 byte: 0x01)
  4. RPC begins over channels (extHost.protocol.ts)
*/

package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// LogLevel mirrors VS Code's LogLevel enum
type LogLevel int

const (
	LogLevelTrace LogLevel = 0
	LogLevelDebug LogLevel = 1
	LogLevelInfo  LogLevel = 2
	LogLevelWarn  LogLevel = 3
	LogLevelError LogLevel = 4
	LogLevelOff   LogLevel = 6
)

// URI components matching VS Code's URI format
type URI struct {
	Mid       int    `json:"$mid"`
	Scheme    string `json:"scheme"`
	Path      string `json:"path,omitempty"`
	Fragment  string `json:"fragment,omitempty"`
	Query     string `json:"query,omitempty"`
}

func NewURI(scheme, path string) URI {
	return URI{Mid: 1, Scheme: scheme, Path: path}
}

// IExtensionHostInitData mirrors the TS interface
type ExtensionHostInitData struct {
	Version       string                   `json:"version"`
	Quality       string                   `json:"quality,omitempty"`
	Commit        string                   `json:"commit,omitempty"`
	Date          string                   `json:"date,omitempty"`
	ParentPID     int                      `json:"parentPid"`
	Environment   ExtensionHostEnvironment `json:"environment"`
	Workspace     *StaticWorkspaceData     `json:"workspace,omitempty"`
	Extensions    ExtensionDescriptionSnapshot `json:"extensions"`
	NLSBaseURL    *URI                     `json:"nlsBaseUrl,omitempty"`
	TelemetryInfo TelemetryInfo            `json:"telemetryInfo"`
	LogLevel      LogLevel                 `json:"logLevel"`
	Loggers       []LoggerResource         `json:"loggers,omitempty"`
	LogsLocation  URI                      `json:"logsLocation"`
	AutoStart     bool                     `json:"autoStart"`
	Remote        RemoteInfo               `json:"remote"`
	ConsoleForward ConsoleForward          `json:"consoleForward"`
	UIKind        uint                     `json:"uiKind"`
	Handle        string                   `json:"handle,omitempty"`
}

type ExtensionHostEnvironment struct {
	IsExtensionDevelopmentDebug bool      `json:"isExtensionDevelopmentDebug"`
	AppName                    string    `json:"appName"`
	AppHost                    string    `json:"appHost"`
	AppRoot                    *URI      `json:"appRoot,omitempty"`
	AppLanguage                string    `json:"appLanguage"`
	IsExtensionTelemetryLoggingOnly bool  `json:"isExtensionTelemetryLoggingOnly"`
	AppURIScheme               string    `json:"appUriScheme"`
	IsPortable                 *bool     `json:"isPortable,omitempty"`
	GlobalStorageHome          URI       `json:"globalStorageHome"`
	WorkspaceStorageHome       URI       `json:"workspaceStorageHome"`
	UseHostProxy               *bool     `json:"useHostProxy,omitempty"`
	SkipWorkspaceStorageLock   *bool     `json:"skipWorkspacestorageLock,omitempty"`
}

type StaticWorkspaceData struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Transient       *bool     `json:"transient,omitempty"`
}

type ExtensionDescriptionSnapshot struct {
	VersionID      int                     `json:"versionId"`
	AllExtensions  []ExtensionDescription  `json:"allExtensions"`
	ActivationEvents map[string][]string   `json:"activationEvents"`
	MyExtensions   []ExtensionIdentifier   `json:"myExtensions"`
}

type ExtensionDescription struct {
	ID             string            `json:"id"`
	Identifier     ExtensionIdentifier `json:"identifier"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Engines        map[string]string `json:"engines"`
	Categories     []string          `json:"categories,omitempty"`
}

type ExtensionIdentifier struct {
	Value string `json:"value"`
}

type TelemetryInfo struct {
	SessionID       string `json:"sessionId"`
	MachineID       string `json:"machineId"`
	SquareID        string `json:"sqmId"`
	DeviceID        string `json:"devDeviceId"`
	FirstSessionDate string `json:"firstSessionDate"`
}

type LoggerResource struct {
	Resource URI     `json:"resource"`
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	LogLevel *int    `json:"logLevel,omitempty"`
	Hidden   *bool   `json:"hidden,omitempty"`
}

type RemoteInfo struct {
	IsRemote      bool   `json:"isRemote"`
	Authority     string `json:"authority,omitempty"`
	ConnectionData json.RawMessage `json:"connectionData"` // null for local
}

type ConsoleForward struct {
	IncludeStack bool `json:"includeStack"`
	LogNative    bool `json:"logNative"`
}

// Handler is called when a Regular (RPC) message is received.
type Handler interface {
	HandleRPC(msg *ProtocolMessage)
}

// Connection wraps the socket connection and handles framing + handshake.
type Connection struct {
	conn     net.Conn
	reader   *bufio.Reader
	writerMu sync.Mutex
	handler  Handler

	// RPC state
	outgoingMsgID uint32
	incomingMsgID uint32
	incomingAckID uint32
	outgoingAckID uint32
}

// NewConnection creates a connection from an accepted socket.
func NewConnection(conn net.Conn, handler Handler) *Connection {
	return &Connection{
		conn:   conn,
		reader: bufio.NewReader(conn),
		handler: handler,
	}
}

// SendInitData sends the initial handshake data as a Regular message.
func (c *Connection) SendInitData(initData *ExtensionHostInitData) error {
	data, err := json.Marshal(initData)
	if err != nil {
		return fmt.Errorf("marshal init data: %w", err)
	}

	msg := &ProtocolMessage{
		Type: MsgTypeRegular,
		ID:   c.nextOutgoingID(),
		Ack:  0,
		Data: data,
	}
	return c.writeMessage(msg)
}

// SendReady sends the Ready control message.
func (c *Connection) SendInitialized() error {
	msg := &ProtocolMessage{
		Type: MsgTypeControl,
		ID:   c.nextOutgoingID(),
		Ack:  0,
		Data: []byte{MsgInitialized},
	}
	return c.writeMessage(msg)
}

func (c *Connection) nextOutgoingID() uint32 {
	c.outgoingMsgID++
	return c.outgoingMsgID
}

func (c *Connection) writeMessage(msg *ProtocolMessage) error {
	c.writerMu.Lock()
	defer c.writerMu.Unlock()
	return NewWriter(c.conn).WriteMessage(msg)
}

// HandleConnection runs the message loop until error.
// Performs handshake first, then dispatches RPC messages.
func (c *Connection) HandleConnection() error {
	// Read and dispatch messages
	for {
		msg, err := ReadMessage(c.reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		switch msg.Type {
		case MsgTypeControl:
			if len(msg.Data) == 1 {
				switch msg.Data[0] {
				case MsgReady:
					log.Println("extension host ready")
				case MsgTerminate:
					log.Println("extension host terminated")
					return nil
				}
			}

		case MsgTypeAck:
			// Ack handling
			c.incomingAckID = msg.Ack

		case MsgTypeRegular:
			// RPC message
			if c.handler != nil {
				c.handler.HandleRPC(msg)
			}

		case MsgTypeKeepAlive:
			// respond to keepalive

		case MsgTypeDisconnect:
			return nil
		}
	}
}

// SendRPC sends a regular RPC message (for requests from Go to ext host).
func (c *Connection) SendRPC(data []byte) error {
	msg := &ProtocolMessage{
		Type: MsgTypeRegular,
		ID:   c.nextOutgoingID(),
		Ack:  0,
		Data: data,
	}
	return c.writeMessage(msg)
}
