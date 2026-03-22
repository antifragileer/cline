// Package tui provides a Bubble Tea based terminal user interface framework.
// It implements the Model-Update-View architecture for building interactive
// terminal applications with support for both TUI and plain text modes.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Mode represents the operating mode of the TUI.
type Mode int

const (
	// ModeTUI runs the application in interactive TUI mode.
	ModeTUI Mode = iota
	// ModePlain runs the application in plain text mode without TUI.
	ModePlain
)

// Dimensions represents terminal window dimensions.
type Dimensions struct {
	Width  int
	Height int
}

// WindowSizeMsg is a custom message for window resize events.
type WindowSizeMsg tea.WindowSizeMsg

// Model represents the TUI application state.
type Model struct {
	// mode indicates whether running in TUI or plain mode.
	mode Mode

	// dimensions holds the current terminal dimensions.
	dimensions Dimensions

	// content is the main content to display.
	content string

	// title is the application title displayed in the header.
	title string

	// ready indicates if the model has been initialized.
	ready bool

	// err holds any error state.
	err error

	// shutdownCallbacks are functions to call during graceful shutdown.
	shutdownCallbacks []func()
}

// NewModel creates a new TUI model with default values.
func NewModel(title string) Model {
	return Model{
		title:             title,
		mode:              ModeTUI,
		dimensions:        Dimensions{Width: 80, Height: 24},
		shutdownCallbacks: make([]func(), 0),
	}
}

// NewPlainModel creates a new model configured for plain text mode.
func NewPlainModel() Model {
	return Model{
		mode:              ModePlain,
		dimensions:        Dimensions{Width: 80, Height: 24},
		shutdownCallbacks: make([]func(), 0),
	}
}

// Mode returns the current operating mode.
func (m Model) Mode() Mode {
	return m.mode
}

// IsTUI returns true if running in TUI mode.
func (m Model) IsTUI() bool {
	return m.mode == ModeTUI
}

// IsPlain returns true if running in plain mode.
func (m Model) IsPlain() bool {
	return m.mode == ModePlain
}

// Dimensions returns the current terminal dimensions.
func (m Model) Dimensions() Dimensions {
	return m.dimensions
}

// Width returns the terminal width.
func (m Model) Width() int {
	return m.dimensions.Width
}

// Height returns the terminal height.
func (m Model) Height() int {
	return m.dimensions.Height
}

// Content returns the current content.
func (m Model) Content() string {
	return m.content
}

// SetContent sets the content to display.
func (m *Model) SetContent(content string) {
	m.content = content
}

// Title returns the application title.
func (m Model) Title() string {
	return m.title
}

// SetTitle sets the application title.
func (m *Model) SetTitle(title string) {
	m.title = title
}

// Ready returns true if the model has been initialized.
func (m Model) Ready() bool {
	return m.ready
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

// Init implements the bubbletea.Model interface.
// It returns the initial command to run when the TUI starts.
func (m Model) Init() tea.Cmd {
	return nil
}