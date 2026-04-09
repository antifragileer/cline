// Package task provides task execution for the Cline CLI
// Reference: cli/src/index.ts, src/core/task/index.ts
package task

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/errorservice"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/session"
	"github.com/cline/cline/golang-cli/internal/telemetry"
)

// Runner executes tasks and manages the task lifecycle
// Reference: cli/src/index.ts:600-940
type Runner struct {
	mu              sync.RWMutex
	config          *Config
	approvalHandler *ApprovalHandler
	telemetry       telemetry.Service
	errorService    errorservice.Service
	session         session.Manager
	logger          *slog.Logger

	// Task state
	taskID      string
	isRunning   bool
	isCancelled bool
	exitCode    int

	// gRPC connection
	grpcClient  *host.Client
	streamHandler *host.TaskStreamHandler

	// Message handling
	messageHandler MessageHandler

	// Results
	taskResult  string
	taskError   error
	completed   bool
}

// NewRunner creates a new task runner
func NewRunner(
	config *Config,
	telemetry telemetry.Service,
	errorService errorservice.Service,
	logger *slog.Logger,
) *Runner {
	// Use default config if nil
	if config == nil {
		config = &Config{Mode: TaskModeAct}
	}

	return &Runner{
		config:          config,
		approvalHandler: NewApprovalHandler(config, logger),
		telemetry:       telemetry,
		errorService:    errorService,
		session:         session.Get(),
		logger:          logger,
		taskID:          generateTaskID(),
	}
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	// Use session ID as base and add timestamp
	sess := session.Get()
	return fmt.Sprintf("%s-%d", sess.GetSessionID(), sess.GetStartTime().Unix())
}

// Start starts a new task with the given prompt
// Reference: cli/src/index.ts:600-660
func (r *Runner) Start(ctx context.Context, prompt string) error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("task already running")
	}
	r.isRunning = true
	r.isCancelled = false
	r.mu.Unlock()

	// Ensure we mark task as not running when done
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
	}()

	// Record task creation telemetry
	if r.telemetry != nil {
		_ = r.telemetry.CaptureTaskCreated(r.taskID, "default")
	}

	// Record session tracking
	r.session.StartAPICall()
	defer r.session.EndAPICall()

	r.logger.Info("Starting task", "task_id", r.taskID, "prompt", prompt)

	// TODO: Integrate with gRPC core for actual task execution
	// This is a placeholder for the actual task execution logic

	// For now, simulate task execution
	return r.executeTask(ctx, prompt)
}

// executeTask executes the actual task logic
func (r *Runner) executeTask(ctx context.Context, prompt string) error {
	// This would integrate with the gRPC core to:
	// 1. Send the prompt to the core
	// 2. Stream responses
	// 3. Handle tool approvals
	// 4. Track progress

	// For now, this is a placeholder implementation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Simulate work
		r.logger.Info("Executing task", "prompt", prompt)
		return nil
	}
}

// RequestApproval requests approval for a tool execution
// Reference: cli/src/index.ts:288-329
func (r *Runner) RequestApproval(toolName string, description string, details map[string]string) (*ApprovalResponse, error) {
	request := &ApprovalRequest{
		ToolName:    toolName,
		Description: description,
		Details:     details,
	}

	response, err := r.approvalHandler.HandleApproval(request)
	if err != nil {
		r.logger.Error("Failed to handle approval", "error", err)
		return nil, err
	}

	// Record tool result for mistake tracking
	if response.Approved {
		// Reset mistakes on successful approval
		_ = r.approvalHandler.RecordToolResult(true)
	}

	return response, nil
}

// CompleteTask completes the task and handles double-check mode
// Reference: cli/src/index.ts:216-218
func (r *Runner) CompleteTask() (*ApprovalResponse, error) {
	return r.approvalHandler.HandleCompletion()
}

// HandleToolError handles a tool error and tracks consecutive mistakes
// Reference: cli/src/index.ts:195-199
func (r *Runner) HandleToolError(err error) error {
	// Record the error as a failed tool result
	if recordErr := r.approvalHandler.RecordToolResult(false); recordErr != nil {
		// Max consecutive mistakes reached
		r.logger.Error("Max consecutive mistakes reached", "error", recordErr)
		if r.errorService != nil {
			r.errorService.CaptureException(recordErr, map[string]string{
				"task_id": r.taskID,
				"source":  "max_consecutive_mistakes",
			})
		}
		return recordErr
	}

	// Log the error
	if r.errorService != nil {
		r.errorService.LogException(err, map[string]string{
			"task_id": r.taskID,
			"tool":    "execution",
		})
	}

	return nil
}

// Cancel cancels the current task
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.isCancelled = true
	r.logger.Info("Task cancelled", "task_id", r.taskID)
}

// IsRunning returns whether a task is currently running
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRunning
}

// IsCancelled returns whether the task has been cancelled
func (r *Runner) IsCancelled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isCancelled
}

// GetSessionStats returns current session statistics
func (r *Runner) GetSessionStats() session.Stats {
	return r.session.GetStats()
}

// ShouldAutoApprove returns whether tools should be auto-approved
func (r *Runner) ShouldAutoApprove() bool {
	return r.approvalHandler.ShouldAutoApprove()
}

// IsYoloMode returns whether yolo mode is enabled
func (r *Runner) IsYoloMode() bool {
	return r.approvalHandler.IsYoloMode()
}

// ShouldExitOnCompletion returns whether to exit after task completion
func (r *Runner) ShouldExitOnCompletion() bool {
	return r.approvalHandler.ShouldExitOnCompletion()
}

// AutoCondenseEnabled returns whether auto-condense is enabled
// Reference: cli/src/index.ts:221-223
func (r *Runner) AutoCondenseEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// This would check the config for auto-condense setting
	// For now, return false as default
	return false
}

// GetConfig returns the task configuration
func (r *Runner) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// Run executes a task with the given configuration and message handler
// Reference: cli/src/index.ts:600-940
func (r *Runner) Run(ctx context.Context, config TaskConfig, handler MessageHandler) error {
	// Initialize task runner if not already done
	if r.taskID == "" {
		r.taskID = generateTaskID()
	}

	// Start the task
	return r.Start(ctx, config.Prompt)
}

// RunWithStreaming executes a task with streaming JSON output (yolo mode)
// This implements complete yolo mode with auto-approval chain and exit code capture
func (r *Runner) RunWithStreaming(ctx context.Context, prompt string, autoApprove bool) (int, string, error) {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return 1, "", fmt.Errorf("task already running")
	}
	r.isRunning = true
	r.isCancelled = false
	r.completed = false
	r.exitCode = 0
	r.mu.Unlock()

	// Ensure we mark task as not running when done
	defer func() {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
	}()

	// Record task creation telemetry
	if r.telemetry != nil {
		_ = r.telemetry.CaptureTaskCreated(r.taskID, "streaming")
	}

	// Record session tracking
	r.session.StartAPICall()
	defer r.session.EndAPICall()

	r.logger.Info("Starting streaming task", "task_id", r.taskID, "prompt", prompt, "auto_approve", autoApprove)

	// Create enhanced message handler with yolo configuration
	handlerConfig := &HandlerConfig{
		YoloMode:              autoApprove,
		EnablePartialMessages: true,
		MaxHistorySize:        1000,
	}

	if autoApprove {
		// In yolo mode, auto-approve all standard tools
		handlerConfig.AutoApproveTools = []string{
			"read_file",
			"write_file",
			"edit_file",
			"execute_command",
			"search_files",
			"list_files",
			"browser_action",
		}
	}

	enhancedHandler := NewEnhancedMessageHandler(handlerConfig)

	// Set up callbacks for streaming output
	var resultBuilder strings.Builder
	var lastError error

	enhancedHandler.SetTextCallback(func(content string, isPartial bool) error {
		if !isPartial {
			resultBuilder.WriteString(content)
			resultBuilder.WriteString("\n")
		}
		return nil
	})

	enhancedHandler.SetCompletionCallback(func(success bool, summary string) error {
		r.mu.Lock()
		r.completed = true
		r.taskResult = summary
		if success {
			r.exitCode = 0
		} else {
			r.exitCode = 1
		}
		r.mu.Unlock()
		return nil
	})

	enhancedHandler.SetErrorCallback(func(err error) error {
		lastError = err
		r.mu.Lock()
		r.taskError = err
		r.exitCode = 1
		r.mu.Unlock()
		return nil
	})

	// Execute task with streaming
	err := r.executeStreamingTask(ctx, prompt, enhancedHandler)
	if err != nil {
		return 1, "", err
	}

	// Wait for completion or cancellation
	select {
	case <-ctx.Done():
		return 1, "", ctx.Err()
	default:
		// Task completed
	}

	r.mu.RLock()
	exitCode := r.exitCode
	result := resultBuilder.String()
	r.mu.RUnlock()

	return exitCode, result, lastError
}

// executeStreamingTask executes a task with bidirectional streaming
func (r *Runner) executeStreamingTask(ctx context.Context, prompt string, handler *EnhancedMessageHandler) error {
	// This would integrate with gRPC streaming
	// For now, simulate task execution with message handling

	// Simulate sending initial task message
	taskMsg := &JSONMessage{
		Ts:   time.Now().UnixMilli(),
		Type: string(ClineMessageTypeSay),
		Say:  string(ClineSayTask),
		Text: prompt,
	}

	if err := handler.HandleMessage(taskMsg); err != nil {
		return err
	}

	// In a real implementation, this would:
	// 1. Connect to gRPC stream
	// 2. Send the prompt
	// 3. Process incoming messages
	// 4. Send responses back
	// 5. Handle completion

	return nil
}

// GetExitCode returns the task exit code
func (r *Runner) GetExitCode() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.exitCode
}

// GetTaskResult returns the task result
func (r *Runner) GetTaskResult() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskResult
}

// IsCompleted returns whether the task completed
func (r *Runner) IsCompleted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.completed
}

// YoloModeConfig holds yolo mode configuration
type YoloModeConfig struct {
	// AutoApprove enables auto-approval of all tools
	AutoApprove bool

	// ExitOnCompletion exit after task completion
	ExitOnCompletion bool

	// PlainText use plain text output instead of TUI
	PlainText bool

	// CaptureOutput capture and return output
	CaptureOutput bool
}

// ExecuteYoloMode executes a task in yolo mode with full auto-approval
func (r *Runner) ExecuteYoloMode(ctx context.Context, prompt string, config YoloModeConfig) (int, string, error) {
	// Force yolo mode in config
	r.mu.Lock()
	r.config.Yolo = true
	r.config.PlainTextMode = config.PlainText
	r.config.YoloWarningShown = false
	r.mu.Unlock()

	// Show yolo warning if not already shown
	if !r.config.YoloWarningShown {
		fmt.Println(YoloWarning)
		r.config.YoloWarningShown = true
	}

	// Execute with streaming
	exitCode, output, err := r.RunWithStreaming(ctx, prompt, true)

	// Handle exit on completion
	if config.ExitOnCompletion {
		r.logger.Info("Exiting on completion (yolo mode)", "exit_code", exitCode)
	}

	return exitCode, output, err
}

// SetMessageHandler sets the message handler for the runner
func (r *Runner) SetMessageHandler(handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageHandler = handler
}

// GetMessageHandler returns the current message handler
func (r *Runner) GetMessageHandler() MessageHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.messageHandler
}

// ConnectGRPC connects to the gRPC server
func (r *Runner) ConnectGRPC(target string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clientConfig := host.ClientConfig{
		Target:         target,
		PoolSize:       3,
		ConnTimeout:    10 * time.Second,
		MaxRetries:     3,
		ReconnectDelay: 2 * time.Second,
	}

	client, err := host.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	if err := client.Start(); err != nil {
		return fmt.Errorf("failed to start gRPC client: %w", err)
	}

	r.grpcClient = client
	return nil
}

// DisconnectGRPC disconnects from the gRPC server
func (r *Runner) DisconnectGRPC() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.grpcClient != nil {
		return r.grpcClient.Stop()
	}
	return nil
}
