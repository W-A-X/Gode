package ipc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ChannelServer handles incoming IPC requests and dispatches them to registered channels.
type ChannelServer struct {
	protocol     IPCMessagePassingProtocol
	ctx          RequestContext
	channels     map[string]ServerChannel
	activeRequests map[int]context.CancelFunc
	pendingRequests map[string][]pendingRequest
	mu           sync.RWMutex
}

type pendingRequest struct {
	request    interface{}
	timer      *time.Timer
}

// NewChannelServer creates a new channel server.
func NewChannelServer(protocol IPCMessagePassingProtocol, ctx RequestContext) *ChannelServer {
	cs := &ChannelServer{
		protocol:        protocol,
		ctx:             ctx,
		channels:        make(map[string]ServerChannel),
		activeRequests:  make(map[int]context.CancelFunc),
		pendingRequests: make(map[string][]pendingRequest),
	}
	go cs.sendInitialize()
	go cs.readLoop()
	return cs
}

func (cs *ChannelServer) sendInitialize() {
	msg := BuildInitializeResponse()
	cs.protocol.Send(msg)
}

// RegisterChannel registers a channel handler for the given name.
func (cs *ChannelServer) RegisterChannel(name string, channel ServerChannel) {
	cs.mu.Lock()
	cs.channels[name] = channel
	cs.mu.Unlock()
	cs.flushPendingRequests(name)
}

func (cs *ChannelServer) getChannel(name string) (ServerChannel, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	ch, ok := cs.channels[name]
	return ch, ok
}

func (cs *ChannelServer) readLoop() {
	for msg := range cs.protocol.OnMessage() {
		if msg == nil {
			return
		}
		cs.handleMessage(msg)
	}
}

func (cs *ChannelServer) handleMessage(data []byte) {
	req, err := ParseRequest(data)
	if err != nil {
		log.Printf("[ChannelServer] failed to parse request: %v", err)
		return
	}

	switch r := req.(type) {
	case *RawPromiseRequest:
		cs.handlePromise(r)
	case *RawEventListenRequest:
		cs.handleEventListen(r)
	case *RawPromiseCancelRequest:
		cs.disposeActiveRequest(r.ID)
	case *RawEventDisposeRequest:
		cs.disposeActiveRequest(r.ID)
	}
}

func (cs *ChannelServer) handlePromise(req *RawPromiseRequest) {
	channel, ok := cs.getChannel(req.ChannelName)
	if !ok {
		cs.collectPendingRequest(req)
		return
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cs.mu.Lock()
	cs.activeRequests[req.ID] = cancel
	cs.mu.Unlock()

	go func() {
		defer func() {
			cs.mu.Lock()
			delete(cs.activeRequests, req.ID)
			cs.mu.Unlock()
			cancel()
		}()

		result, err := channel.Call(cs.ctx, req.Name, req.Arg, cancelCtx.Done())
		if err != nil {
			errMsg := BuildPromiseErrorResponse(req.ID, err)
			cs.protocol.Send(errMsg)
		} else {
			msg := BuildPromiseSuccessResponse(req.ID, result)
			cs.protocol.Send(msg)
		}
	}()
}

func (cs *ChannelServer) handleEventListen(req *RawEventListenRequest) {
	channel, ok := cs.getChannel(req.ChannelName)
	if !ok {
		cs.collectPendingRequest(req)
		return
	}

	ch, dispose := channel.Listen(cs.ctx, req.Name, req.Arg)
	cs.mu.Lock()
	cs.activeRequests[req.ID] = func() { dispose() }
	cs.mu.Unlock()

	go func() {
		for data := range ch {
			msg := BuildEventFireResponse(req.ID, data)
			cs.protocol.Send(msg)
		}
	}()
}

func (cs *ChannelServer) disposeActiveRequest(id int) {
	cs.mu.Lock()
	cancel, ok := cs.activeRequests[id]
	if ok {
		delete(cs.activeRequests, id)
	}
	cs.mu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
}

func (cs *ChannelServer) collectPendingRequest(req interface{}) {
	var channelName string
	switch r := req.(type) {
	case *RawPromiseRequest:
		channelName = r.ChannelName
	case *RawEventListenRequest:
		channelName = r.ChannelName
	}

	timer := time.AfterFunc(5*time.Second, func() {
		log.Printf("[ChannelServer] timed out waiting for channel: %s", channelName)
	})

	cs.mu.Lock()
	cs.pendingRequests[channelName] = append(cs.pendingRequests[channelName], pendingRequest{
		request: req,
		timer:   timer,
	})
	cs.mu.Unlock()
}

func (cs *ChannelServer) flushPendingRequests(channelName string) {
	cs.mu.Lock()
	requests := cs.pendingRequests[channelName]
	delete(cs.pendingRequests, channelName)
	cs.mu.Unlock()

	for _, pr := range requests {
		pr.timer.Stop()
		switch r := pr.request.(type) {
		case *RawPromiseRequest:
			cs.handlePromise(r)
		case *RawEventListenRequest:
			cs.handleEventListen(r)
		}
	}
}

// Stop shuts down the channel server.
func (cs *ChannelServer) Stop() {
	cs.mu.Lock()
	for id, cancel := range cs.activeRequests {
		cancel()
		delete(cs.activeRequests, id)
	}
	cs.mu.Unlock()
}

// ChannelClient handles outgoing IPC requests to a remote server.
type ChannelClient struct {
	protocol    IPCMessagePassingProtocol
	handlers    map[int]func(interface{})
	activeCount int
	mu          sync.RWMutex
}

// NewChannelClient creates a new channel client.
func NewChannelClient(protocol IPCMessagePassingProtocol) *ChannelClient {
	cc := &ChannelClient{
		protocol: protocol,
		handlers: make(map[int]func(interface{})),
	}
	go cc.readLoop()
	return cc
}

func (cc *ChannelClient) readLoop() {
	for msg := range cc.protocol.OnMessage() {
		if msg == nil {
			return
		}
		cc.handleMessage(msg)
	}
}

func (cc *ChannelClient) handleMessage(data []byte) {
	resp, err := ParseResponse(data)
	if err != nil {
		log.Printf("[ChannelClient] failed to parse response: %v", err)
		return
	}

	switch r := resp.(type) {
	case *RawInitializeResponse:
		// Connection initialized
	case *RawPromiseSuccessResponse:
		cc.mu.RLock()
		handler, ok := cc.handlers[r.ID]
		cc.mu.RUnlock()
		if ok {
			delete(cc.handlers, r.ID)
			handler(r)
		}

	case *RawPromiseErrorResponse:
		cc.mu.RLock()
		handler, ok := cc.handlers[r.ID]
		cc.mu.RUnlock()
		if ok {
			delete(cc.handlers, r.ID)
			handler(r)
		}

	case *RawPromiseErrorObjResponse:
		cc.mu.RLock()
		handler, ok := cc.handlers[r.ID]
		cc.mu.RUnlock()
		if ok {
			delete(cc.handlers, r.ID)
			handler(r)
		}

	case *RawEventFireResponse:
		cc.mu.RLock()
		handler, ok := cc.handlers[r.ID]
		cc.mu.RUnlock()
		if ok {
			handler(r)
		}
	}
}

// Call sends a promise request and waits for the response.
func (cc *ChannelClient) Call(channelName, command string, arg interface{}) (interface{}, error) {
	id := cc.nextID()
	msg := BuildPromiseRequest(id, channelName, command, arg)

	resultCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)

	cc.mu.Lock()
	cc.handlers[id] = func(resp interface{}) {
		switch r := resp.(type) {
		case *RawPromiseSuccessResponse:
			resultCh <- r.Data
		case *RawPromiseErrorResponse:
			errCh <- fmt.Errorf("%s", r.Data.Message)
		case *RawPromiseErrorObjResponse:
			errCh <- fmt.Errorf("%v", r.Data)
		}
	}
	cc.mu.Unlock()

	cc.protocol.Send(msg)

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(30 * time.Second):
		cc.mu.Lock()
		delete(cc.handlers, id)
		cc.mu.Unlock()
		return nil, fmt.Errorf("request timeout: %s.%s", channelName, command)
	}
}

// Listen subscribes to an event channel.
func (cc *ChannelClient) Listen(channelName, event string, arg interface{}) (<-chan interface{}, func()) {
	id := cc.nextID()
	msg := BuildEventListenRequest(id, channelName, event, arg)

	ch := make(chan interface{}, 100)

	cc.mu.Lock()
	cc.handlers[id] = func(resp interface{}) {
		if r, ok := resp.(*RawEventFireResponse); ok {
			select {
			case ch <- r.Data:
			default:
				// Drop event if channel is full
			}
		}
	}
	cc.mu.Unlock()

	cc.protocol.Send(msg)

	dispose := func() {
		cc.mu.Lock()
		delete(cc.handlers, id)
		cc.mu.Unlock()
		close(ch)
		cc.protocol.Send(BuildEventDisposeRequest(id))
	}

	return ch, dispose
}

func (cc *ChannelClient) nextID() int {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.activeCount++
	return cc.activeCount
}

// Stop shuts down the channel client.
func (cc *ChannelClient) Stop() {
	cc.mu.Lock()
	for id, handler := range cc.handlers {
		handler(nil)
		delete(cc.handlers, id)
	}
	cc.mu.Unlock()
}
