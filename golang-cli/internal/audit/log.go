// Package audit provides enterprise audit logging for the Cline CLI.
package audit

import (
	"context"
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
	// EventCommandExecution represents command execution
	EventCommandExecution EventType = "command_execution"
	// EventFileRead represents file read operations
	EventFileRead EventType = "file_read"
	// EventFileWrite represents file write operations
	EventFileWrite EventType = "file_write"
	// EventAPICall represents API calls
	EventAPICall EventType = "api_call"
	// EventToolApproval represents tool approval
	EventToolApproval EventType = "tool_approval"
	// EventToolRejection represents tool rejection
	EventToolRejection EventType = "tool_rejection"
	// EventConfigurationChange represents configuration change
	EventConfigurationChange EventType = "configuration_change"
)

// Config provides audit configuration for test compatibility
type Config = LoggerConfig

// DefaultConfig provides default configuration for test compatibility
func DefaultConfig() LoggerConfig {
	homeDir, _ := os.UserHomeDir()
	return LoggerConfig{
		Enabled:     true,
		LogDir:      filepath.Join(homeDir, ".cline", "logs"),
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxBackups:  5,
		BufferSize:  1000,
		SyncWrite:   false,
	}
}

// ensureDefaults sets default values for unset fields
func ensureDefaults(config *LoggerConfig) {
	if !config.Enabled && config.LogDir == "" {
		// If both are defaults, assume this is a fresh config and enable it
		config.Enabled = true
	}
	if config.LogDir == "" {
		homeDir, _ := os.UserHomeDir()
		config.LogDir = filepath.Join(homeDir, ".cline", "logs")
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 5
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
}

// Event represents an audit log event
type Event struct {
	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// EventType is the type of event
	EventType EventType `json:"event_type"`
	// UserContext identifies the user
	UserContext string `json:"user_context,omitempty"`
	// TaskID identifies the task
	TaskID string `json:"task_id,omitempty"`
	// SessionID identifies the session
	SessionID string `json:"session_id,omitempty"`
	// Message is the event message
	Message string `json:"message,omitempty"`
	// Details contains additional event data
	Details map[string]interface{} `json:"details,omitempty"`
}

// LoggerConfig configures the audit logger
type LoggerConfig struct {
	// Enabled enables or disables logging
	Enabled bool
	// LogDir is the directory for log files
	LogDir string
	// MaxFileSize is the maximum log file size before rotation
	MaxFileSize int64
	// MaxBackups is the number of backup files to keep
	MaxBackups int
	// BufferSize is the size of the write buffer
	BufferSize int
	// SyncWrite enables synchronous writing (for testing)
	SyncWrite bool
}

// DefaultLoggerConfig returns default configuration
func DefaultLoggerConfig() LoggerConfig {
	return DefaultConfig()
}

// Logger handles audit logging
type Logger struct {
	config    LoggerConfig
	logger    *slog.Logger
	file      *os.File
	logPath   string
	mu        sync.RWMutex
	closed    bool
	eventChan chan *Event
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewLogger creates a new audit logger
func NewLogger(config LoggerConfig) (*Logger, error) {
	if err := os.MkdirAll(config.LogDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(config.LogDir, "audit.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create slog handler
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	l := &Logger{
		config:    config,
		logger:    slog.New(handler),
		file:      file,
		logPath:   logPath,
		eventChan: make(chan *Event, config.BufferSize),
		done:      make(chan struct{}),
	}

	// Only start async worker if not in sync mode
	if config.Enabled && !config.SyncWrite {
		l.wg.Add(1)
		go l.writeLoop()
	}

	return l, nil
}

// writeLoop processes events in the background for async mode
func (l *Logger) writeLoop() {
	defer l.wg.Done()

	for {
		select {
		case event := <-l.eventChan:
			if event != nil {
				l.writeEventToLogger(event)
			}
		case <-l.done:
			// Drain remaining events
			for {
				select {
				case event := <-l.eventChan:
					if event != nil {
						l.writeEventToLogger(event)
					}
				default:
					return
				}
			}
		}
	}
}

// writeEventToLogger writes event to slog logger (async mode)
func (l *Logger) writeEventToLogger(event *Event) {
	l.logger.Info("audit event",
		"event", map[string]interface{}{
			"timestamp":    event.Timestamp.Format(time.RFC3339Nano),
			"event_type":   string(event.EventType),
			"user_context": event.UserContext,
			"task_id":      event.TaskID,
			"session_id":   event.SessionID,
			"message":      event.Message,
			"details":      event.Details,
		},
	)
}

// logEvent logs a single event synchronously with immediate sync
func (l *Logger) logEvent(event *Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else if event.Timestamp.Location() != time.UTC {
		event.Timestamp = event.Timestamp.UTC()
	}

	l.mu.RLock()
	file := l.file
	l.mu.RUnlock()

	l.logger.Info("audit event",
		"event", map[string]interface{}{
			"timestamp":    event.Timestamp.Format(time.RFC3339Nano),
			"event_type":   string(event.EventType),
			"user_context": event.UserContext,
			"task_id":      event.TaskID,
			"session_id":   event.SessionID,
			"message":      event.Message,
			"details":      event.Details,
		},
	)

	// Always sync after writing - critical for tests and data integrity
	if file != nil {
		file.Sync()
	}
}

// Log logs an audit event
func (l *Logger) Log(event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return fmt.Errorf("logger is closed")
	}
	l.mu.RUnlock()

	if !l.config.Enabled {
		return nil
	}

	if l.config.SyncWrite {
		// In sync mode, write directly and flush immediately
		l.logEvent(event)
		return nil
	}

	// In async mode, send to channel
	select {
	case l.eventChan <- event:
		return nil
	default:
		// Channel full, log synchronously as fallback
		l.logEvent(event)
		return nil
	}
}

// LogWithContext logs an event with all context fields
func (l *Logger) LogWithContext(eventType EventType, userContext, taskID, sessionID, message string, details map[string]interface{}) error {
	return l.Log(&Event{
		EventType:   eventType,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     message,
		Details:     details,
	})
}

// LogCommandExecution logs a command execution event
func (l *Logger) LogCommandExecution(userContext, taskID, sessionID, command string, args []string, cwd string) error {
	return l.Log(&Event{
		EventType:   EventCommandExecution,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Executed: %s", command),
		Details: map[string]interface{}{
			"command": command,
			"args":    args,
			"cwd":     cwd,
		},
	})
}

// LogFileRead logs a file read event
func (l *Logger) LogFileRead(userContext, taskID, sessionID, filePath string, size int64) error {
	return l.Log(&Event{
		EventType:   EventFileRead,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Read file: %s", filePath),
		Details: map[string]interface{}{
			"file_path": filePath,
			"size":      size,
		},
	})
}

// LogFileWrite logs a file write event
func (l *Logger) LogFileWrite(userContext, taskID, sessionID, filePath string, size int64, created bool) error {
	action := "Modified"
	if created {
		action = "Created"
	}
	return l.Log(&Event{
		EventType:   EventFileWrite,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("%s file: %s", action, filePath),
		Details: map[string]interface{}{
			"file_path": filePath,
			"size":      size,
			"created":   created,
		},
	})
}

// LogAPICall logs an API call event
func (l *Logger) LogAPICall(userContext, taskID, sessionID, provider, model, endpoint string, statusCode, tokensUsed int) error {
	return l.Log(&Event{
		EventType:   EventAPICall,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("API call to %s/%s", provider, endpoint),
		Details: map[string]interface{}{
			"provider":    provider,
			"model":       model,
			"endpoint":    endpoint,
			"status_code": statusCode,
			"tokens_used": tokensUsed,
		},
	})
}

// LogToolApproval logs a tool approval event
func (l *Logger) LogToolApproval(userContext, taskID, sessionID, toolName string, toolInput map[string]interface{}) error {
	return l.Log(&Event{
		EventType:   EventToolApproval,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Approved tool: %s", toolName),
		Details: map[string]interface{}{
			"tool_name":  toolName,
			"tool_input": toolInput,
		},
	})
}

// LogToolRejection logs a tool rejection event
func (l *Logger) LogToolRejection(userContext, taskID, sessionID, toolName string, toolInput map[string]interface{}, reason string) error {
	return l.Log(&Event{
		EventType:   EventToolRejection,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Rejected tool: %s - %s", toolName, reason),
		Details: map[string]interface{}{
			"tool_name":  toolName,
			"tool_input": toolInput,
			"reason":     reason,
		},
	})
}

// LogConfigurationChange logs a configuration change event
func (l *Logger) LogConfigurationChange(userContext, taskID, sessionID, settingName, oldValue, newValue string) error {
	return l.Log(&Event{
		EventType:   EventConfigurationChange,
		UserContext: userContext,
		TaskID:      taskID,
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Changed setting: %s", settingName),
		Details: map[string]interface{}{
			"setting_name": settingName,
			"old_value":    oldValue,
			"new_value":    newValue,
		},
	})
}

// GetLogPath returns the path to the log file
func (l *Logger) GetLogPath() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.logPath
}

// IsClosed returns whether the logger is closed
func (l *Logger) IsClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}

// Close closes the logger and flushes remaining events
func (l *Logger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	// Only close done channel if we started the async worker
	if !l.config.SyncWrite {
		close(l.done)
		l.wg.Wait()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		// Sync file to ensure all data is written
		l.file.Sync()
		return l.file.Close()
	}
	return nil
}

// SetEnabled enables or disables logging
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Enabled = enabled
}

// IsEnabled returns whether logging is enabled
func (l *Logger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Enabled
}

// Rotate rotates the log file
func (l *Logger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}

	// Close current file
	if err := l.file.Close(); err != nil {
		return err
	}

	// Rotate backups
	logPath := filepath.Join(l.config.LogDir, "audit.log")
	for i := l.config.MaxBackups - 1; i > 0; i-- {
		oldPath := filepath.Join(l.config.LogDir, fmt.Sprintf("audit.log.%d", i))
		newPath := filepath.Join(l.config.LogDir, fmt.Sprintf("audit.log.%d", i+1))
		os.Rename(oldPath, newPath)
	}

	// Move current to .1
	os.Rename(logPath, filepath.Join(l.config.LogDir, "audit.log.1"))

	// Open new file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return err
	}

	l.file = file
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	l.logger = slog.New(handler)

	return nil
}

// CheckSize checks if the log file needs rotation
func (l *Logger) CheckSize() error {
	l.mu.RLock()
	file := l.file
	maxSize := l.config.MaxFileSize
	l.mu.RUnlock()

	if file == nil || maxSize <= 0 {
		return nil
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if info.Size() >= maxSize {
		return l.Rotate()
	}

	return nil
}

// ContextualLogger returns a logger with context
func (l *Logger) ContextualLogger(ctx context.Context) *ContextualLogger {
	return &ContextualLogger{
		logger: l,
		ctx:    ctx,
	}
}

// ContextualLogger provides logging with context
type ContextualLogger struct {
	logger *Logger
	ctx    context.Context
}

// Log logs an event with context
func (cl *ContextualLogger) Log(event *Event) error {
	// Extract context values if available
	if cl.ctx != nil && event != nil {
		if taskID, ok := cl.ctx.Value("task_id").(string); ok && event.TaskID == "" {
			event.TaskID = taskID
		}
		if sessionID, ok := cl.ctx.Value("session_id").(string); ok && event.SessionID == "" {
			event.SessionID = sessionID
		}
		if userID, ok := cl.ctx.Value("user_id").(string); ok && event.UserContext == "" {
			event.UserContext = userID
		}
	}

	return cl.logger.Log(event)
}

// WithOutput creates a logger that writes to a specific writer
func WithOutput(w io.Writer, config LoggerConfig) *Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return &Logger{
		config:    config,
		logger:    slog.New(handler),
		eventChan: make(chan *Event, config.BufferSize),
		done:      make(chan struct{}),
	}
}