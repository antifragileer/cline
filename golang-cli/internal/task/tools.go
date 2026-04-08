// Package task provides task execution and tool management functionality.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/cline/cline/golang-cli/internal/security"
)

// ToolType represents the type of tool being requested.
type ToolType string

const (
	// ToolTypeReadFile requests reading a file.
	ToolTypeReadFile ToolType = "read_file"
	// ToolTypeWriteFile requests writing a file.
	ToolTypeWriteFile ToolType = "write_file"
	// ToolTypeReplaceInFile requests replacing content in a file.
	ToolTypeReplaceInFile ToolType = "replace_in_file"
	// ToolTypeExecuteCommand requests executing a shell command.
	ToolTypeExecuteCommand ToolType = "execute_command"
	// ToolTypeSearchFiles requests searching files.
	ToolTypeSearchFiles ToolType = "search_files"
	// ToolTypeListFiles requests listing files in a directory.
	ToolTypeListFiles ToolType = "list_files"
	// ToolTypeListCodeDefinitionNames requests listing code definitions.
	ToolTypeListCodeDefinitionNames ToolType = "list_code_definition_names"
)

// ToolRequest represents a request to execute a tool.
type ToolRequest struct {
	// ID is the unique identifier for this tool request.
	ID string `json:"id"`
	// Type is the type of tool being requested.
	Type ToolType `json:"type"`
	// ToolName is the name of the tool (e.g., "read_file", "execute_command").
	ToolName string `json:"tool_name"`
	// Parameters contains the tool-specific parameters.
	Parameters map[string]interface{} `json:"parameters"`
	// Description is a human-readable description of what the tool will do.
	Description string `json:"description"`
	// Timestamp is when the request was created.
	Timestamp time.Time `json:"timestamp"`
	// Timeout is the maximum time to wait for approval/execution.
	Timeout time.Duration `json:"timeout"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	// RequestID is the ID of the original tool request.
	RequestID string `json:"request_id"`
	// Success indicates whether the tool execution was successful.
	Success bool `json:"success"`
	// Output contains the output from the tool execution.
	Output string `json:"output"`
	// Error contains error information if execution failed.
	Error string `json:"error,omitempty"`
	// ExitCode is the exit code for command execution tools.
	ExitCode int `json:"exit_code,omitempty"`
	// Duration is how long the execution took.
	Duration time.Duration `json:"duration"`
	// Timestamp is when the result was created.
	Timestamp time.Time `json:"timestamp"`
}

// ToolApprovalConfig configures tool approval behavior.
type ToolApprovalConfig struct {
	// YoloMode enables auto-approval without confirmation.
	YoloMode bool `json:"yolo_mode"`
	// AutoApproveTools is a list of tool types that can be auto-approved.
	AutoApproveTools []ToolType `json:"auto_approve_tools"`
	// ApprovalTimeout is the maximum time to wait for user approval.
	ApprovalTimeout time.Duration `json:"approval_timeout"`
	// MaxRetries is the maximum number of retries for failed tool executions.
	MaxRetries int `json:"max_retries"`
	// RetryDelay is the delay between retries.
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultToolApprovalConfig returns the default tool approval configuration.
func DefaultToolApprovalConfig() *ToolApprovalConfig {
	return &ToolApprovalConfig{
		YoloMode:         false,
		AutoApproveTools: []ToolType{},
		ApprovalTimeout:  5 * time.Minute,
		MaxRetries:       3,
		RetryDelay:       time.Second,
	}
}

// ToolExecutor handles tool execution with approval workflows.
type ToolExecutor struct {
	config     *ToolApprovalConfig
	approver   ToolApprover
	grpcConn   *grpc.ClientConn
	history    []ToolExecutionRecord
	maxHistory int
	mu         chan struct{}
}

// ToolExecutionRecord records a tool execution for history.
type ToolExecutionRecord struct {
	Request   ToolRequest `json:"request"`
	Result    ToolResult  `json:"result"`
	Approved  bool        `json:"approved"`
	Timestamp time.Time   `json:"timestamp"`
}

// ToolApprover defines the interface for tool approval.
type ToolApprover interface {
	// RequestApproval requests approval for a tool and returns true if approved.
	RequestApproval(ctx context.Context, req ToolRequest) (bool, error)
	// DisplayToolRequest displays the tool request to the user.
	DisplayToolRequest(req ToolRequest) error
}

// NewToolExecutor creates a new tool executor with the given configuration.
func NewToolExecutor(config *ToolApprovalConfig, approver ToolApprover) *ToolExecutor {
	if config == nil {
		config = DefaultToolApprovalConfig()
	}

	return &ToolExecutor{
		config:     config,
		approver:   approver,
		history:    make([]ToolExecutionRecord, 0, 100),
		maxHistory: 100,
		mu:         make(chan struct{}, 1),
	}
}

// SetGRPCConnection sets the gRPC connection for sending results.
func (e *ToolExecutor) SetGRPCConnection(conn *grpc.ClientConn) {
	e.grpcConn = conn
}

// ExecuteTool executes a tool with approval workflow.
func (e *ToolExecutor) ExecuteTool(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	// Acquire lock for thread-safe execution
	select {
	case e.mu <- struct{}{}:
		defer func() { <-e.mu }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Validate request
	if err := e.validateRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid tool request: %w", err)
	}

	// Set defaults
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}
	if req.Timeout == 0 {
		req.Timeout = e.config.ApprovalTimeout
	}

	// Check if auto-approved
	approved := e.isAutoApproved(req)
	var approvalErr error

	if !approved {
		// Request approval from user
		approvalCtx, cancel := context.WithTimeout(ctx, e.config.ApprovalTimeout)
		defer cancel()

		approved, approvalErr = e.approver.RequestApproval(approvalCtx, req)
		if approvalErr != nil {
			return nil, fmt.Errorf("approval request failed: %w", approvalErr)
		}
	}

	// Record the approval decision
	record := ToolExecutionRecord{
		Request:   req,
		Approved:  approved,
		Timestamp: time.Now(),
	}

	if !approved {
		record.Result = ToolResult{
			RequestID: req.ID,
			Success:   false,
			Error:     "tool execution rejected by user",
			Timestamp: time.Now(),
		}
		e.addToHistory(record)
		return &record.Result, fmt.Errorf("tool execution rejected")
	}

	// Execute the tool with retries
	result, err := e.executeWithRetries(ctx, req)
	record.Result = *result
	e.addToHistory(record)

	// Send result via gRPC if connection is available
	if e.grpcConn != nil {
		if sendErr := e.sendResultViaGRPC(ctx, result); sendErr != nil {
			// Log but don't fail - the result is still returned
			fmt.Fprintf(os.Stderr, "failed to send result via gRPC: %v\n", sendErr)
		}
	}

	return result, err
}

// validateRequest validates a tool request.
func (e *ToolExecutor) validateRequest(req *ToolRequest) error {
	if req.ID == "" {
		return fmt.Errorf("tool request ID is required")
	}
	if req.ToolName == "" {
		return fmt.Errorf("tool name is required")
	}
	if req.Type == "" {
		return fmt.Errorf("tool type is required")
	}

	// Validate tool type
	validTypes := []ToolType{
		ToolTypeReadFile,
		ToolTypeWriteFile,
		ToolTypeReplaceInFile,
		ToolTypeExecuteCommand,
		ToolTypeSearchFiles,
		ToolTypeListFiles,
		ToolTypeListCodeDefinitionNames,
	}

	valid := false
	for _, t := range validTypes {
		if req.Type == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid tool type: %s", req.Type)
	}

	return nil
}

// isAutoApproved checks if a tool request should be auto-approved.
func (e *ToolExecutor) isAutoApproved(req ToolRequest) bool {
	// Yolo mode auto-approves everything
	if e.config.YoloMode {
		return true
	}

	// Check if this tool type is in the auto-approve list
	for _, t := range e.config.AutoApproveTools {
		if t == req.Type {
			return true
		}
	}

	return false
}

// executeWithRetries executes a tool with retry logic.
func (e *ToolExecutor) executeWithRetries(ctx context.Context, req ToolRequest) (*ToolResult, error) {
	var result *ToolResult
	var lastErr error

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return &ToolResult{
					RequestID: req.ID,
					Success:   false,
					Error:     fmt.Sprintf("context cancelled during retry: %v", ctx.Err()),
					Timestamp: time.Now(),
				}, ctx.Err()
			case <-time.After(e.config.RetryDelay):
			}
		}

		result = e.executeToolInternal(ctx, req)

		if result.Success {
			return result, nil
		}

		lastErr = fmt.Errorf("%s", result.Error)
	}

	// All retries exhausted
	if lastErr != nil {
		result.Error = fmt.Sprintf("max retries (%d) exceeded: %s", e.config.MaxRetries, result.Error)
	}

	return result, lastErr
}

// executeToolInternal executes the tool based on its type.
func (e *ToolExecutor) executeToolInternal(ctx context.Context, req ToolRequest) *ToolResult {
	startTime := time.Now()
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Create timeout context if not already set
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	switch req.Type {
	case ToolTypeReadFile:
		result = e.executeReadFile(execCtx, req)
	case ToolTypeWriteFile:
		result = e.executeWriteFile(execCtx, req)
	case ToolTypeReplaceInFile:
		result = e.executeReplaceInFile(execCtx, req)
	case ToolTypeExecuteCommand:
		result = e.executeCommand(execCtx, req)
	case ToolTypeSearchFiles:
		result = e.executeSearchFiles(execCtx, req)
	case ToolTypeListFiles:
		result = e.executeListFiles(execCtx, req)
	case ToolTypeListCodeDefinitionNames:
		result = e.executeListCodeDefinitionNames(execCtx, req)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported tool type: %s", req.Type)
	}

	return result
}

// executeReadFile executes a read_file tool.
func (e *ToolExecutor) executeReadFile(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		result.Success = false
		result.Error = "path parameter is required"
		return result
	}

	// Resolve path
	path = e.resolvePath(path)

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to stat file: %v", err)
		return result
	}

	if info.IsDir() {
		result.Success = false
		result.Error = "path is a directory, not a file"
		return result
	}

	// Check file size (limit to 10MB)
	if info.Size() > 10*1024*1024 {
		result.Success = false
		result.Error = "file too large (>10MB)"
		return result
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result
	}

	result.Success = true
	result.Output = string(content)
	return result
}

// executeWriteFile executes a write_file tool.
func (e *ToolExecutor) executeWriteFile(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		result.Success = false
		result.Error = "path parameter is required"
		return result
	}

	content, ok := req.Parameters["content"].(string)
	if !ok {
		result.Success = false
		result.Error = "content parameter is required"
		return result
	}

	// Resolve path
	path = e.resolvePath(path)

	// Ensure parent directory exists
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
		return result
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("File written successfully: %s (%d bytes)", path, len(content))
	return result
}

// executeReplaceInFile executes a replace_in_file tool.
func (e *ToolExecutor) executeReplaceInFile(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		result.Success = false
		result.Error = "path parameter is required"
		return result
	}

	// Support both old and new diff formats
	diff, _ := req.Parameters["diff"].(string)
	oldStr, _ := req.Parameters["old_string"].(string)
	newStr, _ := req.Parameters["new_string"].(string)

	if diff == "" && (oldStr == "" || newStr == "") {
		result.Success = false
		result.Error = "either diff or old_string/new_string parameters are required"
		return result
	}

	// Resolve path
	path = e.resolvePath(path)

	// Read existing content
	content, err := os.ReadFile(path)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result
	}

	// Perform replacement
	var newContent string
	if diff != "" {
		newContent, err = e.applyDiff(string(content), diff)
	} else {
		newContent = strings.Replace(string(content), oldStr, newStr, 1)
		if newContent == string(content) {
			err = fmt.Errorf("old_string not found in file")
		}
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to apply replacement: %v", err)
		return result
	}

	// Write back
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to write file: %v", err)
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("File modified successfully: %s", path)
	return result
}

// applyDiff applies a unified diff to content.
func (e *ToolExecutor) applyDiff(content, diff string) (string, error) {
	lines := strings.Split(content, "\n")
	diffLines := strings.Split(diff, "\n")

	for i, diffLine := range diffLines {
		if strings.HasPrefix(diffLine, "---") || strings.HasPrefix(diffLine, "+++") {
			continue
		}
		if strings.HasPrefix(diffLine, "@@") {
			continue
		}
		if strings.HasPrefix(diffLine, "-") && !strings.HasPrefix(diffLine, "---") {
			// Remove line
			lineToRemove := strings.TrimPrefix(diffLine, "-")
			for j, line := range lines {
				if line == lineToRemove {
					lines = append(lines[:j], lines[j+1:]...)
					break
				}
			}
		} else if strings.HasPrefix(diffLine, "+") && !strings.HasPrefix(diffLine, "+++") {
			// Add line at appropriate position
			lineToAdd := strings.TrimPrefix(diffLine, "+")
			// Simple approach: add after previous context line
			if i > 0 && len(lines) > 0 {
				lines = append(lines, lineToAdd)
			} else {
				lines = append([]string{lineToAdd}, lines...)
			}
		}
	}

	return strings.Join(lines, "\n"), nil
}

// executeCommand executes an execute_command tool.
func (e *ToolExecutor) executeCommand(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	command, ok := req.Parameters["command"].(string)
	if !ok || command == "" {
		result.Success = false
		result.Error = "command parameter is required"
		return result
	}

	// Validate command against CLINE_COMMAND_PERMISSIONS environment variable
	if allowed, reason := security.ValidateCommand(command); !allowed {
		result.Success = false
		result.Error = fmt.Sprintf("command not allowed: %s", reason)
		result.ExitCode = 1
		return result
	}

	cwd, _ := req.Parameters["cwd"].(string)
	if cwd != "" {
		cwd = e.resolvePath(cwd)
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if cwd != "" {
		cmd.Dir = cwd
	}

	// Capture output
	output, err := cmd.CombinedOutput()

	result.ExitCode = cmd.ProcessState.ExitCode()
	if err != nil && result.ExitCode != 0 {
		result.Success = false
		result.Error = fmt.Sprintf("command failed with exit code %d: %v", result.ExitCode, err)
		result.Output = string(output)
		return result
	}

	result.Success = true
	result.Output = string(output)
	return result
}

// executeSearchFiles executes a search_files tool.
func (e *ToolExecutor) executeSearchFiles(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		path = "."
	}
	path = e.resolvePath(path)

	regex, ok := req.Parameters["regex"].(string)
	if !ok || regex == "" {
		result.Success = false
		result.Error = "regex parameter is required"
		return result
	}

	filePattern, _ := req.Parameters["file_pattern"].(string)

	// Use find and grep for searching
	var cmd *exec.Cmd
	if filePattern != "" {
		cmd = exec.CommandContext(ctx, "find", path, "-type", "f", "-name", filePattern, "-exec", "grep", "-l", "-E", regex, "{}", "+")
	} else {
		cmd = exec.CommandContext(ctx, "grep", "-r", "-l", "-E", regex, path)
	}

	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		result.Success = false
		result.Error = fmt.Sprintf("search failed: %v", err)
		return result
	}

	result.Success = true
	result.Output = string(output)
	return result
}

// executeListFiles executes a list_files tool.
func (e *ToolExecutor) executeListFiles(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		path = "."
	}
	path = e.resolvePath(path)

	recursive, _ := req.Parameters["recursive"].(bool)

	var cmd *exec.Cmd
	if recursive {
		cmd = exec.CommandContext(ctx, "find", path, "-type", "f")
	} else {
		cmd = exec.CommandContext(ctx, "ls", "-la", path)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to list files: %v", err)
		return result
	}

	result.Success = true
	result.Output = string(output)
	return result
}

// executeListCodeDefinitionNames executes a list_code_definition_names tool.
func (e *ToolExecutor) executeListCodeDefinitionNames(ctx context.Context, req ToolRequest) *ToolResult {
	result := &ToolResult{
		RequestID: req.ID,
		Timestamp: time.Now(),
	}

	path, ok := req.Parameters["path"].(string)
	if !ok || path == "" {
		path = "."
	}
	path = e.resolvePath(path)

	// Use grep to find function and class definitions
	cmd := exec.CommandContext(ctx, "grep", "-r", "-n", "-E", "^(func|class|interface|struct|type)\\s+", path)

	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		result.Success = false
		result.Error = fmt.Sprintf("failed to list definitions: %v", err)
		return result
	}

	result.Success = true
	result.Output = string(output)
	return result
}

// resolvePath resolves a relative path to an absolute path.
func (e *ToolExecutor) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// Try to resolve relative to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	return filepath.Join(cwd, path)
}

// addToHistory adds a record to the execution history.
// NOTE: This method assumes the caller already holds the lock (e.mu)
func (e *ToolExecutor) addToHistory(record ToolExecutionRecord) {
	e.history = append(e.history, record)
	if len(e.history) > e.maxHistory {
		e.history = e.history[len(e.history)-e.maxHistory:]
	}
}

// GetHistory returns the execution history.
func (e *ToolExecutor) GetHistory() []ToolExecutionRecord {
	select {
	case e.mu <- struct{}{}:
		defer func() { <-e.mu }()
		historyCopy := make([]ToolExecutionRecord, len(e.history))
		copy(historyCopy, e.history)
		return historyCopy
	default:
		return nil
	}
}

// sendResultViaGRPC sends the tool result via gRPC.
func (e *ToolExecutor) sendResultViaGRPC(ctx context.Context, result *ToolResult) error {
	if e.grpcConn == nil {
		return fmt.Errorf("no gRPC connection available")
	}

	// This would use a generated gRPC client
	// For now, serialize as JSON for demonstration
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// In a real implementation, this would call a gRPC method
	// client := pb.NewToolServiceClient(e.grpcConn)
	// _, err := client.SendToolResult(ctx, &pb.ToolResultRequest{Result: resultJSON})

	fmt.Fprintf(os.Stderr, "Sending result via gRPC: %s\n", string(resultJSON))
	return nil
}

// BatchExecuteTools executes multiple tools with proper ordering and error handling.
func (e *ToolExecutor) BatchExecuteTools(ctx context.Context, requests []ToolRequest) ([]*ToolResult, error) {
	results := make([]*ToolResult, 0, len(requests))
	var errors []string

	for _, req := range requests {
		result, err := e.ExecuteTool(ctx, req)
		results = append(results, result)

		if err != nil {
			errors = append(errors, fmt.Sprintf("tool %s failed: %v", req.ToolName, err))
			// Continue with next tool - don't stop on error
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("batch execution had errors: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// CancelPendingRequests cancels any pending tool requests.
func (e *ToolExecutor) CancelPendingRequests() {
	// Signal cancellation for pending operations
	// In a more complex implementation, this would use a context cancel function
}

// GetPendingCount returns the number of pending tool requests.
func (e *ToolExecutor) GetPendingCount() int {
	return 0 // Placeholder - would track pending requests in a real implementation
}

// IsYoloMode returns true if yolo mode (auto-approve) is enabled.
func (e *ToolExecutor) IsYoloMode() bool {
	return e.config.YoloMode
}

// SetYoloMode enables or disables yolo mode.
func (e *ToolExecutor) SetYoloMode(enabled bool) {
	e.config.YoloMode = enabled
}

// AddAutoApproveTool adds a tool type to the auto-approve list.
func (e *ToolExecutor) AddAutoApproveTool(toolType ToolType) {
	for _, t := range e.config.AutoApproveTools {
		if t == toolType {
			return // Already in list
		}
	}
	e.config.AutoApproveTools = append(e.config.AutoApproveTools, toolType)
}

// RemoveAutoApproveTool removes a tool type from the auto-approve list.
func (e *ToolExecutor) RemoveAutoApproveTool(toolType ToolType) {
	filtered := make([]ToolType, 0, len(e.config.AutoApproveTools))
	for _, t := range e.config.AutoApproveTools {
		if t != toolType {
			filtered = append(filtered, t)
		}
	}
	e.config.AutoApproveTools = filtered
}
