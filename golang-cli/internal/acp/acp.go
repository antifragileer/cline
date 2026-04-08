// Package acp provides Agent Client Protocol (ACP) mode implementation.
// ACP mode allows Cline CLI to run as an agent that communicates with
// editors via JSON-RPC over stdio.
package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ProtocolVersion is the ACP protocol version
const ProtocolVersion = "1.0.0"

// JSONRPCVersion is the JSON-RPC version used by ACP
const JSONRPCVersion = "2.0"

// Message types
const (
	// Request methods
	MethodInitialize        = "initialize"
	MethodCreateSession     = "createSession"
	MethodCloseSession      = "closeSession"
	MethodSendMessage       = "sendMessage"
	MethodRequestPermission = "requestPermission"
	MethodExecuteTool       = "executeTool"
	MethodGetCapabilities   = "getCapabilities"

	// Notification methods
	MethodSessionUpdate = "sessionUpdate"
	MethodToolCall      = "toolCall"
	MethodLog           = "log"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC notification
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes
const (
	ErrParseError     = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternalError  = -32603
	ErrServerError    = -32000
)

// AgentCapabilities represents the capabilities of the agent
type AgentCapabilities struct {
	Tools             []ToolCapability `json:"tools,omitempty"`
	SupportsStreaming bool             `json:"supportsStreaming"`
	SupportsPlanning  bool             `json:"supportsPlanning"`
	SupportsImages    bool             `json:"supportsImages"`
}

// ToolCapability represents a tool the agent can use
type ToolCapability struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// ClientCapabilities represents the client's capabilities
type ClientCapabilities struct {
	SupportsStreaming bool `json:"supportsStreaming"`
	SupportsFiles     bool `json:"supportsFiles"`
	SupportsTerminal  bool `json:"supportsTerminal"`
	SupportsBrowser   bool `json:"supportsBrowser"`
}

// InitializeRequest represents an initialize request
type InitializeRequest struct {
	ClientInfo         ClientInfo         `json:"clientInfo"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities,omitempty"`
}

// ClientInfo represents information about the client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult represents the result of initialize
type InitializeResult struct {
	ServerInfo        AgentInfo         `json:"serverInfo"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
	ProtocolVersion   string            `json:"protocolVersion"`
}

// AgentInfo represents information about the agent
type AgentInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Session represents an ACP session
type Session struct {
	ID        string                 `json:"id"`
	State     SessionState           `json:"state"`
	CreatedAt int64                  `json:"createdAt"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// SessionState represents the state of a session
type SessionState string

const (
	SessionStateActive  SessionState = "active"
	SessionStatePaused  SessionState = "paused"
	SessionStateClosing SessionState = "closing"
	SessionStateClosed  SessionState = "closed"
)

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	InitialContext map[string]interface{} `json:"initialContext,omitempty"`
	Mode           string                 `json:"mode,omitempty"` // "act" or "plan"
}

// CreateSessionResult represents the result of createSession
type CreateSessionResult struct {
	SessionID string       `json:"sessionId"`
	State     SessionState `json:"state"`
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
	Role      string `json:"role,omitempty"` // "user" or "system"
}

// SendMessageResult represents the result of sendMessage
type SendMessageResult struct {
	MessageID string `json:"messageId"`
}

// SessionUpdate represents a session update notification
type SessionUpdate struct {
	SessionID string                 `json:"sessionId"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// PermissionRequest represents a permission request
type PermissionRequest struct {
	SessionID   string   `json:"sessionId"`
	ToolCall    ToolCall `json:"toolCall"`
	Description string   `json:"description,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// PermissionResponse represents a permission response
type PermissionResponse struct {
	Approved bool   `json:"approved"`
	Option   string `json:"option,omitempty"` // "always", "once", "never"
}

// ExecuteToolRequest represents a request to execute a tool
type ExecuteToolRequest struct {
	SessionID string                 `json:"sessionId"`
	ToolCall  ToolCall               `json:"toolCall"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// ExecuteToolResult represents the result of tool execution
type ExecuteToolResult struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// Handler handles ACP protocol messages
type Handler interface {
	// Initialize is called when the client initializes the connection
	Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResult, error)

	// CreateSession creates a new session
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResult, error)

	// CloseSession closes a session
	CloseSession(ctx context.Context, sessionID string) error

	// SendMessage sends a message in a session
	SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResult, error)

	// RequestPermission requests permission for a tool call
	RequestPermission(ctx context.Context, req *PermissionRequest) (*PermissionResponse, error)

	// ExecuteTool executes a tool
	ExecuteTool(ctx context.Context, req *ExecuteToolRequest) (*ExecuteToolResult, error)

	// GetCapabilities returns agent capabilities
	GetCapabilities(ctx context.Context) (*AgentCapabilities, error)

	// Shutdown is called when the connection is closing
	Shutdown(ctx context.Context) error
}

// Server represents an ACP server
type Server struct {
	handler   Handler
	reader    *bufio.Reader
	writer    io.Writer
	mu        sync.Mutex
	nextID    int
	sessions  map[string]*Session
	sessionMu sync.RWMutex
	shutdown  chan struct{}
	wg        sync.WaitGroup
}

// ServerOptions contains options for creating a server
type ServerOptions struct {
	Handler Handler
	Reader  io.Reader
	Writer  io.Writer
}

// NewServer creates a new ACP server
func NewServer(opts ServerOptions) *Server {
	if opts.Reader == nil {
		opts.Reader = os.Stdin
	}
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	return &Server{
		handler:  opts.Handler,
		reader:   bufio.NewReader(opts.Reader),
		writer:   opts.Writer,
		nextID:   1,
		sessions: make(map[string]*Session),
		shutdown: make(chan struct{}),
	}
}

// Run starts the ACP server and handles messages
func (s *Server) Run(ctx context.Context) error {
	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start reading messages in a goroutine
	errCh := make(chan error, 1)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.readLoop(ctx); err != nil && err != io.EOF {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.shutdown:
		return nil
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, shutting down...\n", sig)
		return s.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

// readLoop reads and processes messages from stdin
func (s *Server) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.shutdown:
			return nil
		default:
		}

		// Read line
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Parse message
		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, ErrParseError, "Parse error", err.Error())
			continue
		}

		// Handle message
		s.wg.Add(1)
		go func(req JSONRPCRequest) {
			defer s.wg.Done()
			if err := s.handleRequest(ctx, req); err != nil {
				s.sendError(req.ID, ErrInternalError, "Internal error", err.Error())
			}
		}(req)
	}
}

// handleRequest handles a single request
func (s *Server) handleRequest(ctx context.Context, req JSONRPCRequest) error {
	// Validate JSON-RPC version
	if req.JSONRPC != JSONRPCVersion && req.JSONRPC != "" {
		return s.sendError(req.ID, ErrInvalidRequest, "Invalid JSON-RPC version", nil)
	}

	// Route to appropriate handler
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(ctx, req)
	case MethodCreateSession:
		return s.handleCreateSession(ctx, req)
	case MethodCloseSession:
		return s.handleCloseSession(ctx, req)
	case MethodSendMessage:
		return s.handleSendMessage(ctx, req)
	case MethodRequestPermission:
		return s.handleRequestPermission(ctx, req)
	case MethodExecuteTool:
		return s.handleExecuteTool(ctx, req)
	case MethodGetCapabilities:
		return s.handleGetCapabilities(ctx, req)
	default:
		return s.sendError(req.ID, ErrMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(ctx context.Context, req JSONRPCRequest) error {
	var initReq InitializeRequest
	if err := json.Unmarshal(req.Params, &initReq); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	result, err := s.handler.Initialize(ctx, &initReq)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// handleCreateSession handles the createSession request
func (s *Server) handleCreateSession(ctx context.Context, req JSONRPCRequest) error {
	var createReq CreateSessionRequest
	if err := json.Unmarshal(req.Params, &createReq); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	result, err := s.handler.CreateSession(ctx, &createReq)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	// Track the session
	s.sessionMu.Lock()
	s.sessions[result.SessionID] = &Session{
		ID:        result.SessionID,
		State:     SessionStateActive,
		CreatedAt: 0, // Will be set by handler
	}
	s.sessionMu.Unlock()

	return s.sendResult(req.ID, result)
}

// handleCloseSession handles the closeSession request
func (s *Server) handleCloseSession(ctx context.Context, req JSONRPCRequest) error {
	var params struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	if err := s.handler.CloseSession(ctx, params.SessionID); err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	// Remove from tracked sessions
	s.sessionMu.Lock()
	delete(s.sessions, params.SessionID)
	s.sessionMu.Unlock()

	return s.sendResult(req.ID, map[string]bool{"success": true})
}

// handleSendMessage handles the sendMessage request
func (s *Server) handleSendMessage(ctx context.Context, req JSONRPCRequest) error {
	var msgReq SendMessageRequest
	if err := json.Unmarshal(req.Params, &msgReq); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	result, err := s.handler.SendMessage(ctx, &msgReq)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// handleRequestPermission handles the requestPermission request
func (s *Server) handleRequestPermission(ctx context.Context, req JSONRPCRequest) error {
	var permReq PermissionRequest
	if err := json.Unmarshal(req.Params, &permReq); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	result, err := s.handler.RequestPermission(ctx, &permReq)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// handleExecuteTool handles the executeTool request
func (s *Server) handleExecuteTool(ctx context.Context, req JSONRPCRequest) error {
	var execReq ExecuteToolRequest
	if err := json.Unmarshal(req.Params, &execReq); err != nil {
		return s.sendError(req.ID, ErrInvalidParams, "Invalid params", err.Error())
	}

	result, err := s.handler.ExecuteTool(ctx, &execReq)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// handleGetCapabilities handles the getCapabilities request
func (s *Server) handleGetCapabilities(ctx context.Context, req JSONRPCRequest) error {
	result, err := s.handler.GetCapabilities(ctx)
	if err != nil {
		return s.sendError(req.ID, ErrInternalError, err.Error(), nil)
	}

	return s.sendResult(req.ID, result)
}

// SendSessionUpdate sends a session update notification
func (s *Server) SendSessionUpdate(update *SessionUpdate) error {
	notification := JSONRPCNotification{
		JSONRPC: JSONRPCVersion,
		Method:  MethodSessionUpdate,
	}

	params, err := json.Marshal(update)
	if err != nil {
		return err
	}
	notification.Params = params

	return s.sendNotification(&notification)
}

// SendToolCall sends a tool call notification
func (s *Server) SendToolCall(toolCall *ToolCall) error {
	notification := JSONRPCNotification{
		JSONRPC: JSONRPCVersion,
		Method:  MethodToolCall,
	}

	params, err := json.Marshal(toolCall)
	if err != nil {
		return err
	}
	notification.Params = params

	return s.sendNotification(&notification)
}

// sendResult sends a successful response
func (s *Server) sendResult(id interface{}, result interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return s.sendError(id, ErrInternalError, "Failed to marshal result", nil)
	}

	resp := JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}

	return s.writeMessage(&resp)
}

// sendError sends an error response
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) error {
	resp := JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	return s.writeMessage(&resp)
}

// sendNotification sends a notification
func (s *Server) sendNotification(notif *JSONRPCNotification) error {
	return s.writeMessage(notif)
}

// writeMessage writes a message to stdout
func (s *Server) writeMessage(msg interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	close(s.shutdown)

	// Shutdown handler
	if err := s.handler.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during handler shutdown: %v\n", err)
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetSession returns a session by ID
func (s *Server) GetSession(id string) (*Session, bool) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}

// ListSessions returns all active sessions
func (s *Server) ListSessions() []*Session {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// RedirectConsoleToStderr redirects all console output to stderr
// This is required in ACP mode because stdout is reserved for JSON-RPC
func RedirectConsoleToStderr() {
	// In Go, we can't easily redirect fmt.Print* functions
	// Instead, we document that all logging should use stderr
	// and provide helper functions in this package
}
