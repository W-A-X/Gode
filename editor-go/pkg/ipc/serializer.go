package ipc

import (
	"encoding/json"
	"fmt"
)

// BufferReader reads from a byte slice.
type BufferReader struct {
	buf  []byte
	pos  int
}

func NewBufferReader(data []byte) *BufferReader {
	return &BufferReader{buf: data, pos: 0}
}

func (r *BufferReader) Read(n int) []byte {
	if r.pos+n > len(r.buf) {
		return nil
	}
	result := make([]byte, n)
	copy(result, r.buf[r.pos:r.pos+n])
	r.pos += n
	return result
}

func (r *BufferReader) ReadAll() []byte {
	result := make([]byte, len(r.buf)-r.pos)
	copy(result, r.buf[r.pos:])
	r.pos = len(r.buf)
	return result
}

func (r *BufferReader) Pos() int { return r.pos }

// BufferWriter writes to a byte buffer.
type BufferWriter struct {
	buffers [][]byte
	total   int
}

func NewBufferWriter() *BufferWriter {
	return &BufferWriter{}
}

func (w *BufferWriter) Write(data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	w.buffers = append(w.buffers, cp)
	w.total += len(data)
}

func (w *BufferWriter) Buffer() []byte {
	result := make([]byte, w.total)
	offset := 0
	for _, b := range w.buffers {
		copy(result[offset:], b)
		offset += len(b)
	}
	return result
}

func (w *BufferWriter) Reset() {
	w.buffers = w.buffers[:0]
	w.total = 0
}

// readIntVQL reads a Variable Length Quantity encoded 32-bit integer.
func readIntVQL(reader *BufferReader) (int, error) {
	value := 0
	for n := 0; ; n += 7 {
		b := reader.Read(1)
		if b == nil || len(b) == 0 {
			return 0, fmt.Errorf("unexpected end of VQL data")
		}
		value |= int(b[0]&0b01111111) << n
		if b[0]&0b10000000 == 0 {
			return value, nil
		}
	}
}

// writeInt32VQL writes a Variable Length Quantity encoded 32-bit integer.
func writeInt32VQL(writer *BufferWriter, value int) {
	if value == 0 {
		writer.Write([]byte{0})
		return
	}
	var scratch [4]byte
	for i := 0; value != 0; i++ {
		scratch[i] = byte(value & 0b01111111)
		value >>= 7
		if value > 0 {
			scratch[i] |= 0b10000000
		}
	}
	writer.Write(scratch[:])
}

// serialize writes a value using VS Code's binary serialization format.
func serialize(writer *BufferWriter, data interface{}) {
	switch v := data.(type) {
	case nil:
		writer.Write([]byte{byte(DataTypeUndefined)})

	case string:
		strBytes := []byte(v)
		writer.Write([]byte{byte(DataTypeString)})
		writeInt32VQL(writer, len(strBytes))
		writer.Write(strBytes)

	case []byte:
		writer.Write([]byte{byte(DataTypeVSBuffer)})
		writeInt32VQL(writer, len(v))
		writer.Write(v)

	case int:
		writer.Write([]byte{byte(DataTypeInt)})
		writeInt32VQL(writer, v)

	case int32:
		writer.Write([]byte{byte(DataTypeInt)})
		writeInt32VQL(writer, int(v))

	case int64:
		writer.Write([]byte{byte(DataTypeInt)})
		writeInt32VQL(writer, int(v))

	case float64:
		writer.Write([]byte{byte(DataTypeInt)})
		writeInt32VQL(writer, int(v))

	case bool:
		if v {
			writer.Write([]byte{byte(DataTypeInt)})
			writeInt32VQL(writer, 1)
		} else {
			writer.Write([]byte{byte(DataTypeInt)})
			writeInt32VQL(writer, 0)
		}

	case []interface{}:
		writer.Write([]byte{byte(DataTypeArray)})
		writeInt32VQL(writer, len(v))
		for _, el := range v {
			serialize(writer, el)
		}

	case map[string]interface{}:
		jsonBytes, _ := json.Marshal(v)
		writer.Write([]byte{byte(DataTypeObject)})
		writeInt32VQL(writer, len(jsonBytes))
		writer.Write(jsonBytes)

	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			writer.Write([]byte{byte(DataTypeUndefined)})
			return
		}
		writer.Write([]byte{byte(DataTypeObject)})
		writeInt32VQL(writer, len(jsonBytes))
		writer.Write(jsonBytes)
	}
}

// deserialize reads a value using VS Code's binary serialization format.
func deserialize(reader *BufferReader) (interface{}, error) {
	typeByte := reader.Read(1)
	if typeByte == nil {
		return nil, fmt.Errorf("unexpected end of data")
	}
	dt := DataType(typeByte[0])

	switch dt {
	case DataTypeUndefined:
		return nil, nil

	case DataTypeString:
		length, err := readIntVQL(reader)
		if err != nil {
			return nil, err
		}
		data := reader.Read(length)
		if data == nil {
			return nil, fmt.Errorf("unexpected end of string data")
		}
		return string(data), nil

	case DataTypeBuffer, DataTypeVSBuffer:
		length, err := readIntVQL(reader)
		if err != nil {
			return nil, err
		}
		data := reader.Read(length)
		if data == nil {
			return nil, fmt.Errorf("unexpected end of buffer data")
		}
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil

	case DataTypeArray:
		length, err := readIntVQL(reader)
		if err != nil {
			return nil, err
		}
		result := make([]interface{}, 0, length)
		for i := 0; i < length; i++ {
			el, err := deserialize(reader)
			if err != nil {
				return nil, err
			}
			result = append(result, el)
		}
		return result, nil

	case DataTypeObject:
		length, err := readIntVQL(reader)
		if err != nil {
			return nil, err
		}
		data := reader.Read(length)
		if data == nil {
			return nil, fmt.Errorf("unexpected end of object data")
		}
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal object: %w", err)
		}
		return result, nil

	case DataTypeInt:
		return readIntVQL(reader)

	default:
		return nil, fmt.Errorf("unknown data type: %d", dt)
	}
}

// serializeRequestHeader serializes a request header into the wire format.
func serializeRequestHeader(header interface{}) ([]byte, error) {
	writer := NewBufferWriter()
	serialize(writer, header)
	return writer.Buffer(), nil
}

// deserializeHeaderAndBody reads a full message (header + body) from raw bytes.
func deserializeMessage(data []byte) (interface{}, interface{}, error) {
	reader := NewBufferReader(data)
	header, err := deserialize(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize header: %w", err)
	}
	body, err := deserialize(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize body: %w", err)
	}
	return header, body, nil
}

// BuildPromiseRequest builds wire bytes for a promise request.
func BuildPromiseRequest(id int, channelName, name string, arg interface{}) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(RequestTypePromise), id, channelName, name})
	serialize(writer, arg)
	return writer.Buffer()
}

// BuildPromiseCancelRequest builds wire bytes for a promise cancel request.
func BuildPromiseCancelRequest(id int) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(RequestTypePromiseCancel), id})
	serialize(writer, nil)
	return writer.Buffer()
}

// BuildEventListenRequest builds wire bytes for an event listen request.
func BuildEventListenRequest(id int, channelName, name string, arg interface{}) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(RequestTypeEventListen), id, channelName, name})
	serialize(writer, arg)
	return writer.Buffer()
}

// BuildEventDisposeRequest builds wire bytes for an event dispose request.
func BuildEventDisposeRequest(id int) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(RequestTypeEventDispose), id})
	serialize(writer, nil)
	return writer.Buffer()
}

// BuildInitializeResponse builds wire bytes for the initialize response.
func BuildInitializeResponse() []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(ResponseTypeInitialize)})
	serialize(writer, nil)
	return writer.Buffer()
}

// BuildPromiseSuccessResponse builds wire bytes for a promise success response.
func BuildPromiseSuccessResponse(id int, data interface{}) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(ResponseTypePromiseSuccess), id})
	serialize(writer, data)
	return writer.Buffer()
}

// BuildPromiseErrorResponse builds wire bytes for a promise error response.
func BuildPromiseErrorResponse(id int, err error) []byte {
	errData := map[string]interface{}{
		"message": err.Error(),
		"name":    "Error",
	}
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(ResponseTypePromiseError), id})
	serialize(writer, errData)
	return writer.Buffer()
}

// BuildPromiseErrorObjResponse builds wire bytes for a promise error object response.
func BuildPromiseErrorObjResponse(id int, errObj interface{}) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(ResponseTypePromiseErrorObj), id})
	serialize(writer, errObj)
	return writer.Buffer()
}

// BuildEventFireResponse builds wire bytes for an event fire response.
func BuildEventFireResponse(id int, data interface{}) []byte {
	writer := NewBufferWriter()
	serialize(writer, []interface{}{int(ResponseTypeEventFire), id})
	serialize(writer, data)
	return writer.Buffer()
}

// ParseRequest parses a raw message into a structured request.
func ParseRequest(data []byte) (interface{}, error) {
	header, body, err := deserializeMessage(data)
	if err != nil {
		return nil, err
	}
	headerArr, ok := header.([]interface{})
	if !ok || len(headerArr) < 2 {
		return nil, fmt.Errorf("invalid request header")
	}

	reqType := RequestType(toInt(headerArr[0]))
	switch reqType {
	case RequestTypePromise:
		if len(headerArr) < 4 {
			return nil, fmt.Errorf("invalid promise request header")
		}
		return &RawPromiseRequest{
			Type:        reqType,
			ID:          toInt(headerArr[1]),
			ChannelName: toString(headerArr[2]),
			Name:        toString(headerArr[3]),
			Arg:         body,
		}, nil

	case RequestTypeEventListen:
		if len(headerArr) < 4 {
			return nil, fmt.Errorf("invalid event listen request header")
		}
		return &RawEventListenRequest{
			Type:        reqType,
			ID:          toInt(headerArr[1]),
			ChannelName: toString(headerArr[2]),
			Name:        toString(headerArr[3]),
			Arg:         body,
		}, nil

	case RequestTypePromiseCancel:
		return &RawPromiseCancelRequest{
			Type: reqType,
			ID:   toInt(headerArr[1]),
		}, nil

	case RequestTypeEventDispose:
		return &RawEventDisposeRequest{
			Type: reqType,
			ID:   toInt(headerArr[1]),
		}, nil

	default:
		return nil, fmt.Errorf("unknown request type: %d", reqType)
	}
}

// ParseResponse parses a raw message into a structured response.
func ParseResponse(data []byte) (interface{}, error) {
	header, body, err := deserializeMessage(data)
	if err != nil {
		return nil, err
	}
	headerArr, ok := header.([]interface{})
	if !ok || len(headerArr) < 1 {
		return nil, fmt.Errorf("invalid response header")
	}

	respType := ResponseType(toInt(headerArr[0]))
	switch respType {
	case ResponseTypeInitialize:
		return &RawInitializeResponse{Type: respType}, nil

	case ResponseTypePromiseSuccess:
		return &RawPromiseSuccessResponse{Type: respType, ID: toInt(headerArr[1]), Data: body}, nil

	case ResponseTypePromiseError:
		return &RawPromiseErrorResponse{Type: respType, ID: toInt(headerArr[1])}, nil

	case ResponseTypePromiseErrorObj:
		return &RawPromiseErrorObjResponse{Type: respType, ID: toInt(headerArr[1]), Data: body}, nil

	case ResponseTypeEventFire:
		return &RawEventFireResponse{Type: respType, ID: toInt(headerArr[1]), Data: body}, nil

	default:
		return nil, fmt.Errorf("unknown response type: %d", respType)
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}
	return 0
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
