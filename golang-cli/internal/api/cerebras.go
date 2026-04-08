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

// DefaultCerebrasBaseURL is the default base URL for the Cerebras API.
const DefaultCerebrasBaseURL = "https://api.cerebras.ai/v1"

// CerebrasModel represents the Cerebras model identifiers.
type CerebrasModel string

const (
	// CerebrasModelZaiGLM47 is the Z.ai GLM 4.7 model.
	CerebrasModelZaiGLM47 CerebrasModel = "zai-glm-4.7"
	// CerebrasModelGPTOSS120B is the GPT-OSS 120B model.
	CerebrasModelGPTOSS120B CerebrasModel = "gpt-oss-120b"
	// CerebrasModelQwen3235BA22B is the Qwen3 235B A22B model.
	CerebrasModelQwen3235BA22B CerebrasModel = "qwen-3-235b-a22b-instruct-2507"
)

// CerebrasModels contains information about available Cerebras models.
var CerebrasModels = map[CerebrasModel]ModelInfo{
	CerebrasModelZaiGLM47: {
		MaxTokens:           40000,
		ContextWindow:       131072,
		SupportsImages:      false,
		SupportsPromptCache: false,
		Temperature:         0.9,
		InputPrice:          0,
		OutputPrice:         0,
		Description:         "Highly capable general-purpose model on Cerebras (up to 1,000 tokens/s), competitive with leading proprietary models on coding tasks.",
	},
	CerebrasModelGPTOSS120B: {
		MaxTokens:           65536,
		ContextWindow:       128000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0,
		OutputPrice:         0,
		Description:         "Intelligent general purpose model with 3,000 tokens/s",
	},
	CerebrasModelQwen3235BA22B: {
		MaxTokens:           64000,
		ContextWindow:       64000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0,
		OutputPrice:         0,
		Description:         "Intelligent model with ~1400 tokens/s",
	},
}

// CerebrasConfig represents the configuration for the Cerebras provider.
type CerebrasConfig struct {
	APIKey string
	Model  CerebrasModel
}

// CerebrasProvider implements the Provider interface for Cerebras' API.
type CerebrasProvider struct {
	apiKey       string
	baseURL      string
	httpClient   HTTPClient
	model        CerebrasModel
	tokenTracker *CerebrasTokenTracker
}

// CerebrasTokenTracker tracks token usage across requests.
type CerebrasTokenTracker struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalRequests     int
}

// Update updates the tracker with usage from a response.
func (t *CerebrasTokenTracker) Update(usage Usage) {
	t.TotalInputTokens += usage.PromptTokens
	t.TotalOutputTokens += usage.CompletionTokens
	t.TotalRequests++
}

// Reset resets all counters to zero.
func (t *CerebrasTokenTracker) Reset() {
	t.TotalInputTokens = 0
	t.TotalOutputTokens = 0
	t.TotalRequests = 0
}

// TotalTokens returns the total number of tokens used.
func (t *CerebrasTokenTracker) TotalTokens() int {
	return t.TotalInputTokens + t.TotalOutputTokens
}

// CerebrasMessage represents a message in the conversation.
type CerebrasMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CerebrasChatCompletionRequest represents a request to the Cerebras chat completions API.
type CerebrasChatCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []CerebrasMessage `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

// CerebrasChatCompletionResponse represents a response from the Cerebras chat completions API.
type CerebrasChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// CerebrasStreamChunk represents a chunk from a streaming response.
type CerebrasStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// CerebrasAPIError represents an error from the Cerebras API.
type CerebrasAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Error implements the error interface.
func (e *CerebrasAPIError) Error() string {
	return fmt.Sprintf("cerebras error (%s): %s", e.Code, e.Message)
}

// NewCerebrasProvider creates a new Cerebras provider with the given configuration.
func NewCerebrasProvider(config CerebrasConfig) (*CerebrasProvider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	model := config.Model
	if model == "" {
		model = CerebrasModelZaiGLM47
	}

	return &CerebrasProvider{
		apiKey:       config.APIKey,
		baseURL:      DefaultCerebrasBaseURL,
		httpClient:   &http.Client{},
		model:        model,
		tokenTracker: &CerebrasTokenTracker{},
	}, nil
}

// CreateChatCompletion sends a non-streaming chat completion request to the Cerebras API.
func (p *CerebrasProvider) CreateChatCompletion(ctx context.Context, req CerebrasChatCompletionRequest) (*CerebrasChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = string(p.model)
	}
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

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

	var response CerebrasChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.tokenTracker.Update(response.Usage)

	return &response, nil
}

// CreateChatCompletionStream sends a streaming chat completion request to the Cerebras API.
func (p *CerebrasProvider) CreateChatCompletionStream(ctx context.Context, req CerebrasChatCompletionRequest) (<-chan CerebrasStreamChunk, <-chan error) {
	eventChan := make(chan CerebrasStreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		if req.Model == "" {
			req.Model = string(p.model)
		}
		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
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
			errChan <- p.parseErrorResponse(resp.StatusCode, bodyBytes)
			return
		}

		if err := p.parseSSEStream(resp.Body, eventChan); err != nil {
			errChan <- err
		}
	}()

	return eventChan, errChan
}

// setHeaders sets the required headers for Cerebras API requests.
func (p *CerebrasProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")
}

// parseErrorResponse parses an error response from the Cerebras API.
func (p *CerebrasProvider) parseErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error *CerebrasAPIError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil || errorResp.Error == nil {
		// Fallback to generic error if parsing fails
		return fmt.Errorf("API request failed with status %d: %s", statusCode, string(body))
	}

	cerebrasErr := errorResp.Error

	// Map to provider errors based on error code
	switch cerebrasErr.Code {
	case "invalid_api_key":
		return fmt.Errorf("%w: %s", ErrInvalidAPIKey, cerebrasErr.Message)
	case "rate_limit_exceeded":
		return fmt.Errorf("%w: %s", ErrRateLimitExceeded, cerebrasErr.Message)
	case "invalid_request":
		return fmt.Errorf("%w: %s", ErrInvalidRequest, cerebrasErr.Message)
	default:
		return fmt.Errorf("%w (%s): %s", ErrProviderError, cerebrasErr.Code, cerebrasErr.Message)
	}
}

// parseSSEStream parses a Server-Sent Events stream and sends chunks to the channel.
func (p *CerebrasProvider) parseSSEStream(reader io.Reader, chunkChan chan<- CerebrasStreamChunk) error {
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

			var chunk CerebrasStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return fmt.Errorf("failed to unmarshal stream chunk: %w", err)
			}

			// Track token usage from final chunk
			if chunk.Usage != nil {
				p.tokenTracker.Update(*chunk.Usage)
			}

			chunkChan <- chunk
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// GetModel returns the model being used.
func (p *CerebrasProvider) GetModel() CerebrasModel {
	return p.model
}

// SetModel sets the model to use.
func (p *CerebrasProvider) SetModel(model CerebrasModel) {
	p.model = model
}

// GetTokenTracker returns the token tracker for this provider.
func (p *CerebrasProvider) GetTokenTracker() *CerebrasTokenTracker {
	return p.tokenTracker
}

// Complete implements the Provider interface.
func (p *CerebrasProvider) Complete(ctx context.Context, req ProviderCompletionRequest) (*ProviderCompletionResponse, error) {
	// Convert ProviderMessage to CerebrasMessage
	messages := make([]CerebrasMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = CerebrasMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	cerebrasReq := CerebrasChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      false,
	}

	resp, err := p.CreateChatCompletion(ctx, cerebrasReq)
	if err != nil {
		return nil, err
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return &ProviderCompletionResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: content,
		Usage: ProviderUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}
