package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSettingsValidator(t *testing.T) {
	validator := NewSettingsValidator()

	assert.NotNil(t, validator)
	assert.NotNil(t, validator.errors)
	assert.NotNil(t, validator.rules)
	assert.NotNil(t, validator.highlighter)
}

func TestSettingsValidator_ValidateSetting(t *testing.T) {
	validator := NewSettingsValidator()

	t.Run("validates provider - required", func(t *testing.T) {
		result := validator.ValidateSetting("provider", "")
		assert.False(t, result.Valid)
		assert.Equal(t, "provider", result.Field)
	})

	t.Run("validates provider - valid", func(t *testing.T) {
		result := validator.ValidateSetting("provider", "openai")
		assert.True(t, result.Valid)
	})

	t.Run("validates provider - invalid", func(t *testing.T) {
		result := validator.ValidateSetting("provider", "invalid")
		assert.False(t, result.Valid)
		assert.Contains(t, result.Message, "Invalid API provider")
	})

	t.Run("validates model - required", func(t *testing.T) {
		result := validator.ValidateSetting("model", "")
		assert.False(t, result.Valid)
	})

	t.Run("validates model - valid", func(t *testing.T) {
		result := validator.ValidateSetting("model", "gpt-4")
		assert.True(t, result.Valid)
	})

	t.Run("validates apiKey - format too short", func(t *testing.T) {
		result := validator.ValidateSetting("apiKey", "short")
		assert.False(t, result.Valid)
		assert.Contains(t, result.Message, "too short")
	})

	t.Run("validates apiKey - valid with placeholder", func(t *testing.T) {
		result := validator.ValidateSetting("apiKey", "[set]")
		assert.True(t, result.Valid)
	})

	t.Run("validates apiKey - valid empty", func(t *testing.T) {
		result := validator.ValidateSetting("apiKey", "")
		assert.True(t, result.Valid)
	})

	t.Run("validates apiKey - valid format", func(t *testing.T) {
		result := validator.ValidateSetting("apiKey", "this-is-a-valid-key-with-enough-length")
		assert.True(t, result.Valid)
	})

	t.Run("validates language - valid", func(t *testing.T) {
		result := validator.ValidateSetting("language", "en")
		assert.True(t, result.Valid)
	})

	t.Run("validates language - invalid", func(t *testing.T) {
		result := validator.ValidateSetting("language", "invalid")
		assert.False(t, result.Valid)
		assert.Contains(t, result.Message, "Invalid language")
	})

	t.Run("no rules - returns valid", func(t *testing.T) {
		result := validator.ValidateSetting("unknown_key", "any_value")
		assert.True(t, result.Valid)
	})
}

func TestSettingsValidator_ValidateAllSettings(t *testing.T) {
	validator := NewSettingsValidator()

	t.Run("validates all settings - valid", func(t *testing.T) {
		content := DefaultSettingsContent()
		result := validator.ValidateAllSettings(content)

		// Should be valid by default
		assert.True(t, result.Valid)
	})

	t.Run("validates all settings - with errors", func(t *testing.T) {
		content := DefaultSettingsContent()
		// Set an invalid value
		content.API[0].Items[0].Value = "invalid_provider"

		result := validator.ValidateAllSettings(content)

		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})
}

func TestSettingsValidator_ValidateAPIKey(t *testing.T) {
	validator := NewSettingsValidator()

	t.Run("validates OpenAI key format", func(t *testing.T) {
		// Valid OpenAI key format: sk- followed by exactly 48 alphanumeric chars
		// Create a key with exactly 48 alphanumeric chars after sk-
		key := "sk-" + "abcdefghijklmnopqrstuvwxyz1234567890123456789012"
		// Verify length: 3 + 48 = 51
		if len(key) != 51 {
			t.Fatalf("Key length should be 51, got %d", len(key))
		}
		result := validator.ValidateAPIKey("openai", key)
		assert.True(t, result.Valid)
	})

	t.Run("invalid OpenAI key format", func(t *testing.T) {
		result := validator.ValidateAPIKey("openai", "invalid-key")
		assert.False(t, result.Valid)
		assert.Contains(t, result.Message, "Invalid API key format")
	})

	t.Run("validates Anthropic key format", func(t *testing.T) {
		// Valid Anthropic key format: sk-ant- followed by at least 32 alphanumeric chars and dashes
		result := validator.ValidateAPIKey("anthropic", "sk-ant-api03-valid-key-with-sufficient-length")
		assert.True(t, result.Valid)
	})

	t.Run("invalid Anthropic key format", func(t *testing.T) {
		result := validator.ValidateAPIKey("anthropic", "sk-ant-short")
		assert.False(t, result.Valid)
	})

	t.Run("no pattern for unknown provider", func(t *testing.T) {
		result := validator.ValidateAPIKey("unknown", "any-key")
		assert.True(t, result.Valid)
	})
}

func TestSettingsValidator_ValidateModelForProvider(t *testing.T) {
	validator := NewSettingsValidator()

	t.Run("nil persistence returns valid", func(t *testing.T) {
		result := validator.ValidateModelForProvider("openai", "gpt-4", nil)
		assert.True(t, result.Valid)
	})
}

func TestSettingsValidator_GetValidationError(t *testing.T) {
	validator := NewSettingsValidator()
	validator.errors["test_key"] = "test error"

	result := validator.GetValidationError("test_key")
	assert.Equal(t, "test error", result)

	// Non-existent key returns empty
	result = validator.GetValidationError("nonexistent")
	assert.Empty(t, result)
}

func TestSettingsValidator_SetValidationError(t *testing.T) {
	validator := NewSettingsValidator()

	validator.SetValidationError("my_key", "my error message")

	assert.Equal(t, "my error message", validator.errors["my_key"])
}

func TestSettingsValidator_ClearValidationErrors(t *testing.T) {
	validator := NewSettingsValidator()
	validator.errors["key1"] = "error1"
	validator.errors["key2"] = "error2"

	validator.ClearValidationErrors()

	assert.Empty(t, validator.errors)
}

func TestSettingsValidator_HasErrors(t *testing.T) {
	validator := NewSettingsValidator()

	assert.False(t, validator.HasErrors())

	validator.errors["key"] = "error"
	assert.True(t, validator.HasErrors())
}

func TestSettingsValidator_FormatValidationErrors(t *testing.T) {
	validator := NewSettingsValidator()

	t.Run("valid result returns empty", func(t *testing.T) {
		result := ValidationResult{Valid: true}
		formatted := validator.FormatValidationErrors(result)
		assert.Empty(t, formatted)
	})

	t.Run("formats errors", func(t *testing.T) {
		result := ValidationResult{
			Valid: false,
			Errors: map[string]string{
				"field1": "error message 1",
				"field2": "error message 2",
			},
		}
		formatted := validator.FormatValidationErrors(result)
		assert.Contains(t, formatted, "field1:")
		assert.Contains(t, formatted, "error message 1")
		assert.Contains(t, formatted, "field2:")
		assert.Contains(t, formatted, "error message 2")
	})
}

func TestSettingsValidator_AddCustomRule(t *testing.T) {
	validator := NewSettingsValidator()

	// Add a custom rule
	validator.AddCustomRule("custom_field", ValidationRule{
		Name: "test_rule",
		Validate: func(value interface{}) (bool, string) {
			if str, ok := value.(string); ok && str == "valid" {
				return true, ""
			}
			return false, "custom validation failed"
		},
	})

	// Test the custom rule
	result := validator.ValidateSetting("custom_field", "valid")
	assert.True(t, result.Valid)

	result = validator.ValidateSetting("custom_field", "invalid")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "custom validation failed")
}

func TestValidateSettingsModel(t *testing.T) {
	t.Run("validates model with content", func(t *testing.T) {
		model := &SettingsModel{
			content: DefaultSettingsContent(),
		}

		result := ValidateSettingsModel(model)

		assert.True(t, result.Valid)
	})

	t.Run("nil content returns invalid", func(t *testing.T) {
		model := &SettingsModel{
			content: nil,
		}

		result := ValidateSettingsModel(model)

		assert.False(t, result.Valid)
		assert.Contains(t, result.Message, "nil")
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := ValidationResult{
			Valid:   true,
			Field:   "test",
			Message: "",
			Errors:  nil,
		}
		assert.True(t, result.Valid)
	})

	t.Run("invalid result", func(t *testing.T) {
		result := ValidationResult{
			Valid:   false,
			Field:   "test",
			Message: "error",
			Errors:  map[string]string{"test": "error"},
		}
		assert.False(t, result.Valid)
	})
}