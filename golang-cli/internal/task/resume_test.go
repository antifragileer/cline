package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTaskStorage is a mock implementation of TaskStorage for testing
type MockTaskStorage struct {
	mu sync.RWMutex

	tasks    map[string]*TaskInfo
	history  map[string][]ConversationMessage
	getErr   error
	listErr  error
	updateErr error
}

// NewMockTaskStorage creates a new MockTaskStorage
func NewMockTaskStorage() *MockTaskStorage {
	return &MockTaskStorage{
		tasks:   make(map[string]*TaskInfo),
		history: make(map[string][]ConversationMessage),
	}
}

// GetTaskInfo retrieves task metadata by ID
func (m *MockTaskStorage) GetTaskInfo(taskID string) (*TaskInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getErr != nil {
		return nil, m.getErr
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetTaskHistory retrieves conversation history for a task
func (m *MockTaskStorage) GetTaskHistory(taskID string) ([]ConversationMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.history[taskID]
	if !exists {
		return []ConversationMessage{}, nil
	}

	return history, nil
}

// GetAllTasks returns all tasks sorted by timestamp
func (m *MockTaskStorage) GetAllTasks() ([]TaskInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listErr != nil {
		return nil, m.listErr
	}

	var tasks []TaskInfo
	for _, task := range m.tasks {
		tasks = append(tasks, *task)
	}

	// Sort by timestamp, newest first
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Timestamp > tasks[j].Timestamp
	})

	return tasks, nil
}

// UpdateTaskState updates the state of a task
func (m *MockTaskStorage) UpdateTaskState(taskID string, state TaskState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateErr != nil {
		return m.updateErr
	}

	if task, exists := m.tasks[taskID]; exists {
		task.State = state
		task.LastModified = time.Now().UnixMilli()
	}

	return nil
}

// AddTask adds a task to the mock storage
func (m *MockTaskStorage) AddTask(task *TaskInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
}

// AddHistory adds conversation history to the mock storage
func (m *MockTaskStorage) AddHistory(taskID string, history []ConversationMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history[taskID] = history
}

// SetGetError sets the error to return from GetTaskInfo
func (m *MockTaskStorage) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getErr = err
}

// SetListError sets the error to return from GetAllTasks
func (m *MockTaskStorage) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listErr = err
}

func TestTaskState_String(t *testing.T) {
	tests := []struct {
		state    TaskState
		expected string
	}{
		{TaskStateRunning, "running"},
		{TaskStatePaused, "paused"},
		{TaskStateCompleted, "completed"},
		{TaskStateFailed, "failed"},
		{TaskStateAborted, "aborted"},
		{TaskState("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.state))
		})
	}
}

func TestFileTaskStorage_NewFileTaskStorage(t *testing.T) {
	t.Run("creates storage with default directory", func(t *testing.T) {
		storage, err := NewFileTaskStorage("")
		require.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Contains(t, storage.storageDir, ".cline/data/tasks")
	})

	t.Run("creates storage with custom directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, tmpDir, storage.storageDir)
	})

	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "new", "tasks")
		storage, err := NewFileTaskStorage(newDir)
		require.NoError(t, err)
		assert.NotNil(t, storage)

		_, err = os.Stat(newDir)
		assert.NoError(t, err)
	})
}

func TestFileTaskStorage_GetTaskInfo(t *testing.T) {
	t.Run("returns error for empty task ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		_, err = storage.GetTaskInfo("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task ID is required")
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		_, err = storage.GetTaskInfo("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})

	t.Run("returns task info from metadata file", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "test-task-123"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		expectedInfo := &TaskInfo{
			ID:          taskID,
			Task:        "Test task",
			State:       TaskStatePaused,
			Timestamp:   time.Now().UnixMilli(),
			IsFavorited: false,
			TokensIn:    100,
			TokensOut:   50,
		}

		metaPath := filepath.Join(taskDir, "metadata.json")
		data, err := json.Marshal(expectedInfo)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(metaPath, data, 0644))

		info, err := storage.GetTaskInfo(taskID)
		require.NoError(t, err)
		assert.Equal(t, expectedInfo.ID, info.ID)
		assert.Equal(t, expectedInfo.Task, info.Task)
		assert.Equal(t, expectedInfo.State, info.State)
	})

	t.Run("constructs info from directory when metadata missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "test-task-456"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		// Create conversation history file
		historyPath := filepath.Join(taskDir, "conversation_history.json")
		require.NoError(t, os.WriteFile(historyPath, []byte("[]"), 0644))

		info, err := storage.GetTaskInfo(taskID)
		require.NoError(t, err)
		assert.Equal(t, taskID, info.ID)
		assert.Equal(t, TaskStatePaused, info.State)
	})
}

func TestFileTaskStorage_GetTaskHistory(t *testing.T) {
	t.Run("returns empty history for new task", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "test-task"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		history, err := storage.GetTaskHistory(taskID)
		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("returns conversation history", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "test-task"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		expectedHistory := []ConversationMessage{
			{Timestamp: time.Now().UnixMilli(), Type: "user", Content: "Hello"},
			{Timestamp: time.Now().UnixMilli(), Type: "assistant", Content: "Hi there"},
		}

		data, err := json.Marshal(expectedHistory)
		require.NoError(t, err)

		historyPath := filepath.Join(taskDir, "conversation_history.json")
		require.NoError(t, os.WriteFile(historyPath, data, 0644))

		history, err := storage.GetTaskHistory(taskID)
		require.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, "Hello", history[0].Content)
		assert.Equal(t, "Hi there", history[1].Content)
	})

	t.Run("returns error for empty task ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		_, err = storage.GetTaskHistory("")
		assert.Error(t, err)
	})
}

func TestFileTaskStorage_GetAllTasks(t *testing.T) {
	t.Run("returns empty list for new storage", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		tasks, err := storage.GetAllTasks()
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("returns tasks sorted by timestamp", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		// Create multiple tasks
		for i := 0; i < 3; i++ {
			taskID := fmt.Sprintf("task-%d", i)
			taskDir := filepath.Join(tmpDir, taskID)
			require.NoError(t, os.MkdirAll(taskDir, 0755))

			info := &TaskInfo{
				ID:        taskID,
				Timestamp: time.Now().Add(time.Duration(i) * time.Hour).UnixMilli(),
			}

			data, err := json.Marshal(info)
			require.NoError(t, err)

			metaPath := filepath.Join(taskDir, "metadata.json")
			require.NoError(t, os.WriteFile(metaPath, data, 0644))
		}

		tasks, err := storage.GetAllTasks()
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Should be sorted newest first
		for i := 0; i < len(tasks)-1; i++ {
			assert.GreaterOrEqual(t, tasks[i].Timestamp, tasks[i+1].Timestamp)
		}
	})
}

func TestFileTaskStorage_UpdateTaskState(t *testing.T) {
	t.Run("updates existing task state", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "test-task"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		info := &TaskInfo{
			ID:        taskID,
			State:     TaskStatePaused,
			Timestamp: time.Now().UnixMilli(),
		}

		data, err := json.Marshal(info)
		require.NoError(t, err)

		metaPath := filepath.Join(taskDir, "metadata.json")
		require.NoError(t, os.WriteFile(metaPath, data, 0644))

		err = storage.UpdateTaskState(taskID, TaskStateRunning)
		require.NoError(t, err)

		updatedInfo, err := storage.GetTaskInfo(taskID)
		require.NoError(t, err)
		assert.Equal(t, TaskStateRunning, updatedInfo.State)
	})

	t.Run("creates new metadata for non-existent task", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		taskID := "new-task"
		taskDir := filepath.Join(tmpDir, taskID)
		require.NoError(t, os.MkdirAll(taskDir, 0755))

		err = storage.UpdateTaskState(taskID, TaskStateRunning)
		require.NoError(t, err)

		info, err := storage.GetTaskInfo(taskID)
		require.NoError(t, err)
		assert.Equal(t, TaskStateRunning, info.State)
	})

	t.Run("returns error for empty task ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, err := NewFileTaskStorage(tmpDir)
		require.NoError(t, err)

		err = storage.UpdateTaskState("", TaskStateRunning)
		assert.Error(t, err)
	})
}

func TestSanitizeTaskID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple-task", "simple-task"},
		{"task/with/slashes", "task_with_slashes"},
		{"task\\with\\backslashes", "task_with_backslashes"},
		{"task:with:colons", "task_with_colons"},
		{"task*with*asterisks", "task_with_asterisks"},
		{"task?with?questions", "task_with_questions"},
		{`task"with"quotes`, "task_with_quotes"},
		{"task<with>brackets", "task_with_brackets"},
		{"task|with|pipes", "task_with_pipes"},
		{"task..with..dots", "task_with_dots"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeTaskID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewResumer(t *testing.T) {
	mockStorage := NewMockTaskStorage()
	resumer := NewResumer(mockStorage, nil)

	assert.NotNil(t, resumer)
	assert.NotNil(t, resumer.storage)
	assert.NotNil(t, resumer.messageStore)
}

func TestResumer_validateOptions(t *testing.T) {
	mockStorage := NewMockTaskStorage()
	resumer := NewResumer(mockStorage, nil)

	t.Run("validates task ID or continue flag required", func(t *testing.T) {
		opts := ResumeOptions{}
		err := resumer.validateOptions(&opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "TaskID or ContinueFlag")
	})

	t.Run("accepts task ID", func(t *testing.T) {
		opts := ResumeOptions{TaskID: "test-task"}
		err := resumer.validateOptions(&opts)
		assert.NoError(t, err)
	})

	t.Run("accepts continue flag", func(t *testing.T) {
		opts := ResumeOptions{ContinueFlag: true}
		err := resumer.validateOptions(&opts)
		assert.NoError(t, err)
	})

	t.Run("sets default timeout", func(t *testing.T) {
		opts := ResumeOptions{TaskID: "test"}
		err := resumer.validateOptions(&opts)
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, opts.Timeout)
	})

	t.Run("preserves custom timeout", func(t *testing.T) {
		opts := ResumeOptions{
			TaskID:  "test",
			Timeout: 5 * time.Minute,
		}
		err := resumer.validateOptions(&opts)
		require.NoError(t, err)
		assert.Equal(t, 5*time.Minute, opts.Timeout)
	})
}

func TestResumer_resolveTaskID(t *testing.T) {
	mockStorage := NewMockTaskStorage()

	// Add test tasks
	now := time.Now()
	mockStorage.AddTask(&TaskInfo{
		ID:        "recent-task",
		State:     TaskStatePaused,
		Timestamp: now.UnixMilli(),
	})
	mockStorage.AddTask(&TaskInfo{
		ID:        "older-task",
		State:     TaskStateCompleted,
		Timestamp: now.Add(-1 * time.Hour).UnixMilli(),
	})
	mockStorage.AddTask(&TaskInfo{
		ID:        "running-task",
		State:     TaskStateRunning,
		Timestamp: now.Add(-30 * time.Minute).UnixMilli(),
	})

	resumer := NewResumer(mockStorage, nil)

	t.Run("returns specified task ID", func(t *testing.T) {
		opts := ResumeOptions{TaskID: "specific-task"}
		taskID, err := resumer.resolveTaskID(opts)
		require.NoError(t, err)
		assert.Equal(t, "specific-task", taskID)
	})

	t.Run("returns most recent paused/running task with continue flag", func(t *testing.T) {
		opts := ResumeOptions{ContinueFlag: true}
		taskID, err := resumer.resolveTaskID(opts)
		require.NoError(t, err)
		// Should return recent-task (paused) over running-task
		assert.Equal(t, "recent-task", taskID)
	})

	t.Run("returns most recent task when no paused/running tasks", func(t *testing.T) {
		// Remove paused task
		delete(mockStorage.tasks, "recent-task")
		delete(mockStorage.tasks, "running-task")

		opts := ResumeOptions{ContinueFlag: true}
		taskID, err := resumer.resolveTaskID(opts)
		require.NoError(t, err)
		assert.Equal(t, "older-task", taskID)
	})

	t.Run("returns error when no tasks exist", func(t *testing.T) {
		emptyStorage := NewMockTaskStorage()
		emptyResumer := NewResumer(emptyStorage, nil)

		opts := ResumeOptions{ContinueFlag: true}
		_, err := emptyResumer.resolveTaskID(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks found")
	})
}

func TestResumer_loadAndValidateTask(t *testing.T) {
	mockStorage := NewMockTaskStorage()

	now := time.Now()
	mockStorage.AddTask(&TaskInfo{
		ID:        "paused-task",
		State:     TaskStatePaused,
		Timestamp: now.UnixMilli(),
	})
	mockStorage.AddTask(&TaskInfo{
		ID:        "running-task",
		State:     TaskStateRunning,
		Timestamp: now.UnixMilli(),
	})
	mockStorage.AddTask(&TaskInfo{
		ID:        "completed-task",
		State:     TaskStateCompleted,
		Timestamp: now.UnixMilli(),
	})
	mockStorage.AddTask(&TaskInfo{
		ID:        "failed-task",
		State:     TaskStateFailed,
		Timestamp: now.UnixMilli(),
	})

	resumer := NewResumer(mockStorage, nil)

	t.Run("loads paused task", func(t *testing.T) {
		info, err := resumer.loadAndValidateTask("paused-task")
		require.NoError(t, err)
		assert.Equal(t, TaskStatePaused, info.State)
	})

	t.Run("loads running task", func(t *testing.T) {
		info, err := resumer.loadAndValidateTask("running-task")
		require.NoError(t, err)
		assert.Equal(t, TaskStateRunning, info.State)
	})

	t.Run("rejects completed task", func(t *testing.T) {
		_, err := resumer.loadAndValidateTask("completed-task")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already completed")
	})

	t.Run("allows failed task", func(t *testing.T) {
		info, err := resumer.loadAndValidateTask("failed-task")
		require.NoError(t, err)
		assert.Equal(t, TaskStateFailed, info.State)
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		_, err := resumer.loadAndValidateTask("non-existent")
		assert.Error(t, err)
	})
}

func TestResumer_Resume(t *testing.T) {
	t.Run("successfully resumes task by ID", func(t *testing.T) {
		mockStorage := NewMockTaskStorage()

		taskID := "test-task"
		mockStorage.AddTask(&TaskInfo{
			ID:        taskID,
			Task:      "Test task description",
			State:     TaskStatePaused,
			Timestamp: time.Now().UnixMilli(),
		})

		history := []ConversationMessage{
			{Timestamp: time.Now().UnixMilli(), Type: "user", Content: "Hello"},
			{Timestamp: time.Now().UnixMilli(), Type: "assistant", Content: "Hi"},
		}
		mockStorage.AddHistory(taskID, history)

		resumer := NewResumer(mockStorage, nil)

		result, err := resumer.Resume(context.Background(), ResumeOptions{
			TaskID:  taskID,
			Message: "Continuing...",
		})

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, taskID, result.TaskID)
		assert.Equal(t, TaskStateRunning, result.State)
		assert.Len(t, result.Messages, 2)
	})

	t.Run("successfully resumes with continue flag", func(t *testing.T) {
		mockStorage := NewMockTaskStorage()

		taskID := "recent-task"
		mockStorage.AddTask(&TaskInfo{
			ID:        taskID,
			State:     TaskStatePaused,
			Timestamp: time.Now().UnixMilli(),
		})
		mockStorage.AddHistory(taskID, []ConversationMessage{})

		resumer := NewResumer(mockStorage, nil)

		result, err := resumer.Resume(context.Background(), ResumeOptions{
			ContinueFlag: true,
		})

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, taskID, result.TaskID)
	})

	t.Run("returns error result for completed task", func(t *testing.T) {
		mockStorage := NewMockTaskStorage()

		taskID := "completed-task"
		mockStorage.AddTask(&TaskInfo{
			ID:        taskID,
			State:     TaskStateCompleted,
			Timestamp: time.Now().UnixMilli(),
		})

		resumer := NewResumer(mockStorage, nil)

		result, err := resumer.Resume(context.Background(), ResumeOptions{
			TaskID: taskID,
		})

		// Should not return error, but result should indicate failure
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "already completed")
	})

	t.Run("returns error for invalid options", func(t *testing.T) {
		mockStorage := NewMockTaskStorage()
		resumer := NewResumer(mockStorage, nil)

		_, err := resumer.Resume(context.Background(), ResumeOptions{})
		assert.Error(t, err)
	})

	t.Run("handles storage errors gracefully", func(t *testing.T) {
		mockStorage := NewMockTaskStorage()
		mockStorage.SetGetError(errors.New("storage error"))

		resumer := NewResumer(mockStorage, nil)

		result, err := resumer.Resume(context.Background(), ResumeOptions{
			TaskID: "test-task",
		})

		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "storage error")
	})
}

func TestResumer_restoreUIConversation(t *testing.T) {
	mockStorage := NewMockTaskStorage()
	resumer := NewResumer(mockStorage, nil)

	history := []ConversationMessage{
		{Timestamp: time.Now().UnixMilli(), Type: "user", Content: "User message", Metadata: map[string]interface{}{"key": "value"}},
		{Timestamp: time.Now().UnixMilli(), Type: "assistant", Content: "Assistant response"},
		{Timestamp: time.Now().UnixMilli(), Type: "error", Content: "Error message"},
		{Timestamp: time.Now().UnixMilli(), Type: "unknown", Content: "Unknown type"},
	}

	resumer.restoreUIConversation(history)

	store := resumer.GetMessageStore()
	assert.Equal(t, 4, store.Len())

	// Check first message
	msg, ok := store.Get(0)
	require.True(t, ok)
	assert.Equal(t, "User message", msg.Content)

	// Check assistant message
	msg, ok = store.Get(1)
	require.True(t, ok)
	assert.Equal(t, "Assistant response", msg.Content)

	// Check error message
	msg, ok = store.Get(2)
	require.True(t, ok)
	assert.Equal(t, "Error message", msg.Content)
}

func TestResumer_CanResume(t *testing.T) {
	mockStorage := NewMockTaskStorage()

	mockStorage.AddTask(&TaskInfo{
		ID:    "paused-task",
		State: TaskStatePaused,
	})
	mockStorage.AddTask(&TaskInfo{
		ID:    "running-task",
		State: TaskStateRunning,
	})
	mockStorage.AddTask(&TaskInfo{
		ID:    "completed-task",
		State: TaskStateCompleted,
	})
	mockStorage.AddTask(&TaskInfo{
		ID:    "failed-task",
		State: TaskStateFailed,
	})

	resumer := NewResumer(mockStorage, nil)

	tests := []struct {
		taskID   string
		expected bool
	}{
		{"paused-task", true},
		{"running-task", true},
		{"completed-task", false},
		{"failed-task", true},
		{"non-existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.taskID, func(t *testing.T) {
			canResume, err := resumer.CanResume(tt.taskID)
			if tt.taskID == "non-existent" {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, canResume)
		})
	}
}

func TestResumer_GetRecentTasks(t *testing.T) {
	mockStorage := NewMockTaskStorage()

	now := time.Now()
	for i := 0; i < 5; i++ {
		// Add tasks in reverse chronological order so task-0 is newest
		mockStorage.AddTask(&TaskInfo{
			ID:        fmt.Sprintf("task-%d", i),
			Timestamp: now.Add(time.Duration(5-i) * time.Hour).UnixMilli(),
		})
	}

	resumer := NewResumer(mockStorage, nil)

	t.Run("returns all tasks when limit is 0", func(t *testing.T) {
		tasks, err := resumer.GetRecentTasks(0)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
	})

	t.Run("returns limited number of tasks", func(t *testing.T) {
		tasks, err := resumer.GetRecentTasks(3)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})

	t.Run("returns tasks sorted by timestamp", func(t *testing.T) {
		tasks, err := resumer.GetRecentTasks(5)
		require.NoError(t, err)
		for i := 0; i < len(tasks)-1; i++ {
			assert.GreaterOrEqual(t, tasks[i].Timestamp, tasks[i+1].Timestamp)
		}
	})
}

func TestResumer_concurrentAccess(t *testing.T) {
	mockStorage := NewMockTaskStorage()
	mockStorage.AddTask(&TaskInfo{
		ID:        "concurrent-task",
		State:     TaskStatePaused,
		Timestamp: time.Now().UnixMilli(),
	})
	mockStorage.AddHistory("concurrent-task", []ConversationMessage{
		{Type: "user", Content: "Test"},
	})

	resumer := NewResumer(mockStorage, nil)

	// Run multiple concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = resumer.CanResume("concurrent-task")
			_, _ = resumer.GetRecentTasks(5)
		}()
	}

	// Concurrent resume operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = resumer.Resume(context.Background(), ResumeOptions{
				TaskID: "concurrent-task",
			})
		}()
	}

	wg.Wait()
}

func TestResumeOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    ResumeOptions
		wantErr bool
	}{
		{
			name:    "empty options",
			opts:    ResumeOptions{},
			wantErr: true,
		},
		{
			name: "valid with task ID",
			opts: ResumeOptions{
				TaskID: "test-task",
			},
			wantErr: false,
		},
		{
			name: "valid with continue flag",
			opts: ResumeOptions{
				ContinueFlag: true,
			},
			wantErr: false,
		},
		{
			name: "valid with both",
			opts: ResumeOptions{
				TaskID:       "test-task",
				ContinueFlag: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockTaskStorage()
			resumer := NewResumer(mockStorage, nil)

			err := resumer.validateOptions(&tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConversationMessage_Structure(t *testing.T) {
	msg := ConversationMessage{
		Timestamp: time.Now().UnixMilli(),
		Type:      "user",
		Content:   "Test message",
		Metadata:  map[string]interface{}{"key": "value"},
		Partial:   false,
		ToolName:  "test_tool",
		ToolInput: map[string]interface{}{"arg": "value"},
	}

	assert.NotZero(t, msg.Timestamp)
	assert.Equal(t, "user", msg.Type)
	assert.Equal(t, "Test message", msg.Content)
	assert.False(t, msg.Partial)
	assert.Equal(t, "test_tool", msg.ToolName)
}

func TestCreateStorageContext(t *testing.T) {
	tmpDir := t.TempDir()

	ctx, err := CreateStorageContext(tmpDir, "test-workspace")
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify the context was created by closing it
	err = ctx.Close()
	assert.NoError(t, err)
}

// Integration-style test for the complete resume flow
func TestResumer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tmpDir := t.TempDir()
	storage, err := NewFileTaskStorage(tmpDir)
	require.NoError(t, err)

	// Create a task with history
	taskID := "integration-task"
	taskDir := filepath.Join(tmpDir, taskID)
	require.NoError(t, os.MkdirAll(taskDir, 0755))

	// Create metadata
	info := &TaskInfo{
		ID:          taskID,
		Task:        "Integration test task",
		State:       TaskStatePaused,
		Timestamp:   time.Now().UnixMilli(),
		TokensIn:    100,
		TokensOut:   50,
		TotalCost:   0.001,
		IsFavorited: false,
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "metadata.json"), data, 0644))

	// Create conversation history
	history := []ConversationMessage{
		{Timestamp: time.Now().UnixMilli(), Type: "user", Content: "Start the task"},
		{Timestamp: time.Now().UnixMilli(), Type: "assistant", Content: "I'll help you"},
		{Timestamp: time.Now().UnixMilli(), Type: "user", Content: "Do something"},
	}
	historyData, err := json.Marshal(history)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "conversation_history.json"), historyData, 0644))

	// Create resumer and resume the task
	resumer := NewResumer(storage, nil)

	result, err := resumer.Resume(context.Background(), ResumeOptions{
		TaskID:  taskID,
		Message: "Continuing the task",
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, taskID, result.TaskID)
	assert.Equal(t, TaskStateRunning, result.State)
	assert.Len(t, result.Messages, 3)

	// Verify task state was updated
	updatedInfo, err := storage.GetTaskInfo(taskID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateRunning, updatedInfo.State)

	// Verify UI conversation was restored
	msgStore := resumer.GetMessageStore()
	assert.Equal(t, 3, msgStore.Len())
}