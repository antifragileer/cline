package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingsModel(t *testing.T) {
	t.Run("creates settings model with defaults", func(t *testing.T) {
		model := NewSettingsModel()

		assert.NotNil(t, model)
		assert.NotNil(t, model.tabNavigator)
		assert.NotNil(t, model.content)
		assert.NotNil(t, model.styles)
		assert.Equal(t, 0, model.sectionIndex)
		assert.Equal(t, 0, model.itemIndex)
		assert.False(t, model.editing)
		assert.NotEmpty(t, model.providerList)
	})

	t.Run("initializes provider list", func(t *testing.T) {
		model := NewSettingsModel()

		providers := model.getAvailableProviders()
		assert.NotEmpty(t, providers)
		assert.Contains(t, providers, "anthropic")
		assert.Contains(t, providers, "openai")
	})
}

func TestSettingsModel_SetDimensions(t *testing.T) {
	t.Run("sets width and height", func(t *testing.T) {
		model := NewSettingsModel()

		model.SetDimensions(100, 50)

		assert.Equal(t, 100, model.width)
		assert.Equal(t, 50, model.height)
	})
}

func TestSettingsModel_Init(t *testing.T) {
	t.Run("returns nil command", func(t *testing.T) {
		model := NewSettingsModel()
		cmd := model.Init()

		assert.Nil(t, cmd)
	})
}

func TestSettingsModel_Update_Navigation(t *testing.T) {
	t.Run("Tab key switches to next tab", func(t *testing.T) {
		model := NewSettingsModel()
		initialTab := model.tabNavigator.CurrentTab()

		msg := tea.KeyMsg{Type: tea.KeyTab}
		_, _ = model.Update(msg)

		assert.NotEqual(t, initialTab, model.tabNavigator.CurrentTab())
	})

	t.Run("Shift+Tab switches to previous tab", func(t *testing.T) {
		model := NewSettingsModel()

		// First move to next tab
		model.tabNavigator.NextTab()
		currentTab := model.tabNavigator.CurrentTab()

		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		_, _ = model.Update(msg)

		assert.NotEqual(t, currentTab, model.tabNavigator.CurrentTab())
	})

	t.Run("Right arrow switches to next tab", func(t *testing.T) {
		model := NewSettingsModel()
		initialTab := model.tabNavigator.CurrentTab()

		msg := tea.KeyMsg{Type: tea.KeyRight}
		_, _ = model.Update(msg)

		assert.NotEqual(t, initialTab, model.tabNavigator.CurrentTab())
	})

	t.Run("Left arrow switches to previous tab", func(t *testing.T) {
		model := NewSettingsModel()

		// First move to next tab
		model.tabNavigator.NextTab()
		currentTab := model.tabNavigator.CurrentTab()

		msg := tea.KeyMsg{Type: tea.KeyLeft}
		_, _ = model.Update(msg)

		assert.NotEqual(t, currentTab, model.tabNavigator.CurrentTab())
	})

	t.Run("Down arrow moves to next item", func(t *testing.T) {
		model := NewSettingsModel()
		initialItemIndex := model.itemIndex

		msg := tea.KeyMsg{Type: tea.KeyDown}
		_, _ = model.Update(msg)

		// Item index should change if there are multiple items
		// or stay the same if at the end
		assert.GreaterOrEqual(t, model.itemIndex, initialItemIndex)
	})

	t.Run("Up arrow moves to previous item", func(t *testing.T) {
		model := NewSettingsModel()

		// First move down
		model.moveDown()
		itemIndex := model.itemIndex

		msg := tea.KeyMsg{Type: tea.KeyUp}
		_, _ = model.Update(msg)

		assert.LessOrEqual(t, model.itemIndex, itemIndex)
	})

	t.Run("Esc sets goBack flag", func(t *testing.T) {
		model := NewSettingsModel()

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("Ctrl+C sets goBack flag", func(t *testing.T) {
		model := NewSettingsModel()

		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("q key sets goBack flag", func(t *testing.T) {
		model := NewSettingsModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})

	t.Run("Q key sets goBack flag", func(t *testing.T) {
		model := NewSettingsModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}}
		_, _ = model.Update(msg)

		assert.True(t, model.ShouldGoBack())
	})
}

func TestSettingsModel_HandleEditMode(t *testing.T) {
	t.Run("Esc exits edit mode", func(t *testing.T) {
		model := NewSettingsModel()
		model.editing = true
		model.editValue = "test"

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, _ = model.Update(msg)

		assert.False(t, model.editing)
		assert.Empty(t, model.editValue)
	})

	t.Run("Backspace removes last character", func(t *testing.T) {
		model := NewSettingsModel()
		model.editing = true
		model.editValue = "test"

		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		_, _ = model.Update(msg)

		assert.Equal(t, "tes", model.editValue)
	})

	t.Run("Runes are added to edit value", func(t *testing.T) {
		model := NewSettingsModel()
		model.editing = true
		model.editValue = "test"

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
		_, _ = model.Update(msg)

		assert.Equal(t, "testa", model.editValue)
	})
}

func TestSettingsModel_ShouldGoBack(t *testing.T) {
	t.Run("returns goBack state", func(t *testing.T) {
		model := NewSettingsModel()

		assert.False(t, model.ShouldGoBack())

		model.goBack = true

		assert.True(t, model.ShouldGoBack())
	})
}

func TestSettingsModel_Reset(t *testing.T) {
	t.Run("resets all navigation state", func(t *testing.T) {
		model := NewSettingsModel()
		model.sectionIndex = 2
		model.itemIndex = 3
		model.editing = true
		model.editValue = "test"
		model.goBack = true

		model.Reset()

		assert.Equal(t, 0, model.sectionIndex)
		assert.Equal(t, 0, model.itemIndex)
		assert.False(t, model.editing)
		assert.Empty(t, model.editValue)
		assert.False(t, model.goBack)
	})
}

func TestSettingsModel_GetAvailableProviders(t *testing.T) {
	t.Run("returns list of providers", func(t *testing.T) {
		model := NewSettingsModel()

		providers := model.getAvailableProviders()

		assert.NotEmpty(t, providers)
		assert.Contains(t, providers, "anthropic")
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "openrouter")
	})
}

func TestSettingsModel_GetAvailableModels(t *testing.T) {
	tests := []struct {
		provider string
		expected []string
	}{
		{"anthropic", []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-3-5-sonnet-20241022"}},
		{"openai", []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}},
		{"cline", []string{"claude-sonnet-4", "claude-opus-4"}},
		{"unknown", []string{"default"}},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			model := NewSettingsModel()

			models := model.getAvailableModels(tt.provider)

			assert.NotEmpty(t, models)
			if tt.provider == "unknown" {
				assert.Equal(t, tt.expected, models)
			} else {
				for _, expected := range tt.expected {
					assert.Contains(t, models, expected)
				}
			}
		})
	}
}

func TestSettingsModel_GetSetting(t *testing.T) {
	t.Run("returns setting value when found", func(t *testing.T) {
		model := NewSettingsModel()

		// Set a test value
		model.SetSetting("provider", "anthropic")

		value, found := model.GetSetting("provider")

		assert.True(t, found)
		assert.Equal(t, "anthropic", value)
	})

	t.Run("returns false when setting not found", func(t *testing.T) {
		model := NewSettingsModel()

		_, found := model.GetSetting("nonexistent")

		assert.False(t, found)
	})
}

func TestSettingsModel_SetSetting(t *testing.T) {
	t.Run("sets existing setting value", func(t *testing.T) {
		model := NewSettingsModel()

		success := model.SetSetting("provider", "openai")

		assert.True(t, success)
		value, _ := model.GetSetting("provider")
		assert.Equal(t, "openai", value)
	})

	t.Run("returns false for non-existent setting", func(t *testing.T) {
		model := NewSettingsModel()

		success := model.SetSetting("nonexistent", "value")

		assert.False(t, success)
	})
}

func TestSettingsModel_GetSettings(t *testing.T) {
	t.Run("returns all settings as map", func(t *testing.T) {
		model := NewSettingsModel()

		settings := model.GetSettings()

		assert.NotEmpty(t, settings)
		// Should contain at least some settings
		assert.GreaterOrEqual(t, len(settings), 1)
	})
}

func TestSettingsModel_View(t *testing.T) {
	t.Run("renders settings view", func(t *testing.T) {
		model := NewSettingsModel()
		model.SetDimensions(80, 24)

		view := model.View()

		assert.NotEmpty(t, view)
		assert.Contains(t, view, "Settings")
	})

	t.Run("renders with editing mode", func(t *testing.T) {
		model := NewSettingsModel()
		model.SetDimensions(80, 24)
		model.editing = true
		model.editValue = "test"
		model.editingItem = &SettingsItem{
			Key:   "test",
			Label: "Test",
			Type:  SettingsItemTypeText,
		}

		view := model.View()

		assert.NotEmpty(t, view)
	})
}

func TestSettingsModel_formatValue(t *testing.T) {
	tests := []struct {
		name     string
		item     SettingsItem
		expected string
	}{
		{
			name:     "checkbox checked",
			item:     SettingsItem{Type: SettingsItemTypeCheckbox, Value: true},
			expected: "✓",
		},
		{
			name:     "checkbox unchecked",
			item:     SettingsItem{Type: SettingsItemTypeCheckbox, Value: false},
			expected: "✗",
		},
		{
			name:     "password set",
			item:     SettingsItem{Type: SettingsItemTypePassword, Value: "secret123"},
			expected: "********",
		},
		{
			name:     "password not set",
			item:     SettingsItem{Type: SettingsItemTypePassword, Value: ""},
			expected: "[not set]",
		},
		{
			name:     "text empty",
			item:     SettingsItem{Type: SettingsItemTypeText, Value: ""},
			expected: "[not set]",
		},
		{
			name:     "select value",
			item:     SettingsItem{Type: SettingsItemTypeSelect, Value: "option1"},
			expected: "option1",
		},
		{
			name:     "long text truncated",
			item:     SettingsItem{Type: SettingsItemTypeText, Value: "a very long value that exceeds thirty characters"},
			expected: "a very long value that exce...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewSettingsModel()

			result := model.formatValue(&tt.item)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsModel_handleSpace(t *testing.T) {
	t.Run("toggles checkbox value", func(t *testing.T) {
		model := NewSettingsModel()

		// Find a checkbox item
		sections := model.content.GetSectionsForTab(model.tabNavigator.CurrentTab())
		require.Greater(t, len(sections), 0)
		require.Greater(t, len(sections[0].Items), 0)

		// Set the first item as a checkbox
		sections[0].Items[0].Type = SettingsItemTypeCheckbox
		sections[0].Items[0].Value = false

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		_, _ = model.Update(msg)

		// Value should be toggled
		assert.True(t, sections[0].Items[0].GetBoolValue())
	})
}

func TestSettingsModel_handleAction(t *testing.T) {
	t.Run("handles connect account action", func(t *testing.T) {
		model := NewSettingsModel()

		item := &SettingsItem{
			Key:   "connectAccount",
			Type:  SettingsItemTypeAction,
			Value: "Connect",
		}

		model.handleAction(item)

		assert.Equal(t, "Connecting...", item.Value)
	})
}

func TestSettingsModel_moveUp(t *testing.T) {
	t.Run("moves up within section", func(t *testing.T) {
		model := NewSettingsModel()
		sections := model.content.GetSectionsForTab(model.tabNavigator.CurrentTab())

		if len(sections) > 0 && len(sections[0].Items) > 1 {
			model.sectionIndex = 0
			model.itemIndex = 1

			model.moveUp()

			assert.Equal(t, 0, model.itemIndex)
		}
	})

	t.Run("moves to previous section", func(t *testing.T) {
		model := NewSettingsModel()
		sections := model.content.GetSectionsForTab(model.tabNavigator.CurrentTab())

		if len(sections) > 1 {
			model.sectionIndex = 1
			model.itemIndex = 0

			model.moveUp()

			assert.Equal(t, 0, model.sectionIndex)
			assert.Equal(t, len(sections[0].Items)-1, model.itemIndex)
		}
	})
}

func TestSettingsModel_moveDown(t *testing.T) {
	t.Run("moves down within section", func(t *testing.T) {
		model := NewSettingsModel()
		sections := model.content.GetSectionsForTab(model.tabNavigator.CurrentTab())

		if len(sections) > 0 && len(sections[0].Items) > 1 {
			model.sectionIndex = 0
			model.itemIndex = 0

			model.moveDown()

			assert.Equal(t, 1, model.itemIndex)
		}
	})

	t.Run("moves to next section", func(t *testing.T) {
		model := NewSettingsModel()
		sections := model.content.GetSectionsForTab(model.tabNavigator.CurrentTab())

		if len(sections) > 1 {
			model.sectionIndex = 0
			model.itemIndex = len(sections[0].Items) - 1

			model.moveDown()

			assert.Equal(t, 1, model.sectionIndex)
			assert.Equal(t, 0, model.itemIndex)
		}
	})
}

func TestSettingsModel_getCurrentItem(t *testing.T) {
	t.Run("returns current item when valid", func(t *testing.T) {
		model := NewSettingsModel()

		item := model.getCurrentItem()

		// Should return nil if no valid item
		// or a valid item if sections exist
		if item != nil {
			assert.NotEmpty(t, item.Key)
		}
	})

	t.Run("returns nil for invalid section index", func(t *testing.T) {
		model := NewSettingsModel()
		model.sectionIndex = 999

		item := model.getCurrentItem()

		assert.Nil(t, item)
	})
}

func TestDefaultSettingsStyles(t *testing.T) {
	t.Run("returns default styles", func(t *testing.T) {
		styles := DefaultSettingsStyles()

		assert.NotZero(t, styles.containerStyle)
		assert.NotZero(t, styles.titleStyle)
		assert.NotZero(t, styles.tabBarStyle)
		assert.NotZero(t, styles.sectionTitleStyle)
		assert.NotZero(t, styles.itemStyle)
		assert.NotZero(t, styles.selectedStyle)
		assert.NotZero(t, styles.helpStyle)
		assert.NotZero(t, styles.valueStyle)
		assert.NotZero(t, styles.editStyle)
		assert.NotZero(t, styles.checkboxChecked)
		assert.NotZero(t, styles.checkboxUnchecked)
		assert.NotZero(t, styles.descriptionStyle)
	})
}

func TestSettingsModel_saveSetting(t *testing.T) {
	t.Run("saves setting without error", func(t *testing.T) {
		model := NewSettingsModel()

		item := &SettingsItem{
			Key:   "test",
			Value: "value",
		}

		// Should not panic
		model.saveSetting(item)
	})
}

func TestSettingsModel_StartProviderSelection(t *testing.T) {
	t.Run("starts provider selection mode", func(t *testing.T) {
		model := NewSettingsModel()

		model.startProviderSelection()

		assert.True(t, model.isPickingProvider)
		assert.True(t, model.selectionActive)
		assert.Equal(t, 0, model.selectionCursor)
	})
}

func TestSettingsModel_StartModelSelection(t *testing.T) {
	t.Run("starts model selection mode", func(t *testing.T) {
		model := NewSettingsModel()
		model.SetSetting("provider", "anthropic")

		model.startModelSelection()

		assert.True(t, model.isPickingModel)
		assert.True(t, model.selectionActive)
		assert.Equal(t, 0, model.selectionCursor)
		assert.NotEmpty(t, model.modelList)
	})
}