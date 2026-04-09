// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"strings"
)

// Additional settings tabs for the new settings panel layout
// Note: SettingsTab, SettingsTabAPI, SettingsTabAutoApprove, SettingsTabFeatures,
// SettingsTabAccount, SettingsTabOther, AllSettingsTabs, SettingsTabInfo,
// GetSettingsTabInfo, and SettingsTabNavigator are defined in settings_types.go

const (
	// SettingsTabProvider is the provider selection tab.
	SettingsTabProvider SettingsTab = "provider"
	// SettingsTabModel is the model selection tab.
	SettingsTabModel SettingsTab = "model"
	// SettingsTabAPIKey is the API key configuration tab.
	SettingsTabAPIKey SettingsTab = "apikey"
	// SettingsTabAdvanced is the advanced settings tab.
	SettingsTabAdvanced SettingsTab = "advanced"
)

// NewSettingsTabs returns the new tab-based settings layout.
func NewSettingsTabs() []SettingsTab {
	return []SettingsTab{
		SettingsTabProvider,
		SettingsTabModel,
		SettingsTabAPIKey,
		SettingsTabAdvanced,
	}
}

// GetNewSettingsTabInfo returns information about the new settings tabs.
// This extends GetSettingsTabInfo with new tab types.
func GetNewSettingsTabInfo(tab SettingsTab) SettingsTabInfo {
	switch tab {
	case SettingsTabProvider:
		return SettingsTabInfo{
			Key:         SettingsTabProvider,
			Label:       "Provider",
			Description: "Choose your AI provider",
		}
	case SettingsTabModel:
		return SettingsTabInfo{
			Key:         SettingsTabModel,
			Label:       "Model",
			Description: "Select your AI model",
		}
	case SettingsTabAPIKey:
		return SettingsTabInfo{
			Key:         SettingsTabAPIKey,
			Label:       "API Key",
			Description: "Configure API authentication",
		}
	case SettingsTabAdvanced:
		return SettingsTabInfo{
			Key:         SettingsTabAdvanced,
			Label:       "Advanced",
			Description: "Advanced configuration options",
		}
	default:
		// Fall back to the original GetSettingsTabInfo
		return GetSettingsTabInfo(tab)
	}
}

// SettingsTabNavigatorExtended provides additional methods for tab navigation.
type SettingsTabNavigatorExtended struct {
	*SettingsTabNavigator
}

// NewSettingsTabNavigatorWithTabs creates a navigator with specific tabs.
func NewSettingsTabNavigatorWithTabs(tabs []SettingsTab) *SettingsTabNavigatorExtended {
	return &SettingsTabNavigatorExtended{
		SettingsTabNavigator: &SettingsTabNavigator{
			tabs:         tabs,
			currentIndex: 0,
			styles:       DefaultSettingsTabStyles(),
		},
	}
}

// GetTabCount returns the number of tabs.
func (tn *SettingsTabNavigatorExtended) GetTabCount() int {
	return len(tn.tabs)
}

// GetCurrentIndex returns the current tab index.
func (tn *SettingsTabNavigatorExtended) GetCurrentIndex() int {
	return tn.currentIndex
}

// RenderCompact renders a compact version of the tab bar.
func (tn *SettingsTabNavigator) RenderCompact() string {
	var tabs []string

	for i, tab := range tn.tabs {
		info := GetSettingsTabInfo(tab)
		label := info.Label

		if i == tn.currentIndex {
			tabs = append(tabs, tn.styles.activeTabStyle.Render(label))
		} else {
			// Only show first letter for compact view
			if len(label) > 0 {
				tabs = append(tabs, tn.styles.inactiveTabStyle.Render(label[:1]))
			}
		}
	}

	separator := tn.styles.separatorStyle.Render("/")
	content := strings.Join(tabs, separator)

	return tn.styles.containerStyle.Render(content)
}

// TabNavigationHelper provides keyboard navigation for tabs.
type TabNavigationHelper struct {
	tabs         []SettingsTab
	currentIndex int
	onTabChange  func(SettingsTab)
}

// NewTabNavigationHelper creates a new tab navigation helper.
func NewTabNavigationHelper(tabs []SettingsTab, onTabChange func(SettingsTab)) *TabNavigationHelper {
	return &TabNavigationHelper{
		tabs:         tabs,
		currentIndex: 0,
		onTabChange:  onTabChange,
	}
}

// Next moves to the next tab.
func (tnh *TabNavigationHelper) Next() {
	tnh.currentIndex++
	if tnh.currentIndex >= len(tnh.tabs) {
		tnh.currentIndex = 0
	}
	if tnh.onTabChange != nil {
		tnh.onTabChange(tnh.tabs[tnh.currentIndex])
	}
}

// Prev moves to the previous tab.
func (tnh *TabNavigationHelper) Prev() {
	tnh.currentIndex--
	if tnh.currentIndex < 0 {
		tnh.currentIndex = len(tnh.tabs) - 1
	}
	if tnh.onTabChange != nil {
		tnh.onTabChange(tnh.tabs[tnh.currentIndex])
	}
}

// SetTab sets the current tab by key.
func (tnh *TabNavigationHelper) SetTab(tab SettingsTab) {
	for i, t := range tnh.tabs {
		if t == tab {
			tnh.currentIndex = i
			if tnh.onTabChange != nil {
				tnh.onTabChange(tab)
			}
			return
		}
	}
}

// GetCurrentTab returns the currently selected tab.
func (tnh *TabNavigationHelper) GetCurrentTab() SettingsTab {
	if tnh.currentIndex < 0 || tnh.currentIndex >= len(tnh.tabs) {
		if len(tnh.tabs) > 0 {
			return tnh.tabs[0]
		}
		return ""
	}
	return tnh.tabs[tnh.currentIndex]
}

// GetCurrentIndex returns the current tab index.
func (tnh *TabNavigationHelper) GetCurrentIndex() int {
	return tnh.currentIndex
}
