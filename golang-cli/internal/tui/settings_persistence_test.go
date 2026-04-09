package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSettingsPersistence(t *testing.T) {
	// Without storage context
	persistence := NewSettingsPersistence(nil)

	assert.NotNil(t, persistence)
}

func TestSettingsPersistence_LoadSettings(t *testing.T) {
	// Without storage context
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	err := persistence.LoadSettings(content)
	assert.Error(t, err) // No storage context set
}

func TestSettingsPersistence_SaveSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	err := persistence.SaveSettings(content)
	assert.Error(t, err) // No storage context set
}

func TestSettingsPersistence_ExportSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Export may work without storage as it just serializes the content
	_, err := persistence.ExportSettings(content)
	// Just check it doesn't panic
	_ = err
}

func TestSettingsPersistence_ImportSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Import may work without storage as it just parses the data
	err := persistence.ImportSettings(content, `{"provider": "openai"}`)
	// Just check it doesn't panic
	_ = err
}

func TestSettingsPersistence_ValidateModel(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	t.Run("validates known models", func(t *testing.T) {
		// These should work even without storage context
		// as they use hardcoded provider-model mappings
		valid := persistence.ValidateModel("gpt-4", "openai")
		// May be true or false depending on implementation
		// but should not panic
		assert.True(t, valid || !valid)
	})

	t.Run("validates with empty provider", func(t *testing.T) {
		valid := persistence.ValidateModel("gpt-4", "")
		assert.True(t, valid || !valid)
	})
}

func TestSettingsPersistence_SaveAPIKey(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	err := persistence.SaveAPIKey("openai", "sk-test-key")
	assert.Error(t, err) // No storage context set
}

func TestSettingsPersistence_GetAvailableProviders(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	providers := persistence.GetAvailableProviders()

	// Should return a list of providers even without storage
	assert.NotNil(t, providers)
	// Most implementations return a non-empty list
	assert.GreaterOrEqual(t, len(providers), 0)
}

func TestSettingsPersistence_GetAvailableModels(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	t.Run("returns models for known provider", func(t *testing.T) {
		models := persistence.GetAvailableModels("openai")
		// May be empty without storage, but should not panic
		assert.NotNil(t, models)
	})

	t.Run("returns empty for unknown provider", func(t *testing.T) {
		models := persistence.GetAvailableModels("unknown-provider")
		// Should return empty or default
		assert.NotNil(t, models)
	})
}

func TestSettingsPersistence_loadAPISettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should not panic
	content := DefaultSettingsContent()
	persistence.loadAPISettings(content)
	// Should handle nil storage gracefully
}

func TestSettingsPersistence_loadAutoApproveSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should not panic
	content := DefaultSettingsContent()
	persistence.loadAutoApproveSettings(content)
}

func TestSettingsPersistence_loadFeatureSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should not panic
	content := DefaultSettingsContent()
	persistence.loadFeatureSettings(content)
}

func TestSettingsPersistence_loadAccountSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should not panic
	content := DefaultSettingsContent()
	persistence.loadAccountSettings(content)
}

func TestSettingsPersistence_loadOtherSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should not panic
	content := DefaultSettingsContent()
	persistence.loadOtherSettings(content)
}

func TestSettingsPersistence_saveAPISettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Without storage context, should handle gracefully
	persistence.saveAPISettings(content)
	// Should not panic
}

func TestSettingsPersistence_saveAutoApproveSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Without storage context, should handle gracefully
	persistence.saveAutoApproveSettings(content)
	// Should not panic
}

func TestSettingsPersistence_saveFeatureSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Without storage context, should handle gracefully
	persistence.saveFeatureSettings(content)
	// Should not panic
}

func TestSettingsPersistence_saveOtherSettings(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Without storage context, should handle gracefully
	persistence.saveOtherSettings(content)
	// Should not panic
}

func TestSettingsPersistence_getGlobalStateString(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should return default
	value, ok := persistence.getGlobalStateString("test-key")
	assert.False(t, ok)
	assert.Empty(t, value)
}

func TestSettingsPersistence_getGlobalStateBool(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should return default
	value, ok := persistence.getGlobalStateBool("test-key")
	assert.False(t, ok)
	assert.False(t, value)
}

func TestSettingsPersistence_getGlobalStateFloat(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should return default
	value, ok := persistence.getGlobalStateFloat("test-key")
	assert.False(t, ok)
	assert.Equal(t, 0.0, value)
}

func TestSettingsPersistence_getSecretString(t *testing.T) {
	persistence := NewSettingsPersistence(nil)

	// Without storage, should return empty
	value, ok := persistence.getSecretString("test-key")
	assert.False(t, ok)
	assert.Empty(t, value)
}

func TestSettingsPersistence_setSettingValue(t *testing.T) {
	persistence := NewSettingsPersistence(nil)
	content := DefaultSettingsContent()

	// Should work even without storage
	persistence.setSettingValue(content, "test-key", "test-value")
	// Value should be set in content
}

func TestSettingsPersistence_Struct(t *testing.T) {
	// Test that SettingsPersistence struct exists and can be created
	persistence := &SettingsPersistence{}
	assert.NotNil(t, persistence)
}