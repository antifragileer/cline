// Package task provides task execution for the Cline CLI
// Reference: cli/src/index.ts, src/core/task/index.ts
package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/cline/cline/golang-cli/internal/errorservice"
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
