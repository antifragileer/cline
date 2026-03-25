// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Checkpoint represents a saved checkpoint
type Checkpoint struct {
	ID          string
	Description string
	Timestamp   time.Time
	CommitHash  string
}

// CheckpointAction represents the action to take on a checkpoint
type CheckpointAction int

const (
	// CheckpointActionNone means no action taken
	CheckpointActionNone CheckpointAction = iota
	// CheckpointActionRestore restores to this checkpoint
	CheckpointActionRestore
	// CheckpointActionDiff shows diff from this checkpoint
	CheckpointActionDiff
	// CheckpointActionCompare compares with another checkpoint
	CheckpointActionCompare
	// CheckpointActionDelete deletes this checkpoint
	CheckpointActionDelete
)

// CheckpointItem implements the list.Item interface
type CheckpointItem struct {
	checkpoint Checkpoint
}

// FilterValue returns the value to filter on
func (i CheckpointItem) FilterValue() string {
	return i.checkpoint.Description
}

// Title returns the item title
func (i CheckpointItem) Title() string {
	return i.checkpoint.Description
}

// Description returns the item description
func (i CheckpointItem) Description() string {
	return fmt.Sprintf("%s • %s", 
		i.checkpoint.ID[:8], 
		i.checkpoint.Timestamp.Format("2006-01-02 15:04:05"))
}

// CheckpointModel is the Bubble Tea model for checkpoint management
type CheckpointModel struct {
	width   int
	height  int
	list    list.Model
	
	// Styling
	titleStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	helpStyle        lipgloss.Style
	
	// State
	ready      bool
	done       bool
	selected   *Checkpoint
	action     CheckpointAction
	
	// Checkpoints
	checkpoints []Checkpoint
}

// CheckpointResult is sent when checkpoint selection is complete
type CheckpointResult struct {
	Checkpoint *Checkpoint
	Action     CheckpointAction
	Error      error
}

// NewCheckpointModel creates a new checkpoint model
func NewCheckpointModel(checkpoints []Checkpoint) CheckpointModel {
	// Convert checkpoints to list items
	items := make([]list.Item, len(checkpoints))
	for i, cp := range checkpoints {
		items[i] = CheckpointItem{checkpoint: cp}
	}

	// Create list
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Checkpoints"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	return CheckpointModel{
		checkpoints: checkpoints,
		list:        l,
		
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")),
		
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Background(lipgloss.Color("#1a1a1a")),
		
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),
	}
}

// SetDimensions sets the terminal dimensions
func (m *CheckpointModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-6) // Reserve space for header and help
}

// Init initializes the model
func (m CheckpointModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m CheckpointModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit

		case "enter":
			// Get selected item
			if item, ok := m.list.SelectedItem().(CheckpointItem); ok {
				m.selected = &item.checkpoint
				m.action = CheckpointActionRestore
				m.done = true
				return m, tea.Quit
			}

		case "d":
			// Diff action
			if item, ok := m.list.SelectedItem().(CheckpointItem); ok {
				m.selected = &item.checkpoint
				m.action = CheckpointActionDiff
				m.done = true
				return m, tea.Quit
			}

		case "c":
			// Compare action
			if item, ok := m.list.SelectedItem().(CheckpointItem); ok {
				m.selected = &item.checkpoint
				m.action = CheckpointActionCompare
				m.done = true
				return m, tea.Quit
			}

		case "x":
			// Delete action
			if item, ok := m.list.SelectedItem().(CheckpointItem); ok {
				m.selected = &item.checkpoint
				m.action = CheckpointActionDelete
				m.done = true
				return m, tea.Quit
			}

		case "r":
			// Refresh list
			return m, nil
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the checkpoint menu
func (m CheckpointModel) View() string {
	if !m.ready {
		return "Loading checkpoints..."
	}

	var content strings.Builder

	// Header
	header := m.titleStyle.Render("📍 Checkpoint Manager")
	content.WriteString(header)
	content.WriteString("\n\n")

	// List
	content.WriteString(m.list.View())
	content.WriteString("\n")

	// Help text
	help := m.helpStyle.Render(
		"enter: restore • d: diff • c: compare • x: delete • /: filter • q: quit")
	content.WriteString(help)

	return content.String()
}

// IsDone returns true if selection is complete
func (m CheckpointModel) IsDone() bool {
	return m.done
}

// GetSelected returns the selected checkpoint
func (m CheckpointModel) GetSelected() *Checkpoint {
	return m.selected
}

// GetAction returns the selected action
func (m CheckpointModel) GetAction() CheckpointAction {
	return m.action
}

// ShowCheckpointMenu shows a checkpoint menu and returns the selection
func ShowCheckpointMenu(checkpoints []Checkpoint) (*Checkpoint, CheckpointAction, error) {
	model := NewCheckpointModel(checkpoints)
	
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	m, err := p.Run()
	if err != nil {
		return nil, CheckpointActionNone, err
	}
	
	cpModel, ok := m.(CheckpointModel)
	if !ok {
		return nil, CheckpointActionNone, fmt.Errorf("unexpected model type")
	}
	
	return cpModel.GetSelected(), cpModel.GetAction(), nil
}

// ShowCheckpointMenuWithCurrent shows checkpoint menu with current checkpoint highlighted
func ShowCheckpointMenuWithCurrent(checkpoints []Checkpoint, currentID string) (*Checkpoint, CheckpointAction, error) {
	model := NewCheckpointModel(checkpoints)
	
	// Find and select current checkpoint
	for i, cp := range checkpoints {
		if cp.ID == currentID {
			model.list.Select(i)
			break
		}
	}
	
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	m, err := p.Run()
	if err != nil {
		return nil, CheckpointActionNone, err
	}
	
	cpModel, ok := m.(CheckpointModel)
	if !ok {
		return nil, CheckpointActionNone, fmt.Errorf("unexpected model type")
	}
	
	return cpModel.GetSelected(), cpModel.GetAction(), nil
}

// FormatCheckpointDescription formats a checkpoint description for display
func FormatCheckpointDescription(cp Checkpoint) string {
	return fmt.Sprintf("%s - %s", 
		cp.Timestamp.Format("Jan 02 15:04"),
		cp.Description)
}

// GetCheckpointMenuItems returns checkpoint items for a simple menu
func GetCheckpointMenuItems(checkpoints []Checkpoint) []string {
	items := make([]string, len(checkpoints))
	for i, cp := range checkpoints {
		items[i] = FormatCheckpointDescription(cp)
	}
	return items
}