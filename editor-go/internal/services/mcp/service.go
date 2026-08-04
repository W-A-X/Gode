package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type MCPService struct {
	servers map[string]*MCPServer
	mu      sync.RWMutex
	client  *http.Client
}

type MCPServer struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Enabled bool              `json:"enabled"`
}

func NewMCPService() *MCPService {
	return &MCPService{
		servers: make(map[string]*MCPServer),
		client:  &http.Client{},
	}
}

func (mcp *MCPService) RegisterServer(name, url string, headers map[string]string) {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	mcp.servers[name] = &MCPServer{
		Name:    name,
		URL:     url,
		Headers: headers,
		Enabled: true,
	}
}

func (mcp *MCPService) UnregisterServer(name string) {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	delete(mcp.servers, name)
}

func (mcp *MCPService) ListServers() []*MCPServer {
	mcp.mu.RLock()
	defer mcp.mu.RUnlock()
	result := make([]*MCPServer, 0, len(mcp.servers))
	for _, s := range mcp.servers {
		result = append(result, s)
	}
	return result
}

func (mcp *MCPService) CallTool(serverName, toolName string, args map[string]interface{}) (interface{}, error) {
	mcp.mu.RLock()
	server, ok := mcp.servers[serverName]
	mcp.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("MCP server %s not found", serverName)
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", server.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range server.Headers {
		req.Header.Set(k, v)
	}

	resp, err := mcp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if error, ok := result["error"].(map[string]interface{}); ok {
		return nil, fmt.Errorf("MCP error: %v", error["message"])
	}

	return result["result"], nil
}

func (mcp *MCPService) ListTools(serverName string) ([]map[string]interface{}, error) {
	mcp.mu.RLock()
	server, ok := mcp.servers[serverName]
	mcp.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("MCP server %s not found", serverName)
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", server.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range server.Headers {
		req.Header.Set(k, v)
	}

	resp, err := mcp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if error, ok := result["error"].(map[string]interface{}); ok {
		return nil, fmt.Errorf("MCP error: %v", error["message"])
	}

	if tools, ok := result["result"].(map[string]interface{})["tools"].([]interface{}); ok {
		out := make([]map[string]interface{}, len(tools))
		for i, t := range tools {
			if tm, ok := t.(map[string]interface{}); ok {
				out[i] = tm
			}
		}
		return out, nil
	}

	return nil, nil
}