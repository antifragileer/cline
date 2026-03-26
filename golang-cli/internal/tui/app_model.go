// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// AppState represents the overall application state.
type AppState int

const (
	// AppStateWelcome shows the welcome screen.
	AppStateWelcome AppState = iota
	// AppStateChat shows the chat interface.
	AppStateChat
	// AppStateHistory shows task history.
	AppStateHistory
	// AppStateSettings shows settings.
	AppStateSettings
	// AppStateDiff shows diff viewer for approvals.
	AppStateDiff
	// AppStateHelp shows help.
	AppStateHelp
	// AppStateQuitting is when quitting.
	AppStateQuitting
)

// TaskRunner interface for task execution (avoids import cycle).
type TaskRunner interface {
	RunWithStreaming(ctx interface{}, config interface{}, handler MessageHandler) error
}

// AppModel is the main application model that manages all TUI components.
type AppModel struct {
	// State
	state    AppState
	previous AppState

	// Sub-models
	welcome *WelcomeModel
	chat    *ChatModel

	// Integration
	streamingHandler *StreamingHandler
	taskRunner       TaskRunner
	config           *config.LayeredConfig
	storage          *storage.StorageContext
	client           *host.Client

	// Current task
	currentTaskID string
	mode          string
	yolo          bool

	// Dimensions
	width  int
	height int

	// Styling
	styles AppStyles
}

// AppStyles holds styling for the app.
type AppStyles struct {
	quitMessage lipgloss.Style
}

// DefaultAppStyles returns default app styles.
func DefaultAppStyles() AppStyles {
	return AppStyles{
		quitMessage: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
	}
}

// NewAppModel creates a new app model.
func NewAppModel(client *host.Client, storage *storage.StorageContext, cfg *config.LayeredConfig) *AppModel {
	welcome := NewWelcomeModel()
	chat := NewChatModel()

	// Check if there's history
	hasHistory := false
	var recentTask string
	if storage != nil {
		// Try to get history from storage
		// This is simplified - in practice you'd use the history package
		hasHistory = false // Will be determined by actual storage lookup
		recentTask = ""
	}

	welcome.SetHasHistory(hasHistory)
	welcome.SetRecentTask(recentTask)

	return &AppModel{
		state:            AppStateWelcome,
		welcome:          welcome,
		chat:             chat,
		streamingHandler: NewStreamingHandler(),
		client:           client,
		storage:          storage,
		config:           cfg,
		mode:             "act",
		yolo:             false,
		styles:           DefaultAppStyles(),
	}
}

// SetMode sets the execution mode.
func (m *AppModel) SetMode(mode string) {
	m.mode = mode
	if m.chat != nil {
		m.chat.SetMode(mode)
	}
}

// SetYolo sets yolo mode.
func (m *AppModel) SetYolo(yolo bool) {
	m.yolo = yolo
	if m.chat != nil {
		m.chat.SetYolo(yolo)
	}
}

// SetTaskRunner sets the task runner (injected from outside to avoid import cycle).
func (m *AppModel) SetTaskRunner(runner TaskRunner) {
	m.taskRunner = runner
}

// Init initializes the model.
func (m AppModel) Init() tea.Cmd {
	return m.welcome.Init()
}

// Update handles messages and updates the model.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.updateSubmodels(msg)

	case tea.KeyMsg:
		// Global shortcuts
		if msg.Type == tea.KeyCtrlC {
			m.state = AppStateQuitting
			return m, tea.Quit
		}
	}

	// Route to current state
	switch m.state {
	case AppStateWelcome:
		return m.updateWelcome(msg)

	case AppStateChat:
		return m.updateChat(msg)

	case AppStateHistory:
		return m.updateHistory(msg)

	case AppStateHelp:
		return m.updateHelp(msg)

	case AppStateDiff:
		return m.updateDiff(msg)

	case AppStateQuitting:
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

// updateSubmodels updates all submodels with the message.
func (m *AppModel) updateSubmodels(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update welcome
	if m.welcome != nil {
		newWelcome, cmd := m.welcome.Update(msg)
		m.welcome = newWelcome.(*WelcomeModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update chat
	if m.chat != nil {
		newChat, cmd := m.chat.Update(msg)
		m.chat = newChat.(*ChatModel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// updateWelcome handles welcome screen updates.
func (m *AppModel) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	newWelcome, cmd := m.welcome.Update(msg)
	m.welcome = newWelcome.(*WelcomeModel)

	// Check if an action was selected
	if m.welcome.GetSelected() != 0 || m.welcome.IsInputMode() {
		return m.handleWelcomeAction()
	}

	return m, cmd
}

// handleWelcomeAction handles the selected welcome action.
func (m *AppModel) handleWelcomeAction() (tea.Model, tea.Cmd) {
	action := m.welcome.GetSelected()
	inputValue := m.welcome.GetInputValue()

	switch action {
	case ActionNewTask:
		m.state = AppStateChat
		if inputValue != "" {
			// Start task with input - will be handled by caller
			m.chat.AddMessage(Message{
				Type:    MessageTypeUser,
				Content: inputValue,
			})
		}
		return m, nil

	case ActionContinueTask:
		m.state = AppStateChat
		// Continue task - will be handled by caller
		return m, nil

	case ActionHistory:
		m.previous = m.state
		m.state = AppStateHistory
		return m, nil

	case ActionSettings:
		m.previous = m.state
		m.state = AppStateSettings
		return m, nil

	case ActionHelp:
		m.previous = m.state
		m.state = AppStateHelp
		return m, nil

	case ActionQuit:
		m.state = AppStateQuitting
		return m, tea.Quit
	}

	return m, nil
}

// updateChat handles chat interface updates.
func (m *AppModel) updateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle special messages first
	switch msg := msg.(type) {
	case TaskStartedMsg:
		m.currentTaskID = msg.TaskID
		m.chat.SetTaskID(msg.TaskID)
		return m, nil

	case TaskCompletedMsg:
		// Task completed, could show summary or return to welcome
		return m, nil

	case ReturnToWelcomeMsg:
		m.state = AppStateWelcome
		m.chat.ClearMessages()
		m.welcome.Reset()
		return m, nil
	}

	// Update chat model
	newChat, cmd := m.chat.Update(msg)
	m.chat = newChat.(*ChatModel)

	return m, cmd
}

// updateHistory handles history screen updates.
func (m *AppModel) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	// TODO: Implement history screen
	// For now, just go back
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEsc {
		m.state = m.previous
	}
	return m, nil
}

// updateHelp handles help screen updates.
func (m *AppModel) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	// TODO: Implement help screen
	// For now, just go back
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEsc {
		m.state = m.previous
	}
	return m, nil
}

// updateDiff handles diff viewer updates.
func (m *AppModel) updateDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	// TODO: Implement diff viewer integration
	// For now, just go back
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEsc {
		m.state = AppStateChat
	}
	return m, nil
}

// View renders the current view.
func (m AppModel) View() string {
	switch m.state {
	case AppStateWelcome:
		return m.welcome.View()

	case AppStateChat:
		return m.chat.View()

	case AppStateHistory:
		return "Task History (Press Esc to go back)"

	case AppStateSettings:
		return "Settings (Press Esc to go back)"

	case AppStateHelp:
		return m.renderHelp()

	case AppStateQuitting:
		return m.styles.quitMessage.Render("Goodbye! 👋")

	default:
		return ""
	}
}

// renderHelp renders the help screen.
func (m AppModel) renderHelp() string {
	help := `
Cline CLI Help
==============

Keyboard Shortcuts
------------------
Ctrl+C          Quit application
Tab             Toggle Act/Plan mode
Shift+Tab       Toggle auto-approve
↑/↓             Navigate history/menus
Enter           Select/submit
Esc             Cancel/go back

Commands
--------
/help           Show this help
/settings       Open settings
/history        Show task history
/clear          Clear the screen
/exit or /quit  Exit Cline

File Operations
---------------
@filename       Mention a file
@               Start file search

For more information, visit: https://docs.cline.bot
`
	return help
}

// SetProgram sets the tea program for sending messages.
func (m *AppModel) SetProgram(program *tea.Program) {
	m.streamingHandler.SetProgram(program)
}

// GetState returns the current app state.
func (m *AppModel) GetState() AppState {
	return m.state
}

// IsQuitting returns true if the app is quitting.
func (m *AppModel) IsQuitting() bool {
	return m.state == AppStateQuitting
}

// GetChatModel returns the chat model for external access.
func (m *AppModel) GetChatModel() *ChatModel {
	return m.chat
}

// GetStreamingHandler returns the streaming handler.
func (m *AppModel) GetStreamingHandler() *StreamingHandler {
	return m.streamingHandler
}

// Tea message types

// TaskStartedMsg is sent when a task starts.
type TaskStartedMsg struct {
	TaskID string
}

// TaskCompletedMsg is sent when a task completes.
type TaskCompletedMsg struct{}

// ReturnToWelcomeMsg is sent to return to the welcome screen.
type ReturnToWelcomeMsg struct{}