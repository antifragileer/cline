// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// AppModel represents the top-level application state managing multiple sub-models.
type AppModel struct {
	// Dependencies
	client  *host.Client
	storage *storage.StorageContext
	config  *config.LayeredConfig

	// Sub-models
	welcome  *WelcomeModel
	history  *HistoryModel
	settings *SettingsModel
	chat     *ChatModel
	diff     DiffModel

	// Current view state
	state AppState

	// Metadata
	width   int
	height  int
	mode    string // "act" or "plan"
	yolo    bool
	quitting bool

	// Program reference for sending messages
	program *tea.Program
}

// AppState represents the current application view.
type AppState int

const (
	// AppStateWelcome shows the welcome screen.
	AppStateWelcome AppState = iota
	// AppStateChat shows the chat interface.
	AppStateChat
	// AppStateHistory shows the task history.
	AppStateHistory
	// AppStateSettings shows the settings screen.
	AppStateSettings
	// AppStateDiff shows the diff viewer.
	AppStateDiff
)

// NewAppModel creates a new application model with dependencies.
func NewAppModel(client *host.Client, storage *storage.StorageContext, cfg *config.LayeredConfig) *AppModel {
	return &AppModel{
		client:   client,
		storage:  storage,
		config:   cfg,
		welcome:  NewWelcomeModel(),
		history:  NewHistoryModel(),
		settings: NewSettingsModel(),
		chat:     NewChatModel(),
		state:    AppStateWelcome,
		mode:     "act",
		yolo:     false,
	}
}

// Init initializes the model.
func (m AppModel) Init() tea.Cmd {
	// Initialize all sub-models
	return tea.Batch(
		m.welcome.Init(),
		m.history.Init(),
		m.settings.Init(),
		m.chat.Init(),
	)
}

// Update handles messages and updates the appropriate sub-model.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update dimensions for all sub-models
		m.welcome.SetDimensions(msg.Width, msg.Height)
		m.history.SetDimensions(msg.Width, msg.Height)
		m.settings.SetDimensions(msg.Width, msg.Height)
		m.chat.SetDimensions(msg.Width, msg.Height)
		if m.state == AppStateDiff {
			m.diff.SetDimensions(msg.Width, msg.Height)
		}
		return m, nil

	case WelcomeResultMsg:
		// Handle welcome screen result
		switch msg.Action {
		case ActionNewTask:
			m.state = AppStateChat
			if msg.InputValue != "" {
				m.chat.SetInput(msg.InputValue)
			}
		case ActionContinueTask:
			m.state = AppStateChat
		case ActionHistory:
			m.state = AppStateHistory
		case ActionSettings:
			m.state = AppStateSettings
		case ActionQuit:
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		// Global key handlers
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Route to appropriate sub-model based on state
	switch m.state {
	case AppStateWelcome:
		_, cmd := m.welcome.Update(msg)
		cmds = append(cmds, cmd)

	case AppStateHistory:
		_, cmd := m.history.Update(msg)
		cmds = append(cmds, cmd)

	case AppStateSettings:
		_, cmd := m.settings.Update(msg)
		cmds = append(cmds, cmd)

	case AppStateChat:
		_, cmd := m.chat.Update(msg)
		cmds = append(cmds, cmd)

	case AppStateDiff:
		_, cmd := m.diff.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view based on state.
func (m AppModel) View() string {
	switch m.state {
	case AppStateWelcome:
		return m.welcome.View()
	case AppStateHistory:
		return m.history.View()
	case AppStateSettings:
		return m.settings.View()
	case AppStateChat:
		return m.chat.View()
	case AppStateDiff:
		return m.diff.View()
	default:
		return "Unknown state"
	}
}

// SetMode sets the application mode (act/plan).
func (m *AppModel) SetMode(mode string) {
	m.mode = mode
	m.chat.SetMode(mode)
}

// SetYolo sets yolo mode for auto-approve.
func (m *AppModel) SetYolo(yolo bool) {
	m.yolo = yolo
	m.chat.SetYolo(yolo)
}

// SetProgram sets the tea program reference for sending messages.
func (m *AppModel) SetProgram(program *tea.Program) {
	m.program = program
	m.chat.SetProgram(program)
}

// IsQuitting returns true if the application is quitting.
func (m *AppModel) IsQuitting() bool {
	return m.quitting
}

// GetState returns the current application state.
func (m *AppModel) GetState() AppState {
	return m.state
}

// SetState sets the application state.
func (m *AppModel) SetState(state AppState) {
	m.state = state
}

// ShowDiff shows a diff in the diff viewer.
func (m *AppModel) ShowDiff(filename, diff string) {
	m.diff = NewDiffModel(filename, diff)
	m.state = AppStateDiff
}

// GetChatModel returns the chat model.
func (m *AppModel) GetChatModel() *ChatModel {
	return m.chat
}

// GetWelcomeModel returns the welcome model.
func (m *AppModel) GetWelcomeModel() *WelcomeModel {
	return m.welcome
}

// GetHistoryModel returns the history model.
func (m *AppModel) GetHistoryModel() *HistoryModel {
	return m.history
}

// GetSettingsModel returns the settings model.
func (m *AppModel) GetSettingsModel() *SettingsModel {
	return m.settings
}

// AppStyles holds application-wide styles.
type AppStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
}

// DefaultAppStyles returns default application styles.
func DefaultAppStyles() AppStyles {
	return AppStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
	}
}