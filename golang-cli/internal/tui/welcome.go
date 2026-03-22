// Package tui provides a welcome screen component for the Cline CLI.
// This implements the Welcome Screen and Onboarding feature with support
// for first-time user detection, styled banners, quick start hints,
// recent tasks display, and authentication status.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// WelcomeState represents the current state of the welcome screen.
type WelcomeState int

const (
	// WelcomeStateInitial is the initial loading state.
	WelcomeStateInitial WelcomeState = iota
	// WelcomeStateDisplaying shows the welcome screen.
	WelcomeStateDisplaying
	// WelcomeStateCompleted indicates the user continued past the welcome.
	WelcomeStateCompleted
	// WelcomeStateSkipped indicates the user skipped the welcome.
	WelcomeStateSkipped
)

// WelcomeConfig configures the welcome screen behavior.
type WelcomeConfig struct {
	// Version is the CLI version to display.
	Version string
	// MinWidth is the minimum terminal width required.
	MinWidth int
	// MinHeight is the minimum terminal height required.
	MinHeight int
	// MaxRecentTasks is the maximum number of recent tasks to display.
	MaxRecentTasks int
	// StorageContext provides access to persistent storage.
	StorageContext *storage.StorageContext
	// SkipWelcome forces skipping the welcome screen.
	SkipWelcome bool
}

// DefaultWelcomeConfig returns a default welcome configuration.
func DefaultWelcomeConfig() WelcomeConfig {
	return WelcomeConfig{
		Version:        "0.1.0",
		MinWidth:       60,
		MinHeight:      20,
		MaxRecentTasks: 5,
	}
}

// WelcomeKeyMap defines key bindings for the welcome screen.
type WelcomeKeyMap struct {
	Continue key.Binding
	Skip     key.Binding
	Quit     key.Binding
}

// DefaultWelcomeKeyMap returns the default key bindings.
func DefaultWelcomeKeyMap() WelcomeKeyMap {
	return WelcomeKeyMap{
		Continue: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		Skip: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "skip welcome in future"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
	}
}

// RecentTask represents a recently executed task.
type RecentTask struct {
	ID        string
	Task      string
	Timestamp int64
}

// AuthStatus represents the current authentication status.
type AuthStatus struct {
	Provider        string
	IsAuthenticated bool
	Model           string
}

// WelcomeResult contains the outcome of showing the welcome screen.
type WelcomeResult struct {
	State        WelcomeState
	WasSkipped   bool
	WasFirstTime bool
}

// WelcomeModel is the Bubble Tea model for the welcome screen.
type WelcomeModel struct {
	config     WelcomeConfig
	keyMap     WelcomeKeyMap
	state      WelcomeState
	width      int
	height     int
	recentTasks []RecentTask
	authStatus AuthStatus
	isFirstTime bool
}

// NewWelcomeModel creates a new welcome model with the given configuration.
func NewWelcomeModel(config WelcomeConfig) *WelcomeModel {
	return &WelcomeModel{
		config:      config,
		keyMap:      DefaultWelcomeKeyMap(),
		state:       WelcomeStateInitial,
		width:       80,
		height:      24,
		recentTasks: make([]RecentTask, 0),
	}
}

// Init implements tea.Model.
func (m *WelcomeModel) Init() tea.Cmd {
	return m.loadDataCmd()
}

// loadDataCmd returns a command that loads data asynchronously.
func (m *WelcomeModel) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		return m.loadData()
	}
}

// welcomeDataLoadedMsg is sent when data loading completes.
type welcomeDataLoadedMsg struct {
	skipWelcome bool
}

// loadData loads authentication status and recent tasks.
func (m *WelcomeModel) loadData() tea.Msg {
	// Check if welcome should be skipped
	if m.config.SkipWelcome {
		return welcomeDataLoadedMsg{skipWelcome: true}
	}

	// Check if welcome was already shown or disabled
	if m.config.StorageContext != nil {
		if val, ok := m.config.StorageContext.GlobalState.Get("welcomeShown"); ok {
			if shown, ok := val.(bool); ok && shown {
				return welcomeDataLoadedMsg{skipWelcome: true}
			}
		}

		if val, ok := m.config.StorageContext.GlobalState.Get("welcomeDisabled"); ok {
			if disabled, ok := val.(bool); ok && disabled {
				return welcomeDataLoadedMsg{skipWelcome: true}
			}
		}

		// Load auth status
		m.authStatus = m.loadAuthStatus()

		// Load recent tasks
		m.recentTasks = m.loadRecentTasks()

		// Check if this is a first-time user
		m.isFirstTime = m.isFirstTimeUser()
	}

	return welcomeDataLoadedMsg{skipWelcome: false}
}

// isFirstTimeUser determines if this is the user's first time.
func (m *WelcomeModel) isFirstTimeUser() bool {
	if m.config.StorageContext == nil {
		return true
	}

	// Check for any existing configuration
	keys := []string{"apiProvider", "defaultModel", "welcomeShown"}
	for _, key := range keys {
		if _, ok := m.config.StorageContext.GlobalState.Get(key); ok {
			return false
		}
	}

	return true
}

// loadAuthStatus loads the current authentication status from storage.
func (m *WelcomeModel) loadAuthStatus() AuthStatus {
	status := AuthStatus{
		IsAuthenticated: false,
	}

	if m.config.StorageContext == nil {
		return status
	}

	providerVal, ok := m.config.StorageContext.GlobalState.Get("apiProvider")
	if !ok {
		return status
	}

	provider, ok := providerVal.(string)
	if !ok {
		return status
	}

	status.Provider = provider

	// Get model
	if modelVal, ok := m.config.StorageContext.GlobalState.Get("defaultModel"); ok {
		if model, ok := modelVal.(string); ok {
			status.Model = model
		}
	}

	// Check if API key exists
	apiKeyVal, ok := m.config.StorageContext.Secrets.Get(provider + "ApiKey")
	if ok && apiKeyVal != nil {
		if key, ok := apiKeyVal.(string); ok && key != "" {
			status.IsAuthenticated = true
		}
	}

	return status
}

// getTaskHistoryPath returns the path to the task history file.
func getTaskHistoryPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "taskHistory.json"
	}
	return filepath.Join(homeDir, ".cline", "data", "taskHistory.json")
}

// loadRecentTasks loads recent tasks from storage.
func (m *WelcomeModel) loadRecentTasks() []RecentTask {
	tasks := make([]RecentTask, 0)

	// Get task history path
	historyPath := getTaskHistoryPath()

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return tasks
	}

	// Create storage instance
	fileStorage, err := storage.NewClineFileStorage(historyPath, 0644)
	if err != nil {
		return tasks
	}
	defer fileStorage.Close()

	// Try to get entries
	val, ok := fileStorage.Get("entries")
	if !ok {
		return tasks
	}

	// Handle different data structures
	var entries []interface{}
	switch v := val.(type) {
	case []interface{}:
		entries = v
	default:
		// Try to convert through JSON
		return tasks
	}

	// Convert entries to RecentTask
	for _, entry := range entries {
		if task, ok := convertToRecentTask(entry); ok {
			tasks = append(tasks, task)
		}
	}

	// Sort by timestamp (newest first) and limit
	return sortAndLimitTasks(tasks, m.config.MaxRecentTasks)
}

// convertToRecentTask converts interface data to a RecentTask.
func convertToRecentTask(data interface{}) (RecentTask, bool) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return RecentTask{}, false
	}

	task := RecentTask{}

	// Extract ID
	if idVal, ok := m["id"]; ok {
		if id, ok := idVal.(string); ok {
			task.ID = id
		}
	}

	// Extract task description
	if taskVal, ok := m["task"]; ok {
		if t, ok := taskVal.(string); ok {
			task.Task = t
		}
	}

	// Extract timestamp
	if tsVal, ok := m["ts"]; ok {
		switch ts := tsVal.(type) {
		case float64:
			task.Timestamp = int64(ts)
		case int64:
			task.Timestamp = ts
		case int:
			task.Timestamp = int64(ts)
		}
	}

	// Require at least an ID or task description
	if task.ID == "" && task.Task == "" {
		return RecentTask{}, false
	}

	return task, true
}

// sortAndLimitTasks sorts tasks by timestamp (newest first) and limits the count.
func sortAndLimitTasks(tasks []RecentTask, limit int) []RecentTask {
	// Sort by timestamp descending
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Timestamp > tasks[j].Timestamp
	})

	// Apply limit
	if limit > 0 && len(tasks) > limit {
		return tasks[:limit]
	}

	return tasks
}

// Update implements tea.Model.
func (m *WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case welcomeDataLoadedMsg:
		if msg.skipWelcome {
			m.state = WelcomeStateCompleted
			return m, tea.Quit
		}
		m.state = WelcomeStateDisplaying

	case tea.KeyMsg:
		// Handle quit
		if key.Matches(msg, m.keyMap.Quit) {
			return m, tea.Quit
		}

		// Handle skip (only in displaying state)
		if m.state == WelcomeStateDisplaying && key.Matches(msg, m.keyMap.Skip) {
			m.state = WelcomeStateSkipped
			m.saveSkipPreference()
			return m, tea.Quit
		}

		// Handle continue (only in displaying state)
		if m.state == WelcomeStateDisplaying && key.Matches(msg, m.keyMap.Continue) {
			m.state = WelcomeStateCompleted
			m.saveWelcomeShown()
			return m, tea.Quit
		}
	}

	return m, nil
}

// saveSkipPreference saves the skip preference and marks welcome as shown.
func (m *WelcomeModel) saveSkipPreference() {
	if m.config.StorageContext == nil {
		return
	}

	_ = m.config.StorageContext.GlobalState.Set("welcomeDisabled", true)
	_ = m.config.StorageContext.GlobalState.Set("welcomeShown", true)
}

// saveWelcomeShown marks the welcome as shown.
func (m *WelcomeModel) saveWelcomeShown() {
	if m.config.StorageContext == nil {
		return
	}

	_ = m.config.StorageContext.GlobalState.Set("welcomeShown", true)
}

// View implements tea.Model.
func (m *WelcomeModel) View() string {
	switch m.state {
	case WelcomeStateInitial:
		return m.renderLoading()

	case WelcomeStateDisplaying:
		// Check terminal size
		if m.width < m.config.MinWidth || m.height < m.config.MinHeight {
			return m.renderTerminalTooSmall()
		}

		var sections []string

		// Banner
		sections = append(sections, m.renderBanner(m.width-4))

		// Auth status
		sections = append(sections, m.renderAuthStatus(m.width-4))

		// Quick start hints
		sections = append(sections, m.renderQuickStartHints(m.width-4))

		// Recent tasks
		sections = append(sections, m.renderRecentTasks(m.width-4))

		// Footer with key bindings
		sections = append(sections, m.renderFooter(m.width-4))

		// Join all sections
		content := strings.Join(sections, "\n\n")

		// Apply padding and border
		return m.renderContainer(content, m.width-2)

	default:
		return ""
	}
}

// renderLoading renders the loading state.
func (m *WelcomeModel) renderLoading() string {
	return "Loading welcome screen..."
}

// renderContainer applies the main container styling.
func (m *WelcomeModel) renderContainer(content string, width int) string {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		Width(width)

	return style.Render(content)
}

// renderBanner renders the welcome banner with styling.
func (m *WelcomeModel) renderBanner(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(width).
		Align(lipgloss.Center)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0")).
		Width(width).
		Align(lipgloss.Center)

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Width(width).
		Align(lipgloss.Center)

	// ASCII art logo
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Width(width).
		Align(lipgloss.Center)

	logo := `
   ___ _ _     
  / __| (_)_ _ 
 | (__| | | '_|
  \___|_|_|_|  
`

	var parts []string
	parts = append(parts, logoStyle.Render(logo))
	parts = append(parts, titleStyle.Render("Welcome to Cline"))
	parts = append(parts, subtitleStyle.Render("Your AI-powered coding assistant"))
	parts = append(parts, versionStyle.Render(fmt.Sprintf("Version %s", m.config.Version)))

	return strings.Join(parts, "\n")
}

// renderAuthStatus renders the authentication status section.
func (m *WelcomeModel) renderAuthStatus(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(width)

	var content string
	if m.authStatus.IsAuthenticated {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4CAF50"))

		content = successStyle.Render(fmt.Sprintf("✓ Connected to %s", m.authStatus.Provider))
		if m.authStatus.Model != "" {
			content += "\n  Model: " + m.authStatus.Model
		}
	} else {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFC107"))

		content = warningStyle.Render("⚠ Not authenticated") + "\n\n"
		content += "Run " + lipgloss.NewStyle().Bold(true).Render("cline auth") +
			" to set up your API provider"
	}

	return titleStyle.Render("Authentication") + "\n" +
		boxStyle.Render(content)
}

// renderQuickStartHints renders quick start hints.
func (m *WelcomeModel) renderQuickStartHints(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(width)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))

	commandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	hints := []struct {
		cmd  string
		desc string
	}{
		{"cline", "Start interactive mode"},
		{"cline \"your prompt\"", "Execute a single task"},
		{"cline auth", "Configure authentication"},
		{"cline history", "View recent tasks"},
	}

	var lines []string
	for _, h := range hints {
		line := fmt.Sprintf("  %s %s %s",
			commandStyle.Render(h.cmd),
			hintStyle.Render(strings.Repeat(".", width-len(h.cmd)-len(h.desc)-6)),
			hintStyle.Render(h.desc))
		lines = append(lines, line)
	}

	return titleStyle.Render("Quick Start") + "\n" +
		boxStyle.Render(strings.Join(lines, "\n"))
}

// renderRecentTasks renders the recent tasks section.
func (m *WelcomeModel) renderRecentTasks(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(width)

	if len(m.recentTasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true)
		return titleStyle.Render("Recent Tasks") + "\n" +
			boxStyle.Render(emptyStyle.Render("No tasks yet. Start by running: cline \"your prompt\""))
	}

	taskStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))

	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Align(lipgloss.Right)

	var lines []string
	for _, task := range m.recentTasks {
		taskText := truncateString(task.Task, width-20)
		timeText := formatRelativeTime(task.Timestamp)

		line := fmt.Sprintf("  %s %s",
			taskStyle.Render(taskText),
			timeStyle.Render(timeText))
		lines = append(lines, line)
	}

	return titleStyle.Render("Recent Tasks") + "\n" +
		boxStyle.Render(strings.Join(lines, "\n"))
}

// renderFooter renders the footer with key bindings.
func (m *WelcomeModel) renderFooter(width int) string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(width).
		Align(lipgloss.Center)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#E0E0E0"))

	keys := []string{
		fmt.Sprintf("%s to continue", keyStyle.Render("enter")),
		fmt.Sprintf("%s to skip welcome in future", keyStyle.Render("s")),
		fmt.Sprintf("%s to quit", keyStyle.Render("q")),
	}

	return footerStyle.Render(strings.Join(keys, "  •  "))
}

// renderTerminalTooSmall renders a message when the terminal is too small.
func (m *WelcomeModel) renderTerminalTooSmall() string {
	warningStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFC107")).
		Width(m.width).
		Align(lipgloss.Center)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(m.width).
		Align(lipgloss.Center)

	return warningStyle.Render("Terminal Too Small") + "\n\n" +
		infoStyle.Render(fmt.Sprintf("Current: %dx%d", m.width, m.height)) + "\n" +
		infoStyle.Render(fmt.Sprintf("Required: %dx%d", m.config.MinWidth, m.config.MinHeight)) + "\n\n" +
		infoStyle.Render("Please resize your terminal and try again.")
}

// truncateString truncates a string to the specified length with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatRelativeTime formats a timestamp as a relative time string.
func formatRelativeTime(timestamp int64) string {
	if timestamp == 0 {
		return "unknown"
	}

	// Handle millisecond timestamps
	if timestamp > 1e12 {
		timestamp = timestamp / 1000
	}

	t := time.Unix(timestamp, 0)
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 02")
	}
}

// Result returns the welcome result.
func (m *WelcomeModel) Result() WelcomeResult {
	return WelcomeResult{
		State:        m.state,
		WasSkipped:   m.state == WelcomeStateSkipped,
		WasFirstTime: m.isFirstTime,
	}
}

// Run executes the welcome model and returns the result.
func (m *WelcomeModel) Run() (WelcomeResult, error) {
	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return WelcomeResult{}, err
	}

	wm, ok := model.(*WelcomeModel)
	if !ok {
		return WelcomeResult{}, fmt.Errorf("unexpected model type")
	}

	return wm.Result(), nil
}

// ShouldShowWelcome checks if the welcome screen should be displayed.
func ShouldShowWelcome(storageCtx *storage.StorageContext) bool {
	if storageCtx == nil {
		return true
	}

	// Check if welcome was already shown
	if val, ok := storageCtx.GlobalState.Get("welcomeShown"); ok {
		if shown, ok := val.(bool); ok && shown {
			return false
		}
	}

	// Check if welcome is disabled
	if val, ok := storageCtx.GlobalState.Get("welcomeDisabled"); ok {
		if disabled, ok := val.(bool); ok && disabled {
			return false
		}
	}

	return true
}

// ResetWelcomeFlags resets the welcome flags for testing or re-onboarding.
func ResetWelcomeFlags(storageCtx *storage.StorageContext) error {
	if storageCtx == nil {
		return nil
	}

	if err := storageCtx.GlobalState.Delete("welcomeShown"); err != nil {
		return err
	}

	if err := storageCtx.GlobalState.Delete("welcomeDisabled"); err != nil {
		return err
	}

	return nil
}

// ShowWelcome displays the welcome screen and returns the result.
// This is a convenience function for simple use cases.
func ShowWelcome(storageCtx *storage.StorageContext, version string) (WelcomeResult, error) {
	if !ShouldShowWelcome(storageCtx) {
		return WelcomeResult{
			State:        WelcomeStateCompleted,
			WasSkipped:   true,
			WasFirstTime: false,
		}, nil
	}

	config := DefaultWelcomeConfig()
	config.StorageContext = storageCtx
	config.Version = version

	model := NewWelcomeModel(config)
	return model.Run()
}