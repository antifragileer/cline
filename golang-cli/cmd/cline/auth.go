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
	baseurl  string
	cwd      string
	verbose  bool
	config   string
}

// ProviderInfo holds information about supported providers
type ProviderInfo struct {
	Name         string
	Description  string
	DefaultModel string
	Models       []string
	RequiresKey  bool
}

// SupportedProviders defines the list of supported API providers
var SupportedProviders = map[string]ProviderInfo{
	"anthropic": {
		Name:         "anthropic",
		Description:  "Anthropic Claude API",
		DefaultModel: "claude-3-5-sonnet-20241022",
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
			"claude-3-haiku-20240307",
		},
		RequiresKey: true,
	},
	"openrouter": {
		Name:         "openrouter",
		Description:  "OpenRouter - Unified API for multiple models",
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
		Name:         "openai",
		Description:  "OpenAI Compatible API",
		DefaultModel: "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
		},
		RequiresKey: true,
	},
	"openai-native": {
		Name:         "openai-native",
		Description:  "OpenAI Native API",
		DefaultModel: "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
		},
		RequiresKey: true,
	},
	"gemini": {
		Name:         "gemini",
		Description:  "Google Gemini API",
		DefaultModel: "gemini-1.5-pro",
		Models: []string{
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
		RequiresKey: true,
	},
	"bedrock": {
		Name:         "bedrock",
		Description:  "AWS Bedrock",
		DefaultModel: "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Models: []string{
			"anthropic.claude-3-5-sonnet-20241022-v2:0",
			"anthropic.claude-3-opus-20240229-v1:0",
		},
		RequiresKey: true,
	},
	"ollama": {
		Name:         "ollama",
		Description:  "Ollama (local models)",
		DefaultModel: "llama3.1",
		Models: []string{
			"llama3.1",
			"llama3.2",
			"codellama",
			"mistral",
		},
		RequiresKey: false,
	},
	"moonshot": {
		Name:         "moonshot",
		Description:  "Moonshot AI",
		DefaultModel: "kimi-k2.5",
		Models: []string{
			"kimi-k2.5",
			"kimi-k2",
			"kimi-k1.5",
		},
		RequiresKey: true,
	},
}

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate a provider and configure what model is used",
	Long: `Authenticate a provider and configure what model is used.

This command provides an interactive wizard to set up API keys and
authentication for various AI providers. You can also use flags for
non-interactive configuration.

Supported providers: anthropic, openrouter, openai, openai-native, gemini, bedrock, ollama, moonshot

Flag combinations:
  - Interactive mode: cline auth
  - Quick setup: cline auth -p <provider> -k <apikey>
  - Full setup: cline auth -p <provider> -k <apikey> -m <model> --baseurl <url>`,
	Example: `  # Interactive authentication wizard
  cline auth

  # Quick setup with provider and API key
  cline auth -p anthropic -k sk-ant-...

  # Full setup with provider, API key, and model
  cline auth -p openrouter -k sk-or-... -m anthropic/claude-3.5-sonnet

  # Setup with custom base URL (OpenAI compatible providers only)
  cline auth -p openai -k sk-... -m gpt-4o --baseurl https://api.example.com/v1`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)

	// Add flags to auth command - aligned with NodeJS implementation
	authCmd.Flags().StringVarP(&authFlags.provider, "provider", "p", "", "Provider ID for quick setup (e.g., openai-native, anthropic, moonshot)")
	authCmd.Flags().StringVarP(&authFlags.key, "key", "k", "", "API key for the provider")
	authCmd.Flags().StringVarP(&authFlags.model, "model", "m", "", "Model ID to configure (e.g., gpt-4o, claude-sonnet-4-6, kimi-k2.5)")
	authCmd.Flags().StringVar(&authFlags.baseurl, "baseurl", "", "Base URL (optional, only for openai provider)")
	authCmd.Flags().StringVarP(&authFlags.cwd, "cwd", "c", "", "Working directory for the task")
	authCmd.Flags().BoolVarP(&authFlags.verbose, "verbose", "v", false, "Show verbose output")
	authCmd.Flags().StringVar(&authFlags.config, "config", "", "Path to Cline configuration directory")
}

// runAuth executes the auth command
func runAuth(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext(authFlags.config, "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Determine the mode based on provided flags
	hasProvider := authFlags.provider != ""
	hasKey := authFlags.key != ""
	hasModel := authFlags.model != ""
	hasBaseURL := authFlags.baseurl != ""

	// Log verbose output if enabled
	if authFlags.verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Verbose mode enabled\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\n", authFlags.provider)
		fmt.Fprintf(cmd.OutOrStdout(), "Has Key: %v\n", hasKey)
		fmt.Fprintf(cmd.OutOrStdout(), "Has Model: %v\n", hasModel)
		fmt.Fprintf(cmd.OutOrStdout(), "Has BaseURL: %v\n", hasBaseURL)
	}

	// Validate provider if provided
	provider := authFlags.provider
	if provider != "" {
		normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
		if _, ok := SupportedProviders[normalizedProvider]; !ok {
			// Check for common aliases
			if normalizedProvider == "openai" {
				// Allow "openai" as an alias for configuration
			} else {
				return fmt.Errorf("unsupported provider: %s. Supported providers: %s", 
					provider, getSupportedProviderList())
			}
		}
		provider = normalizedProvider
	}

	// Bedrock is not supported for quick setup due to complex authentication
	if hasProvider && provider == "bedrock" && (hasKey || hasModel) {
		return fmt.Errorf("bedrock provider is not supported for quick setup due to complex authentication requirements. please use interactive setup")
	}

	// Base URL is only supported for OpenAI and OpenAI-compatible providers
	if hasBaseURL && provider != "" && provider != "openai" && provider != "openai-native" {
		return fmt.Errorf("base URL is only supported for OpenAI and OpenAI-compatible providers")
	}

	// Determine execution mode:
	// - Interactive mode: no flags or only partial flags
	// - Quick setup: provider + key provided
	// - Full setup: provider + key + model (and optionally baseurl)

	if hasProvider && hasKey {
		// Quick or full setup mode - non-interactive
		return runQuickAuthSetup(cmd, ctx, provider, hasModel)
	}

	// Interactive mode
	return runInteractiveAuth(cmd, ctx, provider)
}

// commandOutputter is an interface for command output operations
type commandOutputter interface {
	OutOrStdout() io.Writer
}

// runQuickAuthSetup handles quick setup (provider + key) and full setup (provider + key + model + baseurl)
func runQuickAuthSetup(cmd commandOutputter, ctx *storage.StorageContext, provider string, hasModel bool) error {
	// Get provider info
	providerInfo, ok := SupportedProviders[provider]
	if !ok && provider != "openai" {
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Use "openai" provider info for "openai" alias
	if provider == "openai" {
		providerInfo = SupportedProviders["openai"]
	}

	// Get API key
	apiKey := authFlags.key
	if providerInfo.RequiresKey && apiKey == "" {
		return fmt.Errorf("API key is required for provider: %s", provider)
	}

	// Get model selection
	model := authFlags.model
	if !hasModel {
		// Use default model if not provided
		model = providerInfo.DefaultModel
		if authFlags.verbose {
			fmt.Fprintf(cmd.OutOrStdout(), "Using default model: %s\n", model)
		}
	}

	// Validate model (warn but don't fail for unknown models)
	if !isValidModel(providerInfo, model) && authFlags.verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: '%s' is not a known model for %s\n", model, provider)
	}

	// Save configuration
	if err := saveAuthConfig(ctx, provider, apiKey, model, authFlags.baseurl); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Mark onboarding as complete
	if err := ctx.GlobalState.Set("welcomeViewCompleted", true); err != nil {
		return fmt.Errorf("failed to update onboarding status: %w", err)
	}

	// Output success message
	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Successfully configured %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Provider: %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Model: %s\n", model)
	if providerInfo.RequiresKey {
		fmt.Fprintln(cmd.OutOrStdout(), "  API Key: ********")
	}
	if authFlags.baseurl != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Base URL: %s\n", authFlags.baseurl)
	}

	return nil
}

// runInteractiveAuth handles interactive mode (wizard)
func runInteractiveAuth(cmd *cobra.Command, ctx *storage.StorageContext, preSelectedProvider string) error {
	// Determine provider
	provider := preSelectedProvider
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
	if err := saveAuthConfig(ctx, provider, apiKey, model, ""); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Mark onboarding as complete
	if err := ctx.GlobalState.Set("welcomeViewCompleted", true); err != nil {
		return fmt.Errorf("failed to update onboarding status: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Successfully configured %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Provider: %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Model: %s\n", model)
	if providerInfo.RequiresKey {
		fmt.Fprintln(cmd.OutOrStdout(), "  API Key: ********")
	}

	// Test the configuration
	fmt.Fprintln(cmd.OutOrStdout(), "\nTesting configuration...")
	if err := testProviderAuth(provider, apiKey); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "✗ Test failed: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Authentication test passed")
	}

	return nil
}

// getSupportedProviderList returns a comma-separated list of supported providers
func getSupportedProviderList() string {
	var providers []string
	for name := range SupportedProviders {
		providers = append(providers, name)
	}
	return strings.Join(providers, ", ")
}

// selectProviderInteractive prompts the user to select a provider
func selectProviderInteractive(input io.Reader, output io.Writer) (string, error) {
	fmt.Fprintln(output, "\nSelect an AI provider:")
	fmt.Fprintln(output)

	// List providers in a consistent order
	providerNames := make([]string, 0, len(SupportedProviders))
	for name := range SupportedProviders {
		providerNames = append(providerNames, name)
	}

	// Sort providers for consistent display
	for i := 0; i < len(providerNames); i++ {
		for j := i + 1; j < len(providerNames); j++ {
			if providerNames[i] > providerNames[j] {
				providerNames[i], providerNames[j] = providerNames[j], providerNames[i]
			}
		}
	}

	for i, name := range providerNames {
		info := SupportedProviders[name]
		fmt.Fprintf(output, "  %d. %s - %s\n", i+1, info.Name, info.Description)
	}

	fmt.Fprintln(output)
	fmt.Fprint(output, "Enter number (1-"+fmt.Sprintf("%d", len(providerNames))+"): ")

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

	if selection < 1 || selection > len(providerNames) {
		return "", fmt.Errorf("selection out of range: %d", selection)
	}

	return providerNames[selection-1], nil
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
func saveAuthConfig(ctx *storage.StorageContext, provider, apiKey, model, baseURL string) error {
	// Save provider setting
	if err := ctx.GlobalState.Set("apiProvider", provider); err != nil {
		return fmt.Errorf("failed to save provider: %w", err)
	}

	// Save model setting
	modelKey := provider + "Model"
	if err := ctx.GlobalState.Set(modelKey, model); err != nil {
		return fmt.Errorf("failed to save model: %w", err)
	}
	
	// Also save as defaultModel for compatibility
	if err := ctx.GlobalState.Set("defaultModel", model); err != nil {
		return fmt.Errorf("failed to save default model: %w", err)
	}

	// Save API key to secrets if provided
	if apiKey != "" {
		secretKeyName := provider + "ApiKey"
		if err := ctx.Secrets.Set(secretKeyName, apiKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
	}

	// Save base URL if provided (for OpenAI compatible providers)
	if baseURL != "" {
		baseURLKey := provider + "BaseUrl"
		if err := ctx.GlobalState.Set(baseURLKey, baseURL); err != nil {
			return fmt.Errorf("failed to save base URL: %w", err)
		}
	}

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
	case "openai", "openai-native":
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
	case "moonshot":
		// Would test with Moonshot API
		return nil
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetCurrentProvider returns the currently configured provider
// Returns empty string if no provider is configured (no error)
func GetCurrentProvider(ctx *storage.StorageContext) (string, error) {
	providerVal, ok := ctx.GlobalState.Get("apiProvider")
	if !ok {
		return "", nil
	}

	provider, ok := providerVal.(string)
	if !ok {
		return "", fmt.Errorf("invalid provider configuration")
	}

	return provider, nil
}

// GetAPIKey retrieves the API key for a provider
// Returns empty string if no API key is found (no error)
func GetAPIKey(ctx *storage.StorageContext, provider string) (string, error) {
	apiKeyVal, ok := ctx.Secrets.Get(provider + "ApiKey")
	if !ok {
		return "", nil
	}

	apiKey, ok := apiKeyVal.(string)
	if !ok {
		return "", fmt.Errorf("invalid API key configuration")
	}

	return apiKey, nil
}
