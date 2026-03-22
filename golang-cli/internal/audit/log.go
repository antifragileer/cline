package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// EventCommandExecution logs command execution events
	EventCommandExecution EventType = "command_execution"
	// EventFileRead logs file read events
	EventFileRead EventType = "file_read"
	// EventFileWrite logs file write events
	EventFileWrite EventType = "file_write"
	// EventAPICall logs API call events
	EventAPICall EventType = "api_call"
	// EventToolApproval logs tool approval events
	EventToolApproval EventType = "tool_approval"
	// EventToolRejection logs tool rejection events
	EventToolRejection EventType = "tool_rejection"
	// EventConfigurationChange logs configuration change events
	EventConfigurationChange EventType = "configuration_change"
)

// Event represents a single audit event
type Event struct {
	// Timestamp is the UTC timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// EventType is the type of audit event
	EventType EventType `json:"event_type"`
	// UserContext identifies the user (e.g., username, user ID)
	UserContext string `json:"user_context"`
	// TaskID is the unique identifier for the task
	TaskID string `json:"task_id"`
	// SessionID is the unique identifier for the session
	SessionID string `json:"session_id"`
	// Details contains event-specific details
	Details map[string]interface{} `json:"details,omitempty"`
	// Message is a human-readable description of the event
	Message string `json:"message"`
}

// Logger handles audit logging with support for rotation and async writes
type Logger struct {
	// config holds the logger configuration
	config Config
	// writer is the current log writer
	writer io.WriteCloser
	// slogLogger is the structured logger
	slogLogger *slog.Logger
	// eventChan is the channel for async event processing
	eventChan chan *Event
	// doneChan is used to signal shutdown completion
	doneChan chan struct{}
	// wg waits for the async worker to finish
	wg sync.WaitGroup
	// mu protects writer and currentSize
	mu sync.RWMutex
	// currentSize tracks the current log file size
	currentSize int64
	// closed indicates if the logger has been closed
	closed bool
	// closeOnce ensures Close is only called once
	closeOnce sync.Once
}

// Config holds configuration for the audit logger
type Config struct {
	// LogDir is the directory where log files are stored
	LogDir string
	// MaxFileSize is the maximum size of a log file before rotation (in bytes)
	MaxFileSize int64
	// MaxBackups is the maximum number of backup files to keep
	MaxBackups int
	// BufferSize is the size of the async event channel
	BufferSize int
	// SyncWrite determines if writes should be synchronous (blocking)
	SyncWrite bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		LogDir:      filepath.Join(homeDir, ".cline", "logs", "audit"),
		MaxFileSize: 10 * 1024 * 1024, // 10 MB
		MaxBackups:  5,
		BufferSize:  1000,
		SyncWrite:   false,
	}
}

// NewLogger creates a new audit logger with the given configuration
func NewLogger(config Config) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger := &Logger{
		config:    config,
		eventChan: make(chan *Event, config.BufferSize),
		doneChan:  make(chan struct{}),
	}

	// Open initial log file
	if err := logger.openLogFile(); err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Start async worker if not in sync mode
	if !config.SyncWrite {
		logger.wg.Add(1)
		go logger.asyncWorker()
	}

	return logger, nil
}

// openLogFile opens or creates the current log file
func (l *Logger) openLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	logPath := filepath.Join(l.config.LogDir, "audit.log")

	// Check if file exists and get its size
	if info, err := os.Stat(logPath); err == nil {
		l.currentSize = info.Size()
	} else {
		l.currentSize = 0
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.writer = file
	l.slogLogger = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return nil
}

// rotate performs log rotation when the current file exceeds MaxFileSize
func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation is needed
	if l.currentSize < l.config.MaxFileSize {
		return nil
	}

	// Close current writer
	if l.writer != nil {
		l.writer.Close()
	}

	logPath := filepath.Join(l.config.LogDir, "audit.log")

	// Rotate existing backups
	for i := l.config.MaxBackups - 1; i >= 0; i-- {
		var srcPath, dstPath string

		if i == 0 {
			srcPath = logPath
		} else {
			srcPath = filepath.Join(l.config.LogDir, fmt.Sprintf("audit.log.%d", i))
		}

		dstPath = filepath.Join(l.config.LogDir, fmt.Sprintf("audit.log.%d", i+1))

		// Remove oldest backup if it exists
		if i == l.config.MaxBackups-1 {
			os.Remove(dstPath)
		}

		// Rename source to destination
		if _, err := os.Stat(srcPath); err == nil {
			os.Rename(srcPath, dstPath)
		}
	}

	// Open new log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	l.writer = file
	l.slogLogger = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	l.currentSize = 0

	return nil
}

// asyncWorker processes events from the channel asynchronously
func (l *Logger) asyncWorker() {
	defer l.wg.Done()

	for event := range l.eventChan {
		if event != nil {
			l.writeEvent(event)
		}
	}
}

// writeEvent writes a single event to the log
func (l *Logger) writeEvent(event *Event) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed || l.writer == nil {
		return
	}

	// Marshal event to JSON for structured logging
	eventData := map[string]interface{}{
		"timestamp":    event.Timestamp.UTC().Format(time.RFC3339Nano),
		"event_type":   string(event.EventType),
		"user_context": event.UserContext,
		"task_id":      event.TaskID,
		"session_id":   event.SessionID,
		"message":      event.Message,
	}

	if len(event.Details) > 0 {
		eventData["details"] = event.Details
	}

	// Write using slog for structured JSON output
	l.slogLogger.Info("audit_event", slog.Any("event", eventData))

	// Update current size estimate (approximate)
	if jsonData, err := json.Marshal(eventData); err == nil {
		l.currentSize += int64(len(jsonData) + 1) // +1 for newline
	}

	// Check if rotation is needed
	if l.currentSize >= l.config.MaxFileSize {
		// We can't call rotate directly here because we hold the lock
		// Instead, we'll check after releasing the lock in Log()
	}
}

// Log records an audit event
func (l *Logger) Log(event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Set default timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		// Ensure UTC
		event.Timestamp = event.Timestamp.UTC()
	}

	// Check if rotation is needed before logging
	l.mu.RLock()
	needsRotation := l.currentSize >= l.config.MaxFileSize
	l.mu.RUnlock()

	if needsRotation {
		if err := l.rotate(); err != nil {
			return fmt.Errorf("failed to rotate log: %w", err)
		}
	}

	if l.config.SyncWrite {
		l.writeEvent(event)
	} else {
		select {
		case l.eventChan <- event:
			// Event queued successfully
		default:
			// Channel is full, drop the event but return an error
			return fmt.Errorf("audit log buffer full, event dropped")
		}
	}

	return nil
}

// LogWithContext is a convenience method to log an event with common fields
func (l *Logger) LogWithContext(
	eventType EventType,
	userContext,
	taskID,
	sessionID,
	message string,
	details map[string]interface{},
) error {
	event := &Event{
		EventType:   eventType,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     message,
		Details:     details,
		Timestamp:   time.Now().UTC(),
	}
	return l.Log(event)
}

// Close shuts down the logger and flushes pending events
func (l *Logger) Close() error {
	var closeErr error

	l.closeOnce.Do(func() {
		l.mu.Lock()
		l.closed = true
		l.mu.Unlock()

		// Close the event channel to prevent new events
		close(l.eventChan)

		// Wait for async worker to finish processing
		l.wg.Wait()

		// Close the writer
		l.mu.Lock()
		if l.writer != nil {
			closeErr = l.writer.Close()
			l.writer = nil
		}
		l.mu.Unlock()
	})

	return closeErr
}

// GetLogPath returns the path to the current log file
func (l *Logger) GetLogPath() string {
	return filepath.Join(l.config.LogDir, "audit.log")
}

// IsClosed returns true if the logger has been closed
func (l *Logger) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

// Helper functions for common event types

// LogCommandExecution logs a command execution event
func (l *Logger) LogCommandExecution(userContext, taskID, sessionID, command string, args []string, workingDir string) error {
	details := map[string]interface{}{
		"command":     command,
		"args":        args,
		"working_dir": workingDir,
	}
	return l.LogWithContext(EventCommandExecution, userContext, taskID, sessionID, "Command executed", details)
}

// LogFileRead logs a file read event
func (l *Logger) LogFileRead(userContext, taskID, sessionID, filePath string, bytesRead int64) error {
	details := map[string]interface{}{
		"file_path":  filePath,
		"bytes_read": bytesRead,
	}
	return l.LogWithContext(EventFileRead, userContext, taskID, sessionID, "File read", details)
}

// LogFileWrite logs a file write event
func (l *Logger) LogFileWrite(userContext, taskID, sessionID, filePath string, bytesWritten int64, isAppend bool) error {
	details := map[string]interface{}{
		"file_path":     filePath,
		"bytes_written": bytesWritten,
		"is_append":     isAppend,
	}
	return l.LogWithContext(EventFileWrite, userContext, taskID, sessionID, "File written", details)
}

// LogAPICall logs an API call event
func (l *Logger) LogAPICall(userContext, taskID, sessionID, provider, model, endpoint string, statusCode int, latencyMs int64) error {
	details := map[string]interface{}{
		"provider":     provider,
		"model":        model,
		"endpoint":     endpoint,
		"status_code":  statusCode,
		"latency_ms":   latencyMs,
	}
	return l.LogWithContext(EventAPICall, userContext, taskID, sessionID, "API call completed", details)
}

// LogToolApproval logs a tool approval event
func (l *Logger) LogToolApproval(userContext, taskID, sessionID, toolName string, toolInput map[string]interface{}) error {
	details := map[string]interface{}{
		"tool_name":  toolName,
		"tool_input": toolInput,
	}
	return l.LogWithContext(EventToolApproval, userContext, taskID, sessionID, "Tool approved", details)
}

// LogToolRejection logs a tool rejection event
func (l *Logger) LogToolRejection(userContext, taskID, sessionID, toolName string, toolInput map[string]interface{}, reason string) error {
	details := map[string]interface{}{
		"tool_name":  toolName,
		"tool_input": toolInput,
		"reason":     reason,
	}
	return l.LogWithContext(EventToolRejection, userContext, taskID, sessionID, "Tool rejected", details)
}

// LogConfigurationChange logs a configuration change event
func (l *Logger) LogConfigurationChange(userContext, taskID, sessionID, configKey string, oldValue, newValue interface{}) error {
	details := map[string]interface{}{
		"config_key": configKey,
		"old_value":  oldValue,
		"new_value":  newValue,
	}
	return l.LogWithContext(EventConfigurationChange, userContext, taskID, sessionID, "Configuration changed", details)
}