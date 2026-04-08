// Package mode provides mode switching logic for the CLI application.
// It implements a strategy pattern for mode-specific behavior, supporting
// Interactive (Bubble Tea), Plain (text output), and JSON (structured) modes.
package mode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// CLIMode represents the CLI operating mode.
type CLIMode int

const (
	// CLIModeInteractive runs the application in interactive TUI mode using Bubble Tea.
	CLIModeInteractive CLIMode = iota
	// CLIModePlain runs the application in plain text mode without TUI.
	CLIModePlain
	// CLIModeJSON runs the application in structured JSON output mode.
	CLIModeJSON
)

// String returns the string representation of the mode.
func (m CLIMode) String() string {
	switch m {
	case CLIModeInteractive:
		return "interactive"
	case CLIModePlain:
		return "plain"
	case CLIModeJSON:
		return "json"
	default:
		return "unknown"
	}
}

// ParseMode parses a mode string into a CLIMode value.
func ParseMode(s string) (CLIMode, error) {
	switch strings.ToLower(s) {
	case "interactive", "tui":
		return CLIModeInteractive, nil
	case "plain", "text":
		return CLIModePlain, nil
	case "json":
		return CLIModeJSON, nil
	default:
		return CLIModePlain, fmt.Errorf("unknown mode: %s", s)
	}
}

// ModeHandler defines the interface for mode-specific behavior.
// Implementations of this interface provide mode-specific initialization,
// output handling, and cleanup operations.
type ModeHandler interface {
	// Initialize prepares the handler for use.
	Initialize() error
	// Output writes a message to the appropriate output destination.
	Output(message string) error
	// OutputStructured writes structured data to the output destination.
	OutputStructured(data interface{}) error
	// Error writes an error message to the appropriate error destination.
	Error(err error) error
	// IsInteractive returns true if the handler supports interactive operations.
	IsInteractive() bool
	// SupportsInput returns true if the handler can read user input.
	SupportsInput() bool
	// ReadInput reads user input and returns it.
	ReadInput() (string, error)
	// Cleanup performs any necessary cleanup operations.
	Cleanup() error
}

// ModeSwitcher manages mode switching and handler initialization.
// It uses the strategy pattern to delegate operations to the appropriate
// mode-specific handler.
type ModeSwitcher struct {
	// mode is the current operating mode.
	mode CLIMode

	// handler is the active mode handler.
	handler ModeHandler

	// options contains configuration options for the switcher.
	options SwitcherOptions

	// output is the output writer.
	output io.Writer

	// errOutput is the error output writer.
	errOutput io.Writer

	// input is the input reader.
	input io.Reader
}

// SwitcherOptions configures the ModeSwitcher.
type SwitcherOptions struct {
	// Mode is the desired operating mode.
	Mode CLIMode

	// ForceMode, if true, prevents automatic mode detection.
	ForceMode bool

	// Output is the output writer (defaults to os.Stdout).
	Output io.Writer

	// ErrOutput is the error output writer (defaults to os.Stderr).
	ErrOutput io.Writer

	// Input is the input reader (defaults to os.Stdin).
	Input io.Reader

	// Title is the application title (used in TUI mode).
	Title string

	// DisableColor disables colored output.
	DisableColor bool
}

// DefaultSwitcherOptions returns default options for the switcher.
func DefaultSwitcherOptions() SwitcherOptions {
	return SwitcherOptions{
		Mode:         CLIModeInteractive,
		ForceMode:    false,
		Output:       os.Stdout,
		ErrOutput:    os.Stderr,
		Input:        os.Stdin,
		Title:        "Cline",
		DisableColor: false,
	}
}

// NewModeSwitcher creates a new ModeSwitcher with the given options.
// It initializes the appropriate handler based on the mode and options.
func NewModeSwitcher(opts SwitcherOptions) (*ModeSwitcher, error) {
	// Set defaults for unset options
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.ErrOutput == nil {
		opts.ErrOutput = os.Stderr
	}
	if opts.Input == nil {
		opts.Input = os.Stdin
	}

	switcher := &ModeSwitcher{
		options:   opts,
		mode:      opts.Mode,
		output:    opts.Output,
		errOutput: opts.ErrOutput,
		input:     opts.Input,
	}

	// Initialize the appropriate handler
	handler, err := switcher.createHandler(opts.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to create mode handler: %w", err)
	}

	switcher.handler = handler

	return switcher, nil
}

// createHandler creates the appropriate handler for the given mode.
func (s *ModeSwitcher) createHandler(mode CLIMode) (ModeHandler, error) {
	switch mode {
	case CLIModeInteractive:
		return NewInteractiveHandler(InteractiveHandlerOptions{
			Title:        s.options.Title,
			Output:       s.output,
			ErrOutput:    s.errOutput,
			Input:        s.input,
			DisableColor: s.options.DisableColor,
		})
	case CLIModePlain:
		return NewPlainHandler(PlainHandlerOptions{
			Output:       s.output,
			ErrOutput:    s.errOutput,
			Input:        s.input,
			DisableColor: s.options.DisableColor,
		})
	case CLIModeJSON:
		return NewJSONHandler(JSONHandlerOptions{
			Output:    s.output,
			ErrOutput: s.errOutput,
			Input:     s.input,
		})
	default:
		return nil, fmt.Errorf("unsupported mode: %v", mode)
	}
}

// Initialize initializes the mode handler.
func (s *ModeSwitcher) Initialize() error {
	return s.handler.Initialize()
}

// SwitchMode switches to a new mode, creating a new handler.
func (s *ModeSwitcher) SwitchMode(mode CLIMode) error {
	// Cleanup current handler
	if s.handler != nil {
		if err := s.handler.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup previous handler: %w", err)
		}
	}

	// Create new handler
	handler, err := s.createHandler(mode)
	if err != nil {
		return fmt.Errorf("failed to create handler for mode %v: %w", mode, err)
	}

	// Initialize new handler
	if err := handler.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize handler for mode %v: %w", mode, err)
	}

	s.mode = mode
	s.handler = handler

	return nil
}

// GetMode returns the current mode.
func (s *ModeSwitcher) GetMode() CLIMode {
	return s.mode
}

// GetHandler returns the current mode handler.
func (s *ModeSwitcher) GetHandler() ModeHandler {
	return s.handler
}

// Output writes a message using the current handler.
func (s *ModeSwitcher) Output(message string) error {
	return s.handler.Output(message)
}

// Outputf writes a formatted message using the current handler.
func (s *ModeSwitcher) Outputf(format string, args ...interface{}) error {
	return s.handler.Output(fmt.Sprintf(format, args...))
}

// OutputStructured writes structured data using the current handler.
func (s *ModeSwitcher) OutputStructured(data interface{}) error {
	return s.handler.OutputStructured(data)
}

// Error writes an error using the current handler.
func (s *ModeSwitcher) Error(err error) error {
	return s.handler.Error(err)
}

// Errorf writes a formatted error using the current handler.
func (s *ModeSwitcher) Errorf(format string, args ...interface{}) error {
	return s.handler.Error(fmt.Errorf(format, args...))
}

// ReadInput reads input using the current handler.
func (s *ModeSwitcher) ReadInput() (string, error) {
	return s.handler.ReadInput()
}

// IsInteractive returns true if the current mode is interactive.
func (s *ModeSwitcher) IsInteractive() bool {
	return s.handler.IsInteractive()
}

// SupportsInput returns true if the current handler supports input.
func (s *ModeSwitcher) SupportsInput() bool {
	return s.handler.SupportsInput()
}

// Cleanup performs cleanup for the current handler.
func (s *ModeSwitcher) Cleanup() error {
	return s.handler.Cleanup()
}

// AutoDetectMode detects the best mode based on the environment.
// It checks for TTY availability, environment variables, and other factors.
func AutoDetectMode() CLIMode {
	// Check if stdout is a terminal
	if !isTerminal() {
		return CLIModePlain
	}

	// Check for dumb terminal
	if os.Getenv("TERM") == "dumb" {
		return CLIModePlain
	}

	// Check for CI environment
	if isCIEnvironment() {
		return CLIModePlain
	}

	// Default to interactive if terminal supports it
	return CLIModeInteractive
}

// isTerminal returns true if stdout is a terminal.
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fileInfo.Mode()&os.ModeCharDevice == os.ModeCharDevice
}

// isCIEnvironment returns true if running in a CI environment.
func isCIEnvironment() bool {
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"TRAVIS",
		"CIRCLECI",
		"BUILDKITE",
		"DRONE",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}

// GetModeFromFlags determines the mode from command-line flags.
// Returns the mode and a boolean indicating if the mode was explicitly forced.
func GetModeFromFlags(interactive, plain, json bool) (CLIMode, bool) {
	// Check for explicit flags in order of precedence
	if interactive {
		return CLIModeInteractive, true
	}
	if plain {
		return CLIModePlain, true
	}
	if json {
		return CLIModeJSON, true
	}

	// No explicit flag, use auto-detection
	return AutoDetectMode(), false
}

// ==================== INTERACTIVE HANDLER ====================

// InteractiveHandler handles interactive TUI mode using Bubble Tea.
type InteractiveHandler struct {
	options     InteractiveHandlerOptions
	initialized bool
	title       string
}

// InteractiveHandlerOptions configures the InteractiveHandler.
type InteractiveHandlerOptions struct {
	Title        string
	Output       io.Writer
	ErrOutput    io.Writer
	Input        io.Reader
	DisableColor bool
}

// NewInteractiveHandler creates a new interactive handler.
func NewInteractiveHandler(opts InteractiveHandlerOptions) (*InteractiveHandler, error) {
	if opts.Title == "" {
		opts.Title = "Cline"
	}

	return &InteractiveHandler{
		options: opts,
		title:   opts.Title,
	}, nil
}

// Initialize prepares the handler for use.
func (h *InteractiveHandler) Initialize() error {
	h.initialized = true
	return nil
}

// Output writes a message to the output.
// In interactive mode, this delegates to the TUI system.
func (h *InteractiveHandler) Output(message string) error {
	// In a full implementation, this would send to the TUI model
	// For now, write directly to output
	_, err := fmt.Fprintln(h.options.Output, message)
	return err
}

// OutputStructured writes structured data to the output.
func (h *InteractiveHandler) OutputStructured(data interface{}) error {
	// In interactive mode, format nicely for display
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.options.Output, string(jsonData))
	return err
}

// Error writes an error message.
func (h *InteractiveHandler) Error(err error) error {
	_, writeErr := fmt.Fprintln(h.options.ErrOutput, err.Error())
	return writeErr
}

// IsInteractive returns true.
func (h *InteractiveHandler) IsInteractive() bool {
	return true
}

// SupportsInput returns true.
func (h *InteractiveHandler) SupportsInput() bool {
	return true
}

// ReadInput reads user input.
func (h *InteractiveHandler) ReadInput() (string, error) {
	// In a full implementation, this would use the TUI input system
	var input string
	_, err := fmt.Fscanln(h.options.Input, &input)
	return input, err
}

// Cleanup performs cleanup operations.
func (h *InteractiveHandler) Cleanup() error {
	h.initialized = false
	return nil
}

// ==================== PLAIN HANDLER ====================

// PlainHandler handles plain text output mode.
type PlainHandler struct {
	options     PlainHandlerOptions
	initialized bool
}

// PlainHandlerOptions configures the PlainHandler.
type PlainHandlerOptions struct {
	Output       io.Writer
	ErrOutput    io.Writer
	Input        io.Reader
	DisableColor bool
}

// NewPlainHandler creates a new plain handler.
func NewPlainHandler(opts PlainHandlerOptions) (*PlainHandler, error) {
	return &PlainHandler{
		options: opts,
	}, nil
}

// Initialize prepares the handler for use.
func (h *PlainHandler) Initialize() error {
	h.initialized = true
	return nil
}

// Output writes a message to the output.
func (h *PlainHandler) Output(message string) error {
	_, err := fmt.Fprintln(h.options.Output, message)
	return err
}

// OutputStructured writes structured data to the output.
// In plain mode, this uses a simple text representation.
func (h *PlainHandler) OutputStructured(data interface{}) error {
	// Convert to JSON for structured output
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.options.Output, string(jsonData))
	return err
}

// Error writes an error message.
func (h *PlainHandler) Error(err error) error {
	_, writeErr := fmt.Fprintln(h.options.ErrOutput, err.Error())
	return writeErr
}

// IsInteractive returns false.
func (h *PlainHandler) IsInteractive() bool {
	return false
}

// SupportsInput returns true (plain mode can read from stdin).
func (h *PlainHandler) SupportsInput() bool {
	return true
}

// ReadInput reads user input from stdin.
func (h *PlainHandler) ReadInput() (string, error) {
	var input string
	_, err := fmt.Fscanln(h.options.Input, &input)
	return input, err
}

// Cleanup performs cleanup operations.
func (h *PlainHandler) Cleanup() error {
	h.initialized = false
	return nil
}

// ==================== JSON HANDLER ====================

// JSONHandler handles structured JSON output mode.
type JSONHandler struct {
	options     JSONHandlerOptions
	initialized bool
	encoder     *json.Encoder
}

// JSONHandlerOptions configures the JSONHandler.
type JSONHandlerOptions struct {
	Output    io.Writer
	ErrOutput io.Writer
	Input     io.Reader
}

// NewJSONHandler creates a new JSON handler.
func NewJSONHandler(opts JSONHandlerOptions) (*JSONHandler, error) {
	return &JSONHandler{
		options: opts,
	}, nil
}

// Initialize prepares the handler for use.
func (h *JSONHandler) Initialize() error {
	h.encoder = json.NewEncoder(h.options.Output)
	h.encoder.SetIndent("", "  ")
	h.initialized = true
	return nil
}

// Output writes a message as a JSON object.
func (h *JSONHandler) Output(message string) error {
	return h.encoder.Encode(map[string]interface{}{
		"type":    "message",
		"content": message,
	})
}

// OutputStructured writes structured data as JSON.
func (h *JSONHandler) OutputStructured(data interface{}) error {
	return h.encoder.Encode(data)
}

// Error writes an error as JSON.
func (h *JSONHandler) Error(err error) error {
	errorObj := map[string]interface{}{
		"type":  "error",
		"error": err.Error(),
	}
	// Write error JSON to error output
	encoder := json.NewEncoder(h.options.ErrOutput)
	encoder.SetIndent("", "  ")
	return encoder.Encode(errorObj)
}

// IsInteractive returns false.
func (h *JSONHandler) IsInteractive() bool {
	return false
}

// SupportsInput returns true (JSON mode can read JSON from stdin).
func (h *JSONHandler) SupportsInput() bool {
	return true
}

// ReadInput reads JSON input from stdin.
func (h *JSONHandler) ReadInput() (string, error) {
	// Read raw input (JSON mode expects structured input)
	var input string
	_, err := fmt.Fscanln(h.options.Input, &input)
	return input, err
}

// Cleanup performs cleanup operations.
func (h *JSONHandler) Cleanup() error {
	h.initialized = false
	h.encoder = nil
	return nil
}

// ==================== CONTEXT-AWARE MODE SWITCHER ====================

// ContextKey is the type for context keys used with the mode switcher.
type ContextKey string

const (
	// ModeSwitcherContextKey is the key for storing the mode switcher in context.
	ModeSwitcherContextKey ContextKey = "mode_switcher"
)

// WithModeSwitcher adds a mode switcher to the context.
func WithModeSwitcher(ctx context.Context, switcher *ModeSwitcher) context.Context {
	return context.WithValue(ctx, ModeSwitcherContextKey, switcher)
}

// GetModeSwitcher retrieves the mode switcher from the context.
// Returns nil if no switcher is found.
func GetModeSwitcher(ctx context.Context) *ModeSwitcher {
	if v := ctx.Value(ModeSwitcherContextKey); v != nil {
		if switcher, ok := v.(*ModeSwitcher); ok {
			return switcher
		}
	}
	return nil
}

// MustGetModeSwitcher retrieves the mode switcher from the context.
// Panics if no switcher is found.
func MustGetModeSwitcher(ctx context.Context) *ModeSwitcher {
	switcher := GetModeSwitcher(ctx)
	if switcher == nil {
		panic("mode switcher not found in context")
	}
	return switcher
}
