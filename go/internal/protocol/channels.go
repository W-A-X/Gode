/*
extension implements extension host protocol channels.

This mirrors the MainThread* channels defined in extHost.protocol.ts:
- MainThreadCommands: command registration/execution
- MainThreadWindow: window/showMessage
- MainThreadEditors: editor management
- MainThreadDocuments: document tracking
- MainThreadWorkspace: workspace/config
- MainThreadExtension: extension lifecycle

For MVP, we implement minimal versions of these channels.
*/

package protocol

import (
	"fmt"
	"log"

	"github.com/microsoft/gode/internal/editor"
)

// MainThreadCommandsHandler handles command-related RPC messages.
type MainThreadCommandsHandler struct {
	editorManager *editor.TextEditorManager
	dispatcher  *Dispatcher
}

// NewMainThreadCommandsHandler creates a new commands handler.
func NewMainThreadCommandsHandler(editorManager *editor.TextEditorManager, dispatcher *Dispatcher) *MainThreadCommandsHandler {
	return &MainThreadCommandsHandler{
		editorManager: editorManager,
		dispatcher:  dispatcher,
	}
}

// HandleRequest processes command requests.
func (h *MainThreadCommandsHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "commands.executeCommand":
		if len(args) < 1 {
			return nil, fmt.Errorf("executeCommand requires at least one argument")
		}
		command := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] executing command: %s", command)
		return nil, nil
	case "commands.registerCommand":
		// TODO: register command with extension host
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command method: %s", method)
	}
}

// HandleEvent processes command events.
func (h *MainThreadCommandsHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] command event: %s, args: %v", event, args)
}

// MainThreadWindowHandler handles window-related RPC messages.
type MainThreadWindowHandler struct {
	dispatcher *Dispatcher
}

// NewMainThreadWindowHandler creates a new window handler.
func NewMainThreadWindowHandler(dispatcher *Dispatcher) *MainThreadWindowHandler {
	return &MainThreadWindowHandler{dispatcher: dispatcher}
}

// HandleRequest processes window requests.
func (h *MainThreadWindowHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "window.showInformationMessage":
		if len(args) >= 1 {
			msg := fmt.Sprintf("%v", args[0])
			log.Printf("[protocol] showing info message: %s", msg)
			return nil, nil
		}
		return nil, fmt.Errorf("showInformationMessage requires message")
	case "window.showErrorMessage":
		if len(args) >= 1 {
			msg := fmt.Sprintf("%v", args[0])
			log.Printf("[protocol] showing error message: %s", msg)
			return nil, nil
		}
		return nil, fmt.Errorf("showErrorMessage requires message")
	case "window.showTextDocument":
		// TODO: show text document
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown window method: %s", method)
	}
}

// HandleEvent processes window events.
func (h *MainThreadWindowHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] window event: %s, args: %v", event, args)
}

// MainThreadEditorsHandler handles editor-related RPC messages.
type MainThreadEditorsHandler struct {
	editorManager *editor.TextEditorManager
	dispatcher  *Dispatcher
}

// NewMainThreadEditorsHandler creates a new editors handler.
func NewMainThreadEditorsHandler(editorManager *editor.TextEditorManager, dispatcher *Dispatcher) *MainThreadEditorsHandler {
	return &MainThreadEditorsHandler{
		editorManager: editorManager,
		dispatcher:  dispatcher,
	}
}

// HandleRequest processes editor requests.
func (h *MainThreadEditorsHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "editors.create":
		// TODO: create editor
		return nil, nil
	case "editors.revealRange":
		// TODO: reveal range
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown editor method: %s", method)
	}
}

// HandleEvent processes editor events.
func (h *MainThreadEditorsHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] editor event: %s, args: %v", event, args)
}

// MainThreadDocumentsHandler handles document-related RPC messages.
type MainThreadDocumentsHandler struct {
	dispatcher  *Dispatcher
}

// NewMainThreadDocumentsHandler creates a new documents handler.
func NewMainThreadDocumentsHandler(dispatcher *Dispatcher) *MainThreadDocumentsHandler {
	return &MainThreadDocumentsHandler{dispatcher: dispatcher}
}

// HandleRequest processes document requests.
func (h *MainThreadDocumentsHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "documents.open":
		if len(args) < 1 {
			return nil, fmt.Errorf("documents.open requires uri")
		}
		uri := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] opening document: %s", uri)
		return nil, nil
	case "documents.close":
		if len(args) < 1 {
			return nil, fmt.Errorf("documents.close requires uri")
		}
		uri := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] closing document: %s", uri)
		return nil, nil
	case "documents.get":
		if len(args) < 1 {
			return nil, fmt.Errorf("documents.get requires uri")
		}
		uri := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] getting document: %s", uri)
		return map[string]interface{}{}, nil
	default:
		return nil, fmt.Errorf("unknown document method: %s", method)
	}
}

// HandleEvent processes document events.
func (h *MainThreadDocumentsHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] document event: %s, args: %v", event, args)
}

// MainThreadWorkspaceHandler handles workspace-related RPC messages.
type MainThreadWorkspaceHandler struct {
	dispatcher *Dispatcher
}

// NewMainThreadWorkspaceHandler creates a new workspace handler.
func NewMainThreadWorkspaceHandler(dispatcher *Dispatcher) *MainThreadWorkspaceHandler {
	return &MainThreadWorkspaceHandler{dispatcher: dispatcher}
}

// HandleRequest processes workspace requests.
func (h *MainThreadWorkspaceHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "workspace.openTextDocument":
		if len(args) < 1 {
			return nil, fmt.Errorf("workspace.openTextDocument requires uri")
		}
		uri := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] opening text document: %s", uri)
		return nil, nil
	case "workspace.findFiles":
		// TODO: implement file search
		return []interface{}{}, nil
	case "workspace.getConfiguration":
		return map[string]interface{}{}, nil
	default:
		return nil, fmt.Errorf("unknown workspace method: %s", method)
	}
}

// HandleEvent processes workspace events.
func (h *MainThreadWorkspaceHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] workspace event: %s, args: %v", event, args)
}

// MainThreadExtensionHandler handles extension-related RPC messages.
type MainThreadExtensionHandler struct {
	dispatcher *Dispatcher
}

// NewMainThreadExtensionHandler creates a new extension handler.
func NewMainThreadExtensionHandler(dispatcher *Dispatcher) *MainThreadExtensionHandler {
	return &MainThreadExtensionHandler{dispatcher: dispatcher}
}

// HandleRequest processes extension requests.
func (h *MainThreadExtensionHandler) HandleRequest(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "extensions.getExtensionPath":
		if len(args) < 1 {
			return nil, fmt.Errorf("extensions.getExtensionPath requires extension id")
		}
		extID := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] getting extension path: %s", extID)
		return "/fake/extension/path", nil
	case "extensions.activateExtension":
		if len(args) < 1 {
			return nil, fmt.Errorf("extensions.activateExtension requires extension id")
		}
		extID := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] activating extension: %s", extID)
		return nil, nil
	case "extensions.getExtension":
		if len(args) < 1 {
			return nil, fmt.Errorf("extensions.getExtension requires extension id")
		}
		extID := fmt.Sprintf("%v", args[0])
		log.Printf("[protocol] getting extension: %s", extID)
		return map[string]interface{}{}, nil
	default:
		return nil, fmt.Errorf("unknown extension method: %s", method)
	}
}

// HandleEvent processes extension events.
func (h *MainThreadExtensionHandler) HandleEvent(event string, args []interface{}) {
	log.Printf("[protocol] extension event: %s, args: %v", event, args)
}

// RegisterAllChannels registers all protocol channels.
func RegisterAllChannels(dispatcher *Dispatcher, editorManager *editor.TextEditorManager) {
	dispatcher.RegisterChannel("MainThreadCommands", NewMainThreadCommandsHandler(editorManager, dispatcher))
	dispatcher.RegisterChannel("MainThreadWindow", NewMainThreadWindowHandler(dispatcher))
	dispatcher.RegisterChannel("MainThreadEditors", NewMainThreadEditorsHandler(editorManager, dispatcher))
	dispatcher.RegisterChannel("MainThreadDocuments", NewMainThreadDocumentsHandler(dispatcher))
	dispatcher.RegisterChannel("MainThreadWorkspace", NewMainThreadWorkspaceHandler(dispatcher))
	dispatcher.RegisterChannel("MainThreadExtension", NewMainThreadExtensionHandler(dispatcher))
}