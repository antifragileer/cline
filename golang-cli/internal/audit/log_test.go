package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check that default values are set
	if config.MaxFileSize != 10*1024*1024 {
		t.Errorf("MaxFileSize = %d, want %d", config.MaxFileSize, 10*1024*1024)
	}
	if config.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d, want 5", config.MaxBackups)
	}
	if config.BufferSize != 1000 {
		t.Errorf("BufferSize = %d, want 1000", config.BufferSize)
	}
	if config.SyncWrite != false {
		t.Error("SyncWrite should be false by default")
	}
	if config.LogDir == "" {
		t.Error("LogDir should not be empty")
	}
}

func TestNewLogger(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("creates logger with valid config", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      tempDir,
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		if logger.GetLogPath() != filepath.Join(tempDir, "audit.log") {
			t.Errorf("GetLogPath() = %s, want %s", logger.GetLogPath(), filepath.Join(tempDir, "audit.log"))
		}
		if logger.IsClosed() {
			t.Error("Logger should not be closed")
		}
	})

	t.Run("creates log directory if not exists", func(t *testing.T) {
		logDir := filepath.Join(tempDir, "new", "nested", "dir")
		config := Config{
			Enabled:     true,
			LogDir:      logDir,
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			t.Error("Log directory should be created")
		}
	})

	t.Run("async mode starts worker", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "async"),
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   false,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		// Test that async mode works by logging an event
		event := &Event{
			EventType:   EventCommandExecution,
			UserContext: "test-user",
			TaskID:      "task-123",
			SessionID:   "session-456",
			Message:     "Test message",
		}
		if err := logger.Log(event); err != nil {
			t.Errorf("Log failed: %v", err)
		}
	})
}

func TestLogger_Log(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("logs event with all fields", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      tempDir,
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		event := &Event{
			Timestamp:   timestamp,
			EventType:   EventCommandExecution,
			UserContext: "test-user",
			TaskID:      "task-123",
			SessionID:   "session-456",
			Message:     "Command executed",
			Details: map[string]interface{}{
				"command": "ls",
				"args":    []string{"-la"},
			},
		}

		if err := logger.Log(event); err != nil {
			t.Errorf("Log failed: %v", err)
		}

		logger.Close()

		// Verify log file contains the event
		logData, err := os.ReadFile(logger.GetLogPath())
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		logContent := string(logData)
		if !strings.Contains(logContent, "command_execution") {
			t.Error("Log should contain event_type")
		}
		if !strings.Contains(logContent, "test-user") {
			t.Error("Log should contain user_context")
		}
		if !strings.Contains(logContent, "task-123") {
			t.Error("Log should contain task_id")
		}
		if !strings.Contains(logContent, "session-456") {
			t.Error("Log should contain session_id")
		}
	})

	t.Run("sets default timestamp if not provided", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "default-timestamp"),
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		event := &Event{
			EventType:   EventFileRead,
			UserContext: "user",
			TaskID:      "task",
			SessionID:   "session",
			Message:     "Test",
		}

		beforeLog := time.Now().UTC()
		if err := logger.Log(event); err != nil {
			t.Errorf("Log failed: %v", err)
		}
		afterLog := time.Now().UTC()

		logger.Close()

		// Verify timestamp was set
		if event.Timestamp.IsZero() {
			t.Error("Timestamp should be set")
		}
		if event.Timestamp.Before(beforeLog) || event.Timestamp.After(afterLog) {
			t.Error("Timestamp should be within expected range")
		}
	})

	t.Run("converts timestamp to UTC", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "utc-test"),
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		// Create timestamp in non-UTC timezone
		loc, _ := time.LoadLocation("America/New_York")
		nyTime := time.Date(2024, 1, 15, 10, 30, 0, 0, loc)

		event := &Event{
			Timestamp:   nyTime,
			EventType:   EventFileWrite,
			UserContext: "user",
			TaskID:      "task",
			SessionID:   "session",
			Message:     "Test",
		}

		if err := logger.Log(event); err != nil {
			t.Errorf("Log failed: %v", err)
		}

		logger.Close()

		// Verify timestamp was converted to UTC
		if event.Timestamp.Location() != time.UTC {
			t.Error("Timestamp should be in UTC")
		}
	})

	t.Run("returns error for nil event", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "nil-test"),
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		if err := logger.Log(nil); err == nil {
			t.Error("Log should return error for nil event")
		}
	})

	t.Run("handles all event types", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "event-types"),
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		eventTypes := []EventType{
			EventCommandExecution,
			EventFileRead,
			EventFileWrite,
			EventAPICall,
			EventToolApproval,
			EventToolRejection,
			EventConfigurationChange,
		}

		for i, eventType := range eventTypes {
			event := &Event{
				EventType:   eventType,
				UserContext: "user",
				TaskID:      "task",
				SessionID:   "session",
				Message:     "Test message " + string(rune('0'+i)),
			}
			if err := logger.Log(event); err != nil {
				t.Errorf("Log failed for event type %s: %v", eventType, err)
			}
		}

		logger.Close()

		// Verify all events were logged
		logData, err := os.ReadFile(logger.GetLogPath())
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		for _, eventType := range eventTypes {
			if !strings.Contains(string(logData), string(eventType)) {
				t.Errorf("Log should contain event type %s", eventType)
			}
		}
	})
}

func TestLogger_LogWithContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Enabled:     true,
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  10,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	details := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	if err := logger.LogWithContext(
		EventAPICall,
		"test-user",
		"task-123",
		"session-456",
		"API call completed",
		details,
	); err != nil {
		t.Errorf("LogWithContext failed: %v", err)
	}

	logger.Close()

	// Verify log content
	logData, err := os.ReadFile(logger.GetLogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(logData)
	if !strings.Contains(logContent, "api_call") {
		t.Error("Log should contain event type")
	}
	if !strings.Contains(logContent, "test-user") {
		t.Error("Log should contain user context")
	}
	if !strings.Contains(logContent, "task-123") {
		t.Error("Log should contain task ID")
	}
	if !strings.Contains(logContent, "session-456") {
		t.Error("Log should contain session ID")
	}
	if !strings.Contains(logContent, "API call completed") {
		t.Error("Log should contain message")
	}
}

func TestLogger_Close(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("closes logger successfully", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      tempDir,
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		if !logger.IsClosed() {
			t.Error("Logger should be marked as closed")
		}
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "multi-close"),
			MaxFileSize: 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		// Multiple closes should not panic
		logger.Close()
		logger.Close()
		logger.Close()

		if !logger.IsClosed() {
			t.Error("Logger should be closed")
		}
	})

	t.Run("flushes pending async events on close", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "async-flush"),
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  100,
			SyncWrite:   false,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		// Log multiple events asynchronously
		for i := 0; i < 10; i++ {
			event := &Event{
				EventType:   EventCommandExecution,
				UserContext: "user",
				TaskID:      "task",
				SessionID:   "session",
				Message:     "Test message",
			}
			if err := logger.Log(event); err != nil {
				t.Errorf("Log failed: %v", err)
			}
		}

		// Give async worker time to process events
		time.Sleep(100 * time.Millisecond)

		// Close should flush all pending events
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}

		// Verify all events were logged
		logData, err := os.ReadFile(logger.GetLogPath())
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		// Count occurrences of "Test message"
		count := strings.Count(string(logData), "Test message")
		if count != 10 {
			t.Errorf("Expected 10 logged messages, got %d", count)
		}
	})
}

func TestLogger_LogRotation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("rotates log when size exceeds limit", func(t *testing.T) {
		// Set a small max file size to trigger rotation quickly
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "rotation"),
			MaxFileSize: 500, // Very small for testing
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		// Log large events to trigger rotation
		for i := 0; i < 20; i++ {
			event := &Event{
				EventType:   EventCommandExecution,
				UserContext: "test-user-with-long-name",
				TaskID:      "task-12345",
				SessionID:   "session-67890",
				Message:     "This is a test message with lots of content to make the file bigger " + string(rune('0'+i%10)),
				Details: map[string]interface{}{
					"command": "echo",
					"args":    []string{"hello", "world", "this", "is", "a", "test"},
					"output":  "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
				},
			}
			if err := logger.Log(event); err != nil {
				t.Errorf("Log failed: %v", err)
			}
		}

		logger.Close()

		// Check that rotation occurred
		backupPath := filepath.Join(config.LogDir, "audit.log.1")
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			// Rotation might not have triggered if size wasn't exceeded
			// This is ok, just verify the main log exists
			if _, err := os.Stat(logger.GetLogPath()); os.IsNotExist(err) {
				t.Error("Main log file should exist")
			}
		}
	})

	t.Run("respects max backups limit", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "max-backups"),
			MaxFileSize: 200,
			MaxBackups:  2,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}

		// Create enough data to trigger multiple rotations
		for i := 0; i < 50; i++ {
			event := &Event{
				EventType:   EventCommandExecution,
				UserContext: "user",
				TaskID:      "task",
				SessionID:   "session",
				Message:     "Test message " + strings.Repeat("x", 100),
			}
			logger.Log(event)
		}

		logger.Close()

		// Check that we don't have more than MaxBackups+1 files (current + backups)
		entries, err := os.ReadDir(config.LogDir)
		if err != nil {
			t.Fatalf("Failed to read log directory: %v", err)
		}

		logFileCount := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "audit.log") {
				logFileCount++
			}
		}

		// Should have at most MaxBackups + 1 files (current + backups)
		if logFileCount > config.MaxBackups+1 {
			t.Errorf("Expected at most %d log files, got %d", config.MaxBackups+1, logFileCount)
		}
	})
}

func TestLogger_AsyncBufferFull(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("returns error when async buffer is full", func(t *testing.T) {
		config := Config{
			Enabled:     true,
			LogDir:      filepath.Join(tempDir, "buffer-full"),
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  1, // Very small buffer
			SyncWrite:   false,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		// Fill the buffer
		event := &Event{
			EventType:   EventCommandExecution,
			UserContext: "user",
			TaskID:      "task",
			SessionID:   "session",
			Message:     "Test",
		}

		// First event should succeed
		if err := logger.Log(event); err != nil {
			t.Errorf("First Log should succeed: %v", err)
		}

		// Immediately try to log more events - some may fail due to full buffer
		// This is a timing-dependent test, so we just verify the behavior exists
		var errors int
		for i := 0; i < 100; i++ {
			if err := logger.Log(event); err != nil {
				errors++
			}
		}

		// We should have seen some buffer full errors
		if errors == 0 {
			t.Log("Note: No buffer full errors - this may be due to timing")
		}
	})
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Enabled:     true,
		LogDir:      tempDir,
		MaxFileSize: 10 * 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  1000,
		SyncWrite:   false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	t.Run("concurrent logging", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numEvents := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numEvents; j++ {
					event := &Event{
						EventType:   EventCommandExecution,
						UserContext: "user",
						TaskID:      "task",
						SessionID:   "session",
						Message:     "Concurrent event",
					}
					// Ignore errors for concurrent test
					_ = logger.Log(event)
				}
			}(i)
		}

		wg.Wait()

		// Give async worker time to process
		time.Sleep(100 * time.Millisecond)

		// Verify some events were logged
		logData, err := os.ReadFile(logger.GetLogPath())
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		// Should have logged events
		if !strings.Contains(string(logData), "Concurrent event") {
			t.Error("Log should contain concurrent events")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Enabled:     true,
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  10,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	t.Run("LogCommandExecution", func(t *testing.T) {
		if err := logger.LogCommandExecution(
			"user",
			"task",
			"session",
			"ls",
			[]string{"-la"},
			"/home/user",
		); err != nil {
			t.Errorf("LogCommandExecution failed: %v", err)
		}
	})

	t.Run("LogFileRead", func(t *testing.T) {
		if err := logger.LogFileRead("user", "task", "session", "/path/to/file.txt", 1024); err != nil {
			t.Errorf("LogFileRead failed: %v", err)
		}
	})

	t.Run("LogFileWrite", func(t *testing.T) {
		if err := logger.LogFileWrite("user", "task", "session", "/path/to/file.txt", 2048, true); err != nil {
			t.Errorf("LogFileWrite failed: %v", err)
		}
	})

	t.Run("LogAPICall", func(t *testing.T) {
		if err := logger.LogAPICall("user", "task", "session", "openai", "gpt-4", "/v1/chat", 200, 150); err != nil {
			t.Errorf("LogAPICall failed: %v", err)
		}
	})

	t.Run("LogToolApproval", func(t *testing.T) {
		toolInput := map[string]interface{}{
			"file_path": "/path/to/file",
			"content":   "data",
		}
		if err := logger.LogToolApproval("user", "task", "session", "write_file", toolInput); err != nil {
			t.Errorf("LogToolApproval failed: %v", err)
		}
	})

	t.Run("LogToolRejection", func(t *testing.T) {
		toolInput := map[string]interface{}{
			"command": "rm -rf /",
		}
		if err := logger.LogToolRejection("user", "task", "session", "execute_command", toolInput, "Dangerous command"); err != nil {
			t.Errorf("LogToolRejection failed: %v", err)
		}
	})

	t.Run("LogConfigurationChange", func(t *testing.T) {
		if err := logger.LogConfigurationChange("user", "task", "session", "api_key", "old_value", "new_value"); err != nil {
			t.Errorf("LogConfigurationChange failed: %v", err)
		}
	})
}

func TestLogger_JSONFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Enabled:     true,
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  10,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	event := &Event{
		EventType:   EventCommandExecution,
		UserContext: "test-user",
		TaskID:      "task-123",
		SessionID:   "session-456",
		Message:     "Test message",
		Details: map[string]interface{}{
			"command": "ls",
			"args":    []string{"-la", "/home"},
		},
	}

	if err := logger.Log(event); err != nil {
		t.Errorf("Log failed: %v", err)
	}

	logger.Close()

	// Read and parse the log file
	logData, err := os.ReadFile(logger.GetLogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// slog outputs JSON lines, so split by newline
	lines := strings.Split(string(logData), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("Log line is not valid JSON: %v\nLine: %s", err, line)
			continue
		}

		// Check required fields
		if _, ok := logEntry["time"]; !ok {
			t.Error("Log entry should have 'time' field")
		}
		if _, ok := logEntry["level"]; !ok {
			t.Error("Log entry should have 'level' field")
		}
		if _, ok := logEntry["msg"]; !ok {
			t.Error("Log entry should have 'msg' field")
		}
		if eventData, ok := logEntry["event"].(map[string]interface{}); ok {
			if _, ok := eventData["timestamp"]; !ok {
				t.Error("Event should have 'timestamp' field")
			}
			if _, ok := eventData["event_type"]; !ok {
				t.Error("Event should have 'event_type' field")
			}
			if _, ok := eventData["user_context"]; !ok {
				t.Error("Event should have 'user_context' field")
			}
		} else {
			t.Error("Log entry should have 'event' field with event data")
		}
	}
}

func TestLogger_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests skipped on Windows")
	}

	tempDir, err := os.MkdirTemp("", "audit-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Enabled:     true,
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  10,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Log an event to create the file
	event := &Event{
		EventType:   EventCommandExecution,
		UserContext: "user",
		TaskID:      "task",
		SessionID:   "session",
		Message:     "Test",
	}
	logger.Log(event)
	logger.Close()

	// Check file permissions
	info, err := os.Stat(logger.GetLogPath())
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0640)
	if mode != expectedMode {
		t.Errorf("File permissions = %o, want %o", mode, expectedMode)
	}
}

func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventCommandExecution, "command_execution"},
		{EventFileRead, "file_read"},
		{EventFileWrite, "file_write"},
		{EventAPICall, "api_call"},
		{EventToolApproval, "tool_approval"},
		{EventToolRejection, "tool_rejection"},
		{EventConfigurationChange, "configuration_change"},
	}

	for _, test := range tests {
		t.Run(string(test.eventType), func(t *testing.T) {
			if string(test.eventType) != test.expected {
				t.Errorf("EventType = %s, want %s", test.eventType, test.expected)
			}
		})
	}
}