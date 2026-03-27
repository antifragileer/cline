// Package security provides command permission control for the Cline CLI.
// This file implements the PermissionEnforcer that integrates permission
// validation with actual command execution.
package security

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cline/cline/golang-cli/internal/audit"
)

// CommandExecutor is the function type for executing commands.
// Implementations should handle the actual command execution.
type CommandExecutor func(ctx context.Context, command string, args []string) (output string, exitCode int, err error)

// ExecutionResult represents the result of a command execution.
type ExecutionResult struct {
	// Output is the command output
	Output string `json:"output"`
	// ExitCode is the command exit code
	ExitCode int `json:"exit_code"`
	// Error is any execution error
	Error error `json:"error,omitempty"`
	// Blocked indicates if the command was blocked by permissions
	Blocked bool `json:"blocked"`
	// BlockReason explains why the command was blocked
	BlockReason string `json:"block_reason,omitempty"`
}

// PermissionEnforcer validates and executes commands with permission checks.
// It wraps a CommandPermissionController and CommandExecutor to provide
// a unified interface for secure command execution.
type PermissionEnforcer struct {
	controller CommandPermissionControllerInterface
	executor   CommandExecutor
	auditor    *audit.Logger
}

// CommandPermissionControllerInterface defines the interface for permission checking.
// This allows for easy mocking in tests.
type CommandPermissionControllerInterface interface {
	ValidateCommand(command string) PermissionValidationResult
	FormatErrorMessage(result PermissionValidationResult, command string) string
	IsEnabled() bool
	GetConfig() *PermissionRules
}

// PermissionEnforcerOption configures the PermissionEnforcer.
type PermissionEnforcerOption func(*PermissionEnforcer)

// WithCommandExecutor sets a custom command executor.
func WithCommandExecutor(executor CommandExecutor) PermissionEnforcerOption {
	return func(pe *PermissionEnforcer) {
		pe.executor = executor
	}
}

// WithAuditor sets the audit logger.
func WithAuditor(auditor *audit.Logger) PermissionEnforcerOption {
	return func(pe *PermissionEnforcer) {
		pe.auditor = auditor
	}
}

// NewPermissionEnforcer creates a new permission enforcer.
func NewPermissionEnforcer(opts ...PermissionEnforcerOption) *PermissionEnforcer {
	pe := &PermissionEnforcer{
		controller: NewCommandPermissionController(),
		executor:   defaultExecutor,
	}

	for _, opt := range opts {
		opt(pe)
	}

	return pe
}

// NewPermissionEnforcerWithController creates an enforcer with a specific controller.
func NewPermissionEnforcerWithController(controller CommandPermissionControllerInterface, opts ...PermissionEnforcerOption) *PermissionEnforcer {
	pe := &PermissionEnforcer{
		controller: controller,
		executor:   defaultExecutor,
	}

	for _, opt := range opts {
		opt(pe)
	}

	return pe
}

// defaultExecutor is the default command executor implementation.
// In production, this should be replaced with a proper executor.
func defaultExecutor(ctx context.Context, command string, args []string) (string, int, error) {
	// This is a placeholder - actual implementation would execute the command
	return "", 0, fmt.Errorf("no executor configured")
}

// Execute validates and executes a command if permitted.
// It first checks permissions, then executes if allowed, and logs the result.
func (pe *PermissionEnforcer) Execute(ctx context.Context, command string, args []string) ExecutionResult {
	// Build full command string for validation
	fullCommand := buildCommandString(command, args)

	// Check permissions
	validationResult := pe.controller.ValidateCommand(fullCommand)

	// Log the attempt
	if pe.auditor != nil {
		pe.logExecutionAttempt(ctx, fullCommand, validationResult)
	}

	// Block if not allowed
	if !validationResult.Allowed {
		blockReason := pe.controller.FormatErrorMessage(validationResult, fullCommand)
		
		if pe.auditor != nil {
			pe.logBlockedExecution(ctx, fullCommand, blockReason)
		}

		return ExecutionResult{
			Output:      "",
			ExitCode:    1,
			Error:       fmt.Errorf("%s", blockReason),
			Blocked:     true,
			BlockReason: blockReason,
		}
	}

	// Execute the command
	output, exitCode, err := pe.executor(ctx, command, args)

	// Log the execution
	if pe.auditor != nil {
		pe.logExecutionSuccess(ctx, fullCommand, output, exitCode)
	}

	return ExecutionResult{
		Output:   output,
		ExitCode: exitCode,
		Error:    err,
		Blocked:  false,
	}
}

// ExecuteWithValidation executes with detailed validation output.
// This is useful for dry-run scenarios or detailed error reporting.
func (pe *PermissionEnforcer) ExecuteWithValidation(ctx context.Context, command string, args []string) (ExecutionResult, PermissionValidationResult) {
	fullCommand := buildCommandString(command, args)
	validationResult := pe.controller.ValidateCommand(fullCommand)

	if !validationResult.Allowed {
		blockReason := pe.controller.FormatErrorMessage(validationResult, fullCommand)
		return ExecutionResult{
			ExitCode:    1,
			Error:       fmt.Errorf("%s", blockReason),
			Blocked:     true,
			BlockReason: blockReason,
		}, validationResult
	}

	// Execute if validation passed
	result := pe.Execute(ctx, command, args)
	return result, validationResult
}

// ValidateOnly validates a command without executing it.
// Useful for dry-run or pre-flight checks.
func (pe *PermissionEnforcer) ValidateOnly(command string, args []string) PermissionValidationResult {
	fullCommand := buildCommandString(command, args)
	return pe.controller.ValidateCommand(fullCommand)
}

// IsEnabled returns true if permission enforcement is enabled.
func (pe *PermissionEnforcer) IsEnabled() bool {
	return pe.controller.IsEnabled()
}

// GetConfig returns the current permission configuration.
func (pe *PermissionEnforcer) GetConfig() *PermissionRules {
	return pe.controller.GetConfig()
}

// buildCommandString reconstructs a command string from command and args.
func buildCommandString(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	return command + " " + strings.Join(args, " ")
}

// logExecutionAttempt logs a command execution attempt.
func (pe *PermissionEnforcer) logExecutionAttempt(ctx context.Context, command string, result PermissionValidationResult) {
	userContext := getUserContextFromCtx(ctx)
	taskID := getTaskIDFromCtx(ctx)
	sessionID := getSessionIDFromCtx(ctx)

	eventType := audit.EventCommandExecution
	if !result.Allowed {
		eventType = audit.EventSafetyCheckFailed
	}

	pe.auditor.LogWithContext(
		eventType,
		userContext,
		taskID,
		sessionID,
		fmt.Sprintf("Command execution attempt: %s", command),
		map[string]interface{}{
			"command":   command,
			"allowed":   result.Allowed,
			"reason":    result.Reason,
			"enforced":  pe.IsEnabled(),
		},
	)
}

// logBlockedExecution logs a blocked command execution.
func (pe *PermissionEnforcer) logBlockedExecution(ctx context.Context, command string, reason string) {
	userContext := getUserContextFromCtx(ctx)
	taskID := getTaskIDFromCtx(ctx)
	sessionID := getSessionIDFromCtx(ctx)

	pe.auditor.LogWithContext(
		audit.EventDangerousCommandDetected,
		userContext,
		taskID,
		sessionID,
		fmt.Sprintf("Blocked command: %s", command),
		map[string]interface{}{
			"command": command,
			"reason":  reason,
		},
	)
}

// logExecutionSuccess logs a successful command execution.
func (pe *PermissionEnforcer) logExecutionSuccess(ctx context.Context, command string, output string, exitCode int) {
	userContext := getUserContextFromCtx(ctx)
	taskID := getTaskIDFromCtx(ctx)
	sessionID := getSessionIDFromCtx(ctx)

	// Truncate output for logging if too large
	outputPreview := output
	if len(outputPreview) > 1000 {
		outputPreview = outputPreview[:1000] + "... (truncated)"
	}

	pe.auditor.LogWithContext(
		audit.EventToolExecutionCompleted,
		userContext,
		taskID,
		sessionID,
		fmt.Sprintf("Command executed: %s (exit code: %d)", command, exitCode),
		map[string]interface{}{
			"command":      command,
			"exit_code":    exitCode,
			"output_size":  len(output),
			"output_preview": outputPreview,
		},
	)
}

// Context key types for type-safe context values.
type contextKey string

const (
	userContextKey contextKey = "user_context"
	taskIDKey      contextKey = "task_id"
	sessionIDKey   contextKey = "session_id"
)

// WithUserContext adds user context to a context.
func WithUserContext(ctx context.Context, userContext string) context.Context {
	return context.WithValue(ctx, userContextKey, userContext)
}

// WithTaskID adds task ID to a context.
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, taskIDKey, taskID)
}

// WithSessionID adds session ID to a context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// getUserContextFromCtx extracts user context from context.
func getUserContextFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(userContextKey).(string); ok {
		return v
	}
	// Fallback to current user
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "unknown"
}

// getTaskIDFromCtx extracts task ID from context.
func getTaskIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(taskIDKey).(string); ok {
		return v
	}
	return ""
}

// getSessionIDFromCtx extracts session ID from context.
func getSessionIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey).(string); ok {
		return v
	}
	return ""
}


