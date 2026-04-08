// Package agent provides the core agent functionality for the Cline CLI.
// It implements the task execution loop, tool execution, and conversation management.
package agent

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/api"
	"github.com/cline/cline/golang-cli/internal/services"
	"github.com/cline/cline/golang-cli/internal/task"
)

// AgentState represents the current state of the agent
type AgentState string

const (
	// StateIdle indicates the agent is waiting for a task
	StateIdle AgentState = "idle"
	// StateRunning indicates the agent is actively processing a task
	StateRunning AgentState = "running"
	// StatePaused indicates the agent is paused waiting for user input
	StatePaused AgentState = "paused"
	// StateCompleted indicates the agent has completed the task
	StateCompleted AgentState = "completed"
	// StateError indicates the agent encountered an error
	StateError AgentState = "error"
	// StateCancelled indicates the agent was cancelled by the user
	StateCancelled AgentState = "cancelled"
)

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	// Provider is the API provider to use
	Provider api.ProviderType

	// Model is the model ID to use
	Model string

	// AutoApprove enables automatic approval of tools and commands
	AutoApprove bool

	// MaxIterations is the maximum number of API calls before stopping
	MaxIterations int

	// Temperature controls randomness (0-1)
	Temperature float64

	// MaxTokens is the maximum tokens per response
	MaxTokens int

	// SystemPrompt is the system prompt to use
	SystemPrompt string

	// WorkingDirectory is the base directory for file operations
	WorkingDirectory string

	// EnableStreaming enables streaming responses
	EnableStreaming bool

	// EnableCheckpoints enables git checkpoint creation
	EnableCheckpoints bool
}

// DefaultConfig returns a default agent configuration
func DefaultConfig() *AgentConfig {
	return &AgentConfig{
		Provider:          api.ProviderAnthropic,
		Model:             "claude-3-5-sonnet-20241022",
		AutoApprove:       false,
		MaxIterations:     100,
		Temperature:       0.7,
		MaxTokens:         8192,
		EnableStreaming:   true,
		EnableCheckpoints: true,
	}
}

// Agent represents the core AI agent that processes tasks
type Agent struct {
	// config holds the agent configuration
	config *AgentConfig

	// state is the current agent state
	state AgentState

	// provider is the API provider instance
	provider api.Provider

	// conversation manages the conversation history
	conversation *task.ConversationManager

	// toolRegistry holds available tools
	toolRegistry *ToolRegistry

	// checkpointManager manages git checkpoints
	checkpointManager *services.CheckpointManager

	// fileService handles file operations
	fileService *services.FileService

	// messageHandler handles outgoing messages
	messageHandler task.MessageHandler

	// iterationCount tracks the number of API calls
	iterationCount int

	// cancelFunc is used to cancel the current operation
	cancelFunc context.CancelFunc

	// mu protects concurrent access to state
	mu sync.RWMutex

	// taskID is the unique identifier for this task
	taskID string

	// ctx is the agent context
	ctx context.Context
}

// NewAgent creates a new agent instance
func NewAgent(config *AgentConfig, taskID string, messageHandler task.MessageHandler) (*Agent, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if taskID == "" {
		taskID = generateTaskID()
	}

	// Create conversation manager
	conversation, err := task.NewConversationManager(taskID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation manager: %w", err)
	}

	// Create checkpoint manager if enabled
	var checkpointManager *services.CheckpointManager
	if config.EnableCheckpoints {
		checkpointManager, err = services.NewCheckpointManager(config.WorkingDirectory)
		if err != nil {
			return nil, fmt.Errorf("failed to create checkpoint manager: %w", err)
		}
	}

	agent := &Agent{
		config:            config,
		state:             StateIdle,
		conversation:      conversation,
		toolRegistry:      NewToolRegistry(),
		checkpointManager: checkpointManager,
		messageHandler:    messageHandler,
		taskID:            taskID,
		iterationCount:    0,
	}

	// Register default tools
	agent.registerDefaultTools()

	return agent, nil
}

// SetProvider sets the API provider for the agent
func (a *Agent) SetProvider(provider api.Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.provider = provider
}

// GetState returns the current agent state
func (a *Agent) GetState() AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// setState updates the agent state
func (a *Agent) setState(state AgentState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = state
}

// ExecuteTask executes a user task
func (a *Agent) ExecuteTask(ctx context.Context, userTask string) error {
	if a.provider == nil {
		return fmt.Errorf("no API provider configured")
	}

	// Set up cancellation context
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel
	defer cancel()

	a.ctx = ctx

	// Update state
	a.setState(StateRunning)

	// Notify task start
	a.say("task", userTask, false)

	// Add user message to conversation
	userMsg := &task.ConversationMessage{
		Type:    "user",
		Content: userTask,
	}
	if err := a.conversation.AddMessage(userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	// Start the main execution loop
	return a.runExecutionLoop(ctx)
}

// runExecutionLoop is the main agent execution loop
func (a *Agent) runExecutionLoop(ctx context.Context) error {
	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			a.setState(StateCancelled)
			return ctx.Err()
		default:
		}

		// Check iteration limit
		if a.iterationCount >= a.config.MaxIterations {
			a.say("error", fmt.Sprintf("Reached maximum iteration limit (%d)", a.config.MaxIterations), false)
			a.setState(StateError)
			return fmt.Errorf("maximum iteration limit reached")
		}

		a.iterationCount++

		// Get conversation history for context
		messages, err := a.getProviderMessages()
		if err != nil {
			return fmt.Errorf("failed to get conversation messages: %w", err)
		}

		// Make API request
		response, err := a.makeAPIRequest(ctx, messages)
		if err != nil {
			a.handleAPIError(err)
			// Ask user if they want to retry
			if a.shouldRetry() {
				continue
			}
			a.setState(StateError)
			return err
		}

		// Process the response
		completed, err := a.processResponse(ctx, response)
		if err != nil {
			a.say("error", fmt.Sprintf("Error processing response: %v", err), false)
			a.setState(StateError)
			return err
		}

		if completed {
			a.setState(StateCompleted)
			return nil
		}
	}
}

// makeAPIRequest makes a request to the API provider
func (a *Agent) makeAPIRequest(ctx context.Context, messages []api.ProviderMessage) (*api.ProviderCompletionResponse, error) {
	// Build system prompt with tool definitions
	systemPrompt := a.buildSystemPrompt()

	req := api.ProviderCompletionRequest{
		Model:       a.config.Model,
		Messages:    messages,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
		Stream:      false, // We'll implement streaming separately
	}

	// Add system prompt as first message if supported by provider
	// For now, prepend to first user message or handle in provider-specific way
	if len(req.Messages) > 0 && req.Messages[0].Role == "user" {
		req.Messages[0].Content = systemPrompt + "\n\n" + req.Messages[0].Content
	}

	a.say("api_req_started", fmt.Sprintf("Iteration %d", a.iterationCount), false)

	// Make the request using the Provider interface
	return a.provider.Complete(ctx, req)
}

// processResponse processes the API response and executes any tools
func (a *Agent) processResponse(ctx context.Context, response *api.ProviderCompletionResponse) (bool, error) {
	if response == nil || response.Content == "" {
		return false, fmt.Errorf("empty response from API")
	}

	// Add assistant message to conversation
	assistantMsg := &task.ConversationMessage{
		Type:    "say",
		SayType: "text",
		Content: response.Content,
	}
	if err := a.conversation.AddMessage(assistantMsg); err != nil {
		return false, err
	}

	// Say the response
	a.say("text", response.Content, false)

	// Check for completion
	if a.isCompletionResponse(response.Content) {
		a.say("completion_result", a.extractCompletionResult(response.Content), false)
		return true, nil
	}

	// Parse and execute tools
	toolCalls := a.parseToolCalls(response.Content)
	if len(toolCalls) > 0 {
		for _, toolCall := range toolCalls {
			result, err := a.executeTool(ctx, toolCall)
			if err != nil {
				// Add error to conversation
				errorMsg := &task.ConversationMessage{
					Type:    "say",
					SayType: "error",
					Content: fmt.Sprintf("Tool error: %v", err),
				}
				a.conversation.AddMessage(errorMsg)
				a.say("error", fmt.Sprintf("Tool execution failed: %v", err), false)
				continue
			}

			// Add tool result to conversation
			toolResultMsg := &task.ConversationMessage{
				Type:       "tool_result",
				ToolName:   toolCall.Name,
				ToolResult: result,
			}
			a.conversation.AddMessage(toolResultMsg)

			// Create checkpoint after successful tool execution
			if a.config.EnableCheckpoints && a.checkpointManager != nil {
				hash, err := a.checkpointManager.CreateCheckpoint(fmt.Sprintf("After tool: %s", toolCall.Name))
				if err != nil {
					a.say("error", fmt.Sprintf("Failed to create checkpoint: %v", err), false)
				} else {
					a.say("checkpoint_created", hash, false)
				}
			}
		}
		return false, nil
	}

	// No tools to execute, check if we should continue
	return false, nil
}

// ToolCall represents a parsed tool invocation
type ToolCall struct {
	Name   string
	Params map[string]interface{}
}

// parseToolCalls parses tool invocations from the response content
func (a *Agent) parseToolCalls(content string) []ToolCall {
	var calls []ToolCall

	// Look for XML-style tool calls
	// Format: <tool_name>\n<parameter>value</parameter>\n</tool_name>
	toolPattern := `<(\w+)>([\s\S]*?)</\1>`
	matches := findAllMatches(content, toolPattern)

	for _, match := range matches {
		if len(match) >= 3 {
			toolName := match[1]
			toolContent := match[2]

			// Parse parameters
			params := make(map[string]interface{})
			paramPattern := `<(\w+)>([\s\S]*?)</\1>`
			paramMatches := findAllMatches(toolContent, paramPattern)

			for _, paramMatch := range paramMatches {
				if len(paramMatch) >= 3 {
					paramName := paramMatch[1]
					paramValue := strings.TrimSpace(paramMatch[2])
					params[paramName] = paramValue
				}
			}

			calls = append(calls, ToolCall{
				Name:   toolName,
				Params: params,
			})
		}
	}

	return calls
}

// findAllMatches finds all regex matches for XML-style tool calls
// This implements proper XML-style tag matching for tool calls
func findAllMatches(content, pattern string) [][]string {
	var matches [][]string

	// Match XML-style tags: <tagname>...</tagname>
	// Go regex doesn't support backreferences, so we need a different approach
	// We use greedy matching and then validate that opening/closing tags match
	tagRegex := regexp.MustCompile(`<(\w+)>([\s\S]*)</(\w+)>`)

	// Find all potential matches
	allMatches := tagRegex.FindAllStringSubmatch(content, -1)

	for _, match := range allMatches {
		if len(match) >= 4 {
			openingTag := match[1]
			closingTag := match[3]
			innerContent := match[2]

			// Verify opening and closing tags match
			if openingTag == closingTag {
				// Valid match - include full match, tag name, and content
				matches = append(matches, []string{
					match[0],     // full match
					openingTag,   // opening tag name
					innerContent, // inner content
					closingTag,   // closing tag name
				})
			}
		}
	}

	return matches
}

// executeTool executes a single tool
func (a *Agent) executeTool(ctx context.Context, toolCall ToolCall) (string, error) {
	tool, err := a.toolRegistry.GetTool(toolCall.Name)
	if err != nil {
		return "", err
	}

	// Ask for approval if not auto-approved
	if !a.config.AutoApprove {
		approvalText := fmt.Sprintf("Execute tool: %s\nParameters: %v", toolCall.Name, toolCall.Params)
		response, err := a.ask("tool", approvalText)
		if err != nil {
			return "", err
		}
		if response != "yesButtonClicked" {
			return "Tool execution cancelled by user", nil
		}
	}

	a.say("tool", fmt.Sprintf("Executing: %s", toolCall.Name), false)

	// Execute the tool
	result, err := tool.Execute(ctx, toolCall.Params)
	if err != nil {
		return "", err
	}

	return result, nil
}

// buildSystemPrompt builds the system prompt with tool definitions
func (a *Agent) buildSystemPrompt() string {
	var sb strings.Builder

	if a.config.SystemPrompt != "" {
		sb.WriteString(a.config.SystemPrompt)
		sb.WriteString("\n\n")
	}

	sb.WriteString("You are Cline, an AI assistant that helps users with software development tasks.\n")
	sb.WriteString("You can use the following tools to accomplish tasks:\n\n")

	// Add tool definitions
	for name, tool := range a.toolRegistry.GetAllTools() {
		sb.WriteString(fmt.Sprintf("## %s\n", name))
		sb.WriteString(tool.GetDescription())
		sb.WriteString("\n\n")
		sb.WriteString("Usage:\n")
		sb.WriteString(tool.GetUsage())
		sb.WriteString("\n\n")
	}

	sb.WriteString("When you have completed the task, use <attempt_completion> to summarize what was done.\n")

	return sb.String()
}

// getProviderMessages converts conversation messages to provider format
func (a *Agent) getProviderMessages() ([]api.ProviderMessage, error) {
	messages, err := a.conversation.GetAllMessages()
	if err != nil {
		return nil, err
	}

	var providerMessages []api.ProviderMessage
	for _, msg := range messages {
		switch msg.Type {
		case "user":
			providerMessages = append(providerMessages, api.ProviderMessage{
				Role:    "user",
				Content: msg.Content,
			})
		case "say", "ask":
			providerMessages = append(providerMessages, api.ProviderMessage{
				Role:    "assistant",
				Content: msg.Content,
			})
		case "tool_result":
			providerMessages = append(providerMessages, api.ProviderMessage{
				Role:    "user",
				Content: fmt.Sprintf("Tool %s result: %s", msg.ToolName, msg.ToolResult),
			})
		}
	}

	return providerMessages, nil
}

// isCompletionResponse checks if the response indicates task completion
func (a *Agent) isCompletionResponse(content string) bool {
	return strings.Contains(content, "<attempt_completion>") ||
		strings.Contains(content, "TASK COMPLETE") ||
		strings.Contains(content, "I have completed")
}

// extractCompletionResult extracts the completion result from the response
func (a *Agent) extractCompletionResult(content string) string {
	// Extract content between attempt_completion tags
	start := strings.Index(content, "<attempt_completion>")
	end := strings.Index(content, "</attempt_completion>")

	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(content[start+len("<attempt_completion>") : end])
	}

	return content
}

// handleAPIError handles API errors
func (a *Agent) handleAPIError(err error) {
	a.say("api_req_failed", err.Error(), false)
}

// shouldRetry asks the user if they want to retry after an error
func (a *Agent) shouldRetry() bool {
	if a.config.AutoApprove {
		return true
	}

	response, err := a.ask("api_req_failed", "API request failed. Retry?")
	if err != nil {
		return false
	}

	return response == "yesButtonClicked"
}

// say sends a SAY message through the message handler
func (a *Agent) say(sayType, text string, partial bool) {
	if a.messageHandler != nil {
		a.messageHandler.OnSay(sayType, text, partial)
	}
}

// ask sends an ASK message through the message handler
func (a *Agent) ask(askType, text string) (string, error) {
	if a.messageHandler != nil {
		return a.messageHandler.OnAsk(askType, text)
	}
	// Default to yes if no handler
	return "yesButtonClicked", nil
}

// Cancel cancels the current operation
func (a *Agent) Cancel() {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	a.setState(StateCancelled)
}

// GetConversation returns the conversation manager
func (a *Agent) GetConversation() *task.ConversationManager {
	return a.conversation
}

// GetTaskID returns the task ID
func (a *Agent) GetTaskID() string {
	return a.taskID
}

// registerDefaultTools registers the default set of tools
func (a *Agent) registerDefaultTools() {
	// Register file tools
	a.toolRegistry.Register(NewReadFileTool(a.config.WorkingDirectory))
	a.toolRegistry.Register(NewWriteFileTool(a.config.WorkingDirectory))
	a.toolRegistry.Register(NewApplyDiffTool(a.config.WorkingDirectory))
	a.toolRegistry.Register(NewSearchFilesTool(a.config.WorkingDirectory))
	a.toolRegistry.Register(NewListFilesTool(a.config.WorkingDirectory))

	// Register command tools
	a.toolRegistry.Register(NewExecuteCommandTool())

	// Register browser tools
	a.toolRegistry.Register(NewBrowserTool())

	// Register MCP tools
	a.toolRegistry.Register(NewUseMcpServerTool())
}

// generateTaskID generates a unique task ID using timestamp and random component
func generateTaskID() string {
	// Use both timestamp and random number to ensure uniqueness
	return fmt.Sprintf("task_%d_%d", time.Now().UnixMilli(), rand.Intn(10000))
}

// Close cleans up the agent resources
func (a *Agent) Close() error {
	if a.conversation != nil {
		if err := a.conversation.Close(); err != nil {
			return err
		}
	}
	return nil
}
