// Package task provides task state management functionality for the Cline CLI.
// This file contains tests for the task state management functionality.
package task

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStateManager tests the creation of a new state manager
func TestNewStateManager(t *testing.T) {
	t.Run("creates manager with valid taskID", func(t *testing.T) {
		tempDir := t.TempDir()
		sm, err := NewStateManager("test-task-1", tempDir)
		require.NoError(t, err)
		assert.NotNil(t, sm)
		assert.Equal(t, "test-task-1", sm.GetTaskID())
		assert.NoError(t, sm.Close())
	})

	t.Run("fails with empty taskID", func(t *testing.T) {
		tempDir := t.TempDir()
		sm, err := NewStateManager("", tempDir)
		assert.Error(t, err)
		assert.Nil(t, sm)
		assert.Contains(t, err.Error(), "taskID is required")
	})

	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		taskDir := filepath.Join(tempDir, "new-task-dir")
		sm, err := NewStateManager("new-task", taskDir)
		require.NoError(t, err)
		assert.DirExists(t, taskDir)
		assert.NoError(t, sm.Close())
	})

	t.Run("uses default directory when dataDir is empty", func(t *testing.T) {
		sm, err := NewStateManager("default-dir-task", "")
		require.NoError(t, err)
		assert.NotNil(t, sm)
		assert.NotEmpty(t, sm.GetDataDir())
		assert.NoError(t, sm.Close())
	})
}

// TestStateManagerLifecycle tests the full lifecycle of state management
func TestStateManagerLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager("lifecycle-task", tempDir)
	require.NoError(t, err)
	defer sm.Close()

	t.Run("initializes state", func(t *testing.T) {
		err := sm.InitializeState("Test prompt", "act")
		require.NoError(t, err)

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, "lifecycle-task", state.TaskID)
		assert.Equal(t, TaskStatusRunning, state.Status)
		assert.Equal(t, "Test prompt", state.CurrentPrompt)
		assert.Equal(t, "act", state.Mode)
		assert.Greater(t, state.StartedAt, int64(0))
		assert.Greater(t, state.LastActivity, int64(0))
	})

	t.Run("updates status", func(t *testing.T) {
		err := sm.UpdateStatus(TaskStatusCompleted)
		require.NoError(t, err)

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, TaskStatusCompleted, state.Status)
	})

	t.Run("updates checkpoint", func(t *testing.T) {
		err := sm.UpdateCheckpoint("abc123def456")
		require.NoError(t, err)

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, "abc123def456", state.CheckpointHash)
	})

	t.Run("updates prompt", func(t *testing.T) {
		err := sm.UpdatePrompt("Updated prompt")
		require.NoError(t, err)

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, "Updated prompt", state.CurrentPrompt)
	})

	t.Run("updates metadata", func(t *testing.T) {
		err := sm.UpdateMetadata("key1", "value1")
		require.NoError(t, err)

		err = sm.UpdateMetadata("key2", 42)
		require.NoError(t, err)

		state, err := sm.GetState()
		require.NoError(t, err)
		assert.Equal(t, "value1", state.Metadata["key1"])
		assert.Equal(t, 42, state.Metadata["key2"])
	})
}

// TestStatusTransitions tests status transition methods
func TestStatusTransitions(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager("status-task", tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Initialize state
	err = sm.InitializeState("Test", "act")
	require.NoError(t, err)

	t.Run("MarkRunning", func(t *testing.T) {
		err := sm.MarkRunning()
		require.NoError(t, err)
		assert.True(t, sm.IsRunning())
		assert.False(t, sm.IsCompleted())
		assert.False(t, sm.IsInterrupted())
	})

	t.Run("MarkCompleted", func(t *testing.T) {
		err := sm.MarkCompleted()
		require.NoError(t, err)
		assert.False(t, sm.IsRunning())
		assert.True(t, sm.IsCompleted())
		assert.False(t, sm.IsInterrupted())
	})

	t.Run("MarkFailed", func(t *testing.T) {
		err := sm.MarkFailed()
		require.NoError(t, err)
		assert.False(t, sm.IsRunning())
		assert.False(t, sm.IsCompleted())
	})

	t.Run("MarkInterrupted", func(t *testing.T) {
		err := sm.MarkInterrupted()
		require.NoError(t, err)
		assert.False(t, sm.IsRunning())
		assert.False(t, sm.IsCompleted())
		assert.True(t, sm.IsInterrupted())
	})
}

// TestStatePersistence tests state persistence across reopen
func TestStatePersistence(t *testing.T) {
	tempDir := t.TempDir()
	taskID := "persistence-task"

	// Create state and save it
	{
		sm, err := NewStateManager(taskID, tempDir)
		require.NoError(t, err)

		err = sm.InitializeState("Persistent prompt", "plan")
		require.NoError(t, err)

		err = sm.UpdateStatus(TaskStatusRunning)
		require.NoError(t, err)

		err = sm.UpdateCheckpoint("checkpoint-123")
		require.NoError(t, err)

		err = sm.UpdateMetadata("model", "claude-sonnet-4-6")
		require.NoError(t, err)

		err = sm.Close()
		require.NoError(t, err)
	}

	// Reopen and verify state is loaded
	{
		sm, err := NewStateManager(taskID, tempDir)
		require.NoError(t, err)
		defer sm.Close()

		state, err := sm.GetState()
		require.NoError(t, err)

		assert.Equal(t, taskID, state.TaskID)
		assert.Equal(t, TaskStatusRunning, state.Status)
		assert.Equal(t, "Persistent prompt", state.CurrentPrompt)
		assert.Equal(t, "plan", state.Mode)
		assert.Equal(t, "checkpoint-123", state.CheckpointHash)
		assert.Equal(t, "claude-sonnet-4-6", state.Metadata["model"])
	}
}

// TestAutoSave tests automatic saving functionality
func TestAutoSave(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager("autosave-task", tempDir)
	require.NoError(t, err)
	defer sm.Close()

	t.Run("auto save is enabled by default", func(t *testing.T) {
		assert.True(t, sm.autoSave)
	})

	t.Run("can disable auto save", func(t *testing.T) {
		sm.SetAutoSave(false)
		assert.False(t, sm.autoSave)
	})

	t.Run("can re-enable auto save", func(t *testing.T) {
		sm.SetAutoSave(true)
		assert.True(t, sm.autoSave)
	})

	t.Run("manual save works", func(t *testing.T) {
		err := sm.InitializeState("Test", "act")
		require.NoError(t, err)

		err = sm.Save()
		require.NoError(t, err)
	})
}

// TestConcurrentAccess tests thread safety
func TestStateConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager("concurrent-task", tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Initialize state
	err = sm.InitializeState("Concurrent test", "act")
	require.NoError(t, err)

	t.Run("concurrent status updates", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer func() { done <- true }()
				if i%2 == 0 {
					sm.MarkRunning()
				} else {
					sm.MarkCompleted()
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		// State should be consistent
		state, err := sm.GetState()
		require.NoError(t, err)
		assert.NotNil(t, state)
	})

	t.Run("concurrent metadata updates", func(t *testing.T) {
		done := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func(i int) {
				defer func() { done <- true }()
				sm.UpdateMetadata("counter", i)
			}(i)
		}

		for i := 0; i < 20; i++ {
			<-done
		}

		state, err := sm.GetState()
		require.NoError(t, err)
		// Last update should be reflected
		assert.NotNil(t, state.Metadata["counter"])
	})
}

// TestStateStore tests the state store functionality
func TestStateStore(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates new store", func(t *testing.T) {
		store, err := NewStateStore(tempDir)
		require.NoError(t, err)
		assert.NotNil(t, store)
		defer store.Close()
	})

	t.Run("gets or creates manager", func(t *testing.T) {
		store, err := NewStateStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		sm1, err := store.GetOrCreateManager("task-1")
		require.NoError(t, err)
		assert.NotNil(t, sm1)

		// Should return same instance
		sm2, err := store.GetOrCreateManager("task-1")
		require.NoError(t, err)
		assert.Equal(t, sm1, sm2)
	})

	t.Run("gets existing manager", func(t *testing.T) {
		store, err := NewStateStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		sm1, _ := store.GetOrCreateManager("task-2")

		sm2, ok := store.GetManager("task-2")
		assert.True(t, ok)
		assert.Equal(t, sm1, sm2)

		_, ok = store.GetManager("non-existent")
		assert.False(t, ok)
	})

	t.Run("removes manager", func(t *testing.T) {
		store, err := NewStateStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		_, _ = store.GetOrCreateManager("task-3")

		err = store.RemoveManager("task-3")
		require.NoError(t, err)

		_, ok := store.GetManager("task-3")
		assert.False(t, ok)
	})

	t.Run("closes all managers", func(t *testing.T) {
		store, err := NewStateStore(tempDir)
		require.NoError(t, err)

		_, _ = store.GetOrCreateManager("task-a")
		_, _ = store.GetOrCreateManager("task-b")

		err = store.Close()
		require.NoError(t, err)
	})
}

// TestErrorCases tests error handling
func TestStateErrorCases(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("GetState without initialization", func(t *testing.T) {
		sm, err := NewStateManager("no-init-task", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		_, err = sm.GetState()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no state initialized")
	})

	t.Run("UpdateStatus without initialization", func(t *testing.T) {
		sm, err := NewStateManager("no-init-task-2", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		err = sm.UpdateStatus(TaskStatusRunning)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no state initialized")
	})

	t.Run("UpdateCheckpoint without initialization", func(t *testing.T) {
		sm, err := NewStateManager("no-init-task-3", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		err = sm.UpdateCheckpoint("hash")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no state initialized")
	})

	t.Run("UpdatePrompt without initialization", func(t *testing.T) {
		sm, err := NewStateManager("no-init-task-4", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		err = sm.UpdatePrompt("prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no state initialized")
	})

	t.Run("UpdateMetadata without initialization", func(t *testing.T) {
		sm, err := NewStateManager("no-init-task-5", tempDir)
		require.NoError(t, err)
		defer sm.Close()

		err = sm.UpdateMetadata("key", "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no state initialized")
	})
}

// TestCopyState tests the copyState function
func TestCopyState(t *testing.T) {
	t.Run("copies all fields", func(t *testing.T) {
		original := &TaskState{
			TaskID:         "test-id",
			Status:         TaskStatusRunning,
			StartedAt:      1234567890,
			LastActivity:   1234567899,
			CurrentPrompt:  "Test prompt",
			Mode:           "act",
			CheckpointHash: "abc123",
			Metadata:       map[string]interface{}{"key": "value"},
		}

		copy := copyState(original)

		// Verify all fields are copied
		assert.Equal(t, original.TaskID, copy.TaskID)
		assert.Equal(t, original.Status, copy.Status)
		assert.Equal(t, original.StartedAt, copy.StartedAt)
		assert.Equal(t, original.LastActivity, copy.LastActivity)
		assert.Equal(t, original.CurrentPrompt, copy.CurrentPrompt)
		assert.Equal(t, original.Mode, copy.Mode)
		assert.Equal(t, original.CheckpointHash, copy.CheckpointHash)
		assert.Equal(t, original.Metadata["key"], copy.Metadata["key"])

		// Verify deep copy - modifying copy shouldn't affect original
		copy.CurrentPrompt = "Modified"
		copy.Metadata["key"] = "modified"

		assert.Equal(t, "Test prompt", original.CurrentPrompt)
		assert.Equal(t, "value", original.Metadata["key"])
	})

	t.Run("handles nil state", func(t *testing.T) {
		copy := copyState(nil)
		assert.Nil(t, copy)
	})

	t.Run("handles nil metadata", func(t *testing.T) {
		original := &TaskState{
			TaskID:   "test",
			Metadata: nil,
		}

		copy := copyState(original)
		assert.NotNil(t, copy)
		assert.Empty(t, copy.Metadata)
	})
}

// TestLastActivityUpdates tests that LastActivity is updated on modifications
func TestLastActivityUpdates(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager("activity-task", tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Initialize
	err = sm.InitializeState("Test", "act")
	require.NoError(t, err)

	state1, err := sm.GetState()
	require.NoError(t, err)
	initialActivity := state1.LastActivity

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Update status
	err = sm.UpdateStatus(TaskStatusCompleted)
	require.NoError(t, err)

	state2, err := sm.GetState()
	require.NoError(t, err)
	assert.Greater(t, state2.LastActivity, initialActivity)

	// Wait again
	time.Sleep(10 * time.Millisecond)

	// Update checkpoint
	err = sm.UpdateCheckpoint("new-hash")
	require.NoError(t, err)

	state3, err := sm.GetState()
	require.NoError(t, err)
	assert.Greater(t, state3.LastActivity, state2.LastActivity)
}

// Benchmark tests
func BenchmarkStateManagerOperations(b *testing.B) {
	tempDir := b.TempDir()
	sm, _ := NewStateManager("bench-task", tempDir)
	defer sm.Close()

	sm.InitializeState("Benchmark test", "act")

	b.Run("UpdateStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sm.UpdateStatus(TaskStatusRunning)
		}
	})

	b.Run("UpdateMetadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sm.UpdateMetadata("key", i)
		}
	})

	b.Run("GetState", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sm.GetState()
		}
	})
}
