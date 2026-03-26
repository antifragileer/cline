// Package mcp provides Model Context Protocol (MCP) server management.
// This file implements tool execution via MCP servers.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents a content item in a tool result
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Executor handles tool execution via MCP servers
type Executor struct {
	process     *Process
	transport   *StdioTransport
	tools       map[string]*Tool
	toolsMu     sync.RWMutex
	requests    map[int]chan *JSONRPCResponse
	requestsMu  sync.RWMutex
	nextID      int
	requestsMu2 sync.Mutex
}

// StdioTransport handles JSON-RPC communication over stdin/stdout
type StdioTransport struct {
	process *Process
	encoder *json.Encoder
	decoder *json.Decoder
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC notification
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Notification represents a generic MCP notification
type Notification struct {
	Method string
	Params map[string]interface{}
}

const (
	// MCP Methods
	MethodInitialize   = "initialize"
	MethodToolsList    = "tools/list"
	MethodToolsCall    = "tools/call"
	MethodResourcesList = "resources/list"
	MethodResourcesRead = "resources/read"
	MethodPromptsList   = "prompts/list"
	MethodPromptsGet    = "prompts/get"
	MethodPing          = "ping"

	// JSON-RPC version
	JSONRPCVersion = "2.0"
)

// NewExecutor creates a new MCP executor for a process
func NewExecutor(process *Process) *Executor {
	return &Executor{
		process:  process,
		tools:    make(map[string]*Tool),
		requests: make(map[int]chan *JSONRPCResponse),
	}
}

// Connect establishes the JSON-RPC connection
func (e *Executor) Connect(ctx context.Context) error {
	// Wait for process to be running
	if !e.process.IsRunning() {
		return fmt.Errorf("process is not running")
	}

	// Create transport
	e.transport = &StdioTransport{
		process: e.process,
	}

	// Start reading responses/notifications
	go e.readLoop()

	// Send initialize request
	initReq := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      e.getNextID(),
		Method:  MethodInitialize,
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "cline-cli",
				"version": "1.0.0",
			},
		},
	}

	resp, err := e.sendRequest(ctx, initReq)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	return nil
}

// ListTools retrieves the list of available tools from the server
func (e *Executor) ListTools(ctx context.Context) ([]*Tool, error) {
	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      e.getNextID(),
		Method:  MethodToolsList,
		Params:  map[string]interface{}{},
	}

	resp, err := e.sendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list tools failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("list tools error: %s", resp.Error.Message)
	}

	// Parse result
	var result struct {
		Tools []*Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools: %w", err)
	}

	// Update internal tool cache
	e.toolsMu.Lock()
	for _, tool := range result.Tools {
		e.tools[tool.Name] = tool
	}
	e.toolsMu.Unlock()

	return result.Tools, nil
}

// ExecuteTool executes a tool on the MCP server
func (e *Executor) ExecuteTool(ctx context.Context, name string, arguments map[string]interface{}) (*ToolResult, error) {
	// Check if tool exists
	e.toolsMu.RLock()
	_, exists := e.tools[name]
	e.toolsMu.RUnlock()

	if !exists {
		// Try to refresh tools list
		if _, err := e.ListTools(ctx); err != nil {
			return nil, fmt.Errorf("failed to refresh tools: %w", err)
		}

		e.toolsMu.RLock()
		_, exists = e.tools[name]
		e.toolsMu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("tool '%s' not found", name)
		}
	}

	// Execute tool
	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      e.getNextID(),
		Method:  MethodToolsCall,
		Params: map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
	}

	resp, err := e.sendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	if resp.Error != nil {
		return &ToolResult{
			IsError: true,
			Content: []ToolContent{
				{Type: "text", Text: resp.Error.Message},
			},
		}, nil
	}

	// Parse result
	var result ToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		// Try to parse as plain text
		var textResult struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(resp.Result, &textResult); err == nil && textResult.Content != "" {
			result.Content = []ToolContent{
				{Type: "text", Text: textResult.Content},
			}
		} else {
			// Return raw JSON as text
			result.Content = []ToolContent{
				{Type: "text", Text: string(resp.Result)},
			}
		}
	}

	return &result, nil
}

// GetTool returns a tool by name
func (e *Executor) GetTool(name string) (*Tool, bool) {
	e.toolsMu.RLock()
	defer e.toolsMu.RUnlock()
	tool, exists := e.tools[name]
	return tool, exists
}

// IsToolAvailable checks if a tool is available
func (e *Executor) IsToolAvailable(name string) bool {
	_, exists := e.GetTool(name)
	return exists
}

// sendRequest sends a JSON-RPC request and waits for the response
func (e *Executor) sendRequest(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	// Create response channel
	respCh := make(chan *JSONRPCResponse, 1)
	e.requestsMu.Lock()
	e.requests[req.ID] = respCh
	e.requestsMu.Unlock()

	defer func() {
		e.requestsMu.Lock()
		delete(e.requests, req.ID)
		e.requestsMu.Unlock()
	}()

	// Send the request
	if err := e.writeRequest(req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// writeRequest writes a JSON-RPC request to the process stdin
func (e *Executor) writeRequest(req *JSONRPCRequest) error {
	if e.transport == nil {
		return fmt.Errorf("transport not connected")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write with newline
	data = append(data, '\n')
	
	return e.process.WriteStdin(data)
}

// readLoop continuously reads responses and notifications from the process
func (e *Executor) readLoop() {
	outputCh := e.process.SubscribeOutput()
	defer e.process.UnsubscribeOutput(outputCh)

	for line := range outputCh {
		// Only process stdout for JSON-RPC
		if line.Stream != "stdout" {
			continue
		}

		// Try to parse as JSON-RPC
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line.Content), &resp); err != nil {
			continue // Not valid JSON-RPC, skip
		}

		// Check if it's a response with an ID
		if resp.ID != 0 {
			e.requestsMu.RLock()
			ch, exists := e.requests[resp.ID]
			e.requestsMu.RUnlock()

			if exists {
				select {
				case ch <- &resp:
				default:
					// Channel is full or closed
				}
			}
			continue
		}

		// It's a notification
		var notif JSONRPCNotification
		if err := json.Unmarshal([]byte(line.Content), &notif); err == nil {
			e.handleNotification(&notif)
		}
	}
}

// handleNotification handles incoming notifications
func (e *Executor) handleNotification(notif *JSONRPCNotification) {
	// Handle specific notification types
	switch notif.Method {
	case "notifications/tools/list_changed":
		// Tools list changed, refresh
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		e.ListTools(ctx)
	}
}

// getNextID returns the next request ID
func (e *Executor) getNextID() int {
	e.requestsMu2.Lock()
	defer e.requestsMu2.Unlock()
	e.nextID++
	return e.nextID
}

// Close closes the executor and releases resources
func (e *Executor) Close() error {
	// Clear pending requests
	e.requestsMu.Lock()
	for id, ch := range e.requests {
		close(ch)
		delete(e.requests, id)
	}
	e.requestsMu.Unlock()

	return nil
}

// ExecutorManager manages executors for multiple processes
type ExecutorManager struct {
	executors map[string]*Executor
	mu        sync.RWMutex
}

// NewExecutorManager creates a new executor manager
func NewExecutorManager() *ExecutorManager {
	return &ExecutorManager{
		executors: make(map[string]*Executor),
	}
}

// AddExecutor adds an executor for a process
func (em *ExecutorManager) AddExecutor(name string, executor *Executor) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.executors[name] = executor
}

// RemoveExecutor removes an executor
func (em *ExecutorManager) RemoveExecutor(name string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if executor, exists := em.executors[name]; exists {
		executor.Close()
		delete(em.executors, name)
	}
}

// GetExecutor gets an executor by name
func (em *ExecutorManager) GetExecutor(name string) (*Executor, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	executor, exists := em.executors[name]
	return executor, exists
}

// ExecuteTool executes a tool on the specified server
func (em *ExecutorManager) ExecuteTool(ctx context.Context, serverName string, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	executor, exists := em.GetExecutor(serverName)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	return executor.ExecuteTool(ctx, toolName, arguments)
}

// ListTools lists tools from the specified server
func (em *ExecutorManager) ListTools(ctx context.Context, serverName string) ([]*Tool, error) {
	executor, exists := em.GetExecutor(serverName)
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	return executor.ListTools(ctx)
}