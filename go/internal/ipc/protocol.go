/*
PersistentProtocol framing implementation for VS Code extension host IPC.

Protocol format (from src/vs/base/parts/ipc/common/ipc.net.ts):
  - Header: 13 bytes
    - Byte 0:   ProtocolMessageType (uint8)
    - Bytes 1-4: id (uint32 big-endian)
    - Bytes 5-8: ack (uint32 big-endian)
    - Bytes 9-12: data length (uint32 big-endian)
  - Body: data length bytes

MessageType (from extensionHostProtocol.ts) - used as Control messages:
  Initialized = 1
  Ready = 2
  Terminate = 3
*/

package ipc

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	HeaderLength = 13

	// ProtocolMessageType values
	MsgTypeNone          uint8 = 0
	MsgTypeRegular       uint8 = 1
	MsgTypeControl       uint8 = 2
	MsgTypeAck           uint8 = 3
	MsgTypeDisconnect    uint8 = 5
	MsgTypeReplayRequest uint8 = 6
	MsgTypePause         uint8 = 7
	MsgTypeResume        uint8 = 8
	MsgTypeKeepAlive     uint8 = 9

	// Handshake message types (sent as Control messages with 1-byte payload)
	MsgInitialized uint8 = 1
	MsgReady       uint8 = 2
	MsgTerminate   uint8 = 3
)

// ProtocolMessage represents a single framed message on the wire.
type ProtocolMessage struct {
	Type uint8
	ID   uint32
	Ack  uint32
	Data []byte
}

// Marshal encodes a ProtocolMessage to its wire format.
func (m *ProtocolMessage) Marshal() ([]byte, error) {
	buf := make([]byte, HeaderLength+len(m.Data))
	buf[0] = m.Type
	binary.BigEndian.PutUint32(buf[1:5], m.ID)
	binary.BigEndian.PutUint32(buf[5:9], m.Ack)
	binary.BigEndian.PutUint32(buf[9:13], uint32(len(m.Data)))
	copy(buf[13:], m.Data)
	return buf, nil
}

// Unmarshal decodes a wire-format message into a ProtocolMessage.
func Unmarshal(data []byte) (*ProtocolMessage, error) {
	if len(data) < HeaderLength {
		return nil, fmt.Errorf("buffer too short: %d < %d", len(data), HeaderLength)
	}
	m := &ProtocolMessage{
		Type: data[0],
		ID:   binary.BigEndian.Uint32(data[1:5]),
		Ack:  binary.BigEndian.Uint32(data[5:9]),
		Data: make([]byte, binary.BigEndian.Uint32(data[9:13])),
	}
	copy(m.Data, data[HeaderLength:HeaderLength+len(m.Data)])
	return m, nil
}

// Writer provides methods to write ProtocolMessages to an io.Writer.
type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (cw *Writer) WriteMessage(msg *ProtocolMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}
	_, err = cw.w.Write(data)
	return err
}

// ReadMessage reads a single frame from the reader.
func ReadMessage(r io.Reader) (*ProtocolMessage, error) {
	header := make([]byte, HeaderLength)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	msgLen := binary.BigEndian.Uint32(header[9:13])
	data := make([]byte, msgLen)
	if msgLen > 0 {
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
	}

	return &ProtocolMessage{
		Type: header[0],
		ID:   binary.BigEndian.Uint32(header[1:5]),
		Ack:  binary.BigEndian.Uint32(header[5:9]),
		Data: data,
	}, nil
}
