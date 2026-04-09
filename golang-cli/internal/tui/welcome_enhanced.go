// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// WelcomeEnhanced extends the welcome screen with additional features.
type WelcomeEnhanced struct {
	// Recent tasks display
	recentTasks    []RecentTask
	showRecent     bool
	maxRecentTasks int

	// First run detection
	isFirstRun     bool
	showOnboarding bool

	// Help overlay
	showHelpOverlay bool

	// Styles
	styles WelcomeEnhancedStyles

	// Parent welcome model reference
	welcomeModel *WelcomeModel
}

// RecentTask represents a recent task to display on the welcome screen.
type RecentTask struct {
	ID          string
	Description string
	Timestamp   time.Time
}

// WelcomeEnhancedStyles holds styles for the enhanced welcome screen.
type WelcomeEnhancedStyles struct {
	recentTasksStyle    lipgloss.Style
	recentTaskItemStyle lipgloss.Style
	onboardingStyle     lipgloss.Style
	helpOverlayStyle    lipgloss.Style
	shortcutHintStyle   lipgloss.Style
	firstRunBadgeStyle  lipgloss.Style
}

// DefaultWelcomeEnhancedStyles returns default styles.
func DefaultWelcomeEnhancedStyles() WelcomeEnhancedStyles {
	return WelcomeEnhancedStyles{
		recentTasksStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),

		recentTaskItemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			PaddingLeft(4),

		onboardingStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(PrimaryBlue)).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1),

		helpOverlayStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(SelectionBlue)).
			Padding(1, 2).
			Background(lipgloss.Color(DarkBackground)),

		shortcutHintStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true),

		firstRunBadgeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			Padding(0, 1),
	}
}

// NewWelcomeEnhanced creates a new enhanced welcome screen component.
func NewWelcomeEnhanced(welcomeModel *WelcomeModel) *WelcomeEnhanced {
	return &WelcomeEnhanced{
		welcomeModel:   welcomeModel,
		styles:         DefaultWelcomeEnhancedStyles(),
		recentTasks:    make([]RecentTask, 0),
		maxRecentTasks: 5,
		showRecent:     true,
	}
}

// SetRecentTasks sets the list of recent tasks to display.
func (we *WelcomeEnhanced) SetRecentTasks(tasks []RecentTask) {
	we.recentTasks = tasks
	if len(we.recentTasks) > we.maxRecentTasks {
		we.recentTasks = we.recentTasks[:we.maxRecentTasks]
	}
}

// SetRecentTasksFromHistory converts HistoryItems to RecentTasks.
func (we *WelcomeEnhanced) SetRecentTasksFromHistory(items []HistoryItem) {
	tasks := make([]RecentTask, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, RecentTask{
			ID:          item.ID,
			Description: item.Task,
			Timestamp:   item.Timestamp,
		})
	}
	we.SetRecentTasks(tasks)
}

// SetFirstRun sets whether this is the first run.
func (we *WelcomeEnhanced) SetFirstRun(isFirstRun bool) {
	we.isFirstRun = isFirstRun
	we.showOnboarding = isFirstRun
}

// ShowRecentTasks enables/disables recent tasks display.
func (we *WelcomeEnhanced) ShowRecentTasks(show bool) {
	we.showRecent = show
}

// ShowHelpOverlay shows/hides the help overlay.
func (we *WelcomeEnhanced) ShowHelpOverlay(show bool) {
	we.showHelpOverlay = show
}

// IsHelpOverlayVisible returns whether the help overlay is visible.
func (we *WelcomeEnhanced) IsHelpOverlayVisible() bool {
	return we.showHelpOverlay
}

// ToggleHelpOverlay toggles the help overlay.
func (we *WelcomeEnhanced) ToggleHelpOverlay() {
	we.showHelpOverlay = !we.showHelpOverlay
}

// DismissOnboarding dismisses the first-run onboarding.
func (we *WelcomeEnhanced) DismissOnboarding() {
	we.showOnboarding = false
}

// RenderRecentTasks renders the recent tasks section.
func (we *WelcomeEnhanced) RenderRecentTasks() string {
	if !we.showRecent || len(we.recentTasks) == 0 {
		return ""
	}

	var content strings.Builder

	content.WriteString(we.styles.recentTasksStyle.Render("Recent Tasks"))
	content.WriteString("\n")

	for i, task := range we.recentTasks {
		// Number the task
		number := fmt.Sprintf("%d.", i+1)

		// Truncate description
		desc := task.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		// Format timestamp
		timeStr := we.formatRelativeTime(task.Timestamp)

		line := fmt.Sprintf("%s %s (%s)", number, desc, timeStr)
		content.WriteString(we.styles.recentTaskItemStyle.Render(line))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(we.styles.shortcutHintStyle.Render("Press 1-5 to quickly resume a task"))

	return content.String()
}

// RenderOnboarding renders the first-run onboarding hint.
func (we *WelcomeEnhanced) RenderOnboarding() string {
	if !we.showOnboarding {
		return ""
	}

	var content strings.Builder

	content.WriteString(we.styles.onboardingStyle.Render(
		"👋 Welcome to Cline!\n\n" +
			"Get started by selecting 'New Task' or configure your API provider in Settings.\n" +
			"Press ? anytime for help.",
	))

	return content.String()
}

// RenderHelpOverlay renders the keyboard shortcuts help overlay.
func (we *WelcomeEnhanced) RenderHelpOverlay(width int) string {
	if !we.showHelpOverlay {
		return ""
	}

	var content strings.Builder

	content.WriteString("Keyboard Shortcuts\n")
	content.WriteString(strings.Repeat("─", width-4))
	content.WriteString("\n\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"↑/↓ or j/k", "Navigate menu"},
		{"Enter", "Select highlighted item"},
		{"n", "New task"},
		{"c", "Continue last task"},
		{"h", "Task history"},
		{"s", "Settings"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
		{"", ""},
		{"Tab", "Switch settings tabs"},
		{"Esc", "Go back / Cancel"},
		{"Space", "Toggle checkbox / Preview"},
		{"e", "Edit setting"},
		{"/", "Search in history"},
	}

	for _, shortcut := range shortcuts {
		if shortcut.key == "" {
			content.WriteString("\n")
			continue
		}
		keyStr := lipgloss.NewStyle().Foreground(lipgloss.Color(PrimaryBlue)).Bold(true).Render(shortcut.key)
		content.WriteString(fmt.Sprintf("  %-12s %s\n", keyStr, shortcut.desc))
	}

	content.WriteString("\n")
	content.WriteString(we.styles.shortcutHintStyle.Render("Press ? or Esc to close"))

	return we.styles.helpOverlayStyle.Render(content.String())
}

// RenderFirstRunBadge renders a first-run indicator badge.
func (we *WelcomeEnhanced) RenderFirstRunBadge() string {
	if !we.isFirstRun {
		return ""
	}
	return we.styles.firstRunBadgeStyle.Render(" First Run ")
}

// RenderEnhancedHeader renders an enhanced header with first-run indicator.
func (we *WelcomeEnhanced) RenderEnhancedHeader() string {
	var content strings.Builder

	// ASCII art logo
	logo := `
   ____ _     ___ _   _ 
  / ___| |   |_ _| \ | |
 | |   | |    | ||  \| |
 | |___| |___ | || |\  |
  \____|_____|___|_| \_|
`
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(PrimaryBlue)).Bold(true).Render(logo))
	content.WriteString("\n")

	// Subtitle with optional first-run badge
	subtitle := "Your AI coding assistant"
	if we.isFirstRun {
		subtitle += " " + we.RenderFirstRunBadge()
	}
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(Gray)).Render(subtitle))

	return content.String()
}

// RenderShortcutsHelp renders a compact shortcuts reference.
func (we *WelcomeEnhanced) RenderShortcutsHelp() string {
	hints := []string{
		"n:new  c:continue  h:history  s:settings  ?:help  q:quit",
	}
	return we.styles.shortcutHintStyle.Render(strings.Join(hints, " • "))
}

// formatRelativeTime formats a timestamp as a relative time string.
func (we *WelcomeEnhanced) formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// HasRecentTasks returns true if there are recent tasks to display.
func (we *WelcomeEnhanced) HasRecentTasks() bool {
	return len(we.recentTasks) > 0
}

// GetRecentTaskAt returns the recent task at the given index (1-based).
func (we *WelcomeEnhanced) GetRecentTaskAt(index int) (RecentTask, bool) {
	if index < 1 || index > len(we.recentTasks) {
		return RecentTask{}, false
	}
	return we.recentTasks[index-1], true
}

// SetMaxRecentTasks sets the maximum number of recent tasks to display.
func (we *WelcomeEnhanced) SetMaxRecentTasks(max int) {
	we.maxRecentTasks = max
	if len(we.recentTasks) > max {
		we.recentTasks = we.recentTasks[:max]
	}
}

// GetMaxRecentTasks returns the maximum number of recent tasks.
func (we *WelcomeEnhanced) GetMaxRecentTasks() int {
	return we.maxRecentTasks
}

// IsFirstRun returns true if this is the first run.
func (we *WelcomeEnhanced) IsFirstRun() bool {
	return we.isFirstRun
}

// ShouldShowOnboarding returns true if onboarding should be shown.
func (we *WelcomeEnhanced) ShouldShowOnboarding() bool {
	return we.showOnboarding
}

// CheckFirstRun checks if this is the first run by looking for settings.
func (we *WelcomeEnhanced) CheckFirstRun(hasSettings bool) {
	we.isFirstRun = !hasSettings
	we.showOnboarding = !hasSettings
}

// WelcomeEnhancedMsg is sent when an enhanced welcome action occurs.
type WelcomeEnhancedMsg struct {
	Type      string
	TaskID    string
	Dismissed bool
}

// SelectRecentTaskMsg is sent when a recent task is selected.
type SelectRecentTaskMsg struct {
	TaskID      string
	Description string
}

// ToSelectRecentTaskMsg converts a recent task selection to a message.
func (we *WelcomeEnhanced) ToSelectRecentTaskMsg(index int) (SelectRecentTaskMsg, bool) {
	if task, ok := we.GetRecentTaskAt(index); ok {
		return SelectRecentTaskMsg{
			TaskID:      task.ID,
			Description: task.Description,
		}, true
	}
	return SelectRecentTaskMsg{}, false
}
