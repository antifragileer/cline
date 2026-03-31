// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// Runner handles task execution and provides a high-level interface
type Runner struct {
	executor      *Executor
	resumeManager *ResumeManager
	modeHandler   *ModeHandler
	attachmentMgr *AttachmentManager
	conn          *grpc.ClientConn
}

// NewRunner creates a new task runner with all necessary components
func NewRunner(conn *grpc.ClientConn) *Runner {
	return &Runner{
		executor:      NewExecutor(conn),
		resumeManager: NewResumeManager(conn),
		attachmentMgr: NewAttachmentManager(),
		conn:          conn,
	}
}

// Run executes a new task with the given configuration
func (r *Runner) Run(ctx context.Context, config TaskConfig, handler MessageHandler) error {
	// Set up mode handler
	r.modeHandler = NewModeHandler(config.Mode)

	// Validate images if provided
	if len(config.Images) > 0 {
		if err := r.attachmentMgr.ValidateImagePaths(config.Images); err != nil {
			return fmt.Errorf("image validation failed: %w", err)
		}
	}

	// Execute the task
	return r.executor.Execute(ctx, config, handler)
}

// Resume resumes an existing task by ID
func (r *Runner) Resume(ctx context.Context, taskID string, prompt string, images []string, handler MessageHandler) error {
	// Resume the task
	resumedID, err := r.resumeManager.ResumeTask(ctx, taskID, prompt, images)
	if err != nil {
		return fmt.Errorf("failed to resume task: %w", err)
	}

	// Set up a config for the resumed task
	config := TaskConfig{
		TaskID: resumedID,
		Mode:   TaskModeAct, // Default to act mode for resumed tasks
		Images: images,
	}

	// Continue execution
	return r.executor.Execute(ctx, config, handler)
}

// ResumeMostRecent resumes the most recent task
func (r *Runner) ResumeMostRecent(ctx context.Context, prompt string, images []string, handler MessageHandler) error {
	// Get the most recent task
	taskInfo, err := r.resumeManager.GetMostRecentTask(ctx)
	if err != nil {
		return fmt.Errorf("failed to get most recent task: %w", err)
	}

	if taskInfo == nil {
		return fmt.Errorf("no recent tasks found to resume")
	}

	// Resume it
	return r.Resume(ctx, taskInfo.TaskID, prompt, images, handler)
}

// GetResumableTasks returns a list of tasks that can be resumed
func (r *Runner) GetResumableTasks(ctx context.Context) ([]*TaskInfo, error) {
	return r.resumeManager.ListResumableTasks(ctx)
}

// CanResume checks if a specific task can be resumed
func (r *Runner) CanResume(ctx context.Context, taskID string) (bool, string) {
	return r.resumeManager.CanResume(ctx, taskID)
}

// Cancel cancels the current task execution
func (r *Runner) Cancel() {
	if r.executor != nil {
		r.executor.Cancel()
	}
}

// IsRunning returns whether a task is currently running
func (r *Runner) IsRunning() bool {
	if r.executor != nil {
		return r.executor.IsRunning()
	}
	return false
}

// GetTaskID returns the current task ID
func (r *Runner) GetTaskID() string {
	if r.executor != nil {
		return r.executor.GetTaskID()
	}
	return ""
}

// Close closes the runner and releases resources
func (r *Runner) Close() error {
	if r.executor != nil {
		return r.executor.Close()
	}
	return nil
}

