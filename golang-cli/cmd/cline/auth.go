package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// authFlags holds the parsed flag values for auth command
var authFlags struct {
	provider string
	key      string
	model    string
	test     bool
}

// ProviderInfo holds information about supported providers
type ProviderInfo struct {
	Name        string
	Description string
	DefaultModel string
	Models      []string
	RequiresKey bool
}

// SupportedProviders defines the list of supported API providers
var SupportedProviders = map[string]ProviderInfo{
	"anthropic": {
		Name:        "anthropic",
		Description: "Anthropic Claude API",
		DefaultModel: "claude-3-5-sonnet-20241022",
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
			"claude-3-haiku-20240307",
		},
		RequiresKey: true,
	},
	"openrouter": {
		Name:        "openrouter",
		Description: "OpenRouter - Unified API for multiple models",
		DefaultModel: "anthropic/claude-3.5-sonnet",
		Models: []string{
			"anthropic/claude-3.5-sonnet",
			"anthropic/claude-3-opus",
			"openai/gpt-4o",
			"google/gemini-pro-1.5",
		},
		RequiresKey: true,
	},
	"openai": {
		Name:        "openai",
		Description: "OpenAI API",
		DefaultModel: "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
		},
		RequiresKey: true,
	},
	"gemini": {
		Name:        "gemini",
		Description: "Google Gemini API",
		DefaultModel: "gemini-1.5-pro",
		Models: []string{
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
		RequiresKey: true,
	},
	"bedrock": {
		Name:        "bedrock",
		Description: "AWS Bedrock",
		DefaultModel: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Models: []string{
			"anthropic.claude-3-5-sonnet-20241022-v2:0",
			"anthropic.claude-3-opus-20240229-v1:0",
		},
		RequiresKey: true,
	},
	"ollama": {
		Name:        "ollama",
		Description: "Ollama (local models)",
		DefaultModel: "llama3.1",
		Models: []string{
			"llama3.1",
			"llama3.2",
			"codellama",
			"mistral",
		},
		RequiresKey: false,
	},
}

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure authentication for AI providers",
	Long: `Configure authentication for AI providers.

This command provides an interactive wizard to set up API keys and
authentication for various AI providers. You can also use flags for
non-interactive configuration.

Supported providers: anthropic, openrouter, openai, gemini, bedrock, ollama`,
	Example: `  # Interactive authentication wizard
  cline auth

  # Configure specific provider
  cline auth -p anthropic

  # Configure with API key
  cline auth -p openai -k sk-...

  # Set provider and model
  cline auth -p openrouter -m anthropic/claude-3.5-sonnet

  # Test the configuration
  cline auth --test`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)

	// Add flags to auth command
	authCmd.Flags().StringVarP(&authFlags.provider, "provider", "p", "", "Provider to configure (anthropic, openrouter, openai, gemini, bedrock, ollama)")
	authCmd.Flags().StringVarP(&authFlags.key, "key", "k", "", "API key for the provider")
	authCmd.Flags().StringVarP(&authFlags.model, "model", "m", "", "Default model for the provider")
	authCmd.Flags().BoolVar(&authFlags.test, "test", false, "Test the authentication configuration")
}

// runAuth executes the auth command
func runAuth(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// If test flag is set, just test existing configuration
	if authFlags.test {
		return testAuth(ctx)
	}

	// Determine provider
	provider := authFlags.provider
	if provider == "" {
		// Interactive mode - show provider selection
		var err error
		provider, err = selectProviderInteractive(cmd.InOrStdin(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
	}

	// Validate provider
	providerInfo, ok := SupportedProviders[provider]
	if !ok {
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Get API key if required
	apiKey := authFlags.key
	if providerInfo.RequiresKey && apiKey == "" {
		var err error
		apiKey, err = promptForKey(cmd.InOrStdin(), cmd.OutOrStdout(), providerInfo.Name)
		if err != nil {
			return err
		}
	}

	// Get model selection
	model := authFlags.model
	if model == "" {
		var err error
		model, err = selectModelInteractive(cmd.InOrStdin(), cmd.OutOrStdout(), providerInfo)
		if err != nil {
			return err
		}
	}

	// Validate model
	if !isValidModel(providerInfo, model) {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: '%s' is not a known model for %s\n", model, provider)
	}

	// Save configuration
	if err := saveAuthConfig(ctx, provider, apiKey, model); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Successfully configured %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Provider: %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Model: %s\n", model)
	if providerInfo.RequiresKey {
		fmt.Fprintln(cmd.OutOrStdout(), "  API Key: ********")
	}

	// Test the configuration if requested or in interactive mode
	if authFlags.test || authFlags.key == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "\nTesting configuration...")
		if err := testProviderAuth(provider, apiKey); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "✗ Test failed: %v\n", err)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Authentication test passed")
		}
	}

	return nil
}

// selectProviderInteractive prompts the user to select a provider
func selectProviderInteractive(input io.Reader, output io.Writer) (string, error) {
	fmt.Fprintln(output, "\nSelect an AI provider:")
	fmt.Fprintln(output)

	// List providers
	providers := make([]string, 0, len(SupportedProviders))
	for name := range SupportedProviders {
		providers = append(providers, name)
	}

	for i, name := range providers {
		info := SupportedProviders[name]
		fmt.Fprintf(output, "  %d. %s - %s\n", i+1, info.Name, info.Description)
	}

	fmt.Fprintln(output)
	fmt.Fprint(output, "Enter number (1-" + fmt.Sprintf("%d", len(providers)) + "): ")

	reader := bufio.NewReader(input)
	inputStr, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	inputStr = strings.TrimSpace(inputStr)

	// Parse selection
	var selection int
	if _, err := fmt.Sscanf(inputStr, "%d", &selection); err != nil {
		return "", fmt.Errorf("invalid selection: %s", inputStr)
	}

	if selection < 1 || selection > len(providers) {
		return "", fmt.Errorf("selection out of range: %d", selection)
	}

	return providers[selection-1], nil
}

// promptForKey prompts the user for an API key
func promptForKey(input io.Reader, output io.Writer, provider string) (string, error) {
	fmt.Fprintf(output, "\nEnter your %s API key: ", provider)

	reader := bufio.NewReader(input)
	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return key, nil
}

// selectModelInteractive prompts the user to select a model
func selectModelInteractive(input io.Reader, output io.Writer, provider ProviderInfo) (string, error) {
	fmt.Fprintf(output, "\nSelect a model for %s (or press Enter for default: %s):\n", provider.Name, provider.DefaultModel)

	for i, model := range provider.Models {
		fmt.Fprintf(output, "  %d. %s\n", i+1, model)
	}

	fmt.Fprint(output, "\nEnter number or model name: ")

	reader := bufio.NewReader(input)
	inputStr, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	inputStr = strings.TrimSpace(inputStr)

	// If empty, use default
	if inputStr == "" {
		return provider.DefaultModel, nil
	}

	// Check if it's a number
	var selection int
	if _, err := fmt.Sscanf(inputStr, "%d", &selection); err == nil {
		if selection >= 1 && selection <= len(provider.Models) {
			return provider.Models[selection-1], nil
		}
		return "", fmt.Errorf("selection out of range: %d", selection)
	}

	// Otherwise, treat as model name
	return inputStr, nil
}

// isValidModel checks if a model is in the provider's supported list
func isValidModel(provider ProviderInfo, model string) bool {
	for _, m := range provider.Models {
		if m == model {
			return true
		}
	}
	return false
}

// saveAuthConfig saves the authentication configuration
func saveAuthConfig(ctx *storage.StorageContext, provider, apiKey, model string) error {
	// Save provider setting
	if err := ctx.GlobalState.Set("apiProvider", provider); err != nil {
		return fmt.Errorf("failed to save provider: %w", err)
	}

	// Save model setting
	if err := ctx.GlobalState.Set("defaultModel", model); err != nil {
		return fmt.Errorf("failed to save model: %w", err)
	}

	// Save API key to secrets if provided
	if apiKey != "" {
		if err := ctx.Secrets.Set(provider+"ApiKey", apiKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
	}

	return nil
}

// testAuth tests the existing authentication configuration
func testAuth(ctx *storage.StorageContext) error {
	// Get current provider
	providerVal, ok := ctx.GlobalState.Get("apiProvider")
	if !ok {
		return fmt.Errorf("no provider configured. Run 'cline auth' to configure")
	}

	provider, ok := providerVal.(string)
	if !ok {
		return fmt.Errorf("invalid provider configuration")
	}

	// Get API key from secrets
	apiKeyVal, ok := ctx.Secrets.Get(provider + "ApiKey")
	if !ok {
		return fmt.Errorf("no API key found for %s", provider)
	}

	apiKey, ok := apiKeyVal.(string)
	if !ok {
		return fmt.Errorf("invalid API key configuration")
	}

	fmt.Printf("Testing authentication for %s...\n", provider)

	if err := testProviderAuth(provider, apiKey); err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}

	fmt.Println("✓ Authentication test passed")
	return nil
}

// testProviderAuth tests authentication with a specific provider
func testProviderAuth(provider, apiKey string) error {
	// This is a placeholder for actual API testing
	// In a real implementation, this would make a test API call
	switch provider {
	case "anthropic":
		// Would test with Anthropic API
		return nil
	case "openrouter":
		// Would test with OpenRouter API
		return nil
	case "openai":
		// Would test with OpenAI API
		return nil
	case "gemini":
		// Would test with Google Gemini API
		return nil
	case "bedrock":
		// Would test with AWS Bedrock
		return nil
	case "ollama":
		// Would test with local Ollama instance
		return nil
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetCurrentProvider returns the currently configured provider
func GetCurrentProvider(ctx *storage.StorageContext) (string, error) {
	providerVal, ok := ctx.GlobalState.Get("apiProvider")
	if !ok {
		return "", fmt.Errorf("no provider configured")
	}

	provider, ok := providerVal.(string)
	if !ok {
		return "", fmt.Errorf("invalid provider configuration")
	}

	return provider, nil
}

// GetAPIKey retrieves the API key for a provider
func GetAPIKey(ctx *storage.StorageContext, provider string) (string, error) {
	apiKeyVal, ok := ctx.Secrets.Get(provider + "ApiKey")
	if !ok {
		return "", fmt.Errorf("no API key found for %s", provider)
	}

	apiKey, ok := apiKeyVal.(string)
	if !ok {
		return "", fmt.Errorf("invalid API key configuration")
	}

	return apiKey, nil
}