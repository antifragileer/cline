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

	// Claude37Sonnet is the Claude 3.7 Sonnet model.
	Claude37Sonnet ClaudeModel = "claude-3-7-sonnet-20250219"
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

// DeltaType represents the type of a content delta.
type DeltaType string

const (
	// DeltaTypeTextDelta represents a text delta.
	DeltaTypeTextDelta DeltaType = "text_delta"

	// DeltaTypeThinkingDelta represents a thinking delta.
	DeltaTypeThinkingDelta DeltaType = "thinking_delta"

	// DeltaTypeSignatureDelta represents a signature delta.
	DeltaTypeSignatureDelta DeltaType = "signature_delta"

	// DeltaTypeInputJSONDelta represents an input_json delta for tool use.
	DeltaTypeInputJSONDelta DeltaType = "input_json_delta"
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

// AnthropicErrorType represents specific Anthropic API error types.
type AnthropicErrorType string

const (
	// ErrorTypeInvalidRequest indicates an invalid request error.
	ErrorTypeInvalidRequest AnthropicErrorType = "invalid_request_error"

	// ErrorTypeAuthentication indicates an authentication error.
	ErrorTypeAuthentication AnthropicErrorType = "authentication_error"

	// ErrorTypePermission indicates a permission error.
	ErrorTypePermission AnthropicErrorType = "permission_error"

	// ErrorTypeNotFound indicates a not found error.
	ErrorTypeNotFound AnthropicErrorType = "not_found_error"

	// ErrorTypeRateLimit indicates a rate limit error.
	ErrorTypeRateLimit AnthropicErrorType = "rate_limit_error"

	// ErrorTypeOverloaded indicates the API is overloaded.
	ErrorTypeOverloaded AnthropicErrorType = "overloaded_error"

	// ErrorTypeAPI indicates a generic API error.
	ErrorTypeAPI AnthropicErrorType = "api_error"
)

// CacheControl represents cache control settings for prompt caching.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral" for prompt caching
}

// AnthropicContentBlock represents a block of content in a message.
type AnthropicContentBlock struct {
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

	// Cache control for prompt caching
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ImageSource represents the source of an image.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicMessage represents a message in the conversation.
type AnthropicMessage struct {
	Role    MessageRole             `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

// AnthropicUsage represents token usage for a request.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	PromptTokens             int `json:"prompt_tokens"`
	CompletionTokens         int `json:"completion_tokens"`
	TotalTokens              int `json:"total_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// AnthropicMessagesRequest represents a request to the Anthropic messages API.
type AnthropicMessagesRequest struct {
	Model         ClaudeModel              `json:"model"`
	MaxTokens     int                      `json:"max_tokens"`
	Messages      []AnthropicMessage       `json:"messages"`
	System        string                   `json:"system,omitempty"`
	Stream        bool                     `json:"stream,omitempty"`
	Temperature   *float64                 `json:"temperature,omitempty"`
	TopP          *float64                 `json:"top_p,omitempty"`
	TopK          *int                     `json:"top_k,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
	Metadata      *RequestMetadata         `json:"metadata,omitempty"`
	Tools         []AnthropicTool          `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice     `json:"tool_choice,omitempty"`
	Thinking      *AnthropicThinkingConfig `json:"thinking,omitempty"`

	// Internal fields for cache control
	SystemCacheControl   *CacheControl `json:"-"`
	MessagesCacheControl *CacheControl `json:"-"`

	// Beta features
	Betas []string `json:"-"`
}

// AnthropicThinkingConfig represents thinking configuration for Claude 3.7+.
type AnthropicThinkingConfig struct {
	Type         string `json:"type"` // "enabled" or "disabled"
	BudgetTokens int    `json:"budget_tokens"`
}

// RequestMetadata contains metadata for the request.
type RequestMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicTool represents a tool that can be used by the model.
type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// AnthropicToolChoice represents how the model should use tools.
type AnthropicToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool", "none"
	Name string `json:"name,omitempty"`
}

// AnthropicMessagesResponse represents a non-streaming response from the Anthropic messages API.
type AnthropicMessagesResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         MessageRole             `json:"role"`
	Model        string                  `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   StopReason              `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicStreamEvent represents an event in a streaming response.
type AnthropicStreamEvent struct {
	Type string `json:"type"`

	// Message start fields
	Message *AnthropicMessagesResponse `json:"message,omitempty"`

	// Content block fields
	Index        int                    `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"`
	Delta        *AnthropicContentDelta `json:"delta,omitempty"`

	// Usage fields
	Usage *AnthropicUsage `json:"usage,omitempty"`

	// Error fields
	Error *AnthropicStreamError `json:"error,omitempty"`
}

// AnthropicContentDelta represents a delta update to content.
type AnthropicContentDelta struct {
	Type        ContentBlockType `json:"type"`
	Text        string           `json:"text,omitempty"`
	Thinking    string           `json:"thinking,omitempty"`
	Signature   string           `json:"signature,omitempty"`
	PartialJSON string           `json:"partial_json,omitempty"`
}

// AnthropicStreamError represents an error in a streaming response.
type AnthropicStreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AnthropicAPIError represents a structured error from the Anthropic API.
type AnthropicAPIError struct {
	Type    AnthropicErrorType `json:"type"`
	Message string             `json:"message"`
	Code    string             `json:"code,omitempty"`
}

// Error implements the error interface.
func (e *AnthropicAPIError) Error() string {
	return fmt.Sprintf("anthropic error (%s): %s", e.Type, e.Message)
}

// IsAnthropicAPIError checks if an error is an Anthropic-specific error.
func IsAnthropicAPIError(err error) (*AnthropicAPIError, bool) {
	var anthropicErr *AnthropicAPIError
	if errors.As(err, &anthropicErr) {
		return anthropicErr, true
	}
	return nil, false
}

// AnthropicProvider implements the Provider interface for Anthropic's API.
type AnthropicProvider struct {
	apiKey       string
	baseURL      string
	httpClient   HTTPClient
	model        ClaudeModel
	tokenTracker *AnthropicTokenTracker
	betas        []string
}

// AnthropicTokenTracker tracks token usage across requests.
type AnthropicTokenTracker struct {
	TotalInputTokens         int
	TotalOutputTokens        int
	TotalRequests            int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// Update updates the tracker with usage from a response.
func (t *AnthropicTokenTracker) Update(usage AnthropicUsage) {
	t.TotalInputTokens += usage.InputTokens
	t.TotalOutputTokens += usage.OutputTokens
	t.CacheCreationInputTokens += usage.CacheCreationInputTokens
	t.CacheReadInputTokens += usage.CacheReadInputTokens
	t.TotalRequests++
}

// Reset resets all counters to zero.
func (t *AnthropicTokenTracker) Reset() {
	t.TotalInputTokens = 0
	t.TotalOutputTokens = 0
	t.TotalRequests = 0
	t.CacheCreationInputTokens = 0
	t.CacheReadInputTokens = 0
}

// TotalTokens returns the total number of tokens used.
func (t *AnthropicTokenTracker) TotalTokens() int {
	return t.TotalInputTokens + t.TotalOutputTokens
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

// WithBetas sets the beta features for the provider.
func WithBetas(betas []string) ProviderOption {
	return func(p *AnthropicProvider) {
		p.betas = betas
	}
}

// NewAnthropicProvider creates a new Anthropic provider with the given options.
func NewAnthropicProvider(opts ...ProviderOption) (*AnthropicProvider, error) {
	provider := &AnthropicProvider{
		baseURL:      DefaultAnthropicBaseURL,
		httpClient:   &http.Client{},
		model:        Claude35Sonnet,
		tokenTracker: &AnthropicTokenTracker{},
		betas:        []string{},
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
func (p *AnthropicProvider) CreateMessage(ctx context.Context, req AnthropicMessagesRequest) (*AnthropicMessagesResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}
	req.Stream = false

	body, err := p.buildRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq, req.Betas)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseErrorResponse(resp.StatusCode, bodyBytes)
	}

	var response AnthropicMessagesResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.tokenTracker.Update(response.Usage)

	return &response, nil
}

// CreateMessageStream sends a streaming message request to the Anthropic API.
func (p *AnthropicProvider) CreateMessageStream(ctx context.Context, req AnthropicMessagesRequest) (<-chan AnthropicStreamEvent, <-chan error) {
	eventChan := make(chan AnthropicStreamEvent)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		if req.Model == "" {
			req.Model = p.model
		}
		req.Stream = true

		body, err := p.buildRequestBody(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to build request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		p.setHeaders(httpReq, req.Betas)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			errChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errChan <- p.parseErrorResponse(resp.StatusCode, bodyBytes)
			return
		}

		if err := p.parseSSEStream(resp.Body, eventChan); err != nil {
			errChan <- err
		}
	}()

	return eventChan, errChan
}

// buildRequestBody builds the JSON request body with proper formatting.
func (p *AnthropicProvider) buildRequestBody(req AnthropicMessagesRequest) ([]byte, error) {
	// Build system with cache control if provided
	var systemBlocks []map[string]interface{}
	if req.System != "" {
		sysBlock := map[string]interface{}{
			"type": "text",
			"text": req.System,
		}
		if req.SystemCacheControl != nil {
			sysBlock["cache_control"] = req.SystemCacheControl
		}
		systemBlocks = append(systemBlocks, sysBlock)
	}

	// Build messages with cache control if provided
	var messages []map[string]interface{}
	for _, msg := range req.Messages {
		msgMap := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if req.MessagesCacheControl != nil {
			// Apply cache control to the last user message
			msgMap["cache_control"] = req.MessagesCacheControl
		}
		messages = append(messages, msgMap)
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     req.Stream,
	}

	if len(systemBlocks) > 0 {
		body["system"] = systemBlocks
	}
	if len(messages) > 0 {
		body["messages"] = messages
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}
	if req.TopK != nil {
		body["top_k"] = *req.TopK
	}
	if len(req.StopSequences) > 0 {
		body["stop_sequences"] = req.StopSequences
	}
	if req.Metadata != nil {
		body["metadata"] = req.Metadata
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = req.ToolChoice
	}
	if req.Thinking != nil {
		body["thinking"] = req.Thinking
	}

	return json.Marshal(body)
}

// setHeaders sets the required headers for Anthropic API requests.
func (p *AnthropicProvider) setHeaders(req *http.Request, betas []string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", p.apiKey)
	req.Header.Set("Anthropic-Version", AnthropicAPIVersion)
	req.Header.Set("Accept", "text/event-stream")

	// Add beta headers if provided
	if len(betas) > 0 {
		req.Header.Set("anthropic-beta", strings.Join(betas, ","))
	}
}

// parseErrorResponse parses an error response from the Anthropic API.
func (p *AnthropicProvider) parseErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error *AnthropicAPIError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil || errorResp.Error == nil {
		// Fallback to generic error if parsing fails
		return fmt.Errorf("API request failed with status %d: %s", statusCode, string(body))
	}

	anthropicErr := errorResp.Error

	// Map to provider errors based on error type
	switch anthropicErr.Type {
	case ErrorTypeInvalidRequest:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, anthropicErr.Message)
	case ErrorTypeAuthentication:
		return fmt.Errorf("%w: %s", ErrInvalidAPIKey, anthropicErr.Message)
	case ErrorTypeRateLimit:
		return fmt.Errorf("%w: %s", ErrRateLimitExceeded, anthropicErr.Message)
	case ErrorTypeOverloaded:
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, anthropicErr.Message)
	case ErrorTypeNotFound:
		return fmt.Errorf("%w: model not found - %s", ErrModelNotFound, anthropicErr.Message)
	default:
		return fmt.Errorf("%w (%s): %s", ErrProviderError, anthropicErr.Type, anthropicErr.Message)
	}
}

// parseSSEStream parses a Server-Sent Events stream and sends events to the channel.
func (p *AnthropicProvider) parseSSEStream(reader io.Reader, eventChan chan<- AnthropicStreamEvent) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				return nil
			}

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				return fmt.Errorf("failed to unmarshal stream event: %w", err)
			}

			// Track token usage from message_stop events
			if event.Type == AnthropicEventTypeMessageStop && event.Usage != nil {
				p.tokenTracker.Update(*event.Usage)
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
func (p *AnthropicProvider) GetTokenTracker() *AnthropicTokenTracker {
	return p.tokenTracker
}

// GetBetas returns the beta features for this provider.
func (p *AnthropicProvider) GetBetas() []string {
	return p.betas
}

// SetBetas sets the beta features for this provider.
func (p *AnthropicProvider) SetBetas(betas []string) {
	p.betas = betas
}

// Complete implements the Provider interface
func (p *AnthropicProvider) Complete(ctx context.Context, req ProviderCompletionRequest) (*ProviderCompletionResponse, error) {
	// Convert ProviderMessage to AnthropicMessage
	messages := make([]AnthropicMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = CreateTextMessage(MessageRole(msg.Role), msg.Content)
	}

	anthropicReq := AnthropicMessagesRequest{
		Model:       ClaudeModel(req.Model),
		Messages:    messages,
		Temperature: &req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}

	resp, err := p.CreateMessage(ctx, anthropicReq)
	if err != nil {
		return nil, err
	}

	content := ExtractTextContent(resp)
	return &ProviderCompletionResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: content,
		Usage: ProviderUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}, nil
}

// IsThinkingModel returns true if the model supports thinking blocks.
func IsThinkingModel(model ClaudeModel) bool {
	switch model {
	case Claude35Sonnet, Claude3Opus, Claude37Sonnet:
		return true
	default:
		return false
	}
}

// SupportsPromptCache returns true if the model supports prompt caching.
func SupportsPromptCache(model ClaudeModel) bool {
	switch model {
	case Claude3Opus, Claude35Sonnet, Claude37Sonnet, Claude35Haiku, Claude3Sonnet, Claude3Haiku:
		return true
	default:
		return false
	}
}

// ExtractTextContent extracts all text content from a response.
func ExtractTextContent(response *AnthropicMessagesResponse) string {
	var texts []string
	for _, block := range response.Content {
		if block.Type == ContentTypeText {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "")
}

// ExtractThinkingContent extracts all thinking content from a response.
func ExtractThinkingContent(response *AnthropicMessagesResponse) string {
	var thoughts []string
	for _, block := range response.Content {
		if block.Type == ContentTypeThinking {
			thoughts = append(thoughts, block.Thinking)
		}
	}
	return strings.Join(thoughts, "")
}

// ExtractRedactedThinkingContent extracts all redacted thinking content from a response.
func ExtractRedactedThinkingContent(response *AnthropicMessagesResponse) string {
	var data []string
	for _, block := range response.Content {
		if block.Type == ContentTypeRedactedThinking {
			data = append(data, block.Data)
		}
	}
	return strings.Join(data, "")
}

// ExtractToolUseContent extracts all tool use blocks from a response.
func ExtractToolUseContent(response *AnthropicMessagesResponse) []AnthropicContentBlock {
	var tools []AnthropicContentBlock
	for _, block := range response.Content {
		if block.Type == ContentTypeToolUse {
			tools = append(tools, block)
		}
	}
	return tools
}

// CreateTextMessage creates a simple text message.
func CreateTextMessage(role MessageRole, text string) AnthropicMessage {
	return AnthropicMessage{
		Role: role,
		Content: []AnthropicContentBlock{
			{
				Type: ContentTypeText,
				Text: text,
			},
		},
	}
}

// CreateThinkingMessage creates a thinking content block.
func CreateThinkingMessage(thinking, signature string) AnthropicContentBlock {
	return AnthropicContentBlock{
		Type:      ContentTypeThinking,
		Thinking:  thinking,
		Signature: signature,
	}
}

// CreateRedactedThinkingMessage creates a redacted thinking content block.
func CreateRedactedThinkingMessage(data string) AnthropicContentBlock {
	return AnthropicContentBlock{
		Type: ContentTypeRedactedThinking,
		Data: data,
	}
}

// CreateToolUseMessage creates a tool use content block.
func CreateToolUseMessage(id, name string, input json.RawMessage) AnthropicContentBlock {
	return AnthropicContentBlock{
		Type:  ContentTypeToolUse,
		ID:    id,
		Name:  name,
		Input: input,
	}
}

// Anthropic event type constants
const (
	AnthropicEventTypeMessageStart      = "message_start"
	AnthropicEventTypeContentBlockStart = "content_block_start"
	AnthropicEventTypeContentBlockDelta = "content_block_delta"
	AnthropicEventTypeContentBlockStop  = "content_block_stop"
	AnthropicEventTypeMessageDelta      = "message_delta"
	AnthropicEventTypeMessageStop       = "message_stop"
	AnthropicEventTypePing              = "ping"
	AnthropicEventTypeError             = "error"
)

// HTTPClient is an interface for HTTP clients.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
