package security

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIsPermissionControlEnabled(t *testing.T) {
	// Should return false by default (no env set)
	enabled := IsPermissionControlEnabled()
	// Result depends on environment, just ensure it doesn't panic
	_ = enabled
}

func TestContainsDangerousCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"safe command", false},
		{"rm -rf /", false},
		// Note: The Detector only checks for command substitution patterns
		{"`whoami`", true},
		{"$(whoami)", true},
		{"echo hello", false},
		{"normal text", false},
		{"cmd `subshell`", true},
		{"cmd $(subshell)", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ContainsDangerousCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecuteWithValidation_Allowed(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "ls -la").Return(PermissionValidationResult{
		Allowed: true,
		Reason:  "allowed",
	}, nil)

	executorCalled := false
	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		executorCalled = true
		return "output", 0, nil
	}

	enforcer := NewPermissionEnforcerWithController(
		mockController,
		WithCommandExecutor(executor),
	)

	result, validation := enforcer.ExecuteWithValidation(context.Background(), "ls", []string{"-la"})

	assert.True(t, executorCalled)
	assert.False(t, result.Blocked)
	assert.True(t, validation.Allowed)
	mockController.AssertExpectations(t)
}

func TestExecuteWithValidation_Blocked(t *testing.T) {
	mockController := new(MockCommandPermissionController)
	mockController.On("ValidateCommand", "rm -rf /").Return(PermissionValidationResult{
		Allowed: false,
		Reason:  "denied",
	}, nil)
	mockController.On("FormatErrorMessage", mock.Anything, "rm -rf /").Return("Command blocked")

	executorCalled := false
	executor := func(ctx context.Context, command string, args []string) (string, int, error) {
		executorCalled = true
		return "", 0, nil
	}

	enforcer := NewPermissionEnforcerWithController(
		mockController,
		WithCommandExecutor(executor),
	)

	result, validation := enforcer.ExecuteWithValidation(context.Background(), "rm", []string{"-rf", "/"})

	assert.False(t, executorCalled)
	assert.True(t, result.Blocked)
	assert.False(t, validation.Allowed)
	mockController.AssertExpectations(t)
}

func TestFormatErrorMessage_DifferentTypes(t *testing.T) {
	controller := NewCommandPermissionController()

	tests := []struct {
		name   string
		result PermissionValidationResult
	}{
		{
			name: "shell operator",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "shell operators not allowed",
			},
		},
		{
			name: "redirect",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "redirects not allowed",
			},
		},
		{
			name: "denied pattern",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "denied by pattern",
			},
		},
		{
			name: "segment denied",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "segment denied",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := controller.FormatErrorMessage(tt.result, "test command")
			assert.NotEmpty(t, msg)
		})
	}
}

func TestFormatErrorMessage_AllCases(t *testing.T) {
	controller := NewCommandPermissionController()

	tests := []struct {
		name     string
		result   PermissionValidationResult
		contains string
	}{
		{
			name: "shell_operator_detected",
			result: PermissionValidationResult{
				Allowed:          false,
				Reason:           "shell_operator_detected",
				DetectedOperator: ";",
			},
			contains: "Detected ;",
		},
		{
			name: "redirect_detected",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "redirect_detected",
			},
			contains: "Redirect operators",
		},
		{
			name: "denied_with_pattern",
			result: PermissionValidationResult{
				Allowed:        false,
				Reason:         "denied",
				MatchedPattern: "rm -rf *",
				FailedSegment:  "rm -rf /",
			},
			contains: "rm -rf /",
		},
		{
			name: "denied_without_pattern",
			result: PermissionValidationResult{
				Allowed:       false,
				Reason:        "denied",
				FailedSegment: "dangerous",
			},
			contains: "explicitly denied",
		},
		{
			name: "segment_denied_with_pattern",
			result: PermissionValidationResult{
				Allowed:        false,
				Reason:         "segment_denied",
				MatchedPattern: "sudo *",
				FailedSegment:  "sudo su",
			},
			contains: "sudo su",
		},
		{
			name: "segment_denied_without_pattern",
			result: PermissionValidationResult{
				Allowed:       false,
				Reason:        "segment_denied",
				FailedSegment: "badcmd",
			},
			contains: "explicitly denied",
		},
		{
			name: "no_match_deny_default_with_pattern",
			result: PermissionValidationResult{
				Allowed:        false,
				Reason:         "no_match_deny_default",
				FailedSegment:  "unknown",
				MatchedPattern: "pattern",
			},
			contains: "did not match any allow pattern",
		},
		{
			name: "no_match_deny_default_without_pattern",
			result: PermissionValidationResult{
				Allowed:       false,
				Reason:        "no_match_deny_default",
				FailedSegment: "unknown",
			},
			contains: "Reason: no_match_deny_default",
		},
		{
			name: "segment_no_match_with_pattern",
			result: PermissionValidationResult{
				Allowed:        false,
				Reason:         "segment_no_match",
				FailedSegment:  "segment",
				MatchedPattern: "pattern",
			},
			contains: "did not match any allow pattern",
		},
		{
			name: "segment_no_match_without_pattern",
			result: PermissionValidationResult{
				Allowed:       false,
				Reason:        "segment_no_match",
				FailedSegment: "segment",
			},
			contains: "Reason: segment_no_match",
		},
		{
			name: "default_reason",
			result: PermissionValidationResult{
				Allowed: false,
				Reason:  "custom_reason",
			},
			contains: "custom_reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := controller.FormatErrorMessage(tt.result, "test command")
			assert.Contains(t, msg, tt.contains)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestCommandPermissionController_GetConfig(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		controller := NewCommandPermissionController()
		config := controller.GetConfig()
		// When no env is set, config should be nil
		assert.Nil(t, config)
	})

	t.Run("with valid config", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["git *"],"deny":[]}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller := NewCommandPermissionController()
		config := controller.GetConfig()
		assert.NotNil(t, config)
	})
}

func TestCommandPermissionController_GetConfig_WithEnv(t *testing.T) {
	// Set up environment
	os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["echo *"],"deny":[]}`)
	defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

	controller := NewCommandPermissionController()
	config := controller.GetConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.Allow)
	assert.NotNil(t, config.Deny)
}
