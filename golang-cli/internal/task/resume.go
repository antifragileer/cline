package task

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// ResumeOptions contains options for resuming a task.
type ResumeOptions struct {
	// TaskID is the ID of the task to resume (empty for most recent)
	TaskID string

	// Prompt is an optional prompt to add when resuming
	Prompt string

	// Verbose enables verbose output
	Verbose bool

	// Timeout is the maximum time to wait for the task
	Timeout time.Duration

	// Storage is the storage context
	Storage *storage.StorageContext

	// Client is the gRPC client
	Client *host.Client

	// Output is the output writer
	Output io.Writer
}

// ContinueTask continues the most recent task.
//
// Parameters:
//   - opts: Resume options
//
// Returns an error if the task cannot be continued.
func ContinueTask(opts ResumeOptions) error {
	// Find most recent task
	history, err := GetTaskHistory(opts.Storage, 1)
	if err != nil {
		return fmt.Errorf("failed to get task history: %w", err)
	}

	if len(history) == 0 {
		return fmt.Errorf("no previous tasks found")
	}

	// Get the most recent task
	recentTask := history[0]

	// Continue that task
	return ResumeTask(recentTask.TaskID, opts)
}

// ResumeTask resumes a specific task by ID.
//
// Parameters:
//   - taskID: The task ID to resume
//   - opts: Resume options
//
// Returns an error if the task cannot be resumed.
func ResumeTask(taskID string, opts ResumeOptions) error {
	if opts.Client == nil {
		return fmt.Errorf("gRPC client is required")
	}

	// Create context with timeout
	ctx := context.Background()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Get connection from pool
	conn, err := opts.Client.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// Create task runner
	runner := NewRunner(conn)

	// Build task config
	config := Config{
		TaskID:  taskID,
		Prompt:  opts.Prompt,
		Mode:    ModeAct,
		Verbose: opts.Verbose,
	}

	// Create message handler
	var handler MessageHandler
	if opts.Verbose {
		handler = NewPlainTextHandler(true, false, false, opts.Output)
	} else {
		handler = NewInteractiveHandler(false, opts.Verbose)
	}

	// Run the task with streaming
	if err := runner.RunWithStreaming(ctx, config, handler); err != nil {
		return fmt.Errorf("failed to resume task: %w", err)
	}

	return nil
}
