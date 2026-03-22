// Package task provides task management functionality for the Cline CLI.
// This package implements task creation, resumption, and conversation management
// with support for loading task state from storage and establishing gRPC streams.
package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/cline/cline/golang-cli/internal/tui"
)

// TaskState represents the current state of a task
type TaskState string

const (
	// TaskStateRunning indicates the task is currently running
	TaskStateRunning TaskState = "running"
	// TaskStatePaused indicates the task is paused and can be resumed
	TaskStatePaused TaskState = "paused"
	// TaskStateCompleted indicates the task has completed
	TaskStateCompleted TaskState = "completed"
	// TaskStateFailed indicates the task has failed
	TaskStateFailed TaskState = "failed"
	// TaskStateAborted indicates the task was aborted
	TaskStateAborted TaskState = "aborted"
)

// TaskInfo holds metadata about a task
type TaskInfo struct {
	ID           string    `json:"id"`
	Task         string    `json:"task"`
	Timestamp    int64     `json:"ts"`
	State        TaskState `json:"state"`
	IsFavorited  bool      `json:"is_favorited"`
	Size         int64     `json:"size"`
	TotalCost    float64   `json:"total_cost"`
	TokensIn     int32     `json:"tokens_in"`
	TokensOut    int32     `json:"tokens_out"`
	CacheWrites  int32     `json:"cache_writes"`
	CacheReads   int32     `json:"cache_reads"`
	ModelID      string    `json:"model_id"`
	LastModified int64     `json:"last_modified"`
}


// TaskStorage defines the interface for task storage operations
type TaskStorage interface {
	// GetTaskInfo retrieves task metadata by ID
	GetTaskInfo(taskID string) (*TaskInfo, error)
	// GetTaskHistory retrieves conversation history for a task
	GetTaskHistory(taskID string) ([]ConversationMessage, error)
	// GetAllTasks returns all tasks sorted by timestamp
	GetAllTasks() ([]TaskInfo, error)
	// UpdateTaskState updates the state of a task
	UpdateTaskState(taskID string, state TaskState) error
}

// FileTaskStorage implements TaskStorage using file-based storage
type FileTaskStorage struct {
	storageDir string
	mu         sync.RWMutex
}

// NewFileTaskStorage creates a new FileTaskStorage instance
func NewFileTaskStorage(baseDir string) (*FileTaskStorage, error) {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".cline", "data", "tasks")
	}

	// Ensure tasks directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	return &FileTaskStorage{
		storageDir: baseDir,
	}, nil
}

// GetTaskInfo retrieves task metadata from storage
func (fs *FileTaskStorage) GetTaskInfo(taskID string) (*TaskInfo, error) {
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Sanitize task ID for filesystem safety
	safeID := sanitizeTaskID(taskID)
	taskDir := filepath.Join(fs.storageDir, safeID)

	// Check if task directory exists
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Read task info from metadata file
	metaPath := filepath.Join(taskDir, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to construct basic info from directory
			return fs.constructTaskInfoFromDir(taskID, taskDir)
		}
		return nil, fmt.Errorf("failed to read task metadata: %w", err)
	}

	var info TaskInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse task metadata: %w", err)
	}

	// Ensure ID is set
	if info.ID == "" {
		info.ID = taskID
	}

	return &info, nil
}

// constructTaskInfoFromDir attempts to construct task info from directory contents
func (fs *FileTaskStorage) constructTaskInfoFromDir(taskID, taskDir string) (*TaskInfo, error) {
	// Get directory info
	stat, err := os.Stat(taskDir)
	if err != nil {
		return nil, err
	}

	// Look for conversation history file
	historyPath := filepath.Join(taskDir, "conversation_history.json")
	var size int64
	if info, err := os.Stat(historyPath); err == nil {
		size = info.Size()
	}

	return &TaskInfo{
		ID:           taskID,
		State:        TaskStatePaused,
		Timestamp:    stat.ModTime().UnixMilli(),
		LastModified: stat.ModTime().UnixMilli(),
		Size:         size,
	}, nil
}

// GetTaskHistory retrieves conversation history for a task
func (fs *FileTaskStorage) GetTaskHistory(taskID string) ([]ConversationMessage, error) {
	if taskID == "" {
		return nil, errors.New("task ID is required")
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	safeID := sanitizeTaskID(taskID)
	historyPath := filepath.Join(fs.storageDir, safeID, "conversation_history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConversationMessage{}, nil
		}
		return nil, fmt.Errorf("failed to read conversation history: %w", err)
	}

	var history []ConversationMessage
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse conversation history: %w", err)
	}

	return history, nil
}

// GetAllTasks returns all tasks sorted by timestamp (newest first)
func (fs *FileTaskStorage) GetAllTasks() ([]TaskInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var tasks []TaskInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		info, err := fs.GetTaskInfo(taskID)
		if err != nil {
			// Skip invalid tasks but continue processing others
			continue
		}
		tasks = append(tasks, *info)
	}

	// Sort by timestamp, newest first
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Timestamp > tasks[j].Timestamp
	})

	return tasks, nil
}

// UpdateTaskState updates the state of a task
func (fs *FileTaskStorage) UpdateTaskState(taskID string, state TaskState) error {
	if taskID == "" {
		return errors.New("task ID is required")
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	safeID := sanitizeTaskID(taskID)
	taskDir := filepath.Join(fs.storageDir, safeID)
	metaPath := filepath.Join(taskDir, "metadata.json")

	// Read existing metadata
	var info TaskInfo
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read task metadata: %w", err)
		}
		// Create new metadata
		info = TaskInfo{
			ID:        taskID,
			Timestamp: time.Now().UnixMilli(),
		}
	} else {
		if err := json.Unmarshal(data, &info); err != nil {
			return fmt.Errorf("failed to parse task metadata: %w", err)
		}
	}

	// Update state
	info.State = state
	info.LastModified = time.Now().UnixMilli()

	// Write back
	updatedData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write task metadata: %w", err)
	}

	return nil
}

// sanitizeTaskID sanitizes a task ID for safe filesystem usage
func sanitizeTaskID(taskID string) string {
	// Remove or replace potentially dangerous characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", ".."}
	result := taskID
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// ResumeOptions contains options for resuming a task
type ResumeOptions struct {
	// TaskID is the ID of the task to resume
	TaskID string
	// Message is an optional message to send when resuming
	Message string
	// ContinueFlag indicates whether to resume the most recent task
	ContinueFlag bool
	// Timeout for the resume operation
	Timeout time.Duration
}

// ResumeResult contains the result of a task resumption
type ResumeResult struct {
	// TaskID is the ID of the resumed task
	TaskID string
	// State is the current state of the task
	State TaskState
	// Messages loaded from conversation history
	Messages []ConversationMessage
	// Success indicates whether the resume was successful
	Success bool
	// Error message if resume failed
	Error string
}

// Resumer handles task resumption operations
type Resumer struct {
	storage    TaskStorage
	grpcClient *host.Client
	messageStore *tui.MessageStore
	mu         sync.RWMutex
}

// NewResumer creates a new Resumer instance
func NewResumer(storage TaskStorage, grpcClient *host.Client) *Resumer {
	return &Resumer{
		storage:      storage,
		grpcClient:   grpcClient,
		messageStore: tui.NewMessageStore(),
	}
}

// Resume resumes a task based on the provided options
func (r *Resumer) Resume(ctx context.Context, opts ResumeOptions) (*ResumeResult, error) {
	// Validate options
	if err := r.validateOptions(&opts); err != nil {
		return nil, err
	}

	// Determine task ID to resume
	taskID, err := r.resolveTaskID(opts)
	if err != nil {
		return nil, err
	}

	// Load task info and validate state
	taskInfo, err := r.loadAndValidateTask(taskID)
	if err != nil {
		return &ResumeResult{
			TaskID:  taskID,
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Load conversation history
	history, err := r.storage.GetTaskHistory(taskID)
	if err != nil {
		return &ResumeResult{
			TaskID:  taskID,
			State:   taskInfo.State,
			Success: false,
			Error:   fmt.Sprintf("failed to load conversation history: %v", err),
		}, nil
	}

	// Establish gRPC stream if client is available
	if r.grpcClient != nil {
		if err := r.establishStream(ctx, taskID, opts.Message); err != nil {
			return &ResumeResult{
				TaskID:   taskID,
				State:    taskInfo.State,
				Messages: history,
				Success:  false,
				Error:    fmt.Sprintf("failed to establish stream: %v", err),
			}, nil
		}
	}

	// Update task state to running
	if err := r.storage.UpdateTaskState(taskID, TaskStateRunning); err != nil {
		// Log but don't fail - the task is already resumed
	}

	// Restore UI conversation
	r.restoreUIConversation(history)

	return &ResumeResult{
		TaskID:   taskID,
		State:    TaskStateRunning,
		Messages: history,
		Success:  true,
	}, nil
}

// validateOptions validates resume options
func (r *Resumer) validateOptions(opts *ResumeOptions) error {
	if opts.TaskID == "" && !opts.ContinueFlag {
		return errors.New("either TaskID or ContinueFlag must be specified")
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}

	return nil
}

// resolveTaskID resolves the task ID to resume
func (r *Resumer) resolveTaskID(opts ResumeOptions) (string, error) {
	// If continue flag is set, find the most recent task
	if opts.ContinueFlag {
		tasks, err := r.storage.GetAllTasks()
		if err != nil {
			return "", fmt.Errorf("failed to get tasks: %w", err)
		}

		if len(tasks) == 0 {
			return "", errors.New("no tasks found to continue")
		}

		// Find the most recent task that can be resumed
		for _, task := range tasks {
			if task.State == TaskStatePaused || task.State == TaskStateRunning {
				return task.ID, nil
			}
		}

		// If no paused/running task, return the most recent
		return tasks[0].ID, nil
	}

	// Use specified task ID
	return opts.TaskID, nil
}

// loadAndValidateTask loads task info and validates it can be resumed
func (r *Resumer) loadAndValidateTask(taskID string) (*TaskInfo, error) {
	info, err := r.storage.GetTaskInfo(taskID)
	if err != nil {
		return nil, err
	}

	// Validate task state
	switch info.State {
	case TaskStateRunning:
		// Task is already running, warn but allow
	case TaskStatePaused:
		// Ideal state for resumption
	case TaskStateCompleted:
		return nil, errors.New("task is already completed")
	case TaskStateFailed:
		// Allow resuming failed tasks
	case TaskStateAborted:
		// Allow resuming aborted tasks
	default:
		// Unknown state, proceed with caution
	}

	return info, nil
}

// establishStream establishes a gRPC stream for the resumed task
func (r *Resumer) establishStream(ctx context.Context, taskID, message string) error {
	if r.grpcClient == nil {
		return nil // No gRPC client, skip stream establishment
	}

	// Wait for client to be ready
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := r.grpcClient.WaitForReady(ctx); err != nil {
		return fmt.Errorf("gRPC client not ready: %w", err)
	}

	// TODO: Implement actual gRPC stream establishment
	// This would involve calling the appropriate gRPC method
	// to resume the task with the optional message

	return nil
}

// restoreUIConversation restores the conversation in the UI
func (r *Resumer) restoreUIConversation(history []ConversationMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messageStore.Clear()

	for _, msg := range history {
		var tuiMsg *tui.Message

		switch msg.Type {
		case "user":
			tuiMsg = tui.NewUserMessage(msg.Content)
		case "assistant":
			tuiMsg = tui.NewAIMessage(msg.Content)
		case "error":
			tuiMsg = tui.NewErrorMessage(msg.Content)
		default:
			tuiMsg = tui.NewMessage(tui.MessageTypeSay, msg.Content)
		}

		tuiMsg.Partial = msg.Partial
		if msg.Timestamp > 0 {
			tuiMsg.Timestamp = time.UnixMilli(msg.Timestamp)
		}

		// Add metadata
		for k, v := range msg.Metadata {
			tuiMsg.SetMetadata(k, v)
		}

		r.messageStore.Add(tuiMsg)
	}
}

// GetMessageStore returns the message store for UI access
func (r *Resumer) GetMessageStore() *tui.MessageStore {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.messageStore
}

// CanResume checks if a task can be resumed
func (r *Resumer) CanResume(taskID string) (bool, error) {
	info, err := r.storage.GetTaskInfo(taskID)
	if err != nil {
		return false, err
	}

	switch info.State {
	case TaskStateRunning, TaskStatePaused, TaskStateFailed, TaskStateAborted:
		return true, nil
	case TaskStateCompleted:
		return false, nil
	default:
		return true, nil
	}
}

// GetRecentTasks returns the most recent tasks for selection
func (r *Resumer) GetRecentTasks(limit int) ([]TaskInfo, error) {
	tasks, err := r.storage.GetAllTasks()
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(tasks) > limit {
		tasks = tasks[:limit]
	}

	return tasks, nil
}

// CreateStorageContext creates a storage context from the resumer's storage
func CreateStorageContext(baseDir string, workspaceHash string) (*storage.StorageContext, error) {
	return storage.NewStorageContext(baseDir, workspaceHash)
}