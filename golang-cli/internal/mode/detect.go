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

// ExtendedMode provides more detailed terminal mode detection
type ExtendedMode int

const (
	// ExtendedModeUnknown means mode could not be determined
	ExtendedModeUnknown ExtendedMode = iota
	// ExtendedModeInteractiveTTY means fully interactive TTY with color support
	ExtendedModeInteractiveTTY
	// ExtendedModeInteractiveNoColor means interactive but without color support
	ExtendedModeInteractiveNoColor
	// ExtendedModeCI means running in CI environment
	ExtendedModeCI
	// ExtendedModePipe means input or output is piped
	ExtendedModePipe
	// ExtendedModeFile means output is redirected to a file
	ExtendedModeFile
	// ExtendedModeDaemon means running as a background process
	ExtendedModeDaemon
)

// String returns the string representation of ExtendedMode
func (m ExtendedMode) String() string {
	switch m {
	case ExtendedModeInteractiveTTY:
		return "interactive_tty"
	case ExtendedModeInteractiveNoColor:
		return "interactive_no_color"
	case ExtendedModeCI:
		return "ci"
	case ExtendedModePipe:
		return "pipe"
	case ExtendedModeFile:
		return "file"
	case ExtendedModeDaemon:
		return "daemon"
	default:
		return "unknown"
	}
}

// IsInteractive returns true if the mode supports interactive input
func (m ExtendedMode) IsInteractive() bool {
	return m == ExtendedModeInteractiveTTY || m == ExtendedModeInteractiveNoColor
}

// SupportsColor returns true if the mode supports color output
func (m ExtendedMode) SupportsColor() bool {
	return m == ExtendedModeInteractiveTTY || m == ExtendedModeCI
}

// OutputMode represents the configured output mode
type OutputMode int

const (
	// OutputModeAuto automatically selects output mode based on terminal
	OutputModeAuto OutputMode = iota
	// OutputModePlain uses plain text output
	OutputModePlain
	// OutputModeJSON uses JSON streaming output
	OutputModeJSON
	// OutputModeQuiet suppresses non-essential output
	OutputModeQuiet
)

// String returns the string representation of OutputMode
func (m OutputMode) String() string {
	switch m {
	case OutputModePlain:
		return "plain"
	case OutputModeJSON:
		return "json"
	case OutputModeQuiet:
		return "quiet"
	case OutputModeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// IsInteractive returns true if the output mode is interactive
func (m OutputMode) IsInteractive() bool {
	return m == OutputModeAuto || m == OutputModePlain
}

// EnhancedDetector provides enhanced terminal mode detection
type EnhancedDetector struct {
	detector *Detector
}

// NewEnhancedDetector creates a new enhanced detector
func NewEnhancedDetector() *EnhancedDetector {
	return &EnhancedDetector{
		detector: NewDetector(),
	}
}

// DetectExtended detects the extended terminal mode
func (ed *EnhancedDetector) DetectExtended() ExtendedMode {
	// Check for CI environment first
	if ed.isCIEnvironment() {
		return ExtendedModeCI
	}

	// Check if running as daemon
	if ed.isDaemon() {
		return ExtendedModeDaemon
	}

	// Check for output redirection
	if ed.isOutputRedirectedToFile() {
		return ExtendedModeFile
	}

	// Check for pipes
	if ed.detector.IsPipedInput() || ed.detector.IsPipedOutput() {
		return ExtendedModePipe
	}

	// Check for interactive terminal
	if ed.detector.IsInteractive() {
		if ed.supportsColor() {
			return ExtendedModeInteractiveTTY
		}
		return ExtendedModeInteractiveNoColor
	}

	return ExtendedModeUnknown
}

// SelectOutputMode selects the appropriate output mode based on configuration
func (ed *EnhancedDetector) SelectOutputMode(jsonFlag, plainFlag bool) OutputMode {
	switch {
	case plainFlag:
		return OutputModePlain
	case jsonFlag:
		return OutputModeJSON
	case ed.DetectExtended() == ExtendedModeCI:
		return OutputModeJSON
	case !ed.detector.IsInteractive():
		return OutputModePlain
	default:
		return OutputModeAuto
	}
}

// isCIEnvironment checks if running in a CI environment
func (ed *EnhancedDetector) isCIEnvironment() bool {
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI",
		"TRAVIS", "JENKINS_URL", "BUILDKITE",
		"DRONE", "APPVEYOR", "AZURE_PIPELINES",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// isDaemon checks if running as a daemon/background process
func (ed *EnhancedDetector) isDaemon() bool {
	// Check for common daemon indicators
	if os.Getenv("DAEMON") != "" {
		return true
	}
	// On Unix systems, check if ppid is 1 (init)
	// This is a simplified check; real implementation would use syscalls
	return false
}

// isOutputRedirectedToFile checks if stdout is redirected to a file
func (ed *EnhancedDetector) isOutputRedirectedToFile() bool {
	// Check if stdout is a regular file
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// supportsColor checks if the terminal supports color output
func (ed *EnhancedDetector) supportsColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if terminal supports color
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Check for color-supporting terminals
	colorTerms := []string{"color", "256color", "truecolor", "xterm", "screen", "tmux"}
	for _, ct := range colorTerms {
		if contains(term, ct) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
