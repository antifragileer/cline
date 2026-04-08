// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"context"
	"errors"
)

// Common errors that all providers should return
var (
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrInvalidResponse     = errors.New("invalid response from API")
	ErrProviderError       = errors.New("provider error")
	ErrContextCanceled     = errors.New("request canceled")
	ErrModelNotFound       = errors.New("model not found")
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// ProviderType represents the type of API provider
type ProviderType string

const (
	// ProviderAnthropic represents the Anthropic API provider
	ProviderAnthropic ProviderType = "anthropic"
	// ProviderOpenAI represents the OpenAI API provider
	ProviderOpenAI ProviderType = "openai"
	// ProviderOpenRouter represents the OpenRouter API provider
	ProviderOpenRouter ProviderType = "openrouter"
	// ProviderGemini represents the Google Gemini API provider
	ProviderGemini ProviderType = "gemini"
	// ProviderBedrock represents the AWS Bedrock API provider
	ProviderBedrock ProviderType = "bedrock"
	// ProviderOllama represents the Ollama local API provider
	ProviderOllama ProviderType = "ollama"
	// ProviderLMStudio represents the LM Studio local API provider
	ProviderLMStudio ProviderType = "lmstudio"
	// ProviderAzureOpenAI represents the Azure OpenAI API provider
	ProviderAzureOpenAI ProviderType = "azure"
	// ProviderCerebras represents the Cerebras API provider
	ProviderCerebras ProviderType = "cerebras"
)

// ProviderMessage represents a generic message structure used across providers
type ProviderMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ProviderUsage represents token usage information
type ProviderUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Usage represents token usage (used by multiple providers including openrouter)
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderCompletionRequest represents a generic completion request
type ProviderCompletionRequest struct {
	Model       string
	Messages    []ProviderMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stream      bool
}

// ProviderCompletionResponse represents a generic completion response
type ProviderCompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   ProviderUsage
}

// Provider is the common interface for all API providers
type Provider interface {
	Complete(ctx context.Context, req ProviderCompletionRequest) (*ProviderCompletionResponse, error)
}

// ProviderStreamChunk represents a chunk from a streaming response
type ProviderStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *ProviderUsage
}

// ModelInfo represents metadata about a model
type ModelInfo struct {
	Name                string  `json:"name"`
	MaxTokens           int     `json:"max_tokens"`
	ContextWindow       int     `json:"context_window"`
	SupportsImages      bool    `json:"supports_images"`
	SupportsPromptCache bool    `json:"supports_prompt_cache"`
	Temperature         float64 `json:"temperature"`
	InputPrice          float64 `json:"input_price"`
	OutputPrice         float64 `json:"output_price"`
	Description         string  `json:"description"`
}

// ProviderFactory creates provider instances based on configuration

type ProviderFactory struct {
	// Configurations for each provider type
	anthropicConfig   *AnthropicProviderConfig
	openAIConfig      *OpenAIConfig
	openRouterConfig  *OpenRouterConfig
	geminiConfig      *GeminiConfig
	bedrockConfig     *BedrockConfig
	ollamaConfig      *OllamaConfig
	lmStudioConfig    *LMStudioConfig
	azureOpenAIConfig *AzureOpenAIConfig
	cerebrasConfig    *CerebrasConfig
}

// AnthropicProviderConfig wraps Anthropic provider options
type AnthropicProviderConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// SetAnthropicConfig sets the Anthropic provider configuration
func (f *ProviderFactory) SetAnthropicConfig(cfg *AnthropicProviderConfig) *ProviderFactory {
	f.anthropicConfig = cfg
	return f
}

// SetOpenAIConfig sets the OpenAI provider configuration
func (f *ProviderFactory) SetOpenAIConfig(cfg *OpenAIConfig) *ProviderFactory {
	f.openAIConfig = cfg
	return f
}

// SetOpenRouterConfig sets the OpenRouter provider configuration
func (f *ProviderFactory) SetOpenRouterConfig(cfg *OpenRouterConfig) *ProviderFactory {
	f.openRouterConfig = cfg
	return f
}

// SetGeminiConfig sets the Gemini provider configuration
func (f *ProviderFactory) SetGeminiConfig(cfg *GeminiConfig) *ProviderFactory {
	f.geminiConfig = cfg
	return f
}

// SetBedrockConfig sets the Bedrock provider configuration
func (f *ProviderFactory) SetBedrockConfig(cfg *BedrockConfig) *ProviderFactory {
	f.bedrockConfig = cfg
	return f
}

// SetOllamaConfig sets the Ollama provider configuration
func (f *ProviderFactory) SetOllamaConfig(cfg *OllamaConfig) *ProviderFactory {
	f.ollamaConfig = cfg
	return f
}

// SetLMStudioConfig sets the LM Studio provider configuration
func (f *ProviderFactory) SetLMStudioConfig(cfg *LMStudioConfig) *ProviderFactory {
	f.lmStudioConfig = cfg
	return f
}

// SetAzureOpenAIConfig sets the Azure OpenAI provider configuration
func (f *ProviderFactory) SetAzureOpenAIConfig(cfg *AzureOpenAIConfig) *ProviderFactory {
	f.azureOpenAIConfig = cfg
	return f
}

// SetCerebrasConfig sets the Cerebras provider configuration
func (f *ProviderFactory) SetCerebrasConfig(cfg *CerebrasConfig) *ProviderFactory {
	f.cerebrasConfig = cfg
	return f
}

// CreateProvider creates a provider instance based on the provider type
// Returns interface{} - caller should type assert to the specific provider type
func (f *ProviderFactory) CreateProvider(providerType ProviderType) (interface{}, error) {
	switch providerType {
	case ProviderAnthropic:
		return f.createAnthropicProvider()
	case ProviderOpenAI:
		return f.createOpenAIProvider()
	case ProviderOpenRouter:
		return f.createOpenRouterProvider()
	case ProviderGemini:
		return f.createGeminiProvider()
	case ProviderBedrock:
		return f.createBedrockProvider()
	case ProviderOllama:
		return f.createOllamaProvider()
	case ProviderLMStudio:
		return f.createLMStudioProvider()
	case ProviderAzureOpenAI:
		return f.createAzureOpenAIProvider()
	case ProviderCerebras:
		return f.createCerebrasProvider()
	default:
		return nil, errors.New("unknown provider type")
	}
}

func (f *ProviderFactory) createAnthropicProvider() (*AnthropicProvider, error) {
	if f.anthropicConfig == nil {
		return nil, errors.New("anthropic configuration not set")
	}

	model := Claude35Sonnet
	if f.anthropicConfig.Model != "" {
		model = ClaudeModel(f.anthropicConfig.Model)
	}

	opts := []ProviderOption{
		WithAPIKey(f.anthropicConfig.APIKey),
		WithModel(model),
	}

	if f.anthropicConfig.BaseURL != "" {
		opts = append(opts, WithBaseURL(f.anthropicConfig.BaseURL))
	}

	return NewAnthropicProvider(opts...)
}

func (f *ProviderFactory) createOpenAIProvider() (*OpenAIProvider, error) {
	if f.openAIConfig == nil {
		return nil, errors.New("openai configuration not set")
	}

	return NewOpenAIProvider(*f.openAIConfig)
}

func (f *ProviderFactory) createOpenRouterProvider() (*OpenRouterProvider, error) {
	if f.openRouterConfig == nil {
		return nil, errors.New("openrouter configuration not set")
	}

	return NewOpenRouterProvider(*f.openRouterConfig)
}

func (f *ProviderFactory) createGeminiProvider() (*GeminiProvider, error) {
	if f.geminiConfig == nil {
		return nil, errors.New("gemini configuration not set")
	}

	return NewGeminiProvider(*f.geminiConfig)
}

func (f *ProviderFactory) createBedrockProvider() (interface{}, error) {
	if f.bedrockConfig == nil {
		return nil, errors.New("bedrock configuration not set")
	}

	// Note: Bedrock requires context for initialization
	return nil, errors.New("bedrock provider requires context for initialization, use NewBedrockProvider directly")
}

func (f *ProviderFactory) createOllamaProvider() (*OllamaProvider, error) {
	if f.ollamaConfig == nil {
		return nil, errors.New("ollama configuration not set")
	}

	return NewOllamaProvider(*f.ollamaConfig)
}

func (f *ProviderFactory) createLMStudioProvider() (*LMStudioProvider, error) {
	if f.lmStudioConfig == nil {
		return nil, errors.New("lmstudio configuration not set")
	}

	return NewLMStudioProvider(*f.lmStudioConfig)
}

func (f *ProviderFactory) createAzureOpenAIProvider() (*AzureOpenAIProvider, error) {
	if f.azureOpenAIConfig == nil {
		return nil, errors.New("azure openai configuration not set")
	}

	return NewAzureOpenAIProvider(*f.azureOpenAIConfig)
}

func (f *ProviderFactory) createCerebrasProvider() (*CerebrasProvider, error) {
	if f.cerebrasConfig == nil {
		return nil, errors.New("cerebras configuration not set")
	}

	return NewCerebrasProvider(*f.cerebrasConfig)
}

// ProviderRegistry maintains a registry of available providers
type ProviderRegistry struct {
	providers map[ProviderType]interface{}
	factory   *ProviderFactory
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry(factory *ProviderFactory) *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderType]interface{}),
		factory:   factory,
	}
}

// Register registers a provider instance
func (r *ProviderRegistry) Register(providerType ProviderType, provider interface{}) {
	r.providers[providerType] = provider
}

// Get retrieves a provider instance by type
// Returns interface{} - caller should type assert to the specific provider type
func (r *ProviderRegistry) Get(providerType ProviderType) (interface{}, error) {
	// Return cached instance if available
	if provider, ok := r.providers[providerType]; ok {
		return provider, nil
	}

	// Create new instance using factory
	if r.factory == nil {
		return nil, errors.New("provider not found and no factory configured")
	}

	provider, err := r.factory.CreateProvider(providerType)
	if err != nil {
		return nil, err
	}

	// Cache the instance
	r.providers[providerType] = provider
	return provider, nil
}

// GetAvailableProviders returns a list of available provider types
func (r *ProviderRegistry) GetAvailableProviders() []ProviderType {
	types := make([]ProviderType, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}
	return types
}

// IsRegistered checks if a provider type is registered
func (r *ProviderRegistry) IsRegistered(providerType ProviderType) bool {
	_, ok := r.providers[providerType]
	return ok
}
