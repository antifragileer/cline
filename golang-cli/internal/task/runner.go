// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cline/cline/golang-cli/internal/audit"
	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/security"
)

// Mode represents the execution mode for a task
type Mode string

const (
	// ModeAct represents act mode (execute actions)
	ModeAct Mode = "act"
	// ModePlan represents plan mode (planning only)
	ModePlan Mode = "plan"
)

// Config holds all configuration options for a task
type Config struct {
	// Mode specifies whether to run in act or plan mode
	Mode Mode
	// Yolo enables auto-approval without confirmation
	Yolo bool
	// Timeout is the maximum duration for task execution
	Timeout time.Duration
	// Model specifies the model to use for the task
	Model string
	// Images is a list of image file paths to attach
	Images []string
	// Verbose enables verbose output
	Verbose bool
	// Cwd is the current working directory for the task
	Cwd string
	// Thinking enables thinking mode
	Thinking bool
	// ThinkingBudget specifies the thinking budget in tokens
	ThinkingBudget int
	// JSON enables JSON output format
	JSON bool
	// TaskID specifies a task ID to resume or reference
	TaskID string
	// Prompt is the task prompt/message
	Prompt string
	// AutoApproveAll enables auto-approve all actions
	AutoApproveAll bool
	// ReasoningEffort specifies the reasoning effort level
	ReasoningEffort string
	// MaxConsecutiveMistakes is the maximum consecutive mistakes
	MaxConsecutiveMistakes int
	// DoubleCheckCompletion rejects first completion attempt
	DoubleCheckCompletion bool
	// AutoCondense enables AI-powered context compaction
	AutoCondense bool
	// HooksDir is the path to additional hooks directory
	HooksDir string
}

// Runner executes tasks via gRPC
type Runner struct {
	taskClient cline.TaskServiceClient
	stateClient cline.StateServiceClient
	uiClient   cline.UiServiceClient
	conn       *grpc.ClientConn
	
	// Task state
	taskID       string
	isRunning    bool
	mu           sync.Mutex
	completionCh chan struct{}
	errorCh      chan error
	
	// Security and audit
	auditLogger *audit.Logger
	securityValidator *security.CommandValidator
}

// NewRunner creates a new task runner
func NewRunner(conn *grpc.ClientConn) *Runner {
	return &Runner{
		taskClient:   cline.NewTaskServiceClient(conn),
		stateClient:  cline.NewStateServiceClient(conn),
		uiClient:     cline.NewUiServiceClient(conn),
		conn:         conn,
		completionCh: make(chan struct{}),
		errorCh:      make(chan error, 1),
	}
}

// Run executes a task with the given configuration
func (r *Runner) Run(ctx context.Context, config Config, handler MessageHandler) error {
	// Validate configuration
	if config.Prompt == "" && config.TaskID == "" {
		return fmt.Errorf("task prompt required (or use TaskID to resume)")
	}

	// Initialize audit logger
	if err := r.initAuditLogger(); err != nil && config.Verbose {
		handler.OnInfo(fmt.Sprintf("Warning: failed to initialize audit logger: %v", err))
	}
	
	// Initialize security validator from environment
	if err := r.initSecurityValidator(); err != nil && config.Verbose {
		handler.OnInfo(fmt.Sprintf("Warning: failed to initialize security validator: %v", err))
	}
	
	// Log task start
	if r.auditLogger != nil {
		r.auditLogger.Log(&audit.Event{
			EventType:   audit.EventConfigurationChange,
			TaskID:      config.TaskID,
			SessionID:   r.taskID,
			Message:     "Task started",
			Details: map[string]interface{}{
				"mode":   string(config.Mode),
				"yolo":   config.Yolo,
				"model":  config.Model,
				"prompt": config.Prompt,
			},
		})
	}

	// Prepare images
	var imageData []string
	for _, imgPath := range config.Images {
		data, err := loadImageData(imgPath)
		if err != nil {
			return fmt.Errorf("failed to load image %s: %w", imgPath, err)
		}
		imageData = append(imageData, data)
	}

	// Create or resume the task
	var taskID string
	var err error
	
	if config.TaskID != "" {
		// Resume existing task
		taskID, err = r.resumeTask(ctx, config.TaskID, config.Prompt, imageData)
		if err != nil {
			return fmt.Errorf("failed to resume task: %w", err)
		}
	} else {
		// Create new task
		taskID, err = r.createTask(ctx, config, imageData)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
	}
	
	r.taskID = taskID
	
	if config.Verbose {
		handler.OnInfo(fmt.Sprintf("Task created: %s", taskID))
	}

	// Subscribe to state updates and process messages
	return r.subscribeAndProcessState(ctx, config, handler)
}

// initAuditLogger initializes the audit logger
func (r *Runner) initAuditLogger() error {
	config := audit.DefaultLoggerConfig()
	logger, err := audit.NewLogger(config)
	if err != nil {
		return err
	}
	r.auditLogger = logger
	return nil
}

// initSecurityValidator initializes the security validator from environment
func (r *Runner) initSecurityValidator() error {
	validator, err := security.LoadFromEnv()
	if err != nil {
		return err
	}
	r.securityValidator = validator
	return nil
}

// Close cleans up resources
func (r *Runner) Close() error {
	if r.auditLogger != nil {
		r.auditLogger.Close()
	}
	if r.securityValidator != nil {
		r.securityValidator.Close()
	}
	return nil
}

// RunWithStreaming executes a task and streams messages to the handler
func (r *Runner) RunWithStreaming(ctx context.Context, config Config, handler MessageHandler) error {
	return r.Run(ctx, config, handler)
}

// createTask creates a new task via gRPC
func (r *Runner) createTask(ctx context.Context, config Config, imageData []string) (string, error) {
	// Build settings based on configuration
	settings := buildSettingsFromConfig(config)
	
	req := &cline.NewTaskRequest{
		Metadata:     &cline.Metadata{},
		Text:         config.Prompt,
		Images:       imageData,
		TaskSettings: settings,
	}

	resp, err := r.taskClient.NewTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create new task: %w", err)
	}

	return resp.GetValue(), nil
}

// resumeTask resumes an existing task via gRPC
func (r *Runner) resumeTask(ctx context.Context, taskID string, prompt string, imageData []string) (string, error) {
	// First, show the task to load it
	_, err := r.taskClient.ShowTaskWithId(ctx, &cline.StringRequest{Value: taskID})
	if err != nil {
		return "", fmt.Errorf("failed to show task: %w", err)
	}
	
	// If there's a prompt to send, we'll handle it after subscription starts
	if prompt != "" {
		// Send the prompt as an ask response
		_, err := r.taskClient.AskResponse(ctx, &cline.AskResponseRequest{
			Metadata:     &cline.Metadata{},
			ResponseType: "messageResponse",
			Text:         prompt,
			Images:       imageData,
		})
		if err != nil {
			return "", fmt.Errorf("failed to send prompt to resumed task: %w", err)
		}
	}
	
	return taskID, nil
}

// buildSettingsFromConfig builds Settings from Config
func buildSettingsFromConfig(config Config) *cline.Settings {
	settings := &cline.Settings{}
	
	// Set mode
	mode := cline.PlanActMode_ACT
	if config.Mode == ModePlan {
		mode = cline.PlanActMode_PLAN
	}
	settings.Mode = &mode
	
	// Set yolo mode
	if config.Yolo {
		yoloEnabled := true
		settings.YoloModeToggled = &yoloEnabled
	}
	
	// Set auto-approve all
	if config.AutoApproveAll {
		autoApproveEnabled := true
		settings.AutoApproveAllToggled = &autoApproveEnabled
	}
	
	// Set thinking budget
	if config.Thinking && config.ThinkingBudget > 0 {
		actTokens := int64(config.ThinkingBudget)
		planTokens := int64(config.ThinkingBudget)
		settings.ActModeThinkingBudgetTokens = &actTokens
		settings.PlanModeThinkingBudgetTokens = &planTokens
	}
	
	// Set reasoning effort
	if config.ReasoningEffort != "" {
		effort := config.ReasoningEffort
		settings.ActModeReasoningEffort = &effort
		settings.PlanModeReasoningEffort = &effort
	}
	
	// Set max consecutive mistakes
	if config.MaxConsecutiveMistakes > 0 {
		maxMistakes := int32(config.MaxConsecutiveMistakes)
		settings.MaxConsecutiveMistakes = &maxMistakes
	}
	
	// Set double check completion
	if config.DoubleCheckCompletion {
		enabled := true
		settings.DoubleCheckCompletionEnabled = &enabled
	}
	
	// Set auto condense
	if config.AutoCondense {
		enabled := true
		settings.UseAutoCondense = &enabled
	}
	
	return settings
}

// subscribeAndProcessState subscribes to state updates and processes messages
func (r *Runner) subscribeAndProcessState(ctx context.Context, config Config, handler MessageHandler) error {
	// Subscribe to state updates
	stream, err := r.stateClient.SubscribeToState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to state: %w", err)
	}
	
	// Track processed messages to avoid duplicates
	processedMessages := make(map[int64]bool)
	var mu sync.Mutex
	
	// Track completion
	var completed bool
	var completionResult string
	
	// Process messages from the stream
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		state, err := stream.Recv()
		if err == io.EOF {
			// Stream closed normally
			if completed {
				handler.OnSay("completion_result", completionResult, false)
				return nil
			}
			return nil
		}
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Canceled {
				// Context was cancelled
				return ctx.Err()
			}
			return fmt.Errorf("error receiving state: %w", err)
		}
		
		// Parse state JSON
		var stateData map[string]interface{}
		if err := json.Unmarshal([]byte(state.StateJson), &stateData); err != nil {
			if config.Verbose {
				handler.OnInfo(fmt.Sprintf("Failed to parse state: %v", err))
			}
			continue
		}
		
		// Extract messages from state
		messages, ok := stateData["clineMessages"].([]interface{})
		if !ok {
			continue
		}
		
		// Process each message
		for _, msgData := range messages {
			msg, ok := msgData.(map[string]interface{})
			if !ok {
				continue
			}
			
			// Get message timestamp
			tsFloat, ok := msg["ts"].(float64)
			if !ok {
				continue
			}
			ts := int64(tsFloat)
			
			// Skip already processed messages
			mu.Lock()
			if processedMessages[ts] {
				mu.Unlock()
				continue
			}
			mu.Unlock()
			
			// Process the message
			if err := r.processMessage(ctx, msg, ts, config, handler); err != nil {
				return err
			}
			
			mu.Lock()
			processedMessages[ts] = true
			mu.Unlock()
			
			// Check for completion
			msgType, _ := msg["type"].(string)
			say, _ := msg["say"].(string)
			ask, _ := msg["ask"].(string)
			text, _ := msg["text"].(string)
			
			if (msgType == "say" && say == "completion_result") || 
			   (msgType == "ask" && ask == "completion_result") {
				completed = true
				completionResult = text
				
				// Don't return yet - let the user see the completion
				// The handler will receive this message via OnSay
			}
			
			// Check for errors
			if msgType == "say" && say == "error" {
				return fmt.Errorf("task error: %s", text)
			}
			if msgType == "ask" && ask == "api_req_failed" {
				return fmt.Errorf("API request failed: %s", text)
			}
		}
	}
}

// processMessage processes a single message
func (r *Runner) processMessage(ctx context.Context, msg map[string]interface{}, ts int64, config Config, handler MessageHandler) error {
	msgType, _ := msg["type"].(string)
	partial, _ := msg["partial"].(bool)
	text, _ := msg["text"].(string)
	
	switch msgType {
	case "say":
		say, _ := msg["say"].(string)
		return r.processSayMessage(ctx, say, text, partial, ts, config, handler)
		
	case "ask":
		ask, _ := msg["ask"].(string)
		return r.processAskMessage(ctx, ask, text, msg, ts, config, handler)
	}
	
	return nil
}

// processSayMessage processes a SAY message
func (r *Runner) processSayMessage(ctx context.Context, say, text string, partial bool, ts int64, config Config, handler MessageHandler) error {
	// Skip partial messages in non-verbose mode
	if partial && !config.Verbose {
		return nil
	}
	
	// Map say type to handler type
	sayType := mapSayType(say)
	
	// Call handler
	handler.OnSay(sayType, text, partial)
	
	// Audit log certain events
	if !partial && r.auditLogger != nil {
		switch say {
		case "api_req_started":
			r.auditLogger.Log(&audit.Event{
				EventType: audit.EventAPICall,
				TaskID:    r.taskID,
				Message:   "API request started",
				Details: map[string]interface{}{
					"status": "started",
				},
			})
			
		case "api_req_finished":
			r.auditLogger.Log(&audit.Event{
				EventType: audit.EventAPICall,
				TaskID:    r.taskID,
				Message:   "API request finished",
				Details: map[string]interface{}{
					"status": "finished",
				},
			})
			
		case "command":
			r.auditLogger.Log(&audit.Event{
				EventType: audit.EventCommandExecution,
				TaskID:    r.taskID,
				Message:   "Command executed",
				Details: map[string]interface{}{
					"command": text,
				},
			})
			
		case "checkpoint_created":
			r.auditLogger.Log(&audit.Event{
				EventType: audit.EventConfigurationChange,
				TaskID:    r.taskID,
				Message:   "Checkpoint created",
				Details: map[string]interface{}{
					"checkpoint": text,
				},
			})
		}
	}
	
	return nil
}

// processAskMessage processes an ASK message (requires user response)
func (r *Runner) processAskMessage(ctx context.Context, ask, text string, msg map[string]interface{}, ts int64, config Config, handler MessageHandler) error {
	// Check for auto-approval
	if shouldAutoApprove(ask, config) {
		// Validate command against security permissions before auto-approving
		if ask == "command" && r.securityValidator != nil {
			allowed, reason := r.validateCommand(text)
			if !allowed {
				// Log the rejection
				if r.auditLogger != nil {
					r.auditLogger.Log(&audit.Event{
						EventType: audit.EventToolRejection,
						TaskID:    r.taskID,
						Message:   fmt.Sprintf("Command rejected by security policy: %s", reason),
						Details: map[string]interface{}{
							"command": text,
							"reason":  reason,
						},
					})
				}
				
				// Send rejection response
				if err := r.sendAskResponse(ctx, "noButtonClicked", fmt.Sprintf("Command blocked by security policy: %s", reason), nil, nil); err != nil {
					return fmt.Errorf("failed to send security rejection: %w", err)
				}
				handler.OnInfo(fmt.Sprintf("Security: Command blocked - %s", reason))
				return nil
			}
		}
		
		if err := r.sendAskResponse(ctx, "yesButtonClicked", "", nil, nil); err != nil {
			return fmt.Errorf("failed to send auto-approval: %w", err)
		}
		
		// Log the approval
		if r.auditLogger != nil {
			r.auditLogger.Log(&audit.Event{
				EventType: audit.EventToolApproval,
				TaskID:    r.taskID,
				Message:   fmt.Sprintf("Auto-approved: %s", ask),
				Details: map[string]interface{}{
					"ask_type": ask,
					"text":     text,
				},
			})
		}
		
		if config.Verbose {
			handler.OnInfo(fmt.Sprintf("Auto-approved: %s", ask))
		}
		return nil
	}
	
	// Validate command against security permissions (even in interactive mode)
	if ask == "command" && r.securityValidator != nil {
		allowed, reason := r.validateCommand(text)
		if !allowed {
			// Log the rejection
			if r.auditLogger != nil {
				r.auditLogger.Log(&audit.Event{
					EventType: audit.EventToolRejection,
					TaskID:    r.taskID,
					Message:   fmt.Sprintf("Command rejected by security policy: %s", reason),
					Details: map[string]interface{}{
						"command": text,
						"reason":  reason,
					},
				})
			}
			
			// Send rejection response
			if err := r.sendAskResponse(ctx, "noButtonClicked", fmt.Sprintf("Command blocked by security policy: %s", reason), nil, nil); err != nil {
				return fmt.Errorf("failed to send security rejection: %w", err)
			}
			handler.OnInfo(fmt.Sprintf("Security: Command blocked - %s", reason))
			return nil
		}
	}
	
	// Get user response through handler
	response, err := handler.OnAsk(ask, text)
	if err != nil {
		return err
	}
	
	// Log the user response
	if r.auditLogger != nil {
		eventType := audit.EventToolApproval
		if response == "noButtonClicked" {
			eventType = audit.EventToolRejection
		}
		r.auditLogger.Log(&audit.Event{
			EventType: eventType,
			TaskID:    r.taskID,
			Message:   fmt.Sprintf("User response to %s: %s", ask, response),
			Details: map[string]interface{}{
				"ask_type": ask,
				"text":     text,
				"response": response,
			},
		})
	}
	
	// Send response back to core
	if err := r.sendAskResponse(ctx, response, "", nil, nil); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	
	return nil
}

// validateCommand validates a command against security permissions
func (r *Runner) validateCommand(text string) (bool, string) {
	if r.securityValidator == nil {
		return true, "no security validator configured"
	}
	
	// Extract command from text (first line)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return true, "empty command"
	}
	
	command := strings.TrimSpace(lines[0])
	if command == "" {
		return true, "empty command"
	}
	
	result := r.securityValidator.Validate(nil, command)
	return result.Allowed, result.Reason
}

// sendAskResponse sends a response to an ask message
func (r *Runner) sendAskResponse(ctx context.Context, responseType, text string, images, files []string) error {
	req := &cline.AskResponseRequest{
		Metadata:     &cline.Metadata{},
		ResponseType: responseType,
		Text:         text,
		Images:       images,
		Files:        files,
	}
	
	_, err := r.taskClient.AskResponse(ctx, req)
	return err
}

// mapSayType maps internal say type to handler type
func mapSayType(say string) string {
	switch say {
	case "text":
		return "text"
	case "task":
		return "task"
	case "error":
		return "error"
	case "api_req_started":
		return "api_req_started"
	case "api_req_finished":
		return "api_req_finished"
	case "command":
		return "command"
	case "command_output":
		return "command_output"
	case "tool":
		return "tool"
	case "completion_result":
		return "completion_result"
	case "user_feedback":
		return "user_feedback"
	case "reasoning", "thinking":
		return "thinking"
	case "checkpoint_created":
		return "checkpoint_created"
	case "browser_action":
		return "browser_action"
	case "browser_action_result":
		return "browser_action_result"
	case "mcp_server_request":
		return "mcp_server_request"
	case "mcp_server_response":
		return "mcp_server_response"
	default:
		return say
	}
}

// shouldAutoApprove determines if an ask should be auto-approved
func shouldAutoApprove(askType string, config Config) bool {
	if !config.Yolo && !config.AutoApproveAll {
		return false
	}
	
	// In yolo mode, auto-approve certain ask types
	switch askType {
	case "command", "tool", "browser_action_launch":
		return config.Yolo || config.AutoApproveAll
	default:
		return false
	}
}

// loadImageData loads an image file and returns base64 encoded data
// Note: Uses the getMimeType function from init.go
func loadImageData(path string) (string, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	
	// Determine mime type from extension
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := getMimeType(ext)
	if mimeType == "" {
		mimeType = "image/png" // Default to png
	}
	
	// Encode as data URL
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}
