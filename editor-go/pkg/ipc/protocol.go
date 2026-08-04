package ipc

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Protocol handles the VS Code persistent protocol layer.
type Protocol struct {
	socket   ISocket
	reader   *ProtocolReader
	writer   *ProtocolWriter
	onMessage chan []byte
	onDisconnect chan struct{}
	mu       sync.RWMutex
	closed   bool
}

// ISocket abstracts the underlying transport.
type ISocket interface {
	OnData() <-chan []byte
	OnClose() <-chan struct{}
	Write(data []byte) error
	Close()
	Drain() error
}

// ProtocolMessage represents a protocol-level message.
type ProtocolMessage struct {
	Type ProtocolMessageType
	ID   int
	Ack  int
	Data []byte
}

// ProtocolReader reads protocol messages from raw data.
type ProtocolReader struct {
	socket    ISocket
	incoming  *ChunkStream
	onMessage chan *ProtocolMessage
	lastRead  time.Time
	state     struct {
		readHead    bool
		readLen     int
		messageType ProtocolMessageType
		id          int
		ack         int
	}
}

// NewProtocolReader creates a new protocol reader.
func NewProtocolReader(socket ISocket) *ProtocolReader {
	pr := &ProtocolReader{
		socket:    socket,
		incoming:  NewChunkStream(),
		onMessage: make(chan *ProtocolMessage, 100),
		lastRead:  time.Now(),
	}
	pr.state.readHead = true
	pr.state.readLen = HeaderLength
	pr.state.messageType = ProtocolMessageTypeNone

	go pr.readLoop()
	return pr
}

func (pr *ProtocolReader) readLoop() {
	for chunk := range pr.socket.OnData() {
		if chunk == nil || len(chunk) == 0 {
			continue
		}
		pr.lastRead = time.Now()
		pr.incoming.AcceptChunk(chunk)
		pr.processChunks()
	}
}

func (pr *ProtocolReader) processChunks() {
	for pr.incoming.ByteLength() >= pr.state.readLen {
		buff := pr.incoming.Read(pr.state.readLen)
		if pr.state.readHead {
			pr.state.readHead = false
			pr.state.readLen = int(binary.BigEndian.Uint32(buff[9:13]))
			pr.state.messageType = ProtocolMessageType(buff[0])
			pr.state.id = int(binary.BigEndian.Uint32(buff[1:5]))
			pr.state.ack = int(binary.BigEndian.Uint32(buff[5:9]))
		} else {
			msgType := pr.state.messageType
			id := pr.state.id
			ack := pr.state.ack

			pr.state.readHead = true
			pr.state.readLen = HeaderLength
			pr.state.messageType = ProtocolMessageTypeNone
			pr.state.id = 0
			pr.state.ack = 0

			msg := &ProtocolMessage{
				Type: msgType,
				ID:   id,
				Ack:  ack,
				Data: buff,
			}
			pr.onMessage <- msg
		}
	}
}

// ProtocolWriter writes protocol messages.
type ProtocolWriter struct {
	socket  ISocket
	buffer  []byte
	pending [][]byte
	pLen    int
	mu      sync.Mutex
	paused  bool
}

// NewProtocolWriter creates a new protocol writer.
func NewProtocolWriter(socket ISocket) *ProtocolWriter {
	return &ProtocolWriter{
		socket: socket,
	}
}

func (pw *ProtocolWriter) Write(msg *ProtocolMessage) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.paused {
		return
	}

	header := make([]byte, HeaderLength)
	header[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(header[1:5], uint32(msg.ID))
	binary.BigEndian.PutUint32(header[5:9], uint32(msg.Ack))
	binary.BigEndian.PutUint32(header[9:13], uint32(len(msg.Data)))

	pw.pending = append(pw.pending, header, msg.Data)
	pw.pLen += len(header) + len(msg.Data)

	if len(pw.pending) == 2 {
		go pw.flush()
	}
}

func (pw *ProtocolWriter) flush() {
	pw.mu.Lock()
	if pw.pLen == 0 || pw.paused {
		pw.mu.Unlock()
		return
	}

	data := make([]byte, pw.pLen)
	offset := 0
	for _, chunk := range pw.pending {
		copy(data[offset:], chunk)
		offset += len(chunk)
	}
	pw.pending = nil
	pw.pLen = 0
	pw.mu.Unlock()

	_ = pw.socket.Write(data)
}

func (pw *ProtocolWriter) Pause() {
	pw.mu.Lock()
	pw.paused = true
	pw.mu.Unlock()
}

func (pw *ProtocolWriter) Resume() {
	pw.mu.Lock()
	pw.paused = false
	pw.mu.Unlock()
	pw.flush()
}

// ChunkStream accumulates incoming data chunks.
type ChunkStream struct {
	chunks [][]byte
	total  int
}

func NewChunkStream() *ChunkStream {
	return &ChunkStream{}
}

func (cs *ChunkStream) AcceptChunk(data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	cs.chunks = append(cs.chunks, cp)
	cs.total += len(data)
}

func (cs *ChunkStream) ByteLength() int {
	return cs.total
}

func (cs *ChunkStream) Read(n int) []byte {
	if n == 0 {
		return []byte{}
	}
	if n > cs.total {
		return nil
	}

	if len(cs.chunks) > 0 && len(cs.chunks[0]) == n {
		result := cs.chunks[0]
		cs.chunks = cs.chunks[1:]
		cs.total -= n
		return result
	}

	if len(cs.chunks) > 0 && len(cs.chunks[0]) > n {
		result := cs.chunks[0][:n]
		cs.chunks[0] = cs.chunks[0][n:]
		cs.total -= n
		return result
	}

	result := make([]byte, n)
	offset := 0
	for n > 0 && len(cs.chunks) > 0 {
		chunk := cs.chunks[0]
		if len(chunk) > n {
			copy(result[offset:], chunk[:n])
			offset += n
			cs.chunks[0] = chunk[n:]
			cs.total -= n
			n = 0
		} else {
			copy(result[offset:], chunk)
			offset += len(chunk)
			cs.total -= len(chunk)
			n -= len(chunk)
			cs.chunks = cs.chunks[1:]
		}
	}
	return result
}

// NewProtocol creates a new IPC protocol instance.
func NewProtocol(socket ISocket) *Protocol {
	p := &Protocol{
		socket:       socket,
		onMessage:    make(chan []byte, 100),
		onDisconnect: make(chan struct{}),
	}

	p.reader = NewProtocolReader(socket)
	p.writer = NewProtocolWriter(socket)

	go p.messageLoop()
	go p.disconnectLoop()

	return p
}

func (p *Protocol) messageLoop() {
	for msg := range p.reader.onMessage {
		if msg.Type == ProtocolMessageTypeRegular {
			p.onMessage <- msg.Data
		}
	}
}

func (p *Protocol) disconnectLoop() {
	<-p.socket.OnClose()
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.onDisconnect)
	}
	p.mu.Unlock()
}

// Send sends a raw message over the protocol.
func (p *Protocol) Send(buffer []byte) {
	p.writer.Write(&ProtocolMessage{
		Type: ProtocolMessageTypeRegular,
		ID:   0,
		Ack:  0,
		Data: buffer,
	})
}

// OnMessage returns the channel for incoming messages.
func (p *Protocol) OnMessage() <-chan []byte {
	return p.onMessage
}

// Drain flushes pending writes.
func (p *Protocol) Drain() error {
	p.writer.flush()
	return p.socket.Drain()
}

// Close closes the protocol.
func (p *Protocol) Close() {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.onMessage)
		p.socket.Close()
	}
	p.mu.Unlock()
}

// WebSocketSocket wraps a gorilla/websocket connection as an ISocket.
type WebSocketSocket struct {
	conn     *websocket.Conn
	onData   chan []byte
	onClose  chan struct{}
	closeOnce sync.Once
}

// NewWebSocketSocket creates a new WebSocket socket wrapper.
func NewWebSocketSocket(conn *websocket.Conn) *WebSocketSocket {
	ws := &WebSocketSocket{
		conn:    conn,
		onData:  make(chan []byte, 100),
		onClose: make(chan struct{}),
	}
	go ws.readLoop()
	return ws
}

func (ws *WebSocketSocket) readLoop() {
	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			ws.Close()
			return
		}
		cp := make([]byte, len(data))
		copy(cp, data)
		ws.onData <- cp
	}
}

// OnData returns the channel for incoming data.
func (ws *WebSocketSocket) OnData() <-chan []byte { return ws.onData }

// OnClose returns the channel for close events.
func (ws *WebSocketSocket) OnClose() <-chan struct{} { return ws.onClose }

// Write sends data over the WebSocket.
func (ws *WebSocketSocket) Write(data []byte) error {
	return ws.conn.WriteMessage(websocket.BinaryMessage, data)
}

// Close closes the WebSocket connection.
func (ws *WebSocketSocket) Close() {
	ws.closeOnce.Do(func() {
		ws.conn.Close()
		close(ws.onClose)
	})
}

// Drain flushes any pending writes (no-op for WebSocket).
func (ws *WebSocketSocket) Drain() error { return nil }

// TCPWrapper wraps a net.Conn as an ISocket.
type TCPWrapper struct {
	conn     net.Conn
	onData   chan []byte
	onClose  chan struct{}
	closeOnce sync.Once
}

// NewTCPWrapper creates a new TCP socket wrapper.
func NewTCPWrapper(conn net.Conn) *TCPWrapper {
	tw := &TCPWrapper{
		conn:    conn,
		onData:  make(chan []byte, 100),
		onClose: make(chan struct{}),
	}
	go tw.readLoop()
	return tw
}

func (tw *TCPWrapper) readLoop() {
	buf := make([]byte, 65536)
	for {
		n, err := tw.conn.Read(buf)
		if err != nil {
			tw.Close()
			return
		}
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, buf[:n])
			tw.onData <- cp
		}
	}
}

// OnData returns the channel for incoming data.
func (tw *TCPWrapper) OnData() <-chan []byte { return tw.onData }

// OnClose returns the channel for close events.
func (tw *TCPWrapper) OnClose() <-chan struct{} { return tw.onClose }

// Write sends data over the TCP connection.
func (tw *TCPWrapper) Write(data []byte) error {
	_, err := tw.conn.Write(data)
	return err
}

// Close closes the TCP connection.
func (tw *TCPWrapper) Close() {
	tw.closeOnce.Do(func() {
		tw.conn.Close()
		close(tw.onClose)
	})
}

// Drain flushes any pending writes.
func (tw *TCPWrapper) Drain() error { return nil }

// WebSocketServer handles WebSocket connections and creates IPC protocols.
type WebSocketServer struct {
	upgrader websocket.Upgrader
	handler  func(*Protocol)
	mux      *http.ServeMux
	server   *http.Server
}

// NewWebSocketServer creates a new WebSocket server.
func NewWebSocketServer(path string, port int, handler func(*Protocol)) *WebSocketServer {
	wss := &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		handler: handler,
		mux:     http.NewServeMux(),
	}

	wss.mux.HandleFunc(path, wss.handleWebSocket)
	wss.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wss.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: wss.mux,
	}

	return wss
}

func (wss *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wss.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocketServer] upgrade failed: %v", err)
		return
	}

	socket := NewWebSocketSocket(conn)
	protocol := NewProtocol(socket)
	wss.handler(protocol)
}

// Start starts the WebSocket server.
func (wss *WebSocketServer) Start() error {
	log.Printf("[WebSocketServer] starting on %s", wss.server.Addr)
	return wss.server.ListenAndServe()
}

// Stop gracefully stops the WebSocket server.
func (wss *WebSocketServer) Stop() error {
	return wss.server.Close()
}

// SocketServer manages multiple IPC connections and routes channels.
type SocketServer struct {
	channels      map[string]ServerChannel
	connections   map[string]*ServerConnection
	onConnect     chan *ServerConnection
	onDisconnect  chan *ServerConnection
	mu            sync.RWMutex
}

// NewSocketServer creates a new socket server.
func NewSocketServer() *SocketServer {
	return &SocketServer{
		channels:     make(map[string]ServerChannel),
		connections:  make(map[string]*ServerConnection),
		onConnect:    make(chan *ServerConnection, 100),
		onDisconnect: make(chan *ServerConnection, 100),
	}
}

// RegisterChannel registers a channel on all current and future connections.
func (ss *SocketServer) RegisterChannel(name string, channel ServerChannel) {
	ss.mu.Lock()
	ss.channels[name] = channel
	connections := make([]*ServerConnection, 0, len(ss.connections))
	for _, conn := range ss.connections {
		connections = append(connections, conn)
	}
	ss.mu.Unlock()

	for _, conn := range connections {
		conn.ChannelServer.RegisterChannel(name, channel)
	}
}

// AddConnection adds a new connection to the server.
func (ss *SocketServer) AddConnection(protocol IPCMessagePassingProtocol, ctx RequestContext) *ServerConnection {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	cs := NewChannelServer(protocol, ctx)
	cc := NewChannelClient(protocol)

	// Register all existing channels on the new connection
	for name, channel := range ss.channels {
		cs.RegisterChannel(name, channel)
	}

	conn := &ServerConnection{
		Ctx:           ctx,
		ChannelServer: cs,
		ChannelClient: cc,
	}

	ss.connections[string(ctx)] = conn
	ss.onConnect <- conn

	go func() {
		<-protocol.OnMessage()
		// Connection handling is done by the ChannelServer read loop
	}()

	// Monitor for disconnection
	go func() {
		select {
		case <-protocol.OnMessage():
			// Check if connection is still alive
		case <-time.After(30 * time.Second):
			// Timeout check
		}
	}()

	return conn
}

// OnConnect returns the channel for new connection events.
func (ss *SocketServer) OnConnect() <-chan *ServerConnection {
	return ss.onConnect
}

// OnDisconnect returns the channel for disconnection events.
func (ss *SocketServer) OnDisconnect() <-chan *ServerConnection {
	return ss.onDisconnect
}

// Stop shuts down all connections.
func (ss *SocketServer) Stop() {
	ss.mu.Lock()
	for _, conn := range ss.connections {
		conn.ChannelServer.Stop()
		conn.ChannelClient.Stop()
	}
	ss.connections = make(map[string]*ServerConnection)
	ss.mu.Unlock()
}
