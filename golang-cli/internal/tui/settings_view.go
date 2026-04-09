// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SettingsView handles rendering of the settings panel.
type SettingsView struct {
	styles SettingsStyles
}

// NewSettingsView creates a new settings view.
func NewSettingsView() *SettingsView {
	return &SettingsView{
		styles: DefaultSettingsStyles(),
	}
}

// SetStyles sets custom styles for the view.
func (v *SettingsView) SetStyles(styles SettingsStyles) {
	v.styles = styles
}

// Render renders the complete settings screen.
func (v *SettingsView) Render(m *SettingsModel) string {
	var content strings.Builder

	// Title
	content.WriteString(v.styles.titleStyle.Render("Settings"))
	content.WriteString("\n\n")

	// Tab bar
	if m.tabNavigator != nil {
		content.WriteString(m.tabNavigator.Render())
		content.WriteString("\n")
	}

	// Current tab description
	tabInfo := GetSettingsTabInfo(m.tabNavigator.CurrentTab())
	content.WriteString(v.styles.dimStyle.Render(tabInfo.Description))
	content.WriteString("\n\n")

	// Sections and items
	sections := m.content.GetSectionsForTab(m.tabNavigator.CurrentTab())
	for i, section := range sections {
		content.WriteString(v.renderSection(section, i, m.sectionIndex, m.itemIndex))
	}

	// Edit mode or help
	if m.editing && m.editingItem != nil {
		content.WriteString("\n")
		content.WriteString(v.styles.editStyle.Render(m.editValue + "▌"))
		content.WriteString("\n")
		content.WriteString(v.styles.helpStyle.Render("Enter to save • Esc to cancel"))
	} else {
		// Help
		content.WriteString("\n")
		content.WriteString(v.styles.helpStyle.Render("↑↓ navigate • Tab switch tabs • Enter/E edit • Space toggle • Esc/q back"))
	}

	return v.styles.containerStyle.Render(content.String())
}

// renderSection renders a settings section.
func (v *SettingsView) renderSection(section SettingsSection, sectionIdx, currentSectionIdx, currentItemIdx int) string {
	var content strings.Builder

	// Section title
	content.WriteString(v.styles.sectionTitleStyle.Render(section.Title))
	content.WriteString("\n")

	// Items
	for itemIdx, item := range section.Items {
		isSelected := sectionIdx == currentSectionIdx && itemIdx == currentItemIdx
		line := v.renderItem(&item, isSelected)

		if isSelected {
			content.WriteString(v.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(v.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")

		// Show description for selected item
		if isSelected && item.Description != "" {
			content.WriteString(v.styles.descriptionStyle.Render(item.Description))
			content.WriteString("\n")
		}
	}

	return v.styles.sectionStyle.Render(content.String())
}

// renderItem renders a single settings item.
func (v *SettingsView) renderItem(item *SettingsItem, selected bool) string {
	var parts []string

	// Indent for sub-items
	indent := ""
	if item.IsSubItem {
		indent = "  "
	}

	// Checkbox indicator
	if item.Type == SettingsItemTypeCheckbox {
		if item.GetBoolValue() {
			parts = append(parts, indent+v.styles.checkboxChecked.Render("[✓]"))
		} else {
			parts = append(parts, indent+v.styles.checkboxUnchecked.Render("[ ]"))
		}
	}

	// Label
	parts = append(parts, item.Label)

	// Value
	if item.Type != SettingsItemTypeCheckbox && item.Type != SettingsItemTypeAction {
		valueStr := v.formatValue(item)
		parts = append(parts, v.styles.valueStyle.Render(valueStr))
	} else if item.Type == SettingsItemTypeAction {
		parts = append(parts, v.styles.valueStyle.Render("["+item.GetStringValue()+"]"))
	}

	// Editable indicator
	if item.Editable && !selected {
		parts = append(parts, v.styles.dimStyle.Render("[e]"))
	}

	return strings.Join(parts, " ")
}

// formatValue formats a setting value for display.
func (v *SettingsView) formatValue(item *SettingsItem) string {
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

// RenderProviderSelection renders the provider selection dialog.
func (v *SettingsView) RenderProviderSelection(providers []string, cursor int, currentProvider string) string {
	var content strings.Builder

	content.WriteString(v.styles.titleStyle.Render("Select API Provider"))
	content.WriteString("\n\n")

	for i, provider := range providers {
		prefix := "  "
		if i == cursor {
			prefix = "▶ "
		}

		line := prefix + provider
		if provider == currentProvider {
			line += " (current)"
		}

		if i == cursor {
			content.WriteString(v.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(v.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(v.styles.helpStyle.Render("↑↓ navigate • Enter select • Esc cancel"))

	return v.styles.containerStyle.Render(content.String())
}

// RenderModelSelection renders the model selection dialog.
func (v *SettingsView) RenderModelSelection(models []string, cursor int, currentModel string) string {
	var content strings.Builder

	content.WriteString(v.styles.titleStyle.Render("Select Model"))
	content.WriteString("\n\n")

	for i, model := range models {
		prefix := "  "
		if i == cursor {
			prefix = "▶ "
		}

		line := prefix + model
		if model == currentModel {
			line += " (current)"
		}

		if i == cursor {
			content.WriteString(v.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(v.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(v.styles.helpStyle.Render("↑↓ navigate • Enter select • Esc cancel"))

	return v.styles.containerStyle.Render(content.String())
}

// RenderAPIKeyInput renders the API key input dialog.
func (v *SettingsView) RenderAPIKeyInput(value string, isMasked bool) string {
	var content strings.Builder

	content.WriteString(v.styles.titleStyle.Render("Enter API Key"))
	content.WriteString("\n\n")

	displayValue := value
	if isMasked && value != "" {
		displayValue = strings.Repeat("*", len(value))
	}

	content.WriteString(v.styles.editStyle.Render(displayValue + "▌"))
	content.WriteString("\n\n")

	content.WriteString(v.styles.helpStyle.Render("Enter to save • Esc to cancel"))

	return v.styles.containerStyle.Render(content.String())
}

// RenderSaveMessage renders a save confirmation message.
func (v *SettingsView) RenderSaveMessage(message string, isError bool) string {
	style := v.styles.valueStyle
	if isError {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(ErrorRed))
	}
	return style.Render(message)
}
