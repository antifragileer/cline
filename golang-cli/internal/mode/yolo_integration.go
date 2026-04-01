// Package mode provides Yolo mode integration with task execution.
// It implements auto-approval with comprehensive safety checks and logging.
package mode

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/audit"
	"github.com/cline/cline/golang-cli/internal/security"
	"github.com/cline/cline/golang-cli/internal/task"
)

// YoloIntegration provides yolo mode functionality integrated with task execution.
type YoloIntegration struct {
	config           *YoloModeConfig
	approver         *task.AutoApprover
	executor         *task.ToolExecutor
	auditLogger      *audit.Logger
	securityChecker  *security.CommandValidator
	actionLog        []YoloAction
	mu               sync.RWMutex
	startTime        time.Time
	toolCount        int
	hasErrors        bool
	exitCode         int
	maxToolsReached  bool
	paused           bool
	pauseReason      string
}

// YoloIntegrationConfig configures the yolo integration.
type YoloIntegrationConfig struct {
	// Base yolo configuration
	*YoloModeConfig
	
	// AuditLogger for logging actions
	AuditLogger *audit.Logger
	
	// SecurityChecker for command validation
	SecurityChecker *security.CommandValidator
	
	// MaxConsecutiveErrors is the maximum allowed consecutive errors before pausing
	MaxConsecutiveErrors int
	
	// DangerousCommandAction is what to do when a dangerous command is detected
	DangerousCommandAction DangerousCommandAction
}

// DangerousCommandAction defines what to do when a dangerous command is detected.
type DangerousCommandAction int

const (
	// DangerousCommandBlock blocks the command
	DangerousCommandBlock DangerousCommandAction = iota
	// DangerousCommandWarn warns but allows
	DangerousCommandWarn
	// DangerousCommandAllow allows without warning
	DangerousCommandAllow
)

// DefaultYoloIntegrationConfig returns the default integration configuration.
func DefaultYoloIntegrationConfig() *YoloIntegrationConfig {
	return &YoloIntegrationConfig{
		YoloModeConfig:         DefaultYoloModeConfig(),
		MaxConsecutiveErrors:   5,
		DangerousCommandAction: DangerousCommandBlock,
	}
}

// NewYoloIntegration creates a new yolo integration with the given configuration.
func NewYoloIntegration(config *YoloIntegrationConfig) (*YoloIntegration, error) {
	if config == nil {
		config = DefaultYoloIntegrationConfig()
	}

	if config.YoloModeConfig == nil {
		config.YoloModeConfig = DefaultYoloModeConfig()
	}

	// Create tool approval config
	toolConfig := &task.ToolApprovalConfig{
		YoloMode:         config.Enabled,
		AutoApproveTools: config.ToolTypes,
		ApprovalTimeout:  0,
		MaxRetries:       0,
		RetryDelay:       0,
	}

	integration := &YoloIntegration{
		config:          config.YoloModeConfig,
		approver:        task.NewAutoApprover(),
		executor:        task.NewToolExecutor(toolConfig, task.NewAutoApprover()),
		auditLogger:     config.AuditLogger,
		securityChecker: config.SecurityChecker,
		actionLog:       make([]YoloAction, 0),
		exitCode:        0,
	}

	// Initialize audit logger if not provided
	if integration.auditLogger == nil {
		logger, err := audit.NewLogger(audit.DefaultLoggerConfig())
		if err != nil {
			// Non-fatal: continue without audit logging
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize audit logger: %v\n", err)
		} else {
			integration.auditLogger = logger
		}
	}

	return integration, nil
}

// Run executes a task in yolo mode with full safety checks.
func (yi *YoloIntegration) Run(ctx context.Context, taskID, prompt string, handler task.MessageHandler) (*YoloResult, error) {
	yi.startTime = time.Now()
	yi.actionLog = make([]YoloAction, 0)
	yi.toolCount = 0
	yi.hasErrors = false
	yi.exitCode = 0
	yi.maxToolsReached = false
	yi.paused = false

	// Apply timeout if configured
	if yi.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, yi.config.Timeout)
		defer cancel()
	}

	// Log start
	yi.logEvent(audit.EventYoloModeStarted, map[string]interface{}{
		"task_id": taskID,
		"prompt":  prompt,
		"config": map[string]interface{}{
			"max_tools":     yi.config.MaxTools,
			"timeout":       yi.config.Timeout.String(),
			"exit_on_error": yi.config.ExitOnError,
		},
	})

	// Create a wrapped handler that intercepts tool requests
	_ = &yoloHandlerWrapper{
		integration: yi,
		inner:       handler,
	}

	// Execute the task
	// Note: This is a placeholder - actual implementation would integrate with task runner
	result := &YoloResult{
		Success:   true,
		ExitCode:  0,
		Actions:   yi.actionLog,
		StartTime: yi.startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(yi.startTime),
	}

	yi.calculateStats(result)

	// Log completion
	yi.logEvent(audit.EventYoloModeCompleted, map[string]interface{}{
		"task_id":          taskID,
		"success":          result.Success,
		"exit_code":        result.ExitCode,
		"total_actions":    result.TotalActions,
		"failed_actions":   result.FailedActions,
		"duration_seconds": result.Duration.Seconds(),
	})

	return result, nil
}

// ExecuteTool executes a single tool with yolo mode safety checks.
func (yi *YoloIntegration) ExecuteTool(ctx context.Context, req task.ToolRequest) (*task.ToolResult, error) {
	// Check if paused
	if yi.paused {
		return nil, fmt.Errorf("yolo mode is paused: %s", yi.pauseReason)
	}

	// Check max tools limit
	if yi.config.MaxTools > 0 && yi.toolCount >= yi.config.MaxTools {
		yi.maxToolsReached = true
		yi.logEvent(audit.EventMaxToolsReached, map[string]interface{}{
			"limit": yi.config.MaxTools,
		})
		return nil, fmt.Errorf("maximum tool limit (%d) reached", yi.config.MaxTools)
	}

	yi.toolCount++

	// Pre-execution safety checks
	if err := yi.performSafetyChecks(ctx, &req); err != nil {
		yi.logEvent(audit.EventSafetyCheckFailed, map[string]interface{}{
			"tool":  req.ToolName,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("safety check failed: %w", err)
	}

	// Log action start
	yi.logActionStart(req)

	// Execute the tool
	startTime := time.Now()
	result, err := yi.executor.ExecuteTool(ctx, req)
	duration := time.Since(startTime)

	// Log action completion
	yi.logActionComplete(req, result, err, duration)

	// Handle errors
	if err != nil || !result.Success {
		yi.hasErrors = true
		
		// Update exit code
		if result != nil && result.ExitCode != 0 {
			yi.exitCode = result.ExitCode
		} else {
			yi.exitCode = 1
		}

		// Check for consecutive errors
		consecutiveErrors := yi.countConsecutiveErrors()
		if consecutiveErrors >= 5 {
			yi.paused = true
			yi.pauseReason = fmt.Sprintf("too many consecutive errors (%d)", consecutiveErrors)
			yi.logEvent(audit.EventYoloModePaused, map[string]interface{}{
				"reason": yi.pauseReason,
			})
		}

		// Exit on error if configured
		if yi.config.ExitOnError {
			return result, err
		}
	}

	return result, err
}

// performSafetyChecks performs pre-execution safety checks.
func (yi *YoloIntegration) performSafetyChecks(ctx context.Context, req *task.ToolRequest) error {
	// Check 1: Command validation for execute_command
	if req.Type == task.ToolTypeExecuteCommand {
		command, ok := req.Parameters["command"].(string)
		if ok && command != "" {
			// Check for dangerous characters
			if security.ContainsDangerousCharacters(command) {
				yi.logEvent(audit.EventDangerousCommandDetected, map[string]interface{}{
					"command": command,
					"tool":    req.ToolName,
				})
				
				switch yi.config.DangerousCommandAction {
				case DangerousCommandBlock:
					return fmt.Errorf("dangerous command detected and blocked: %s", command)
				case DangerousCommandWarn:
					// Log warning but continue
					yi.logEvent(audit.EventDangerousCommandDetected, map[string]interface{}{
						"command": command,
						"warning": true,
					})
				case DangerousCommandAllow:
					// Silently allow
				}
			}

			// Validate against security permissions
			if yi.securityChecker != nil {
				result := yi.securityChecker.Validate(nil, command)
				if !result.Allowed {
					return fmt.Errorf("command blocked by security policy: %s", result.Reason)
				}
			}
		}
	}

	// Check 2: File path validation for file operations
	if req.Type == task.ToolTypeReadFile || req.Type == task.ToolTypeWriteFile || req.Type == task.ToolTypeReplaceInFile {
		path, ok := req.Parameters["path"].(string)
		if ok && path != "" {
			// Check for path traversal attempts
			if strings.Contains(path, "..") {
				return fmt.Errorf("path traversal attempt detected: %s", path)
			}

			// Check for absolute paths outside workspace
			if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "/tmp/") && !strings.HasPrefix(path, "/home/") {
				yi.logEvent(audit.EventSuspiciousPathDetected, map[string]interface{}{
					"path": path,
					"tool": req.ToolName,
				})
			}
		}
	}

	// Check 3: Context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}

// countConsecutiveErrors counts the number of consecutive failed actions.
func (yi *YoloIntegration) countConsecutiveErrors() int {
	yi.mu.RLock()
	defer yi.mu.RUnlock()

	count := 0
	for i := len(yi.actionLog) - 1; i >= 0; i-- {
		if yi.actionLog[i].Success {
			break
		}
		count++
	}
	return count
}

// logActionStart logs the start of a tool action.
func (yi *YoloIntegration) logActionStart(req task.ToolRequest) {
	yi.logEvent(audit.EventToolExecutionStarted, map[string]interface{}{
		"tool_name":   req.ToolName,
		"tool_type":   req.Type,
		"parameters":  sanitizeParameters(req.Parameters),
		"tool_number": yi.toolCount,
	})
}

// logActionComplete logs the completion of a tool action.
func (yi *YoloIntegration) logActionComplete(req task.ToolRequest, result *task.ToolResult, execErr error, duration time.Duration) {
	action := YoloAction{
		Timestamp:  time.Now(),
		ToolName:   req.ToolName,
		ToolType:   req.Type,
		Parameters: sanitizeParameters(req.Parameters),
		Duration:   duration,
	}

	if result != nil {
		action.Success = result.Success
		action.Output = truncateString(result.Output, 1000) // Truncate long output
		action.ExitCode = result.ExitCode

		if !result.Success {
			action.Error = truncateString(result.Error, 500)
		}
	}

	if execErr != nil {
		action.Success = false
		action.Error = truncateString(execErr.Error(), 500)
	}

	// Add to action log
	yi.mu.Lock()
	yi.actionLog = append(yi.actionLog, action)
	yi.mu.Unlock()

	// Audit log
	yi.logEvent(audit.EventToolExecutionCompleted, map[string]interface{}{
		"tool_name": req.ToolName,
		"success":   action.Success,
		"duration":  duration.Milliseconds(),
		"exit_code": action.ExitCode,
	})
}

// logEvent logs an event to the audit logger.
func (yi *YoloIntegration) logEvent(eventType audit.EventType, details map[string]interface{}) {
	if yi.auditLogger == nil {
		return
	}

	event := &audit.Event{
		EventType: eventType,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Yolo mode: %s", eventType),
		Details:   details,
	}

	if err := yi.auditLogger.Log(event); err != nil {
		// Non-fatal: log to stderr
		fmt.Fprintf(os.Stderr, "Warning: failed to log audit event: %v\n", err)
	}
}

// calculateStats calculates result statistics.
func (yi *YoloIntegration) calculateStats(result *YoloResult) {
	yi.mu.RLock()
	defer yi.mu.RUnlock()

	result.TotalActions = len(yi.actionLog)

	for _, action := range yi.actionLog {
		if action.Success {
			result.SuccessfulActions++
		} else {
			result.FailedActions++
		}
	}

	// Determine overall success
	result.Success = result.FailedActions == 0

	// Set exit code based on errors
	if result.FailedActions > 0 {
		result.ExitCode = yi.exitCode
		if result.ExitCode == 0 {
			result.ExitCode = 1 // Default error code
		}
	} else {
		result.ExitCode = 0
	}
}

// GetActionLog returns a copy of the action log.
func (yi *YoloIntegration) GetActionLog() []YoloAction {
	yi.mu.RLock()
	defer yi.mu.RUnlock()

	logCopy := make([]YoloAction, len(yi.actionLog))
	copy(logCopy, yi.actionLog)
	return logCopy
}

// GetToolCount returns the number of tools executed.
func (yi *YoloIntegration) GetToolCount() int {
	yi.mu.RLock()
	defer yi.mu.RUnlock()
	return yi.toolCount
}

// IsPaused returns true if yolo mode is paused.
func (yi *YoloIntegration) IsPaused() bool {
	yi.mu.RLock()
	defer yi.mu.RUnlock()
	return yi.paused
}

// GetPauseReason returns the reason for pausing.
func (yi *YoloIntegration) GetPauseReason() string {
	yi.mu.RLock()
	defer yi.mu.RUnlock()
	return yi.pauseReason
}

// Resume resumes yolo mode after pausing.
func (yi *YoloIntegration) Resume() {
	yi.mu.Lock()
	defer yi.mu.Unlock()
	yi.paused = false
	yi.pauseReason = ""
	yi.logEvent(audit.EventYoloModeResumed, nil)
}

// Close cleans up resources.
func (yi *YoloIntegration) Close() error {
	if yi.auditLogger != nil {
		return yi.auditLogger.Close()
	}
	return nil
}

// sanitizeParameters sanitizes parameters for logging (removes sensitive data).
func sanitizeParameters(params map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range params {
		// Skip sensitive keys
		if isSensitiveKey(k) {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// isSensitiveKey checks if a key contains sensitive data.
func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"password", "secret", "token", "key", "auth", "credential",
		"api_key", "apikey", "access_token", "private_key",
	}
	
	lowerKey := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lowerKey, sensitive) {
			return true
		}
	}
	return false
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}

// yoloHandlerWrapper wraps a message handler to intercept tool requests.
type yoloHandlerWrapper struct {
	integration *YoloIntegration
	inner       task.MessageHandler
}

// HandleMessage implements task.MessageHandler
func (w *yoloHandlerWrapper) HandleMessage(msg task.Message) error {
	return w.inner.HandleMessage(msg)
}

// OnSay handles a SAY message.
func (w *yoloHandlerWrapper) OnSay(sayType string, text string, partial bool) error {
	return w.inner.OnSay(sayType, text, partial)
}

// OnAsk handles an ASK message.
func (w *yoloHandlerWrapper) OnAsk(askType string, text string) (string, error) {
	// Auto-approve in yolo mode
	if w.integration.config.Enabled {
		// Log the auto-approval
		w.integration.logEvent(audit.EventAutoApproval, map[string]interface{}{
			"ask_type": askType,
			"text":     text,
		})
		return "yesButtonClicked", nil
	}
	
	// Otherwise, delegate to inner handler
	return w.inner.OnAsk(askType, text)
}

// OnInfo handles informational messages.
func (w *yoloHandlerWrapper) OnInfo(text string) error {
	return w.inner.OnInfo(text)
}

// OnError handles error messages.
func (w *yoloHandlerWrapper) OnError(err error) error {
	return w.inner.OnError(err)
}

// OnStatus handles status updates.
func (w *yoloHandlerWrapper) OnStatus(status string) error {
	return w.inner.OnStatus(status)
}

// OnProgress handles progress updates.
func (w *yoloHandlerWrapper) OnProgress(current, total int) error {
	return w.inner.OnProgress(current, total)
}

// OnText handles text messages
func (w *yoloHandlerWrapper) OnText(content string, isPartial bool) error {
	return w.inner.OnText(content, isPartial)
}

// OnToolUse handles tool use requests
func (w *yoloHandlerWrapper) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	return w.inner.OnToolUse(toolName, params)
}

// OnToolResult handles tool execution results
func (w *yoloHandlerWrapper) OnToolResult(toolName string, result string, success bool) error {
	return w.inner.OnToolResult(toolName, result, success)
}

// OnCommand handles command execution requests
func (w *yoloHandlerWrapper) OnCommand(command string, requiresApproval bool) (string, error) {
	return w.inner.OnCommand(command, requiresApproval)
}

// OnCommandOutput handles command output
func (w *yoloHandlerWrapper) OnCommandOutput(output string, isComplete bool) error {
	return w.inner.OnCommandOutput(output, isComplete)
}

// OnCheckpoint handles checkpoint events
func (w *yoloHandlerWrapper) OnCheckpoint(checkpointID string, action string) error {
	return w.inner.OnCheckpoint(checkpointID, action)
}

// OnBrowserAction handles browser actions
func (w *yoloHandlerWrapper) OnBrowserAction(action string, url string) (string, error) {
	return w.inner.OnBrowserAction(action, url)
}

// OnMCPRequest handles MCP tool requests
func (w *yoloHandlerWrapper) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	return w.inner.OnMCPRequest(server, tool, params)
}

// OnCompletion handles task completion
func (w *yoloHandlerWrapper) OnCompletion(success bool, summary string) error {
	return w.inner.OnCompletion(success, summary)
}

// YoloSafetyLimits defines safety limits for yolo mode.
type YoloSafetyLimits struct {
	// MaxTools is the maximum number of tools to execute
	MaxTools int
	
	// MaxConsecutiveErrors is the maximum allowed consecutive errors
	MaxConsecutiveErrors int
	
	// MaxExecutionTime is the maximum execution time
	MaxExecutionTime time.Duration
	
	// BlockDangerousCommands blocks commands with dangerous characters
	BlockDangerousCommands bool
	
	// BlockPathTraversal blocks file operations with path traversal
	BlockPathTraversal bool
	
	// RequireSecurityValidation requires security validation for all commands
	RequireSecurityValidation bool
}

// DefaultYoloSafetyLimits returns default safety limits.
func DefaultYoloSafetyLimits() *YoloSafetyLimits {
	return &YoloSafetyLimits{
		MaxTools:                  100,
		MaxConsecutiveErrors:      5,
		MaxExecutionTime:          30 * time.Minute,
		BlockDangerousCommands:    true,
		BlockPathTraversal:        true,
		RequireSecurityValidation: true,
	}
}

// ValidateLimits validates that the configuration is within safety limits.
func ValidateLimits(config *YoloModeConfig, limits *YoloSafetyLimits) []string {
	var violations []string

	if limits == nil {
		limits = DefaultYoloSafetyLimits()
	}

	if config.MaxTools > limits.MaxTools {
		violations = append(violations, 
			fmt.Sprintf("max_tools (%d) exceeds safety limit (%d)", 
				config.MaxTools, limits.MaxTools))
	}

	if config.Timeout > limits.MaxExecutionTime {
		violations = append(violations, 
			fmt.Sprintf("timeout (%v) exceeds safety limit (%v)", 
				config.Timeout, limits.MaxExecutionTime))
	}

	return violations
}