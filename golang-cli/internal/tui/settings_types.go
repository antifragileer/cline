// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SettingsTab represents a settings tab in the settings panel.
type SettingsTab string

const (
	// SettingsTabAPI is the API configuration tab.
	SettingsTabAPI SettingsTab = "api"
	// SettingsTabAutoApprove is the auto-approval settings tab.
	SettingsTabAutoApprove SettingsTab = "auto-approve"
	// SettingsTabFeatures is the features settings tab.
	SettingsTabFeatures SettingsTab = "features"
	// SettingsTabAccount is the account settings tab.
	SettingsTabAccount SettingsTab = "account"
	// SettingsTabOther is the other settings tab.
	SettingsTabOther SettingsTab = "other"
)

// AllSettingsTabs returns all available settings tabs.
func AllSettingsTabs() []SettingsTab {
	return []SettingsTab{
		SettingsTabAPI,
		SettingsTabAutoApprove,
		SettingsTabFeatures,
		SettingsTabAccount,
		SettingsTabOther,
	}
}

// SettingsTabInfo holds information about a settings tab.
type SettingsTabInfo struct {
	Key         SettingsTab
	Label       string
	Description string
}

// GetSettingsTabInfo returns information about a settings tab.
func GetSettingsTabInfo(tab SettingsTab) SettingsTabInfo {
	switch tab {
	case SettingsTabAPI:
		return SettingsTabInfo{
			Key:         SettingsTabAPI,
			Label:       "API",
			Description: "Configure API provider and model settings",
		}
	case SettingsTabAutoApprove:
		return SettingsTabInfo{
			Key:         SettingsTabAutoApprove,
			Label:       "Auto-approve",
			Description: "Configure automatic approval settings",
		}
	case SettingsTabFeatures:
		return SettingsTabInfo{
			Key:         SettingsTabFeatures,
			Label:       "Features",
			Description: "Enable or disable features",
		}
	case SettingsTabAccount:
		return SettingsTabInfo{
			Key:         SettingsTabAccount,
			Label:       "Account",
			Description: "Manage your Cline account",
		}
	case SettingsTabOther:
		return SettingsTabInfo{
			Key:         SettingsTabOther,
			Label:       "Other",
			Description: "Language, telemetry, and other settings",
		}
	default:
		return SettingsTabInfo{
			Key:         tab,
			Label:       string(tab),
			Description: "",
		}
	}
}

// SettingsTabNavigator provides tab navigation for the settings panel.
type SettingsTabNavigator struct {
	tabs         []SettingsTab
	currentIndex int
	styles       SettingsTabStyles
}

// SettingsTabStyles holds styles for tab navigation.
type SettingsTabStyles struct {
	containerStyle   lipgloss.Style
	tabStyle         lipgloss.Style
	activeTabStyle   lipgloss.Style
	inactiveTabStyle lipgloss.Style
	separatorStyle   lipgloss.Style
}

// DefaultSettingsTabStyles returns default tab styles.
func DefaultSettingsTabStyles() SettingsTabStyles {
	return SettingsTabStyles{
		containerStyle: lipgloss.NewStyle().
			MarginBottom(1),

		tabStyle: lipgloss.NewStyle().
			Padding(0, 2),

		activeTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			Underline(true).
			Padding(0, 2),

		inactiveTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Padding(0, 2),

		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),
	}
}

// NewSettingsTabNavigator creates a new tab navigator.
func NewSettingsTabNavigator() *SettingsTabNavigator {
	return &SettingsTabNavigator{
		tabs:         AllSettingsTabs(),
		currentIndex: 0,
		styles:       DefaultSettingsTabStyles(),
	}
}

// CurrentTab returns the currently selected tab.
func (tn *SettingsTabNavigator) CurrentTab() SettingsTab {
	if tn.currentIndex < 0 || tn.currentIndex >= len(tn.tabs) {
		return SettingsTabAPI
	}
	return tn.tabs[tn.currentIndex]
}

// NextTab moves to the next tab.
func (tn *SettingsTabNavigator) NextTab() {
	tn.currentIndex++
	if tn.currentIndex >= len(tn.tabs) {
		tn.currentIndex = 0
	}
}

// PrevTab moves to the previous tab.
func (tn *SettingsTabNavigator) PrevTab() {
	tn.currentIndex--
	if tn.currentIndex < 0 {
		tn.currentIndex = len(tn.tabs) - 1
	}
}

// SetTab sets the current tab by key.
func (tn *SettingsTabNavigator) SetTab(tab SettingsTab) {
	for i, t := range tn.tabs {
		if t == tab {
			tn.currentIndex = i
			return
		}
	}
}

// Render renders the tab bar.
func (tn *SettingsTabNavigator) Render() string {
	var tabs []string

	for i, tab := range tn.tabs {
		info := GetSettingsTabInfo(tab)
		var rendered string

		if i == tn.currentIndex {
			rendered = tn.styles.activeTabStyle.Render(info.Label)
		} else {
			rendered = tn.styles.inactiveTabStyle.Render(info.Label)
		}

		tabs = append(tabs, rendered)
	}

	separator := tn.styles.separatorStyle.Render(" | ")
	content := strings.Join(tabs, separator)

	return tn.styles.containerStyle.Render(content)
}

// SettingsItemType represents the type of a settings item.
type SettingsItemType string

const (
	// SettingsItemTypeCheckbox is a boolean toggle.
	SettingsItemTypeCheckbox SettingsItemType = "checkbox"
	// SettingsItemTypeSelect is a selection from options.
	SettingsItemTypeSelect SettingsItemType = "select"
	// SettingsItemTypeText is a text input.
	SettingsItemTypeText SettingsItemType = "text"
	// SettingsItemTypePassword is a password input (masked).
	SettingsItemTypePassword SettingsItemType = "password"
	// SettingsItemTypeReadOnly is a read-only display.
	SettingsItemTypeReadOnly SettingsItemType = "readonly"
	// SettingsItemTypeAction is an action button.
	SettingsItemTypeAction SettingsItemType = "action"
	// SettingsItemTypeHeader is a section header.
	SettingsItemTypeHeader SettingsItemType = "header"
	// SettingsItemTypeSeparator is a visual separator.
	SettingsItemTypeSeparator SettingsItemType = "separator"
)

// SettingsItem represents a single settings item.
type SettingsItem struct {
	Key          string
	Label        string
	Description  string
	Type         SettingsItemType
	Value        interface{}
	Options      []string // For select type
	Placeholder  string   // For text input
	Editable     bool
	IsSubItem    bool
	ParentKey    string
	RequiresAuth bool
}

// GetStringValue returns the value as a string.
func (si *SettingsItem) GetStringValue() string {
	switch v := si.Value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetBoolValue returns the value as a bool.
func (si *SettingsItem) GetBoolValue() bool {
	switch v := si.Value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "yes" || v == "1"
	default:
		return false
	}
}

// SetValue sets the item value.
func (si *SettingsItem) SetValue(value interface{}) {
	si.Value = value
}

// ToggleBool toggles a boolean value.
func (si *SettingsItem) ToggleBool() bool {
	if si.Type == SettingsItemTypeCheckbox {
		current := si.GetBoolValue()
		si.Value = !current
		return !current
	}
	return false
}

// CycleOption cycles to the next option for select types.
func (si *SettingsItem) CycleOption() string {
	if si.Type != SettingsItemTypeSelect || len(si.Options) == 0 {
		return si.GetStringValue()
	}

	current := si.GetStringValue()
	currentIdx := 0
	for i, opt := range si.Options {
		if opt == current {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(si.Options)
	si.Value = si.Options[nextIdx]
	return si.Options[nextIdx]
}

// SettingsSection represents a section of settings items.
type SettingsSection struct {
	Title string
	Items []SettingsItem
}

// SettingsContent holds all settings content organized by tab.
type SettingsContent struct {
	API         []SettingsSection
	AutoApprove []SettingsSection
	Features    []SettingsSection
	Account     []SettingsSection
	Other       []SettingsSection
}

// GetSectionsForTab returns the sections for a given tab.
func (sc *SettingsContent) GetSectionsForTab(tab SettingsTab) []SettingsSection {
	switch tab {
	case SettingsTabAPI:
		return sc.API
	case SettingsTabAutoApprove:
		return sc.AutoApprove
	case SettingsTabFeatures:
		return sc.Features
	case SettingsTabAccount:
		return sc.Account
	case SettingsTabOther:
		return sc.Other
	default:
		return nil
	}
}

// DefaultSettingsContent returns the default settings content structure.
func DefaultSettingsContent() *SettingsContent {
	return &SettingsContent{
		API: []SettingsSection{
			{
				Title: "Provider",
				Items: []SettingsItem{
					{
						Key:         "provider",
						Label:       "API Provider",
						Description: "Choose your AI provider",
						Type:        SettingsItemTypeSelect,
						Value:       "cline",
						Options:     []string{"cline", "openai", "anthropic", "openrouter", "bedrock", "ollama", "lmstudio"},
						Editable:    true,
					},
					{
						Key:         "model",
						Label:       "Model",
						Description: "Choose your AI model",
						Type:        SettingsItemTypeSelect,
						Value:       "claude-sonnet-4",
						Options:     []string{"claude-sonnet-4", "claude-opus-4", "gpt-4", "gpt-3.5-turbo"},
						Editable:    true,
					},
					{
						Key:          "apiKey",
						Label:        "API Key",
						Description:  "Your API key for authentication",
						Type:         SettingsItemTypePassword,
						Value:        "",
						Placeholder:  "Enter API key...",
						Editable:     true,
						RequiresAuth: true,
					},
				},
			},
			{
				Title: "Model Settings",
				Items: []SettingsItem{
					{
						Key:         "separateModels",
						Label:       "Separate models for Plan/Act",
						Description: "Use different models for planning and acting",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "planModel",
						Label:       "Plan Model",
						Description: "Model to use in plan mode",
						Type:        SettingsItemTypeSelect,
						Value:       "claude-sonnet-4",
						Options:     []string{"claude-sonnet-4", "claude-opus-4", "gpt-4"},
						Editable:    true,
						IsSubItem:   true,
						ParentKey:   "separateModels",
					},
				},
			},
		},
		AutoApprove: []SettingsSection{
			{
				Title: "Auto-approval Settings",
				Items: []SettingsItem{
					{
						Key:         "yoloMode",
						Label:       "YOLO Mode",
						Description: "Auto-approve all actions (use with caution)",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "autoApproveReadFiles",
						Label:       "Read files",
						Description: "Auto-approve reading files",
						Type:        SettingsItemTypeCheckbox,
						Value:       true,
						Editable:    true,
					},
					{
						Key:         "autoApproveEditFiles",
						Label:       "Edit files",
						Description: "Auto-approve editing files",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "autoApproveRunCommands",
						Label:       "Run commands",
						Description: "Auto-approve running terminal commands",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "autoApproveMcpTools",
						Label:       "MCP tools",
						Description: "Auto-approve MCP tool calls",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
				},
			},
		},
		Features: []SettingsSection{
			{
				Title: "Feature Toggles",
				Items: []SettingsItem{
					{
						Key:         "subagents",
						Label:       "Subagents",
						Description: "Let Cline run focused subagents in parallel",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "autoCondense",
						Label:       "Auto-condense",
						Description: "Automatically summarize long conversations",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
					{
						Key:         "webTools",
						Label:       "Web tools",
						Description: "Enable web search and fetch tools",
						Type:        SettingsItemTypeCheckbox,
						Value:       true,
						Editable:    true,
					},
					{
						Key:         "strictPlanMode",
						Label:       "Strict plan mode",
						Description: "Require explicit mode switching",
						Type:        SettingsItemTypeCheckbox,
						Value:       true,
						Editable:    true,
					},
					{
						Key:         "nativeToolCall",
						Label:       "Native tool call",
						Description: "Use model's native tool calling API",
						Type:        SettingsItemTypeCheckbox,
						Value:       true,
						Editable:    true,
					},
				},
			},
		},
		Account: []SettingsSection{
			{
				Title: "Account",
				Items: []SettingsItem{
					{
						Key:         "accountStatus",
						Label:       "Account Status",
						Description: "Your account status",
						Type:        SettingsItemTypeReadOnly,
						Value:       "Not connected",
						Editable:    false,
					},
					{
						Key:         "balance",
						Label:       "Balance",
						Description: "Your current balance",
						Type:        SettingsItemTypeReadOnly,
						Value:       "$0.00",
						Editable:    false,
					},
					{
						Key:         "connectAccount",
						Label:       "Connect Account",
						Description: "Link your Cline account",
						Type:        SettingsItemTypeAction,
						Value:       "Connect",
						Editable:    true,
					},
				},
			},
		},
		Other: []SettingsSection{
			{
				Title: "Preferences",
				Items: []SettingsItem{
					{
						Key:         "language",
						Label:       "Language",
						Description: "Interface language",
						Type:        SettingsItemTypeSelect,
						Value:       "en",
						Options:     []string{"en", "es", "fr", "de", "ja", "ko", "zh-cn"},
						Editable:    true,
					},
					{
						Key:         "telemetry",
						Label:       "Telemetry",
						Description: "Enable anonymous usage analytics",
						Type:        SettingsItemTypeCheckbox,
						Value:       true,
						Editable:    true,
					},
					{
						Key:         "debug",
						Label:       "Debug Mode",
						Description: "Enable debug logging",
						Type:        SettingsItemTypeCheckbox,
						Value:       false,
						Editable:    true,
					},
				},
			},
		},
	}
}
