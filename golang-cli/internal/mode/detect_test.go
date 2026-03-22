package mode

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFile is a mock implementation of a file with a file descriptor
type mockFile struct {
	io.Reader
	fd         uintptr
	isTerminal bool
}

func (m *mockFile) Fd() uintptr {
	return m.fd
}

// mockWriter is a mock implementation of a file writer with a file descriptor
type mockWriter struct {
	io.Writer
	fd         uintptr
	isTerminal bool
}

func (m *mockWriter) Fd() uintptr {
	return m.fd
}

// mockReadWriter implements both io.Reader and io.Writer with Fd()
type mockReadWriter struct {
	*bytes.Buffer
	fd         uintptr
	isTerminal bool
}

func (m *mockReadWriter) Fd() uintptr {
	return m.fd
}

// createMockTerminalChecker creates a terminal checker that respects the isTerminal field
// of mock files based on their file descriptor.
func createMockTerminalChecker(stdin, stdout, stderr *mockFile) terminalChecker {
	return func(fd int) bool {
		switch fd {
		case 0:
			return stdin.isTerminal
		case 1:
			return stdout.isTerminal
		case 2:
			return stderr.isTerminal
		default:
			return false
		}
	}
}

// createMockTerminalCheckerFromWriters creates a terminal checker for mock writers.
func createMockTerminalCheckerFromWriters(stdin *mockFile, stdout, stderr *mockWriter) terminalChecker {
	return func(fd int) bool {
		switch fd {
		case 0:
			return stdin.isTerminal
		case 1:
			return stdout.isTerminal
		case 2:
			return stderr.isTerminal
		default:
			return false
		}
	}
}

func TestNewDetector(t *testing.T) {
	t.Run("creates detector with defaults", func(t *testing.T) {
		d := NewDetector()
		require.NotNil(t, d)
		assert.Equal(t, os.Stdin, d.stdin)
		assert.Equal(t, os.Stdout, d.stdout)
		assert.Equal(t, os.Stderr, d.stderr)
		assert.False(t, d.forceTTY)
	})

	t.Run("applies options", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2}

		d := NewDetector(
			WithForceTTY(true),
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
		)

		require.NotNil(t, d)
		assert.True(t, d.forceTTY)
		assert.Equal(t, stdin, d.stdin)
		assert.Equal(t, stdout, d.stdout)
		assert.Equal(t, stderr, d.stderr)
	})

	t.Run("reads CLINE_FORCE_TTY from environment", func(t *testing.T) {
		// Set environment variable
		originalValue := os.Getenv("CLINE_FORCE_TTY")
		os.Setenv("CLINE_FORCE_TTY", "1")
		defer os.Setenv("CLINE_FORCE_TTY", originalValue)

		d := NewDetector()
		assert.True(t, d.forceTTY)
	})

	t.Run("empty CLINE_FORCE_TTY does not force TTY", func(t *testing.T) {
		originalValue := os.Getenv("CLINE_FORCE_TTY")
		os.Setenv("CLINE_FORCE_TTY", "")
		defer os.Setenv("CLINE_FORCE_TTY", originalValue)

		d := NewDetector()
		assert.False(t, d.forceTTY)
	})

	t.Run("WithForceTTY overrides environment", func(t *testing.T) {
		originalValue := os.Getenv("CLINE_FORCE_TTY")
		os.Setenv("CLINE_FORCE_TTY", "1")
		defer os.Setenv("CLINE_FORCE_TTY", originalValue)

		d := NewDetector(WithForceTTY(false))
		assert.False(t, d.forceTTY)
	})
}

func TestDetector_Detect(t *testing.T) {
	t.Run("forced TTY mode takes precedence", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithForceTTY(true),
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		mode := d.Detect()
		assert.Equal(t, TerminalModeForcedTTY, mode)
	})

	t.Run("caches mode result", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: true}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: true}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: true}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		mode1 := d.Detect()
		assert.NotNil(t, d.cachedMode)
		mode2 := d.Detect()
		assert.Equal(t, mode1, mode2)
	})

	t.Run("reset clears cache", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: true}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: true}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: true}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		d.Detect()
		assert.NotNil(t, d.cachedMode)

		d.Reset()
		assert.Nil(t, d.cachedMode)
	})
}

func TestDetector_IsInteractive(t *testing.T) {
	tests := []struct {
		name        string
		stdinTTY    bool
		stdoutTTY   bool
		forceTTY    bool
		interactive bool
	}{
		{
			name:        "fully interactive terminal",
			stdinTTY:    true,
			stdoutTTY:   true,
			forceTTY:    false,
			interactive: true,
		},
		{
			name:        "forced TTY is interactive",
			stdinTTY:    false,
			stdoutTTY:   false,
			forceTTY:    true,
			interactive: true,
		},
		{
			name:        "piped input not interactive",
			stdinTTY:    false,
			stdoutTTY:   true,
			forceTTY:    false,
			interactive: false,
		},
		{
			name:        "piped output not interactive",
			stdinTTY:    true,
			stdoutTTY:   false,
			forceTTY:    false,
			interactive: false,
		},
		{
			name:        "both piped not interactive",
			stdinTTY:    false,
			stdoutTTY:   false,
			forceTTY:    false,
			interactive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: tt.stdinTTY}
			stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: tt.stdoutTTY}
			stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

			checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

			d := NewDetector(
				WithForceTTY(tt.forceTTY),
				WithStdin(stdin),
				WithStdout(stdout),
				WithStderr(stderr),
				withTerminalChecker(checker),
			)

			assert.Equal(t, tt.interactive, d.IsInteractive())
		})
	}
}

func TestDetector_IsPipedInput(t *testing.T) {
	tests := []struct {
		name       string
		stdinTTY   bool
		stdoutTTY  bool
		pipedInput bool
	}{
		{
			name:       "piped input detected",
			stdinTTY:   false,
			stdoutTTY:  true,
			pipedInput: true,
		},
		{
			name:       "both piped includes input",
			stdinTTY:   false,
			stdoutTTY:  false,
			pipedInput: true,
		},
		{
			name:       "interactive no piped input",
			stdinTTY:   true,
			stdoutTTY:  true,
			pipedInput: false,
		},
		{
			name:       "piped output only",
			stdinTTY:   true,
			stdoutTTY:  false,
			pipedInput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: tt.stdinTTY}
			stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: tt.stdoutTTY}
			stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

			checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

			d := NewDetector(
				WithStdin(stdin),
				WithStdout(stdout),
				WithStderr(stderr),
				withTerminalChecker(checker),
			)

			assert.Equal(t, tt.pipedInput, d.IsPipedInput())
		})
	}
}

func TestDetector_IsPipedOutput(t *testing.T) {
	tests := []struct {
		name        string
		stdinTTY    bool
		stdoutTTY   bool
		pipedOutput bool
	}{
		{
			name:        "piped output detected",
			stdinTTY:    true,
			stdoutTTY:   false,
			pipedOutput: true,
		},
		{
			name:        "both piped includes output",
			stdinTTY:    false,
			stdoutTTY:   false,
			pipedOutput: true,
		},
		{
			name:        "interactive no piped output",
			stdinTTY:    true,
			stdoutTTY:   true,
			pipedOutput: false,
		},
		{
			name:        "piped input only",
			stdinTTY:    false,
			stdoutTTY:   true,
			pipedOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: tt.stdinTTY}
			stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: tt.stdoutTTY}
			stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

			checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

			d := NewDetector(
				WithStdin(stdin),
				WithStdout(stdout),
				WithStderr(stderr),
				withTerminalChecker(checker),
			)

			assert.Equal(t, tt.pipedOutput, d.IsPipedOutput())
		})
	}
}

func TestDetector_IsForcedTTY(t *testing.T) {
	t.Run("detects forced TTY", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithForceTTY(true),
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.True(t, d.IsForcedTTY())
	})

	t.Run("detects non-forced TTY", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: true}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: true}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: true}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithForceTTY(false),
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.False(t, d.IsForcedTTY())
	})
}

func TestDetector_TerminalChecks(t *testing.T) {
	t.Run("IsStdinTerminal with Fd() interface", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: true}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.True(t, d.IsStdinTerminal())
	})

	t.Run("IsStdinTerminal without Fd() interface", func(t *testing.T) {
		d := NewDetector(WithStdin(&bytes.Buffer{}))
		assert.False(t, d.IsStdinTerminal())
	})

	t.Run("IsStdoutTerminal with Fd() interface", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: true}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.True(t, d.IsStdoutTerminal())
	})

	t.Run("IsStdoutTerminal without Fd() interface", func(t *testing.T) {
		d := NewDetector(WithStdout(&bytes.Buffer{}))
		assert.False(t, d.IsStdoutTerminal())
	})

	t.Run("IsStderrTerminal with Fd() interface", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: true}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.True(t, d.IsStderrTerminal())
	})

	t.Run("IsStderrTerminal without Fd() interface", func(t *testing.T) {
		d := NewDetector(WithStderr(&bytes.Buffer{}))
		assert.False(t, d.IsStderrTerminal())
	})
}

func TestTerminalMode_String(t *testing.T) {
	tests := []struct {
		mode     TerminalMode
		expected string
	}{
		{TerminalModeInteractive, "interactive"},
		{TerminalModePipedInput, "piped_input"},
		{TerminalModePipedOutput, "piped_output"},
		{TerminalModePipedBoth, "piped_both"},
		{TerminalModeForcedTTY, "forced_tty"},
		{TerminalMode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.String())
		})
	}
}

func TestTerminalMode_IsTerminal(t *testing.T) {
	tests := []struct {
		name       string
		mode       TerminalMode
		isTerminal bool
	}{
		{"interactive", TerminalModeInteractive, true},
		{"forced TTY", TerminalModeForcedTTY, true},
		{"piped input", TerminalModePipedInput, false},
		{"piped output", TerminalModePipedOutput, false},
		{"piped both", TerminalModePipedBoth, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isTerminal, tt.mode.IsTerminal())
		})
	}
}

func TestDetector_ModeCombinations(t *testing.T) {
	// Test all combinations of stdin/stdout TTY states
	combinations := []struct {
		stdinTTY  bool
		stdoutTTY bool
		expected  TerminalMode
	}{
		{true, true, TerminalModeInteractive},
		{false, true, TerminalModePipedInput},
		{true, false, TerminalModePipedOutput},
		{false, false, TerminalModePipedBoth},
	}

	for _, tc := range combinations {
		name := "stdin_"
		if tc.stdinTTY {
			name += "tty_"
		} else {
			name += "pipe_"
		}
		name += "stdout_"
		if tc.stdoutTTY {
			name += "tty"
		} else {
			name += "pipe"
		}

		t.Run(name, func(t *testing.T) {
			stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: tc.stdinTTY}
			stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: tc.stdoutTTY}
			stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}

			checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

			d := NewDetector(
				WithStdin(stdin),
				WithStdout(stdout),
				WithStderr(stderr),
				withTerminalChecker(checker),
			)

			mode := d.Detect()
			assert.Equal(t, tc.expected, mode)
		})
	}
}

func TestDetector_WithStderr(t *testing.T) {
	t.Run("sets stderr writer", func(t *testing.T) {
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: false}
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}

		d := NewDetector(
			WithStderr(stderr),
			WithStdin(stdin),
			WithStdout(stdout),
		)

		assert.Equal(t, stderr, d.stderr)
	})

	t.Run("detects stderr terminal", func(t *testing.T) {
		stdin := &mockFile{Reader: bytes.NewReader([]byte("test")), fd: 0, isTerminal: false}
		stdout := &mockWriter{Writer: &bytes.Buffer{}, fd: 1, isTerminal: false}
		stderr := &mockWriter{Writer: &bytes.Buffer{}, fd: 2, isTerminal: true}

		checker := createMockTerminalCheckerFromWriters(stdin, stdout, stderr)

		d := NewDetector(
			WithStdin(stdin),
			WithStdout(stdout),
			WithStderr(stderr),
			withTerminalChecker(checker),
		)

		assert.True(t, d.IsStderrTerminal())
	})
}

func TestDetector_RealStdio(t *testing.T) {
	// This test uses real os.Stdin/os.Stdout but won't be a terminal in test environment
	t.Run("handles real stdio", func(t *testing.T) {
		d := NewDetector()

		// Should not panic
		mode := d.Detect()
		// In CI/test environments, this will typically be TerminalModePipedBoth or TerminalModeInteractive
		// depending on how tests are run
		assert.True(t, mode >= TerminalModeInteractive && mode <= TerminalModeForcedTTY)

		// These should not panic
		_ = d.IsInteractive()
		_ = d.IsPipedInput()
		_ = d.IsPipedOutput()
		_ = d.IsStdinTerminal()
		_ = d.IsStdoutTerminal()
		_ = d.IsStderrTerminal()
	})
}