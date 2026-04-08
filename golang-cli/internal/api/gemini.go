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
	"time"
)

// Gemini errors
var (
	ErrGeminiInvalidAPIKey     = errors.New("invalid Gemini API key")
	ErrGeminiRateLimitExceeded = errors.New("Gemini rate limit exceeded")
	ErrGeminiInvalidRequest    = errors.New("invalid Gemini request")
	ErrGeminiInvalidResponse   = errors.New("invalid response from Gemini API")
	ErrGeminiProviderError     = errors.New("Gemini provider error")
	ErrGeminiContextCanceled   = errors.New("Gemini request canceled")
	ErrGeminiModelNotFound     = errors.New("Gemini model not found")
	ErrGeminiSafetyBlocked     = errors.New("content blocked by Gemini safety filters")
)

// Default settings for Gemini
const (
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	DefaultGeminiTimeout = 120 * time.Second
)

// GeminiModel represents supported Gemini models
type GeminiModel string

const (
	// Gemini 2.0 models
	Gemini20Flash     GeminiModel = "gemini-2.0-flash-exp"
	Gemini20FlashLite GeminiModel = "gemini-2.0-flash-lite-preview-02-05"

	// Gemini 1.5 models
	Gemini15Pro           GeminiModel = "gemini-1.5-pro"
	Gemini15ProLatest     GeminiModel = "gemini-1.5-pro-latest"
	Gemini15Flash         GeminiModel = "gemini-1.5-flash"
	Gemini15FlashLatest   GeminiModel = "gemini-1.5-flash-latest"
	Gemini15Flash8B       GeminiModel = "gemini-1.5-flash-8b"
	Gemini15Flash8BLatest GeminiModel = "gemini-1.5-flash-8b-latest"

	// Gemini 1.0 models
	Gemini10Pro GeminiModel = "gemini-1.0-pro"
)

// geminiModels is the list of supported Gemini models
var geminiModels = []string{
	string(Gemini20Flash),
	string(Gemini20FlashLite),
	string(Gemini15Pro),
	string(Gemini15ProLatest),
	string(Gemini15Flash),
	string(Gemini15FlashLatest),
	string(Gemini15Flash8B),
	string(Gemini15Flash8BLatest),
	string(Gemini10Pro),
}

// GeminiConfig contains configuration for the Gemini provider
type GeminiConfig struct {
	// APIKey is the Gemini API key (required)
	APIKey string

	// BaseURL is the base URL for the API (defaults to DefaultGeminiBaseURL)
	BaseURL string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient HTTPClient
}

// GeminiProvider implements the Provider interface for Google's Gemini API
type GeminiProvider struct {
	config     GeminiConfig
	httpClient HTTPClient
	baseURL    string
	model      GeminiModel
}

// GeminiMessage represents a message in the conversation for Gemini
type GeminiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GeminiUsage represents token usage for a Gemini request
type GeminiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GeminiCompletionRequest represents a request for Gemini chat completion
type GeminiCompletionRequest struct {
	Model       string
	Messages    []GeminiMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	Stream      bool
}

// GeminiCompletionResponse represents a response from Gemini
type GeminiCompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   GeminiUsage
}

// GeminiStreamChunk represents a chunk from a streaming response
type GeminiStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *GeminiUsage
}

// geminiRequest represents the request body for Gemini API
type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []geminiSafetySetting   `json:"safetySettings,omitempty"`
}

// geminiContent represents content in Gemini format
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

// geminiPart represents a part of content
type geminiPart struct {
	Text string `json:"text,omitempty"`
}

// geminiGenerationConfig represents generation configuration
type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// geminiSafetySetting represents safety settings
type geminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// geminiResponse represents a non-streaming response from Gemini
type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
}

// geminiCandidate represents a response candidate
type geminiCandidate struct {
	Content       geminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason,omitempty"`
	Index         int                  `json:"index"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
}

// geminiSafetyRating represents safety rating
type geminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// geminiUsageMetadata represents usage metadata
type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// geminiPromptFeedback represents prompt feedback
type geminiPromptFeedback struct {
	BlockReason   string               `json:"blockReason,omitempty"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings,omitempty"`
}

// geminiStreamResponse represents a streaming chunk from Gemini
type geminiStreamResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// geminiErrorResponse represents an error response from Gemini
type geminiErrorResponse struct {
	Err *geminiErrorDetail `json:"error"`
}

// geminiErrorDetail represents error details
type geminiErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Error implements the error interface
func (e *geminiErrorResponse) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("gemini error: %s (code: %d, status: %s)", e.Err.Message, e.Err.Code, e.Err.Status)
	}
	return "unknown gemini error"
}

// NewGeminiProvider creates a new Gemini provider with the given configuration
func NewGeminiProvider(config GeminiConfig) (*GeminiProvider, error) {
	if config.APIKey == "" {
		return nil, ErrGeminiInvalidAPIKey
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultGeminiTimeout,
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}

	return &GeminiProvider{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      Gemini15Pro,
	}, nil
}

// NewGeminiProviderWithHTTPClient creates a new Gemini provider with a custom HTTP client
func NewGeminiProviderWithHTTPClient(config GeminiConfig, client HTTPClient) (*GeminiProvider, error) {
	if config.APIKey == "" {
		return nil, ErrGeminiInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}

	return &GeminiProvider{
		config:     config,
		httpClient: client,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      Gemini15Pro,
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *GeminiProvider) Complete(ctx context.Context, req GeminiCompletionRequest) (*GeminiCompletionResponse, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}

	geminiReq := p.toGeminiRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for blocked content
	if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("%w: %s", ErrGeminiSafetyBlocked, geminiResp.PromptFeedback.BlockReason)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, ErrGeminiInvalidResponse
	}

	// Extract content from first candidate
	content := p.extractContent(geminiResp.Candidates[0])

	// Extract usage metadata
	usage := GeminiUsage{}
	if geminiResp.UsageMetadata != nil {
		usage.PromptTokens = geminiResp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = geminiResp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = geminiResp.UsageMetadata.TotalTokenCount
	}

	return &GeminiCompletionResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().Unix()),
		Model:   req.Model,
		Content: content,
		Usage:   usage,
	}, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *GeminiProvider) CompleteStream(ctx context.Context, req GeminiCompletionRequest) (<-chan GeminiStreamChunk, <-chan error, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, nil, err
	}

	geminiReq := p.toGeminiRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, req.Model, p.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if err := p.handleErrorResponse(resp); err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	chunkChan := make(chan GeminiStreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, req, resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the SSE stream from Gemini
func (p *GeminiProvider) processStream(ctx context.Context, req GeminiCompletionRequest, body io.ReadCloser, chunkChan chan<- GeminiStreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	// Track accumulated usage for the final chunk
	var accumulatedUsage GeminiUsage
	contentBuffer := ""

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ErrGeminiContextCanceled
			return
		default:
		}

		line := scanner.Text()

		// SSE format: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Stream terminator
		if data == "[DONE]" {
			// Send final chunk with accumulated usage
			if accumulatedUsage.TotalTokens > 0 {
				chunkChan <- GeminiStreamChunk{
					Content:      contentBuffer,
					FinishReason: "stop",
					Usage:        &accumulatedUsage,
				}
			}
			return
		}

		var streamResp geminiStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			errChan <- fmt.Errorf("failed to decode stream chunk: %w", err)
			return
		}

		if len(streamResp.Candidates) == 0 {
			continue
		}

		candidate := streamResp.Candidates[0]

		// Accumulate usage if provided
		if streamResp.UsageMetadata != nil {
			accumulatedUsage = GeminiUsage{
				PromptTokens:     streamResp.UsageMetadata.PromptTokenCount,
				CompletionTokens: streamResp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      streamResp.UsageMetadata.TotalTokenCount,
			}
		}

		// Extract delta content
		delta := p.extractContent(candidate)
		contentBuffer += delta

		chunk := GeminiStreamChunk{
			Delta:   delta,
			Content: contentBuffer,
		}

		if candidate.FinishReason != "" {
			chunk.FinishReason = candidate.FinishReason

			// Estimate usage if not provided by the API
			if accumulatedUsage.TotalTokens == 0 {
				accumulatedUsage = p.estimateUsage(req.Messages, contentBuffer)
			}

			chunk.Usage = &accumulatedUsage
		}

		select {
		case chunkChan <- chunk:
		case <-ctx.Done():
			errChan <- ErrGeminiContextCanceled
			return
		}

		// If finished, we can stop processing
		if candidate.FinishReason != "" {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("stream read error: %w", err)
	}
}

// toGeminiRequest converts a GeminiCompletionRequest to a Gemini-specific request
func (p *GeminiProvider) toGeminiRequest(req GeminiCompletionRequest) geminiRequest {
	contents := make([]geminiContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := msg.Role
		// Map standard roles to Gemini roles
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
		})
	}

	genConfig := &geminiGenerationConfig{
		Temperature:     req.Temperature,
		MaxOutputTokens: req.MaxTokens,
		TopP:            req.TopP,
		TopK:            req.TopK,
	}

	// Set defaults if not provided
	if genConfig.Temperature == 0 {
		genConfig.Temperature = 0.7
	}
	if genConfig.MaxOutputTokens == 0 {
		genConfig.MaxOutputTokens = 4096
	}
	if genConfig.TopP == 0 {
		genConfig.TopP = 1.0
	}

	// Default safety settings - allow most content
	safetySettings := []geminiSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
	}

	return geminiRequest{
		Contents:         contents,
		GenerationConfig: genConfig,
		SafetySettings:   safetySettings,
	}
}

// extractContent extracts text content from a candidate
func (p *GeminiProvider) extractContent(candidate geminiCandidate) string {
	var result strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			result.WriteString(part.Text)
		}
	}
	return result.String()
}

// estimateUsage estimates token usage when the API doesn't provide it
func (p *GeminiProvider) estimateUsage(messages []GeminiMessage, completion string) GeminiUsage {
	// Rough estimation: ~4 characters per token for English text
	promptChars := 0
	for _, msg := range messages {
		promptChars += len(msg.Role) + len(msg.Content)
	}
	completionChars := len(completion)

	promptTokens := promptChars / 4
	if promptTokens < 1 {
		promptTokens = 1
	}

	completionTokens := completionChars / 4
	if completionTokens < 1 {
		completionTokens = 1
	}

	return GeminiUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// GetSupportedModels returns a list of supported model IDs
func (p *GeminiProvider) GetSupportedModels() []string {
	models := make([]string, len(geminiModels))
	copy(models, geminiModels)
	return models
}

// ValidateModel checks if a model is supported
func (p *GeminiProvider) ValidateModel(model string) error {
	for _, m := range geminiModels {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrGeminiModelNotFound, model)
}

// setHeaders sets the required HTTP headers for Gemini requests
func (p *GeminiProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
}

// handleErrorResponse handles HTTP error responses
func (p *GeminiProvider) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body: %w", resp.StatusCode, err)
	}

	// Try to parse as Gemini error
	var errResp geminiErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Err != nil {
		return p.convertGeminiError(&errResp)
	}

	// Fallback to generic error
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return ErrGeminiRateLimitExceeded
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrGeminiInvalidAPIKey
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrGeminiInvalidRequest, string(body))
	default:
		return fmt.Errorf("%w: HTTP %d - %s", ErrGeminiProviderError, resp.StatusCode, string(body))
	}
}

// convertGeminiError converts a Gemini error to the appropriate error type
func (p *GeminiProvider) convertGeminiError(err *geminiErrorResponse) error {
	if err.Err == nil {
		return ErrGeminiProviderError
	}

	switch err.Err.Status {
	case "PERMISSION_DENIED", "UNAUTHENTICATED":
		return fmt.Errorf("%w: %s", ErrGeminiInvalidAPIKey, err.Err.Message)
	case "RESOURCE_EXHAUSTED":
		return fmt.Errorf("%w: %s", ErrGeminiRateLimitExceeded, err.Err.Message)
	case "INVALID_ARGUMENT":
		return fmt.Errorf("%w: %s", ErrGeminiInvalidRequest, err.Err.Message)
	default:
		return fmt.Errorf("%w: %s", ErrGeminiProviderError, err.Err.Message)
	}
}

// GetModel returns the current default model
func (p *GeminiProvider) GetModel() string {
	return string(p.model)
}

// SetModel sets the default model
func (p *GeminiProvider) SetModel(model string) {
	p.model = GeminiModel(model)
}

// GetModelContextWindow returns the context window size for a model
func GetModelContextWindow(model GeminiModel) int {
	switch model {
	case Gemini15Pro, Gemini15ProLatest:
		return 2000000 // 2M tokens
	case Gemini15Flash, Gemini15FlashLatest, Gemini15Flash8B, Gemini15Flash8BLatest:
		return 1000000 // 1M tokens
	case Gemini20Flash, Gemini20FlashLite:
		return 1000000 // 1M tokens
	case Gemini10Pro:
		return 32768 // 32K tokens
	default:
		return 32768 // Default to 32K
	}
}

// IsMultimodalModel returns true if the model supports multimodal inputs
func IsMultimodalModel(model GeminiModel) bool {
	switch model {
	case Gemini15Pro, Gemini15ProLatest,
		Gemini15Flash, Gemini15FlashLatest,
		Gemini15Flash8B, Gemini15Flash8BLatest,
		Gemini20Flash, Gemini20FlashLite:
		return true
	default:
		return false
	}
}
