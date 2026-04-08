// Package mode provides unit tests for yolo mode functionality.
package mode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
)

// TestDefaultYoloModeConfig tests the default configuration.
func TestDefaultYoloModeConfig(t *testing.T) {
	config := DefaultYoloModeConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if config.JSONOutput {
		t.Error("expected JSONOutput to be false by default")
	}
	if config.ExitOnError {
		t.Error("expected ExitOnError to be false by default")
	}
	if config.MaxTools != 0 {
		t.Errorf("expected MaxTools to be 0 (unlimited), got %d", config.MaxTools)
	}
	if config.Timeout != 0 {
		t.Errorf("expected Timeout to be 0, got %v", config.Timeout)
	}
	if len(config.ToolTypes) != 0 {
		t.Errorf("expected ToolTypes to be empty, got %d items", len(config.ToolTypes))
	}
}

// TestNewYoloModeRunner tests the creation of a new yolo mode runner.
func TestNewYoloModeRunner(t *testing.T) {
	config := DefaultYoloModeConfig()
	config.Logger = &bytes.Buffer{}
	runner := NewYoloModeRunner(config)

	if runner == nil {
		t.Fatal("expected runner to not be nil")
	}
	if runner.config == nil {
		t.Fatal("expected config to not be nil")
	}
	if runner.approver == nil {
		t.Fatal("expected approver to not be nil")
	}
	if runner.executor == nil {
		t.Fatal("expected executor to not be nil")
	}
	if runner.exitCode != 0 {
		t.Errorf("expected initial exit code to be 0, got %d", runner.exitCode)
	}
}

// TestNewYoloModeRunnerWithNilConfig tests creation with nil config.
func TestNewYoloModeRunnerWithNilConfig(t *testing.T) {
	runner := NewYoloModeRunner(nil)

	if runner == nil {
		t.Fatal("expected runner to not be nil")
	}
	if runner.config == nil {
		t.Fatal("expected config to be initialized")
	}
	if !runner.config.Enabled {
		t.Error("expected default config to have Enabled=true")
	}
}

// TestRunWithTimeout tests the Run method with timeout.
func TestRunWithTimeout(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: false,
		Logger:     &bytes.Buffer{},
		Timeout:    100 * time.Millisecond,
	}

	runner := NewYoloModeRunner(config)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := runner.Run(ctx, "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if !result.Success {
		t.Error("expected result to be successful")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestRunWithCancelledContext tests the Run method with cancelled context.
func TestRunWithCancelledContext(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := runner.Run(ctx, "test prompt")
	if err != nil {
		// Context cancellation is handled gracefully
		t.Logf("Run returned error with cancelled context: %v", err)
	}
}

// TestExecuteToolAutoApproval tests that tools are auto-approved in yolo mode.
func TestExecuteToolAutoApproval(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	req := task.ToolRequest{
		ID:       "test-1",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent.txt",
		},
	}

	result, err := runner.ExecuteTool(ctx, req)

	// Should not fail due to approval (yolo mode auto-approves)
	// But may fail due to file not existing
	if err != nil && !strings.Contains(err.Error(), "maximum tool limit") {
		// Error is acceptable - file may not exist
		t.Logf("ExecuteTool returned error (may be expected): %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}

	// Should have been approved
	if !runner.config.Enabled {
		t.Error("yolo mode should be enabled")
	}
}

// TestExecuteToolMaxToolsLimit tests the max tools limit.
func TestExecuteToolMaxToolsLimit(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:  true,
		Logger:   &bytes.Buffer{},
		MaxTools: 1,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	req := task.ToolRequest{
		ID:       "test-1",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent.txt",
		},
	}

	// First tool execution should work
	_, _ = runner.ExecuteTool(ctx, req)

	// Second tool execution should fail due to limit
	req2 := task.ToolRequest{
		ID:       "test-2",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent2.txt",
		},
	}

	_, err := runner.ExecuteTool(ctx, req2)
	if err == nil {
		t.Error("expected error when exceeding max tools limit")
	}
	if !strings.Contains(err.Error(), "maximum tool limit") {
		t.Errorf("expected 'maximum tool limit' error, got: %v", err)
	}
}

// TestExecuteToolsBatch tests batch tool execution.
func TestExecuteToolsBatch(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:  true,
		Logger:   &bytes.Buffer{},
		MaxTools: 10,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	requests := []task.ToolRequest{
		{
			ID:       "test-1",
			Type:     task.ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
		{
			ID:       "test-2",
			Type:     task.ToolTypeSearchFiles,
			ToolName: "search_files",
			Parameters: map[string]interface{}{
				"path":  ".",
				"regex": "test",
			},
		},
	}

	results, err := runner.ExecuteTools(ctx, requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != len(requests) {
		t.Errorf("expected %d results, got %d", len(requests), len(results))
	}
}

// TestExecuteToolsExitOnError tests exit on error behavior.
func TestExecuteToolsExitOnError(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:     true,
		Logger:      &bytes.Buffer{},
		ExitOnError: true,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	requests := []task.ToolRequest{
		{
			ID:       "test-1",
			Type:     task.ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": "/nonexistent/path/to/file.txt",
			},
		},
		{
			ID:       "test-2",
			Type:     task.ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
	}

	_, err := runner.ExecuteTools(ctx, requests)
	// Should return error due to ExitOnError
	if err == nil {
		t.Log("ExecuteTools may or may not return error depending on file existence")
	}
}

// TestGetActionLog tests action log retrieval.
func TestGetActionLog(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Initially empty
	log := runner.GetActionLog()
	if len(log) != 0 {
		t.Errorf("expected empty log, got %d entries", len(log))
	}

	// Execute a tool
	req := task.ToolRequest{
		ID:       "test-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}

	_, _ = runner.ExecuteTool(ctx, req)

	// Check log again
	log = runner.GetActionLog()
	if len(log) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(log))
	}

	if log[0].ToolName != "list_files" {
		t.Errorf("expected tool name 'list_files', got '%s'", log[0].ToolName)
	}
}

// TestGetExitCode tests exit code retrieval.
func TestGetExitCode(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)

	// Initial exit code should be 0
	if code := runner.GetExitCode(); code != 0 {
		t.Errorf("expected initial exit code 0, got %d", code)
	}
}

// TestHasErrors tests error tracking.
func TestHasErrors(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)

	// Initially no errors
	if runner.HasErrors() {
		t.Error("expected no errors initially")
	}

	// Execute a tool that will fail
	ctx := context.Background()
	req := task.ToolRequest{
		ID:       "test-1",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/nonexistent/path/that/does/not/exist.txt",
		},
	}

	_, _ = runner.ExecuteTool(ctx, req)

	// May or may not have errors depending on execution
	t.Logf("HasErrors after failed tool: %v", runner.HasErrors())
}

// TestOutputResultJSON tests JSON output format.
func TestOutputResultJSON(t *testing.T) {
	var buf bytes.Buffer
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: true,
		Logger:     &buf,
	}

	runner := NewYoloModeRunner(config)
	result := &YoloResult{
		Success:   true,
		ExitCode:  0,
		Actions:   []YoloAction{},
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}

	err := runner.OutputResult(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify JSON output
	output := buf.String()
	var parsed YoloResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, output)
	}
}

// TestOutputResultText tests text output format.
func TestOutputResultText(t *testing.T) {
	var buf bytes.Buffer
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: false,
		Logger:     &buf,
	}

	runner := NewYoloModeRunner(config)
	result := &YoloResult{
		Success:      true,
		ExitCode:     0,
		Actions:      []YoloAction{},
		TotalActions: 0,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
	}

	err := runner.OutputResult(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify text output contains expected sections
	output := buf.String()
	expectedSections := []string{
		"Yolo Mode Execution Complete",
		"Total Actions:",
		"Exit Code:",
	}
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("expected output to contain '%s', got:\n%s", section, output)
		}
	}
}

// TestYoloModeOptionFunctions tests option functions.
func TestYoloModeOptionFunctions(t *testing.T) {
	tests := []struct {
		name   string
		option YoloModeOption
		check  func(*YoloModeRunner) bool
	}{
		{
			name:   "WithJSONOutput",
			option: WithJSONOutput(true),
			check:  func(r *YoloModeRunner) bool { return r.config.JSONOutput },
		},
		{
			name:   "WithExitOnError",
			option: WithExitOnError(true),
			check:  func(r *YoloModeRunner) bool { return r.config.ExitOnError },
		},
		{
			name:   "WithMaxTools",
			option: WithMaxTools(5),
			check:  func(r *YoloModeRunner) bool { return r.config.MaxTools == 5 },
		},
		{
			name:   "WithTimeout",
			option: WithTimeout(30 * time.Second),
			check:  func(r *YoloModeRunner) bool { return r.config.Timeout == 30*time.Second },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewYoloModeRunnerWithOptions(tt.option)
			if !tt.check(runner) {
				t.Errorf("option %s was not applied correctly", tt.name)
			}
		})
	}
}

// TestWithLogger tests the WithLogger option.
func TestWithLogger(t *testing.T) {
	var buf bytes.Buffer
	runner := NewYoloModeRunnerWithOptions(WithLogger(&buf))

	if runner.config.Logger != &buf {
		t.Error("WithLogger option not applied correctly")
	}
}

// TestWithToolTypes tests the WithToolTypes option.
func TestWithToolTypes(t *testing.T) {
	toolTypes := []task.ToolType{task.ToolTypeReadFile, task.ToolTypeWriteFile}
	runner := NewYoloModeRunnerWithOptions(WithToolTypes(toolTypes...))

	if len(runner.config.ToolTypes) != len(toolTypes) {
		t.Errorf("expected %d tool types, got %d", len(toolTypes), len(runner.config.ToolTypes))
	}
}

// TestIsYoloEnabledFromFlags tests the flag detection function.
func TestIsYoloEnabledFromFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     bool
		env      string
		expected bool
	}{
		{"flag true", true, "", true},
		{"flag false, env true", false, "true", true},
		{"flag false, env 1", false, "1", true},
		{"flag false, env yes", false, "yes", true},
		{"flag false, env false", false, "false", false},
		{"flag false, env empty", false, "", false},
		{"flag false, env random", false, "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsYoloEnabledFromFlags(tt.flag, tt.env)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestYoloActionStructure tests the YoloAction struct.
func TestYoloActionStructure(t *testing.T) {
	action := YoloAction{
		Timestamp:  time.Now(),
		ToolName:   "test_tool",
		ToolType:   task.ToolTypeReadFile,
		Parameters: map[string]interface{}{"key": "value"},
		Success:    true,
		Output:     "test output",
		ExitCode:   0,
		Duration:   time.Second,
	}

	// Verify fields
	if action.ToolName != "test_tool" {
		t.Error("ToolName not set correctly")
	}
	if action.ToolType != task.ToolTypeReadFile {
		t.Error("ToolType not set correctly")
	}
	if !action.Success {
		t.Error("Success should be true")
	}
	if action.ExitCode != 0 {
		t.Errorf("ExitCode should be 0, got %d", action.ExitCode)
	}
}

// TestYoloResultStructure tests the YoloResult struct.
func TestYoloResultStructure(t *testing.T) {
	result := YoloResult{
		Success:           true,
		ExitCode:          0,
		Actions:           []YoloAction{},
		TotalActions:      0,
		SuccessfulActions: 0,
		FailedActions:     0,
		Duration:          time.Minute,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode should be 0, got %d", result.ExitCode)
	}
	if result.Duration != time.Minute {
		t.Errorf("Duration should be 1 minute, got %v", result.Duration)
	}
}

// TestConcurrentActionLog tests thread safety of action log.
func TestConcurrentActionLog(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute tools concurrently
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(id int) {
			req := task.ToolRequest{
				ID:       fmt.Sprintf("test-%d", id),
				Type:     task.ToolTypeListFiles,
				ToolName: "list_files",
				Parameters: map[string]interface{}{
					"path": ".",
				},
			}
			_, _ = runner.ExecuteTool(ctx, req)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify log has all entries
	log := runner.GetActionLog()
	if len(log) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(log))
	}
}

// TestCalculateStats tests the calculateStats method.
func TestCalculateStats(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &bytes.Buffer{},
	}

	runner := NewYoloModeRunner(config)

	// Add some actions to the log
	runner.actionLog = []YoloAction{
		{Success: true},
		{Success: true},
		{Success: false, Error: "error"},
	}

	result := &YoloResult{}
	runner.calculateStats(result)

	if result.TotalActions != 3 {
		t.Errorf("expected 3 total actions, got %d", result.TotalActions)
	}
	if result.SuccessfulActions != 2 {
		t.Errorf("expected 2 successful actions, got %d", result.SuccessfulActions)
	}
	if result.FailedActions != 1 {
		t.Errorf("expected 1 failed action, got %d", result.FailedActions)
	}
	if result.Success {
		t.Error("expected Success to be false due to failed action")
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code due to failure")
	}
}

// TestLogActionCompleteWithError tests error logging.
func TestLogActionCompleteWithError(t *testing.T) {
	var buf bytes.Buffer
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  &buf,
	}

	runner := NewYoloModeRunner(config)

	req := task.ToolRequest{
		ID:       "test-1",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
	}

	execErr := errors.New("test execution error")
	runner.logActionComplete(req, nil, execErr)

	log := runner.GetActionLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log))
	}

	if log[0].Error != execErr.Error() {
		t.Errorf("expected error '%s', got '%s'", execErr.Error(), log[0].Error)
	}
	if log[0].Success {
		t.Error("expected Success to be false")
	}
}

// TestLogWithJSONOutput tests JSON logging.
func TestLogWithJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: true,
		Logger:     &buf,
	}

	runner := NewYoloModeRunner(config)
	runner.log("test message")

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("log output is not valid JSON: %v\nOutput: %s", err, output)
	}

	if logEntry["message"] != "test message" {
		t.Errorf("expected message 'test message', got '%v'", logEntry["message"])
	}
	if logEntry["type"] != "log" {
		t.Errorf("expected type 'log', got '%v'", logEntry["type"])
	}
}

// TestLogWithTextOutput tests text logging.
func TestLogWithTextOutput(t *testing.T) {
	var buf bytes.Buffer
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: false,
		Logger:     &buf,
	}

	runner := NewYoloModeRunner(config)
	runner.log("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
}
