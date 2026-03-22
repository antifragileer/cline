// Package mode provides terminal mode detection capabilities for the Cline CLI.
// It handles TTY detection, pipe/redirect identification, and mode override
// mechanisms to support both interactive and non-interactive usage scenarios.
package mode

import (
	"io"
	"os"

	"golang.org/x/term"
)

// TerminalMode represents the detected terminal interaction mode.
type TerminalMode int

const (
	// TerminalModeInteractive indicates a fully interactive terminal session
	// with TTY attached to stdin, stdout, and stderr.
	TerminalModeInteractive TerminalMode = iota

	// TerminalModePipedInput indicates input is being piped from another command
	// or file rather than typed interactively.
	TerminalModePipedInput

	// TerminalModePipedOutput indicates output is being redirected to a file
	// or piped to another command rather than displayed on a terminal.
	TerminalModePipedOutput

	// TerminalModePipedBoth indicates both input and output are redirected/piped.
	TerminalModePipedBoth

	// TerminalModeForcedTTY indicates TTY mode was explicitly forced via
	// environment variable or flag, regardless of actual terminal state.
	TerminalModeForcedTTY
)

// terminalChecker is a function type for checking if a file descriptor is a terminal.
// This allows for dependency injection in tests.
type terminalChecker func(fd int) bool

// Detector handles terminal mode detection and provides information about
// the current input/output configuration.
type Detector struct {
	// forceTTY overrides automatic detection when true
	forceTTY bool

	// stdin is the input reader to check (typically os.Stdin)
	stdin io.Reader

	// stdout is the output writer to check (typically os.Stdout)
	stdout io.Writer

	// stderr is the error writer to check (typically os.Stderr)
	stderr io.Writer

	// cachedMode stores the computed mode to avoid redundant checks
	cachedMode *TerminalMode

	// isTerminalFn is the function used to check if a file descriptor is a terminal.
	// Defaults to term.IsTerminal but can be overridden for testing.
	isTerminalFn terminalChecker
}

// DetectorOption configures a Detector.
type DetectorOption func(*Detector)

// WithForceTTY forces TTY mode regardless of actual terminal state.
func WithForceTTY(force bool) DetectorOption {
	return func(d *Detector) {
		d.forceTTY = force
	}
}

// WithStdin sets the stdin reader for detection.
func WithStdin(r io.Reader) DetectorOption {
	return func(d *Detector) {
		d.stdin = r
	}
}

// WithStdout sets the stdout writer for detection.
func WithStdout(w io.Writer) DetectorOption {
	return func(d *Detector) {
		d.stdout = w
	}
}

// WithStderr sets the stderr writer for detection.
func WithStderr(w io.Writer) DetectorOption {
	return func(d *Detector) {
		d.stderr = w
	}
}

// withTerminalChecker sets the terminal checker function (for testing).
func withTerminalChecker(fn terminalChecker) DetectorOption {
	return func(d *Detector) {
		d.isTerminalFn = fn
	}
}

// NewDetector creates a new Detector with the provided options.
// By default, it detects modes for os.Stdin, os.Stdout, and os.Stderr.
func NewDetector(opts ...DetectorOption) *Detector {
	d := &Detector{
		stdin:        os.Stdin,
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		isTerminalFn: term.IsTerminal,
	}

	// Check environment variable for force TTY
	if os.Getenv("CLINE_FORCE_TTY") != "" {
		d.forceTTY = true
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Detect determines the current terminal mode by checking if stdin, stdout,
// and stderr are connected to a terminal. It caches the result for subsequent
// calls.
func (d *Detector) Detect() TerminalMode {
	if d.cachedMode != nil {
		return *d.cachedMode
	}

	// Check for forced TTY mode first
	if d.forceTTY {
		mode := TerminalModeForcedTTY
		d.cachedMode = &mode
		return mode
	}

	// Check if stdin is a terminal
	stdinTTY := d.isStdinTerminal()

	// Check if stdout is a terminal
	stdoutTTY := d.isStdoutTerminal()

	// Determine mode based on terminal attachment
	var mode TerminalMode
	switch {
	case !stdinTTY && !stdoutTTY:
		mode = TerminalModePipedBoth
	case !stdinTTY:
		mode = TerminalModePipedInput
	case !stdoutTTY:
		mode = TerminalModePipedOutput
	default:
		mode = TerminalModeInteractive
	}

	d.cachedMode = &mode
	return mode
}

// IsInteractive returns true if the terminal is in interactive mode
// (both stdin and stdout are connected to a TTY).
func (d *Detector) IsInteractive() bool {
	mode := d.Detect()
	return mode == TerminalModeInteractive || mode == TerminalModeForcedTTY
}

// IsPipedInput returns true if input is being piped or redirected.
func (d *Detector) IsPipedInput() bool {
	mode := d.Detect()
	return mode == TerminalModePipedInput || mode == TerminalModePipedBoth
}

// IsPipedOutput returns true if output is being piped or redirected.
func (d *Detector) IsPipedOutput() bool {
	mode := d.Detect()
	return mode == TerminalModePipedOutput || mode == TerminalModePipedBoth
}

// IsForcedTTY returns true if TTY mode was forced via environment variable
// or explicit flag.
func (d *Detector) IsForcedTTY() bool {
	return d.Detect() == TerminalModeForcedTTY
}

// IsStdinTerminal returns true if stdin is connected to a terminal.
func (d *Detector) IsStdinTerminal() bool {
	return d.isStdinTerminal()
}

// IsStdoutTerminal returns true if stdout is connected to a terminal.
func (d *Detector) IsStdoutTerminal() bool {
	return d.isStdoutTerminal()
}

// IsStderrTerminal returns true if stderr is connected to a terminal.
func (d *Detector) IsStderrTerminal() bool {
	return d.isStderrTerminal()
}

// isStdinTerminal checks if stdin is a terminal.
func (d *Detector) isStdinTerminal() bool {
	f, ok := d.stdin.(interface {
		Fd() uintptr
	})
	if !ok {
		return false
	}
	return d.isTerminalFn(int(f.Fd()))
}

// isStdoutTerminal checks if stdout is a terminal.
func (d *Detector) isStdoutTerminal() bool {
	f, ok := d.stdout.(interface {
		Fd() uintptr
	})
	if !ok {
		return false
	}
	return d.isTerminalFn(int(f.Fd()))
}

// isStderrTerminal checks if stderr is a terminal.
func (d *Detector) isStderrTerminal() bool {
	f, ok := d.stderr.(interface {
		Fd() uintptr
	})
	if !ok {
		return false
	}
	return d.isTerminalFn(int(f.Fd()))
}

// Reset clears the cached mode, forcing re-detection on next call.
// This is useful when the terminal state may have changed.
func (d *Detector) Reset() {
	d.cachedMode = nil
}

// String returns a human-readable description of the terminal mode.
func (m TerminalMode) String() string {
	switch m {
	case TerminalModeInteractive:
		return "interactive"
	case TerminalModePipedInput:
		return "piped_input"
	case TerminalModePipedOutput:
		return "piped_output"
	case TerminalModePipedBoth:
		return "piped_both"
	case TerminalModeForcedTTY:
		return "forced_tty"
	default:
		return "unknown"
	}
}

// IsTerminal returns true if the mode represents an interactive terminal session.
// This includes both naturally interactive mode and forced TTY mode.
func (m TerminalMode) IsTerminal() bool {
	return m == TerminalModeInteractive || m == TerminalModeForcedTTY
}
