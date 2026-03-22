// Package mode provides yolo mode functionality for the Cline CLI.
// Yolo mode enables auto-approval of all tool requests, suppresses confirmation
// prompts, runs tasks to completion, and returns appropriate exit codes.
package mode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
)

// YoloModeConfig configures yolo mode behavior.
type YoloModeConfig struct {
	// Enabled enables yolo mode (auto-approve all tools)
	Enabled bool

	// JSONOutput enables JSON output format
	JSONOutput bool

	// Logger is the output writer for logging actions
	Logger io.Writer

	// ExitOnError exits immediately on first error if true
	ExitOnError bool

	// MaxTools is the maximum number of tools to execute (0 = unlimited)
	MaxTools int

	// Timeout is the maximum duration for task execution
	Timeout time.Duration

	// ToolTypes is a list of specific tool types to auto-approve (empty = all)
	ToolTypes []task.ToolType
}

// DefaultYoloModeConfig returns the default yolo mode configuration.
func DefaultYoloModeConfig() *YoloModeConfig {
	return &YoloModeConfig{
		Enabled:     true,
		JSONOutput:  false,
		Logger:      os.Stdout,
		ExitOnError: false,
		MaxTools:    0,
		Timeout:     0,
		ToolTypes:   []task.ToolType{},
	}
}

// YoloModeRunner executes tasks in yolo mode with auto-approval.
type YoloModeRunner struct {
	config     *YoloModeConfig
	approver   *task.AutoApprover
	executor   *task.ToolExecutor
	actionLog  []YoloAction
	mu         sync.RWMutex
	startTime  time.Time
	toolCount  int
	hasErrors  bool
	exitCode   int
}

// YoloAction represents a logged action in yolo mode.
type YoloAction struct {
	// Timestamp is when the action occurred
	Timestamp time.Time `json:"timestamp"`

	// ToolName is the name of the tool executed
	ToolName string `json:"tool_name"`

	// ToolType is the type of tool executed
	ToolType task.ToolType `json:"tool_type"`

	// Parameters contains the tool parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Success indicates if the action succeeded
	Success bool `json:"success"`

	// Output contains the tool output
	Output string `json:"output,omitempty"`

	// Error contains error information if the action failed
	Error string `json:"error,omitempty"`

	// ExitCode is the exit code for command execution
	ExitCode int `json:"exit_code,omitempty"`

	// Duration is how long the action took
	Duration time.Duration `json:"duration"`
}

// YoloResult represents the final result of a yolo mode execution.
type YoloResult struct {
	// Success indicates overall success
	Success bool `json:"success"`

	// ExitCode is the final exit code (0 = success, non-zero = failure)
	ExitCode int `json:"exit_code"`

	// Actions contains all logged actions
	Actions []YoloAction `json:"actions"`

	// TotalActions is the total number of actions executed
	TotalActions int `json:"total_actions"`

	// SuccessfulActions is the count of successful actions
	SuccessfulActions int `json:"successful_actions"`

	// FailedActions is the count of failed actions
	FailedActions int `json:"failed_actions"`

	// Duration is the total execution time
	Duration time.Duration `json:"duration"`

	// StartTime is when execution started
	StartTime time.Time `json:"start_time"`

	// EndTime is when execution ended
	EndTime time.Time `json:"end_time"`
}

// NewYoloModeRunner creates a new yolo mode runner with the given configuration.
func NewYoloModeRunner(config *YoloModeConfig) *YoloModeRunner {
	if config == nil {
		config = DefaultYoloModeConfig()
	}

	// Ensure logger is set
	if config.Logger == nil {
		config.Logger = os.Stdout
	}

	approver := task.NewAutoApprover()

	// Create tool approval config with yolo mode enabled
	toolConfig := &task.ToolApprovalConfig{
		YoloMode:         config.Enabled,
		AutoApproveTools: config.ToolTypes,
		ApprovalTimeout:  0, // No timeout in yolo mode
		MaxRetries:       0, // No retries in yolo mode
		RetryDelay:       0,
	}

	executor := task.NewToolExecutor(toolConfig, approver)

	return &YoloModeRunner{
		config:    config,
		approver:  approver,
		executor:  executor,
		actionLog: make([]YoloAction, 0),
		exitCode:  0,
	}
}

// Run executes a task in yolo mode with the given prompt.
func (r *YoloModeRunner) Run(ctx context.Context, prompt string) (*YoloResult, error) {
	r.startTime = time.Now()
	r.actionLog = make([]YoloAction, 0)
	r.toolCount = 0
	r.hasErrors = false
	r.exitCode = 0

	// Apply timeout if configured
	if r.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	r.log("Yolo mode enabled - auto-approving all tool requests")
	r.log(fmt.Sprintf("Prompt: %s", prompt))

	// TODO: Integrate with actual task execution system
	// For now, return a placeholder result
	result := &YoloResult{
		Success:   true,
		ExitCode:  0,
		Actions:   r.actionLog,
		StartTime: r.startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(r.startTime),
	}

	r.calculateStats(result)

	return result, nil
}

// ExecuteTool executes a single tool in yolo mode.
func (r *YoloModeRunner) ExecuteTool(ctx context.Context, req task.ToolRequest) (*task.ToolResult, error) {
	// Check max tools limit
	if r.config.MaxTools > 0 && r.toolCount >= r.config.MaxTools {
		return nil, fmt.Errorf("maximum tool limit (%d) reached", r.config.MaxTools)
	}

	r.toolCount++

	// Log the action start
	r.logActionStart(req)

	// Execute the tool through the executor
	result, err := r.executor.ExecuteTool(ctx, req)

	// Log the action completion
	r.logActionComplete(req, result, err)

	// Track errors
	if err != nil || !result.Success {
		r.hasErrors = true
		if r.config.ExitOnError {
			r.exitCode = 1
			return result, err
		}
	}

	// Update exit code based on tool result
	if result != nil && result.ExitCode != 0 {
		r.exitCode = result.ExitCode
	}

	return result, err
}

// ExecuteTools executes multiple tools in batch.
func (r *YoloModeRunner) ExecuteTools(ctx context.Context, requests []task.ToolRequest) ([]*task.ToolResult, error) {
	results := make([]*task.ToolResult, 0, len(requests))

	for _, req := range requests {
		result, err := r.ExecuteTool(ctx, req)
		results = append(results, result)

		if err != nil && r.config.ExitOnError {
			return results, err
		}
	}

	return results, nil
}

// GetActionLog returns a copy of the action log.
func (r *YoloModeRunner) GetActionLog() []YoloAction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logCopy := make([]YoloAction, len(r.actionLog))
	copy(logCopy, r.actionLog)
	return logCopy
}

// GetExitCode returns the current exit code.
func (r *YoloModeRunner) GetExitCode() int {
	return r.exitCode
}

// HasErrors returns true if any errors occurred.
func (r *YoloModeRunner) HasErrors() bool {
	return r.hasErrors
}

// log writes a log message.
func (r *YoloModeRunner) log(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	if r.config.JSONOutput {
		// In JSON mode, log as structured data
		jsonLog := map[string]interface{}{
			"timestamp": timestamp,
			"message":   message,
			"type":      "log",
		}
		jsonBytes, _ := json.Marshal(jsonLog)
		logLine = string(jsonBytes) + "\n"
	}

	fmt.Fprint(r.config.Logger, logLine)
}

// logActionStart logs the start of a tool action.
func (r *YoloModeRunner) logActionStart(req task.ToolRequest) {
	message := fmt.Sprintf("Executing tool: %s (%s)", req.ToolName, req.Type)
	r.log(message)
}

// logActionComplete logs the completion of a tool action.
func (r *YoloModeRunner) logActionComplete(req task.ToolRequest, result *task.ToolResult, execErr error) {
	action := YoloAction{
		Timestamp:  time.Now(),
		ToolName:   req.ToolName,
		ToolType:   req.Type,
		Parameters: req.Parameters,
	}

	if result != nil {
		action.Success = result.Success
		action.Output = result.Output
		action.ExitCode = result.ExitCode
		action.Duration = result.Duration

		if !result.Success {
			action.Error = result.Error
		}
	}

	if execErr != nil {
		action.Success = false
		action.Error = execErr.Error()
	}

	// Add to action log
	r.mu.Lock()
	r.actionLog = append(r.actionLog, action)
	r.mu.Unlock()

	// Log result
	status := "SUCCESS"
	if !action.Success {
		status = "FAILED"
	}
	r.log(fmt.Sprintf("Tool %s: %s (duration: %v)", req.ToolName, status, action.Duration))
}

// calculateStats calculates result statistics.
func (r *YoloModeRunner) calculateStats(result *YoloResult) {
	result.TotalActions = len(r.actionLog)

	for _, action := range r.actionLog {
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
		result.ExitCode = r.exitCode
		if result.ExitCode == 0 {
			result.ExitCode = 1 // Default error code
		}
	} else {
		result.ExitCode = 0
	}
}

// OutputResult outputs the final result in the appropriate format.
func (r *YoloModeRunner) OutputResult(result *YoloResult) error {
	if r.config.JSONOutput {
		return r.outputJSONResult(result)
	}
	return r.outputTextResult(result)
}

// outputJSONResult outputs the result as JSON.
func (r *YoloModeRunner) outputJSONResult(result *YoloResult) error {
	encoder := json.NewEncoder(r.config.Logger)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputTextResult outputs the result as human-readable text.
func (r *YoloModeRunner) outputTextResult(result *YoloResult) error {
	var output string

	output += "========================================\n"
	output += "Yolo Mode Execution Complete\n"
	output += "========================================\n\n"

	output += fmt.Sprintf("Total Actions:    %d\n", result.TotalActions)
	output += fmt.Sprintf("Successful:       %d\n", result.SuccessfulActions)
	output += fmt.Sprintf("Failed:           %d\n", result.FailedActions)
	output += fmt.Sprintf("Duration:         %v\n", result.Duration)
	output += fmt.Sprintf("Exit Code:        %d\n", result.ExitCode)
	output += fmt.Sprintf("Success:          %v\n\n", result.Success)

	if len(result.Actions) > 0 {
		output += "Action Details:\n"
		output += "----------------------------------------\n"
		for i, action := range result.Actions {
			status := "✓"
			if !action.Success {
				status = "✗"
			}
			output += fmt.Sprintf("\n%d. %s %s (%s)\n", i+1, status, action.ToolName, action.ToolType)
			output += fmt.Sprintf("   Duration: %v\n", action.Duration)
			if action.Error != "" {
				output += fmt.Sprintf("   Error: %s\n", action.Error)
			}
		}
	}

	output += "\n========================================\n"

	_, err := fmt.Fprint(r.config.Logger, output)
	return err
}

// YoloModeOption configures YoloModeRunner options.
type YoloModeOption func(*YoloModeRunner)

// WithJSONOutput enables JSON output.
func WithJSONOutput(enabled bool) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.JSONOutput = enabled
	}
}

// WithExitOnError enables exit on first error.
func WithExitOnError(enabled bool) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.ExitOnError = enabled
	}
}

// WithMaxTools sets the maximum number of tools to execute.
func WithMaxTools(max int) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.MaxTools = max
	}
}

// WithTimeout sets the execution timeout.
func WithTimeout(timeout time.Duration) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.Timeout = timeout
	}
}

// WithLogger sets the output writer for logging.
func WithLogger(w io.Writer) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.Logger = w
	}
}

// WithToolTypes sets the specific tool types to auto-approve.
func WithToolTypes(types ...task.ToolType) YoloModeOption {
	return func(r *YoloModeRunner) {
		r.config.ToolTypes = types
	}
}

// NewYoloModeRunnerWithOptions creates a new yolo mode runner with options.
func NewYoloModeRunnerWithOptions(options ...YoloModeOption) *YoloModeRunner {
	config := DefaultYoloModeConfig()
	runner := &YoloModeRunner{
		config:    config,
		approver:  task.NewAutoApprover(),
		actionLog: make([]YoloAction, 0),
		exitCode:  0,
	}

	// Create tool approval config
	toolConfig := &task.ToolApprovalConfig{
		YoloMode:         true,
		AutoApproveTools: config.ToolTypes,
		ApprovalTimeout:  0,
		MaxRetries:       0,
		RetryDelay:       0,
	}

	runner.executor = task.NewToolExecutor(toolConfig, runner.approver)

	// Apply options
	for _, opt := range options {
		opt(runner)
	}

	return runner
}

// IsYoloEnabledFromFlags determines if yolo mode is enabled from command-line flags.
func IsYoloEnabledFromFlags(yoloFlag bool, yoloEnv string) bool {
	// Check flag first
	if yoloFlag {
		return true
	}

	// Check environment variable
	if yoloEnv == "true" || yoloEnv == "1" || yoloEnv == "yes" {
		return true
	}

	return false
}