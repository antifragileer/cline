// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"regexp"
	"strings"
)

// SettingsValidator provides validation for settings values
type SettingsValidator struct {
	errors   map[string]string
	rules    map[string][]ValidationRule
	highlighter *SyntaxHighlighter
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name     string
	Validate func(value interface{}) (bool, string)
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	Valid   bool
	Errors  map[string]string
	Field   string
	Message string
}

// NewSettingsValidator creates a new settings validator
func NewSettingsValidator() *SettingsValidator {
	v := &SettingsValidator{
		errors:   make(map[string]string),
		rules:    make(map[string][]ValidationRule),
		highlighter: NewSyntaxHighlighter(true, "dark"),
	}
	v.setupDefaultRules()
	return v
}

// setupDefaultRules sets up default validation rules
func (v *SettingsValidator) setupDefaultRules() {
	// API Provider validation
	v.rules["provider"] = []ValidationRule{
		{
			Name: "required",
			Validate: func(value interface{}) (bool, string) {
				if str, ok := value.(string); ok && str != "" {
					return true, ""
				}
				return false, "API provider is required"
			},
		},
		{
			Name: "valid_provider",
			Validate: func(value interface{}) (bool, string) {
				validProviders := []string{"cline", "openai", "anthropic", "openrouter", "bedrock", "ollama", "lmstudio", "gemini", "azure", "cerebras"}
				if str, ok := value.(string); ok {
					for _, p := range validProviders {
						if p == str {
							return true, ""
						}
					}
				}
				return false, "Invalid API provider"
			},
		},
	}

	// Model validation
	v.rules["model"] = []ValidationRule{
		{
			Name: "required",
			Validate: func(value interface{}) (bool, string) {
				if str, ok := value.(string); ok && str != "" {
					return true, ""
				}
				return false, "Model is required"
			},
		},
	}

	// API Key validation
	v.rules["apiKey"] = []ValidationRule{
		{
			Name: "format",
			Validate: func(value interface{}) (bool, string) {
				if str, ok := value.(string); ok {
					// Allow [set] placeholder or empty
					if str == "[set]" || str == "" {
						return true, ""
					}
					// Basic API key format checks
					if len(str) < 10 {
						return false, "API key seems too short"
					}
					return true, ""
				}
				return false, "Invalid API key format"
			},
		},
	}

	// Language validation
	v.rules["language"] = []ValidationRule{
		{
			Name: "valid_language",
			Validate: func(value interface{}) (bool, string) {
				validLangs := []string{"en", "es", "fr", "de", "ja", "ko", "zh-cn", "zh-tw", "pt-BR", "ar-sa"}
				if str, ok := value.(string); ok {
					for _, l := range validLangs {
						if l == str {
							return true, ""
						}
					}
				}
				return false, "Invalid language selection"
			},
		},
	}
}

// ValidateSetting validates a single setting
func (v *SettingsValidator) ValidateSetting(key string, value interface{}) ValidationResult {
	rules, exists := v.rules[key]
	if !exists {
		// No validation rules for this setting
		return ValidationResult{Valid: true}
	}

	for _, rule := range rules {
		valid, msg := rule.Validate(value)
		if !valid {
			return ValidationResult{
				Valid:   false,
				Field:   key,
				Message: msg,
				Errors:  map[string]string{key: msg},
			}
		}
	}

	return ValidationResult{Valid: true, Field: key}
}

// ValidateAllSettings validates all settings in a SettingsContent
func (v *SettingsValidator) ValidateAllSettings(content *SettingsContent) ValidationResult {
	allErrors := make(map[string]string)
	valid := true

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
				result := v.ValidateSetting(item.Key, item.Value)
				if !result.Valid {
					valid = false
					allErrors[item.Key] = result.Message
				}
			}
		}
	}

	return ValidationResult{
		Valid:  valid,
		Errors: allErrors,
	}
}

// ValidateAPIKey validates an API key for a specific provider
func (v *SettingsValidator) ValidateAPIKey(provider, apiKey string) ValidationResult {
	// Provider-specific validation patterns
	patterns := map[string]*regexp.Regexp{
		"openai":    regexp.MustCompile(`^sk-[a-zA-Z0-9]{48}$`),
		"anthropic": regexp.MustCompile(`^sk-ant-[a-zA-Z0-9-]{32,}$`),
	}

	if pattern, exists := patterns[provider]; exists {
		if !pattern.MatchString(apiKey) {
			return ValidationResult{
				Valid:   false,
				Field:   "apiKey",
				Message: fmt.Sprintf("Invalid API key format for %s", provider),
			}
		}
	}

	return ValidationResult{Valid: true}
}

// ValidateModelForProvider validates if a model is valid for a provider
func (v *SettingsValidator) ValidateModelForProvider(provider, model string, persistence *SettingsPersistence) ValidationResult {
	if persistence == nil {
		return ValidationResult{Valid: true}
	}

	validModels := persistence.GetAvailableModels(provider)
	for _, m := range validModels {
		if strings.EqualFold(m, model) {
			return ValidationResult{Valid: true}
		}
	}

	return ValidationResult{
		Valid:   false,
		Field:   "model",
		Message: fmt.Sprintf("Model '%s' is not available for provider '%s'", model, provider),
	}
}

// GetValidationError returns the validation error for a setting
func (v *SettingsValidator) GetValidationError(key string) string {
	return v.errors[key]
}

// SetValidationError sets a validation error for a setting
func (v *SettingsValidator) SetValidationError(key, message string) {
	v.errors[key] = message
}

// ClearValidationErrors clears all validation errors
func (v *SettingsValidator) ClearValidationErrors() {
	v.errors = make(map[string]string)
}

// HasErrors returns true if there are validation errors
func (v *SettingsValidator) HasErrors() bool {
	return len(v.errors) > 0
}

// FormatValidationErrors formats validation errors for display
func (v *SettingsValidator) FormatValidationErrors(result ValidationResult) string {
	if result.Valid {
		return ""
	}

	var parts []string
	for field, msg := range result.Errors {
		parts = append(parts, fmt.Sprintf("• %s: %s", field, msg))
	}

	return strings.Join(parts, "\n")
}

// AddCustomRule adds a custom validation rule for a setting
func (v *SettingsValidator) AddCustomRule(key string, rule ValidationRule) {
	if _, exists := v.rules[key]; !exists {
		v.rules[key] = []ValidationRule{}
	}
	v.rules[key] = append(v.rules[key], rule)
}

// ValidateSettingsModel validates a SettingsModel's current state
func ValidateSettingsModel(model *SettingsModel) ValidationResult {
	validator := NewSettingsValidator()
	
	if model.content == nil {
		return ValidationResult{
			Valid:   false,
			Message: "Settings content is nil",
		}
	}

	return validator.ValidateAllSettings(model.content)
}