package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllSettingsTabs(t *testing.T) {
	tabs := AllSettingsTabs()

	assert.Equal(t, 5, len(tabs))
	assert.Contains(t, tabs, SettingsTabAPI)
	assert.Contains(t, tabs, SettingsTabAutoApprove)
	assert.Contains(t, tabs, SettingsTabFeatures)
	assert.Contains(t, tabs, SettingsTabAccount)
	assert.Contains(t, tabs, SettingsTabOther)
}

func TestGetSettingsTabInfo(t *testing.T) {
	tests := []struct {
		tab             SettingsTab
		expectedLabel   string
		expectedDesc    string
	}{
		{SettingsTabAPI, "API", "Configure API provider and model settings"},
		{SettingsTabAutoApprove, "Auto-approve", "Configure automatic approval settings"},
		{SettingsTabFeatures, "Features", "Enable or disable features"},
		{SettingsTabAccount, "Account", "Manage your Cline account"},
		{SettingsTabOther, "Other", "Language, telemetry, and other settings"},
		{"unknown", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			info := GetSettingsTabInfo(tt.tab)
			assert.Equal(t, tt.tab, info.Key)
			assert.Equal(t, tt.expectedLabel, info.Label)
			assert.Equal(t, tt.expectedDesc, info.Description)
		})
	}
}

func TestDefaultSettingsTabStyles(t *testing.T) {
	styles := DefaultSettingsTabStyles()

	assert.NotZero(t, styles.containerStyle)
	assert.NotZero(t, styles.tabStyle)
	assert.NotZero(t, styles.activeTabStyle)
	assert.NotZero(t, styles.inactiveTabStyle)
	assert.NotZero(t, styles.separatorStyle)
}

func TestNewSettingsTabNavigator(t *testing.T) {
	navigator := NewSettingsTabNavigator()

	assert.NotNil(t, navigator)
	assert.Equal(t, 5, len(navigator.tabs))
	assert.Equal(t, 0, navigator.currentIndex)
}

func TestSettingsTabNavigator_CurrentTab(t *testing.T) {
	t.Run("returns current tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		tab := navigator.CurrentTab()
		assert.Equal(t, SettingsTabAPI, tab)
	})

	t.Run("returns default for negative index", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.currentIndex = -1
		tab := navigator.CurrentTab()
		assert.Equal(t, SettingsTabAPI, tab)
	})

	t.Run("returns default for out of bounds index", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.currentIndex = 100
		tab := navigator.CurrentTab()
		assert.Equal(t, SettingsTabAPI, tab)
	})
}

func TestSettingsTabNavigator_NextTab(t *testing.T) {
	t.Run("moves to next tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.NextTab()
		assert.Equal(t, 1, navigator.currentIndex)
	})

	t.Run("wraps to first tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.currentIndex = 4 // Last tab
		navigator.NextTab()
		assert.Equal(t, 0, navigator.currentIndex)
	})
}

func TestSettingsTabNavigator_PrevTab(t *testing.T) {
	t.Run("moves to previous tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.currentIndex = 2
		navigator.PrevTab()
		assert.Equal(t, 1, navigator.currentIndex)
	})

	t.Run("wraps to last tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.currentIndex = 0
		navigator.PrevTab()
		assert.Equal(t, 4, navigator.currentIndex)
	})
}

func TestSettingsTabNavigator_SetTab(t *testing.T) {
	t.Run("sets tab by key", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.SetTab(SettingsTabFeatures)
		assert.Equal(t, 2, navigator.currentIndex)
	})

	t.Run("does nothing for unknown tab", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		navigator.SetTab("unknown")
		// Should remain at 0
		assert.Equal(t, 0, navigator.currentIndex)
	})
}

func TestSettingsTabNavigator_Render(t *testing.T) {
	navigator := NewSettingsTabNavigator()
	view := navigator.Render()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "API")
	assert.Contains(t, view, "Auto-approve")
	assert.Contains(t, view, "|")
}

func TestSettingsItem_GetStringValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"float64", 3.14, "3.14"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := SettingsItem{Value: tt.value}
			result := item.GetStringValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsItem_GetBoolValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string yes", "yes", true},
		{"string 1", "1", true},
		{"string false", "false", false},
		{"string other", "other", false},
		{"int", 42, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := SettingsItem{Value: tt.value}
			result := item.GetBoolValue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSettingsItem_SetValue(t *testing.T) {
	item := SettingsItem{}
	item.SetValue("new value")
	assert.Equal(t, "new value", item.Value)
}

func TestSettingsItem_ToggleBool(t *testing.T) {
	t.Run("toggles checkbox", func(t *testing.T) {
		item := SettingsItem{Type: SettingsItemTypeCheckbox, Value: false}
		result := item.ToggleBool()
		assert.True(t, result)
		assert.True(t, item.GetBoolValue())
	})

	t.Run("returns false for non-checkbox", func(t *testing.T) {
		item := SettingsItem{Type: SettingsItemTypeText, Value: true}
		result := item.ToggleBool()
		assert.False(t, result)
	})
}

func TestSettingsItem_CycleOption(t *testing.T) {
	t.Run("cycles to next option", func(t *testing.T) {
		item := SettingsItem{
			Type:    SettingsItemTypeSelect,
			Value:   "option1",
			Options: []string{"option1", "option2", "option3"},
		}
		result := item.CycleOption()
		assert.Equal(t, "option2", result)
	})

	t.Run("wraps to first option", func(t *testing.T) {
		item := SettingsItem{
			Type:    SettingsItemTypeSelect,
			Value:   "option3",
			Options: []string{"option1", "option2", "option3"},
		}
		result := item.CycleOption()
		assert.Equal(t, "option1", result)
	})

	t.Run("returns current for non-select", func(t *testing.T) {
		item := SettingsItem{
			Type:  SettingsItemTypeText,
			Value: "value",
		}
		result := item.CycleOption()
		assert.Equal(t, "value", result)
	})

	t.Run("returns current for empty options", func(t *testing.T) {
		item := SettingsItem{
			Type:    SettingsItemTypeSelect,
			Value:   "value",
			Options: []string{},
		}
		result := item.CycleOption()
		assert.Equal(t, "value", result)
	})
}

func TestSettingsContent_GetSectionsForTab(t *testing.T) {
	content := DefaultSettingsContent()

	tests := []struct {
		tab      SettingsTab
		expected int // expected minimum number of sections
	}{
		{SettingsTabAPI, 1},
		{SettingsTabAutoApprove, 1},
		{SettingsTabFeatures, 1},
		{SettingsTabAccount, 1},
		{SettingsTabOther, 1},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			sections := content.GetSectionsForTab(tt.tab)
			if tt.expected == 0 {
				assert.Nil(t, sections)
			} else {
				assert.GreaterOrEqual(t, len(sections), tt.expected)
			}
		})
	}
}

func TestDefaultSettingsContent(t *testing.T) {
	content := DefaultSettingsContent()

	assert.NotNil(t, content)
	assert.NotEmpty(t, content.API)
	assert.NotEmpty(t, content.AutoApprove)
	assert.NotEmpty(t, content.Features)
	assert.NotEmpty(t, content.Account)
	assert.NotEmpty(t, content.Other)

	// Check API section has provider item
	assert.GreaterOrEqual(t, len(content.API[0].Items), 1)
	assert.Equal(t, "provider", content.API[0].Items[0].Key)
}