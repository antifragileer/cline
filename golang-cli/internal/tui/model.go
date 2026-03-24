// Package tui provides a Bubble Tea based terminal user interface framework.
// It implements the Model-Update-View architecture for building interactive
// terminal applications with support for both TUI and plain text modes.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// view represents the current view state
type view int

const (
	welcomeView view = iota
	chatView
	settingsView
)

// Model represents the TUI model
type Model struct {
	// Core components
	viewport   viewport.Model
	textInput  textinput.Model
	spinner    spinner.Model
	help       help.Model
	
	// State
	mode        Mode
	currentView view
	messages    []Message
	width       int
	height      int
	loading     bool
	errorMsg    string
	inputFocused bool
	showWelcome  bool
	showHelp     bool
	showCharCount bool
	showHints    bool
	showTips     bool
	showFeatureHighlights bool
	showConfigWizard bool
	configWizardStep string
	onboardingStep int
	ready        bool
	err          error
	title        string
	content      string
	dimensions   Dimensions
	shutdownCallbacks []func()
	
	// Input history
	inputHistory      []string
	inputHistoryIndex int
	
	// Task state
	currentTaskID string
	taskHistory   []TaskHistoryItem
	selectedTaskIndex int
	
	// Menu
	welcomeMenuItems  []string
	selectedMenuIndex int
	
	// Approval
	pendingApproval *ApprovalRequest
	approvalQueue   []ApprovalRequest
	approvalCount   int
	
	// Auto-scroll
	autoScroll bool
	
	// Config
	config Config
	
	// Version
	version string
}

// Mode represents the operating mode of the TUI
type Mode int

const (
	// ModeTUI runs in full TUI mode with interactive UI
	ModeTUI Mode = iota
	// ModePlain runs in plain text mode without TUI styling
	ModePlain
)

// Dimensions represents terminal dimensions
type Dimensions struct {
	Width  int
	Height int
}

// CodeBlock represents a code block in a message
type CodeBlock struct {
	Language string
	Code     string
}

// AddMessageMsg is sent when a new message should be added
type AddMessageMsg struct {
	Role    string
	Content string
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// Config represents user configuration
type Config struct {
	APIKey      string
	Provider    string
	FirstRun    bool
	LastUsed    string
	TaskHistory []TaskHistoryItem
}

// TaskHistoryItem represents a task in history
type TaskHistoryItem struct {
	ID          string
	Description string
	Timestamp   string
}

// ApprovalRequest represents a pending approval
type ApprovalRequest struct {
	ID      string
	Type    string
	Message string
	Options []string
}

// default dimensions and limits
const (
	defaultWidth      = 80
	defaultHeight     = 24
	maxHistorySize    = 100
	maxInputLength    = 10000
	maxRecentTasks    = 10
)

// initialModel creates a new initial model
func initialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	
	s := spinner.New()
	s.Spinner = spinner.Dot
	
	vp := viewport.New(defaultWidth, defaultHeight-3)
	
	return Model{
		viewport:          vp,
		textInput:         ti,
		spinner:           s,
		help:              help.New(),
		currentView:       welcomeView,
		messages:          []Message{},
		width:             defaultWidth,
		height:            defaultHeight,
		autoScroll:        true,
		inputHistory:      []string{},
		inputHistoryIndex: -1,
		showWelcome:       true,
		welcomeMenuItems:  []string{"New Task", "Recent Tasks", "Settings", "Help"},
		config:            Config{FirstRun: true},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

// isFirstTimeUser returns true if this is the first time the user is running the app
func (m Model) isFirstTimeUser() bool {
	return m.config.FirstRun || len(m.config.TaskHistory) == 0
}

// needsConfiguration returns true if the app needs configuration
func (m Model) needsConfiguration() bool {
	return m.config.APIKey == "" || m.config.Provider == ""
}

// hasValidConfiguration returns true if the app has valid configuration
func (m Model) hasValidConfiguration() bool {
	return m.config.APIKey != "" && m.config.Provider != ""
}

// renderMessage renders a single message
func (m Model) renderMessage(msg Message) string {
	return renderMessage(m, msg)
}

// renderChatView renders the chat view
func (m Model) renderChatView() string {
	return renderChatView(m)
}

// renderInputArea renders the input area
func (m Model) renderInputArea() string {
	return renderInputArea(m)
}

// renderWelcomeView renders the welcome view
func (m Model) renderWelcomeView() string {
	return renderWelcomeView(m)
}

// renderRecentTasks renders the recent tasks list
func (m Model) renderRecentTasks() string {
	return renderRecentTasks(m)
}

// highlightCode highlights code with syntax highlighting
func (m Model) highlightCode(code, language string) string {
	return highlightCode(m, code, language)
}

// wrapText wraps text to a specified width
func (m Model) wrapText(text string, width int) string {
	lines := wrapText(text, width)
	return strings.Join(lines, "\n")
}

// extractCodeBlocks extracts code blocks from content
func (m Model) extractCodeBlocks(content string) []CodeBlock {
	return extractCodeBlocks(m, content)
}

// getWelcomeStyle returns the style for the welcome screen
func (m Model) getWelcomeStyle() lipgloss.Style {
	return getWelcomeStyle(m)
}

// getLogoStyle returns the style for the logo
func (m Model) getLogoStyle() lipgloss.Style {
	return getLogoStyle(m)
}

// getMenuStyle returns the style for menu items
func (m Model) getMenuStyle() lipgloss.Style {
	return getMenuStyle(m)
}

// getSelectedMenuStyle returns the style for selected menu items
func (m Model) getSelectedMenuStyle() lipgloss.Style {
	return getSelectedMenuStyle(m)
}

// getTaskItemStyle returns the style for task items
func (m Model) getTaskItemStyle() lipgloss.Style {
	return getTaskItemStyle(m)
}

// quitCmd returns a quit command
func quitCmd() tea.Cmd {
	return tea.Quit
}

// Error returns any error state.
func (m Model) Error() error {
	return m.err
}

// SetError sets the error state.
func (m *Model) SetError(err error) {
	m.err = err
}

// RegisterShutdownCallback registers a function to be called during shutdown.
func (m *Model) RegisterShutdownCallback(fn func()) {
	m.shutdownCallbacks = append(m.shutdownCallbacks, fn)
}

// Shutdown executes all registered shutdown callbacks.
func (m *Model) Shutdown() {
	for _, fn := range m.shutdownCallbacks {
		if fn != nil {
			fn()
		}
	}
}

// NewModel creates a new TUI model with the given title
func NewModel(title string) Model {
	m := initialModel()
	m.title = title
	m.mode = ModeTUI
	m.ready = true
	return m
}

// NewPlainModel creates a new plain text model
func NewPlainModel() Model {
	m := initialModel()
	m.mode = ModePlain
	m.ready = true
	return m
}

// SetContent sets the content for plain mode
func (m *Model) SetContent(content string) {
	m.content = content
}
