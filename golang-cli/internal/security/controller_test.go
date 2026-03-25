package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandPermissionController(t *testing.T) {
	t.Run("creates controller with no config when env not set", func(t *testing.T) {
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")
		controller := NewCommandPermissionController()
		assert.NotNil(t, controller)
		assert.False(t, controller.IsEnabled())
	})

	t.Run("creates controller with config when env is set", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["ls","cat"]}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller := NewCommandPermissionController()
		assert.NotNil(t, controller)
		assert.True(t, controller.IsEnabled())
	})
}

func TestCommandPermissionController_ValidateCommand(t *testing.T) {
	t.Run("allows all commands when no config", func(t *testing.T) {
		controller := NewCommandPermissionControllerWithConfig(nil)
		result := controller.ValidateCommand("rm -rf /")
		assert.True(t, result.Allowed)
		assert.Equal(t, "no_config", result.Reason)
	})

	t.Run("denies command matching deny pattern", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"*"},
			Deny:  []string{"rm*"},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("rm -rf /")
		assert.False(t, result.Allowed)
		assert.Equal(t, "denied", result.Reason)
		assert.Equal(t, "rm*", result.MatchedPattern)
	})

	t.Run("allows command matching allow pattern", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"ls*", "cat*", "echo*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("ls -la")
		assert.True(t, result.Allowed)
		assert.Equal(t, "allowed", result.Reason)
	})

	t.Run("denies command not matching allow pattern", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"ls", "cat"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("rm -rf /")
		assert.False(t, result.Allowed)
		assert.Equal(t, "no_match_deny_default", result.Reason)
	})

	t.Run("deny takes precedence over allow", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"*"},
			Deny:  []string{"rm*"},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("rm -rf /")
		assert.False(t, result.Allowed)
		assert.Equal(t, "denied", result.Reason)
	})

	t.Run("validates each segment of chained command - no match", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"cat*", "echo*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("ls -la && cat file")
		assert.False(t, result.Allowed)
		// The "ls -la" segment doesn't match any allow pattern
		assert.Equal(t, "segment_no_match", result.Reason)
		assert.Equal(t, "ls -la", result.FailedSegment)
	})

	t.Run("denies chained command with denied segment", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"ls*", "cat*", "echo*"},
			Deny:  []string{"rm*"},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		// "ls -la" matches "ls*" and is allowed
		// "rm -rf /" matches "rm*" deny pattern
		result := controller.ValidateCommand("ls -la && rm -rf /")
		assert.False(t, result.Allowed)
		assert.Equal(t, "segment_denied", result.Reason)
		assert.Equal(t, "rm -rf /", result.FailedSegment)
	})

	t.Run("allows chained command when all segments allowed", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"ls", "cat", "echo", "git*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("git status && git log")
		assert.True(t, result.Allowed)
	})

	t.Run("detects dangerous backticks", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("echo `whoami`")
		assert.False(t, result.Allowed)
		assert.Equal(t, "shell_operator_detected", result.Reason)
		assert.Contains(t, result.DetectedOperator, "backtick")
	})

	t.Run("detects dangerous subshell", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("echo $(whoami)")
		assert.False(t, result.Allowed)
		assert.Equal(t, "shell_operator_detected", result.Reason)
		assert.Contains(t, result.DetectedOperator, "subshell")
	})

	t.Run("allows backticks in single quotes", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("echo 'hello `world`'")
		assert.True(t, result.Allowed)
	})

	t.Run("blocks redirect when not allowed", func(t *testing.T) {
		config := &PermissionRules{
			Allow:          []string{"*"},
			Deny:           []string{},
			AllowRedirects: false,
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("echo hello > file.txt")
		assert.False(t, result.Allowed)
		assert.Equal(t, "redirect_detected", result.Reason)
	})

	t.Run("allows redirect when allowed", func(t *testing.T) {
		config := &PermissionRules{
			Allow:          []string{"*"},
			Deny:           []string{},
			AllowRedirects: true,
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("echo hello > file.txt")
		assert.True(t, result.Allowed)
	})

	t.Run("wildcard matching works", func(t *testing.T) {
		config := &PermissionRules{
			Allow: []string{"git*", "npm*"},
			Deny:  []string{},
		}
		controller := NewCommandPermissionControllerWithConfig(config)

		result := controller.ValidateCommand("git status")
		assert.True(t, result.Allowed)

		result = controller.ValidateCommand("npm install")
		assert.True(t, result.Allowed)

		result = controller.ValidateCommand("yarn add")
		assert.False(t, result.Allowed)
	})
}

func TestCommandPermissionController_matchesPattern(t *testing.T) {
	controller := NewCommandPermissionControllerWithConfig(&PermissionRules{
		Allow: []string{"*"},
	})

	tests := []struct {
		command string
		pattern string
		matches bool
	}{
		{"ls", "ls", true},
		{"ls -la", "ls*", true},
		{"git status", "git*", true},
		{"ls", "cat", false},
		{"ls", "*", true},
		{"git status", "git status", true},
		{"git log", "git status", false},
		{"cat", "c?t", true},
		{"cut", "c?t", true},
		{"c t", "c?t", true}, // ? matches any single character including space
	}

	for _, tt := range tests {
		t.Run(tt.command+"_"+tt.pattern, func(t *testing.T) {
			result := controller.matchesPattern(tt.command, tt.pattern)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestCommandPermissionController_parseCommandSegments(t *testing.T) {
	controller := NewCommandPermissionControllerWithConfig(&PermissionRules{
		Allow:     []string{"*"},
		AllowPipes: false,
	})

	t.Run("parses single command", func(t *testing.T) {
		segments := controller.parseCommandSegments("ls -la")
		assert.Equal(t, []string{"ls -la"}, segments)
	})

	t.Run("parses commands with &&", func(t *testing.T) {
		segments := controller.parseCommandSegments("git status && git log")
		assert.Equal(t, []string{"git status", "git log"}, segments)
	})

	t.Run("parses commands with ||", func(t *testing.T) {
		segments := controller.parseCommandSegments("test -f file || echo missing")
		assert.Equal(t, []string{"test -f file", "echo missing"}, segments)
	})

	t.Run("parses commands with ;", func(t *testing.T) {
		segments := controller.parseCommandSegments("cd /tmp; ls")
		assert.Equal(t, []string{"cd /tmp", "ls"}, segments)
	})

	t.Run("parses commands with | when pipes not allowed", func(t *testing.T) {
		segments := controller.parseCommandSegments("cat file | grep test")
		assert.Equal(t, []string{"cat file", "grep test"}, segments)
	})

	t.Run("does not split pipes when allowed", func(t *testing.T) {
		controllerWithPipes := NewCommandPermissionControllerWithConfig(&PermissionRules{
			Allow:     []string{"*"},
			AllowPipes: true,
		})
		segments := controllerWithPipes.parseCommandSegments("cat file | grep test")
		assert.Equal(t, []string{"cat file | grep test"}, segments)
	})

	t.Run("handles mixed operators", func(t *testing.T) {
		segments := controller.parseCommandSegments("cd /tmp && ls || echo fail")
		assert.Equal(t, []string{"cd /tmp", "ls", "echo fail"}, segments)
	})

	t.Run("handles empty commands", func(t *testing.T) {
		segments := controller.parseCommandSegments("  ")
		assert.Empty(t, segments)
	})
}

func TestCommandPermissionController_FormatErrorMessage(t *testing.T) {
	controller := NewCommandPermissionControllerWithConfig(&PermissionRules{
		Allow: []string{"*"},
	})

	t.Run("formats shell operator error", func(t *testing.T) {
		result := PermissionValidationResult{
			Allowed:          false,
			Reason:           "shell_operator_detected",
			DetectedOperator: "backtick_command_substitution",
		}
		msg := controller.FormatErrorMessage(result, "echo `whoami`")
		assert.Contains(t, msg, "CLINE_COMMAND_PERMISSIONS")
		assert.Contains(t, msg, "backtick")
	})

	t.Run("formats redirect error", func(t *testing.T) {
		result := PermissionValidationResult{
			Allowed: false,
			Reason:  "redirect_detected",
		}
		msg := controller.FormatErrorMessage(result, "echo hello > file")
		assert.Contains(t, msg, "CLINE_COMMAND_PERMISSIONS")
		assert.Contains(t, msg, "Redirect")
	})

	t.Run("formats denied error", func(t *testing.T) {
		result := PermissionValidationResult{
			Allowed:        false,
			Reason:         "denied",
			MatchedPattern: "rm*",
		}
		msg := controller.FormatErrorMessage(result, "rm -rf /")
		assert.Contains(t, msg, "denied")
		assert.Contains(t, msg, "rm*")
	})

	t.Run("formats segment denied error", func(t *testing.T) {
		result := PermissionValidationResult{
			Allowed:        false,
			Reason:         "segment_denied",
			MatchedPattern: "rm*",
			FailedSegment:  "rm -rf /",
		}
		msg := controller.FormatErrorMessage(result, "ls && rm -rf /")
		assert.Contains(t, msg, "Segment")
		assert.Contains(t, msg, "rm -rf /")
	})
}

func TestCommandPermissionController_ReloadConfig(t *testing.T) {
	t.Run("reloads config from environment", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["ls"]}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller := NewCommandPermissionControllerWithConfig(nil)
		require.False(t, controller.IsEnabled())

		err := controller.ReloadConfig()
		require.NoError(t, err)
		assert.True(t, controller.IsEnabled())
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{invalid}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller := NewCommandPermissionController()
		err := controller.ReloadConfig()
		assert.Error(t, err)
	})
}

func TestCheckAndLoadEnv(t *testing.T) {
	t.Run("returns nil when env not set", func(t *testing.T) {
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")
		controller, err := CheckAndLoadEnv()
		assert.NoError(t, err)
		assert.Nil(t, controller)
	})

	t.Run("returns controller when env is set", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["ls"]}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller, err := CheckAndLoadEnv()
		require.NoError(t, err)
		assert.NotNil(t, controller)
		assert.True(t, controller.IsEnabled())
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", `{invalid}`)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		controller, err := CheckAndLoadEnv()
		assert.Error(t, err)
		assert.Nil(t, controller)
	})
}

func TestEnvVarName(t *testing.T) {
	assert.Equal(t, "CLINE_COMMAND_PERMISSIONS", EnvVarName)
}

func BenchmarkValidateCommand(b *testing.B) {
	config := &PermissionRules{
		Allow: []string{"git*", "ls", "cat", "echo", "npm*", "docker*"},
		Deny:  []string{"rm*", "sudo*"},
	}
	controller := NewCommandPermissionControllerWithConfig(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.ValidateCommand("git status && git log")
	}
}