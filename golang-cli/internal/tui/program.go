// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Program wraps a Bubble Tea program for TUI integration
type Program struct {
	program *tea.Program
}

// NewProgram creates a new TUI program
func NewProgram(model tea.Model, opts ...tea.ProgramOption) *Program {
	p := tea.NewProgram(model, opts...)
	return &Program{
		program: p,
	}
}

// Send sends a message to the program
func (p *Program) Send(msg tea.Msg) {
	if p.program != nil {
		p.program.Send(msg)
	}
}

// Run starts the program
func (p *Program) Run() (tea.Model, error) {
	if p.program == nil {
		return nil, nil
	}
	return p.program.Run()
}

// Quit signals the program to quit
func (p *Program) Quit() {
	if p.program != nil {
		p.program.Quit()
	}
}

// Kill terminates the program immediately
func (p *Program) Kill() {
	if p.program != nil {
		p.program.Kill()
	}
}

// Wait waits for the program to finish
func (p *Program) Wait() (tea.Model, error) {
	if p.program == nil {
		return nil, nil
	}
	return p.program.Run()
}
