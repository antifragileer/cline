// Package agent provides the core agent functionality for the Cline CLI.
package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/api"
	"github.com/cline/cline/golang-cli/internal/task"
)

// mockMessageHandler is a mock implementation of MessageHandler for testing
type mockMessageHandler struct {
	sayMessages []struct {
		sayType string
		text    string
		partial bool
	}
	askResponses map[string]string
}

func newMockMessageHandler() *mockMessageHandler {
	return &mockMessageHandler{
		sayMessages:  make([]struct{ sayType, text string; partial bool }, 0),
		askResponses: make(map[string]string),
	}
}

func (m *mockMessageHandler) OnSay(sayType string, text string, partial bool) error {
	m.sayMessages = append(m.sayMessages, struct {
		sayType string
		text    string
		partial bool
	}{sayType, text, partial})
	return nil
}

func (m *mockMessageHandler) OnAsk(askType string, text string) (string, error) {
	if response, ok := m.askResponses[askType]; ok {
		return response, nil
	}
	return "yesButtonClicked", nil
}

func (m *mockMessageHandler) OnInfo(text string) error {
	return nil
}

func (m *mockMessageHandler) OnError(err error) error {
	return nil
}

func (m *mockMessageHandler) OnStatus(status string) error {
	return nil
}

func (m *mockMessageHandler) OnProgress(current, total int) error {
	return nil
}

func (m *mockMessageHandler) HandleMessage(msg task.Message) error {
	// Handle different message types
	switch msg.Type {
	case "text":
		m.OnSay("text", msg.Content, msg.IsPartial)
		return nil
	case "error":
		m.OnError(fmt.Errorf("%s", msg.Content))
		return nil
	default:
		return nil
	}
}

func (m *mockMessageHandler) OnBrowserAction(action string, url string) (string, error) {
	return "", nil
}

func (m *mockMessageHandler) OnText(content string, isPartial bool) error {
	m.OnSay("text", content, isPartial)
	return nil
}

func (m *mockMessageHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	return true, nil
}

func (m *mockMessageHandler) OnToolResult(toolName string, result string, success bool) error {
	return nil
}

func (m *mockMessageHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	return "execute", nil
}

func (m *mockMessageHandler) OnCommandOutput(output string, isComplete bool) error {
	return nil
}

func (m *mockMessageHandler) OnCheckpoint(checkpointID string, action string) error {
	return nil
}

func (m *mockMessageHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	return "", nil
}

func (m *mockMessageHandler) OnCompletion(success bool, summary string) error {
	return nil
}

// mockProvider is a mock API provider for testing
type mockProvider struct {
	responses []string
	index     int
}

func newMockProvider(responses []string) *mockProvider {
	return &mockProvider{
		responses: responses,
		index:     0,
	}
}

func (m *mockProvider) Complete(ctx context.Context, req api.ProviderCompletionRequest) (*api.ProviderCompletionResponse, error) {
	if m.index >= len(m.responses) {
		return &api.ProviderCompletionResponse{
			Content: "<attempt_completion>Task completed</attempt_completion>",
		}, nil
	}

	response := m.responses[m.index]
	m.index++

	return &api.ProviderCompletionResponse{
		Content: response,
	}, nil
}

func TestNewAgent(t *testing.T) {
	config := DefaultConfig()
	handler := newMockMessageHandler()

	agent, err := NewAgent(config, "test-task-123", handler)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	if agent.GetTaskID() != "test-task-123" {
		t.Errorf("Expected task ID 'test-task-123', got '%s'", agent.GetTaskID())
	}

	if agent.GetState() != StateIdle {
		t.Errorf("Expected initial state 'idle', got '%s'", agent.GetState())
	}

	if agent.conversation == nil {
		t.Error("Expected conversation manager to be initialized")
	}

	if agent.toolRegistry == nil {
		t.Error("Expected tool registry to be initialized")
	}
}

func TestAgentDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Provider != api.ProviderAnthropic {
		t.Errorf("Expected default provider 'anthropic', got '%s'", config.Provider)
	}

	if config.MaxIterations != 100 {
		t.Errorf("Expected default max iterations 100, got %d", config.MaxIterations)
	}

	if !config.EnableCheckpoints {
		t.Error("Expected checkpoints to be enabled by default")
	}
}

func TestAgentExecuteTask(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{
		Provider:          api.ProviderAnthropic,
		Model:             "test-model",
		AutoApprove:       true,
		MaxIterations:     10,
		Temperature:       0.5,
		MaxTokens:         1000,
		EnableCheckpoints: false,
		WorkingDirectory:  tempDir,
	}

	handler := newMockMessageHandler()
	agent, err := NewAgent(config, "test-task", handler)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Set up mock provider
	mockProv := newMockProvider([]string{
		"I will help you with that task.",
		"<attempt_completion>Task completed successfully</attempt_completion>",
	})
	agent.SetProvider(mockProv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = agent.ExecuteTask(ctx, "Test task")
	if err != nil {
		t.Errorf("ExecuteTask failed: %v", err)
	}

	if agent.GetState() != StateCompleted {
		t.Errorf("Expected state 'completed', got '%s'", agent.GetState())
	}
}

func TestAgentCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{
		Provider:         api.ProviderAnthropic,
		AutoApprove:      true,
		MaxIterations:    100,
		WorkingDirectory: tempDir,
	}

	handler := newMockMessageHandler()
	agent, err := NewAgent(config, "cancel-test", handler)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Set up mock provider that never completes (keeps agent running)
	// Provide many responses so the loop continues until cancelled
	mockProv := newMockProvider([]string{
		"Still working on step 1...",
		"Still working on step 2...",
		"Still working on step 3...",
		"Still working on step 4...",
		"Still working on step 5...",
		"Still working on step 6...",
		"Still working on step 7...",
		"Still working on step 8...",
		"Still working on step 9...",
		"Still working on step 10...",
	})
	agent.SetProvider(mockProv)

	// Cancel immediately - this ensures cancellation happens before completion
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = agent.ExecuteTask(ctx, "Long running task")
	if err != context.Canceled && err != context.DeadlineExceeded {
		// Expected cancellation error
		t.Logf("Expected cancellation error, got: %v", err)
	}

	// The agent should be in a cancelled or error state
	state := agent.GetState()
	if state != StateCancelled && state != StateError {
		t.Errorf("Expected state 'cancelled' or 'error', got '%s'", state)
	}
}

func TestAgentIterationLimit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{
		Provider:         api.ProviderAnthropic,
		AutoApprove:      true,
		MaxIterations:    2,
		WorkingDirectory: tempDir,
	}

	handler := newMockMessageHandler()
	agent, err := NewAgent(config, "limit-test", handler)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Set up mock provider that never completes
	mockProv := newMockProvider([]string{
		"Still working...",
		"Still working...",
		"Still working...",
	})
	agent.SetProvider(mockProv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = agent.ExecuteTask(ctx, "Never ending task")
	if err == nil {
		t.Error("Expected error for iteration limit")
	}

	if agent.GetState() != StateError {
		t.Errorf("Expected state 'error', got '%s'", agent.GetState())
	}
}

func TestAgentToolExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{
		Provider:         api.ProviderAnthropic,
		AutoApprove:      true,
		MaxIterations:    5,
		WorkingDirectory: tempDir,
	}

	handler := newMockMessageHandler()
	agent, err := NewAgent(config, "tool-test", handler)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Test that tools are registered
	tools := agent.toolRegistry.GetAllTools()
	if len(tools) == 0 {
		t.Error("Expected tools to be registered")
	}

	// Test specific tools
	expectedTools := []string{"read_file", "write_to_file", "apply_diff", "search_files", "list_files", "execute_command"}
	for _, toolName := range expectedTools {
		if !agent.toolRegistry.IsToolAvailable(toolName) {
			t.Errorf("Expected tool '%s' to be registered", toolName)
		}
	}
}

func TestParseToolCalls(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{
		WorkingDirectory: tempDir,
	}
	agent, err := NewAgent(config, "parse-test", nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Register a test tool
	agent.toolRegistry.Register(&ReadFileTool{workingDir: tempDir})

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "No tools",
			content:  "This is just a regular response",
			expected: 0,
		},
		{
			name: "Single tool",
			content: `<read_file>
<path>test.txt</path>
</read_file>`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := agent.parseToolCalls(tt.content)
			if len(calls) != tt.expected {
				t.Errorf("Expected %d tool calls, got %d", tt.expected, len(calls))
			}
		})
	}
}

func TestIsCompletionResponse(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "agent-test-*")
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{WorkingDirectory: tempDir}
	agent, _ := NewAgent(config, "test", nil)
	defer agent.Close()

	tests := []struct {
		content  string
		expected bool
	}{
		{"<attempt_completion>Done</attempt_completion>", true},
		{"TASK COMPLETE", true},
		{"I have completed the task", true},
		{"Still working on it", false},
		{"", false},
	}

	for _, tt := range tests {
		result := agent.isCompletionResponse(tt.content)
		if result != tt.expected {
			t.Errorf("isCompletionResponse(%q) = %v, expected %v", tt.content, result, tt.expected)
		}
	}
}

func TestExtractCompletionResult(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "agent-test-*")
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{WorkingDirectory: tempDir}
	agent, _ := NewAgent(config, "test", nil)
	defer agent.Close()

	tests := []struct {
		content  string
		expected string
	}{
		{
			"<attempt_completion>Task completed successfully</attempt_completion>",
			"Task completed successfully",
		},
		{
			"Some text <attempt_completion>Result</attempt_completion> more text",
			"Result",
		},
		{
			"No completion tags",
			"No completion tags",
		},
	}

	for _, tt := range tests {
		result := agent.extractCompletionResult(tt.content)
		if result != tt.expected {
			t.Errorf("extractCompletionResult(%q) = %q, expected %q", tt.content, result, tt.expected)
		}
	}
}

func TestAgentStateTransitions(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "agent-test-*")
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{WorkingDirectory: tempDir}
	agent, _ := NewAgent(config, "state-test", nil)
	defer agent.Close()

	// Test initial state
	if agent.GetState() != StateIdle {
		t.Errorf("Initial state should be idle, got %s", agent.GetState())
	}

	// Test state update
	agent.setState(StateRunning)
	if agent.GetState() != StateRunning {
		t.Errorf("State should be running, got %s", agent.GetState())
	}

	agent.setState(StatePaused)
	if agent.GetState() != StatePaused {
		t.Errorf("State should be paused, got %s", agent.GetState())
	}

	agent.setState(StateCompleted)
	if agent.GetState() != StateCompleted {
		t.Errorf("State should be completed, got %s", agent.GetState())
	}
}

func TestAgentConversationIntegration(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "agent-test-*")
	defer os.RemoveAll(tempDir)

	// Use a unique task ID with timestamp to avoid conflicts with persisted data
	taskID := fmt.Sprintf("conv-test-%d", time.Now().UnixMilli())
	
	config := &AgentConfig{WorkingDirectory: tempDir}
	handler := newMockMessageHandler()
	agent, _ := NewAgent(config, taskID, handler)
	defer agent.Close()

	// Test that conversation manager is properly linked
	if agent.GetConversation() == nil {
		t.Error("Conversation manager should not be nil")
	}

	// Get initial count (should be 0 for fresh conversation)
	initialCount := agent.conversation.GetMessageCount()

	// Add a message through the conversation manager
	msg := &task.ConversationMessage{
		Type:    "user",
		Content: "Test message",
	}
	err := agent.conversation.AddMessage(msg)
	if err != nil {
		t.Errorf("Failed to add message: %v", err)
	}

	// Verify message was added (should be initial + 1)
	count := agent.conversation.GetMessageCount()
	expectedCount := initialCount + 1
	if count != expectedCount {
		t.Errorf("Expected %d message(s), got %d", expectedCount, count)
	}
}

func TestAgentCheckpointIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent-checkpoint-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git in temp directory
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}

	config := &AgentConfig{
		WorkingDirectory:  tempDir,
		EnableCheckpoints: true,
	}

	agent, err := NewAgent(config, "checkpoint-test", nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	if agent.checkpointManager == nil {
		t.Error("Checkpoint manager should be initialized when enabled")
	}
}

func TestGenerateTaskID(t *testing.T) {
	id1 := generateTaskID()
	id2 := generateTaskID()

	if id1 == "" {
		t.Error("Generated task ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated task IDs should be unique")
	}

	if !strings.HasPrefix(id1, "task_") {
		t.Errorf("Task ID should start with 'task_', got %s", id1)
	}
}

func TestAgentClose(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "agent-test-*")
	defer os.RemoveAll(tempDir)

	config := &AgentConfig{WorkingDirectory: tempDir}
	agent, _ := NewAgent(config, "close-test", nil)

	err := agent.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}