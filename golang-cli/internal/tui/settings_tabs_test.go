package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSettingsTabs(t *testing.T) {
	tabs := NewSettingsTabs()

	assert.Equal(t, 4, len(tabs))
	assert.Equal(t, SettingsTabProvider, tabs[0])
	assert.Equal(t, SettingsTabModel, tabs[1])
	assert.Equal(t, SettingsTabAPIKey, tabs[2])
	assert.Equal(t, SettingsTabAdvanced, tabs[3])
}

func TestGetNewSettingsTabInfo(t *testing.T) {
	t.Run("returns provider tab info", func(t *testing.T) {
		info := GetNewSettingsTabInfo(SettingsTabProvider)
		assert.Equal(t, SettingsTabProvider, info.Key)
		assert.Equal(t, "Provider", info.Label)
		assert.Equal(t, "Choose your AI provider", info.Description)
	})

	t.Run("returns model tab info", func(t *testing.T) {
		info := GetNewSettingsTabInfo(SettingsTabModel)
		assert.Equal(t, SettingsTabModel, info.Key)
		assert.Equal(t, "Model", info.Label)
		assert.Equal(t, "Select your AI model", info.Description)
	})

	t.Run("returns apikey tab info", func(t *testing.T) {
		info := GetNewSettingsTabInfo(SettingsTabAPIKey)
		assert.Equal(t, SettingsTabAPIKey, info.Key)
		assert.Equal(t, "API Key", info.Label)
		assert.Equal(t, "Configure API authentication", info.Description)
	})

	t.Run("returns advanced tab info", func(t *testing.T) {
		info := GetNewSettingsTabInfo(SettingsTabAdvanced)
		assert.Equal(t, SettingsTabAdvanced, info.Key)
		assert.Equal(t, "Advanced", info.Label)
		assert.Equal(t, "Advanced configuration options", info.Description)
	})

	t.Run("falls back to GetSettingsTabInfo for unknown tabs", func(t *testing.T) {
		info := GetNewSettingsTabInfo(SettingsTabAPI)
		assert.Equal(t, SettingsTabAPI, info.Key)
		assert.Equal(t, "API", info.Label)
	})
}

func TestNewSettingsTabNavigatorWithTabs(t *testing.T) {
	tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
	navigator := NewSettingsTabNavigatorWithTabs(tabs)

	assert.NotNil(t, navigator)
	assert.Equal(t, 2, len(navigator.tabs))
	assert.Equal(t, 0, navigator.currentIndex)
}

func TestSettingsTabNavigatorExtended_GetTabCount(t *testing.T) {
	tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel, SettingsTabAPIKey}
	navigator := NewSettingsTabNavigatorWithTabs(tabs)

	count := navigator.GetTabCount()

	assert.Equal(t, 3, count)
}

func TestSettingsTabNavigatorExtended_GetCurrentIndex(t *testing.T) {
	tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
	navigator := NewSettingsTabNavigatorWithTabs(tabs)

	assert.Equal(t, 0, navigator.GetCurrentIndex())

	navigator.NextTab()
	assert.Equal(t, 1, navigator.GetCurrentIndex())
}

func TestSettingsTabNavigator_RenderCompact(t *testing.T) {
	t.Run("renders compact view", func(t *testing.T) {
		navigator := NewSettingsTabNavigator()
		view := navigator.RenderCompact()

		assert.NotEmpty(t, view)
		// Should contain separators
		assert.Contains(t, view, "/")
	})
}

func TestNewTabNavigationHelper(t *testing.T) {
	tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
	var called bool
	onChange := func(tab SettingsTab) {
		called = true
	}

	helper := NewTabNavigationHelper(tabs, onChange)

	assert.NotNil(t, helper)
	assert.Equal(t, 2, len(helper.tabs))
	assert.Equal(t, 0, helper.currentIndex)
	assert.NotNil(t, helper.onTabChange)

	// Test callback
	helper.Next()
	assert.True(t, called)
}

func TestTabNavigationHelper_Next(t *testing.T) {
	t.Run("moves to next tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel, SettingsTabAPIKey}
		helper := NewTabNavigationHelper(tabs, nil)

		helper.Next()

		assert.Equal(t, 1, helper.currentIndex)
	})

	t.Run("wraps to first tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		helper := NewTabNavigationHelper(tabs, nil)
		helper.currentIndex = 1

		helper.Next()

		assert.Equal(t, 0, helper.currentIndex)
	})

	t.Run("calls onTabChange", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		var changedTab SettingsTab
		onChange := func(tab SettingsTab) {
			changedTab = tab
		}
		helper := NewTabNavigationHelper(tabs, onChange)

		helper.Next()

		assert.Equal(t, SettingsTabModel, changedTab)
	})
}

func TestTabNavigationHelper_Prev(t *testing.T) {
	t.Run("moves to previous tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel, SettingsTabAPIKey}
		helper := NewTabNavigationHelper(tabs, nil)
		helper.currentIndex = 2

		helper.Prev()

		assert.Equal(t, 1, helper.currentIndex)
	})

	t.Run("wraps to last tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		helper := NewTabNavigationHelper(tabs, nil)

		helper.Prev()

		assert.Equal(t, 1, helper.currentIndex)
	})

	t.Run("calls onTabChange", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		var changedTab SettingsTab
		onChange := func(tab SettingsTab) {
			changedTab = tab
		}
		helper := NewTabNavigationHelper(tabs, onChange)

		helper.Prev()

		assert.Equal(t, SettingsTabModel, changedTab)
	})
}

func TestTabNavigationHelper_SetTab(t *testing.T) {
	t.Run("sets tab by key", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel, SettingsTabAPIKey}
		helper := NewTabNavigationHelper(tabs, nil)

		helper.SetTab(SettingsTabAPIKey)

		assert.Equal(t, 2, helper.currentIndex)
	})

	t.Run("does nothing for unknown tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		helper := NewTabNavigationHelper(tabs, nil)

		helper.SetTab(SettingsTabAdvanced) // Not in list

		// Should remain at 0
		assert.Equal(t, 0, helper.currentIndex)
	})

	t.Run("calls onTabChange", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		var changedTab SettingsTab
		onChange := func(tab SettingsTab) {
			changedTab = tab
		}
		helper := NewTabNavigationHelper(tabs, onChange)

		helper.SetTab(SettingsTabModel)

		assert.Equal(t, SettingsTabModel, changedTab)
	})
}

func TestTabNavigationHelper_GetCurrentTab(t *testing.T) {
	t.Run("returns current tab", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		helper := NewTabNavigationHelper(tabs, nil)

		tab := helper.GetCurrentTab()

		assert.Equal(t, SettingsTabProvider, tab)
	})

	t.Run("returns empty for empty tabs", func(t *testing.T) {
		helper := NewTabNavigationHelper([]SettingsTab{}, nil)

		tab := helper.GetCurrentTab()

		assert.Empty(t, tab)
	})

	t.Run("returns first tab for invalid index", func(t *testing.T) {
		tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
		helper := NewTabNavigationHelper(tabs, nil)
		helper.currentIndex = -1

		tab := helper.GetCurrentTab()

		assert.Equal(t, SettingsTabProvider, tab)
	})
}

func TestTabNavigationHelper_GetCurrentIndex(t *testing.T) {
	tabs := []SettingsTab{SettingsTabProvider, SettingsTabModel}
	helper := NewTabNavigationHelper(tabs, nil)

	assert.Equal(t, 0, helper.GetCurrentIndex())

	helper.Next()
	assert.Equal(t, 1, helper.GetCurrentIndex())
}