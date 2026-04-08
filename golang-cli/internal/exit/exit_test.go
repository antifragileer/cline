package exit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler()
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}

	if h.GetExitCode() != Success {
		t.Errorf("Expected initial exit code %d, got %d", Success, h.GetExitCode())
	}
}

func TestSetExitCode(t *testing.T) {
	h := NewHandler()

	tests := []Code{
		Success,
		GeneralError,
		InvalidArguments,
		ConnectionError,
		TaskFailed,
	}

	for _, code := range tests {
		h.SetExitCode(code)
		if h.GetExitCode() != code {
			t.Errorf("SetExitCode(%d) failed, got %d", code, h.GetExitCode())
		}
	}
}

func TestMapErrorToCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Code
	}{
		{"nil error", nil, Success},
		{"general error", errors.New("something went wrong"), GeneralError},
		{"permission denied", errors.New("permission denied"), PermissionDenied},
		{"connection refused", errors.New("connection refused"), ConnectionError},
		{"connection reset", errors.New("connection reset by peer"), ConnectionError},
		{"timeout", errors.New("operation timeout"), Timeout},
		{"deadline exceeded", errors.New("context deadline exceeded"), Timeout},
		{"config error", errors.New("configuration invalid"), ConfigurationError},
		{"task failed", errors.New("task execution failed"), TaskFailed},
		{"command not found", errors.New("command not found"), CommandNotFound},
		{"executable not found", errors.New("executable file not found in PATH"), CommandNotFound},
		{"invalid argument", errors.New("invalid argument provided"), InvalidArguments},
		{"flag not defined", errors.New("flag provided but not defined"), InvalidArguments},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapErrorToCode(tt.err)
			if got != tt.expected {
				t.Errorf("MapErrorToCode(%q) = %d, want %d", tt.err, got, tt.expected)
			}
		})
	}
}

func TestCodeString(t *testing.T) {
	tests := []struct {
		code     Code
		expected string
	}{
		{Success, "success"},
		{GeneralError, "general error"},
		{InvalidArguments, "invalid arguments"},
		{CommandNotFound, "command not found"},
		{Interrupted, "interrupted"},
		{Timeout, "timeout"},
		{PermissionDenied, "permission denied"},
		{ConfigurationError, "configuration error"},
		{ConnectionError, "connection error"},
		{TaskFailed, "task failed"},
		{Code(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.code.String()
			if got != tt.expected {
				t.Errorf("Code(%d).String() = %q, want %q", tt.code, got, tt.expected)
			}
		})
	}
}

func TestIsSuccess(t *testing.T) {
	if !IsSuccess(Success) {
		t.Error("IsSuccess(Success) should return true")
	}

	if IsSuccess(GeneralError) {
		t.Error("IsSuccess(GeneralError) should return false")
	}
}

func TestIsError(t *testing.T) {
	if IsError(Success) {
		t.Error("IsError(Success) should return false")
	}

	if !IsError(GeneralError) {
		t.Error("IsError(GeneralError) should return true")
	}
}

func TestIsSignal(t *testing.T) {
	tests := []struct {
		code     Code
		expected bool
	}{
		{Success, false},
		{GeneralError, false},
		{Code(128), true},
		{Code(130), true}, // SIGINT
		{Code(143), true}, // SIGTERM
		{Code(165), true},
		{Code(166), false},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := IsSignal(tt.code)
			if got != tt.expected {
				t.Errorf("IsSignal(%d) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestHandlerRunSuccess(t *testing.T) {
	h := NewHandler()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.Run(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if h.GetExitCode() != Success {
		t.Errorf("Expected exit code %d, got %d", Success, h.GetExitCode())
	}
}

func TestHandlerRunError(t *testing.T) {
	h := NewHandler()

	testErr := errors.New("test error")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := h.Run(ctx, func(ctx context.Context) error {
		return testErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// The error should cause a non-success exit code
	if h.GetExitCode() == Success {
		t.Error("Expected non-success exit code after error")
	}
}

func TestHandlerRegisterCleanup(t *testing.T) {
	h := NewHandler()

	cleanupCalled := false
	h.RegisterCleanup(func() {
		cleanupCalled = true
	})

	// Note: We can't actually test the cleanup is called without calling Exit()
	// which would terminate the test process. The registration itself is tested
	// by the fact that no panic occurs.
	_ = cleanupCalled
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "WORLD", true}, // case insensitive
		{"hello world", "foo", false},
		{"", "", true},
		{"hello", "", true},
		{"hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    byte
		expected byte
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'a', 'a'},
		{'z', 'z'},
		{'0', '0'},
		{'@', '@'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := toLower(tt.input)
			if got != tt.expected {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContextWithHandler(t *testing.T) {
	h := NewHandler()
	ctx := ContextWithHandler(context.Background(), h)

	retrieved := HandlerFromContext(ctx)
	if retrieved != h {
		t.Error("HandlerFromContext did not return the expected handler")
	}

	// Test with context that has no handler
	emptyCtx := context.Background()
	if HandlerFromContext(emptyCtx) != nil {
		t.Error("HandlerFromContext should return nil for context without handler")
	}
}
