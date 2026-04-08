// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
)

// ResumeManager handles task resumption functionality
type ResumeManager struct {
	taskClient  cline.TaskServiceClient
	stateClient cline.StateServiceClient
	conn        *grpc.ClientConn
}

// TaskInfo holds information about a task for resumption
type TaskInfo struct {
	// TaskID is the unique identifier for the task
	TaskID string `json:"taskId"`

	// Prompt is the original task prompt
	Prompt string `json:"prompt"`

	// Mode is the execution mode (act/plan)
	Mode TaskMode `json:"mode"`

	// Cwd is the working directory
	Cwd string `json:"cwd"`

	// Model is the model used
	Model string `json:"model"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"createdAt"`

	// LastActivity is when the task was last active
	LastActivity time.Time `json:"lastActivity"`

	// Status is the current status
	Status string `json:"status"`

	// MessageCount is the number of messages in the conversation
	MessageCount int `json:"messageCount"`

	// IsResumable indicates if the task can be resumed
	IsResumable bool `json:"isResumable"`
}

// NewResumeManager creates a new resume manager
func NewResumeManager(conn *grpc.ClientConn) *ResumeManager {
	return &ResumeManager{
		taskClient:  cline.NewTaskServiceClient(conn),
		stateClient: cline.NewStateServiceClient(conn),
		conn:        conn,
	}
}

// ResumeTask resumes an existing task by ID
func (rm *ResumeManager) ResumeTask(ctx context.Context, taskID string, prompt string, images []string) (string, error) {
	if taskID == "" {
		return "", fmt.Errorf("task ID is required for resumption")
	}

	// First, show the task to load it into the core extension
	_, err := rm.taskClient.ShowTaskWithId(ctx, &cline.StringRequest{Value: taskID})
	if err != nil {
		return "", fmt.Errorf("failed to show task %s: %w", taskID, err)
	}

	// Get task info to check if it can be resumed
	taskInfo, err := rm.GetTaskInfo(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task info: %w", err)
	}

	if !taskInfo.IsResumable {
		return "", fmt.Errorf("task %s cannot be resumed (status: %s)", taskID, taskInfo.Status)
	}

	// If there's a prompt to send, send it as a user message
	if prompt != "" {
		// Prepare images if provided
		var imageData []string
		if len(images) > 0 {
			imageData, err = rm.prepareImagesForResume(images)
			if err != nil {
				return "", fmt.Errorf("failed to prepare images: %w", err)
			}
		}

		// Send the prompt as an ask response
		_, err = rm.taskClient.AskResponse(ctx, &cline.AskResponseRequest{
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

// ResumeMostRecentTask resumes the most recent task from history
func (rm *ResumeManager) ResumeMostRecentTask(ctx context.Context, prompt string, images []string) (string, error) {
	// Get the most recent task
	taskInfo, err := rm.GetMostRecentTask(ctx)
	if err != nil {
		return "", err
	}

	if taskInfo == nil {
		return "", fmt.Errorf("no recent tasks found to resume")
	}

	return rm.ResumeTask(ctx, taskInfo.TaskID, prompt, images)
}

// GetTaskInfo retrieves information about a specific task
func (rm *ResumeManager) GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error) {
	// Get task state
	stateResp, err := rm.stateClient.GetLatestState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Parse state to extract task info
	return rm.parseTaskInfoFromState(stateResp.StateJson, taskID)
}

// GetMostRecentTask gets the most recent task from history
func (rm *ResumeManager) GetMostRecentTask(ctx context.Context) (*TaskInfo, error) {
	// Get all tasks
	tasks, err := rm.ListResumableTasks(ctx)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, nil
	}

	// Return the most recent one
	return tasks[0], nil
}

// ListResumableTasks returns a list of tasks that can be resumed
func (rm *ResumeManager) ListResumableTasks(ctx context.Context) ([]*TaskInfo, error) {
	// Get task history from state
	stateResp, err := rm.stateClient.GetLatestState(ctx, &cline.EmptyRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Parse tasks from state
	return rm.parseTasksFromState(stateResp.StateJson)
}

// parseTaskInfoFromState parses task info from state JSON
func (rm *ResumeManager) parseTaskInfoFromState(stateJSON, taskID string) (*TaskInfo, error) {
	if stateJSON == "" {
		return nil, fmt.Errorf("empty state")
	}

	// Parse state JSON
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	// Extract task info
	info := &TaskInfo{
		TaskID:      taskID,
		IsResumable: true, // Assume resumable unless proven otherwise
	}

	// Try to extract from state
	if taskState, ok := state["task"].(map[string]interface{}); ok {
		if id, ok := taskState["id"].(string); ok {
			info.TaskID = id
		}
		if prompt, ok := taskState["prompt"].(string); ok {
			info.Prompt = prompt
		}
		if mode, ok := taskState["mode"].(string); ok {
			info.Mode = ModeFromString(mode)
		}
		if cwd, ok := taskState["cwd"].(string); ok {
			info.Cwd = cwd
		}
		if model, ok := taskState["model"].(string); ok {
			info.Model = model
		}
		if status, ok := taskState["status"].(string); ok {
			info.Status = status
			// Check if resumable based on status
			info.IsResumable = rm.isResumableStatus(status)
		}
	}

	// Try to get messages count
	if messages, ok := state["clineMessages"].([]interface{}); ok {
		info.MessageCount = len(messages)
	}

	return info, nil
}

// parseTasksFromState parses multiple tasks from state
func (rm *ResumeManager) parseTasksFromState(stateJSON string) ([]*TaskInfo, error) {
	if stateJSON == "" {
		return []*TaskInfo{}, nil
	}

	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	var tasks []*TaskInfo

	// Check for task history
	if history, ok := state["taskHistory"].([]interface{}); ok {
		for _, entry := range history {
			taskInfo := rm.parseHistoryEntry(entry)
			if taskInfo != nil && taskInfo.IsResumable {
				tasks = append(tasks, taskInfo)
			}
		}
	}

	// Also check current task
	if currentTask, ok := state["currentTask"].(map[string]interface{}); ok {
		taskInfo := rm.parseCurrentTask(currentTask)
		if taskInfo != nil && taskInfo.IsResumable {
			// Check if already in list
			found := false
			for _, t := range tasks {
				if t.TaskID == taskInfo.TaskID {
					found = true
					break
				}
			}
			if !found {
				tasks = append(tasks, taskInfo)
			}
		}
	}

	// Sort by last activity (most recent first)
	rm.sortTasksByActivity(tasks)

	return tasks, nil
}

// parseHistoryEntry parses a single history entry
func (rm *ResumeManager) parseHistoryEntry(entry interface{}) *TaskInfo {
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return nil
	}

	info := &TaskInfo{
		IsResumable: true,
	}

	if id, ok := entryMap["id"].(string); ok {
		info.TaskID = id
	}
	if prompt, ok := entryMap["task"].(string); ok {
		info.Prompt = prompt
	}
	if mode, ok := entryMap["mode"].(string); ok {
		info.Mode = ModeFromString(mode)
	}
	if cwd, ok := entryMap["cwd"].(string); ok {
		info.Cwd = cwd
	}
	if model, ok := entryMap["model"].(string); ok {
		info.Model = model
	}
	if ts, ok := entryMap["ts"].(float64); ok {
		info.CreatedAt = time.Unix(int64(ts), 0)
		info.LastActivity = info.CreatedAt
	}

	return info
}

// parseCurrentTask parses the current task from state
func (rm *ResumeManager) parseCurrentTask(task map[string]interface{}) *TaskInfo {
	info := &TaskInfo{
		IsResumable: true,
	}

	if id, ok := task["id"].(string); ok {
		info.TaskID = id
	}
	if prompt, ok := task["prompt"].(string); ok {
		info.Prompt = prompt
	}
	if mode, ok := task["mode"].(string); ok {
		info.Mode = ModeFromString(mode)
	}
	if cwd, ok := task["cwd"].(string); ok {
		info.Cwd = cwd
	}
	if model, ok := task["model"].(string); ok {
		info.Model = model
	}
	if status, ok := task["status"].(string); ok {
		info.Status = status
		info.IsResumable = rm.isResumableStatus(status)
	}

	return info
}

// isResumableStatus checks if a task status allows resumption
func (rm *ResumeManager) isResumableStatus(status string) bool {
	nonResumableStatuses := map[string]bool{
		"completed": true,
		"failed":    true,
		"cancelled": true,
		"error":     true,
		"abandoned": true,
	}

	return !nonResumableStatuses[status]
}

// sortTasksByActivity sorts tasks by last activity (most recent first)
func (rm *ResumeManager) sortTasksByActivity(tasks []*TaskInfo) {
	// Simple bubble sort for small lists
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].LastActivity.After(tasks[i].LastActivity) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// prepareImagesForResume prepares images for a resumed task
func (rm *ResumeManager) prepareImagesForResume(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	imageData := make([]string, 0, len(paths))
	for _, path := range paths {
		data, err := loadImageData(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", path, err)
		}
		imageData = append(imageData, data)
	}

	return imageData, nil
}

// loadImageData loads and encodes an image file to base64
func loadImageData(path string) (string, error) {
	// Expand path if needed
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(content), nil
}

// ResumeOptions provides options for task resumption
type ResumeOptions struct {
	// TaskID is the specific task to resume (empty for most recent)
	TaskID string

	// Prompt is an optional prompt to send when resuming
	Prompt string

	// Images are optional images to attach
	Images []string

	// Mode is the mode to resume in (may switch modes)
	Mode TaskMode

	// Verbose enables verbose output
	Verbose bool

	// Yolo enables auto-approval
	Yolo bool
}

// ResumeResult represents the result of a task resumption
type ResumeResult struct {
	// TaskID is the resumed task ID
	TaskID string

	// Success indicates if resumption was successful
	Success bool

	// Message describes the result
	Message string

	// PreviousTaskInfo contains info about the resumed task
	PreviousTaskInfo *TaskInfo
}

// ResumeWithOptions resumes a task with the given options
func (rm *ResumeManager) ResumeWithOptions(ctx context.Context, opts ResumeOptions) (*ResumeResult, error) {
	var taskID string
	var taskInfo *TaskInfo
	var err error

	// Determine which task to resume
	if opts.TaskID != "" {
		taskID = opts.TaskID
		taskInfo, err = rm.GetTaskInfo(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task info: %w", err)
		}
	} else {
		taskInfo, err = rm.GetMostRecentTask(ctx)
		if err != nil {
			return nil, err
		}
		if taskInfo == nil {
			return nil, fmt.Errorf("no recent tasks found to resume")
		}
		taskID = taskInfo.TaskID
	}

	// Validate task can be resumed
	if !taskInfo.IsResumable {
		return &ResumeResult{
			TaskID:           taskID,
			Success:          false,
			Message:          fmt.Sprintf("Task %s cannot be resumed (status: %s)", taskID, taskInfo.Status),
			PreviousTaskInfo: taskInfo,
		}, nil
	}

	// Resume the task
	resumedID, err := rm.ResumeTask(ctx, taskID, opts.Prompt, opts.Images)
	if err != nil {
		return nil, err
	}

	return &ResumeResult{
		TaskID:           resumedID,
		Success:          true,
		Message:          fmt.Sprintf("Successfully resumed task %s", resumedID),
		PreviousTaskInfo: taskInfo,
	}, nil
}

// CanResume checks if a task can be resumed without actually resuming it
func (rm *ResumeManager) CanResume(ctx context.Context, taskID string) (bool, string) {
	taskInfo, err := rm.GetTaskInfo(ctx, taskID)
	if err != nil {
		return false, fmt.Sprintf("Cannot get task info: %v", err)
	}

	if taskInfo == nil {
		return false, "Task not found"
	}

	if !taskInfo.IsResumable {
		return false, fmt.Sprintf("Task status '%s' does not allow resumption", taskInfo.Status)
	}

	return true, "Task can be resumed"
}
