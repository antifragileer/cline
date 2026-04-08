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

// ChatModel represents the chat interface state.
type ChatModel struct {
	// Dimensions
	width  int
	height int

	// Messages
	messages     []Message
	messageStore *MessageStore

	// Input handling
	input        textinput.Model
	inputEnabled bool

	// State
	state        ChatState
	isStreaming  bool
	streamBuffer string

	// Approval handling
	pendingApproval      *ApprovalRequest
	approvalResponseChan chan string

	// Action buttons
	actionButtons *ActionButtons
	buttonConfig  ButtonConfig

	// Static/Dynamic rendering
	staticRenderer *StaticRenderer
	staticContent  map[string]string // key -> rendered content
	userScrolled   bool

	// Metadata
	taskID string
	mode   string // "act" or "plan"
	yolo   bool

	// Telemetry
	telemetryEnabled bool
	telemetry        *Telemetry

	// Styling
	styles   ChatStyles
	renderer *MarkdownRenderer
}

// ChatState represents the current state of the chat.
type ChatState int

const (
	// ChatStateIdle is when waiting for user input.
	ChatStateIdle ChatState = iota
	// ChatStateStreaming is when receiving streaming response.
	ChatStateStreaming
	// ChatStateWaitingForApproval is when waiting for user approval.
	ChatStateWaitingForApproval
	// ChatStateError is when an error has occurred.
	ChatStateError
)

// ApprovalRequest represents a pending approval request.
type ApprovalRequest struct {
	AskType  string
	Text     string
	Response chan<- string
}

// ChatStyles holds styling for the chat interface.
type ChatStyles struct {
	containerStyle   lipgloss.Style
	userMsgStyle     lipgloss.Style
	aiMsgStyle       lipgloss.Style
	systemMsgStyle   lipgloss.Style
	errorMsgStyle    lipgloss.Style
	toolMsgStyle     lipgloss.Style
	inputStyle       lipgloss.Style
	statusStyle      lipgloss.Style
	approvalBoxStyle lipgloss.Style
}

// DefaultChatStyles returns default chat styles.
func DefaultChatStyles() ChatStyles {
	return ChatStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		userMsgStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true),

		aiMsgStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)),

		systemMsgStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true),

		errorMsgStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)).
			Bold(true),

		toolMsgStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(PrimaryBlue)).
			Padding(0, 1),

		statusStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		approvalBoxStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(PlanYellow)).
			Padding(1),
	}
}

// NewChatModel creates a new chat model.
func NewChatModel() *ChatModel {
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

	return &ChatModel{
		messages:             make([]Message, 0),
		messageStore:         store,
		input:                ti,
		state:                ChatStateIdle,
		inputEnabled:         true,
		styles:               DefaultChatStyles(),
		renderer:             renderer,
		mode:                 "act",
		yolo:                 false,
		actionButtons:        NewActionButtons(ButtonConfig{}, "act", 80),
		buttonConfig:         ButtonConfig{},
		approvalResponseChan: make(chan string, 1),
		staticRenderer:       NewStaticRenderer(),
		staticContent:        make(map[string]string),
	}
}

// Init initializes the model.
func (m ChatModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model.
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 6
		if m.renderer != nil {
			m.renderer.SetWidth(msg.Width - 10)
		}
		if m.actionButtons != nil {
			m.actionButtons.SetTerminalWidth(msg.Width)
		}

	case tea.KeyMsg:
		// If in YOLO mode and waiting for approval, auto-approve
		if m.yolo && m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
			response := m.HandleYOLOApproval(m.pendingApproval.AskType, m.pendingApproval.Text)
			if response != "" {
				m.pendingApproval.Response <- response
				m.pendingApproval = nil
				m.state = ChatStateIdle
				m.inputEnabled = true
				m.updateButtonConfig()
				return m, nil
			}
		}
		cmds = append(cmds, m.handleKeyMsg(msg)...)

	case ChatUpdateMsg:
		m.messages = msg.Messages
		// Update message store
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
		// Update button config for streaming state
		m.updateButtonConfig()

	case StreamMessageMsg:
		// Handle messages from gRPC stream
		if msg.Message != nil {
			m.handleStreamMessage(*msg.Message)
		}

	case ApprovalRequestMsg:
		m.state = ChatStateWaitingForApproval
		m.pendingApproval = &ApprovalRequest{
			AskType:  msg.AskType,
			Text:     msg.Text,
			Response: msg.Response,
		}
		m.inputEnabled = false
		m.updateButtonConfig()

	case ApprovalResponseMsg:
		// Handle approval response
		m.state = ChatStateIdle
		m.pendingApproval = nil
		m.inputEnabled = true
		m.updateButtonConfig()

	case StreamStateMsg:
		// Handle stream state changes
		switch msg.State {
		case 2: // StreamStateReady
			m.state = ChatStateIdle
		case 3: // StreamStateReconnecting
			m.isStreaming = true
		}
		m.updateButtonConfig()

	case StatusUpdateMsg:
		// Status updates don't change model state directly
		// Could update a status bar in future

	case ProgressUpdateMsg:
		// Progress updates don't change model state directly
		// Could update a progress bar in future

	case ChatErrorMsg:
		m.state = ChatStateError
		// Add error message
		m.AddMessage(Message{
			Type:      MessageTypeError,
			Content:   msg.Error.Error(),
			Timestamp: time.Now(),
		})
		m.state = ChatStateIdle
		m.updateButtonConfig()
	}

	// Update input component
	if m.inputEnabled {
		newInput, cmd := m.input.Update(msg)
		m.input = newInput
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input.
func (m *ChatModel) handleKeyMsg(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
			// Cancel approval - send reject
			m.pendingApproval.Response <- "noButtonClicked"
			m.pendingApproval = nil
			m.state = ChatStateIdle
			m.inputEnabled = true
			m.updateButtonConfig()
		} else {
			// Quit the application
			return []tea.Cmd{tea.Quit}
		}

	case tea.KeyEnter:
		if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
			// Approve with Enter (primary action)
			response := HandleButtonAction(m.buttonConfig.PrimaryAction, m.pendingApproval.AskType)
			m.pendingApproval.Response <- response
			m.pendingApproval = nil
			m.state = ChatStateIdle
			m.inputEnabled = true
			m.updateButtonConfig()
		} else if m.inputEnabled && m.input.Value() != "" {
			// Submit user message
			cmds = append(cmds, m.handleUserInput(m.input.Value()))
			m.input.SetValue("")
		}

	case tea.KeyRunes:
		// Handle quick approval keys and button shortcuts
		if m.state == ChatStateWaitingForApproval && m.pendingApproval != nil {
			switch msg.String() {
			case "y", "Y":
				m.pendingApproval.Response <- "yesButtonClicked"
				m.pendingApproval = nil
				m.state = ChatStateIdle
				m.inputEnabled = true
				m.updateButtonConfig()
			case "n", "N":
				m.pendingApproval.Response <- "noButtonClicked"
				m.pendingApproval = nil
				m.state = ChatStateIdle
				m.inputEnabled = true
				m.updateButtonConfig()
			}
		}

		// Handle button shortcuts (1 for primary, 2 for secondary)
		if m.buttonConfig.EnableButtons && m.state == ChatStateWaitingForApproval {
			input := msg.String()
			hasPrimary := m.buttonConfig.PrimaryText != ""
			hasSecondary := m.buttonConfig.SecondaryText != ""

			if input == "1" {
				var action ButtonActionType
				if hasPrimary {
					action = m.buttonConfig.PrimaryAction
				} else if hasSecondary {
					action = m.buttonConfig.SecondaryAction
				}
				response := HandleButtonAction(action, m.pendingApproval.AskType)
				m.pendingApproval.Response <- response
				m.pendingApproval = nil
				m.state = ChatStateIdle
				m.inputEnabled = true
				m.updateButtonConfig()
			} else if input == "2" && hasPrimary && hasSecondary {
				response := HandleButtonAction(m.buttonConfig.SecondaryAction, m.pendingApproval.AskType)
				m.pendingApproval.Response <- response
				m.pendingApproval = nil
				m.state = ChatStateIdle
				m.inputEnabled = true
				m.updateButtonConfig()
			}
		}
	}

	return cmds
}

// handleUserInput handles user input submission.
func (m *ChatModel) handleUserInput(input string) tea.Cmd {
	return func() tea.Msg {
		// Add user message
		m.AddMessage(Message{
			Type:      MessageTypeUser,
			Content:   input,
			Timestamp: time.Now(),
		})
		return ChatUpdateMsg{
			Messages: m.messages,
		}
	}
}

// View renders the chat interface.
func (m ChatModel) View() string {
	var content strings.Builder

	// Partition messages into static and dynamic regions
	partition := m.staticRenderer.PartitionMessages(m.messages, m.userScrolled)

	// Render static content (completed messages)
	staticView := m.renderStaticContent(partition.StaticItems)
	if staticView != "" {
		content.WriteString(staticView)
		content.WriteString("\n")
	}

	// Render dynamic content (streaming/current message)
	if partition.HasDynamic && partition.DynamicItem != nil {
		dynamicView := m.renderDynamicContent(*partition.DynamicItem)
		if dynamicView != "" {
			content.WriteString(dynamicView)
			content.WriteString("\n")
		}
	}

	// Render action buttons if enabled
	if m.actionButtons != nil && m.actionButtons.ShouldShow() {
		buttonsView := m.actionButtons.Render()
		if buttonsView != "" {
			content.WriteString(buttonsView)
			content.WriteString("\n")
		}
	}

	// Render input if enabled
	if m.inputEnabled {
		inputView := m.styles.inputStyle.Render(m.input.View())
		content.WriteString(inputView)
		content.WriteString("\n")
	}

	// Render status bar
	statusView := m.renderStatusBar()
	content.WriteString(statusView)

	return m.styles.containerStyle.Render(content.String())
}

// renderStaticContent renders completed messages that have been "logged" to static region.
func (m ChatModel) renderStaticContent(staticItems []ContentItem) string {
	if len(staticItems) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, item := range staticItems {
		// Find the message with this key
		for _, msg := range m.messages {
			if msg.GetKey() == item.Key {
				rendered := m.renderMessage(msg)
				if rendered != "" {
					if builder.Len() > 0 {
						builder.WriteString("\n")
					}
					builder.WriteString(rendered)
				}
				break
			}
		}
	}

	return builder.String()
}

// renderDynamicContent renders the current streaming message.
func (m ChatModel) renderDynamicContent(item ContentItem) string {
	// Find the message with this key
	for _, msg := range m.messages {
		if msg.GetKey() == item.Key {
			return m.renderMessage(msg)
		}
	}
	return ""
}

// renderMessages renders all messages (legacy method for compatibility).
func (m ChatModel) renderMessages() string {
	var content strings.Builder

	for _, msg := range m.messages {
		rendered := m.renderMessage(msg)
		content.WriteString(rendered)
		content.WriteString("\n")
	}

	// Show streaming indicator if active
	if m.isStreaming {
		content.WriteString(m.styles.systemMsgStyle.Render("▌"))
	}

	return content.String()
}

// renderMessage renders a single message.
func (m ChatModel) renderMessage(msg Message) string {
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

// renderUserMessage renders a user message.
func (m ChatModel) renderUserMessage(msg Message) string {
	prefix := m.styles.userMsgStyle.Render("You: ")
	return prefix + msg.Content
}

// renderAIMessage renders an AI assistant message.
func (m ChatModel) renderAIMessage(msg Message) string {
	content := msg.Content
	if m.renderer != nil && !msg.Partial {
		// Try to render as markdown
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}
	prefix := m.styles.aiMsgStyle.Render("Cline: ")
	return prefix + content
}

// renderErrorMessage renders an error message.
func (m ChatModel) renderErrorMessage(msg Message) string {
	return m.styles.errorMsgStyle.Render("Error: " + msg.Content)
}

// renderToolMessage renders a tool use message.
func (m ChatModel) renderToolMessage(msg Message) string {
	toolName := msg.ToolName
	if toolName == "" {
		toolName = "tool"
	}
	return m.styles.toolMsgStyle.Render(fmt.Sprintf("🔧 Using %s...", toolName))
}

// renderToolResultMessage renders a tool result message.
func (m ChatModel) renderToolResultMessage(msg Message) string {
	return m.styles.systemMsgStyle.Render(fmt.Sprintf("Result: %s", msg.ToolResult))
}

// renderAskMessage renders an ask message.
func (m ChatModel) renderAskMessage(msg Message) string {
	return m.styles.aiMsgStyle.Render("Cline: " + msg.Content)
}

// renderSystemMessage renders a system message.
func (m ChatModel) renderSystemMessage(msg Message) string {
	return m.styles.systemMsgStyle.Render(msg.Content)
}

// renderStatusBar renders the status bar.
func (m ChatModel) renderStatusBar() string {
	var parts []string

	// Mode indicator
	modeStr := "Act"
	modeColor := SelectionBlue
	if m.mode == "plan" {
		modeStr = "Plan"
		modeColor = PlanYellow
	}
	parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(modeColor)).Render(modeStr))

	// Yolo indicator
	if m.yolo {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color(SuccessGreen)).Render("YOLO"))
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
		parts = append(parts, m.styles.statusStyle.Render("Waiting for approval"))
	}

	return m.styles.statusStyle.Render(strings.Join(parts, " | "))
}

// SetTaskID sets the task ID.
func (m *ChatModel) SetTaskID(taskID string) {
	m.taskID = taskID
}

// SetMode sets the mode (act/plan).
func (m *ChatModel) SetMode(mode string) {
	m.mode = mode
	m.actionButtons.SetMode(mode)
}

// SetYolo sets yolo mode for auto-approve.
func (m *ChatModel) SetYolo(yolo bool) {
	m.yolo = yolo
}

// IsYOLO returns true if YOLO mode is enabled
func (m *ChatModel) IsYOLO() bool {
	return m.yolo
}

// SetTelemetryEnabled enables or disables telemetry
func (m *ChatModel) SetTelemetryEnabled(enabled bool) {
	m.telemetryEnabled = enabled
	if enabled && m.telemetry == nil {
		m.telemetry = NewTelemetry()
	}
}

// ShouldAutoApprove returns true if the current operation should be auto-approved
func (m *ChatModel) ShouldAutoApprove(askType string) bool {
	// YOLO mode auto-approves all tools
	if m.yolo {
		return true
	}

	// Individual auto-approve settings can be checked here
	return false
}

// HandleYOLOApproval handles auto-approval in YOLO mode
func (m *ChatModel) HandleYOLOApproval(askType string, text string) string {
	if !m.yolo {
		return ""
	}

	// Capture telemetry for YOLO mode approval
	if m.telemetryEnabled && m.telemetry != nil {
		m.telemetry.RecordAutoApproval(askType, text)
	}

	// Return appropriate response based on ask type
	switch askType {
	case "tool", "command", "browser_action", "use_mcp_server":
		return "yesButtonClicked"
	case "followup":
		return "messageResponse"
	default:
		return "yesButtonClicked"
	}
}

// AddMessage adds a message to the chat.
func (m *ChatModel) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.messageStore.Add(&msg)
}

// GetMessages returns all messages.
func (m *ChatModel) GetMessages() []Message {
	return m.messages
}

// GetInput returns the current input value.
func (m *ChatModel) GetInput() string {
	return m.input.Value()
}

// SetInput sets the input value.
func (m *ChatModel) SetInput(value string) {
	m.input.SetValue(value)
}

// ClearMessages clears all messages.
func (m *ChatModel) ClearMessages() {
	m.messages = make([]Message, 0)
	m.messageStore.Clear()
	if m.staticRenderer != nil {
		m.staticRenderer.Clear()
	}
	m.staticContent = make(map[string]string)
}

// SetUserScrolled sets whether the user has scrolled.
func (m *ChatModel) SetUserScrolled(scrolled bool) {
	m.userScrolled = scrolled
}

// IsIdle returns true if the chat is idle.
func (m *ChatModel) IsIdle() bool {
	return m.state == ChatStateIdle
}

// IsStreaming returns true if streaming.
func (m *ChatModel) IsStreaming() bool {
	return m.isStreaming
}

// IsWaitingForApproval returns true if waiting for approval.
func (m *ChatModel) IsWaitingForApproval() bool {
	return m.state == ChatStateWaitingForApproval
}

// GetPendingApproval returns the pending approval request.
func (m *ChatModel) GetPendingApproval() *ApprovalRequest {
	return m.pendingApproval
}

// SetProgram sets the tea program for sending messages.
func (m *ChatModel) SetProgram(program *tea.Program) {
	// Store program reference if needed for sending messages
}

// ShouldReturnToWelcome returns true if user wants to return to welcome screen.
func (m *ChatModel) ShouldReturnToWelcome() bool {
	// Check if user pressed a shortcut to go back
	return false
}

// SetDimensions sets the terminal dimensions.
func (m *ChatModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = width - 6
	if m.renderer != nil {
		m.renderer.SetWidth(width - 10)
	}
	if m.actionButtons != nil {
		m.actionButtons.SetTerminalWidth(width)
	}
}

// handleStreamMessage handles messages from the gRPC stream
func (m *ChatModel) handleStreamMessage(msg Message) {
	// Update streaming state based on partial flag
	if msg.Partial {
		m.isStreaming = true
		m.state = ChatStateStreaming
	} else {
		m.isStreaming = false
		m.state = ChatStateIdle
	}

	// Check if this is an update to the last message (for partial/streaming)
	if len(m.messages) > 0 {
		lastMsg := &m.messages[len(m.messages)-1]
		if lastMsg.Type == msg.Type && lastMsg.Partial && msg.Partial {
			// Update the last message
			lastMsg.Content = msg.Content
			lastMsg.Partial = msg.Partial
			return
		}
	}

	// Add as new message
	m.messages = append(m.messages, msg)
	m.messageStore.Add(&msg)

	// Update button config based on new message
	m.updateButtonConfig()
}

// GetMessageHandler returns a function that can be used to handle messages
func (m *ChatModel) GetMessageHandler() func(Message) {
	return func(msg Message) {
		// This will be called by the gRPC integration
		// The message will be sent to the program in the actual implementation
		m.handleStreamMessage(msg)
	}
}

// updateButtonConfig updates the button configuration based on current state
func (m *ChatModel) updateButtonConfig() {
	var msgType, msgSubType string
	var isPartial bool

	// Get the last message info
	if len(m.messages) > 0 {
		lastMsg := m.messages[len(m.messages)-1]
		msgType = string(lastMsg.Type)
		if askType, ok := lastMsg.GetMetadata("askType"); ok {
			msgSubType = askType.(string)
		} else if lastMsg.Type == MessageTypeAsk {
			// Try to extract from message content or use default
			msgSubType = "tool"
		}
		isPartial = lastMsg.Partial
	}

	m.buttonConfig = GetButtonConfig(msgType, msgSubType, m.isStreaming, isPartial)
	if m.actionButtons != nil {
		m.actionButtons.SetConfig(m.buttonConfig)
	}
}

// ApprovalResponseMsg is sent when an approval response is received
type ApprovalResponseMsg struct {
	Response string
}
