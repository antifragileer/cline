// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EnhancedChatModel represents the enhanced chat interface with Phase 2 features.
type EnhancedChatModel struct {
	// Base ChatModel fields
	width  int
	height int

	// Messages
	messages     []Message
	messageStore *MessageStore

	// Input handling
	input        textinput.Model
	inputEnabled bool

	// State
	state              ChatState
	isStreaming        bool
	streamBuffer       string
	thinkingStartTime  time.Time
	thinkingFrameIndex int

	// Approval handling
	pendingApproval      *ApprovalRequest
	approvalResponseChan chan string

	// Action buttons
	actionButtons *ActionButtons
	buttonConfig  ButtonConfig

	// Phase 2: Menu components
	fileMentionMenu  *FileMentionMenu
	slashCommandMenu *SlashCommandMenu
	showFileMenu     bool
	showSlashMenu    bool

	// Phase 2: Info displays
	gitStats      *GitStats
	gitDisplay    *GitStatsDisplay
	contextInfo   *ContextInfo
	contextBar    *ContextBar

	// Phase 2: Mode toggle
	modeToggleExpanded bool

	// Metadata
	taskID  string
	mode    string // "act" or "plan"
	yolo    bool
	modelID string

	// Styling
	styles   ChatStyles
	renderer *MarkdownRenderer

	// Callbacks
	onSubmit   func(string)
	onModeToggle func()
}

// NewEnhancedChatModel creates a new enhanced chat model with Phase 2 features.
func NewEnhancedChatModel() *EnhancedChatModel {
	ti := textinput.New()
	ti.Placeholder = "What can I do for you?"
	ti.Focus()
	ti.CharLimit = 4000
	ti.Width = 80

	renderer, _ := NewMarkdownRenderer(
		WithMarkdownStyle("dark"),
		WithMarkdownWidth(100),
	)

	store := NewMessageStore()

	return &EnhancedChatModel{
		messages:             make([]Message, 0),
		messageStore:         store,
		input:                ti,
		state:                ChatStateIdle,
		inputEnabled:         true,
		styles:               DefaultChatStyles(),
		renderer:             renderer,
		mode:                 "act",
		yolo:                 false,
		modelID:              "claude-sonnet-4",
		actionButtons:        NewActionButtons(ButtonConfig{}, "act", 80),
		buttonConfig:         ButtonConfig{},
		fileMentionMenu:      NewFileMentionMenu(),
		slashCommandMenu:     NewSlashCommandMenu(),
		gitDisplay:           NewGitStatsDisplay(),
		contextBar:           NewContextBar(),
		contextInfo:          &ContextInfo{},
		approvalResponseChan: make(chan string, 1),
	}
}

// Init initializes the model.
func (m EnhancedChatModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model.
func (m *EnhancedChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		cmds = append(cmds, m.handleKeyMsg(msg)...)

	case ChatUpdateMsg:
		m.messages = msg.Messages
		m.messageStore.Clear()
		for i := range msg.Messages {
			m.messageStore.Add(&msg.Messages[i])
		}

	case StreamChunkMsg:
		m.isStreaming = true
		m.state = ChatStateStreaming
		if msg.Done {
			m.isStreaming = false
			m.state = ChatStateIdle
		}
		m.updateButtonConfig()

	case StreamMessageMsg:
		if msg.Message != nil {
			m.handleStreamMessage(*msg.Message)
		}

	case ApprovalRequestMsg:
		m.handleApprovalRequest(msg)

	case ApprovalResponseMsg:
		m.handleApprovalResponse(msg)

	case ChatErrorMsg:
		m.handleError(msg)

	case tickMsg:
		// Update thinking animation frame
		if m.isStreaming {
			m.thinkingFrameIndex = (m.thinkingFrameIndex + 1) % len(SpinnerFrames)
			return m, m.tick()
		}

	case GitStatsMsg:
		m.gitStats = msg.Stats

	case ContextBarMsg:
		m.contextInfo = msg.Info
	}

	// Update input component
	if m.inputEnabled {
		newInput, cmd := m.input.Update(msg)
		m.input = newInput
		cmds = append(cmds, cmd)
	}

	// Update menu state based on input
	m.updateMenuState()

	return m, tea.Batch(cmds...)
}

// handleWindowSize handles window resize
func (m *EnhancedChatModel) handleWindowSize(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = width - 6
	if m.renderer != nil {
		m.renderer.SetWidth(width - 10)
	}
	if m.actionButtons != nil {
		m.actionButtons.SetTerminalWidth(width)
	}
	m.fileMentionMenu.SetDimensions(width, height)
	m.slashCommandMenu.SetDimensions(width, height)
}

// handleKeyMsg handles keyboard input with Phase 2 features.
func (m *EnhancedChatModel) handleKeyMsg(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// Handle menu navigation when menus are open
	if m.showFileMenu {
		return m.handleFileMenuKeys(msg)
	}
	if m.showSlashMenu {
		return m.handleSlashMenuKeys(msg)
	}

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
			m.pendingApproval.Response <- "noButtonClicked"
			m.clearApproval()
		} else {
			return []tea.Cmd{tea.Quit}
		}

	case tea.KeyEnter:
		return m.handleEnterKey()

	case tea.KeyUp, tea.KeyDown:
		// Navigate message history if input is empty
		if m.input.Value() == "" && msg.Type == tea.KeyUp {
			// TODO: Navigate to previous message in history
		}

	case tea.KeyTab:
		// Toggle mode
		if m.onModeToggle != nil {
			m.onModeToggle()
		}

	case tea.KeyRunes:
		cmds = append(cmds, m.handleRuneKeys(msg)...)
	}

	return cmds
}

// handleFileMenuKeys handles keys when file menu is open
func (m *EnhancedChatModel) handleFileMenuKeys(msg tea.KeyMsg) []tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.showFileMenu = false
		return nil

	case tea.KeyEnter:
		if result := m.fileMentionMenu.GetSelectedResult(); result != nil {
			// Insert the selected file into input
			cursorPos := len(m.input.Value())
			newText := InsertMention(m.input.Value(), cursorPos-len(m.fileMentionMenu.query)-1, result.Path)
			m.input.SetValue(newText)
			m.input.SetCursor(len(newText))
			m.showFileMenu = false
		}
		return nil

	case tea.KeyUp:
		m.fileMentionMenu.MoveSelectionUp()
		return nil

	case tea.KeyDown:
		m.fileMentionMenu.MoveSelectionDown()
		return nil
	}

	switch msg.String() {
	case "q":
		m.showFileMenu = false
	}

	return nil
}

// handleSlashMenuKeys handles keys when slash menu is open
func (m *EnhancedChatModel) handleSlashMenuKeys(msg tea.KeyMsg) []tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.showSlashMenu = false
		return nil

	case tea.KeyEnter:
		if cmd := m.slashCommandMenu.GetSelectedCommand(); cmd != nil {
			// Insert the selected command into input
			cursorPos := len(m.input.Value())
			newText := InsertSlashCommand(m.input.Value(), cursorPos-len(m.slashCommandMenu.query)-1, cmd.Name)
			m.input.SetValue(newText)
			m.input.SetCursor(len(newText))
			m.showSlashMenu = false
		}
		return nil

	case tea.KeyUp:
		m.slashCommandMenu.MoveSelectionUp()
		return nil

	case tea.KeyDown:
		m.slashCommandMenu.MoveSelectionDown()
		return nil
	}

	return nil
}

// handleEnterKey handles Enter key press
func (m *EnhancedChatModel) handleEnterKey() []tea.Cmd {
	if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
		response := HandleButtonAction(m.buttonConfig.PrimaryAction, m.pendingApproval.AskType)
		m.pendingApproval.Response <- response
		m.clearApproval()
		return nil
	}

	if m.inputEnabled && m.input.Value() != "" {
		input := m.input.Value()

		// Check for standalone slash command execution
		if command := GetStandaloneSlashCommandToExecute(
			input,
			m.showSlashMenu,
			m.showSlashMenu,
			m.pendingApproval != nil,
			m.isStreaming,
		); command != "" {
			m.handleSlashCommand(command)
			m.input.SetValue("")
			return nil
		}

		// Regular message submission
		if m.onSubmit != nil {
			m.onSubmit(input)
		}

		m.AddMessage(Message{
			Type:      MessageTypeUser,
			Content:   input,
			Timestamp: time.Now(),
		})
		m.input.SetValue("")
	}

	return nil
}

// handleRuneKeys handles character input with shortcuts
func (m *EnhancedChatModel) handleRuneKeys(msg tea.KeyMsg) []tea.Cmd {
	input := msg.String()

	// Handle approval shortcuts
	if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
		switch input {
		case "y", "Y":
			m.pendingApproval.Response <- "yesButtonClicked"
			m.clearApproval()
		case "n", "N":
			m.pendingApproval.Response <- "noButtonClicked"
			m.clearApproval()
		case "1":
			response := HandleButtonAction(m.buttonConfig.PrimaryAction, m.pendingApproval.AskType)
			m.pendingApproval.Response <- response
			m.clearApproval()
		case "2":
			if m.buttonConfig.SecondaryText != "" {
				response := HandleButtonAction(m.buttonConfig.SecondaryAction, m.pendingApproval.AskType)
				m.pendingApproval.Response <- response
				m.clearApproval()
			}
		}
		return nil
	}

	return nil
}

// updateMenuState updates menu visibility based on input
func (m *EnhancedChatModel) updateMenuState() {
	// Get cursor position from the textinput model
	cursorPos := len(m.input.Value())
	text := m.input.Value()

	// Check for file mention
	mentionState := ExtractMentionQuery(text, cursorPos)
	if mentionState.InMentionMode {
		m.showFileMenu = true
		m.showSlashMenu = false
		m.fileMentionMenu.SetQuery(mentionState.Query)
		return
	}

	// Check for slash command
	slashState := ExtractSlashQuery(text, cursorPos)
	if slashState.InSlashMode {
		m.showSlashMenu = true
		m.showFileMenu = false
		m.slashCommandMenu.SetQuery(slashState.Query)
		return
	}

	// Close menus if not in special mode
	m.showFileMenu = false
	m.showSlashMenu = false
}

// handleApprovalRequest handles approval requests
func (m *EnhancedChatModel) handleApprovalRequest(msg ApprovalRequestMsg) {
	m.state = ChatStateWaitingForApproval
	m.pendingApproval = &ApprovalRequest{
		AskType:  msg.AskType,
		Text:     msg.Text,
		Response: msg.Response,
	}
	m.inputEnabled = false
	m.updateButtonConfig()
}

// handleApprovalResponse handles approval responses
func (m *EnhancedChatModel) handleApprovalResponse(msg ApprovalResponseMsg) {
	m.clearApproval()
}

// clearApproval clears the approval state
func (m *EnhancedChatModel) clearApproval() {
	m.state = ChatStateIdle
	m.pendingApproval = nil
	m.inputEnabled = true
	m.updateButtonConfig()
}

// handleError handles errors
func (m *EnhancedChatModel) handleError(msg ChatErrorMsg) {
	m.state = ChatStateError
	m.AddMessage(Message{
		Type:      MessageTypeError,
		Content:   msg.Error.Error(),
		Timestamp: time.Now(),
	})
	m.state = ChatStateIdle
	m.updateButtonConfig()
}

// handleSlashCommand executes a slash command
func (m *EnhancedChatModel) handleSlashCommand(command string) {
	switch command {
	case "clear":
		m.ClearMessages()
	case "help", "?":
		// Show help - could open help view
	case "settings", "s":
		// Open settings
	case "history", "h":
		// Show history
	case "exit", "quit", "q":
		// Exit handled elsewhere
	default:
		// Treat as regular message
	}
}

// tick returns a tick command for animation
func (m *EnhancedChatModel) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// View renders the enhanced chat interface.
func (m EnhancedChatModel) View() string {
	var content strings.Builder

	// Header with git stats and context bar
	header := m.renderHeader()
	if header != "" {
		content.WriteString(header)
		content.WriteString("\n")
	}

	// Separator
	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#333333")).
		Render(strings.Repeat("─", m.width-4)))
	content.WriteString("\n")

	// Messages area
	messagesView := m.renderMessages()
	content.WriteString(messagesView)

	// Menus (if open)
	if m.showFileMenu {
		content.WriteString("\n")
		content.WriteString(m.fileMentionMenu.Render())
	} else if m.showSlashMenu {
		content.WriteString("\n")
		content.WriteString(m.slashCommandMenu.Render())
	}

	// Action buttons
	if m.actionButtons != nil && m.actionButtons.ShouldShow() {
		buttonsView := m.actionButtons.Render()
		if buttonsView != "" {
			content.WriteString("\n")
			content.WriteString(buttonsView)
		}
	}

	// Input area
	if m.inputEnabled {
		inputView := m.styles.inputStyle.Render(m.input.View())
		content.WriteString("\n")
		content.WriteString(inputView)
	}

	// Status bar
	statusView := m.renderStatusBar()
	content.WriteString("\n")
	content.WriteString(statusView)

	return m.styles.containerStyle.Render(content.String())
}

// renderHeader renders the header with git stats and context bar
func (m *EnhancedChatModel) renderHeader() string {
	var parts []string

	// Git stats
	if m.gitStats != nil {
		parts = append(parts, m.gitDisplay.RenderCompact(m.gitStats))
	}

	// Context bar
	if m.contextInfo != nil {
		parts = append(parts, m.contextBar.RenderCompact(m.contextInfo))
	}

	// Mode toggle
	modeToggle := m.renderModeToggle()
	parts = append(parts, modeToggle)

	if len(parts) == 0 {
		return ""
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		Render(strings.Join(parts, "  "))
}

// renderModeToggle renders the mode toggle button
func (m *EnhancedChatModel) renderModeToggle() string {
	actStyle := lipgloss.NewStyle()
	planStyle := lipgloss.NewStyle()
	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("#606060")).Render(" | ")

	if m.mode == "act" {
		actStyle = actStyle.
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true).
			Background(lipgloss.Color("#1a3a4a")).
			Padding(0, 1)
		planStyle = planStyle.
			Foreground(lipgloss.Color("#808080")).
			Padding(0, 1)
	} else {
		actStyle = actStyle.
			Foreground(lipgloss.Color("#808080")).
			Padding(0, 1)
		planStyle = planStyle.
			Foreground(lipgloss.Color("#FFB000")).
			Bold(true).
			Background(lipgloss.Color("#4a3a1a")).
			Padding(0, 1)
	}

	return actStyle.Render("Act") + separator + planStyle.Render("Plan")
}

// renderMessages renders all messages
func (m *EnhancedChatModel) renderMessages() string {
	var content strings.Builder

	// Calculate available height for messages
	headerHeight := 4  // header + separator
	footerHeight := 6  // input + status + padding
	menuHeight := 0
	if m.showFileMenu || m.showSlashMenu {
		menuHeight = 12
	}
	buttonHeight := 0
	if m.actionButtons != nil && m.actionButtons.ShouldShow() {
		buttonHeight = 3
	}

	maxMessagesHeight := m.height - headerHeight - footerHeight - menuHeight - buttonHeight
	if maxMessagesHeight < 10 {
		maxMessagesHeight = 10
	}

	// Render messages from bottom up to fit in available space
	startIdx := 0
	if len(m.messages) > maxMessagesHeight/3 {
		startIdx = len(m.messages) - maxMessagesHeight/3
	}

	for i := startIdx; i < len(m.messages); i++ {
		rendered := m.renderMessage(m.messages[i])
		content.WriteString(rendered)
		content.WriteString("\n")
	}

	// Show thinking indicator if streaming
	if m.isStreaming {
		elapsed := time.Since(m.thinkingStartTime)
		thinkingText := StaticThinkingIndicator(
			m.thinkingFrameIndex,
			"Thinking...",
			m.mode,
			elapsed,
		)
		content.WriteString(thinkingText)
	}

	return content.String()
}

// renderMessage renders a single message
func (m *EnhancedChatModel) renderMessage(msg Message) string {
	switch msg.Type {
	case MessageTypeUser:
		return m.renderUserMessage(msg)
	case MessageTypeSay:
		return m.renderAIMessage(msg)
	case MessageTypeError:
		return m.renderErrorMessage(msg)
	case MessageTypeToolUse:
		return m.renderToolMessage(msg)
	case MessageTypeToolResult:
		return m.renderToolResultMessage(msg)
	case MessageTypeAsk:
		return m.renderAskMessage(msg)
	default:
		return m.renderSystemMessage(msg)
	}
}

// renderUserMessage renders a user message
func (m *EnhancedChatModel) renderUserMessage(msg Message) string {
	prefix := m.styles.userMsgStyle.Render("You: ")
	return prefix + msg.Content
}

// renderAIMessage renders an AI assistant message
func (m *EnhancedChatModel) renderAIMessage(msg Message) string {
	content := msg.Content
	if m.renderer != nil && !msg.Partial {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}
	prefix := m.styles.aiMsgStyle.Render("Cline: ")
	return prefix + content
}

// renderErrorMessage renders an error message
func (m *EnhancedChatModel) renderErrorMessage(msg Message) string {
	return m.styles.errorMsgStyle.Render("Error: " + msg.Content)
}

// renderToolMessage renders a tool use message
func (m *EnhancedChatModel) renderToolMessage(msg Message) string {
	toolName := msg.ToolName
	if toolName == "" {
		toolName = "tool"
	}
	return m.styles.toolMsgStyle.Render(fmt.Sprintf("🔧 Using %s...", toolName))
}

// renderToolResultMessage renders a tool result message
func (m *EnhancedChatModel) renderToolResultMessage(msg Message) string {
	return m.styles.systemMsgStyle.Render(fmt.Sprintf("Result: %s", msg.ToolResult))
}

// renderAskMessage renders an ask message
func (m *EnhancedChatModel) renderAskMessage(msg Message) string {
	return m.styles.aiMsgStyle.Render("Cline: " + msg.Content)
}

// renderSystemMessage renders a system message
func (m *EnhancedChatModel) renderSystemMessage(msg Message) string {
	return m.styles.systemMsgStyle.Render(msg.Content)
}

// renderStatusBar renders the status bar
func (m *EnhancedChatModel) renderStatusBar() string {
	var parts []string

	// Yolo indicator
	if m.yolo {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			Render("⚡ YOLO"))
	}

	// Task ID
	if m.taskID != "" {
		parts = append(parts, m.styles.statusStyle.Render(fmt.Sprintf("Task: %s", m.taskID[:8])))
	}

	// State indicator
	switch m.state {
	case ChatStateStreaming:
		parts = append(parts, m.styles.statusStyle.Render("Streaming..."))
	case ChatStateWaitingForApproval:
		parts = append(parts, m.styles.statusStyle.Render("⌛ Waiting for approval"))
	}

	// Shortcuts help
	parts = append(parts, m.styles.statusStyle.Render("Tab: Mode • ?: Help"))

	if len(parts) == 0 {
		return ""
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Render(strings.Join(parts, " | "))
}

// Setters and getters

func (m *EnhancedChatModel) SetTaskID(taskID string) {
	m.taskID = taskID
}

func (m *EnhancedChatModel) SetMode(mode string) {
	m.mode = mode
	m.actionButtons.SetMode(mode)
}

func (m *EnhancedChatModel) SetYolo(yolo bool) {
	m.yolo = yolo
}

func (m *EnhancedChatModel) SetModelID(modelID string) {
	m.modelID = modelID
	if m.contextInfo != nil {
		m.contextInfo.ModelID = modelID
		m.contextInfo.ContextSize = DefaultContextSize(modelID)
	}
}

func (m *EnhancedChatModel) SetGitStats(stats *GitStats) {
	m.gitStats = stats
}

func (m *EnhancedChatModel) SetContextInfo(info *ContextInfo) {
	m.contextInfo = info
}

func (m *EnhancedChatModel) SetOnSubmit(fn func(string)) {
	m.onSubmit = fn
}

func (m *EnhancedChatModel) SetOnModeToggle(fn func()) {
	m.onModeToggle = fn
}

func (m *EnhancedChatModel) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.messageStore.Add(&msg)
}

func (m *EnhancedChatModel) GetMessages() []Message {
	return m.messages
}

func (m *EnhancedChatModel) ClearMessages() {
	m.messages = make([]Message, 0)
	m.messageStore.Clear()
}

func (m *EnhancedChatModel) IsIdle() bool {
	return m.state == ChatStateIdle
}

func (m *EnhancedChatModel) IsStreaming() bool {
	return m.isStreaming
}

func (m *EnhancedChatModel) GetInput() string {
	return m.input.Value()
}

func (m *EnhancedChatModel) SetInput(value string) {
	m.input.SetValue(value)
}

func (m *EnhancedChatModel) SetFileMentionResults(results []FileResult) {
	m.fileMentionMenu.SetResults(results)
}

func (m *EnhancedChatModel) SetFileMentionLoading(loading bool) {
	m.fileMentionMenu.SetLoading(loading)
}

// handleStreamMessage handles messages from the gRPC stream
func (m *EnhancedChatModel) handleStreamMessage(msg Message) {
	if msg.Partial {
		m.isStreaming = true
		m.state = ChatStateStreaming
		if m.thinkingStartTime.IsZero() {
			m.thinkingStartTime = time.Now()
		}
	} else {
		m.isStreaming = false
		m.state = ChatStateIdle
		m.thinkingStartTime = time.Time{}
		m.thinkingFrameIndex = 0
	}

	// Update or add message
	if len(m.messages) > 0 {
		lastMsg := &m.messages[len(m.messages)-1]
		if lastMsg.Type == msg.Type && lastMsg.Partial && msg.Partial {
			lastMsg.Content = msg.Content
			lastMsg.Partial = msg.Partial
			return
		}
	}

	m.messages = append(m.messages, msg)
	m.messageStore.Add(&msg)
	m.updateButtonConfig()
}

// updateButtonConfig updates button configuration
func (m *EnhancedChatModel) updateButtonConfig() {
	var msgType, msgSubType string
	var isPartial bool

	if len(m.messages) > 0 {
		lastMsg := m.messages[len(m.messages)-1]
		msgType = string(lastMsg.Type)
		if askType, ok := lastMsg.GetMetadata("askType"); ok {
			msgSubType = askType.(string)
		}
		isPartial = lastMsg.Partial
	}

	m.buttonConfig = GetButtonConfig(msgType, msgSubType, m.isStreaming, isPartial)
	if m.actionButtons != nil {
		m.actionButtons.SetConfig(m.buttonConfig)
	}
}