// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// SettingItem represents a settings item.
type SettingItem struct {
	Key         string
	Name        string
	Description string
	Value       string
	Type        string // "string", "bool", "select"
	Options     []string
	Editable    bool
}

// SettingsModel represents the settings screen state.
type SettingsModel struct {
	width    int
	height   int
	items    []SettingItem
	cursor   int
	goBack   bool
	editing  bool
	editValue string
	styles   SettingsStyles
}

// SettingsStyles holds styling for the settings screen.
type SettingsStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
	itemStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	dimStyle       lipgloss.Style
	helpStyle      lipgloss.Style
	valueStyle     lipgloss.Style
	editStyle      lipgloss.Style
}

// DefaultSettingsStyles returns default settings styles.
func DefaultSettingsStyles() SettingsStyles {
	return SettingsStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(2, 4),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			PaddingLeft(2),

		dimStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),

		valueStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)),

		editStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SelectionBlue)).
			Padding(0, 1),
	}
}

// NewSettingsModel creates a new settings model.
func NewSettingsModel() *SettingsModel {
	m := &SettingsModel{
		items:  make([]SettingItem, 0),
		cursor: 0,
		styles: DefaultSettingsStyles(),
	}
	m.loadDefaultSettings()
	return m
}

// loadDefaultSettings loads default settings.
func (m *SettingsModel) loadDefaultSettings() {
	m.items = []SettingItem{
		{
			Key:         "mode",
			Name:        "Default Mode",
			Description: "Default mode for new tasks",
			Value:       "act",
			Type:        "select",
			Options:     []string{"act", "plan"},
			Editable:    true,
		},
		{
			Key:         "apiProvider",
			Name:        "API Provider",
			Description: "Default AI provider",
			Value:       "cline",
			Type:        "select",
			Options:     []string{"cline", "openai", "anthropic", "openrouter"},
			Editable:    true,
		},
		{
			Key:         "modelId",
			Name:        "Model",
			Description: "Default AI model",
			Value:       "claude-sonnet-4",
			Type:        "string",
			Editable:    true,
		},
		{
			Key:         "yoloMode",
			Name:        "YOLO Mode",
			Description: "Auto-approve all actions",
			Value:       "false",
			Type:        "bool",
			Editable:    true,
		},
		{
			Key:         "autoApprove",
			Name:        "Auto-approve",
			Description: "Auto-approve safe operations",
			Value:       "true",
			Type:        "bool",
			Editable:    true,
		},
	}
}

// SetDimensions sets the terminal dimensions.
func (m *SettingsModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			return m.handleEditMode(msg)
		}
		return m.handleNavigation(msg)
	}

	return m, nil
}

// handleNavigation handles navigation in the settings list.
func (m *SettingsModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.goBack = true
		return m, nil

	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		return m, nil

	case tea.KeyEnter:
		if m.cursor < len(m.items) {
			item := &m.items[m.cursor]
			if item.Editable {
				m.startEditing(item)
			}
		}
		return m, nil

	case tea.KeyRunes:
		switch msg.String() {
		case "q", "Q":
			m.goBack = true
			return m, nil
		case "e", "E":
			if m.cursor < len(m.items) {
				item := &m.items[m.cursor]
				if item.Editable {
					m.startEditing(item)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// handleEditMode handles input when editing a setting.
func (m *SettingsModel) handleEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.editing = false
		m.editValue = ""
		return m, nil

	case tea.KeyEnter:
		// Save the value
		if m.cursor < len(m.items) {
			m.items[m.cursor].Value = m.editValue
		}
		m.editing = false
		m.editValue = ""
		return m, nil

	case tea.KeyBackspace:
		if len(m.editValue) > 0 {
			m.editValue = m.editValue[:len(m.editValue)-1]
		}
		return m, nil

	case tea.KeyRunes:
		m.editValue += msg.String()
		return m, nil
	}

	return m, nil
}

// startEditing starts editing a setting.
func (m *SettingsModel) startEditing(item *SettingItem) {
	m.editing = true
	m.editValue = item.Value

	// For boolean and select types, cycle through options
	if item.Type == "bool" {
		if item.Value == "true" {
			item.Value = "false"
		} else {
			item.Value = "true"
		}
		m.editing = false
		m.editValue = ""
	} else if item.Type == "select" && len(item.Options) > 0 {
		// Find current option and cycle to next
		currentIdx := 0
		for i, opt := range item.Options {
			if opt == item.Value {
				currentIdx = i
				break
			}
		}
		nextIdx := (currentIdx + 1) % len(item.Options)
		item.Value = item.Options[nextIdx]
		m.editing = false
		m.editValue = ""
	}
}

// View renders the settings screen.
func (m SettingsModel) View() string {
	var content strings.Builder

	// Title
	content.WriteString(m.styles.titleStyle.Render("Settings"))
	content.WriteString("\n\n")

	// Calculate visible range
	maxVisible := m.height - 8
	startIdx := 0
	endIdx := len(m.items)

	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
		endIdx = startIdx + maxVisible
		if endIdx > len(m.items) {
			endIdx = len(m.items)
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Items
	for i := startIdx; i < endIdx && i < len(m.items); i++ {
		item := m.items[i]
		isSelected := i == m.cursor

		line := m.formatItem(item, isSelected)
		if isSelected {
			content.WriteString(m.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(m.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")

		// Show description for selected item
		if isSelected {
			content.WriteString(m.styles.dimStyle.Render("  " + item.Description))
			content.WriteString("\n")
		}
	}

	// Edit mode
	if m.editing {
		content.WriteString("\n")
		content.WriteString(m.styles.editStyle.Render(m.editValue + "▌"))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	if m.editing {
		content.WriteString(m.styles.helpStyle.Render("Enter to save • Esc to cancel"))
	} else {
		content.WriteString(m.styles.helpStyle.Render("↑↓ to navigate • Enter/E to edit • Esc/q to go back"))
	}

	return m.styles.containerStyle.Render(content.String())
}

// formatItem formats a setting item for display.
func (m *SettingsModel) formatItem(item SettingItem, selected bool) string {
	var parts []string

	// Name
	parts = append(parts, item.Name)

	// Value
	valueStr := m.formatValue(item)
	parts = append(parts, m.styles.valueStyle.Render(valueStr))

	// Editable indicator
	if item.Editable && !selected {
		parts = append(parts, m.styles.dimStyle.Render("[e]"))
	}

	return strings.Join(parts, " ")
}

// formatValue formats a setting value for display.
func (m *SettingsModel) formatValue(item SettingItem) string {
	switch item.Type {
	case "bool":
		if item.Value == "true" {
			return "✓"
		}
		return "✗"
	default:
		return item.Value
	}
}

// ShouldGoBack returns true if the user wants to go back.
func (m *SettingsModel) ShouldGoBack() bool {
	return m.goBack
}

// GetSettings returns all settings.
func (m *SettingsModel) GetSettings() map[string]string {
	result := make(map[string]string)
	for _, item := range m.items {
		result[item.Key] = item.Value
	}
	return result
}

// Reset resets the model state.
func (m *SettingsModel) Reset() {
	m.cursor = 0
	m.goBack = false
	m.editing = false
	m.editValue = ""
}

// LoadFromStorage loads settings from the storage context.
func (m *SettingsModel) LoadFromStorage(storageCtx *storage.StorageContext) {
	if storageCtx == nil {
		return
	}

	// Update settings values from storage
	for i := range m.items {
		item := &m.items[i]
		var val interface{}
		var ok bool

		// Try global state first, then workspace state
		val, ok = storageCtx.GlobalState.Get(item.Key)
		if !ok {
			val, ok = storageCtx.WorkspaceState.Get(item.Key)
		}

		if ok {
			switch v := val.(type) {
			case string:
				item.Value = v
			case bool:
				item.Value = fmt.Sprintf("%t", v)
			case int, int64:
				item.Value = fmt.Sprintf("%d", v)
			case float64:
				item.Value = fmt.Sprintf("%g", v)
			}
		}
	}
}

// SaveToStorage saves the current settings to storage.
func (m *SettingsModel) SaveToStorage(storageCtx *storage.StorageContext) error {
	if storageCtx == nil {
		return fmt.Errorf("storage context is nil")
	}

	for _, item := range m.items {
		var val interface{}
		
		switch item.Type {
		case "bool":
			val = item.Value == "true"
		case "select", "string":
			val = item.Value
		default:
			val = item.Value
		}

		// Save to global state
		if err := storageCtx.GlobalState.Set(item.Key, val); err != nil {
			return fmt.Errorf("failed to save setting %s: %w", item.Key, err)
		}
	}

	return nil
}

// SetStorageContext loads settings from storage.
func (m *SettingsModel) SetStorageContext(storageCtx *storage.StorageContext) {
	m.LoadFromStorage(storageCtx)
}
