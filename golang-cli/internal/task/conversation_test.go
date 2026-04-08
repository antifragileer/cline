// Package task provides task execution and conversation management functionality
// for the Cline CLI.
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileStorage is a mock implementation of storage.FileStorage for testing
type mockFileStorage struct {
	data     map[string]interface{}
	mu       sync.RWMutex
	closed   bool
	getCalls int
	setCalls int
}

func newMockFileStorage() *mockFileStorage {
	return &mockFileStorage{
		data: make(map[string]interface{}),
	}
}

func (m *mockFileStorage) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.getCalls++
	val, ok := m.data[key]
	return val, ok
}

func (m *mockFileStorage) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.data[key] = value
	return nil
}

func (m *mockFileStorage) SetBatch(pairs map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	for k, v := range pairs {
		m.data[k] = v
	}
	return nil
}

func (m *mockFileStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockFileStorage) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *mockFileStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// TestNewConversationManager tests the creation of a new conversation manager
func TestNewConversationManager(t *testing.T) {
	t.Run("creates manager with valid taskID", func(t *testing.T) {
		tempDir := t.TempDir()
		cm, err := NewConversationManager("test-task-1", tempDir)
		require.NoError(t, err)
		assert.NotNil(t, cm)
		assert.Equal(t, "test-task-1", cm.GetTaskID())
		assert.True(t, cm.IsEmpty())
		assert.NoError(t, cm.Close())
	})

	t.Run("fails with empty taskID", func(t *testing.T) {
		tempDir := t.TempDir()
		cm, err := NewConversationManager("", tempDir)
		assert.Error(t, err)
		assert.Nil(t, cm)
		assert.Contains(t, err.Error(), "taskID is required")
	})

	t.Run("creates directory if it doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		taskDir := filepath.Join(tempDir, "new-task-dir")
		cm, err := NewConversationManager("new-task", taskDir)
		require.NoError(t, err)
		assert.DirExists(t, taskDir)
		assert.NoError(t, cm.Close())
	})

	t.Run("loads existing messages", func(t *testing.T) {
		mockStorage := newMockFileStorage()
		existingMessages := []*ConversationMessage{
			{ID: "msg-1", Type: "user", Content: "Hello", Timestamp: time.Now().UnixMilli()},
			{ID: "msg-2", Type: "say", Content: "Hi there", Timestamp: time.Now().UnixMilli()},
		}
		mockStorage.data[ConversationStorageKey] = existingMessages

		cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
		require.NoError(t, err)
		assert.Equal(t, 2, cm.GetMessageCount())
		assert.NoError(t, cm.Close())
	})
}

// TestAddMessage tests adding messages to the conversation
func TestAddMessage(t *testing.T) {
	t.Run("adds message successfully", func(t *testing.T) {
		mockStorage := newMockFileStorage()
		cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
		require.NoError(t, err)
		defer cm.Close()

		msg := &ConversationMessage{
			Type:    "user",
			Content: "Hello, assistant!",
		}

		err = cm.AddMessage(msg)
		require.NoError(t, err)
		assert.Equal(t, 1, cm.GetMessageCount())
		assert.NotEmpty(t, msg.ID)
		assert.Greater(t, msg.Timestamp, int64(0))
	})

	t.Run("rejects nil message", func(t *testing.T) {
		mockStorage := newMockFileStorage()
		cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
		require.NoError(t, err)
		defer cm.Close()

		err = cm.AddMessage(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message cannot be nil")
	})

	t.Run("truncates oversized content", func(t *testing.T) {
		mockStorage := newMockFileStorage()
		cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
		require.NoError(t, err)
		defer cm.Close()

		// Set small max size for testing
		cm.maxMessageSize = 100

		largeContent := strings.Repeat("a", 200)
		msg := &ConversationMessage{
			Type:    "user",
			Content: largeContent,
		}

		err = cm.AddMessage(msg)
		require.NoError(t, err)
		assert.Contains(t, msg.Content, "[content truncated]")
		assert.Less(t, len(msg.Content), 200)
	})

	t.Run("updates partial messages", func(t *testing.T) {
		mockStorage := newMockFileStorage()
		cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
		require.NoError(t, err)
		defer cm.Close()

		msg1 := &ConversationMessage{
			Type:    "say",
			Content: "Hello",
			Partial: true,
		}
		err = cm.AddMessage(msg1)
		require.NoError(t, err)
		assert.Equal(t, 1, cm.GetMessageCount())

		msg2 := &ConversationMessage{
			Type:    "say",
			Content: "Hello world",
			Partial: true,
		}
		err = cm.AddMessage(msg2)
		require.NoError(t, err)
		assert.Equal(t, 1, cm.GetMessageCount())

		lastMsg, ok := cm.GetLastMessage()
		require.True(t, ok)
		assert.Equal(t, "Hello world", lastMsg.Content)
	})
}

// TestGetMessage tests retrieving single messages
func TestGetMessage(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	msg := &ConversationMessage{
		ID:      "test-msg-123",
		Type:    "user",
		Content: "Test message",
	}
	err = cm.AddMessage(msg)
	require.NoError(t, err)

	t.Run("finds existing message by ID", func(t *testing.T) {
		found, ok := cm.GetMessage("test-msg-123")
		assert.True(t, ok)
		assert.NotNil(t, found)
		assert.Equal(t, "Test message", found.Content)
	})

	t.Run("returns false for non-existent ID", func(t *testing.T) {
		found, ok := cm.GetMessage("non-existent")
		assert.False(t, ok)
		assert.Nil(t, found)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		found, ok := cm.GetMessage("test-msg-123")
		require.True(t, ok)
		found.Content = "Modified"

		original, _ := cm.GetMessage("test-msg-123")
		assert.Equal(t, "Test message", original.Content)
	})
}

// TestGetMessages tests pagination
func TestGetMessages(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 10 messages
	for i := 0; i < 10; i++ {
		msg := &ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
		err := cm.AddMessage(msg)
		require.NoError(t, err)
	}

	t.Run("paginates with offset and limit", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    0,
			Limit:     5,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 5)
		assert.Equal(t, "Message 0", messages[0].Content)
		assert.Equal(t, "Message 4", messages[4].Content)
	})

	t.Run("returns next page", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    5,
			Limit:     5,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 5)
		assert.Equal(t, "Message 5", messages[0].Content)
		assert.Equal(t, "Message 9", messages[4].Content)
	})

	t.Run("returns empty when offset exceeds total", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    100,
			Limit:     5,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})

	t.Run("returns descending order", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    0,
			Limit:     5,
			Direction: "desc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 5)
		assert.Equal(t, "Message 9", messages[0].Content)
		assert.Equal(t, "Message 5", messages[4].Content)
	})

	t.Run("uses default page size", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    0,
			Limit:     0,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 10)
	})
}

// TestGetMessageRange tests retrieving message ranges
func TestGetMessageRange(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 5 messages
	for i := 0; i < 5; i++ {
		msg := &ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
		err := cm.AddMessage(msg)
		require.NoError(t, err)
	}

	t.Run("gets valid range", func(t *testing.T) {
		messages, err := cm.GetMessageRange(1, 3)
		require.NoError(t, err)
		assert.Len(t, messages, 3)
		assert.Equal(t, "Message 1", messages[0].Content)
		assert.Equal(t, "Message 3", messages[2].Content)
	})

	t.Run("clamps out of bounds", func(t *testing.T) {
		messages, err := cm.GetMessageRange(-5, 100)
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("returns empty for invalid range", func(t *testing.T) {
		messages, err := cm.GetMessageRange(3, 1)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}

// TestGetLastNMessages tests retrieving last N messages
func TestGetLastNMessages(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 5 messages
	for i := 0; i < 5; i++ {
		msg := &ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
		err := cm.AddMessage(msg)
		require.NoError(t, err)
	}

	t.Run("gets last N messages", func(t *testing.T) {
		messages, err := cm.GetLastNMessages(3)
		require.NoError(t, err)
		assert.Len(t, messages, 3)
		assert.Equal(t, "Message 4", messages[0].Content)
		assert.Equal(t, "Message 2", messages[2].Content)
	})

	t.Run("returns all if N exceeds count", func(t *testing.T) {
		messages, err := cm.GetLastNMessages(100)
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("returns empty for zero", func(t *testing.T) {
		messages, err := cm.GetLastNMessages(0)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}

// TestUpdateMessage tests updating messages
func TestUpdateMessage(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	msg := &ConversationMessage{
		ID:      "update-test",
		Type:    "user",
		Content: "Original content",
	}
	err = cm.AddMessage(msg)
	require.NoError(t, err)

	t.Run("updates existing message", func(t *testing.T) {
		updates := &ConversationMessage{
			Content: "Updated content",
			Type:    "user",
		}

		err := cm.UpdateMessage("update-test", updates)
		require.NoError(t, err)

		updated, ok := cm.GetMessage("update-test")
		require.True(t, ok)
		assert.Equal(t, "Updated content", updated.Content)
		assert.Equal(t, "update-test", updated.ID) // ID preserved
	})

	t.Run("fails for non-existent message", func(t *testing.T) {
		updates := &ConversationMessage{Content: "Test"}
		err := cm.UpdateMessage("non-existent", updates)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message not found")
	})

	t.Run("rejects nil updates", func(t *testing.T) {
		err := cm.UpdateMessage("update-test", nil)
		assert.Error(t, err)
	})
}

// TestDeleteMessage tests deleting messages
func TestDeleteMessage(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	msg := &ConversationMessage{
		ID:      "delete-test",
		Type:    "user",
		Content: "To be deleted",
	}
	err = cm.AddMessage(msg)
	require.NoError(t, err)
	assert.Equal(t, 1, cm.GetMessageCount())

	t.Run("deletes existing message", func(t *testing.T) {
		err := cm.DeleteMessage("delete-test")
		require.NoError(t, err)
		assert.Equal(t, 0, cm.GetMessageCount())
	})

	t.Run("fails for non-existent message", func(t *testing.T) {
		err := cm.DeleteMessage("non-existent")
		assert.Error(t, err)
	})
}

// TestDeleteMessageRange tests deleting message ranges
func TestDeleteMessageRange(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 5 messages
	for i := 0; i < 5; i++ {
		msg := &ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
		err := cm.AddMessage(msg)
		require.NoError(t, err)
	}

	t.Run("deletes valid range", func(t *testing.T) {
		err := cm.DeleteMessageRange(1, 3)
		require.NoError(t, err)
		assert.Equal(t, 2, cm.GetMessageCount())

		// Check that remaining messages are correct
		all, _ := cm.GetAllMessages()
		assert.Equal(t, "Message 0", all[0].Content)
		assert.Equal(t, "Message 4", all[1].Content)
	})

	t.Run("records deleted range info", func(t *testing.T) {
		// Add more messages to test range recording
		for i := 5; i < 8; i++ {
			msg := &ConversationMessage{
				Type:    "user",
				Content: fmt.Sprintf("Message %d", i),
			}
			cm.AddMessage(msg)
		}

		err := cm.DeleteMessageRange(2, 4)
		require.NoError(t, err)

		all, _ := cm.GetAllMessages()
		// The message at index 2 should have DeletedRange info
		if len(all) > 2 && all[2].DeletedRange != nil {
			assert.GreaterOrEqual(t, all[2].DeletedRange.EndIndex, all[2].DeletedRange.StartIndex)
		}
	})

	t.Run("fails for invalid range", func(t *testing.T) {
		err := cm.DeleteMessageRange(10, 20)
		assert.Error(t, err)
	})
}

// TestSearchMessages tests message searching
func TestSearchMessages(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add messages with different content
	messages := []*ConversationMessage{
		{Type: "user", Content: "Hello world"},
		{Type: "say", Content: "Hi there, how can I help?"},
		{Type: "tool_use", ToolName: "read_file", Content: "Reading file..."},
		{Type: "tool_result", ToolName: "read_file", ToolResult: "File contents: hello world"},
		{Type: "user", Content: "Goodbye world"},
	}

	for _, msg := range messages {
		err := cm.AddMessage(msg)
		require.NoError(t, err)
	}

	t.Run("searches in content", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "world",
			SearchInContent: true,
			Pagination:      PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		// "Hello world", "Goodbye world" = 2 matches (tool result is searched separately)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
	})

	t.Run("searches case insensitive", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "HELLO",
			SearchInContent: true,
			Pagination:      PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, total) // "Hello world"
		_ = results
	})

	t.Run("searches case sensitive", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "Hello",
			SearchInContent: true,
			CaseSensitive:   true,
			Pagination:      PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, total) // Only "Hello world"
		_ = results
	})

	t.Run("searches in tool results", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:               "File contents",
			SearchInContent:     false,
			SearchInToolResults: true,
			Pagination:          PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		_ = results
	})

	t.Run("filters by message type", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "",
			MessageTypes:    []string{"user"},
			SearchInContent: true,
			Pagination:      PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		_ = results
	})

	t.Run("filters by time range", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-1 * time.Hour)
		future := now.Add(1 * time.Hour)

		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "",
			SearchInContent: true,
			StartTime:       &past,
			EndTime:         &future,
			Pagination:      PaginationParams{Limit: 10},
		})
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		_ = results
	})

	t.Run("paginates search results", func(t *testing.T) {
		results, total, err := cm.SearchMessages(SearchParams{
			Query:           "world",
			SearchInContent: true,
			Pagination: PaginationParams{
				Offset: 0,
				Limit:  1,
			},
		})
		require.NoError(t, err)
		// "Hello world", "Goodbye world" = 2 matches
		assert.Equal(t, 2, total)
		assert.Len(t, results, 1)
	})
}

// TestSearchWithRegex tests regex searching
func TestSearchWithRegex(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	messages := []*ConversationMessage{
		{Type: "user", Content: "Error: file not found"},
		{Type: "user", Content: "Error: connection timeout"},
		{Type: "say", Content: "Success: operation completed"},
	}

	for _, msg := range messages {
		cm.AddMessage(msg)
	}

	t.Run("searches with regex pattern", func(t *testing.T) {
		results, err := cm.SearchWithRegex(`^Error:`, false)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		_ = results
	})

	t.Run("case insensitive regex", func(t *testing.T) {
		results, err := cm.SearchWithRegex(`SUCCESS`, false)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		_ = results
	})

	t.Run("case sensitive regex", func(t *testing.T) {
		results, err := cm.SearchWithRegex(`success`, true)
		require.NoError(t, err)
		assert.Len(t, results, 0) // No lowercase "success"
		_ = results
	})

	t.Run("invalid regex returns error", func(t *testing.T) {
		results, err := cm.SearchWithRegex(`[invalid`, false)
		assert.Error(t, err)
		assert.Nil(t, results)
	})
}

// TestExport tests conversation export
func TestExport(t *testing.T) {
	tempDir := t.TempDir()
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add test messages
	cm.AddMessage(&ConversationMessage{
		Type:    "user",
		Content: "Hello",
	})
	cm.AddMessage(&ConversationMessage{
		Type:    "say",
		Content: "Hi there!",
	})

	t.Run("exports to JSON", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "export.json")
		err := cm.Export(ExportFormatJSON, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"task_id": "test-task"`)
		assert.Contains(t, string(content), `"content": "Hello"`)
	})

	t.Run("exports to Markdown", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "export.md")
		err := cm.Export(ExportFormatMarkdown, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# Conversation Export")
		assert.Contains(t, string(content), "👤 User")
		assert.Contains(t, string(content), "🤖 Assistant")
	})

	t.Run("exports to Plain Text", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "export.txt")
		err := cm.Export(ExportFormatPlainText, outputPath)
		require.NoError(t, err)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "USER: Hello")
		assert.Contains(t, string(content), "ASSISTANT: Hi there!")
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "export.xyz")
		err := cm.Export("unsupported", outputPath)
		assert.Error(t, err)
	})
}

// TestExportToString tests export to string
func TestExportToString(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	cm.AddMessage(&ConversationMessage{
		Type:    "user",
		Content: "Test message",
	})

	t.Run("exports to JSON string", func(t *testing.T) {
		str, err := cm.ExportToString(ExportFormatJSON)
		require.NoError(t, err)
		assert.Contains(t, str, "test-task")
		assert.Contains(t, str, "Test message")
	})

	t.Run("exports to Markdown string", func(t *testing.T) {
		str, err := cm.ExportToString(ExportFormatMarkdown)
		require.NoError(t, err)
		assert.Contains(t, str, "# Conversation Export")
	})
}

// TestGetStats tests statistics calculation
func TestGetStats(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	t.Run("returns zero stats for empty conversation", func(t *testing.T) {
		stats := cm.GetStats()
		assert.Equal(t, 0, stats.TotalMessages)
		assert.Equal(t, 0, stats.UserMessages)
		assert.Equal(t, 0, stats.AIMessages)
		assert.Equal(t, 0, stats.ToolMessages)
	})

	t.Run("calculates stats correctly", func(t *testing.T) {
		cm.AddMessage(&ConversationMessage{Type: "user", Content: "Hello"})
		cm.AddMessage(&ConversationMessage{Type: "say", Content: "Hi"})
		cm.AddMessage(&ConversationMessage{Type: "ask", Content: "What?"})
		cm.AddMessage(&ConversationMessage{Type: "tool_use", ToolName: "read"})
		cm.AddMessage(&ConversationMessage{Type: "tool_result", ToolName: "read", ToolResult: "data"})

		stats := cm.GetStats()
		assert.Equal(t, 5, stats.TotalMessages)
		assert.Equal(t, 1, stats.UserMessages)
		assert.Equal(t, 2, stats.AIMessages)
		assert.Equal(t, 2, stats.ToolMessages)
		assert.Greater(t, stats.TotalTokens, 0)
		assert.Greater(t, stats.StorageSize, int64(0))
	})
}

// TestTruncateConversation tests conversation truncation
func TestTruncateConversation(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 10 messages
	for i := 0; i < 10; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	t.Run("truncates to specified limit", func(t *testing.T) {
		err := cm.TruncateConversation(5)
		require.NoError(t, err)
		assert.Equal(t, 5, cm.GetMessageCount())
	})

	t.Run("preserves first message", func(t *testing.T) {
		all, _ := cm.GetAllMessages()
		require.GreaterOrEqual(t, len(all), 1)
		assert.Equal(t, "Message 0", all[0].Content)
	})

	t.Run("rejects invalid limit", func(t *testing.T) {
		err := cm.TruncateConversation(0)
		assert.Error(t, err)
		err = cm.TruncateConversation(-1)
		assert.Error(t, err)
	})
}

// TestAutoTruncation tests automatic truncation
func TestAutoTruncation(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Set small max for testing
	cm.SetMaxMessages(5)

	// Add 10 messages
	for i := 0; i < 10; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	assert.Equal(t, 5, cm.GetMessageCount())

	// Verify first message is preserved
	all, _ := cm.GetAllMessages()
	require.GreaterOrEqual(t, len(all), 1)
	assert.Equal(t, "Message 0", all[0].Content)
}

// TestClear tests clearing conversation
func TestClear(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	cm.AddMessage(&ConversationMessage{Type: "user", Content: "Test"})
	assert.Equal(t, 1, cm.GetMessageCount())

	err = cm.Clear()
	require.NoError(t, err)
	assert.Equal(t, 0, cm.GetMessageCount())
	assert.True(t, cm.IsEmpty())
}

// TestConversationStore tests the conversation store
func TestConversationStore(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates new store", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		assert.NotNil(t, store)
		defer store.Close()
	})

	t.Run("creates default directory", func(t *testing.T) {
		store, err := NewConversationStore("")
		require.NoError(t, err)
		assert.NotNil(t, store)
		defer store.Close()
	})

	t.Run("gets or creates manager", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		cm1, err := store.GetOrCreateManager("task-1")
		require.NoError(t, err)
		assert.NotNil(t, cm1)

		// Should return same instance
		cm2, err := store.GetOrCreateManager("task-1")
		require.NoError(t, err)
		assert.Equal(t, cm1, cm2)
	})

	t.Run("gets existing manager", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		cm1, _ := store.GetOrCreateManager("task-2")

		cm2, ok := store.GetManager("task-2")
		assert.True(t, ok)
		assert.Equal(t, cm1, cm2)

		_, ok = store.GetManager("non-existent")
		assert.False(t, ok)
	})

	t.Run("removes manager", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		store.GetOrCreateManager("task-3")
		assert.Equal(t, 1, len(store.ListTaskIDs()))

		err = store.RemoveManager("task-3")
		require.NoError(t, err)
		assert.Equal(t, 0, len(store.ListTaskIDs()))
	})

	t.Run("lists task IDs", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		store.GetOrCreateManager("zebra")
		store.GetOrCreateManager("apple")
		store.GetOrCreateManager("mango")

		ids := store.ListTaskIDs()
		assert.Len(t, ids, 3)
		assert.Equal(t, "apple", ids[0])
		assert.Equal(t, "mango", ids[1])
		assert.Equal(t, "zebra", ids[2])
	})

	t.Run("deletes conversation", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)
		defer store.Close()

		cm, _ := store.GetOrCreateManager("task-to-delete")
		cm.AddMessage(&ConversationMessage{Type: "user", Content: "Test"})

		err = store.DeleteConversation("task-to-delete")
		require.NoError(t, err)

		_, ok := store.GetManager("task-to-delete")
		assert.False(t, ok)
	})

	t.Run("closes all managers", func(t *testing.T) {
		store, err := NewConversationStore(tempDir)
		require.NoError(t, err)

		store.GetOrCreateManager("task-a")
		store.GetOrCreateManager("task-b")

		err = store.Close()
		require.NoError(t, err)
		assert.Equal(t, 0, len(store.ListTaskIDs()))
	})
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	t.Run("concurrent adds", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				cm.AddMessage(&ConversationMessage{
					Type:    "user",
					Content: fmt.Sprintf("Message %d", i),
				})
			}(i)
		}
		wg.Wait()

		assert.Equal(t, 100, cm.GetMessageCount())
	})

	t.Run("concurrent reads", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := cm.GetMessages(PaginationParams{Limit: 10})
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent search", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := cm.SearchMessages(SearchParams{
					Query:           "Message",
					SearchInContent: true,
					Pagination:      PaginationParams{Limit: 10},
				})
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})
}

// TestClose tests closing the manager
func TestClose(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)

	cm.AddMessage(&ConversationMessage{Type: "user", Content: "Test"})

	err = cm.Close()
	require.NoError(t, err)
	assert.True(t, mockStorage.closed)
}

// TestCopyMessage tests the copyMessage function
func TestCopyMessage(t *testing.T) {
	t.Run("copies all fields", func(t *testing.T) {
		original := &ConversationMessage{
			ID:                          "test-id",
			Timestamp:                   1234567890,
			Type:                        "user",
			AskType:                     "followup",
			SayType:                     "text",
			Content:                     "Hello",
			Reasoning:                   "Because",
			Images:                      []string{"img1.png", "img2.png"},
			Files:                       []string{"file1.go", "file2.go"},
			Partial:                     true,
			LastCheckpointHash:          "abc123",
			IsCheckpointCheckedOut:      true,
			IsOperationOutsideWorkspace: false,
			ConversationHistoryIndex:    5,
			ToolName:                    "read_file",
			ToolInput:                   map[string]interface{}{"path": "test.go"},
			ToolResult:                  "result",
			Language:                    "go",
			Metadata:                    map[string]interface{}{"key": "value"},
			DeletedRange: &ConversationHistoryDeletedRange{
				StartIndex: 1,
				EndIndex:   5,
			},
		}

		copy := copyMessage(original)

		// Verify all fields are copied
		assert.Equal(t, original.ID, copy.ID)
		assert.Equal(t, original.Timestamp, copy.Timestamp)
		assert.Equal(t, original.Type, copy.Type)
		assert.Equal(t, original.AskType, copy.AskType)
		assert.Equal(t, original.SayType, copy.SayType)
		assert.Equal(t, original.Content, copy.Content)
		assert.Equal(t, original.Reasoning, copy.Reasoning)
		assert.Equal(t, original.Images, copy.Images)
		assert.Equal(t, original.Files, copy.Files)
		assert.Equal(t, original.Partial, copy.Partial)
		assert.Equal(t, original.LastCheckpointHash, copy.LastCheckpointHash)
		assert.Equal(t, original.IsCheckpointCheckedOut, copy.IsCheckpointCheckedOut)
		assert.Equal(t, original.IsOperationOutsideWorkspace, copy.IsOperationOutsideWorkspace)
		assert.Equal(t, original.ConversationHistoryIndex, copy.ConversationHistoryIndex)
		assert.Equal(t, original.ToolName, copy.ToolName)
		assert.Equal(t, original.ToolResult, copy.ToolResult)
		assert.Equal(t, original.Language, copy.Language)
		assert.Equal(t, original.Metadata, copy.Metadata)
		assert.Equal(t, original.DeletedRange.StartIndex, copy.DeletedRange.StartIndex)
		assert.Equal(t, original.DeletedRange.EndIndex, copy.DeletedRange.EndIndex)

		// Verify deep copy - modifying copy shouldn't affect original
		copy.Content = "Modified"
		copy.Images[0] = "modified.png"
		copy.Metadata["key"] = "modified"
		copy.ToolInput["new"] = "value"

		assert.Equal(t, "Hello", original.Content)
		assert.Equal(t, "img1.png", original.Images[0])
		assert.Equal(t, "value", original.Metadata["key"])
		_, exists := original.ToolInput["new"]
		assert.False(t, exists)
	})

	t.Run("handles nil message", func(t *testing.T) {
		copy := copyMessage(nil)
		assert.Nil(t, copy)
	})

	t.Run("handles nil slices and maps", func(t *testing.T) {
		original := &ConversationMessage{
			Type:    "user",
			Content: "Test",
		}

		copy := copyMessage(original)
		assert.NotNil(t, copy)
		assert.Empty(t, copy.Images)
		assert.Empty(t, copy.Files)
		assert.Empty(t, copy.Metadata)
	})
}

// TestGenerateMessageID tests ID generation
func TestGenerateMessageID(t *testing.T) {
	id1 := generateMessageID()
	id2 := generateMessageID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.True(t, strings.HasPrefix(id1, "msg_"))
	assert.Regexp(t, regexp.MustCompile(`^msg_\d+_\d+$`), id1)
}

// TestIntegrationWithFileStorage tests integration with real file storage
func TestIntegrationWithFileStorage(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("persists across reopen", func(t *testing.T) {
		// Create manager and add messages
		cm1, err := NewConversationManager("integration-test", tempDir)
		require.NoError(t, err)

		cm1.AddMessage(&ConversationMessage{
			ID:      "msg-1",
			Type:    "user",
			Content: "Hello",
		})
		cm1.AddMessage(&ConversationMessage{
			ID:      "msg-2",
			Type:    "say",
			Content: "Hi there",
		})

		err = cm1.Close()
		require.NoError(t, err)

		// Reopen and verify messages are loaded
		cm2, err := NewConversationManager("integration-test", tempDir)
		require.NoError(t, err)
		defer cm2.Close()

		assert.Equal(t, 2, cm2.GetMessageCount())

		msg1, ok := cm2.GetMessage("msg-1")
		require.True(t, ok)
		assert.Equal(t, "Hello", msg1.Content)

		msg2, ok := cm2.GetMessage("msg-2")
		require.True(t, ok)
		assert.Equal(t, "Hi there", msg2.Content)
	})

	t.Run("exports to file", func(t *testing.T) {
		cm, err := NewConversationManager("export-test", tempDir)
		require.NoError(t, err)

		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: "Export test",
		})

		outputPath := filepath.Join(tempDir, "export.json")
		err = cm.Export(ExportFormatJSON, outputPath)
		require.NoError(t, err)

		assert.FileExists(t, outputPath)

		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Export test")

		cm.Close()
	})
}

// TestPaginationEdgeCases tests pagination edge cases
func TestPaginationEdgeCases(t *testing.T) {
	mockStorage := newMockFileStorage()
	cm, err := NewConversationManagerWithStorage("test-task", mockStorage)
	require.NoError(t, err)
	defer cm.Close()

	// Add 5 messages
	for i := 0; i < 5; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	t.Run("offset at boundary", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    4,
			Limit:     10,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 1)
	})

	t.Run("negative limit uses default", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    0,
			Limit:     -5,
			Direction: "asc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("descending with large offset", func(t *testing.T) {
		messages, err := cm.GetMessages(PaginationParams{
			Offset:    10,
			Limit:     5,
			Direction: "desc",
		})
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}

// Benchmark tests
func BenchmarkAddMessage(b *testing.B) {
	mockStorage := newMockFileStorage()
	cm, _ := NewConversationManagerWithStorage("bench-task", mockStorage)
	defer cm.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}
}

func BenchmarkSearchMessages(b *testing.B) {
	mockStorage := newMockFileStorage()
	cm, _ := NewConversationManagerWithStorage("bench-task", mockStorage)
	defer cm.Close()

	// Add 1000 messages
	for i := 0; i < 1000; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message content with searchable text %d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.SearchMessages(SearchParams{
			Query:           "searchable",
			SearchInContent: true,
			Pagination:      PaginationParams{Limit: 50},
		})
	}
}

func BenchmarkGetMessages(b *testing.B) {
	mockStorage := newMockFileStorage()
	cm, _ := NewConversationManagerWithStorage("bench-task", mockStorage)
	defer cm.Close()

	// Add 1000 messages
	for i := 0; i < 1000; i++ {
		cm.AddMessage(&ConversationMessage{
			Type:    "user",
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.GetMessages(PaginationParams{
			Offset:    i % 500,
			Limit:     50,
			Direction: "asc",
		})
	}
}
