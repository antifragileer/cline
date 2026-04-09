// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// SettingsPersistence handles loading and saving settings to storage
type SettingsPersistence struct {
	storageCtx *storage.StorageContext
}

// NewSettingsPersistence creates a new settings persistence handler
func NewSettingsPersistence(storageCtx *storage.StorageContext) *SettingsPersistence {
	return &SettingsPersistence{
		storageCtx: storageCtx,
	}
}

// LoadSettings loads all settings from storage into the content
func (sp *SettingsPersistence) LoadSettings(content *SettingsContent) error {
	if sp.storageCtx == nil {
		return fmt.Errorf("storage context is nil")
	}

	// Load API settings
	sp.loadAPISettings(content)

	// Load auto-approve settings
	sp.loadAutoApproveSettings(content)

	// Load feature settings
	sp.loadFeatureSettings(content)

	// Load account settings
	sp.loadAccountSettings(content)

	// Load other settings
	sp.loadOtherSettings(content)

	return nil
}

// loadAPISettings loads API provider and model settings
func (sp *SettingsPersistence) loadAPISettings(content *SettingsContent) {
	// Get current provider
	if provider, ok := sp.getGlobalStateString("apiProvider"); ok && provider != "" {
		sp.setSettingValue(content, "provider", provider)
	}

	// Get current model
	if model, ok := sp.getGlobalStateString("openAiModelId"); ok && model != "" {
		sp.setSettingValue(content, "model", model)
	} else if model, ok := sp.getGlobalStateString("anthropicModelId"); ok && model != "" {
		sp.setSettingValue(content, "model", model)
	} else if model, ok := sp.getGlobalStateString("clineModelId"); ok && model != "" {
		sp.setSettingValue(content, "model", model)
	}

	// Get API key status (don't load actual key, just indicate if set)
	if apiKey, ok := sp.getSecretString("apiKey"); ok && apiKey != "" {
		sp.setSettingValue(content, "apiKey", "[set]")
	}

	// Get separate models setting
	if separate, ok := sp.getGlobalStateBool("separateModelsForPlanAct"); ok {
		sp.setSettingValue(content, "separateModels", separate)
	}

	// Get plan model
	if planModel, ok := sp.getGlobalStateString("planModelId"); ok && planModel != "" {
		sp.setSettingValue(content, "planModel", planModel)
	}
}

// loadAutoApproveSettings loads auto-approval settings
func (sp *SettingsPersistence) loadAutoApproveSettings(content *SettingsContent) {
	// YOLO mode
	if yolo, ok := sp.getGlobalStateBool("yoloMode"); ok {
		sp.setSettingValue(content, "yoloMode", yolo)
	}

	// Auto-approve read files
	if auto, ok := sp.getGlobalStateBool("autoApproveReadFiles"); ok {
		sp.setSettingValue(content, "autoApproveReadFiles", auto)
	}

	// Auto-approve edit files
	if auto, ok := sp.getGlobalStateBool("autoApproveEditFiles"); ok {
		sp.setSettingValue(content, "autoApproveEditFiles", auto)
	}

	// Auto-approve run commands
	if auto, ok := sp.getGlobalStateBool("autoApproveRunCommands"); ok {
		sp.setSettingValue(content, "autoApproveRunCommands", auto)
	}

	// Auto-approve MCP tools
	if auto, ok := sp.getGlobalStateBool("autoApproveMcpTools"); ok {
		sp.setSettingValue(content, "autoApproveMcpTools", auto)
	}
}

// loadFeatureSettings loads feature toggle settings
func (sp *SettingsPersistence) loadFeatureSettings(content *SettingsContent) {
	// Subagents
	if enabled, ok := sp.getGlobalStateBool("subagentsEnabled"); ok {
		sp.setSettingValue(content, "subagents", enabled)
	}

	// Auto-condense
	if enabled, ok := sp.getGlobalStateBool("autoCondenseEnabled"); ok {
		sp.setSettingValue(content, "autoCondense", enabled)
	}

	// Web tools
	if enabled, ok := sp.getGlobalStateBool("webToolsEnabled"); ok {
		sp.setSettingValue(content, "webTools", enabled)
	}

	// Strict plan mode
	if enabled, ok := sp.getGlobalStateBool("strictPlanMode"); ok {
		sp.setSettingValue(content, "strictPlanMode", enabled)
	}

	// Native tool call
	if enabled, ok := sp.getGlobalStateBool("nativeToolCallEnabled"); ok {
		sp.setSettingValue(content, "nativeToolCall", enabled)
	}
}

// loadAccountSettings loads account settings
func (sp *SettingsPersistence) loadAccountSettings(content *SettingsContent) {
	// Account status
	if accountStatus, ok := sp.getGlobalStateString("accountStatus"); ok && accountStatus != "" {
		sp.setSettingValue(content, "accountStatus", accountStatus)
	}

	// Balance
	if balance, ok := sp.getGlobalStateFloat("accountBalance"); ok {
		sp.setSettingValue(content, "balance", fmt.Sprintf("$%.2f", balance))
	}
}

// loadOtherSettings loads other/miscellaneous settings
func (sp *SettingsPersistence) loadOtherSettings(content *SettingsContent) {
	// Language
	if lang, ok := sp.getGlobalStateString("language"); ok && lang != "" {
		sp.setSettingValue(content, "language", lang)
	}

	// Telemetry
	if enabled, ok := sp.getGlobalStateBool("telemetryEnabled"); ok {
		sp.setSettingValue(content, "telemetry", enabled)
	}

	// Debug mode
	if enabled, ok := sp.getGlobalStateBool("debugMode"); ok {
		sp.setSettingValue(content, "debug", enabled)
	}
}

// SaveSettings saves all settings from content to storage
func (sp *SettingsPersistence) SaveSettings(content *SettingsContent) error {
	if sp.storageCtx == nil {
		return fmt.Errorf("storage context is nil")
	}

	// Save API settings
	sp.saveAPISettings(content)

	// Save auto-approve settings
	sp.saveAutoApproveSettings(content)

	// Save feature settings
	sp.saveFeatureSettings(content)

	// Save other settings
	sp.saveOtherSettings(content)

	return nil
}

// saveAPISettings saves API settings to storage
func (sp *SettingsPersistence) saveAPISettings(content *SettingsContent) {
	// Save provider
	if provider, ok := sp.getSettingValue(content, "provider"); ok {
		if providerStr, ok := provider.(string); ok {
			sp.setGlobalState("apiProvider", providerStr)
		}
	}

	// Save model (based on provider)
	if model, ok := sp.getSettingValue(content, "model"); ok {
		if modelStr, ok := model.(string); ok {
			provider, _ := sp.getSettingValue(content, "provider")
			providerStr, _ := provider.(string)

			switch providerStr {
			case "openai":
				sp.setGlobalState("openAiModelId", modelStr)
			case "anthropic":
				sp.setGlobalState("anthropicModelId", modelStr)
			case "cline":
				sp.setGlobalState("clineModelId", modelStr)
			default:
				sp.setGlobalState("openAiModelId", modelStr)
			}
		}
	}

	// Save separate models setting
	if separate, ok := sp.getSettingValue(content, "separateModels"); ok {
		if separateBool, ok := separate.(bool); ok {
			sp.setGlobalState("separateModelsForPlanAct", separateBool)
		}
	}

	// Save plan model
	if planModel, ok := sp.getSettingValue(content, "planModel"); ok {
		if planModelStr, ok := planModel.(string); ok {
			sp.setGlobalState("planModelId", planModelStr)
		}
	}
}

// saveAutoApproveSettings saves auto-approval settings
func (sp *SettingsPersistence) saveAutoApproveSettings(content *SettingsContent) {
	if yolo, ok := sp.getSettingValue(content, "yoloMode"); ok {
		if yoloBool, ok := yolo.(bool); ok {
			sp.setGlobalState("yoloMode", yoloBool)
		}
	}

	if auto, ok := sp.getSettingValue(content, "autoApproveReadFiles"); ok {
		if autoBool, ok := auto.(bool); ok {
			sp.setGlobalState("autoApproveReadFiles", autoBool)
		}
	}

	if auto, ok := sp.getSettingValue(content, "autoApproveEditFiles"); ok {
		if autoBool, ok := auto.(bool); ok {
			sp.setGlobalState("autoApproveEditFiles", autoBool)
		}
	}

	if auto, ok := sp.getSettingValue(content, "autoApproveRunCommands"); ok {
		if autoBool, ok := auto.(bool); ok {
			sp.setGlobalState("autoApproveRunCommands", autoBool)
		}
	}

	if auto, ok := sp.getSettingValue(content, "autoApproveMcpTools"); ok {
		if autoBool, ok := auto.(bool); ok {
			sp.setGlobalState("autoApproveMcpTools", autoBool)
		}
	}
}

// saveFeatureSettings saves feature settings
func (sp *SettingsPersistence) saveFeatureSettings(content *SettingsContent) {
	if enabled, ok := sp.getSettingValue(content, "subagents"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("subagentsEnabled", enabledBool)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "autoCondense"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("autoCondenseEnabled", enabledBool)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "webTools"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("webToolsEnabled", enabledBool)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "strictPlanMode"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("strictPlanMode", enabledBool)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "nativeToolCall"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("nativeToolCallEnabled", enabledBool)
		}
	}
}

// saveOtherSettings saves other settings
func (sp *SettingsPersistence) saveOtherSettings(content *SettingsContent) {
	if lang, ok := sp.getSettingValue(content, "language"); ok {
		if langStr, ok := lang.(string); ok {
			sp.setGlobalState("language", langStr)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "telemetry"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("telemetryEnabled", enabledBool)
		}
	}

	if enabled, ok := sp.getSettingValue(content, "debug"); ok {
		if enabledBool, ok := enabled.(bool); ok {
			sp.setGlobalState("debugMode", enabledBool)
		}
	}
}

// Helper methods for getting/setting values

func (sp *SettingsPersistence) getGlobalStateString(key string) (string, bool) {
	if sp.storageCtx == nil {
		return "", false
	}
	val, ok := sp.storageCtx.GlobalState.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

func (sp *SettingsPersistence) getGlobalStateBool(key string) (bool, bool) {
	if sp.storageCtx == nil {
		return false, false
	}
	val, ok := sp.storageCtx.GlobalState.Get(key)
	if !ok {
		return false, false
	}

	switch v := val.(type) {
	case bool:
		return v, true
	case string:
		return v == "true" || v == "yes" || v == "1", true
	case int:
		return v != 0, true
	}
	return false, false
}

func (sp *SettingsPersistence) getGlobalStateFloat(key string) (float64, bool) {
	if sp.storageCtx == nil {
		return 0, false
	}
	val, ok := sp.storageCtx.GlobalState.Get(key)
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	}
	return 0, false
}

func (sp *SettingsPersistence) getSecretString(key string) (string, bool) {
	if sp.storageCtx == nil {
		return "", false
	}
	val, ok := sp.storageCtx.Secrets.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

func (sp *SettingsPersistence) setGlobalState(key string, value interface{}) {
	if sp.storageCtx == nil {
		return
	}
	sp.storageCtx.GlobalState.Set(key, value)
}

func (sp *SettingsPersistence) setSecret(key string, value string) {
	if sp.storageCtx == nil {
		return
	}
	sp.storageCtx.Secrets.Set(key, value)
}

func (sp *SettingsPersistence) setSettingValue(content *SettingsContent, key string, value interface{}) {
	// Search through all sections in all tabs
	allSections := [][]SettingsSection{
		content.API,
		content.AutoApprove,
		content.Features,
		content.Account,
		content.Other,
	}

	for _, sections := range allSections {
		for i := range sections {
			for j := range sections[i].Items {
				if sections[i].Items[j].Key == key {
					sections[i].Items[j].Value = value
					return
				}
			}
		}
	}
}

func (sp *SettingsPersistence) getSettingValue(content *SettingsContent, key string) (interface{}, bool) {
	// Search through all sections in all tabs
	allSections := [][]SettingsSection{
		content.API,
		content.AutoApprove,
		content.Features,
		content.Account,
		content.Other,
	}

	for _, sections := range allSections {
		for _, section := range sections {
			for _, item := range section.Items {
				if item.Key == key {
					return item.Value, true
				}
			}
		}
	}
	return nil, false
}

// GetAvailableProviders returns the list of available API providers
func (sp *SettingsPersistence) GetAvailableProviders() []string {
	return []string{
		"cline",
		"openai",
		"anthropic",
		"openrouter",
		"bedrock",
		"ollama",
		"lmstudio",
		"gemini",
		"azure",
		"cerebras",
	}
}

// GetAvailableModels returns available models for a provider
func (sp *SettingsPersistence) GetAvailableModels(provider string) []string {
	// Return default models for each provider
	switch provider {
	case "anthropic":
		return []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-3-5-sonnet-20241022"}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}
	case "cline":
		return []string{"claude-sonnet-4", "claude-opus-4"}
	case "openrouter":
		return []string{"anthropic/claude-sonnet-4", "openai/gpt-4o", "meta-llama/llama-3.3-70b-instruct"}
	case "gemini":
		return []string{"gemini-2.5-pro-preview-03-25", "gemini-2.0-flash-001"}
	case "ollama":
		return []string{"llama3.3", "codellama", "mistral"}
	case "bedrock":
		return []string{"anthropic.claude-sonnet-4-20250514-v1:0", "anthropic.claude-opus-4-20250514-v1:0"}
	case "azure":
		return []string{"gpt-4o", "gpt-4-turbo"}
	case "cerebras":
		return []string{"llama-3.3-70b", "llama-4-scout-17b-16e"}
	default:
		return []string{"default"}
	}
}

// ValidateModel validates if a model is valid for a provider
func (sp *SettingsPersistence) ValidateModel(provider, model string) bool {
	models := sp.GetAvailableModels(provider)
	for _, m := range models {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}

// SaveAPIKey saves the API key for the current provider
func (sp *SettingsPersistence) SaveAPIKey(provider, apiKey string) error {
	if sp.storageCtx == nil {
		return fmt.Errorf("storage context is nil")
	}

	// Store API key with provider-specific key
	keyName := fmt.Sprintf("%sApiKey", provider)
	sp.setSecret(keyName, apiKey)

	// Also store as generic apiKey for backward compatibility
	sp.setSecret("apiKey", apiKey)

	return nil
}

// ExportSettings exports settings to JSON format
func (sp *SettingsPersistence) ExportSettings(content *SettingsContent) (string, error) {
	settings := make(map[string]interface{})

	// Collect all settings
	allSections := [][]SettingsSection{
		content.API,
		content.AutoApprove,
		content.Features,
		content.Account,
		content.Other,
	}

	for _, sections := range allSections {
		for _, section := range sections {
			for _, item := range section.Items {
				// Skip sensitive data
				if item.Type == SettingsItemTypePassword {
					continue
				}
				settings[item.Key] = item.Value
			}
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal settings: %w", err)
	}

	return string(data), nil
}

// ImportSettings imports settings from JSON format
func (sp *SettingsPersistence) ImportSettings(content *SettingsContent, jsonData string) error {
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	// Apply imported settings
	for key, value := range settings {
		sp.setSettingValue(content, key, value)
	}

	return nil
}
