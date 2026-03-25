// Package task provides task initialization and management functionality for the Cline CLI.
// This file implements task resumption functionality using the ShowTaskWithId RPC and
// provides utilities for finding and continuing recent tasks.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
	"google.golang.org/grpc"
)

// HistoryItem represents a task history entry for resumption
type HistoryItem struct {
	// TaskID is the unique task identifier
	TaskID string `json:"taskId"`

	// Prompt is the original task prompt
	Prompt string `json:"prompt"`

	// Mode is the task execution mode (act/plan)
	Mode TaskMode `json:"mode"`

	// Cwd is the working directory where the task was created
	Cwd string `json:"cwd"`

	// Model is the model used for the task
	Model string `json:"model"`

	// CreatedAt is the timestamp when the task was created
	CreatedAt int64 `json:"createdAt"`

	// UpdatedAt is the timestamp when the task was last updated
	UpdatedAt int64 `json:"updatedAt"`

	// Status is the task status (initialized, running, completed, error)
	Status string `json:"status"`

	// MessageCount is the number of messages in the conversation
	MessageCount int `json:"messageCount,omitempty"`
}

// ResumeOptions provides options for task resumption
type ResumeOptions struct {
	// TaskID is the ID of the task to resume
	TaskID string `json:"taskId"`

	// Prompt is an optional new prompt to add when resuming
	Prompt string `json:"prompt"`

	// Images is a list of image file paths to attach
	Images []string `json:"images"`

	// Verbose enables verbose output
	Verbose bool `json:"verbose"`

	// Timeout is the maximum duration for task execution
	Timeout time.Duration `json:"timeout"`

	// Storage is the storage context for accessing task history
	Storage *storage.StorageContext `json:"-"`

	// Client is the gRPC client for communication
	Client *host.Client `json:"-"`

	// Output is the output writer for logging
	Output interface {
		Printf(format string, a ...interface{}) (n int, err error)
	} `json:"-"`
}

// ResumeResult contains the result of a task resumption
type ResumeResult struct {
	// TaskID is the resumed task ID
	TaskID string `json:"taskId"`

	// Task contains the task details
	Task *cline.TaskResponse `json:"task"`

	// IsResumed indicates if the task was successfully resumed
	IsResumed bool `json:"isResumed"`

	// Message contains any additional message
	Message string `json:"message"`

	// HistoryItem contains the task history information
	HistoryItem *HistoryItem `json:"historyItem,omitempty"`
}

// Resumer handles task resumption
type Resumer struct {
	client  *host.Client
	storage *storage.StorageContext
	output  interface {
		Printf(format string, a ...interface{}) (n int, err error)
	}
}

// NewResumer creates a new task resumer
func NewResumer(client *host.Client) *Resumer {
	return &Resumer{
		client: client,
		output: &nopWriter{},
	}
}

// NewResumerWithStorage creates a new task resumer with storage access
func NewResumerWithStorage(client *host.Client, storage *storage.StorageContext) *Resumer {
	return &Resumer{
		client:  client,
		storage: storage,
		output:  &nopWriter{},
	}
}

// SetOutput sets the output writer for verbose logging
func (r *Resumer) SetOutput(w interface{ Printf(format string, a ...interface{}) (n int, err error) }) {
	r.output = w
}

// Resume resumes a task by its ID
func (r *Resumer) Resume(ctx context.Context, opts ResumeOptions) (*ResumeResult, error) {
	if opts.TaskID == "" {
		return nil, fmt.Errorf("task ID is required for resumption")
	}

	if r.client == nil {
		return nil, fmt.Errorf("gRPC client not available")
	}

	// Wait for client to be ready
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use defer/recover to handle potential panics from nil connection pool
	var waitErr error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				waitErr = fmt.Errorf("gRPC client not available")
			}
		}()
		waitErr = r.client.WaitForReady(ctx)
	}()

	if waitErr != nil {
		return nil, fmt.Errorf("gRPC client not ready: %w", waitErr)
	}

	if opts.Verbose {
		r.output.Printf("Resuming task: %s\n", opts.TaskID)
	}

	// Build the ShowTaskWithId request
	req := &cline.StringRequest{
		Value: opts.TaskID,
	}

	// Execute the gRPC call with retry
	var taskResponse *cline.TaskResponse
	err := r.client.WithRetry(ctx, func(conn *grpc.ClientConn) error {
		client := cline.NewTaskServiceClient(conn)
		var err error
		taskResponse, err = client.ShowTaskWithId(ctx, req)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("showTaskWithId RPC failed: %w", err)
	}

	if taskResponse == nil {
		return nil, fmt.Errorf("task not found: %s", opts.TaskID)
	}

	if opts.Verbose {
		r.output.Printf("Task found: %s\n", taskResponse.GetTask())
		r.output.Printf("Created: %s\n", time.Unix(taskResponse.GetTs(), 0).Format(time.RFC3339))
	}

	// Load and validate images if provided
	var imageDataUrls []string
	if len(opts.Images) > 0 {
		// Use the initializer's image loading logic
		// This is a simplified version - in production, you'd want to reuse the code
		imageDataUrls = make([]string, 0, len(opts.Images))
		for _, imgPath := range opts.Images {
			// Simple validation - in production, use full validation
			imageDataUrls = append(imageDataUrls, imgPath)
		}
	}

	// If a new prompt is provided, send it as part of the resumption
	if opts.Prompt != "" {
		// The prompt would typically be sent through the streaming mechanism
		// after the task is resumed. This is handled by the caller.
		if opts.Verbose {
			r.output.Printf("Additional prompt provided for resumption\n")
		}
	}

	return &ResumeResult{
		TaskID:    opts.TaskID,
		Task:      taskResponse,
		IsResumed: true,
		Message:   "Task resumed successfully",
	}, nil
}

// ResumeWithPrompt resumes a task and sends an additional prompt
func (r *Resumer) ResumeWithPrompt(ctx context.Context, taskID string, prompt string, images []string) (*ResumeResult, error) {
	// First resume the task
	result, err := r.Resume(ctx, ResumeOptions{
		TaskID:  taskID,
		Prompt:  prompt,
		Images:  images,
		Verbose: true,
	})
	if err != nil {
		return nil, err
	}

	// If a prompt was provided, we need to send it
	// In a full implementation, this would use the streaming mechanism
	// to send the prompt to the resumed task
	if prompt != "" {
		// The actual sending would happen here via the stream
		// This is simplified for the current implementation
		result.Message = "Task resumed with additional prompt"
	}

	return result, nil
}

// ResumeTask resumes an existing task by ID with full execution.
// This is the high-level function called from the CLI.
func ResumeTask(taskID string, opts ResumeOptions) error {
	if taskID == "" {
		return fmt.Errorf("task ID is required")
	}

	// Ensure we have a storage context
	if opts.Storage == nil {
		return fmt.Errorf("storage context is required for task resumption")
	}

	// Ensure we have a client
	if opts.Client == nil {
		return fmt.Errorf("gRPC client is required for task resumption")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add timeout if specified
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create resumer
	resumer := NewResumerWithStorage(opts.Client, opts.Storage)
	if opts.Output != nil {
		resumer.SetOutput(opts.Output)
	}

	// Verify task exists in history
	history, err := GetTaskHistory(opts.Storage, 0)
	if err != nil {
		return fmt.Errorf("failed to load task history: %w", err)
	}

	var targetTask *HistoryItem
	for _, item := range history {
		if item.TaskID == taskID {
			targetTask = &item
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task %s not found in history", taskID)
	}

	if opts.Verbose {
		fmt.Printf("Resuming task: %s\n", taskID)
		fmt.Printf("Original prompt: %s\n", targetTask.Prompt)
		fmt.Printf("Working directory: %s\n", targetTask.Cwd)
		fmt.Printf("Mode: %s\n", targetTask.Mode)
	}

	// Resume the task via gRPC
	result, err := resumer.Resume(ctx, ResumeOptions{
		TaskID:  taskID,
		Prompt:  opts.Prompt,
		Images:  opts.Images,
		Verbose: opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("failed to resume task: %w", err)
	}

	if !result.IsResumed {
		return fmt.Errorf("task resumption failed: %s", result.Message)
	}

	// If a follow-up prompt was provided, continue execution with it
	if opts.Prompt != "" {
		if opts.Verbose {
			fmt.Printf("Sending follow-up message: %s\n", opts.Prompt)
		}

		// Continue task execution with the new prompt
		if err := continueTaskExecution(ctx, opts, taskID, opts.Prompt); err != nil {
			return fmt.Errorf("failed to continue task execution: %w", err)
		}
	}

	if opts.Verbose {
		fmt.Printf("Task resumed successfully: %s\n", taskID)
	}

	return nil
}

// FindMostRecentTask finds the most recent task for the current workspace.
// It returns the most recent task history item or an error if no tasks exist.
func FindMostRecentTask(storage *storage.StorageContext, workspacePath string) (*HistoryItem, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage context is required")
	}

	// Get current working directory if not provided
	if workspacePath == "" {
		var err error
		workspacePath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Normalize workspace path
	workspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load task history
	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load task history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no tasks found in history")
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(history, func(i, j int) bool {
		return history[i].UpdatedAt > history[j].UpdatedAt
	})

	// Find the most recent task for this workspace
	for _, item := range history {
		itemCwd, err := filepath.Abs(item.Cwd)
		if err != nil {
			continue
		}

		if itemCwd == workspacePath {
			return &item, nil
		}
	}

	// If no task found for this workspace, return the most recent overall
	return &history[0], nil
}

// ContinueTask resumes the most recent task for the current workspace.
// This is the high-level function called when using --continue flag.
func ContinueTask(opts ResumeOptions) error {
	if opts.Storage == nil {
		return fmt.Errorf("storage context is required")
	}

	if opts.Client == nil {
		return fmt.Errorf("gRPC client is required")
	}

	// Find the most recent task
	historyItem, err := FindMostRecentTask(opts.Storage, "")
	if err != nil {
		return fmt.Errorf("failed to find recent task: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("Continuing most recent task: %s\n", historyItem.TaskID)
		fmt.Printf("Original prompt: %s\n", historyItem.Prompt)
	}

	// Resume the task
	return ResumeTask(historyItem.TaskID, ResumeOptions{
		TaskID:   historyItem.TaskID,
		Prompt:   opts.Prompt,
		Images:   opts.Images,
		Verbose:  opts.Verbose,
		Timeout:  opts.Timeout,
		Storage:  opts.Storage,
		Client:   opts.Client,
		Output:   opts.Output,
	})
}

// GetTaskHistory retrieves the task history from storage.
// If limit > 0, returns only the most recent 'limit' entries.
func GetTaskHistory(storage *storage.StorageContext, limit int) ([]HistoryItem, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage context is required")
	}

	var history []HistoryItem
	if data, ok := storage.GlobalState.Get("taskHistory"); ok {
		// Try to unmarshal based on type
		switch v := data.(type) {
		case []byte:
			if err := json.Unmarshal(v, &history); err != nil {
				return nil, fmt.Errorf("failed to unmarshal history from bytes: %w", err)
			}
		case string:
			if v != "" {
				if err := json.Unmarshal([]byte(v), &history); err != nil {
					return nil, fmt.Errorf("failed to unmarshal history from string: %w", err)
				}
			}
		case []interface{}:
			// Convert from []interface{} to JSON and back
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal history: %w", err)
			}
			if err := json.Unmarshal(bytes, &history); err != nil {
				return nil, fmt.Errorf("failed to unmarshal history: %w", err)
			}
		default:
			// Try JSON marshaling as fallback
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal history: %w", err)
			}
			if err := json.Unmarshal(bytes, &history); err != nil {
				return nil, fmt.Errorf("failed to unmarshal history: %w", err)
			}
		}
	}

	// Apply limit if specified
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}

// UpdateTaskHistory updates a task's history entry
func UpdateTaskHistory(storage *storage.StorageContext, taskID string, updates map[string]interface{}) error {
	if storage == nil {
		return fmt.Errorf("storage context is required")
	}

	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return err
	}

	// Find and update the task
	found := false
	for i, item := range history {
		if item.TaskID == taskID {
			// Apply updates
			if status, ok := updates["status"].(string); ok {
				history[i].Status = status
			}
			if count, ok := updates["messageCount"].(int); ok {
				history[i].MessageCount = count
			}
			history[i].UpdatedAt = time.Now().Unix()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task %s not found in history", taskID)
	}

	// Save updated history
	return storage.GlobalState.Set("taskHistory", history)
}

// continueTaskExecution continues task execution with a new prompt
func continueTaskExecution(ctx context.Context, opts ResumeOptions, taskID string, prompt string) error {
	// Get a connection from the pool
	conn, err := opts.Client.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// Create a task runner to continue the task
	runner := NewRunner(conn)

	// Build task config
	config := Config{
		Mode:      ModeAct, // Default to act mode for continuation
		Prompt:    prompt,
		TaskID:    taskID,
		Verbose:   opts.Verbose,
		Timeout:   opts.Timeout,
		Yolo:      false, // Don't auto-approve on continuation unless specified
	}

	// Create message handler based on output mode
	var handler MessageHandler
	handler = &PlainTextHandler{
		Verbose:     opts.Verbose,
		Output:      os.Stdout,
		AutoApprove: false,
	}

	if opts.Output != nil {
		// Custom output handler if provided
		opts.Output.Printf("Continuing task with new prompt...\n")
	}

	// Run the task with streaming
	if err := runner.RunWithStreaming(ctx, config, handler); err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}

// nopWriter is a no-op writer for default output
type nopWriter struct{}

func (n *nopWriter) Printf(format string, a ...interface{}) (int, error) {
	return 0, nil
}