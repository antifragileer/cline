// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// SettingsModel represents the settings screen state.
type SettingsModel struct {
	// Dimensions
	width  int
	height int

	// Tab navigation
	tabNavigator *SettingsTabNavigator

	// Settings content
	content *SettingsContent

	// Navigation state
	sectionIndex int
	itemIndex    int
	goBack       bool

	// Editing state
	editing     bool
	editValue   string
	editingItem *SettingsItem

	// Sub-modes
	isPickingProvider bool
	isPickingModel    bool
	isEnteringAPIKey  bool

	// Storage
	storageCtx      *storage.StorageContext
	persistence     *SettingsPersistence

	// Provider/Model selection
	providerList     []string
	modelList        []string
	selectionCursor  int
	selectionActive  bool

	// Styles
	styles SettingsStyles

	// Message channel for async operations
	messageChan chan tea.Msg
}

// SettingsStyles holds styling for the settings screen.
type SettingsStyles struct {
	containerStyle    lipgloss.Style
	titleStyle        lipgloss.Style
	tabBarStyle       lipgloss.Style
	sectionStyle      lipgloss.Style
	sectionTitleStyle lipgloss.Style
	itemStyle         lipgloss.Style
	selectedStyle     lipgloss.Style
	dimStyle          lipgloss.Style
	helpStyle         lipgloss.Style
	valueStyle        lipgloss.Style
	editStyle         lipgloss.Style
	checkboxChecked   lipgloss.Style
	checkboxUnchecked lipgloss.Style
	descriptionStyle  lipgloss.Style
}

// DefaultSettingsStyles returns default settings styles.
func DefaultSettingsStyles() SettingsStyles {
	return SettingsStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		tabBarStyle: lipgloss.NewStyle().
			MarginBottom(1),

		sectionStyle: lipgloss.NewStyle().
			MarginBottom(1),

		sectionTitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginTop(1).
			MarginBottom(0),

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

		checkboxChecked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			Bold(true),

		checkboxUnchecked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true).
			PaddingLeft(4),
	}
}

// NewSettingsModel creates a new settings model.
func NewSettingsModel() *SettingsModel {
	m := &SettingsModel{
		tabNavigator:    NewSettingsTabNavigator(),
		content:         DefaultSettingsContent(),
		sectionIndex:    0,
		itemIndex:       0,
		styles:          DefaultSettingsStyles(),
		editing:         false,
		editValue:       "",
		selectionActive: false,
		selectionCursor: 0,
		messageChan:     make(chan tea.Msg, 10),
	}
	
	// Initialize provider list
	m.providerList = m.getAvailableProviders()
	
	return m
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
	case tea.KeyEsc, tea.KeyCtrlC:
		m.goBack = true
		return m, nil

	case tea.KeyTab, tea.KeyRight:
		// Next tab
		m.tabNavigator.NextTab()
		m.resetNavigation()
		return m, nil

	case tea.KeyShiftTab, tea.KeyLeft:
		// Previous tab
		m.tabNavigator.PrevTab()
		m.resetNavigation()
		return m, nil

	case tea.KeyUp:
		m.moveUp()
		return m, nil

	case tea.KeyDown:
		m.moveDown()
		return m, nil

	case tea.KeyEnter:
		return m.handleEnter()

	case tea.KeyRunes:
		switch msg.String() {
		case "q", "Q":
			m.goBack = true
			return m, nil
		case "e", "E":
			return m.handleEnter()
		case " ":
			return m.handleSpace()
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
		m.editingItem = nil
		return m, nil

	case tea.KeyEnter:
		// Save the value
		if m.editingItem != nil {
			m.editingItem.SetValue(m.editValue)
		}
		m.editing = false
		m.editValue = ""
		m.editingItem = nil
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

// handleEnter handles the Enter key press.
func (m *SettingsModel) handleEnter() (tea.Model, tea.Cmd) {
	item := m.getCurrentItem()
	if item == nil {
		return m, nil
	}

	switch item.Type {
	case SettingsItemTypeCheckbox:
		item.ToggleBool()
		m.saveSetting(item)
	case SettingsItemTypeSelect:
		// Enter selection mode for provider/model
		if item.Key == "provider" {
			m.startProviderSelection()
		} else if item.Key == "model" || item.Key == "planModel" {
			m.startModelSelection()
		} else {
			item.CycleOption()
			m.saveSetting(item)
		}
	case SettingsItemTypeText, SettingsItemTypePassword:
		m.editing = true
		m.editValue = item.GetStringValue()
		m.editingItem = item
	case SettingsItemTypeAction:
		// Handle action
		m.handleAction(item)
	}

	return m, nil
}

// handleSpace handles the Space key press.
func (m *SettingsModel) handleSpace() (tea.Model, tea.Cmd) {
	item := m.getCurrentItem()
	if item == nil {
		return m, nil
	}

	if item.Type == SettingsItemTypeCheckbox {
		item.ToggleBool()
	}

	return m, nil
}

// handleAction handles action items.
func (m *SettingsModel) handleAction(item *SettingsItem) {
	switch item.Key {
	case "connectAccount":
		// Trigger account connection flow
		item.Value = "Connecting..."
	}
}

// startProviderSelection enters provider selection mode
func (m *SettingsModel) startProviderSelection() {
	m.isPickingProvider = true
	m.selectionActive = true
	m.selectionCursor = 0
	
	// Find current provider index
	currentProvider, _ := m.GetSetting("provider")
	if currentProviderStr, ok := currentProvider.(string); ok {
		for i, p := range m.providerList {
			if p == currentProviderStr {
				m.selectionCursor = i
				break
			}
		}
	}
}

// startModelSelection enters model selection mode
func (m *SettingsModel) startModelSelection() {
	m.isPickingModel = true
	m.selectionActive = true
	m.selectionCursor = 0
	
	// Get models for current provider
	provider, _ := m.GetSetting("provider")
	providerStr, _ := provider.(string)
	m.modelList = m.getAvailableModels(providerStr)
	
	// Find current model index
	currentModel, _ := m.GetSetting("model")
	if currentModelStr, ok := currentModel.(string); ok {
		for i, model := range m.modelList {
			if model == currentModelStr {
				m.selectionCursor = i
				break
			}
		}
	}
}

// getAvailableProviders returns list of available API providers
func (m *SettingsModel) getAvailableProviders() []string {
	if m.persistence != nil {
		return m.persistence.GetAvailableProviders()
	}
	return []string{"cline", "openai", "anthropic", "openrouter", "bedrock", "ollama", "lmstudio"}
}

// getAvailableModels returns list of available models for a provider
func (m *SettingsModel) getAvailableModels(provider string) []string {
	if m.persistence != nil {
		return m.persistence.GetAvailableModels(provider)
	}
	// Default model lists
	switch provider {
	case "anthropic":
		return []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-3-5-sonnet-20241022"}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}
	case "cline":
		return []string{"claude-sonnet-4", "claude-opus-4"}
	default:
		return []string{"default"}
	}
}

// saveSetting saves a single setting to persistence
func (m *SettingsModel) saveSetting(item *SettingsItem) {
	if m.persistence != nil && m.storageCtx != nil {
		// The persistence layer will handle saving to storage
		// For now, we just mark that settings have changed
	}
}

// moveUp moves the selection up.
func (m *SettingsModel) moveUp() {
	if m.itemIndex > 0 {
		m.itemIndex--
	} else if m.sectionIndex > 0 {
		m.sectionIndex--
		// Move to last item of previous section
		sections := m.content.GetSectionsForTab(m.tabNavigator.CurrentTab())
		if m.sectionIndex < len(sections) {
			m.itemIndex = len(sections[m.sectionIndex].Items) - 1
		}
	}
}

// moveDown moves the selection down.
func (m *SettingsModel) moveDown() {
	sections := m.content.GetSectionsForTab(m.tabNavigator.CurrentTab())
	if m.sectionIndex >= len(sections) {
		return
	}

	section := sections[m.sectionIndex]
	if m.itemIndex < len(section.Items)-1 {
		m.itemIndex++
	} else if m.sectionIndex < len(sections)-1 {
		m.sectionIndex++
		m.itemIndex = 0
	}
}

// resetNavigation resets navigation state when changing tabs.
func (m *SettingsModel) resetNavigation() {
	m.sectionIndex = 0
	m.itemIndex = 0
	m.editing = false
	m.editValue = ""
	m.editingItem = nil
}

// getCurrentItem returns the currently selected item.
func (m *SettingsModel) getCurrentItem() *SettingsItem {
	sections := m.content.GetSectionsForTab(m.tabNavigator.CurrentTab())
	if m.sectionIndex >= len(sections) {
		return nil
	}

	section := sections[m.sectionIndex]
	if m.itemIndex >= len(section.Items) {
		return nil
	}

	return &section.Items[m.itemIndex]
}

// View renders the settings screen.
func (m SettingsModel) View() string {
	var content strings.Builder

	// Title
	content.WriteString(m.styles.titleStyle.Render("Settings"))
	content.WriteString("\n\n")

	// Tab bar
	content.WriteString(m.tabNavigator.Render())
	content.WriteString("\n")

	// Current tab description
	tabInfo := GetSettingsTabInfo(m.tabNavigator.CurrentTab())
	content.WriteString(m.styles.dimStyle.Render(tabInfo.Description))
	content.WriteString("\n\n")

	// Sections and items
	sections := m.content.GetSectionsForTab(m.tabNavigator.CurrentTab())
	for i, section := range sections {
		content.WriteString(m.renderSection(section, i))
	}

	// Edit mode
	if m.editing && m.editingItem != nil {
		content.WriteString("\n")
		content.WriteString(m.styles.editStyle.Render(m.editValue + "▌"))
		content.WriteString("\n")
		content.WriteString(m.styles.helpStyle.Render("Enter to save • Esc to cancel"))
	} else {
		// Help
		content.WriteString("\n")
		content.WriteString(m.styles.helpStyle.Render("↑↓ navigate • Tab switch tabs • Enter/E edit • Space toggle • Esc/q back"))
	}

	return m.styles.containerStyle.Render(content.String())
}

// renderSection renders a settings section.
func (m SettingsModel) renderSection(section SettingsSection, sectionIdx int) string {
	var content strings.Builder

	// Section title
	content.WriteString(m.styles.sectionTitleStyle.Render(section.Title))
	content.WriteString("\n")

	// Items
	for itemIdx, item := range section.Items {
		isSelected := sectionIdx == m.sectionIndex && itemIdx == m.itemIndex
		line := m.renderItem(&item, isSelected)

		if isSelected {
			content.WriteString(m.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(m.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")

		// Show description for selected item
		if isSelected && item.Description != "" {
			content.WriteString(m.styles.descriptionStyle.Render(item.Description))
			content.WriteString("\n")
		}
	}

	return m.styles.sectionStyle.Render(content.String())
}

// renderItem renders a single settings item.
func (m *SettingsModel) renderItem(item *SettingsItem, selected bool) string {
	var parts []string

	// Indent for sub-items
	indent := ""
	if item.IsSubItem {
		indent = "  "
	}

	// Checkbox indicator
	if item.Type == SettingsItemTypeCheckbox {
		if item.GetBoolValue() {
			parts = append(parts, indent+m.styles.checkboxChecked.Render("[✓]"))
		} else {
			parts = append(parts, indent+m.styles.checkboxUnchecked.Render("[ ]"))
		}
	}

	// Label
	parts = append(parts, item.Label)

	// Value
	if item.Type != SettingsItemTypeCheckbox && item.Type != SettingsItemTypeAction {
		valueStr := m.formatValue(item)
		parts = append(parts, m.styles.valueStyle.Render(valueStr))
	} else if item.Type == SettingsItemTypeAction {
		parts = append(parts, m.styles.valueStyle.Render("["+item.GetStringValue()+"]"))
	}

	// Editable indicator
	if item.Editable && !selected && !m.editing {
		parts = append(parts, m.styles.dimStyle.Render("[e]"))
	}

	return strings.Join(parts, " ")
}

// formatValue formats a setting value for display.
func (m *SettingsModel) formatValue(item *SettingsItem) string {
	switch item.Type {
	case SettingsItemTypeCheckbox:
		if item.GetBoolValue() {
			return "✓"
		}
		return "✗"
	case SettingsItemTypePassword:
		if item.GetStringValue() != "" {
			return "********"
		}
		return "[not set]"
	case SettingsItemTypeSelect:
		return item.GetStringValue()
	default:
		value := item.GetStringValue()
		if value == "" {
			return "[not set]"
		}
		// Truncate long values
		if len(value) > 30 {
			return value[:27] + "..."
		}
		return value
	}
}

// ShouldGoBack returns true if the user wants to go back.
func (m *SettingsModel) ShouldGoBack() bool {
	return m.goBack
}

// Reset resets the model state.
func (m *SettingsModel) Reset() {
	m.sectionIndex = 0
	m.itemIndex = 0
	m.goBack = false
	m.editing = false
	m.editValue = ""
	m.editingItem = nil
	m.tabNavigator.SetTab(SettingsTabAPI)
}

// SetStorageContext loads settings from the storage context.
func (m *SettingsModel) SetStorageContext(storageCtx *storage.StorageContext) {
	m.storageCtx = storageCtx
	if storageCtx == nil {
		return
	}

	// Initialize persistence layer
	m.persistence = NewSettingsPersistence(storageCtx)
	
	// Load settings from storage
	if err := m.persistence.LoadSettings(m.content); err != nil {
		// Log error but continue with defaults
		fmt.Printf("Warning: failed to load settings: %v\n", err)
	}
	
	// Update provider and model lists based on loaded settings
	m.providerList = m.getAvailableProviders()
	if provider, ok := m.GetSetting("provider"); ok {
		if providerStr, ok := provider.(string); ok {
			m.modelList = m.getAvailableModels(providerStr)
		}
	}
}

// SaveToStorage saves the current settings to storage.
func (m *SettingsModel) SaveToStorage(storageCtx ...*storage.StorageContext) error {
	ctx := m.storageCtx
	if len(storageCtx) > 0 {
		ctx = storageCtx[0]
	}

	if ctx == nil {
		return fmt.Errorf("storage context is nil")
	}

	// Iterate through all settings and save to storage
	// Implementation depends on the storage structure

	return nil
}

// LoadFromStorage loads settings from storage.
func (m *SettingsModel) LoadFromStorage(storageCtx ...*storage.StorageContext) {
	ctx := m.storageCtx
	if len(storageCtx) > 0 {
		ctx = storageCtx[0]
		m.storageCtx = ctx
	}

	if ctx == nil {
		return
	}

	// Iterate through all settings and load values from storage
	// Implementation depends on the storage structure
}

// GetSettings returns a map of all settings values.
func (m *SettingsModel) GetSettings() map[string]interface{} {
	settings := make(map[string]interface{})

	// Search through all tabs
	allSections := [][]SettingsSection{
		m.content.API,
		m.content.AutoApprove,
		m.content.Features,
		m.content.Account,
		m.content.Other,
	}

	for _, sections := range allSections {
		for _, section := range sections {
			for i := range section.Items {
				settings[section.Items[i].Key] = section.Items[i].Value
			}
		}
	}

	return settings
}

// GetSetting returns a setting value by key.
func (m *SettingsModel) GetSetting(key string) (interface{}, bool) {
	// Search through all tabs
	allSections := [][]SettingsSection{
		m.content.API,
		m.content.AutoApprove,
		m.content.Features,
		m.content.Account,
		m.content.Other,
	}

	for _, sections := range allSections {
		for _, section := range sections {
			for i := range section.Items {
				if section.Items[i].Key == key {
					return section.Items[i].Value, true
				}
			}
		}
	}

	return nil, false
}

// SetSetting sets a setting value by key.
func (m *SettingsModel) SetSetting(key string, value interface{}) bool {
	// Search through all tabs
	allSections := [][]SettingsSection{
		m.content.API,
		m.content.AutoApprove,
		m.content.Features,
		m.content.Account,
		m.content.Other,
	}

	for _, sections := range allSections {
		for _, section := range sections {
			for i := range section.Items {
				if section.Items[i].Key == key {
					section.Items[i].Value = value
					return true
				}
			}
		}
	}

	return false
}
