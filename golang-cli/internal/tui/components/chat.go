// Package components provides TUI components for the Cline CLI.
package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatMode represents the chat mode (act/plan)
type ChatMode string

const (
	// ChatModeAct is for execution mode
	ChatModeAct ChatMode = "act"
	// ChatModePlan is for planning mode
	ChatModePlan ChatMode = "plan"
)

// ChatState represents the current state of the chat
type ChatState int

const (
	// ChatStateIdle means waiting for input
	ChatStateIdle ChatState = iota
	// ChatStateStreaming means AI is responding
	ChatStateStreaming
	// ChatStateAsking means waiting for user approval
	ChatStateAsking
	// ChatStateError means an error occurred
	ChatStateError
)

// Chat is the main chat interface component that integrates all TUI components
type Chat struct {
	// Dimensions
	width  int
	height int

	// Components
	messageList    *ChatMessageList
	statusBar      *StatusBar
	input          *ChatInput
	slashMenu      *SlashCommandMenu
	fileMenu       *FileMentionMenu
	askPrompt      *AskPrompt
	diffView       *DiffView
	spinner        *Spinner
	checkpointMenu *CheckpointMenu

	// State
	mode     ChatMode
	state    ChatState
	yolo     bool
	taskID   string
	provider string
	model    string

	// Input state
	inputText   string
	cursorPos   int
	inSlashMode bool
	inFileMode  bool
	slashQuery  SlashQueryState
	fileQuery   FileMentionState

	// Message tracking
	lastMessageTime time.Time
	pendingChanges  int

	// Callbacks
	onSend       func(text string)
	onCancel     func()
	onApprove    func(response AskResponse)
	onCheckpoint func(action CheckpointAction, checkpoint *Checkpoint)

	// Styling
	containerStyle lipgloss.Style
}

// ChatInput handles the text input area
type ChatInput struct {
	text      string
	cursorPos int
	width     int
	height    int

	// Styling
	promptStyle      lipgloss.Style
	textStyle        lipgloss.Style
	cursorStyle      lipgloss.Style
	placeholderStyle lipgloss.Style
}

// NewChatInput creates a new chat input
func NewChatInput() *ChatInput {
	return &ChatInput{
		promptStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
		textStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		cursorStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#000000")),
		placeholderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),
	}
}

// SetText sets the input text
func (ci *ChatInput) SetText(text string) {
	ci.text = text
	ci.cursorPos = len(text)
}

// GetText returns the input text
func (ci *ChatInput) GetText() string {
	return ci.text
}

// SetCursorPos sets the cursor position
func (ci *ChatInput) SetCursorPos(pos int) {
	if pos >= 0 && pos <= len(ci.text) {
		ci.cursorPos = pos
	}
}

// Insert inserts text at cursor position
func (ci *ChatInput) Insert(text string) {
	before := ci.text[:ci.cursorPos]
	after := ci.text[ci.cursorPos:]
	ci.text = before + text + after
	ci.cursorPos += len(text)
}

// Delete deletes the character before cursor
func (ci *ChatInput) Delete() bool {
	if ci.cursorPos > 0 {
		ci.text = ci.text[:ci.cursorPos-1] + ci.text[ci.cursorPos:]
		ci.cursorPos--
		return true
	}
	return false
}

// DeleteForward deletes the character after cursor
func (ci *ChatInput) DeleteForward() bool {
	if ci.cursorPos < len(ci.text) {
		ci.text = ci.text[:ci.cursorPos] + ci.text[ci.cursorPos+1:]
		return true
	}
	return false
}

// MoveCursorLeft moves cursor left
func (ci *ChatInput) MoveCursorLeft() {
	if ci.cursorPos > 0 {
		ci.cursorPos--
	}
}

// MoveCursorRight moves cursor right
func (ci *ChatInput) MoveCursorRight() {
	if ci.cursorPos < len(ci.text) {
		ci.cursorPos++
	}
}

// MoveCursorStart moves cursor to start
func (ci *ChatInput) MoveCursorStart() {
	ci.cursorPos = 0
}

// MoveCursorEnd moves cursor to end
func (ci *ChatInput) MoveCursorEnd() {
	ci.cursorPos = len(ci.text)
}

// Clear clears the input
func (ci *ChatInput) Clear() {
	ci.text = ""
	ci.cursorPos = 0
}

// SetWidth sets the input width
func (ci *ChatInput) SetWidth(width int) {
	ci.width = width
}

// Render renders the input
func (ci *ChatInput) Render(mode ChatMode, disabled bool) string {
	prompt := ci.promptStyle.Render(">")
	if mode == ChatModePlan {
		prompt = ci.promptStyle.Render("?")
	}

	if ci.text == "" && !disabled {
		placeholder := ci.placeholderStyle.Render("Type a message...")
		return prompt + " " + placeholder
	}

	if disabled {
		return prompt + " " + ci.textStyle.Render(ci.text)
	}

	// Render text with cursor
	if ci.cursorPos >= len(ci.text) {
		// Cursor at end
		return prompt + " " + ci.textStyle.Render(ci.text) + ci.cursorStyle.Render(" ")
	}

	// Cursor in middle
	before := ci.text[:ci.cursorPos]
	at := string(ci.text[ci.cursorPos])
	after := ci.text[ci.cursorPos+1:]

	return prompt + " " +
		ci.textStyle.Render(before) +
		ci.cursorStyle.Render(at) +
		ci.textStyle.Render(after)
}

// NewChat creates a new chat interface
func NewChat() *Chat {
	c := &Chat{
		mode:   ChatModeAct,
		state:  ChatStateIdle,
		width:  120,
		height: 40,

		messageList: NewChatMessageList(),
		statusBar:   NewStatusBar(120),
		input:       NewChatInput(),
		slashMenu:   NewSlashCommandMenu(),
		fileMenu:    NewFileMentionMenu(),
		diffView:    NewDiffView(),
		spinner:     NewSpinner(WithSpinnerText("Thinking...")),

		containerStyle: lipgloss.NewStyle().
			Padding(1),
	}

	// Initialize status bar
	c.statusBar.SetMode(string(c.mode))
	c.statusBar.SetYolo(c.yolo)

	return c
}

// SetDimensions sets the chat dimensions
func (c *Chat) SetDimensions(width, height int) {
	c.width = width
	c.height = height

	// Update component dimensions
	c.messageList.SetWidth(width - 4)
	c.statusBar.SetWidth(width)
	c.input.SetWidth(width - 4)
	c.slashMenu.SetDimensions(width, height)
	c.fileMenu.SetDimensions(width, height)
}

// SetMode sets the chat mode
func (c *Chat) SetMode(mode ChatMode) {
	c.mode = mode
	c.statusBar.SetMode(string(mode))
}

// SetYolo sets yolo mode
func (c *Chat) SetYolo(yolo bool) {
	c.yolo = yolo
	c.statusBar.SetYolo(yolo)
}

// SetTaskID sets the task ID
func (c *Chat) SetTaskID(taskID string) {
	c.taskID = taskID
	c.statusBar.SetTaskID(taskID)
}

// SetProvider sets the provider
func (c *Chat) SetProvider(provider string) {
	c.provider = provider
	c.statusBar.SetProvider(provider)
}

// SetModel sets the model
func (c *Chat) SetModel(model string) {
	c.model = model
	c.statusBar.SetModel(model)
}

// SetState sets the chat state
func (c *Chat) SetState(state ChatState) {
	c.state = state
	c.statusBar.SetStreaming(state == ChatStateStreaming)
}

// AddMessage adds a message to the chat
func (c *Chat) AddMessage(msgType MessageType, content string) *ChatMessage {
	msg := NewChatMessage(msgType, content)
	c.messageList.AddMessage(msg)
	c.statusBar.SetMessageCount(len(c.messageList.GetMessages()))
	return msg
}

// UpdateLastMessage updates the last message (for streaming)
func (c *Chat) UpdateLastMessage(content string, partial bool) bool {
	return c.messageList.UpdateLast(content, partial)
}

// GetMessages returns all messages
func (c *Chat) GetMessages() []*ChatMessage {
	return c.messageList.GetMessages()
}

// ClearMessages clears all messages
func (c *Chat) ClearMessages() {
	c.messageList.Clear()
	c.statusBar.SetMessageCount(0)
}

// SetInputText sets the input text
func (c *Chat) SetInputText(text string) {
	c.input.SetText(text)
	c.updateQueryStates()
}

// GetInputText returns the input text
func (c *Chat) GetInputText() string {
	return c.input.GetText()
}

// InsertAtCursor inserts text at cursor position
func (c *Chat) InsertAtCursor(text string) {
	c.input.Insert(text)
	c.updateQueryStates()
}

// DeleteBeforeCursor deletes character before cursor
func (c *Chat) DeleteBeforeCursor() bool {
	result := c.input.Delete()
	if result {
		c.updateQueryStates()
	}
	return result
}

// DeleteAfterCursor deletes character after cursor
func (c *Chat) DeleteAfterCursor() bool {
	result := c.input.DeleteForward()
	if result {
		c.updateQueryStates()
	}
	return result
}

// MoveCursorLeft moves cursor left
func (c *Chat) MoveCursorLeft() {
	c.input.MoveCursorLeft()
	c.updateQueryStates()
}

// MoveCursorRight moves cursor right
func (c *Chat) MoveCursorRight() {
	c.input.MoveCursorRight()
	c.updateQueryStates()
}

// updateQueryStates updates slash and file query states
func (c *Chat) updateQueryStates() {
	text := c.input.GetText()
	pos := c.input.cursorPos

	c.slashQuery = ExtractSlashQuery(text, pos)
	c.fileQuery = ExtractMentionQuery(text, pos)

	c.inSlashMode = c.slashQuery.InSlashMode
	c.inFileMode = c.fileQuery.InMentionMode

	if c.inSlashMode {
		c.slashMenu.SetQuery(c.slashQuery.Query)
	}
	if c.inFileMode {
		c.fileMenu.SetQuery(c.fileQuery.Query)
	}
}

// IsInSlashMode returns true if in slash command mode
func (c *Chat) IsInSlashMode() bool {
	return c.inSlashMode
}

// IsInFileMode returns true if in file mention mode
func (c *Chat) IsInFileMode() bool {
	return c.inFileMode
}

// GetSlashMenu returns the slash command menu
func (c *Chat) GetSlashMenu() *SlashCommandMenu {
	return c.slashMenu
}

// GetFileMenu returns the file mention menu
func (c *Chat) GetFileMenu() *FileMentionMenu {
	return c.fileMenu
}

// SelectSlashCommand selects a slash command
func (c *Chat) SelectSlashCommand() {
	if cmd := c.slashMenu.GetSelectedCommand(); cmd != nil {
		newText := InsertSlashCommand(c.input.GetText(), c.slashQuery.SlashIndex, cmd.Name)
		c.input.SetText(newText)
		c.inSlashMode = false
	}
}

// SelectFileMention selects a file mention
func (c *Chat) SelectFileMention() {
	if result := c.fileMenu.GetSelectedResult(); result != nil {
		newText := InsertMention(c.input.GetText(), c.fileQuery.AtIndex, result.Path)
		c.input.SetText(newText)
		c.inFileMode = false
	}
}

// MoveSlashSelectionUp moves slash selection up
func (c *Chat) MoveSlashSelectionUp() {
	c.slashMenu.MoveSelectionUp()
}

// MoveSlashSelectionDown moves slash selection down
func (c *Chat) MoveSlashSelectionDown() {
	c.slashMenu.MoveSelectionDown()
}

// MoveFileSelectionUp moves file selection up
func (c *Chat) MoveFileSelectionUp() {
	c.fileMenu.MoveSelectionUp()
}

// MoveFileSelectionDown moves file selection down
func (c *Chat) MoveFileSelectionDown() {
	c.fileMenu.MoveSelectionDown()
}

// SetOnSend sets the send callback
func (c *Chat) SetOnSend(fn func(text string)) {
	c.onSend = fn
}

// SetOnCancel sets the cancel callback
func (c *Chat) SetOnCancel(fn func()) {
	c.onCancel = fn
	c.spinner.SetOnCancel(fn)
}

// Send sends the current message
func (c *Chat) Send() {
	if c.onSend != nil && c.input.GetText() != "" {
		text := c.input.GetText()
		c.onSend(text)
		c.input.Clear()
		c.inSlashMode = false
		c.inFileMode = false
	}
}

// Cancel cancels the current operation
func (c *Chat) Cancel() {
	if c.onCancel != nil {
		c.onCancel()
	}
}

// AddDiff adds a file diff
func (c *Chat) AddDiff(path, oldContent, newContent string) {
	c.diffView.AddDiff(path, oldContent, newContent)
	c.pendingChanges++
	c.statusBar.SetPendingChanges(c.pendingChanges)
}

// ClearDiffs clears all diffs
func (c *Chat) ClearDiffs() {
	c.diffView.Clear()
	c.pendingChanges = 0
	c.statusBar.SetPendingChanges(0)
}

// Render renders the full chat interface
func (c *Chat) Render() string {
	var content strings.Builder

	// Calculate available height
	availableHeight := c.height - 3 // Reserve space for status bar and input

	// Messages area
	messagesHeight := availableHeight - 2
	if c.inSlashMode || c.inFileMode {
		messagesHeight = availableHeight / 2
	}

	// Render messages (last N that fit)
	messages := c.renderMessages(messagesHeight)
	content.WriteString(messages)
	content.WriteString("\n")

	// Render menus if active
	if c.inSlashMode {
		menu := c.slashMenu.Render()
		content.WriteString(menu)
		content.WriteString("\n")
	} else if c.inFileMode {
		menu := c.fileMenu.Render()
		content.WriteString(menu)
		content.WriteString("\n")
	}

	// Render spinner if streaming
	if c.state == ChatStateStreaming {
		spinnerText := c.spinner.View()
		content.WriteString(spinnerText)
		content.WriteString("\n")
	}

	// Render input
	disabled := c.state == ChatStateStreaming || c.state == ChatStateAsking
	input := c.input.Render(c.mode, disabled)
	content.WriteString(input)
	content.WriteString("\n")

	// Render status bar
	status := c.statusBar.Render()
	content.WriteString(status)

	return content.String()
}

// renderMessages renders the message list
func (c *Chat) renderMessages(maxHeight int) string {
	if c.messageList.GetTotalHeight() <= maxHeight {
		return c.messageList.Render()
	}

	// Show only the last messages that fit
	messages := c.messageList.GetMessages()
	var visible []*ChatMessage
	height := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		msgHeight := msg.GetHeight()
		if height+msgHeight > maxHeight {
			break
		}
		visible = append([]*ChatMessage{msg}, visible...)
		height += msgHeight
	}

	var result strings.Builder
	for i, msg := range visible {
		result.WriteString(msg.Render())
		if i < len(visible)-1 {
			result.WriteString("\n\n")
		}
	}

	return result.String()
}

// Init implements tea.Model
func (c *Chat) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (c *Chat) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.SetDimensions(msg.Width, msg.Height)
		return c, nil

	case tea.KeyMsg:
		// Handle global keys
		switch msg.String() {
		case "ctrl+c":
			if c.state == ChatStateStreaming {
				c.Cancel()
			}
			return c, tea.Quit

		case "ctrl+d":
			return c, tea.Quit
		}

		// Handle input when not disabled
		if c.state != ChatStateStreaming && c.state != ChatStateAsking {
			switch msg.String() {
			case "enter":
				c.Send()
				return c, nil

			case "backspace":
				c.DeleteBeforeCursor()
				return c, nil

			case "delete":
				c.DeleteAfterCursor()
				return c, nil

			case "left":
				c.MoveCursorLeft()
				return c, nil

			case "right":
				c.MoveCursorRight()
				return c, nil

			case "up":
				if c.inSlashMode {
					c.MoveSlashSelectionUp()
				} else if c.inFileMode {
					c.MoveFileSelectionUp()
				}
				return c, nil

			case "down":
				if c.inSlashMode {
					c.MoveSlashSelectionDown()
				} else if c.inFileMode {
					c.MoveFileSelectionDown()
				}
				return c, nil

			case "tab":
				if c.inSlashMode {
					c.SelectSlashCommand()
				} else if c.inFileMode {
					c.SelectFileMention()
				}
				return c, nil

			case "esc":
				c.inSlashMode = false
				c.inFileMode = false
				return c, nil
			}

			// Regular character input
			if msg.Type == tea.KeyRunes {
				c.InsertAtCursor(string(msg.Runes))
				return c, nil
			}
		}
	}

	return c, nil
}

// View implements tea.Model
func (c *Chat) View() string {
	return c.Render()
}

// ChatProgram runs the chat as a Bubble Tea program
type ChatProgram struct {
	chat *Chat
}

// NewChatProgram creates a new chat program
func NewChatProgram(chat *Chat) *ChatProgram {
	return &ChatProgram{chat: chat}
}

// Run runs the chat program
func (cp *ChatProgram) Run() error {
	p := tea.NewProgram(cp.chat, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
