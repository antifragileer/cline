package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSettingsView(t *testing.T) {
	view := NewSettingsView()

	assert.NotNil(t, view)
	assert.NotNil(t, view.styles)
}

func TestSettingsView_SetStyles(t *testing.T) {
	view := NewSettingsView()
	styles := DefaultSettingsStyles()

	view.SetStyles(styles)

	assert.Equal(t, styles, view.styles)
}

func TestSettingsView_Render(t *testing.T) {
	view := NewSettingsView()
	model := NewSettingsModel()

	result := view.Render(model)

	assert.NotEmpty(t, result)
}

func TestSettingsView_renderSection(t *testing.T) {
	view := NewSettingsView()
	section := SettingsSection{
		Title: "Test Section",
		Items: []SettingsItem{
			{Label: "Item 1", Key: "key1", Value: "value1", Type: SettingsItemTypeText},
		},
	}

	result := view.renderSection(section, 0, 0, 0)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Test Section")
	assert.Contains(t, result, "Item 1")
}

func TestSettingsView_renderItem(t *testing.T) {
	view := NewSettingsView()

	t.Run("renders text item", func(t *testing.T) {
		item := &SettingsItem{
			Label:       "Test Label",
			Key:         "test_key",
			Value:       "test value",
			Type:        SettingsItemTypeText,
			Description: "Test description",
		}
		result := view.renderItem(item, false)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Test Label")
	})

	t.Run("renders checkbox item", func(t *testing.T) {
		item := &SettingsItem{
			Label: "Checkbox",
			Key:   "checkbox",
			Value: true,
			Type:  SettingsItemTypeCheckbox,
		}
		result := view.renderItem(item, false)
		assert.NotEmpty(t, result)
	})

	t.Run("renders select item", func(t *testing.T) {
		item := &SettingsItem{
			Label:   "Select",
			Key:     "select",
			Value:   "option1",
			Type:    SettingsItemTypeSelect,
			Options: []string{"option1", "option2"},
		}
		result := view.renderItem(item, false)
		assert.NotEmpty(t, result)
	})

	t.Run("renders password item", func(t *testing.T) {
		item := &SettingsItem{
			Label: "Password",
			Key:   "password",
			Value: "secret123",
			Type:  SettingsItemTypePassword,
		}
		result := view.renderItem(item, false)
		assert.NotEmpty(t, result)
	})

	t.Run("renders with active state", func(t *testing.T) {
		item := &SettingsItem{
			Label: "Active Item",
			Key:   "active",
			Value: "value",
			Type:  SettingsItemTypeText,
		}
		result := view.renderItem(item, true)
		assert.NotEmpty(t, result)
	})
}

func TestSettingsView_formatValue(t *testing.T) {
	view := NewSettingsView()

	tests := []struct {
		name     string
		item     *SettingsItem
		expected string
	}{
		{"string", &SettingsItem{Type: SettingsItemTypeText, Value: "test"}, "test"},
		{"bool true", &SettingsItem{Type: SettingsItemTypeCheckbox, Value: true}, "✓"},
		{"bool false", &SettingsItem{Type: SettingsItemTypeCheckbox, Value: false}, "✗"},
		{"int", &SettingsItem{Type: SettingsItemTypeText, Value: 42}, "42"},
		{"password", &SettingsItem{Type: SettingsItemTypePassword, Value: "secret"}, "********"},
		{"not set", &SettingsItem{Type: SettingsItemTypeText, Value: ""}, "[not set]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := view.formatValue(tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsView_RenderProviderSelection(t *testing.T) {
	view := NewSettingsView()
	providers := []string{"openai", "anthropic"}

	result := view.RenderProviderSelection(providers, 0, "openai")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "openai")
	assert.Contains(t, result, "anthropic")
}

func TestSettingsView_RenderModelSelection(t *testing.T) {
	view := NewSettingsView()
	models := []string{"gpt-4", "claude-3"}

	result := view.RenderModelSelection(models, 0, "gpt-4")

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "gpt-4")
}

func TestSettingsView_RenderAPIKeyInput(t *testing.T) {
	view := NewSettingsView()

	result := view.RenderAPIKeyInput("sk-test-key", true)

	assert.NotEmpty(t, result)
}

func TestSettingsView_RenderSaveMessage(t *testing.T) {
	view := NewSettingsView()

	result := view.RenderSaveMessage("Settings saved", false)

	assert.NotEmpty(t, result)
}
