package ipc

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// IPCServer is the main server that manages channels and connections.
type IPCServer struct {
	server       *SocketServer
	channels     map[string]ServerChannel
	mu           sync.RWMutex
}

// NewIPCServer creates a new IPC server.
func NewIPCServer() *IPCServer {
	return &IPCServer{
		server:   NewSocketServer(),
		channels: make(map[string]ServerChannel),
	}
}

// RegisterChannel registers a channel on the server.
func (s *IPCServer) RegisterChannel(name string, channel ServerChannel) {
	s.mu.Lock()
	s.channels[name] = channel
	s.mu.Unlock()
	s.server.RegisterChannel(name, channel)
}

// CreateConnection creates a new connection from a protocol.
func (s *IPCServer) CreateConnection(protocol IPCMessagePassingProtocol, ctx RequestContext) *ServerConnection {
	conn := s.server.AddConnection(protocol, ctx)
	s.mu.RLock()
	channelNames := make([]string, 0, len(s.channels))
	for name := range s.channels {
		channelNames = append(channelNames, name)
	}
	s.mu.RUnlock()

	log.Printf("[IPCServer] new connection: ctx=%s, channels=%v", ctx, channelNames)
	return conn
}

// OnConnect returns the channel for new connection events.
func (s *IPCServer) OnConnect() <-chan *ServerConnection {
	return s.server.OnConnect()
}

// Stop shuts down the IPC server.
func (s *IPCServer) Stop() {
	s.server.Stop()
}

// NewWeblateIPCServer creates an IPC server backed by WebSocket transport.
func NewWeblateIPCServer(port int) (*IPCServer, *WebSocketServer) {
	server := NewIPCServer()

	wsServer := NewWebSocketServer("/vscode-remote", port, func(protocol *Protocol) {
		ctx := RequestContext("remote")
		server.CreateConnection(protocol, ctx)
	})

	return server, wsServer
}

// NewTCPIPCServer creates an IPC server backed by TCP transport.
func NewTCPIPCServer(port int) (*IPCServer, net.Listener, error) {
	server := NewIPCServer()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			tcpSocket := NewTCPWrapper(conn)
			protocol := NewProtocol(tcpSocket)
			server.CreateConnection(protocol, "tcp")
		}
	}()

	return server, listener, nil
}
