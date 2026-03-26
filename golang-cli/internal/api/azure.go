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

// DefaultAzureOpenAiDefaultApiVersion is the default API version for Azure OpenAI.
const DefaultAzureOpenAiDefaultApiVersion = "2024-08-01-preview"

// AzureOpenAIConfig represents the configuration for the Azure OpenAI provider.
type AzureOpenAIConfig struct {
	APIKey       string
	Endpoint     string
	DeploymentID string
	APIVersion   string
}

// AzureOpenAIProvider implements the Provider interface for Azure OpenAI's API.
type AzureOpenAIProvider struct {
	apiKey       string
	endpoint     string
	deploymentID string
	apiVersion   string
	httpClient   HTTPClient
	tokenTracker *AzureOpenAITokenTracker
}

// AzureOpenAITokenTracker tracks token usage across requests.
type AzureOpenAITokenTracker struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalRequests     int
}

// Update updates the tracker with usage from a response.
func (t *AzureOpenAITokenTracker) Update(usage Usage) {
	t.TotalInputTokens += usage.PromptTokens
	t.TotalOutputTokens += usage.CompletionTokens
	t.TotalRequests++
}

// Reset resets all counters to zero.
func (t *AzureOpenAITokenTracker) Reset() {
	t.TotalInputTokens = 0
	t.TotalOutputTokens = 0
	t.TotalRequests = 0
}

// TotalTokens returns the total number of tokens used.
func (t *AzureOpenAITokenTracker) TotalTokens() int {
	return t.TotalInputTokens + t.TotalOutputTokens
}

// AzureOpenAIMessage represents a message in the conversation.
type AzureOpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AzureOpenAIChatCompletionRequest represents a request to the Azure OpenAI chat completions API.
type AzureOpenAIChatCompletionRequest struct {
	Messages         []AzureOpenAIMessage `json:"messages"`
	Temperature      float64              `json:"temperature,omitempty"`
	MaxTokens        int                  `json:"max_tokens,omitempty"`
	TopP             float64              `json:"top_p,omitempty"`
	FrequencyPenalty float64              `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64              `json:"presence_penalty,omitempty"`
	Stream           bool                 `json:"stream,omitempty"`
}

// AzureOpenAIChatCompletionResponse represents a response from the Azure OpenAI chat completions API.
type AzureOpenAIChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
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

// AzureOpenAIStreamChunk represents a chunk from a streaming response.
type AzureOpenAIStreamChunk struct {
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

// AzureOpenAIAPIError represents an error from the Azure OpenAI API.
type AzureOpenAIAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// Error implements the error interface.
func (e *AzureOpenAIAPIError) Error() string {
	return fmt.Sprintf("azure openai error (%s): %s", e.Code, e.Message)
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider with the given configuration.
func NewAzureOpenAIProvider(config AzureOpenAIConfig) (*AzureOpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if config.DeploymentID == "" {
		return nil, errors.New("deployment ID is required")
	}

	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = DefaultAzureOpenAiDefaultApiVersion
	}

	return &AzureOpenAIProvider{
		apiKey:       config.APIKey,
		endpoint:     config.Endpoint,
		deploymentID: config.DeploymentID,
		apiVersion:   apiVersion,
		httpClient:   &http.Client{},
		tokenTracker: &AzureOpenAITokenTracker{},
	}, nil
}

// CreateChatCompletion sends a non-streaming chat completion request to the Azure OpenAI API.
func (p *AzureOpenAIProvider) CreateChatCompletion(ctx context.Context, req AzureOpenAIChatCompletionRequest) (*AzureOpenAIChatCompletionResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.endpoint, p.deploymentID, p.apiVersion)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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

	var response AzureOpenAIChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	p.tokenTracker.Update(response.Usage)

	return &response, nil
}

// CreateChatCompletionStream sends a streaming chat completion request to the Azure OpenAI API.
func (p *AzureOpenAIProvider) CreateChatCompletionStream(ctx context.Context, req AzureOpenAIChatCompletionRequest) (<-chan AzureOpenAIStreamChunk, <-chan error) {
	eventChan := make(chan AzureOpenAIStreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
			p.endpoint, p.deploymentID, p.apiVersion)

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
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

// setHeaders sets the required headers for Azure OpenAI API requests.
func (p *AzureOpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Accept", "text/event-stream")
}

// parseErrorResponse parses an error response from the Azure OpenAI API.
func (p *AzureOpenAIProvider) parseErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error *AzureOpenAIAPIError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil || errorResp.Error == nil {
		// Fallback to generic error if parsing fails
		return fmt.Errorf("API request failed with status %d: %s", statusCode, string(body))
	}

	azureErr := errorResp.Error

	// Map to provider errors based on error code
	switch azureErr.Code {
	case "invalid_api_key":
		return fmt.Errorf("%w: %s", ErrInvalidAPIKey, azureErr.Message)
	case "rate_limit_exceeded":
		return fmt.Errorf("%w: %s", ErrRateLimitExceeded, azureErr.Message)
	case "invalid_request":
		return fmt.Errorf("%w: %s", ErrInvalidRequest, azureErr.Message)
	case "not_found":
		return fmt.Errorf("%w: %s", ErrModelNotFound, azureErr.Message)
	default:
		return fmt.Errorf("%w (%s): %s", ErrProviderError, azureErr.Code, azureErr.Message)
	}
}

// parseSSEStream parses a Server-Sent Events stream and sends chunks to the channel.
func (p *AzureOpenAIProvider) parseSSEStream(reader io.Reader, chunkChan chan<- AzureOpenAIStreamChunk) error {
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

			var chunk AzureOpenAIStreamChunk
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

// GetDeploymentID returns the deployment ID being used.
func (p *AzureOpenAIProvider) GetDeploymentID() string {
	return p.deploymentID
}

// SetDeploymentID sets the deployment ID to use.
func (p *AzureOpenAIProvider) SetDeploymentID(deploymentID string) {
	p.deploymentID = deploymentID
}

// GetTokenTracker returns the token tracker for this provider.
func (p *AzureOpenAIProvider) GetTokenTracker() *AzureOpenAITokenTracker {
	return p.tokenTracker
}

// Complete implements the Provider interface.
func (p *AzureOpenAIProvider) Complete(ctx context.Context, req ProviderCompletionRequest) (*ProviderCompletionResponse, error) {
	// Convert ProviderMessage to AzureOpenAIMessage
	messages := make([]AzureOpenAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = AzureOpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	azureReq := AzureOpenAIChatCompletionRequest{
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stream:      false,
	}

	resp, err := p.CreateChatCompletion(ctx, azureReq)
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