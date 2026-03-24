package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3 // Reserve space for input

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.inputFocused && m.textInput.Value() != "" {
				// Add user message
				m.messages = append(m.messages, Message{
					Type:    MessageTypeUser,
					Content: m.textInput.Value(),
				})

				// Add to history
				m.inputHistory = append(m.inputHistory, m.textInput.Value())
				m.inputHistoryIndex = -1

				// Clear input
				m.textInput.SetValue("")

				// TODO: Send to backend
			}

		case tea.KeyUp:
			if m.inputFocused && len(m.inputHistory) > 0 {
				if m.inputHistoryIndex < len(m.inputHistory)-1 {
					m.inputHistoryIndex++
					m.textInput.SetValue(m.inputHistory[len(m.inputHistory)-1-m.inputHistoryIndex])
				}
			}

		case tea.KeyDown:
			if m.inputFocused && m.inputHistoryIndex >= 0 {
				m.inputHistoryIndex--
				if m.inputHistoryIndex < 0 {
					m.textInput.SetValue("")
				} else {
					m.textInput.SetValue(m.inputHistory[len(m.inputHistory)-1-m.inputHistoryIndex])
				}
			}

		case tea.KeyTab:
			m.inputFocused = !m.inputFocused
			if m.inputFocused {
				m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}

		case tea.KeyEsc:
			if m.currentView == settingsView {
				m.currentView = chatView
			}

		case tea.KeyPgUp:
			// Scroll up in viewport
			m.viewport.LineUp(3)

		case tea.KeyPgDown:
			// Scroll down in viewport
			m.viewport.LineDown(3)

		case tea.KeyHome:
			if m.inputFocused {
				m.textInput.SetCursor(0)
			}

		case tea.KeyEnd:
			if m.inputFocused {
				m.textInput.SetCursor(len(m.textInput.Value()))
			}

		case tea.KeyCtrlL:
			// Clear screen - reset messages
			m.messages = []Message{}

		case tea.KeyRunes:
			switch msg.String() {
			case "q":
				if !m.inputFocused && m.currentView == welcomeView {
					return m, tea.Quit
				}
			case "n":
				if m.currentView == welcomeView {
					m.currentView = chatView
					m.showWelcome = false
					m.inputFocused = true
					m.textInput.Focus()
				} else if m.pendingApproval != nil {
					// Handle rejection
					m.pendingApproval = nil
				}
			case "s":
				if m.currentView == welcomeView {
					m.currentView = settingsView
				}
			case "y":
				if m.pendingApproval != nil {
					// Handle approval
					m.pendingApproval = nil
				}
			}
		}

	case AddMessageMsg:
		m.messages = append(m.messages, Message{
			Type:    MessageTypeUser,
			Content: msg.Content,
		})
		m.viewport.GotoBottom()

	case ErrorMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		}
	}

	// Update components
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// SignalCmd creates a command that listens for system signals.
func SignalCmd(sigChan chan os.Signal) tea.Cmd {
	return func() tea.Msg {
		sig := <-sigChan
		switch sig {
		case os.Interrupt:
			return ShutdownMsg{}
		default:
			// Window resize and other signals are handled automatically by bubbletea
			return nil
		}
	}
}

// BatchCmd creates a command that sends a batch of messages.
func BatchCmd(msgs ...tea.Msg) tea.Cmd {
	return func() tea.Msg {
		// Return the first message, subsequent messages can be sent via tea.Batch
		if len(msgs) > 0 {
			return msgs[0]
		}
		return nil
	}
}

// UpdateContent creates a command to update the content.
func UpdateContent(content string) tea.Cmd {
	return func() tea.Msg {
		return ContentMsg{Content: content}
	}
}

// ShutdownCmd creates a command to shut down the TUI.
func ShutdownCmd() tea.Cmd {
	return func() tea.Msg {
		return ShutdownMsg{}
	}
}

// ShutdownMsg is sent when the TUI should shut down
type ShutdownMsg struct{}

// ContentMsg is sent when the content should be updated
type ContentMsg struct {
	Content string
}