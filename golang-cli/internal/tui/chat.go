// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/glamour"
)

// ChatModel is the Bubble Tea model for the chat view
type ChatModel struct {
	width       int
	height      int
	viewport    viewport.Model
	textInput   textinput.Model
	messages    []Message
	isStreaming bool
	streamContent strings.Builder
	
	// Styling
	userStyle       lipgloss.Style
	assistantStyle  lipgloss.Style
	systemStyle     lipgloss.Style
	timestampStyle  lipgloss.Style
	codeBlockStyle  lipgloss.Style
	
	// Markdown renderer
	mdRenderer    *glamour.TermRenderer
	
	// State
	ready         bool
	focused       bool
	onSubmit      func(string) error
	onInterrupt   func() error
}

// ChatUpdateMsg is sent when the chat should be updated
type ChatUpdateMsg struct {
	Messages []Message
}

// StreamChunkMsg is sent when a new chunk of streaming content arrives
type StreamChunkMsg struct {
	Chunk string
	Done  bool
}

// ChatErrorMsg is sent when a chat error occurs
type ChatErrorMsg struct {
	Error error
}

// NewChatModel creates a new chat model
func NewChatModel() ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 10000
	ti.Width = 80
	
	return ChatModel{
		textInput: ti,
		messages:  []Message{},
		userStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
		assistantStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")),
		systemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		timestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),
		codeBlockStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E1E")).
			Foreground(lipgloss.Color("#D4D4D4")).
			Padding(1, 2),
		focused: true,
	}
}

// SetDimensions sets the terminal dimensions
func (m *ChatModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 3 // Reserve space for input
	m.textInput.Width = width - 4
}

// Init initializes the model
func (m ChatModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-3)
			m.ready = true
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.isStreaming && m.onInterrupt != nil {
				return m, func() tea.Msg {
					if err := m.onInterrupt(); err != nil {
						return ChatErrorMsg{Error: err}
					}
					return nil
				}
			}
			return m, tea.Quit

		case "enter":
			if m.textInput.Value() != "" {
				content := m.textInput.Value()
				m.textInput.SetValue("")
				
				// Add user message
				m.messages = append(m.messages, Message{
					Type:      MessageTypeUser,
					Content:   content,
					Timestamp: time.Now(),
				})
				
				// Scroll to bottom
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				
				// Call submit handler
				if m.onSubmit != nil {
					return m, func() tea.Msg {
						if err := m.onSubmit(content); err != nil {
							return ChatErrorMsg{Error: err}
						}
						return nil
					}
				}
			}
			return m, nil

		case "esc":
			// Clear input
			m.textInput.SetValue("")
			return m, nil
		}

	case StreamChunkMsg:
		if msg.Done {
			m.isStreaming = false
			// Finalize the streaming message
			if m.streamContent.Len() > 0 {
				content := m.streamContent.String()
				m.streamContent.Reset()
				m.messages = append(m.messages, Message{
					Type:      MessageTypeSay,
					Content:   content,
					Timestamp: time.Now(),
					Partial:   false,
				})
			}
		} else {
			m.isStreaming = true
			m.streamContent.WriteString(msg.Chunk)
			// Update viewport content
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case ChatUpdateMsg:
		m.messages = msg.Messages
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case ChatErrorMsg:
		// Add error as system message
		m.messages = append(m.messages, Message{
			Type:      MessageTypeError,
			Content:   fmt.Sprintf("Error: %v", msg.Error),
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	// Update text input
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View renders the chat view
func (m ChatModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content strings.Builder

	// Render messages viewport
	content.WriteString(m.viewport.View())
	content.WriteString("\n")

	// Render input area
	content.WriteString(m.renderInputArea())

	return content.String()
}

// renderMessages renders all messages
func (m ChatModel) renderMessages() string {
	var content strings.Builder

	for i, msg := range m.messages {
		content.WriteString(m.renderMessage(msg))
		if i < len(m.messages)-1 {
			content.WriteString("\n\n")
		}
	}

	// Render streaming content if any
	if m.isStreaming && m.streamContent.Len() > 0 {
		if len(m.messages) > 0 {
			content.WriteString("\n\n")
		}
		streamMsg := Message{
			Type:      MessageTypeSay,
			Content:   m.streamContent.String(),
			Timestamp: time.Now(),
			Partial:   true,
		}
		content.WriteString(m.renderMessage(streamMsg))
	}

	return content.String()
}

// renderMessage renders a single message
func (m ChatModel) renderMessage(msg Message) string {
	var content strings.Builder

	// Role header
	roleStr := m.formatRole(msg.Type)
	content.WriteString(roleStr)
	content.WriteString("\n")

	// Message content
	renderedContent := m.renderContent(msg.Content, string(msg.Type))
	content.WriteString(renderedContent)

	// Timestamp
	if !msg.Timestamp.IsZero() {
		content.WriteString("\n")
		content.WriteString(m.timestampStyle.Render(msg.Timestamp.Format("15:04")))
	}

	return content.String()
}

// formatRole formats the role indicator
func (m ChatModel) formatRole(msgType MessageType) string {
	switch msgType {
	case MessageTypeUser:
		return m.userStyle.Render("You")
	case MessageTypeSay, MessageTypeAsk:
		if m.isStreaming {
			return m.assistantStyle.Render("Cline ●")
		}
		return m.assistantStyle.Render("Cline")
	case MessageTypeError:
		return m.systemStyle.Render("Error")
	case MessageTypeToolUse:
		return m.systemStyle.Render("Tool")
	case MessageTypeToolResult:
		return m.systemStyle.Render("Result")
	default:
		return m.systemStyle.Render(string(msgType))
	}
}

// renderContent renders message content with markdown support
func (m ChatModel) renderContent(content string, msgType string) string {
	if msgType == string(MessageTypeUser) {
		// Simple text for user messages
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			Render(content)
	}

	// For assistant messages, try to render markdown
	rendered, err := m.renderMarkdown(content)
	if err != nil {
		// Fallback to plain text
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			Render(content)
	}

	return rendered
}

// renderMarkdown renders markdown content using glamour
func (m *ChatModel) renderMarkdown(content string) (string, error) {
	if m.mdRenderer == nil {
		var err error
		m.mdRenderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.width-4),
		)
		if err != nil {
			return "", err
		}
	}

	rendered, err := m.mdRenderer.Render(content)
	if err != nil {
		return "", err
	}

	// Remove trailing newlines
	return strings.TrimRight(rendered, "\n"), nil
}

// renderInputArea renders the input area
func (m ChatModel) renderInputArea() string {
	var content strings.Builder

	// Separator line
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040"))
	content.WriteString(separatorStyle.Render(strings.Repeat("─", m.width)))
	content.WriteString("\n")

	// Input prompt
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)
	content.WriteString(promptStyle.Render("> "))

	// Text input
	content.WriteString(m.textInput.View())

	return content.String()
}

// AddMessage adds a message to the chat
func (m *ChatModel) AddMessage(msgType MessageType, content string) {
	m.messages = append(m.messages, Message{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
		Partial:   false,
	})
}

// AddStreamingChunk adds a chunk to the streaming content
func (m *ChatModel) AddStreamingChunk(chunk string) {
	m.streamContent.WriteString(chunk)
}

// FinishStreaming finalizes the streaming message
func (m *ChatModel) FinishStreaming() {
	if m.streamContent.Len() > 0 {
		content := m.streamContent.String()
		m.streamContent.Reset()
		m.messages = append(m.messages, Message{
			Type:      MessageTypeSay,
			Content:   content,
			Timestamp: time.Now(),
			Partial:   false,
		})
		m.isStreaming = false
	}
}

// ClearMessages clears all messages
func (m *ChatModel) ClearMessages() {
	m.messages = []Message{}
	m.streamContent.Reset()
	m.isStreaming = false
}

// GetMessages returns the current messages
func (m ChatModel) GetMessages() []Message {
	return m.messages
}

// SetOnSubmit sets the submit handler
func (m *ChatModel) SetOnSubmit(fn func(string) error) {
	m.onSubmit = fn
}

// SetOnInterrupt sets the interrupt handler
func (m *ChatModel) SetOnInterrupt(fn func() error) {
	m.onInterrupt = fn
}

// IsStreaming returns true if currently streaming
func (m ChatModel) IsStreaming() bool {
	return m.isStreaming
}

// SetStreaming sets the streaming state
func (m *ChatModel) SetStreaming(streaming bool) {
	m.isStreaming = streaming
	if !streaming {
		m.FinishStreaming()
	}
}

// ScrollToBottom scrolls the viewport to the bottom
func (m *ChatModel) ScrollToBottom() {
	m.viewport.GotoBottom()
}

// RefreshContent refreshes the viewport content
func (m *ChatModel) RefreshContent() {
	m.viewport.SetContent(m.renderMessages())
}

// ChatScreen runs an interactive chat screen and returns the conversation
func ChatScreen(initialPrompt string, onSubmit func(string) error, onInterrupt func() error) ([]Message, error) {
	model := NewChatModel()
	model.SetOnSubmit(onSubmit)
	model.SetOnInterrupt(onInterrupt)
	
	if initialPrompt != "" {
		model.AddMessage(MessageTypeUser, initialPrompt)
	}
	
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	
	chatModel, ok := m.(ChatModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	
	return chatModel.GetMessages(), nil
}
