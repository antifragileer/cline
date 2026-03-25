// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// TaskService implements the TaskService gRPC interface
type TaskService struct {
	cline.UnimplementedTaskServiceServer
	state          *storage.ClineFileStorage
	taskHistory    *storage.ClineFileStorage
	activeTasks    map[string]*ActiveTask
}

// ActiveTask represents a running task
type ActiveTask struct {
	ID        string
	Prompt    string
	Status    string
	StartTime time.Time
}

// NewTaskService creates a new TaskService instance
func NewTaskService(state, taskHistory *storage.ClineFileStorage) *TaskService {
	return &TaskService{
		state:          state,
		taskHistory:    taskHistory,
		activeTasks:    make(map[string]*ActiveTask),
	}
}

// CancelTask cancels the currently running task
func (s *TaskService) CancelTask(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Find and cancel active task
	for id, task := range s.activeTasks {
		if task.Status == "running" {
			task.Status = "cancelled"
			s.activeTasks[id] = task
			break
		}
	}
	return &cline.Empty{}, nil
}

// CancelBackgroundCommand cancels the background command
func (s *TaskService) CancelBackgroundCommand(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	return &cline.Empty{}, nil
}

// ClearTask clears the current task
func (s *TaskService) ClearTask(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	_ = s.state.Delete("current_task_id")
	return &cline.Empty{}, nil
}

// GetTotalTasksSize gets the total size of all tasks
func (s *TaskService) GetTotalTasksSize(ctx context.Context, req *cline.EmptyRequest) (*cline.Int64, error) {
	tasksDir := filepath.Join(getDataDir(), "tasks")
	
	var totalSize int64
	err := filepath.Walk(tasksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return &cline.Int64{Value: 0}, nil
	}
	
	return &cline.Int64{Value: totalSize}, nil
}

// DeleteTasksWithIds deletes multiple tasks with the given IDs
func (s *TaskService) DeleteTasksWithIds(ctx context.Context, req *cline.StringArrayRequest) (*cline.Empty, error) {
	for _, id := range req.Value {
		// Remove from active tasks
		delete(s.activeTasks, id)
		
		// Remove task directory
		taskDir := filepath.Join(getDataDir(), "tasks", id)
		_ = os.RemoveAll(taskDir)
	}
	
	// Update task history
	entries, _ := s.loadTaskHistory()
	var newEntries []map[string]interface{}
	for _, entry := range entries {
		if id, ok := entry["id"].(string); ok {
			found := false
			for _, delID := range req.Value {
				if id == delID {
					found = true
					break
				}
			}
			if !found {
				newEntries = append(newEntries, entry)
			}
		}
	}
	_ = s.saveTaskHistory(newEntries)
	
	return &cline.Empty{}, nil
}

// NewTask creates a new task
func (s *TaskService) NewTask(ctx context.Context, req *cline.NewTaskRequest) (*cline.String, error) {
	// Generate task ID
	taskID := generateTaskID()
	
	// Store task info
	task := &ActiveTask{
		ID:        taskID,
		Prompt:    req.GetText(),
		Status:    "running",
		StartTime: time.Now(),
	}
	s.activeTasks[taskID] = task
	
	// Save current task ID
	_ = s.state.Set("current_task_id", taskID)
	
	// Create task directory
	taskDir := filepath.Join(getDataDir(), "tasks", taskID)
	_ = os.MkdirAll(taskDir, 0755)
	
	// Add to history
	s.addToHistory(taskID, req.GetText())
	
	return &cline.String{Value: taskID}, nil
}

// ShowTaskWithId shows a task with the specified ID
func (s *TaskService) ShowTaskWithId(ctx context.Context, req *cline.StringRequest) (*cline.TaskResponse, error) {
	taskID := req.GetValue()
	
	// Load task from history
	entries, _ := s.loadTaskHistory()
	var taskText string
	var ts int64
	for _, entry := range entries {
		if id, ok := entry["id"].(string); ok && id == taskID {
			if t, ok := entry["task"].(string); ok {
				taskText = t
			}
			if t, ok := entry["ts"].(float64); ok {
				ts = int64(t)
			}
			break
		}
	}
	
	return &cline.TaskResponse{
		Id:   taskID,
		Task: taskText,
		Ts:   ts,
	}, nil
}

// ExportTaskWithId exports a task to markdown
func (s *TaskService) ExportTaskWithId(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	// Implementation would export task to markdown
	return &cline.Empty{}, nil
}

// ToggleTaskFavorite toggles the favorite status of a task
func (s *TaskService) ToggleTaskFavorite(ctx context.Context, req *cline.TaskFavoriteRequest) (*cline.Empty, error) {
	// Get current favorites
	var favorites []string
	if data, ok := s.state.Get("favorite_tasks"); ok && data != nil {
		if f, ok := data.([]interface{}); ok {
			for _, v := range f {
				if s, ok := v.(string); ok {
					favorites = append(favorites, s)
				}
			}
		}
	}
	
	// Toggle
	taskID := req.GetTaskId()
	found := false
	for i, f := range favorites {
		if f == taskID {
			favorites = append(favorites[:i], favorites[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		favorites = append(favorites, taskID)
	}
	
	_ = s.state.Set("favorite_tasks", favorites)
	return &cline.Empty{}, nil
}

// GetTaskHistory gets filtered task history
func (s *TaskService) GetTaskHistory(ctx context.Context, req *cline.GetTaskHistoryRequest) (*cline.TaskHistoryArray, error) {
	entries, err := s.loadTaskHistory()
	if err != nil {
		return &cline.TaskHistoryArray{Tasks: []*cline.TaskItem{}}, nil
	}
	
	// Apply search filter
	if req.SearchQuery != "" {
		search := strings.ToLower(req.SearchQuery)
		var filtered []map[string]interface{}
		for _, entry := range entries {
			if task, ok := entry["task"].(string); ok && strings.Contains(strings.ToLower(task), search) {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	
	// Convert to proto format
	var tasks []*cline.TaskItem
	for _, entry := range entries {
		item := &cline.TaskItem{}
		
		if id, ok := entry["id"].(string); ok {
			item.Id = id
		}
		if task, ok := entry["task"].(string); ok {
			item.Task = task
		}
		if ts, ok := entry["ts"].(float64); ok {
			item.Ts = int64(ts)
		}
		tasks = append(tasks, item)
	}
	
	return &cline.TaskHistoryArray{Tasks: tasks}, nil
}

// AskResponse sends a response to a previous ask operation
func (s *TaskService) AskResponse(ctx context.Context, req *cline.AskResponseRequest) (*cline.Empty, error) {
	// Store the response for the active task
	currentTaskID, _ := s.state.Get("current_task_id")
	if currentTaskID != nil {
		responses, _ := s.state.Get("pending_responses")
		var respMap map[string]interface{}
		if responses == nil {
			respMap = make(map[string]interface{})
		} else {
			respMap, _ = responses.(map[string]interface{})
		}
		respMap[currentTaskID.(string)] = map[string]interface{}{
			"type":    req.GetResponseType(),
			"text":    req.GetText(),
			"images":  req.GetImages(),
			"files":   req.GetFiles(),
		}
		_ = s.state.Set("pending_responses", respMap)
	}
	
	return &cline.Empty{}, nil
}

// TaskFeedback records task feedback
func (s *TaskService) TaskFeedback(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	// Store feedback
	feedback := req.GetValue()
	_ = s.state.Set("last_task_feedback", feedback)
	return &cline.Empty{}, nil
}

// TaskCompletionViewChanges shows task completion changes diff
func (s *TaskService) TaskCompletionViewChanges(ctx context.Context, req *cline.Int64Request) (*cline.Empty, error) {
	// Implementation would show diff
	return &cline.Empty{}, nil
}

// ExecuteQuickWin executes a quick win task
func (s *TaskService) ExecuteQuickWin(ctx context.Context, req *cline.ExecuteQuickWinRequest) (*cline.Empty, error) {
	// Execute the quick command
	return &cline.Empty{}, nil
}

// DeleteAllTaskHistory deletes all task history
func (s *TaskService) DeleteAllTaskHistory(ctx context.Context, req *cline.EmptyRequest) (*cline.DeleteAllTaskHistoryCount, error) {
	// Clear all tasks
	s.activeTasks = make(map[string]*ActiveTask)
	
	// Clear history file
	_ = s.saveTaskHistory([]map[string]interface{}{})
	
	// Remove task directories
	tasksDir := filepath.Join(getDataDir(), "tasks")
	entries, _ := os.ReadDir(tasksDir)
	count := int64(len(entries))
	
	for _, entry := range entries {
		_ = os.RemoveAll(filepath.Join(tasksDir, entry.Name()))
	}
	
	return &cline.DeleteAllTaskHistoryCount{TasksDeleted: int32(count)}, nil
}

// ExplainChanges explains changes with AI
func (s *TaskService) ExplainChanges(ctx context.Context, req *cline.ExplainChangesRequest) (*cline.Empty, error) {
	// Implementation would generate explanation
	return &cline.Empty{}, nil
}

// Helper methods

func (s *TaskService) loadTaskHistory() ([]map[string]interface{}, error) {
	val, ok := s.taskHistory.Get("entries")
	if !ok || val == nil {
		return []map[string]interface{}{}, nil
	}
	
	// Convert to []map[string]interface{}
	data, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	
	var entries []map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		ti, _ := entries[i]["ts"].(float64)
		tj, _ := entries[j]["ts"].(float64)
		return ti > tj
	})
	
	return entries, nil
}

func (s *TaskService) saveTaskHistory(entries []map[string]interface{}) error {
	return s.taskHistory.Set("entries", entries)
}

func (s *TaskService) addToHistory(taskID, task string) {
	entries, _ := s.loadTaskHistory()
	
	entry := map[string]interface{}{
		"id":   taskID,
		"task": task,
		"ts":   time.Now().UnixMilli(),
	}
	
	entries = append([]map[string]interface{}{entry}, entries...)
	_ = s.saveTaskHistory(entries)
}

func generateTaskID() string {
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}

func getDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cline", "data")
}