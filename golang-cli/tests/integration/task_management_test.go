// Package integration provides integration tests for the Go CLI.
// This file contains integration tests for task management functionality (Phase 2).
package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskCreationAndResumption tests creating and resuming tasks
func TestTaskCreationAndResumption(t *testing.T) {
	t.Run("creates new task and persists to history", func(t *testing.T) {
		// Create temporary storage
		tempDir := t.TempDir()
		storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer storageCtx.Close()

		// Create task
		taskID := "test-task-123"
		historyItem := task.HistoryItem{
			TaskID:    taskID,
			Prompt:    "Test task prompt",
			Mode:      "act",
			Status:    "initialized",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save to history
		err = task.SaveTaskHistory(storageCtx, historyItem)
		require.NoError(t, err)

		// Retrieve from history
		history, err := task.GetTaskHistory(storageCtx, 10)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, taskID, history[0].TaskID)
		assert.Equal(t, "Test task prompt", history[0].Prompt)
	})

	t.Run("finds most recent task", func(t *testing.T) {
		tempDir := t.TempDir()
		storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer storageCtx.Close()

		// Create multiple tasks
		for i := 0; i < 3; i++ {
			item := task.HistoryItem{
				TaskID:    fmt.Sprintf("task-%d", i),
				Prompt:    fmt.Sprintf("Prompt %d", i),
				Mode:      "act",
				Status:    "completed",
				CreatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
				UpdatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
			}
			err := task.SaveTaskHistory(storageCtx, item)
			require.NoError(t, err)
		}

		// Find most recent
		recent, err := task.FindMostRecentTask(storageCtx, "")
		require.NoError(t, err)
		assert.Equal(t, "task-0", recent.TaskID)
	})

	t.Run("finds task by ID", func(t *testing.T) {
		tempDir := t.TempDir()
		storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer storageCtx.Close()

		item := task.HistoryItem{
			TaskID:    "specific-task",
			Prompt:    "Specific prompt",
			Mode:      "plan",
			Status:    "running",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = task.SaveTaskHistory(storageCtx, item)
		require.NoError(t, err)

		found, err := task.FindTaskByID(storageCtx, "specific-task")
		require.NoError(t, err)
		assert.Equal(t, "Specific prompt", found.Prompt)
		assert.Equal(t, "plan", found.Mode)

		_, err = task.FindTaskByID(storageCtx, "non-existent")
		assert.Error(t, err)
	})
}

// TestTaskStateManagement tests task state persistence
func TestTaskStateManagement(t *testing.T) {
	t.Run("initializes and persists task state", func(t *testing.T) {
		tempDir := t.TempDir()
		sm, err := task.NewStateManager("state-test-task", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		// Initialize state
		err = sm.InitializeState("Test prompt", "act")
		require.NoError(t, err)

		// Verify state
		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, task.TaskStatusRunning, state.Status)
		assert.Equal(t, "Test prompt", state.CurrentPrompt)
		assert.Equal(t, "act", state.Mode)

		// Update checkpoint
		err = sm.UpdateCheckpoint("abc123")
		require.NoError(t, err)

		state, err = sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, "abc123", state.CheckpointHash)
	})

	t.Run("handles interruptions gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		sm, err := task.NewStateManager("interrupt-task", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		// Initialize and mark as running
		err = sm.InitializeState("Long running task", "act")
		require.NoError(t, err)

		err = sm.MarkRunning()
		require.NoError(t, err)
		assert.True(t, sm.IsRunning())

		// Simulate interruption
		err = sm.MarkInterrupted()
		require.NoError(t, err)
		assert.True(t, sm.IsInterrupted())
		assert.False(t, sm.IsRunning())

		// Verify state persists
		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, task.TaskStatusInterrupted, state.Status)
	})

	t.Run("state persists across reopen", func(t *testing.T) {
		tempDir := t.TempDir()
		taskID := "persist-task"

		// Create and update state
		{
			sm, err := task.NewStateManager(taskID, tempDir)
			require.NoError(t, err)

			err = sm.InitializeState("Persistent task", "plan")
			require.NoError(t, err)

			err = sm.UpdateStatus(task.TaskStatusCompleted)
			require.NoError(t, err)

			err = sm.UpdateCheckpoint("checkpoint-hash")
			require.NoError(t, err)

			err = sm.UpdateMetadata("model", "claude-4")
			require.NoError(t, err)

			err = sm.Close()
			require.NoError(t, err)
		}

		// Reopen and verify
		{
			sm, err := task.NewStateManager(taskID, tempDir)
			require.NoError(t, err)
			defer sm.Close()

			state, err := sm.GetState()
			require.NoError(t, err)

			assert.Equal(t, task.TaskStatusCompleted, state.Status)
			assert.Equal(t, "Persistent task", state.CurrentPrompt)
			assert.Equal(t, "plan", state.Mode)
			assert.Equal(t, "checkpoint-hash", state.CheckpointHash)
			assert.Equal(t, "claude-4", state.Metadata["model"])
		}
	})
}

// TestConversationManagement tests conversation management
func TestConversationManagement(t *testing.T) {
	t.Run("creates and manages conversation", func(t *testing.T) {
		tempDir := t.TempDir()
		cm, err := task.NewConversationManager("conv-task", tempDir)
		require.NoError(t, err)
		defer cm.Close()

		// Add messages
		err = cm.AddMessage(&task.ConversationMessage{
			Type:    "user",
			Content: "Hello",
		})
		require.NoError(t, err)

		err = cm.AddMessage(&task.ConversationMessage{
			Type:    "say",
			Content: "Hi there!",
		})
		require.NoError(t, err)

		// Verify messages
		assert.Equal(t, 2, cm.GetMessageCount())

		messages, err := cm.GetAllMessages()
		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "Hello", messages[0].Content)
		assert.Equal(t, "Hi there!", messages[1].Content)
	})

	t.Run("conversation persists across reopen", func(t *testing.T) {
		tempDir := t.TempDir()
		taskID := "persist-conv-task"

		// Create conversation
		{
			cm, err := task.NewConversationManager(taskID, tempDir)
			require.NoError(t, err)

			cm.AddMessage(&task.ConversationMessage{
				ID:      "msg-1",
				Type:    "user",
				Content: "Persistent message",
			})

			cm.AddMessage(&task.ConversationMessage{
				ID:        "msg-2",
				Type:      "say",
				Content:   "Persistent response",
				Reasoning: "Some reasoning",
			})

			err = cm.Close()
			require.NoError(t, err)
		}

		// Reopen and verify
		{
			cm, err := task.NewConversationManager(taskID, tempDir)
			require.NoError(t, err)
			defer cm.Close()

			assert.Equal(t, 2, cm.GetMessageCount())

			msg1, ok := cm.GetMessage("msg-1")
			require.True(t, ok)
			assert.Equal(t, "Persistent message", msg1.Content)

			msg2, ok := cm.GetMessage("msg-2")
			require.True(t, ok)
			assert.Equal(t, "Persistent response", msg2.Content)
			assert.Equal(t, "Some reasoning", msg2.Reasoning)
		}
	})

	t.Run("supports pagination", func(t *testing.T) {
		tempDir := t.TempDir()
		cm, err := task.NewConversationManager("page-task", tempDir)
		require.NoError(t, err)
		defer cm.Close()

		// Add 10 messages
		for i := 0; i < 10; i++ {
			cm.AddMessage(&task.ConversationMessage{
				Type:    "user",
				Content: fmt.Sprintf("Message %d", i),
			})
		}

		// Get first 5
		page1, err := cm.GetMessages(task.PaginationParams{
			Offset:    0,
			Limit:     5,
			Direction: "asc",
		})
		require.NoError(t, err)
		require.Len(t, page1, 5)
		assert.Equal(t, "Message 0", page1[0].Content)
		assert.Equal(t, "Message 4", page1[4].Content)

		// Get next 5
		page2, err := cm.GetMessages(task.PaginationParams{
			Offset:    5,
			Limit:     5,
			Direction: "asc",
		})
		require.NoError(t, err)
		require.Len(t, page2, 5)
		assert.Equal(t, "Message 5", page2[0].Content)
		assert.Equal(t, "Message 9", page2[4].Content)
	})
}

// TestContinueFlagFunctionality tests the --continue flag behavior
func TestContinueFlagFunctionality(t *testing.T) {
	t.Run("continues most recent task", func(t *testing.T) {
		tempDir := t.TempDir()
		storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer storageCtx.Close()

		// Create multiple tasks
		for i := 0; i < 3; i++ {
			item := task.HistoryItem{
				TaskID:    fmt.Sprintf("task-%d", i),
				Prompt:    fmt.Sprintf("Prompt %d", i),
				Mode:      "act",
				Status:    "completed",
				CreatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
				UpdatedAt: time.Now().Add(time.Duration(-i) * time.Hour),
			}
			err := task.SaveTaskHistory(storageCtx, item)
			require.NoError(t, err)
		}

		// Find most recent (should be task-0)
		history, err := task.GetTaskHistory(storageCtx, 1)
		require.NoError(t, err)
		require.Len(t, history, 1)
		assert.Equal(t, "task-0", history[0].TaskID)
	})

	t.Run("handles no previous tasks", func(t *testing.T) {
		tempDir := t.TempDir()
		storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer storageCtx.Close()

		history, err := task.GetTaskHistory(storageCtx, 1)
		require.NoError(t, err)
		assert.Len(t, history, 0)
	})
}

// TestImageAttachments tests image attachment functionality
func TestImageAttachments(t *testing.T) {
	t.Run("loads and validates image", func(t *testing.T) {
		// Create a test image file
		tempDir := t.TempDir()
		imagePath := filepath.Join(tempDir, "test.png")

		// Create minimal PNG file
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, // IHDR chunk length
			0x49, 0x48, 0x44, 0x52, // IHDR
			0x00, 0x00, 0x00, 0x01, // Width: 1
			0x00, 0x00, 0x00, 0x01, // Height: 1
			0x08, 0x02, 0x00, 0x00, 0x00, // Bit depth, color type, etc.
			0x90, 0x77, 0x53, 0xDE, // CRC
			0x00, 0x00, 0x00, 0x00, // IDAT chunk length
			0x49, 0x44, 0x41, 0x54, // IDAT
			0x08, 0x1D, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, // Some data
			0x00, 0x00, 0x00, 0x00, // IEND chunk length
			0x49, 0x45, 0x4E, 0x44, // IEND
			0xAE, 0x42, 0x60, 0x82, // IEND CRC
		}
		err := os.WriteFile(imagePath, pngData, 0644)
		require.NoError(t, err)

		// Test image loading via task initialization
		imageData, err := task.LoadAndValidateImage(imagePath)
		require.NoError(t, err)

		assert.Equal(t, imagePath, imageData.Path)
		assert.Equal(t, "image/png", imageData.MimeType)
		assert.NotEmpty(t, imageData.Content)
		assert.Greater(t, imageData.Size, int64(0))
	})

	t.Run("rejects invalid image formats", func(t *testing.T) {
		tempDir := t.TempDir()
		txtPath := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(txtPath, []byte("not an image"), 0644)
		require.NoError(t, err)

		_, err = task.LoadAndValidateImage(txtPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("rejects non-existent files", func(t *testing.T) {
		_, err := task.LoadAndValidateImage("/nonexistent/path/image.png")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not accessible")
	})
}

// TestTaskExecutionFlow tests the complete task execution flow
func TestTaskExecutionLifecycle(t *testing.T) {
	t.Run("task lifecycle with state tracking", func(t *testing.T) {
		tempDir := t.TempDir()
		taskID := "lifecycle-task"

		// Create state manager
		sm, err := task.NewStateManager(taskID, tempDir)
		require.NoError(t, err)

		// Create conversation manager
		cm, err := task.NewConversationManager(taskID, tempDir)
		require.NoError(t, err)

		// Initialize task
		err = sm.InitializeState("Test task", "act")
		require.NoError(t, err)

		// Add initial user message
		cm.AddMessage(&task.ConversationMessage{
			Type:    "user",
			Content: "Test task",
		})

		// Simulate task execution with messages
		cm.AddMessage(&task.ConversationMessage{
			Type:    "say",
			Content: "I'll help you with that",
		})

		// Update checkpoint
		sm.UpdateCheckpoint("checkpoint-1")

		// Add more messages
		cm.AddMessage(&task.ConversationMessage{
			Type:     "tool_use",
			ToolName: "read_file",
			Content:  "Reading file...",
		})

		cm.AddMessage(&task.ConversationMessage{
			Type:       "tool_result",
			ToolName:   "read_file",
			ToolResult: "File contents here",
		})

		// Complete task
		err = sm.MarkCompleted()
		require.NoError(t, err)

		// Close managers
		sm.Close()
		cm.Close()

		// Reopen and verify
		sm2, err := task.NewStateManager(taskID, tempDir)
		require.NoError(t, err)
		defer sm2.Close()

		cm2, err := task.NewConversationManager(taskID, tempDir)
		require.NoError(t, err)
		defer cm2.Close()

		// Verify state
		state, err := sm2.GetState()
		require.NoError(t, err)
		assert.Equal(t, task.TaskStatusCompleted, state.Status)
		assert.Equal(t, "checkpoint-1", state.CheckpointHash)

		// Verify conversation (4 messages: user, say, tool_use, tool_result)
		assert.Equal(t, 4, cm2.GetMessageCount())
	})
}

// TestConcurrentTaskOperations tests concurrent operations
func TestConcurrentTaskOperations(t *testing.T) {
	t.Run("concurrent message additions", func(t *testing.T) {
		tempDir := t.TempDir()
		cm, err := task.NewConversationManager("concurrent-task", tempDir)
		require.NoError(t, err)
		defer cm.Close()

		// Add messages concurrently
		done := make(chan bool, 50)
		for i := 0; i < 50; i++ {
			go func(i int) {
				defer func() { done <- true }()
				cm.AddMessage(&task.ConversationMessage{
					Type:    "user",
					Content: fmt.Sprintf("Message %d", i),
				})
			}(i)
		}

		for i := 0; i < 50; i++ {
			<-done
		}

		assert.Equal(t, 50, cm.GetMessageCount())
	})

	t.Run("concurrent state updates", func(t *testing.T) {
		tempDir := t.TempDir()
		sm, err := task.NewStateManager("concurrent-state-task", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		err = sm.InitializeState("Test", "act")
		require.NoError(t, err)

		// Update state concurrently
		done := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func(i int) {
				defer func() { done <- true }()
				if i%2 == 0 {
					sm.UpdateMetadata("even", i)
				} else {
					sm.UpdateMetadata("odd", i)
				}
			}(i)
		}

		for i := 0; i < 20; i++ {
			<-done
		}

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.NotNil(t, state.Metadata["even"])
		assert.NotNil(t, state.Metadata["odd"])
	})
}

// BenchmarkTaskOperations benchmarks task operations
func BenchmarkTaskOperations(b *testing.B) {
	b.Run("StateUpdate", func(b *testing.B) {
		tempDir := b.TempDir()
		sm, _ := task.NewStateManager("bench-state-task", tempDir)
		defer sm.Close()

		sm.InitializeState("Benchmark", "act")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sm.UpdateStatus(task.TaskStatusRunning)
		}
	})

	b.Run("MessageAdd", func(b *testing.B) {
		tempDir := b.TempDir()
		cm, _ := task.NewConversationManager("bench-conv-task", tempDir)
		defer cm.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cm.AddMessage(&task.ConversationMessage{
				Type:    "user",
				Content: "Benchmark message",
			})
		}
	})

	b.Run("HistorySave", func(b *testing.B) {
		tempDir := b.TempDir()
		storageCtx, _ := storage.NewStorageContext(tempDir, "bench-workspace")
		defer storageCtx.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			item := task.HistoryItem{
				TaskID:    fmt.Sprintf("task-%d", i),
				Prompt:    "Benchmark prompt",
				Mode:      "act",
				Status:    "completed",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			task.SaveTaskHistory(storageCtx, item)
		}
	})
}
