package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// AuthMethod represents the authentication method for a provider
type AuthMethod string

const (
	// AuthMethodAPIKey uses a direct API key
	AuthMethodAPIKey AuthMethod = "api_key"
	// AuthMethodOAuth uses OAuth 2.0 flow
	AuthMethodOAuth AuthMethod = "oauth"
	// AuthMethodNone requires no authentication (local providers)
	AuthMethodNone AuthMethod = "none"
)

// ModelInfo represents information about an AI model
type ModelInfo struct {
	ID          string
	Name        string
	Description string
	ContextSize int
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Name         string
	DisplayName  string
	Description  string
	AuthMethods  []AuthMethod
	Models       []ModelInfo
	DefaultModel string
}

// WizardFlags contains command-line flags for non-interactive mode
type WizardFlags struct {
	Provider       string
	AuthMethod     string
	APIKey         string
	Model          string
	NonInteractive bool
	Force          bool
}

// WizardResult contains the result of the configuration wizard
type WizardResult struct {
	Provider   string
	AuthMethod AuthMethod
	APIKey     string
	Model      string
	Validated  bool
}

// Wizard manages the provider configuration wizard
type Wizard struct {
	providers     map[string]ProviderInfo
	apiKeyManager *APIKeyManager
	input         terminal.FileReader
	output        terminal.FileWriter
}

// NewWizard creates a new configuration wizard
func NewWizard(apiKeyManager *APIKeyManager) *Wizard {
	return NewWizardWithIO(apiKeyManager, os.Stdin, os.Stdout)
}

// NewWizardWithIO creates a wizard with custom input/output
func NewWizardWithIO(apiKeyManager *APIKeyManager, input terminal.FileReader, output terminal.FileWriter) *Wizard {
	w := &Wizard{
		providers:     make(map[string]ProviderInfo),
		apiKeyManager: apiKeyManager,
		input:         input,
		output:        output,
	}
	w.registerProviders()
	return w
}

// registerProviders initializes the built-in provider configurations
func (w *Wizard) registerProviders() {
	w.providers[ProviderAnthropic] = ProviderInfo{
		Name:         ProviderAnthropic,
		DisplayName:  "Anthropic",
		Description:  "Claude models by Anthropic",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey},
		DefaultModel: "claude-3-sonnet-20240229",
		Models: []ModelInfo{
			{
				ID:          "claude-3-opus-20240229",
				Name:        "Claude 3 Opus",
				Description: "Most powerful model for complex tasks",
				ContextSize: 200000,
			},
			{
				ID:          "claude-3-sonnet-20240229",
				Name:        "Claude 3 Sonnet",
				Description: "Balanced performance and speed",
				ContextSize: 200000,
			},
			{
				ID:          "claude-3-haiku-20240307",
				Name:        "Claude 3 Haiku",
				Description: "Fastest model for everyday tasks",
				ContextSize: 200000,
			},
		},
	}

	w.providers[ProviderOpenAI] = ProviderInfo{
		Name:         ProviderOpenAI,
		DisplayName:  "OpenAI",
		Description:  "GPT models by OpenAI",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey},
		DefaultModel: "gpt-4",
		Models: []ModelInfo{
			{
				ID:          "gpt-4",
				Name:        "GPT-4",
				Description: "Most capable GPT-4 model",
				ContextSize: 8192,
			},
			{
				ID:          "gpt-4-turbo",
				Name:        "GPT-4 Turbo",
				Description: "Latest GPT-4 model with improved performance",
				ContextSize: 128000,
			},
			{
				ID:          "gpt-3.5-turbo",
				Name:        "GPT-3.5 Turbo",
				Description: "Fast and cost-effective",
				ContextSize: 16385,
			},
		},
	}

	w.providers[ProviderOpenRouter] = ProviderInfo{
		Name:         ProviderOpenRouter,
		DisplayName:  "OpenRouter",
		Description:  "Unified API for multiple providers",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey},
		DefaultModel: "anthropic/claude-3-sonnet",
		Models: []ModelInfo{
			{
				ID:          "anthropic/claude-3-opus",
				Name:        "Claude 3 Opus (via OpenRouter)",
				Description: "Most powerful Claude model",
				ContextSize: 200000,
			},
			{
				ID:          "anthropic/claude-3-sonnet",
				Name:        "Claude 3 Sonnet (via OpenRouter)",
				Description: "Balanced Claude model",
				ContextSize: 200000,
			},
			{
				ID:          "openai/gpt-4-turbo",
				Name:        "GPT-4 Turbo (via OpenRouter)",
				Description: "Latest GPT-4 model",
				ContextSize: 128000,
			},
		},
	}

	w.providers[ProviderGemini] = ProviderInfo{
		Name:         ProviderGemini,
		DisplayName:  "Google Gemini",
		Description:  "Gemini models by Google",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey},
		DefaultModel: "gemini-1.5-flash",
		Models: []ModelInfo{
			{
				ID:          "gemini-1.5-flash",
				Name:        "Gemini 1.5 Flash",
				Description: "Fast and versatile",
				ContextSize: 1000000,
			},
			{
				ID:          "gemini-1.5-pro",
				Name:        "Gemini 1.5 Pro",
				Description: "Advanced reasoning and coding",
				ContextSize: 2000000,
			},
		},
	}

	w.providers[ProviderBedrock] = ProviderInfo{
		Name:         ProviderBedrock,
		DisplayName:  "AWS Bedrock",
		Description:  "Amazon Web Services Bedrock",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey, AuthMethodOAuth},
		DefaultModel: "anthropic.claude-3-sonnet-20240229-v1:0",
		Models: []ModelInfo{
			{
				ID:          "anthropic.claude-3-opus-20240229-v1:0",
				Name:        "Claude 3 Opus",
				Description: "Most capable model on Bedrock",
				ContextSize: 200000,
			},
			{
				ID:          "anthropic.claude-3-sonnet-20240229-v1:0",
				Name:        "Claude 3 Sonnet",
				Description: "Balanced performance",
				ContextSize: 200000,
			},
		},
	}

	w.providers[ProviderOllama] = ProviderInfo{
		Name:         ProviderOllama,
		DisplayName:  "Ollama",
		Description:  "Local models via Ollama",
		AuthMethods:  []AuthMethod{AuthMethodNone},
		DefaultModel: "llama3",
		Models: []ModelInfo{
			{
				ID:          "llama3",
				Name:        "Llama 3",
				Description: "Meta's Llama 3 model",
				ContextSize: 8192,
			},
			{
				ID:          "llama3.1",
				Name:        "Llama 3.1",
				Description: "Updated Llama 3.1 model",
				ContextSize: 128000,
			},
			{
				ID:          "codellama",
				Name:        "Code Llama",
				Description: "Specialized for code",
				ContextSize: 16384,
			},
		},
	}

	w.providers[ProviderLMStudio] = ProviderInfo{
		Name:         ProviderLMStudio,
		DisplayName:  "LM Studio",
		Description:  "Local models via LM Studio",
		AuthMethods:  []AuthMethod{AuthMethodNone, AuthMethodAPIKey},
		DefaultModel: "local-model",
		Models: []ModelInfo{
			{
				ID:          "local-model",
				Name:        "Local Model",
				Description: "Your loaded local model",
				ContextSize: 8192,
			},
		},
	}
}

// Run executes the configuration wizard
func (w *Wizard) Run(ctx context.Context, flags WizardFlags) (*WizardResult, error) {
	if flags.NonInteractive {
		return w.runNonInteractive(ctx, flags)
	}
	return w.runInteractive(ctx, flags)
}

// runInteractive runs the wizard in interactive mode
func (w *Wizard) runInteractive(ctx context.Context, flags WizardFlags) (*WizardResult, error) {
	result := &WizardResult{}

	// Step 1: Provider Selection
	provider, err := w.selectProvider(flags.Provider)
	if err != nil {
		return nil, fmt.Errorf("provider selection failed: %w", err)
	}
	result.Provider = provider

	// Get provider info
	providerInfo, ok := w.providers[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	// Step 2: Auth Method Selection
	authMethod, err := w.selectAuthMethod(providerInfo, flags.AuthMethod)
	if err != nil {
		return nil, fmt.Errorf("auth method selection failed: %w", err)
	}
	result.AuthMethod = authMethod

	// Step 3: Credentials Input
	if authMethod == AuthMethodAPIKey {
		apiKey, err := w.inputAPIKey(provider, flags.APIKey)
		if err != nil {
			return nil, fmt.Errorf("API key input failed: %w", err)
		}
		result.APIKey = apiKey
	}

	// Step 4: Model Selection
	model, err := w.selectModel(providerInfo, flags.Model)
	if err != nil {
		return nil, fmt.Errorf("model selection failed: %w", err)
	}
	result.Model = model

	// Step 5: Validation
	if err := w.validateConfiguration(ctx, result); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	result.Validated = true

	// Step 6: Save Confirmation
	if !flags.Force {
		confirmed, err := w.confirmSave()
		if err != nil {
			return nil, fmt.Errorf("confirmation failed: %w", err)
		}
		if !confirmed {
			return nil, errors.New("configuration cancelled by user")
		}
	}

	// Save the configuration
	if err := w.saveConfiguration(result); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	return result, nil
}

// runNonInteractive runs the wizard in non-interactive mode
func (w *Wizard) runNonInteractive(ctx context.Context, flags WizardFlags) (*WizardResult, error) {
	if flags.Provider == "" {
		return nil, errors.New("provider flag is required in non-interactive mode")
	}

	result := &WizardResult{
		Provider: flags.Provider,
	}

	// Get provider info
	providerInfo, ok := w.providers[flags.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", flags.Provider)
	}

	// Determine auth method
	if flags.AuthMethod != "" {
		result.AuthMethod = AuthMethod(flags.AuthMethod)
	} else {
		// Use first available auth method
		if len(providerInfo.AuthMethods) > 0 {
			result.AuthMethod = providerInfo.AuthMethods[0]
		} else {
			return nil, errors.New("no auth method available for provider")
		}
	}

	// Validate auth method is supported
	supported := false
	for _, method := range providerInfo.AuthMethods {
		if method == result.AuthMethod {
			supported = true
			break
		}
	}
	if !supported {
		return nil, fmt.Errorf("auth method %s not supported for provider %s", result.AuthMethod, flags.Provider)
	}

	// Get API key if needed
	if result.AuthMethod == AuthMethodAPIKey {
		if flags.APIKey == "" {
			return nil, errors.New("api-key flag is required for API key authentication in non-interactive mode")
		}
		result.APIKey = flags.APIKey
	}

	// Set model
	if flags.Model != "" {
		result.Model = flags.Model
	} else {
		result.Model = providerInfo.DefaultModel
	}

	// Validate
	if err := w.validateConfiguration(ctx, result); err != nil {
		return nil, err
	}
	result.Validated = true

	// Save
	if err := w.saveConfiguration(result); err != nil {
		return nil, err
	}

	return result, nil
}

// selectProvider prompts the user to select a provider
func (w *Wizard) selectProvider(preSelected string) (string, error) {
	if preSelected != "" {
		return preSelected, nil
	}

	providerNames := make([]string, 0, len(w.providers))
	for _, info := range w.providers {
		providerNames = append(providerNames, info.DisplayName)
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select a provider:",
		Options: providerNames,
		Description: func(value string, index int) string {
			for _, info := range w.providers {
				if info.DisplayName == value {
					return info.Description
				}
			}
			return ""
		},
	}

	opts := []survey.AskOpt{}
	if w.input != nil && w.output != nil {
		opts = append(opts, survey.WithStdio(w.input, w.output, w.output))
	}

	if err := survey.AskOne(prompt, &selected, opts...); err != nil {
		return "", err
	}

	// Find provider key from display name
	for key, info := range w.providers {
		if info.DisplayName == selected {
			return key, nil
		}
	}

	return "", errors.New("could not find selected provider")
}

// selectAuthMethod prompts the user to select an authentication method
func (w *Wizard) selectAuthMethod(providerInfo ProviderInfo, preSelected string) (AuthMethod, error) {
	if preSelected != "" {
		return AuthMethod(preSelected), nil
	}

	// If only one auth method, use it automatically
	if len(providerInfo.AuthMethods) == 1 {
		return providerInfo.AuthMethods[0], nil
	}

	methodOptions := make([]string, len(providerInfo.AuthMethods))
	for i, method := range providerInfo.AuthMethods {
		switch method {
		case AuthMethodAPIKey:
			methodOptions[i] = "API Key"
		case AuthMethodOAuth:
			methodOptions[i] = "OAuth 2.0"
		case AuthMethodNone:
			methodOptions[i] = "None (Local)"
		}
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select authentication method:",
		Options: methodOptions,
	}

	opts := []survey.AskOpt{}
	if w.input != nil && w.output != nil {
		opts = append(opts, survey.WithStdio(w.input, w.output, w.output))
	}

	if err := survey.AskOne(prompt, &selected, opts...); err != nil {
		return "", err
	}

	switch selected {
	case "API Key":
		return AuthMethodAPIKey, nil
	case "OAuth 2.0":
		return AuthMethodOAuth, nil
	case "None (Local)":
		return AuthMethodNone, nil
	}

	return "", errors.New("invalid auth method selected")
}

// inputAPIKey prompts the user to enter an API key
func (w *Wizard) inputAPIKey(provider, preEntered string) (string, error) {
	if preEntered != "" {
		return preEntered, nil
	}

	var apiKey string
	prompt := &survey.Password{
		Message: "Enter your API key:",
		Help:    "Your API key will be securely stored.",
	}

	opts := []survey.AskOpt{}
	if w.input != nil && w.output != nil {
		opts = append(opts, survey.WithStdio(w.input, w.output, w.output))
	}

	if err := survey.AskOne(prompt, &apiKey, opts...); err != nil {
		return "", err
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("API key cannot be empty")
	}

	return apiKey, nil
}

// selectModel prompts the user to select a model
func (w *Wizard) selectModel(providerInfo ProviderInfo, preSelected string) (string, error) {
	if preSelected != "" {
		return preSelected, nil
	}

	modelOptions := make([]string, len(providerInfo.Models))
	for i, model := range providerInfo.Models {
		modelOptions[i] = model.Name
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select a model:",
		Options: modelOptions,
		Description: func(value string, index int) string {
			for _, model := range providerInfo.Models {
				if model.Name == value {
					return model.Description
				}
			}
			return ""
		},
		Default: providerInfo.DefaultModel,
	}

	opts := []survey.AskOpt{}
	if w.input != nil && w.output != nil {
		opts = append(opts, survey.WithStdio(w.input, w.output, w.output))
	}

	if err := survey.AskOne(prompt, &selected, opts...); err != nil {
		return "", err
	}

	// Find model ID from name
	for _, model := range providerInfo.Models {
		if model.Name == selected {
			return model.ID, nil
		}
	}

	return providerInfo.DefaultModel, nil
}

// confirmSave asks the user to confirm saving the configuration
func (w *Wizard) confirmSave() (bool, error) {
	var confirmed bool
	prompt := &survey.Confirm{
		Message: "Save this configuration?",
		Default: true,
	}

	opts := []survey.AskOpt{}
	if w.input != nil && w.output != nil {
		opts = append(opts, survey.WithStdio(w.input, w.output, w.output))
	}

	if err := survey.AskOne(prompt, &confirmed, opts...); err != nil {
		return false, err
	}

	return confirmed, nil
}

// validateConfiguration validates the wizard result
func (w *Wizard) validateConfiguration(ctx context.Context, result *WizardResult) error {
	// Validate provider
	providerInfo, ok := w.providers[result.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", result.Provider)
	}

	// Validate auth method
	supported := false
	for _, method := range providerInfo.AuthMethods {
		if method == result.AuthMethod {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("auth method %s not supported for provider %s", result.AuthMethod, result.Provider)
	}

	// Validate API key if required
	if result.AuthMethod == AuthMethodAPIKey {
		if result.APIKey == "" {
			return errors.New("API key is required")
		}

		// Validate key format
		if w.apiKeyManager != nil {
			if err := w.apiKeyManager.ValidateKey(result.Provider, result.APIKey); err != nil {
				return fmt.Errorf("invalid API key: %w", err)
			}

			// Test the key
			if err := w.apiKeyManager.TestKey(ctx, result.Provider, result.APIKey); err != nil {
				return fmt.Errorf("API key validation failed: %w", err)
			}
		}
	}

	// Validate model
	if result.Model != "" {
		modelFound := false
		for _, model := range providerInfo.Models {
			if model.ID == result.Model {
				modelFound = true
				break
			}
		}
		if !modelFound {
			return fmt.Errorf("unknown model: %s", result.Model)
		}
	}

	return nil
}

// saveConfiguration saves the wizard result
func (w *Wizard) saveConfiguration(result *WizardResult) error {
	if result.AuthMethod == AuthMethodAPIKey && result.APIKey != "" && w.apiKeyManager != nil {
		if err := w.apiKeyManager.StoreKey(result.Provider, result.APIKey, nil); err != nil {
			return fmt.Errorf("failed to store API key: %w", err)
		}
	}
	return nil
}

// GetProviderInfo returns information about a provider
func (w *Wizard) GetProviderInfo(name string) (ProviderInfo, bool) {
	info, ok := w.providers[name]
	return info, ok
}

// ListProviders returns a list of all available providers
func (w *Wizard) ListProviders() []string {
	names := make([]string, 0, len(w.providers))
	for name := range w.providers {
		names = append(names, name)
	}
	return names
}
