// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AnthropicAPIVersion is the version of the Anthropic API to use.
const AnthropicAPIVersion = "2023-06-01"

// DefaultAnthropicBaseURL is the default base URL for the Anthropic API.
const DefaultAnthropicBaseURL = "https://api.anthropic.com"

// ClaudeModel represents the Claude model identifiers.
type ClaudeModel string

const (
	// Claude3Opus is the Claude 3 Opus model.
	Claude3Opus ClaudeModel = "claude-3-opus-20240229"

	// Claude3Sonnet is the Claude 3 Sonnet model.
	Claude3Sonnet ClaudeModel = "claude-3-sonnet-20240229"

	// Claude3Haiku is the Claude 3 Haiku model.
	Claude3Haiku ClaudeModel = "claude-3-haiku-20240307"

	// Claude35Sonnet is the Claude 3.5 Sonnet model.
	Claude35Sonnet ClaudeModel = "claude-3-5-sonnet-20241022"

	// Claude35Haiku is the Claude 3.5 Haiku model.
	Claude35Haiku ClaudeModel = "claude-3-5-haiku-20241022"
)

// MessageRole represents the role of a message.
type MessageRole string

const (
	// RoleUser represents a user message.
	RoleUser MessageRole = "user"

	// RoleAssistant represents an assistant message.
	RoleAssistant MessageRole = "assistant"
)

// ContentBlockType represents the type of a content block.
type ContentBlockType string

const (
	// ContentTypeText represents a text content block.
	ContentTypeText ContentBlockType = "text"

	// ContentTypeThinking represents a thinking content block.
	ContentTypeThinking ContentBlockType = "thinking"

	// ContentTypeRedactedThinking represents a redacted thinking block.
	ContentTypeRedactedThinking ContentBlockType = "redacted_thinking"

	// ContentTypeToolUse represents a tool use block.
	ContentTypeToolUse ContentBlockType = "tool_use"

	// ContentTypeToolResult represents a tool result block.
	ContentTypeToolResult ContentBlockType = "tool_result"

	// ContentTypeImage represents an image block.
	ContentTypeImage ContentBlockType = "image"
)

// StopReason represents the reason why the model stopped generating.
type StopReason string

const (
	// StopReasonEndTurn indicates the model reached a natural stopping point.
	StopReasonEndTurn StopReason = "end_turn"

	// StopReasonMaxTokens indicates the model reached the maximum token limit.
	StopReasonMaxTokens StopReason = "max_tokens"

	// StopReasonStopSequence indicates the model encountered a stop sequence.
	StopReasonStopSequence StopReason = "stop_sequence"

	// StopReasonToolUse indicates the model wants to use a tool.
	StopReasonToolUse StopReason = "tool_use"
)

// ContentBlock represents a block of content in a message.
type ContentBlock struct {
	Type ContentBlockType `json:"type"`

	// Text fields (for text blocks)
	Text string `json:"text,omitempty"`

	// Thinking fields (for thinking blocks)
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// Redacted thinking fields
	Data string `json:"data,omitempty"`

	// Tool use fields
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// Tool result fields
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`

	// Image fields
	Source *ImageSource `json:"source,omitempty"`
}

// ImageSource represents the source of an image.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Message represents a message in the conversation.
type Message struct {
	Role    MessageRole    `json:"role"`
	Content []ContentBlock `json:"content"`
}

// Usage represents token usage for a request.
type Usage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// MessagesRequest represents a request to the Anthropic messages API.
type MessagesRequest struct {
	Model         ClaudeModel      `json:"model"`
	MaxTokens     int              `json:"max_tokens"`
	Messages      []Message        `json:"messages"`
	System        string           `json:"system,omitempty"`
	Stream        bool             `json:"stream,omitempty"`
	Temperature   *float64         `json:"temperature,omitempty"`
	TopP          *float64         `json:"top_p,omitempty"`
	TopK          *int             `json:"top_k,omitempty"`
	StopSequences []string         `json:"stop_sequences,omitempty"`
	Metadata      *RequestMetadata `json:"metadata,omitempty"`
	Tools         []Tool           `json:"tools,omitempty"`
	ToolChoice    *ToolChoice      `json:"tool_choice,omitempty"`
}

// RequestMetadata contains metadata for the request.
type RequestMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// Tool represents a tool that can be used by the model.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolChoice represents how the model should use tools.
type ToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"`
}

// MessagesResponse represents a non-streaming response from the Anthropic messages API.
type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         MessageRole    `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   StopReason     `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// StreamEvent represents an event in a streaming response.
type StreamEvent struct {
	Type string `json:"type"`

	// Message start fields
	Message *MessagesResponse `json:"message,omitempty"`

	// Content block fields
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *ContentDelta `json:"delta,omitempty"`

	// Usage fields
	Usage *Usage `json:"usage,omitempty"`

	// Error fields
	Error *StreamError `json:"error,omitempty"`
}

// ContentDelta represents a delta update to content.
type ContentDelta struct {
	Type     ContentBlockType `json:"type"`
	Text     string           `json:"text,omitempty"`
	Thinking string           `json:"thinking,omitempty"`
	Partial  bool             `json:"partial,omitempty"`
}

// StreamError represents an error in a streaming response.
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// HTTPClient is an interface for HTTP clients.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Provider defines the interface for API providers.
type Provider interface {
	// CreateMessage sends a non-streaming message request.
	CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error)

	// CreateMessageStream sends a streaming message request.
	CreateMessageStream(ctx context.Context, req MessagesRequest) (<-chan StreamEvent, <-chan error)

	// GetModel returns the model being used.
	GetModel() ClaudeModel

	// SetModel sets the model to use.
	SetModel(model ClaudeModel)
}

// TokenTracker tracks token usage across requests.
type TokenTracker struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalRequests     int
}

// Update updates the tracker with usage from a response.
func (t *TokenTracker) Update(usage Usage) {
	t.TotalInputTokens += usage.InputTokens
	t.TotalOutputTokens += usage.OutputTokens
	t.TotalRequests++
}

// Reset resets all counters to zero.
func (t *TokenTracker) Reset() {
	t.TotalInputTokens = 0
	t.TotalOutputTokens = 0
	t.TotalRequests = 0
}

// TotalTokens returns the total number of tokens used.
func (t *TokenTracker) TotalTokens() int {
	return t.TotalInputTokens + t.TotalOutputTokens
}

// AnthropicProvider implements the Provider interface for Anthropic's API.
type AnthropicProvider struct {
	apiKey       string
	baseURL      string
	httpClient   HTTPClient
	model        ClaudeModel
	tokenTracker *TokenTracker
}

// ProviderOption is a functional option for configuring AnthropicProvider.
type ProviderOption func(*AnthropicProvider)

// WithAPIKey sets the API key for the provider.
func WithAPIKey(apiKey string) ProviderOption {
	return func(p *AnthropicProvider) {
		p.apiKey = apiKey
	}
}

// WithBaseURL sets the base URL for the provider.
func WithBaseURL(baseURL string) ProviderOption {
	return func(p *AnthropicProvider) {
		p.baseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client for the provider.
func WithHTTPClient(client HTTPClient) ProviderOption {
	return func(p *AnthropicProvider) {
		p.httpClient = client
	}
}

// WithModel sets the model for the provider.
func WithModel(model ClaudeModel) ProviderOption {
	return func(p *AnthropicProvider) {
		p.model = model
	}
}

// NewAnthropicProvider creates a new Anthropic provider with the given options.
func NewAnthropicProvider(opts ...ProviderOption) (*AnthropicProvider, error) {
	provider := &AnthropicProvider{
		baseURL:      DefaultAnthropicBaseURL,
		httpClient:   &http.Client{},
		model:        Claude35Sonnet,
		tokenTracker: &TokenTracker{},
	}

	for _, opt := range opts {
		opt(provider)
	}

	if provider.apiKey == "" {
		return nil, errors.New("API key is required")
	}

	return provider, nil
}

// CreateMessage sends a non-streaming message request to the Anthropic API.
func (p *AnthropicProvider) CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.tokenTracker.Update(response.Usage)

	return &response, nil
}

// CreateMessageStream sends a streaming message request to the Anthropic API.
func (p *AnthropicProvider) CreateMessageStream(ctx context.Context, req MessagesRequest) (<-chan StreamEvent, <-chan error) {
	eventChan := make(chan StreamEvent)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		if req.Model == "" {
			req.Model = p.model
		}
		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		p.setHeaders(httpReq)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			errChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		if err := p.parseSSEStream(resp.Body, eventChan); err != nil {
			errChan <- err
		}
	}()

	return eventChan, errChan
}

// setHeaders sets the required headers for Anthropic API requests.
func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", p.apiKey)
	req.Header.Set("Anthropic-Version", AnthropicAPIVersion)
	req.Header.Set("Accept", "text/event-stream")
}

// parseSSEStream parses a Server-Sent Events stream and sends events to the channel.
func (p *AnthropicProvider) parseSSEStream(reader io.Reader, eventChan chan<- StreamEvent) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "event: ") {
			// Event type line - we'll read the data on the next iteration
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				return nil
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				return fmt.Errorf("failed to unmarshal stream event: %w", err)
			}

			// Track token usage from message_stop events
			if event.Type == "message_stop" && event.Message != nil {
				p.tokenTracker.Update(event.Message.Usage)
			}

			eventChan <- event
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// GetModel returns the model being used.
func (p *AnthropicProvider) GetModel() ClaudeModel {
	return p.model
}

// SetModel sets the model to use.
func (p *AnthropicProvider) SetModel(model ClaudeModel) {
	p.model = model
}

// GetTokenTracker returns the token tracker for this provider.
func (p *AnthropicProvider) GetTokenTracker() *TokenTracker {
	return p.tokenTracker
}

// IsThinkingModel returns true if the model supports thinking blocks.
func IsThinkingModel(model ClaudeModel) bool {
	switch model {
	case Claude35Sonnet, Claude3Opus:
		return true
	default:
		return false
	}
}

// ExtractTextContent extracts all text content from a response.
func ExtractTextContent(response *MessagesResponse) string {
	var texts []string
	for _, block := range response.Content {
		if block.Type == ContentTypeText {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "")
}

// ExtractThinkingContent extracts all thinking content from a response.
func ExtractThinkingContent(response *MessagesResponse) string {
	var thoughts []string
	for _, block := range response.Content {
		if block.Type == ContentTypeThinking {
			thoughts = append(thoughts, block.Thinking)
		}
	}
	return strings.Join(thoughts, "")
}

// CreateTextMessage creates a simple text message.
func CreateTextMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Content: []ContentBlock{
			{
				Type: ContentTypeText,
				Text: text,
			},
		},
	}
}

// StreamEventType constants for stream event types.
const (
	EventTypeMessageStart      = "message_start"
	EventTypeContentBlockStart = "content_block_start"
	EventTypeContentBlockDelta = "content_block_delta"
	EventTypeContentBlockStop  = "content_block_stop"
	EventTypeMessageDelta      = "message_delta"
	EventTypeMessageStop       = "message_stop"
	EventTypePing              = "ping"
	EventTypeError             = "error"
)