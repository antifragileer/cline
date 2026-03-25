package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

func TestHistoryItem(t *testing.T) {
	item := HistoryItem{
		TaskID:    "test-task-123",
		Prompt:    "test prompt",
		Mode:      TaskModeAct,
		Cwd:       "/tmp",
		Model:     "claude-sonnet-4",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Status:    "initialized",
	}

	if item.TaskID != "test-task-123" {
		t.Errorf("Expected TaskID 'test-task-123', got %s", item.TaskID)
	}
}

func TestGetTaskHistory(t *testing.T) {
	// Create temporary storage
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	// Test empty history
	history, err := GetTaskHistory(storageCtx, 0)
	if err != nil {
		t.Errorf("Expected no error for empty history, got: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d items", len(history))
	}

	// Add some history entries
	testHistory := []HistoryItem{
		{
			TaskID:    "task-1",
			Prompt:    "prompt 1",
			Mode:      TaskModeAct,
			Cwd:       "/tmp",
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			Status:    "completed",
		},
		{
			TaskID:    "task-2",
			Prompt:    "prompt 2",
			Mode:      TaskModePlan,
			Cwd:       "/home",
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			Status:    "running",
		},
	}

	// Save history
	err = storageCtx.GlobalState.Set("taskHistory", testHistory)
	if err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Load history
	history, err = GetTaskHistory(storageCtx, 0)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 history items, got %d", len(history))
	}

	// Test with limit
	limited, err := GetTaskHistory(storageCtx, 1)
	if err != nil {
		t.Fatalf("Failed to get limited history: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("Expected 1 item with limit, got %d", len(limited))
	}
}

func TestGetTaskHistory_NilStorage(t *testing.T) {
	_, err := GetTaskHistory(nil, 0)
	if err == nil {
		t.Error("Expected error for nil storage, got nil")
	}
}

func TestFindMostRecentTask(t *testing.T) {
	// Create temporary storage
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	// Test empty history
	_, err = FindMostRecentTask(storageCtx, "")
	if err == nil {
		t.Error("Expected error for empty history, got nil")
	}

	// Add history with different updated times
	now := time.Now().Unix()
	testHistory := []HistoryItem{
		{
			TaskID:    "task-old",
			Prompt:    "old prompt",
			Mode:      TaskModeAct,
			Cwd:       "/tmp",
			CreatedAt: now - 100,
			UpdatedAt: now - 100,
			Status:    "completed",
		},
		{
			TaskID:    "task-new",
			Prompt:    "new prompt",
			Mode:      TaskModePlan,
			Cwd:       "/tmp",
			CreatedAt: now - 50,
			UpdatedAt: now,
			Status:    "running",
		},
	}

	err = storageCtx.GlobalState.Set("taskHistory", testHistory)
	if err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Find most recent task
	recent, err := FindMostRecentTask(storageCtx, "/tmp")
	if err != nil {
		t.Fatalf("Failed to find recent task: %v", err)
	}

	if recent.TaskID != "task-new" {
		t.Errorf("Expected 'task-new', got %s", recent.TaskID)
	}
}

func TestFindMostRecentTask_NilStorage(t *testing.T) {
	_, err := FindMostRecentTask(nil, "")
	if err == nil {
		t.Error("Expected error for nil storage, got nil")
	}
}

func TestUpdateTaskHistory(t *testing.T) {
	// Create temporary storage
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	// Add initial history
	testHistory := []HistoryItem{
		{
			TaskID:    "task-1",
			Prompt:    "prompt 1",
			Mode:      TaskModeAct,
			Cwd:       "/tmp",
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			Status:    "running",
		},
	}

	err = storageCtx.GlobalState.Set("taskHistory", testHistory)
	if err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Update task status
	err = UpdateTaskHistory(storageCtx, "task-1", map[string]interface{}{
		"status":       "completed",
		"messageCount": 10,
	})
	if err != nil {
		t.Fatalf("Failed to update history: %v", err)
	}

	// Verify update
	history, _ := GetTaskHistory(storageCtx, 0)
	if history[0].Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", history[0].Status)
	}
	if history[0].MessageCount != 10 {
		t.Errorf("Expected messageCount 10, got %d", history[0].MessageCount)
	}

	// Test update non-existent task
	err = UpdateTaskHistory(storageCtx, "non-existent", map[string]interface{}{
		"status": "completed",
	})
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestResumeOptions(t *testing.T) {
	opts := ResumeOptions{
		TaskID:  "task-123",
		Prompt:  "follow-up prompt",
		Images:  []string{"image1.png", "image2.png"},
		Verbose: true,
		Timeout: 5 * time.Minute,
	}

	if opts.TaskID != "task-123" {
		t.Errorf("Expected TaskID 'task-123', got %s", opts.TaskID)
	}
	if opts.Prompt != "follow-up prompt" {
		t.Errorf("Expected Prompt 'follow-up prompt', got %s", opts.Prompt)
	}
	if len(opts.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(opts.Images))
	}
	if !opts.Verbose {
		t.Error("Expected Verbose to be true")
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Expected 5 minute timeout, got %v", opts.Timeout)
	}
}

func TestResumeTask_MissingTaskID(t *testing.T) {
	err := ResumeTask("", ResumeOptions{})
	if err == nil {
		t.Error("Expected error for missing task ID, got nil")
	}
}

func TestResumeTask_MissingStorage(t *testing.T) {
	err := ResumeTask("task-123", ResumeOptions{})
	if err == nil {
		t.Error("Expected error for missing storage, got nil")
	}
}

func TestContinueTask_MissingStorage(t *testing.T) {
	err := ContinueTask(ResumeOptions{})
	if err == nil {
		t.Error("Expected error for missing storage, got nil")
	}
}

func TestContinueTask_MissingClient(t *testing.T) {
	// Create temporary storage
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	err = ContinueTask(ResumeOptions{Storage: storageCtx})
	if err == nil {
		t.Error("Expected error for missing client, got nil")
	}
}

// Helper to create test storage
func createTestStorage(t *testing.T) *storage.StorageContext {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	return storageCtx
}

func TestHistoryItem_JSON(t *testing.T) {
	item := HistoryItem{
		TaskID:       "test-123",
		Prompt:       "test prompt",
		Mode:         TaskModeAct,
		Cwd:          "/tmp/test",
		Model:        "claude-sonnet-4",
		CreatedAt:    1234567890,
		UpdatedAt:    1234567899,
		Status:       "running",
		MessageCount: 5,
	}

	// Marshal to JSON
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded HistoryItem
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.TaskID != item.TaskID {
		t.Errorf("TaskID mismatch: got %s, want %s", decoded.TaskID, item.TaskID)
	}
	if decoded.MessageCount != item.MessageCount {
		t.Errorf("MessageCount mismatch: got %d, want %d", decoded.MessageCount, item.MessageCount)
	}
}

func TestFindMostRecentTask_WithWorkspace(t *testing.T) {
	// Create temporary storage
	tempDir, err := os.MkdirTemp("", "cline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}

	// Create test directories
	dir1 := filepath.Join(tempDir, "workspace1")
	dir2 := filepath.Join(tempDir, "workspace2")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	now := time.Now().Unix()
	testHistory := []HistoryItem{
		{
			TaskID:    "task-dir1",
			Prompt:    "prompt in dir1",
			Mode:      TaskModeAct,
			Cwd:       dir1,
			CreatedAt: now - 100,
			UpdatedAt: now - 10,
			Status:    "completed",
		},
		{
			TaskID:    "task-dir2",
			Prompt:    "prompt in dir2",
			Mode:      TaskModePlan,
			Cwd:       dir2,
			CreatedAt: now - 50,
			UpdatedAt: now, // Most recent overall
			Status:    "running",
		},
	}

	err = storageCtx.GlobalState.Set("taskHistory", testHistory)
	if err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Find most recent task for dir1
	recent, err := FindMostRecentTask(storageCtx, dir1)
	if err != nil {
		t.Fatalf("Failed to find recent task: %v", err)
	}

	if recent.TaskID != "task-dir1" {
		t.Errorf("Expected 'task-dir1' for dir1 workspace, got %s", recent.TaskID)
	}

	// Find most recent task for dir2
	recent, err = FindMostRecentTask(storageCtx, dir2)
	if err != nil {
		t.Fatalf("Failed to find recent task: %v", err)
	}

	if recent.TaskID != "task-dir2" {
		t.Errorf("Expected 'task-dir2' for dir2 workspace, got %s", recent.TaskID)
	}
}