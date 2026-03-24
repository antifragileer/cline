// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
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
	uiClient   cline.UiServiceClient
	conn       *grpc.ClientConn
}

// NewRunner creates a new task runner
func NewRunner(conn *grpc.ClientConn) *Runner {
	return &Runner{
		taskClient: cline.NewTaskServiceClient(conn),
		uiClient:   cline.NewUiServiceClient(conn),
		conn:       conn,
	}
}

// Run executes a task with the given configuration
func (r *Runner) Run(ctx context.Context, config Config, handler MessageHandler) error {
	// Validate configuration
	if config.Prompt == "" && config.TaskID == "" {
		return fmt.Errorf("task prompt required (or use TaskID to resume)")
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

	// Create the task
	newTaskReq := &cline.NewTaskRequest{
		Text:   config.Prompt,
		Images: imageData,
	}

	// Create task
	resp, err := r.taskClient.NewTask(ctx, newTaskReq)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	taskID := resp.Value
	if config.Verbose {
		handler.OnInfo(fmt.Sprintf("Task created: %s", taskID))
	}

	// Subscribe to message updates
	if err := r.subscribeToMessages(ctx, handler); err != nil {
		return fmt.Errorf("failed to subscribe to messages: %w", err)
	}

	return nil
}

// RunWithStreaming executes a task and streams messages to the handler
func (r *Runner) RunWithStreaming(ctx context.Context, config Config, handler MessageHandler) error {
	// Validate configuration
	if config.Prompt == "" && config.TaskID == "" {
		return fmt.Errorf("task prompt required (or use TaskID to resume)")
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

	// Create the task
	newTaskReq := &cline.NewTaskRequest{
		Text:   config.Prompt,
		Images: imageData,
	}

	// Create task
	resp, err := r.taskClient.NewTask(ctx, newTaskReq)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	taskID := resp.Value
	if config.Verbose && !config.JSON {
		handler.OnInfo(fmt.Sprintf("Task created: %s", taskID))
	}

	// Subscribe to message updates and process them
	return r.subscribeAndProcessMessages(ctx, config, taskID, handler)
}

// subscribeAndProcessMessages subscribes to partial messages and processes them
func (r *Runner) subscribeAndProcessMessages(ctx context.Context, config Config, taskID string, handler MessageHandler) error {
	// Create subscription request
	req := &cline.EmptyRequest{}

	// Subscribe to partial messages
	stream, err := r.uiClient.SubscribeToPartialMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to messages: %w", err)
	}

	// Process messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			// Stream closed normally
			return nil
		}
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Canceled {
				// Context was cancelled
				return ctx.Err()
			}
			return fmt.Errorf("error receiving message: %w", err)
		}

		// Process the message
		if err := r.handleMessage(ctx, msg, config, handler); err != nil {
			return err
		}
	}
}

// handleMessage processes a single ClineMessage
func (r *Runner) handleMessage(ctx context.Context, msg *cline.ClineMessage, config Config, handler MessageHandler) error {
	// Handle based on message type
	switch msg.Type {
	case cline.ClineMessageType_SAY:
		return r.handleSayMessage(ctx, msg, config, handler)
	case cline.ClineMessageType_ASK:
		return r.handleAskMessage(ctx, msg, config, handler)
	default:
		// Unknown message type, just log it
		if config.Verbose {
			handler.OnInfo(fmt.Sprintf("Unknown message type: %v", msg.Type))
		}
		return nil
	}
}

// handleSayMessage handles a SAY message from the assistant
func (r *Runner) handleSayMessage(ctx context.Context, msg *cline.ClineMessage, config Config, handler MessageHandler) error {
	// Check if it's a partial message
	if msg.Partial {
		// For partial messages, we may want to update in-place in TUI mode
		// For now, just pass to handler
	}

	// Convert say type to string for handler
	sayType := sayTypeToString(msg.Say)
	
	// Pass to handler
	handler.OnSay(sayType, msg.Text, msg.Partial)

	return nil
}

// handleAskMessage handles an ASK message (requires user response)
func (r *Runner) handleAskMessage(ctx context.Context, msg *cline.ClineMessage, config Config, handler MessageHandler) error {
	askType := askTypeToString(msg.Ask)

	// Check for auto-approval
	if shouldAutoApprove(msg.Ask, config) {
		// Auto-approve
		if err := r.sendAskResponse(ctx, "yesButtonClicked", "", nil, nil); err != nil {
			return fmt.Errorf("failed to send auto-approval: %w", err)
		}
		handler.OnInfo("Auto-approved: " + askType)
		return nil
	}

	// Get user response
	response, err := handler.OnAsk(askType, msg.Text)
	if err != nil {
		return err
	}

	// Send response
	if err := r.sendAskResponse(ctx, response, "", nil, nil); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// sendAskResponse sends a response to an ask message
func (r *Runner) sendAskResponse(ctx context.Context, responseType, text string, images, files []string) error {
	req := &cline.AskResponseRequest{
		ResponseType: responseType,
		Text:         text,
		Images:       images,
		Files:        files,
	}

	_, err := r.taskClient.AskResponse(ctx, req)
	return err
}

// shouldAutoApprove determines if an ask should be auto-approved
func shouldAutoApprove(askType cline.ClineAsk, config Config) bool {
	if !config.Yolo && !config.AutoApproveAll {
		return false
	}

	// In yolo mode, auto-approve certain ask types
	switch askType {
	case cline.ClineAsk_COMMAND:
		return config.Yolo || config.AutoApproveAll
	case cline.ClineAsk_TOOL:
		return config.Yolo || config.AutoApproveAll
	default:
		return false
	}
}

// loadImageData loads an image file and returns base64 encoded data
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

// sayTypeToString converts ClineSay to string
func sayTypeToString(say cline.ClineSay) string {
	switch say {
	case cline.ClineSay_TEXT:
		return "text"
	case cline.ClineSay_TASK:
		return "task"
	case cline.ClineSay_ERROR:
		return "error"
	case cline.ClineSay_API_REQ_STARTED:
		return "api_req_started"
	case cline.ClineSay_API_REQ_FINISHED:
		return "api_req_finished"
	case cline.ClineSay_COMMAND_SAY:
		return "command"
	case cline.ClineSay_COMMAND_OUTPUT_SAY:
		return "command_output"
	case cline.ClineSay_TOOL_SAY:
		return "tool"
	case cline.ClineSay_COMPLETION_RESULT_SAY:
		return "completion_result"
	case cline.ClineSay_USER_FEEDBACK:
		return "user_feedback"
	case cline.ClineSay_USER_FEEDBACK_DIFF:
		return "user_feedback_diff"
	case cline.ClineSay_CHECKPOINT_CREATED:
		return "checkpoint_created"
	case cline.ClineSay_GENERATE_EXPLANATION:
		return "generate_explanation"
	case cline.ClineSay_REASONING:
		return "thinking"
	case cline.ClineSay_BROWSER_ACTION:
		return "browser_action"
	case cline.ClineSay_BROWSER_ACTION_RESULT:
		return "browser_action_result"
	case cline.ClineSay_MCP_SERVER_REQUEST_STARTED:
		return "mcp_server_request"
	case cline.ClineSay_MCP_SERVER_RESPONSE:
		return "mcp_server_response"
	case cline.ClineSay_MCP_NOTIFICATION:
		return "mcp_notification"
	case cline.ClineSay_SHELL_INTEGRATION_WARNING:
		return "shell_integration_warning"
	case cline.ClineSay_API_REQ_RETRIED:
		return "api_req_retried"
	case cline.ClineSay_DIFF_ERROR:
		return "diff_error"
	case cline.ClineSay_DELETED_API_REQS:
		return "deleted_api_reqs"
	case cline.ClineSay_CLINEIGNORE_ERROR:
		return "clineignore_error"
	case cline.ClineSay_LOAD_MCP_DOCUMENTATION:
		return "load_mcp_documentation"
	case cline.ClineSay_INFO:
		return "info"
	case cline.ClineSay_TASK_PROGRESS:
		return "task_progress"
	case cline.ClineSay_ERROR_RETRY:
		return "error_retry"
	case cline.ClineSay_HOOK_STATUS:
		return "hook_status"
	case cline.ClineSay_HOOK_OUTPUT_STREAM:
		return "hook_output_stream"
	case cline.ClineSay_COMMAND_PERMISSION_DENIED:
		return "command_permission_denied"
	case cline.ClineSay_CONDITIONAL_RULES_APPLIED:
		return "conditional_rules_applied"
	case cline.ClineSay_SUBAGENT_STATUS:
		return "subagent_status"
	case cline.ClineSay_USE_SUBAGENTS_SAY:
		return "use_subagents"
	case cline.ClineSay_SUBAGENT_USAGE:
		return "subagent_usage"
	default:
		return "unknown"
	}
}

// askTypeToString converts ClineAsk to string
func askTypeToString(ask cline.ClineAsk) string {
	switch ask {
	case cline.ClineAsk_FOLLOWUP:
		return "followup"
	case cline.ClineAsk_COMMAND:
		return "command_approval"
	case cline.ClineAsk_TOOL:
		return "tool_approval"
	case cline.ClineAsk_COMMAND_OUTPUT:
		return "command_output"
	case cline.ClineAsk_COMPLETION_RESULT:
		return "completion_result"
	case cline.ClineAsk_RESUME_TASK:
		return "resume_task"
	case cline.ClineAsk_RESUME_COMPLETED_TASK:
		return "resume_completed_task"
	case cline.ClineAsk_USE_MCP_SERVER:
		return "use_mcp_server"
	case cline.ClineAsk_NEW_TASK:
		return "new_task"
	case cline.ClineAsk_BROWSER_ACTION_LAUNCH:
		return "browser_action_launch"
	case cline.ClineAsk_CONDENSE:
		return "condense"
	case cline.ClineAsk_REPORT_BUG:
		return "report_bug"
	case cline.ClineAsk_SUMMARIZE_TASK:
		return "summarize_task"
	case cline.ClineAsk_ACT_MODE_RESPOND:
		return "act_mode_respond"
	case cline.ClineAsk_USE_SUBAGENTS:
		return "use_subagents"
	case cline.ClineAsk_MISTAKE_LIMIT_REACHED:
		return "mistake_limit_reached"
	case cline.ClineAsk_API_REQ_FAILED:
		return "api_req_failed"
	default:
		return "unknown"
	}
}

// SubscribeToPartialMessage subscribes to partial message updates
func (r *Runner) subscribeToMessages(ctx context.Context, handler MessageHandler) error {
	req := &cline.EmptyRequest{}
	
	stream, err := r.uiClient.SubscribeToPartialMessage(ctx, req)
	if err != nil {
		return err
	}

	// Start a goroutine to handle messages
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					handler.OnError(fmt.Errorf("stream error: %w", err))
				}
				return
			}

			// Handle message
			if msg.Type == cline.ClineMessageType_SAY {
				handler.OnSay(sayTypeToString(msg.Say), msg.Text, msg.Partial)
			} else if msg.Type == cline.ClineMessageType_ASK {
				// For asks in background mode, auto-approve or log
				handler.OnInfo(fmt.Sprintf("Ask received: %s", askTypeToString(msg.Ask)))
			}
		}
	}()

	return nil
}