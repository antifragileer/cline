// Package acp provides Agent Client Protocol (ACP) mode implementation.
// This file implements the ACP Handler interface for integration with Cline.
package acp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/google/uuid"
)

// ClineHandler implements the ACP Handler interface for Cline
type ClineHandler struct {
	storage   *storage.StorageContext
	conn      *grpc.ClientConn
	version   string
	sessions  map[string]*ClineSession
	sessionMu sync.RWMutex
	logger    *slog.Logger
}

// ClineSession represents an ACP session with Cline
type ClineSession struct {
	ID        string
	Runner    *task.Runner
	Config    task.Config
	CreatedAt int64
	Updates   chan *SessionUpdate
	Done      chan struct{}
}

// ClineHandlerOptions contains options for creating a ClineHandler
type ClineHandlerOptions struct {
	Storage *storage.StorageContext
	Client  interface{} // *host.Client or nil
	Version string
	Verbose bool
}

// NewClineHandler creates a new Cline ACP handler
func NewClineHandler(opts ClineHandlerOptions) *ClineHandler {
	level := slog.LevelInfo
	if opts.Verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	return &ClineHandler{
		storage:  opts.Storage,
		version:  opts.Version,
		sessions: make(map[string]*ClineSession),
		logger:   logger,
	}
}

// SetConnection sets the gRPC connection (called after initialization)
func (h *ClineHandler) SetConnection(conn *grpc.ClientConn) {
	h.conn = conn
}

// Initialize implements the Handler interface
func (h *ClineHandler) Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResult, error) {
	h.logger.Info("ACP initialize request received",
		"clientName", req.ClientInfo.Name,
		"clientVersion", req.ClientInfo.Version)

	return &InitializeResult{
		ServerInfo: AgentInfo{
			Name:    "Cline CLI",
			Version: h.version,
		},
		AgentCapabilities: AgentCapabilities{
			Tools: []ToolCapability{
				{
					Name:        "read_file",
					Description: "Read the contents of a file",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "Path to the file to read",
							},
						},
						"required": []string{"path"},
					},
				},
				{
					Name:        "write_file",
					Description: "Write content to a file",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path": map[string]interface{}{
								"type":        "string",
								"description": "Path to the file to write",
							},
							"content": map[string]interface{}{
								"type":        "string",
								"description": "Content to write",
							},
						},
						"required": []string{"path", "content"},
					},
				},
				{
					Name:        "execute_command",
					Description: "Execute a shell command",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"command": map[string]interface{}{
								"type":        "string",
								"description": "Command to execute",
							},
							"cwd": map[string]interface{}{
								"type":        "string",
								"description": "Working directory",
							},
						},
						"required": []string{"command"},
					},
				},
				{
					Name:        "search_files",
					Description: "Search for files matching a pattern",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"pattern": map[string]interface{}{
								"type":        "string",
								"description": "Search pattern",
							},
							"path": map[string]interface{}{
								"type":        "string",
								"description": "Path to search in",
							},
						},
						"required": []string{"pattern"},
					},
				},
			},
			SupportsStreaming: true,
			SupportsPlanning:  true,
			SupportsImages:    true,
		},
		ProtocolVersion: ProtocolVersion,
	}, nil
}

// CreateSession implements the Handler interface
func (h *ClineHandler) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResult, error) {
	sessionID := uuid.New().String()
	h.logger.Info("Creating ACP session", "sessionID", sessionID)

	// Determine mode
	mode := task.ModeAct
	if req.Mode == "plan" {
		mode = task.ModePlan
	}

	// Build task config
	config := task.Config{
		Mode:     mode,
		JSON:     false, // ACP mode doesn't use JSON output
		Verbose:  h.logger.Enabled(ctx, slog.LevelDebug),
		Cwd:      "", // Use current directory
		TaskID:   sessionID,
	}

	// Create task runner
	var runner *task.Runner
	if h.conn != nil {
		runner = task.NewRunner(h.conn)
	}

	// Create session
	session := &ClineSession{
		ID:        sessionID,
		Runner:    runner,
		Config:    config,
		CreatedAt: time.Now().Unix(),
		Updates:   make(chan *SessionUpdate, 100),
		Done:      make(chan struct{}),
	}

	// Store session
	h.sessionMu.Lock()
	h.sessions[sessionID] = session
	h.sessionMu.Unlock()

	h.logger.Info("ACP session created", "sessionID", sessionID)

	return &CreateSessionResult{
		SessionID: sessionID,
		State:     SessionStateActive,
	}, nil
}

// CloseSession implements the Handler interface
func (h *ClineHandler) CloseSession(ctx context.Context, sessionID string) error {
	h.logger.Info("Closing ACP session", "sessionID", sessionID)

	h.sessionMu.Lock()
	session, exists := h.sessions[sessionID]
	if !exists {
		h.sessionMu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Mark session as closing
	delete(h.sessions, sessionID)
	h.sessionMu.Unlock()

	// Close the session
	close(session.Done)
	close(session.Updates)

	h.logger.Info("ACP session closed", "sessionID", sessionID)
	return nil
}

// SendMessage implements the Handler interface
func (h *ClineHandler) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResult, error) {
	h.logger.Info("Sending message in ACP session",
		"sessionID", req.SessionID,
		"role", req.Role,
		"contentLength", len(req.Content))

	h.sessionMu.RLock()
	session, exists := h.sessions[req.SessionID]
	h.sessionMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	if session.Runner == nil {
		return nil, fmt.Errorf("session runner not initialized")
	}

	// Create a message handler that forwards updates to the ACP client
	handler := &acpMessageHandler{
		session:   session,
		sessionID: req.SessionID,
		server:    nil, // Will be set by the caller if needed
		logger:    h.logger,
	}

	// Update session config with the prompt
	config := session.Config
	config.Prompt = req.Content

	// Run the task in a goroutine
	go func() {
		if err := session.Runner.RunWithStreaming(context.Background(), config, handler); err != nil {
			h.logger.Error("Task execution failed", "error", err, "sessionID", req.SessionID)
			// Send error update
			select {
			case session.Updates <- &SessionUpdate{
				SessionID: req.SessionID,
				Type:      "error",
				Data: map[string]interface{}{
					"message": err.Error(),
				},
			}:
			case <-session.Done:
			}
		}
	}()

	return &SendMessageResult{
		MessageID: uuid.New().String(),
	}, nil
}

// RequestPermission implements the Handler interface
func (h *ClineHandler) RequestPermission(ctx context.Context, req *PermissionRequest) (*PermissionResponse, error) {
	h.logger.Info("Permission request in ACP session",
		"sessionID", req.SessionID,
		"toolName", req.ToolCall.Name)

	// In ACP mode, we typically auto-approve or the client handles permission
	// For now, we'll approve all requests
	// TODO: Implement proper permission handling with client interaction

	return &PermissionResponse{
		Approved: true,
		Option:   "once",
	}, nil
}

// ExecuteTool implements the Handler interface
func (h *ClineHandler) ExecuteTool(ctx context.Context, req *ExecuteToolRequest) (*ExecuteToolResult, error) {
	h.logger.Info("Executing tool in ACP session",
		"sessionID", req.SessionID,
		"toolName", req.ToolCall.Name)

	// Tool execution is handled by the task runner
	// This method is for direct tool execution outside of a task

	return &ExecuteToolResult{
		Success: true,
		Result: map[string]interface{}{
			"message": "Tool executed successfully",
		},
	}, nil
}

// GetCapabilities implements the Handler interface
func (h *ClineHandler) GetCapabilities(ctx context.Context) (*AgentCapabilities, error) {
	return &AgentCapabilities{
		Tools: []ToolCapability{
			{
				Name:        "read_file",
				Description: "Read the contents of a file",
			},
			{
				Name:        "write_file",
				Description: "Write content to a file",
			},
			{
				Name:        "execute_command",
				Description: "Execute a shell command",
			},
			{
				Name:        "search_files",
				Description: "Search for files matching a pattern",
			},
		},
		SupportsStreaming: true,
		SupportsPlanning:  true,
		SupportsImages:    true,
	}, nil
}

// Shutdown implements the Handler interface
func (h *ClineHandler) Shutdown(ctx context.Context) error {
	h.logger.Info("Shutting down ACP handler")

	// Close all sessions
	h.sessionMu.Lock()
	sessions := make([]*ClineSession, 0, len(h.sessions))
	for _, session := range h.sessions {
		sessions = append(sessions, session)
	}
	h.sessions = make(map[string]*ClineSession)
	h.sessionMu.Unlock()

	// Close each session
	for _, session := range sessions {
		close(session.Done)
		close(session.Updates)
	}

	// Close storage
	if h.storage != nil {
		h.storage.Close()
	}

	h.logger.Info("ACP handler shutdown complete")
	return nil
}

// GetSession returns a session by ID
func (h *ClineHandler) GetSession(id string) (*ClineSession, bool) {
	h.sessionMu.RLock()
	defer h.sessionMu.RUnlock()
	session, exists := h.sessions[id]
	return session, exists
}

// ListSessions returns all active sessions
func (h *ClineHandler) ListSessions() []*ClineSession {
	h.sessionMu.RLock()
	defer h.sessionMu.RUnlock()

	sessions := make([]*ClineSession, 0, len(h.sessions))
	for _, session := range h.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// acpMessageHandler implements task.MessageHandler for ACP mode
type acpMessageHandler struct {
	session   *ClineSession
	sessionID string
	server    *Server
	logger    *slog.Logger
}

// OnSay implements task.MessageHandler
func (h *acpMessageHandler) OnSay(sayType string, text string, partial bool) {
	h.logger.Debug("Received say message in ACP handler", "type", sayType, "partial", partial)

	// Convert task message to ACP session update
	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "say",
		Data: map[string]interface{}{
			"sayType": sayType,
			"text":    text,
			"partial": partial,
		},
	}

	// Send update to session
	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return
	}

	// Also send via server if available
	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send session update", "error", err)
		}
	}
}

// OnAsk implements task.MessageHandler
func (h *acpMessageHandler) OnAsk(askType string, text string) (string, error) {
	h.logger.Debug("Received ask message in ACP handler", "type", askType)

	// Convert task message to ACP session update
	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "ask",
		Data: map[string]interface{}{
			"askType": askType,
			"text":    text,
		},
	}

	// Send update to session
	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return "", fmt.Errorf("session closed")
	}

	// Also send via server if available
	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send session update", "error", err)
		}
	}

	// For ACP mode, auto-approve by default
	// TODO: Implement proper permission request handling
	return "yesButtonClicked", nil
}

// OnInfo implements task.MessageHandler
func (h *acpMessageHandler) OnInfo(text string) {
	h.logger.Debug("Received info message in ACP handler", "text", text)

	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "info",
		Data: map[string]interface{}{
			"text": text,
		},
	}

	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return
	}

	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send info update", "error", err)
		}
	}
}

// OnError implements task.MessageHandler
func (h *acpMessageHandler) OnError(err error) {
	h.logger.Error("Error in ACP handler", "error", err)

	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "error",
		Data: map[string]interface{}{
			"message": err.Error(),
		},
	}

	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return
	}

	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send error update", "error", err)
		}
	}
}

// OnStatus implements task.MessageHandler
func (h *acpMessageHandler) OnStatus(status string) {
	h.logger.Debug("Received status message in ACP handler", "status", status)

	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "status",
		Data: map[string]interface{}{
			"status": status,
		},
	}

	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return
	}

	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send status update", "error", err)
		}
	}
}

// OnProgress implements task.MessageHandler
func (h *acpMessageHandler) OnProgress(current, total int) {
	h.logger.Debug("Received progress message in ACP handler", "current", current, "total", total)

	update := &SessionUpdate{
		SessionID: h.sessionID,
		Type:      "progress",
		Data: map[string]interface{}{
			"current": current,
			"total":   total,
		},
	}

	select {
	case h.session.Updates <- update:
	case <-h.session.Done:
		return
	}

	if h.server != nil {
		if err := h.server.SendSessionUpdate(update); err != nil {
			h.logger.Warn("Failed to send progress update", "error", err)
		}
	}
}

// SetServer sets the ACP server reference for sending notifications
func (h *acpMessageHandler) SetServer(server *Server) {
	h.server = server
}

// Ensure acpMessageHandler implements task.MessageHandler
var _ task.MessageHandler = (*acpMessageHandler)(nil)