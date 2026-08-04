// Package ipc implements a Go-based IPC (Inter-Process Communication) framework
// compatible with VS Code's IPC protocol. It provides binary serialization,
// channel-based request/response communication, and event subscription
// over WebSocket or pipe transports.
package ipc

// RequestType identifies the kind of request being sent from client to server.
type RequestType int

const (
	RequestTypePromise        RequestType = 100
	RequestTypePromiseCancel  RequestType = 101
	RequestTypeEventListen    RequestType = 102
	RequestTypeEventDispose   RequestType = 103
)

func (r RequestType) String() string {
	switch r {
	case RequestTypePromise:
		return "req"
	case RequestTypePromiseCancel:
		return "cancel"
	case RequestTypeEventListen:
		return "subscribe"
	case RequestTypeEventDispose:
		return "unsubscribe"
	}
	return "unknown"
}

// ResponseType identifies the kind of response being sent from server to client.
type ResponseType int

const (
	ResponseTypeInitialize     ResponseType = 200
	ResponseTypePromiseSuccess ResponseType = 201
	ResponseTypePromiseError   ResponseType = 202
	ResponseTypePromiseErrorObj ResponseType = 203
	ResponseTypeEventFire      ResponseType = 204
)

// DataType identifies the type of a serialized value.
type DataType int

const (
	DataTypeUndefined DataType = 0
	DataTypeString    DataType = 1
	DataTypeBuffer    DataType = 2
	DataTypeVSBuffer  DataType = 3
	DataTypeArray     DataType = 4
	DataTypeObject    DataType = 5
	DataTypeInt       DataType = 6
)

// ProtocolMessageType identifies the type of a protocol-level message.
type ProtocolMessageType int

const (
	ProtocolMessageTypeNone           ProtocolMessageType = 0
	ProtocolMessageTypeRegular        ProtocolMessageType = 1
	ProtocolMessageTypeControl        ProtocolMessageType = 2
	ProtocolMessageTypeAck            ProtocolMessageType = 3
	ProtocolMessageTypeDisconnect     ProtocolMessageType = 5
	ProtocolMessageTypeReplayRequest  ProtocolMessageType = 6
	ProtocolMessageTypePause          ProtocolMessageType = 7
	ProtocolMessageTypeResume         ProtocolMessageType = 8
	ProtocolMessageTypeKeepAlive      ProtocolMessageType = 9
)

// Protocol constants.
const (
	HeaderLength         = 13
	AcknowledgeTime      = 2000
	TimeoutTime          = 20000
	KeepAliveSendTime    = 5000
	ReconnectionGraceTime = 3 * 60 * 60 * 1000
	ReconnectionShortGraceTime = 5 * 60 * 1000
)

// Raw request types matching VS Code's IPC wire format.
type RawPromiseRequest struct {
	Type        RequestType
	ID          int
	ChannelName string
	Name        string
	Arg         interface{}
}

type RawPromiseCancelRequest struct {
	Type RequestType
	ID   int
}

type RawEventListenRequest struct {
	Type        RequestType
	ID          int
	ChannelName string
	Name        string
	Arg         interface{}
}

type RawEventDisposeRequest struct {
	Type RequestType
	ID   int
}

// Raw response types matching VS Code's IPC wire format.
type RawInitializeResponse struct {
	Type ResponseType
}

type RawPromiseSuccessResponse struct {
	Type ResponseType
	ID   int
	Data interface{}
}

type RawPromiseErrorResponse struct {
	Type ResponseType
	ID   int
	Data struct {
		Message string   `json:"message"`
		Name    string   `json:"name"`
		Stack   []string `json:"stack,omitempty"`
	}
}

type RawPromiseErrorObjResponse struct {
	Type ResponseType
	ID   int
	Data interface{}
}

type RawEventFireResponse struct {
	Type ResponseType
	ID   int
	Data interface{}
}

// RequestContext is the type passed to channel handlers.
type RequestContext string

// ServerChannel defines the server-side channel interface.
type ServerChannel interface {
	Call(ctx RequestContext, command string, arg interface{}, cancelToken <-chan struct{}) (interface{}, error)
	Listen(ctx RequestContext, event string, arg interface{}) (<-chan interface{}, func())
}

// IPCMessagePassingProtocol defines the transport layer interface.
type IPCMessagePassingProtocol interface {
	Send(buffer []byte)
	OnMessage() <-chan []byte
	Drain() error
	Close()
}

// ServerConnection represents a single client connection.
type ServerConnection struct {
	Ctx            RequestContext
	ChannelServer  *ChannelServer
	ChannelClient  *ChannelClient
}

// ClientConnectionEvent is emitted when a new client connects.
type ClientConnectionEvent struct {
	Protocol        IPCMessagePassingProtocol
	OnClientDisconnect func()
}

// ClientFilter is a function that filters connections.
type ClientFilter func(ctx RequestContext) bool
