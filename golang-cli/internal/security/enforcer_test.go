// Package security provides command permission control for the Cline CLI.
package security

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cline/cline/golang-cli/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCommandPermissionController is a mock controller for testing.
type MockCommandPermissionController struct {
	mock.Mock
}

func (m *MockCommandPermissionController) ValidateCommand(command string) PermissionValidationResult {
	args := m.Called(command)
	return args.Get(0).(PermissionValidationResult)
}

func (m *MockCommandPermissionController) FormatErrorMessage(result PermissionValidationResult, command string) string {
	args := m.Called(result, command)
	return args.String(0)
}

func (m *MockCommandPermissionController) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockCommandPermissionController) GetConfig() *PermissionRules {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*PermissionRules)
}

// mockExecutor creates a mock executor for testing.
func mockExecutor(output string, exitCode int, err error) CommandExecutor {
	return func(ctx context.Context, command string, args []string) (string, int, error) {
		return output, exitCode, err
	}
}

func TestNewPermissionEnforcer(t *testing.T) {
	enforcer := NewPermissionEnforcer()
	assert.NotNil(t, enforcer)
	assert.NotNil(t, enforcer.controller)
	assert.NotNil(t, enforcer.executor)
}

func TestNewPermissionEnforcerWithController(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	enforcer := NewPermissionEnforcerWithController(mockController)
	assert.NotNil(t, enforcer)
	assert.Equal(t, mockController, enforcer.controller)
}

func TestNewPermissionEnforcerWithOptions(t *testing.T) {
	executor := mockExecutor("output", 0, nil)

	// Use default config which has proper LogDir set
	config := audit.DefaultConfig()
	config.Enabled = false
	config.SyncWrite = true
	auditor, err := audit.NewLogger(config)
	require.NoError(t, err)
	defer auditor.Close()

	enforcer := NewPermissionEnforcer(
		WithCommandExecutor(executor),
		WithAuditor(auditor),
	)

	assert.NotNil(t, enforcer)
	assert.NotNil(t, enforcer.executor)
	assert.NotNil(t, enforcer.auditor)
}

func TestPermissionEnforcer_Execute_Allowed(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "ls -la").Return(PermissionValidationResult{
		Allowed: true,
		Reason:  "allowed",
	}, nil)

	executorCalled := false
	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		executorCalled = true
		assert.Equal(t, "ls", command)
		assert.Equal(t, []string{"-la"}, args)
		return "file1 file2", 0, nil
	}

	enforcer := NewPermissionEnforcerWithController(
		mockController,
		WithCommandExecutor(executor),
	)

	ctx := context.Background()
	result := enforcer.Execute(ctx, "ls", []string{"-la"})

	assert.True(t, executorCalled)
	assert.False(t, result.Blocked)
	assert.Equal(t, "file1 file2", result.Output)
	assert.Equal(t, 0, result.ExitCode)
	mockController.AssertExpectations(t)
}

func TestPermissionEnforcer_Execute_Blocked(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "rm -rf /").Return(PermissionValidationResult{
		Allowed: false,
		Reason:  "denied",
	}, nil)
	mockController.On("FormatErrorMessage", mock.Anything, "rm -rf /").Return("Command blocked: rm -rf /")

	executorCalled := false
	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		executorCalled = true
		return "", 0, nil
	}

	enforcer := NewPermissionEnforcerWithController(
		mockController,
		WithCommandExecutor(executor),
	)

	ctx := context.Background()
	result := enforcer.Execute(ctx, "rm", []string{"-rf", "/"})

	assert.False(t, executorCalled)
	assert.True(t, result.Blocked)
	assert.Equal(t, 1, result.ExitCode)
	assert.NotNil(t, result.Error)
	assert.Equal(t, "Command blocked: rm -rf /", result.BlockReason)
	mockController.AssertExpectations(t)
}

func TestPermissionEnforcer_Execute_NoArgs(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "pwd").Return(PermissionValidationResult{
		Allowed: true,
		Reason:  "allowed",
	}, nil)

	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		return "/home/user", 0, nil
	}

	enforcer := NewPermissionEnforcerWithController(
		mockController,
		WithCommandExecutor(executor),
	)

	ctx := context.Background()
	result := enforcer.Execute(ctx, "pwd", nil)

	assert.False(t, result.Blocked)
	assert.Equal(t, "/home/user", result.Output)
	mockController.AssertExpectations(t)
}

func TestPermissionEnforcer_ValidateOnly(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "git status").Return(PermissionValidationResult{
		Allowed: true,
		Reason:  "allowed",
	}, nil)

	enforcer := NewPermissionEnforcerWithController(mockController)
	result := enforcer.ValidateOnly("git", []string{"status"})

	assert.True(t, result.Allowed)
	assert.Equal(t, "allowed", result.Reason)
	mockController.AssertExpectations(t)
}

func TestPermissionEnforcer_IsEnabled(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("IsEnabled").Return(true)

	enforcer := NewPermissionEnforcerWithController(mockController)
	assert.True(t, enforcer.IsEnabled())
	mockController.AssertExpectations(t)
}

func TestPermissionEnforcer_GetConfig(t *testing.T) {
	config := &PermissionRules{
		Allow: []string{"git *"},
		Deny:  []string{"rm -rf /"},
	}

	mockController := new(MockCommandPermissionController)
	mockController.On("GetConfig").Return(config)

	enforcer := NewPermissionEnforcerWithController(mockController)
	result := enforcer.GetConfig()

	assert.Equal(t, config, result)
	mockController.AssertExpectations(t)
}

func TestBuildCommandString(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		expected string
	}{
		{
			name:     "no args",
			command:  "pwd",
			args:     nil,
			expected: "pwd",
		},
		{
			name:     "single arg",
			command:  "ls",
			args:     []string{"-la"},
			expected: "ls -la",
		},
		{
			name:     "multiple args",
			command:  "git",
			args:     []string{"commit", "-m", "message"},
			expected: "git commit -m message",
		},
		{
			name:     "empty args",
			command:  "clear",
			args:     []string{},
			expected: "clear",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommandString(tt.command, tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test WithUserContext
	ctx = WithUserContext(ctx, "testuser")
	assert.Equal(t, "testuser", getUserContextFromCtx(ctx))

	// Test WithTaskID
	ctx = WithTaskID(ctx, "task-123")
	assert.Equal(t, "task-123", getTaskIDFromCtx(ctx))

	// Test WithSessionID
	ctx = WithSessionID(ctx, "session-456")
	assert.Equal(t, "session-456", getSessionIDFromCtx(ctx))
}

func TestGetUserContextFromCtx_Fallback(t *testing.T) {
	ctx := context.Background()
	user := getUserContextFromCtx(ctx)
	assert.NotEmpty(t, user)
}

func TestExecutionResult(t *testing.T) {
	result := ExecutionResult{
		Output:      "test output",
		ExitCode:    0,
		Error:       nil,
		Blocked:     false,
		BlockReason: "",
	}

	assert.Equal(t, "test output", result.Output)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, result.Blocked)

	result = ExecutionResult{
		Output:      "",
		ExitCode:    1,
		Error:       fmt.Errorf("blocked"),
		Blocked:     true,
		BlockReason: "command not allowed",
	}

	assert.True(t, result.Blocked)
	assert.Equal(t, "command not allowed", result.BlockReason)
}

func TestDefaultExecutor(t *testing.T) {
	result, exitCode, err := defaultExecutor(context.Background(), "test", []string{"arg"})
	assert.Empty(t, result)
	assert.Equal(t, 0, exitCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no executor configured")
}

func TestPermissionEnforcer_WithAuditor(t *testing.T) {
	// Create a temporary directory for audit logs
	config := &PermissionRules{
		Allow: []string{"echo*"},
	}
	controller := NewCommandPermissionControllerWithConfig(config)

	// Create auditor
	auditConfig := audit.DefaultConfig()
	auditConfig.Enabled = true
	auditConfig.SyncWrite = true
	auditor, err := audit.NewLogger(auditConfig)
	require.NoError(t, err)
	defer auditor.Close()

	// Create executor
	executor := mockExecutor("hello", 0, nil)

	// Create enforcer with controller and auditor
	pe := NewPermissionEnforcerWithController(controller,
		WithCommandExecutor(executor),
		WithAuditor(auditor),
	)

	// Execute a command - this should trigger audit logging
	result := pe.Execute(context.Background(), "echo", []string{"hello"})
	assert.False(t, result.Blocked)
	assert.Equal(t, "hello", result.Output)
}

func TestPermissionEnforcer_BlockedWithAuditor(t *testing.T) {
	// Create a temporary directory for audit logs
	config := &PermissionRules{
		Allow: []string{"echo*"},
		Deny:  []string{"rm*"},
	}
	controller := NewCommandPermissionControllerWithConfig(config)

	// Create auditor
	auditConfig := audit.DefaultConfig()
	auditConfig.Enabled = true
	auditConfig.SyncWrite = true
	auditor, err := audit.NewLogger(auditConfig)
	require.NoError(t, err)
	defer auditor.Close()

	// Create executor (should not be called)
	executor := mockExecutor("", 0, nil)

	// Create enforcer with controller and auditor
	pe := NewPermissionEnforcerWithController(controller,
		WithCommandExecutor(executor),
		WithAuditor(auditor),
	)

	// Execute a blocked command - this should trigger audit logging
	result := pe.Execute(context.Background(), "rm", []string{"-rf", "/"})
	assert.True(t, result.Blocked)
	assert.NotEmpty(t, result.BlockReason)
}

func TestPermissionEnforcer_ExecuteWithNilAuditor(t *testing.T) {
	config := &PermissionRules{
		Allow: []string{"echo*"},
	}
	controller := NewCommandPermissionControllerWithConfig(config)

	executor := mockExecutor("hello", 0, nil)

	// Create enforcer without auditor using the controller
	pe := NewPermissionEnforcerWithController(controller,
		WithCommandExecutor(executor),
	)

	// Execute a command - no auditor, no logging
	result := pe.Execute(context.Background(), "echo", []string{"hello"})
	assert.False(t, result.Blocked)
	assert.Equal(t, "hello", result.Output)
}

func TestValidateFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid file path within working directory", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test.txt")
		err := ValidateFilePath(filePath, tmpDir)
		assert.NoError(t, err)
	})

	t.Run("invalid file path outside working directory", func(t *testing.T) {
		err := ValidateFilePath("/etc/passwd", tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside of working directory")
	})

	t.Run("parent directory traversal blocked", func(t *testing.T) {
		err := ValidateFilePath("../outside.txt", tmpDir)
		assert.Error(t, err)
	})

	t.Run("nested parent directory traversal blocked", func(t *testing.T) {
		err := ValidateFilePath("../../etc/passwd", tmpDir)
		assert.Error(t, err)
	})

	t.Run("invalid file path with null byte", func(t *testing.T) {
		// Create a file path that will cause filepath.Abs to fail
		err := ValidateFilePath(string([]byte{0}), tmpDir)
		assert.Error(t, err)
	})

	t.Run("file path equals working directory", func(t *testing.T) {
		err := ValidateFilePath(tmpDir, tmpDir)
		assert.NoError(t, err)
	})

	t.Run("nested file path within working directory", func(t *testing.T) {
		nestedPath := filepath.Join(tmpDir, "subdir", "deep", "file.txt")
		err := ValidateFilePath(nestedPath, tmpDir)
		assert.NoError(t, err)
	})
}

func TestPermissionEnforcer_ExecuteWithError(t *testing.T) {
	config := &PermissionRules{
		Allow: []string{"*"},
	}
	controller := NewCommandPermissionControllerWithConfig(config)

	// Create an executor that returns an error
	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		return "", 1, fmt.Errorf("command failed")
	}

	pe := NewPermissionEnforcerWithController(controller,
		WithCommandExecutor(executor),
	)

	result := pe.Execute(context.Background(), "failing", []string{"command"})
	assert.False(t, result.Blocked)
	assert.Equal(t, 1, result.ExitCode)
	assert.Error(t, result.Error)
}
