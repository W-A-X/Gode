/*
protocol implements the VS Code extension host RPC protocol.

This mirrors the channel-based RPC defined in:
  src/vs/base/parts/ipc/common/ipc.ts

The RPC messages are the Data payload inside PersistentProtocol frames
(handled by the internal/ipc package). Each RPC message is JSON-encoded.

Request types:
  Promise       = 100  {type, id, channelName, name, arg}
  PromiseCancel = 101  {type, id}
  EventListen   = 102  {type, id, channelName, name, arg}
  EventDispose  = 103  {type, id}

Response types:
  Initialize       = 200
  PromiseSuccess   = 201  {type, id, data}
  PromiseError     = 202  {type, id, data: {message, name, stack}}
  PromiseErrorObj  = 203  {type, id, data}
  EventFire        = 204  {type, id, data}
*/

package protocol

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/microsoft/gode/internal/ipc"
)

const (
	// Request types
	ReqPromise       int = 100
	ReqPromiseCancel int = 101
	ReqEventListen   int = 102
	ReqEventDispose  int = 103

	// Response types
	RespInitialize      int = 200
	RespPromiseSuccess  int = 201
	RespPromiseError    int = 202
	RespPromiseErrorObj int = 203
	RespEventFire       int = 204
)

// RawRequest is the JSON wire format for requests.
type RawRequest struct {
	Type        int           `json:"type"`
	ID          uint32        `json:"id"`
	ChannelName string        `json:"channelName,omitempty"`
	Name        string        `json:"name,omitempty"`
	Arg         interface{}   `json:"arg,omitempty"`
}

// RawResponse is the JSON wire format for responses.
type RawResponse struct {
	Type int            `json:"type"`
	ID   uint32         `json:"id"`
	Data interface{}    `json:"data,omitempty"`
}

// RawError is the error data format for PromiseError responses.
type RawError struct {
	Message string   `json:"message"`
	Name    string   `json:"name"`
	Stack   []string `json:"stack,omitempty"`
}

// ChannelHandler handles RPC messages for a specific channel.
// This mirrors the ServerChannel interface from ipc.ts.
type ChannelHandler interface {
	// Call handles a Promise request (method call)
	Call(name string, arg interface{}) (interface{}, error)
	// Listen handles an EventListen request
	Listen(name string, arg interface{}) (<-chan interface{}, error)
	// Dispose handles an EventDispose request
	Dispose(eventID uint32)
}

// Dispatcher routes RPC messages to channel handlers.
type Dispatcher struct {
	mu       sync.RWMutex
	channels map[string]ChannelHandler
	requests map[uint32]chan *RawResponse
	nextID   uint32
	replyFn  func(*ipc.ProtocolMessage) error
}

// NewDispatcher creates a new RPC dispatcher.
func NewDispatcher(replyFunc func(*ipc.ProtocolMessage) error) *Dispatcher {
	return &Dispatcher{
		channels: make(map[string]ChannelHandler),
		requests: make(map[uint32]chan *RawResponse),
		replyFn:  replyFunc,
	}
}

// RegisterChannel registers a handler for a named channel
// (e.g., "MainThreadCommands", "MainThreadWindow").
func (d *Dispatcher) RegisterChannel(name string, handler ChannelHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.channels[name] = handler
	log.Printf("[protocol] registered channel: %s", name)
}

// HandleRPCMessage processes an incoming RPC message embedded in a ProtocolMessage.
func (d *Dispatcher) HandleRPCMessage(msg *ipc.ProtocolMessage) {
	if msg.Type != ipc.MsgTypeRegular {
		return
	}

	// Try parsing as a request first
	var raw RawRequest
	if err := json.Unmarshal(msg.Data, &raw); err == nil && raw.Type >= ReqPromise {
		switch raw.Type {
		case ReqPromise:
			d.handlePromise(raw)
		case ReqPromiseCancel:
			// TODO: cancellation
		case ReqEventListen:
			d.handleEventListen(raw)
		case ReqEventDispose:
			d.handleEventDispose(raw)
		default:
			log.Printf("[protocol] unknown request type: %d", raw.Type)
		}
		return
	}

	// Otherwise, try parsing as a response
	var resp RawResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil || resp.Type < RespInitialize {
		log.Printf("[protocol] failed to unmarshal RPC message")
		return
	}
	d.handleResponse(&resp)
}

func (d *Dispatcher) handlePromise(req RawRequest) {
	d.mu.RLock()
	handler, ok := d.channels[req.ChannelName]
	d.mu.RUnlock()

	if !ok {
		d.sendError(req.ID, fmt.Sprintf("channel '%s' not found", req.ChannelName))
		return
	}

	result, err := handler.Call(req.Name, req.Arg)
	if err != nil {
		d.sendErrorObj(req.ID, err.Error())
		return
	}

	d.sendSuccess(req.ID, result)
}

func (d *Dispatcher) handleEventListen(req RawRequest) {
	d.mu.RLock()
	handler, ok := d.channels[req.ChannelName]
	d.mu.RUnlock()

	if !ok {
		d.sendError(req.ID, fmt.Sprintf("channel '%s' not found", req.ChannelName))
		return
	}

	eventCh, err := handler.Listen(req.Name, req.Arg)
	if err != nil {
		d.sendError(req.ID, err.Error())
		return
	}

	// Start listening for events
	go func() {
		for evt := range eventCh {
			d.sendEventFire(req.ID, evt)
		}
	}()

	d.sendSuccess(req.ID, nil)
}

func (d *Dispatcher) handleEventDispose(req RawRequest) {
	d.mu.RLock()
	handler, ok := d.channels[req.ChannelName]
	d.mu.RUnlock()

	if ok {
		handler.Dispose(req.ID)
	}
}

func (d *Dispatcher) handleResponse(resp *RawResponse) {
	d.mu.Lock()
	ch, ok := d.requests[resp.ID]
	d.mu.Unlock()

	if !ok {
		return
	}

	ch <- &RawResponse{Type: resp.Type, ID: resp.ID, Data: resp.Data}
	close(ch)

	d.mu.Lock()
	delete(d.requests, resp.ID)
	d.mu.Unlock()
}

// ReplyFn is a callback function for sending protocol messages back.
type ReplyFn func(*ipc.ProtocolMessage) error

func (d *Dispatcher) sendSuccess(id uint32, data interface{}) {
	resp := RawResponse{Type: RespPromiseSuccess, ID: id, Data: data}
	d.sendResponse(resp)
}

func (d *Dispatcher) sendError(id uint32, message string) {
	err := RawError{Message: message, Name: "Error"}
	resp := RawResponse{Type: RespPromiseError, ID: id, Data: err}
	d.sendResponse(resp)
}

func (d *Dispatcher) sendErrorObj(id uint32, message string) {
	resp := RawResponse{Type: RespPromiseErrorObj, ID: id, Data: message}
	d.sendResponse(resp)
}

func (d *Dispatcher) sendEventFire(id uint32, data interface{}) {
	resp := RawResponse{Type: RespEventFire, ID: id, Data: data}
	d.sendResponse(resp)
}

func (d *Dispatcher) sendResponse(resp RawResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[protocol] failed to marshal response: %v", err)
		return
	}

	msg := &ipc.ProtocolMessage{
		Type: ipc.MsgTypeRegular,
		Data: data,
	}

	if d.replyFn != nil {
		if err := d.replyFn(msg); err != nil {
			log.Printf("[protocol] failed to send response: %v", err)
		}
	}
}

// Call sends a request to a channel and waits for a response.
func (d *Dispatcher) Call(channel, method string, arg interface{}) (interface{}, error) {
	d.mu.Lock()
	id := d.nextID + 1
	d.nextID = id
	ch := make(chan *RawResponse, 1)
	d.requests[id] = ch
	d.mu.Unlock()

	req := RawRequest{
		Type:        ReqPromise,
		ID:          id,
		ChannelName: channel,
		Name:        method,
		Arg:         arg,
	}

	data, err := json.Marshal(req)
	if err != nil {
		d.mu.Lock()
		delete(d.requests, id)
		d.mu.Unlock()
		return nil, err
	}

	msg := &ipc.ProtocolMessage{
		Type: ipc.MsgTypeRegular,
		Data: data,
	}

	if err := d.replyFn(msg); err != nil {
		d.mu.Lock()
		delete(d.requests, id)
		d.mu.Unlock()
		return nil, err
	}

	resp := <-ch

	switch resp.Type {
	case RespPromiseSuccess:
		return resp.Data, nil
	case RespPromiseError, RespPromiseErrorObj:
		if e, ok := resp.Data.(*RawError); ok {
			return nil, fmt.Errorf("%s: %s", e.Name, e.Message)
		}
		return nil, fmt.Errorf("%v", resp.Data)
	default:
		return nil, fmt.Errorf("unexpected response type: %d", resp.Type)
	}
}
