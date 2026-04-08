// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go"
)

// Bedrock errors
var (
	ErrBedrockInvalidCredentials = errors.New("invalid AWS credentials")
	ErrBedrockRateLimitExceeded  = errors.New("AWS Bedrock rate limit exceeded")
	ErrBedrockInvalidRequest     = errors.New("invalid Bedrock request")
	ErrBedrockInvalidResponse    = errors.New("invalid response from Bedrock API")
	ErrBedrockProviderError      = errors.New("Bedrock provider error")
	ErrBedrockContextCanceled    = errors.New("Bedrock request canceled")
	ErrBedrockModelNotFound      = errors.New("Bedrock model not found")
	ErrBedrockModelNotAccessible = errors.New("Bedrock model not accessible in this region")
	ErrBedrockValidationError    = errors.New("Bedrock validation error")
	ErrBedrockThrottling         = errors.New("Bedrock throttling error")
)

// Default settings for Bedrock
const (
	DefaultBedrockRegion  = "us-east-1"
	DefaultBedrockTimeout = 120 * time.Second
)

// BedrockModel represents supported Bedrock models
type BedrockModel string

const (
	// Anthropic Claude models
	BedrockClaude35Sonnet   BedrockModel = "anthropic.claude-3-5-sonnet-20241022-v2:0"
	BedrockClaude35SonnetV1 BedrockModel = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	BedrockClaude3Opus      BedrockModel = "anthropic.claude-3-opus-20240229-v1:0"
	BedrockClaude3Sonnet    BedrockModel = "anthropic.claude-3-sonnet-20240229-v1:0"
	BedrockClaude3Haiku     BedrockModel = "anthropic.claude-3-haiku-20240307-v1:0"

	// Amazon Titan models
	BedrockTitanPremier BedrockModel = "amazon.titan-text-premier-v1:0"
	BedrockTitanExpress BedrockModel = "amazon.titan-text-express-v1"
	BedrockTitanLite    BedrockModel = "amazon.titan-text-lite-v1"

	// Meta Llama models
	BedrockLlama405B BedrockModel = "meta.llama3-1-405b-instruct-v1:0"
	BedrockLlama70B  BedrockModel = "meta.llama3-1-70b-instruct-v1:0"
	BedrockLlama8B   BedrockModel = "meta.llama3-1-8b-instruct-v1:0"

	// Mistral models
	BedrockMistralLarge BedrockModel = "mistral.mistral-large-2402-v1:0"
	BedrockMistral7B    BedrockModel = "mistral.mistral-7b-instruct-v0:2"
	BedrockMixtral8x7B  BedrockModel = "mistral.mixtral-8x7b-instruct-v0:1"

	// Cohere models
	BedrockCohereCommandR     BedrockModel = "cohere.command-r-v1:0"
	BedrockCohereCommandRPlus BedrockModel = "cohere.command-r-plus-v1:0"

	// AI21 models
	BedrockAI21JambaInstruct BedrockModel = "ai21.jamba-instruct-v1:0"
)

// bedrockModels is the list of supported Bedrock models
var bedrockModels = []string{
	string(BedrockClaude35Sonnet),
	string(BedrockClaude35SonnetV1),
	string(BedrockClaude3Opus),
	string(BedrockClaude3Sonnet),
	string(BedrockClaude3Haiku),
	string(BedrockTitanPremier),
	string(BedrockTitanExpress),
	string(BedrockTitanLite),
	string(BedrockLlama405B),
	string(BedrockLlama70B),
	string(BedrockLlama8B),
	string(BedrockMistralLarge),
	string(BedrockMistral7B),
	string(BedrockMixtral8x7B),
	string(BedrockCohereCommandR),
	string(BedrockCohereCommandRPlus),
	string(BedrockAI21JambaInstruct),
}

// BedrockConfig contains configuration for the Bedrock provider
type BedrockConfig struct {
	// Region is the AWS region (defaults to us-east-1)
	Region string

	// AccessKeyID is the AWS access key ID (optional, uses default credential chain if not provided)
	AccessKeyID string

	// SecretAccessKey is the AWS secret access key (optional, uses default credential chain if not provided)
	SecretAccessKey string

	// SessionToken is the AWS session token for temporary credentials (optional)
	SessionToken string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient *http.Client
}

// BedrockProvider implements the Provider interface for AWS Bedrock
type BedrockProvider struct {
	config    BedrockConfig
	client    *bedrockruntime.Client
	model     BedrockModel
	awsConfig aws.Config
}

// BedrockMessage represents a message in the conversation for Bedrock
type BedrockMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// BedrockUsage represents token usage for a Bedrock request
type BedrockUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// BedrockCompletionRequest represents a request for Bedrock chat completion
type BedrockCompletionRequest struct {
	Model       string
	Messages    []BedrockMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	Stream      bool
}

// BedrockCompletionResponse represents a response from Bedrock
type BedrockCompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   BedrockUsage
}

// BedrockStreamChunk represents a chunk from a streaming response
type BedrockStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *BedrockUsage
}

// NewBedrockProvider creates a new Bedrock provider with the given configuration
func NewBedrockProvider(ctx context.Context, cfg BedrockConfig) (*BedrockProvider, error) {
	// Set default region
	region := cfg.Region
	if region == "" {
		region = DefaultBedrockRegion
	}

	// Build AWS configuration options
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(region))

	// Use explicit credentials if provided
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			cfg.SessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock runtime client
	client := bedrockruntime.NewFromConfig(awsCfg)

	return &BedrockProvider{
		config:    cfg,
		client:    client,
		model:     BedrockClaude35Sonnet,
		awsConfig: awsCfg,
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *BedrockProvider) Complete(ctx context.Context, req BedrockCompletionRequest) (*BedrockCompletionResponse, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}

	// Prepare the request body based on model type
	body, err := p.prepareRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Invoke the model
	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(req.Model),
		ContentType: aws.String("application/json"),
		Body:        body,
	}

	output, err := p.client.InvokeModel(ctx, input)
	if err != nil {
		return nil, p.convertAWSError(err)
	}

	// Parse the response
	response, err := p.parseResponse(output.Body, req.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *BedrockProvider) CompleteStream(ctx context.Context, req BedrockCompletionRequest) (<-chan BedrockStreamChunk, <-chan error, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, nil, err
	}

	// Prepare the request body with streaming enabled
	body, err := p.prepareRequestBody(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Invoke the model with streaming
	input := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(req.Model),
		ContentType: aws.String("application/json"),
		Body:        body,
	}

	output, err := p.client.InvokeModelWithResponseStream(ctx, input)
	if err != nil {
		return nil, nil, p.convertAWSError(err)
	}

	chunkChan := make(chan BedrockStreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, req, output, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the response stream from Bedrock
func (p *BedrockProvider) processStream(ctx context.Context, req BedrockCompletionRequest, output *bedrockruntime.InvokeModelWithResponseStreamOutput, chunkChan chan<- BedrockStreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)

	// Track accumulated usage for the final chunk
	var accumulatedUsage BedrockUsage
	contentBuffer := ""

	// Determine the model type for parsing
	modelType := p.getModelType(req.Model)

	for event := range output.GetStream().Events() {
		select {
		case <-ctx.Done():
			errChan <- ErrBedrockContextCanceled
			return
		default:
		}

		// Try to extract chunk bytes from the event
		// The AWS SDK uses different event types for different models
		if chunkData, ok := getChunkFromEvent(event); ok {
			chunk, finishReason, usage, err := p.parseStreamChunk(chunkData, modelType)
			if err != nil {
				errChan <- fmt.Errorf("failed to parse stream chunk: %w", err)
				return
			}

			contentBuffer += chunk

			streamChunk := BedrockStreamChunk{
				Delta:   chunk,
				Content: contentBuffer,
			}

			if finishReason != "" {
				streamChunk.FinishReason = finishReason

				// Use provided usage or estimate
				if usage != nil {
					accumulatedUsage = *usage
				} else {
					accumulatedUsage = p.estimateUsage(req.Messages, contentBuffer)
				}

				streamChunk.Usage = &accumulatedUsage
			}

			select {
			case chunkChan <- streamChunk:
			case <-ctx.Done():
				errChan <- ErrBedrockContextCanceled
				return
			}

			if finishReason != "" {
				return
			}
		}
	}

	// Check for stream errors
	if err := output.GetStream().Err(); err != nil {
		errChan <- p.convertAWSError(err)
	}
}

// getChunkFromEvent extracts chunk data from a stream event
func getChunkFromEvent(event interface{}) ([]byte, bool) {
	// Try to extract bytes from common event types
	switch v := event.(type) {
	case interface{ GetBytes() []byte }:
		return v.GetBytes(), true
	default:
		return nil, false
	}
}

// prepareRequestBody prepares the request body based on the model type
func (p *BedrockProvider) prepareRequestBody(req BedrockCompletionRequest) ([]byte, error) {
	modelType := p.getModelType(req.Model)

	switch modelType {
	case "anthropic":
		return p.prepareAnthropicBody(req)
	case "amazon":
		return p.prepareAmazonBody(req)
	case "meta":
		return p.prepareMetaBody(req)
	case "mistral":
		return p.prepareMistralBody(req)
	case "cohere":
		return p.prepareCohereBody(req)
	case "ai21":
		return p.prepareAI21Body(req)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// prepareAnthropicBody prepares request body for Anthropic models
func (p *BedrockProvider) prepareAnthropicBody(req BedrockCompletionRequest) ([]byte, error) {
	// Convert messages to Anthropic format
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, map[string]interface{}{
			"role": role,
			"content": []map[string]interface{}{
				{"type": "text", "text": msg.Content},
			},
		})
	}

	body := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"messages":          messages,
		"max_tokens":        req.MaxTokens,
	}

	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if req.TopK > 0 {
		body["top_k"] = req.TopK
	}

	// Add default max_tokens if not specified
	if req.MaxTokens == 0 {
		body["max_tokens"] = 4096
	}

	return json.Marshal(body)
}

// prepareAmazonBody prepares request body for Amazon Titan models
func (p *BedrockProvider) prepareAmazonBody(req BedrockCompletionRequest) ([]byte, error) {
	// Combine messages into a single prompt
	var prompt strings.Builder
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		} else {
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		}
	}

	body := map[string]interface{}{
		"inputText": prompt.String(),
		"textGenerationConfig": map[string]interface{}{
			"maxTokenCount": req.MaxTokens,
			"temperature":   req.Temperature,
			"topP":          req.TopP,
		},
	}

	if req.MaxTokens == 0 {
		body["textGenerationConfig"].(map[string]interface{})["maxTokenCount"] = 4096
	}
	if req.Temperature == 0 {
		body["textGenerationConfig"].(map[string]interface{})["temperature"] = 0.7
	}

	return json.Marshal(body)
}

// prepareMetaBody prepares request body for Meta Llama models
func (p *BedrockProvider) prepareMetaBody(req BedrockCompletionRequest) ([]byte, error) {
	// Combine messages into a prompt with special tokens
	var prompt strings.Builder
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			prompt.WriteString(fmt.Sprintf("<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\n%s<|eot_id|>", msg.Content))
		case "user":
			prompt.WriteString(fmt.Sprintf("<|start_header_id|>user<|end_header_id|>\n\n%s<|eot_id|>", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("<|start_header_id|>assistant<|end_header_id|>\n\n%s<|eot_id|>", msg.Content))
		}
	}
	prompt.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")

	body := map[string]interface{}{
		"prompt":      prompt.String(),
		"max_gen_len": req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
	}

	if req.MaxTokens == 0 {
		body["max_gen_len"] = 4096
	}
	if req.Temperature == 0 {
		body["temperature"] = 0.7
	}

	return json.Marshal(body)
}

// prepareMistralBody prepares request body for Mistral models
func (p *BedrockProvider) prepareMistralBody(req BedrockCompletionRequest) ([]byte, error) {
	// Combine messages into a prompt
	var prompt strings.Builder
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			prompt.WriteString(fmt.Sprintf("<s>[INST] %s [/INST]</s>\n", msg.Content))
		} else if msg.Role == "user" {
			prompt.WriteString(fmt.Sprintf("<s>[INST] %s [/INST]</s>\n", msg.Content))
		} else {
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		}
	}

	body := map[string]interface{}{
		"prompt":      prompt.String(),
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
	}

	if req.MaxTokens == 0 {
		body["max_tokens"] = 4096
	}
	if req.Temperature == 0 {
		body["temperature"] = 0.7
	}

	return json.Marshal(body)
}

// prepareCohereBody prepares request body for Cohere models
func (p *BedrockProvider) prepareCohereBody(req BedrockCompletionRequest) ([]byte, error) {
	// Extract the last user message as the main message
	var message string
	var chatHistory []map[string]string

	for i, msg := range req.Messages {
		if msg.Role == "system" {
			continue
		}
		if i == len(req.Messages)-1 && msg.Role == "user" {
			message = msg.Content
		} else {
			role := msg.Role
			if role == "assistant" {
				role = "CHATBOT"
			} else if role == "user" {
				role = "USER"
			}
			chatHistory = append(chatHistory, map[string]string{
				"role":    role,
				"message": msg.Content,
			})
		}
	}

	body := map[string]interface{}{
		"message":      message,
		"chat_history": chatHistory,
		"max_tokens":   req.MaxTokens,
		"temperature":  req.Temperature,
	}

	if req.MaxTokens == 0 {
		body["max_tokens"] = 4096
	}
	if req.Temperature == 0 {
		body["temperature"] = 0.7
	}

	return json.Marshal(body)
}

// prepareAI21Body prepares request body for AI21 models
func (p *BedrockProvider) prepareAI21Body(req BedrockCompletionRequest) ([]byte, error) {
	// Combine messages into a single prompt
	var prompt strings.Builder
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n\n")
		} else {
			prompt.WriteString(msg.Content)
			prompt.WriteString("\n")
		}
	}

	body := map[string]interface{}{
		"prompt":      prompt.String(),
		"maxTokens":   req.MaxTokens,
		"temperature": req.Temperature,
		"topP":        req.TopP,
	}

	if req.MaxTokens == 0 {
		body["maxTokens"] = 4096
	}
	if req.Temperature == 0 {
		body["temperature"] = 0.7
	}

	return json.Marshal(body)
}

// parseResponse parses the response based on model type
func (p *BedrockProvider) parseResponse(body []byte, modelID string) (*BedrockCompletionResponse, error) {
	modelType := p.getModelType(modelID)

	switch modelType {
	case "anthropic":
		return p.parseAnthropicResponse(body)
	case "amazon":
		return p.parseAmazonResponse(body)
	case "meta":
		return p.parseMetaResponse(body)
	case "mistral":
		return p.parseMistralResponse(body)
	case "cohere":
		return p.parseCohereResponse(body)
	case "ai21":
		return p.parseAI21Response(body)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// parseAnthropicResponse parses Anthropic model response
func (p *BedrockProvider) parseAnthropicResponse(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var content string
	for _, c := range response.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &BedrockCompletionResponse{
		ID:      response.ID,
		Model:   string(p.model),
		Content: content,
		Usage: BedrockUsage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
	}, nil
}

// parseAmazonResponse parses Amazon Titan model response
func (p *BedrockProvider) parseAmazonResponse(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		InputTextTokenCount int `json:"inputTextTokenCount"`
		Results             []struct {
			TokenCount       int    `json:"tokenCount"`
			OutputText       string `json:"outputText"`
			CompletionReason string `json:"completionReason"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.Results) == 0 {
		return nil, ErrBedrockInvalidResponse
	}

	return &BedrockCompletionResponse{
		ID:      fmt.Sprintf("bedrock-%d", time.Now().Unix()),
		Model:   string(p.model),
		Content: response.Results[0].OutputText,
		Usage: BedrockUsage{
			PromptTokens:     response.InputTextTokenCount,
			CompletionTokens: response.Results[0].TokenCount,
			TotalTokens:      response.InputTextTokenCount + response.Results[0].TokenCount,
		},
	}, nil
}

// parseMetaResponse parses Meta Llama model response
func (p *BedrockProvider) parseMetaResponse(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		Generation           string `json:"generation"`
		PromptTokenCount     int    `json:"prompt_token_count"`
		GenerationTokenCount int    `json:"generation_token_count"`
		StopReason           string `json:"stop_reason"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &BedrockCompletionResponse{
		ID:      fmt.Sprintf("bedrock-%d", time.Now().Unix()),
		Model:   string(p.model),
		Content: response.Generation,
		Usage: BedrockUsage{
			PromptTokens:     response.PromptTokenCount,
			CompletionTokens: response.GenerationTokenCount,
			TotalTokens:      response.PromptTokenCount + response.GenerationTokenCount,
		},
	}, nil
}

// parseMistralResponse parses Mistral model response
func (p *BedrockProvider) parseMistralResponse(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		Outputs []struct {
			Text       string `json:"text"`
			StopReason string `json:"stop_reason"`
		} `json:"outputs"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.Outputs) == 0 {
		return nil, ErrBedrockInvalidResponse
	}

	return &BedrockCompletionResponse{
		ID:      fmt.Sprintf("bedrock-%d", time.Now().Unix()),
		Model:   string(p.model),
		Content: response.Outputs[0].Text,
		Usage:   BedrockUsage{}, // Mistral doesn't provide usage in response
	}, nil
}

// parseCohereResponse parses Cohere model response
func (p *BedrockProvider) parseCohereResponse(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &BedrockCompletionResponse{
		ID:      fmt.Sprintf("bedrock-%d", time.Now().Unix()),
		Model:   string(p.model),
		Content: response.Text,
		Usage:   BedrockUsage{}, // Cohere doesn't provide usage in response
	}, nil
}

// parseAI21Response parses AI21 model response
func (p *BedrockProvider) parseAI21Response(body []byte) (*BedrockCompletionResponse, error) {
	var response struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"promptTokens"`
			CompletionTokens int `json:"completionTokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, ErrBedrockInvalidResponse
	}

	return &BedrockCompletionResponse{
		ID:      fmt.Sprintf("bedrock-%d", time.Now().Unix()),
		Model:   string(p.model),
		Content: response.Choices[0].Text,
		Usage: BedrockUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.PromptTokens + response.Usage.CompletionTokens,
		},
	}, nil
}

// parseStreamChunk parses a streaming chunk based on model type
func (p *BedrockProvider) parseStreamChunk(chunk []byte, modelType string) (string, string, *BedrockUsage, error) {
	switch modelType {
	case "anthropic":
		return p.parseAnthropicStreamChunk(chunk)
	case "amazon":
		return p.parseAmazonStreamChunk(chunk)
	case "meta":
		return p.parseMetaStreamChunk(chunk)
	case "mistral":
		return p.parseMistralStreamChunk(chunk)
	default:
		// For models without specific streaming support, treat as raw text
		return string(chunk), "", nil, nil
	}
}

// parseAnthropicStreamChunk parses Anthropic streaming chunk
func (p *BedrockProvider) parseAnthropicStreamChunk(chunk []byte) (string, string, *BedrockUsage, error) {
	var event struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
		ContentBlock struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content_block"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(chunk, &event); err != nil {
		// Try to parse as raw text if JSON parsing fails
		return string(chunk), "", nil, nil
	}

	switch event.Type {
	case "content_block_delta":
		return event.Delta.Text, "", nil, nil
	case "message_stop":
		return "", "stop", &BedrockUsage{
			PromptTokens:     event.Usage.InputTokens,
			CompletionTokens: event.Usage.OutputTokens,
			TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
		}, nil
	default:
		return "", "", nil, nil
	}
}

// parseAmazonStreamChunk parses Amazon Titan streaming chunk
func (p *BedrockProvider) parseAmazonStreamChunk(chunk []byte) (string, string, *BedrockUsage, error) {
	var event struct {
		OutputText       string `json:"outputText"`
		CompletionReason string `json:"completionReason"`
	}

	if err := json.Unmarshal(chunk, &event); err != nil {
		return string(chunk), "", nil, nil
	}

	finishReason := ""
	if event.CompletionReason != "" {
		finishReason = event.CompletionReason
	}

	return event.OutputText, finishReason, nil, nil
}

// parseMetaStreamChunk parses Meta Llama streaming chunk
func (p *BedrockProvider) parseMetaStreamChunk(chunk []byte) (string, string, *BedrockUsage, error) {
	var event struct {
		Generation string `json:"generation"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(chunk, &event); err != nil {
		return string(chunk), "", nil, nil
	}

	finishReason := ""
	if event.StopReason != "" {
		finishReason = event.StopReason
	}

	return event.Generation, finishReason, nil, nil
}

// parseMistralStreamChunk parses Mistral streaming chunk
func (p *BedrockProvider) parseMistralStreamChunk(chunk []byte) (string, string, *BedrockUsage, error) {
	var event struct {
		Outputs []struct {
			Text       string `json:"text"`
			StopReason string `json:"stop_reason"`
		} `json:"outputs"`
	}

	if err := json.Unmarshal(chunk, &event); err != nil {
		return string(chunk), "", nil, nil
	}

	if len(event.Outputs) == 0 {
		return "", "", nil, nil
	}

	finishReason := ""
	if event.Outputs[0].StopReason != "" {
		finishReason = event.Outputs[0].StopReason
	}

	return event.Outputs[0].Text, finishReason, nil, nil
}

// getModelType returns the model type based on model ID
func (p *BedrockProvider) getModelType(modelID string) string {
	if strings.Contains(modelID, "anthropic") {
		return "anthropic"
	}
	if strings.Contains(modelID, "amazon") {
		return "amazon"
	}
	if strings.Contains(modelID, "meta") {
		return "meta"
	}
	if strings.Contains(modelID, "mistral") {
		return "mistral"
	}
	if strings.Contains(modelID, "cohere") {
		return "cohere"
	}
	if strings.Contains(modelID, "ai21") {
		return "ai21"
	}
	return "unknown"
}

// estimateUsage estimates token usage when the API doesn't provide it
func (p *BedrockProvider) estimateUsage(messages []BedrockMessage, completion string) BedrockUsage {
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

	return BedrockUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// GetSupportedModels returns a list of supported model IDs
func (p *BedrockProvider) GetSupportedModels() []string {
	models := make([]string, len(bedrockModels))
	copy(models, bedrockModels)
	return models
}

// ValidateModel checks if a model is supported
func (p *BedrockProvider) ValidateModel(model string) error {
	for _, m := range bedrockModels {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrBedrockModelNotFound, model)
}

// convertAWSError converts an AWS error to the appropriate error type
func (p *BedrockProvider) convertAWSError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific AWS error types
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "AccessDeniedException":
			return fmt.Errorf("%w: %s", ErrBedrockInvalidCredentials, apiErr.ErrorMessage())
		case "ResourceNotFoundException":
			return fmt.Errorf("%w: %s", ErrBedrockModelNotAccessible, apiErr.ErrorMessage())
		case "ValidationException":
			return fmt.Errorf("%w: %s", ErrBedrockValidationError, apiErr.ErrorMessage())
		case "ThrottlingException":
			return fmt.Errorf("%w: %s", ErrBedrockThrottling, apiErr.ErrorMessage())
		case "ServiceQuotaExceededException":
			return fmt.Errorf("%w: %s", ErrBedrockRateLimitExceeded, apiErr.ErrorMessage())
		}
	}

	// Check for context cancellation
	if errors.Is(err, context.Canceled) {
		return ErrBedrockContextCanceled
	}

	return fmt.Errorf("%w: %s", ErrBedrockProviderError, err.Error())
}

// GetModel returns the current default model
func (p *BedrockProvider) GetModel() BedrockModel {
	return p.model
}

// SetModel sets the default model
func (p *BedrockProvider) SetModel(model BedrockModel) {
	p.model = model
}

// GetRegion returns the AWS region
func (p *BedrockProvider) GetRegion() string {
	return p.config.Region
}

// IsAnthropicModel returns true if the model is an Anthropic model
func IsAnthropicModel(model BedrockModel) bool {
	return strings.HasPrefix(string(model), "anthropic.")
}

// IsAmazonModel returns true if the model is an Amazon model
func IsAmazonModel(model BedrockModel) bool {
	return strings.HasPrefix(string(model), "amazon.")
}

// IsMetaModel returns true if the model is a Meta model
func IsMetaModel(model BedrockModel) bool {
	return strings.HasPrefix(string(model), "meta.")
}
