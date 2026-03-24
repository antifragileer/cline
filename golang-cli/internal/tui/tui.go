package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Program represents a running TUI program.
type Program struct {
	// program is the underlying bubbletea program.
	program *tea.Program

	// model is the TUI model.
	model Model

	// options are the program options.
	options ProgramOptions

	// ctx is the context for cancellation.
	ctx context.Context

	// cancel is the cancel function for the context.
	cancel context.CancelFunc
}

// ProgramOptions configures the TUI program.
type ProgramOptions struct {
	// Title is the application title.
	Title string

	// Mode is the operating mode (TUI or Plain).
	Mode Mode

	// Input is the input reader (defaults to os.Stdin).
	Input io.Reader

	// Output is the output writer (defaults to os.Stdout).
	Output io.Writer

	// AltScreen enables alternate screen buffer.
	AltScreen bool

	// Mouse enables mouse support.
	Mouse bool

	// InitialContent is the initial content to display.
	InitialContent string

	// OnShutdown is called when the program shuts down.
	OnShutdown func()
}

// DefaultProgramOptions returns default program options.
func DefaultProgramOptions() ProgramOptions {
	return ProgramOptions{
		Title:     "Cline",
		Mode:      ModeTUI,
		Input:     os.Stdin,
		Output:    os.Stdout,
		AltScreen: true,
		Mouse:     false,
	}
}

// NewProgram creates a new TUI program with the given options.
func NewProgram(opts ProgramOptions) (*Program, error) {
	// Get terminal dimensions if in TUI mode
	if opts.Mode == ModeTUI && opts.Input == os.Stdin {
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err == nil {
			// We have terminal dimensions
			_ = width
			_ = height
		}
	}

	// Create the model
	var model Model
	if opts.Mode == ModePlain {
		model = NewPlainModel()
		model.SetContent(opts.InitialContent)
	} else {
		model = NewModel(opts.Title)
		model.SetContent(opts.InitialContent)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	p := &Program{
		model:   model,
		options: opts,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Register shutdown callback
	if opts.OnShutdown != nil {
		p.model.RegisterShutdownCallback(opts.OnShutdown)
	}

	return p, nil
}

// Start starts the TUI program and blocks until it exits.
// Returns the final model state and any error.
func (p *Program) Start() (Model, error) {
	if p.options.Mode == ModePlain {
		return p.runPlain()
	}
	return p.runTUI()
}

// runTUI runs the program in TUI mode using bubbletea.
func (p *Program) runTUI() (Model, error) {
	// Create bubbletea options
	teaOpts := []tea.ProgramOption{
		tea.WithInput(p.options.Input),
		tea.WithOutput(p.options.Output),
	}

	if p.options.AltScreen {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}

	if p.options.Mouse {
		teaOpts = append(teaOpts, tea.WithMouseCellMotion())
	}

	// Create the program
	p.program = tea.NewProgram(p.model, teaOpts...)

	// Setup signal handling
	sigChan := SetupSignalHandling()

	// Run signal handling in background
	go func() {
		for {
			select {
			case sig := <-sigChan:
				switch sig {
				case os.Interrupt:
					p.Shutdown()
					return
				case syscall.SIGTERM:
					p.Shutdown()
					return
				}
			case <-p.ctx.Done():
				return
			}
		}
	}()

	// Run the program
	finalModel, err := p.program.Run()
	if err != nil {
		return Model{}, fmt.Errorf("program error: %w", err)
	}

	// Get the final model state
	if m, ok := finalModel.(Model); ok {
		return m, nil
	}

	return Model{}, fmt.Errorf("unexpected model type")
}

// runPlain runs the program in plain mode without TUI.
func (p *Program) runPlain() (Model, error) {
	// In plain mode, just output the content directly
	if p.options.InitialContent != "" {
		fmt.Fprintln(p.options.Output, p.options.InitialContent)
	}

	// Call shutdown callback if registered
	if p.options.OnShutdown != nil {
		p.options.OnShutdown()
	}

	return p.model, nil
}

// Shutdown gracefully shuts down the TUI program.
func (p *Program) Shutdown() {
	// Cancel context to stop background goroutines
	if p.cancel != nil {
		p.cancel()
	}

	// Shutdown the model (calls registered callbacks)
	p.model.Shutdown()

	// Quit the bubbletea program if running
	if p.program != nil {
		p.program.Quit()
	}
}

// Send sends a message to the running program.
func (p *Program) Send(msg tea.Msg) {
	if p.program != nil {
		p.program.Send(msg)
	}
}

// Quit sends a quit message to the program.
func (p *Program) Quit() {
	if p.program != nil {
		p.program.Quit()
	}
}

// IsRunning returns true if the program is currently running.
func (p *Program) IsRunning() bool {
	return p.program != nil
}

// Run is a convenience function that creates and runs a new TUI program.
// It handles the full lifecycle from initialization to shutdown.
func Run(opts ProgramOptions) error {
	program, err := NewProgram(opts)
	if err != nil {
		return err
	}

	_, err = program.Start()
	return err
}

// RunSimple runs the TUI with default options and the given content.
func RunSimple(content string) error {
	opts := DefaultProgramOptions()
	opts.InitialContent = content
	return Run(opts)
}

// RunPlain runs in plain mode with the given content.
func RunPlain(content string) error {
	opts := DefaultProgramOptions()
	opts.Mode = ModePlain
	opts.InitialContent = content
	return Run(opts)
}

// GetTerminalDimensions returns the current terminal width and height.
func GetTerminalDimensions() (width, height int, err error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

// IsTerminal returns true if stdout is a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// SupportsTUI returns true if the current environment supports TUI mode.
func SupportsTUI() bool {
	return IsTerminal() && os.Getenv("TERM") != "dumb"
}