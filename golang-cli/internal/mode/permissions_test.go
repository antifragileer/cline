package mode

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NewPermissionValidator Tests ====================

func TestNewPermissionValidator(t *testing.T) {
	t.Run("creates validator with valid permissions", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow:          []string{"ls", "cat", "echo"},
			Deny:           []string{"rm", "sudo"},
			AllowRedirects: true,
			AllowPipes:     false,
		}

		v, err := NewPermissionValidator(perms)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Equal(t, perms, v.permissions)
		assert.Len(t, v.compiledAllow, 3)
		assert.Len(t, v.compiledDeny, 2)
	})

	t.Run("returns error for nil permissions", func(t *testing.T) {
		v, err := NewPermissionValidator(nil)
		assert.Error(t, err)
		assert.Nil(t, v)
		assert.Contains(t, err.Error(), "permissions cannot be nil")
	})

	t.Run("handles empty allow list", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow:          []string{},
			Deny:           []string{"*"},
			AllowRedirects: false,
			AllowPipes:     false,
		}

		v, err := NewPermissionValidator(perms)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Len(t, v.compiledAllow, 0)
	})

	t.Run("handles empty deny list", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow:          []string{"*"},
			Deny:           []string{},
			AllowRedirects: true,
			AllowPipes:     true,
		}

		v, err := NewPermissionValidator(perms)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.Len(t, v.compiledDeny, 0)
	})

	t.Run("returns error for empty allow pattern", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow: []string{""},
		}

		v, err := NewPermissionValidator(perms)
		assert.Error(t, err)
		assert.Nil(t, v)
	})

	t.Run("returns error for empty deny pattern", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow: []string{"*"},
			Deny:  []string{""},
		}

		v, err := NewPermissionValidator(perms)
		assert.Error(t, err)
		assert.Nil(t, v)
	})
}

// ==================== LoadPermissionsFromEnv Tests ====================

func TestLoadPermissionsFromEnv(t *testing.T) {
	t.Run("loads valid permissions from environment", func(t *testing.T) {
		jsonData := `{"allow":["ls","cat"],"deny":["rm"],"allowRedirects":true,"allowPipes":false}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := LoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, []string{"ls", "cat"}, perms.Allow)
		assert.Equal(t, []string{"rm"}, perms.Deny)
		assert.True(t, perms.AllowRedirects)
		assert.False(t, perms.AllowPipes)
	})

	t.Run("returns nil when environment variable not set", func(t *testing.T) {
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := LoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.Nil(t, perms)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "{invalid json")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := LoadPermissionsFromEnv()
		assert.Error(t, err)
		assert.Nil(t, perms)
		assert.Contains(t, err.Error(), "failed to parse CLINE_COMMAND_PERMISSIONS")
	})

	t.Run("handles complex JSON with nested structures", func(t *testing.T) {
		jsonData := `{
			"allow": ["git*", "ls", "cat", "echo"],
			"deny": ["rm -rf /", "sudo *"],
			"allowRedirects": true,
			"allowPipes": true
		}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := LoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Len(t, perms.Allow, 4)
		assert.Len(t, perms.Deny, 2)
	})

	t.Run("handles empty environment variable", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := LoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.Nil(t, perms)
	})
}

// ==================== MustLoadPermissionsFromEnv Tests ====================

func TestMustLoadPermissionsFromEnv(t *testing.T) {
	t.Run("returns permissions from environment when set", func(t *testing.T) {
		jsonData := `{"allow":["ls"],"deny":["rm"]}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := MustLoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, []string{"ls"}, perms.Allow)
	})

	t.Run("returns default permissive permissions when not set", func(t *testing.T) {
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := MustLoadPermissionsFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, []string{"*"}, perms.Allow)
		assert.Equal(t, []string{}, perms.Deny)
		assert.True(t, perms.AllowRedirects)
		assert.True(t, perms.AllowPipes)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "invalid")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		perms, err := MustLoadPermissionsFromEnv()
		assert.Error(t, err)
		assert.Nil(t, perms)
	})
}

// ==================== ValidateCommand Tests ====================

func TestPermissionValidator_ValidateCommand(t *testing.T) {
	t.Run("allows simple command matching allow list", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat", "echo"},
			Deny:  []string{},
		})

		err := v.ValidateCommand("ls")
		assert.NoError(t, err)

		err = v.ValidateCommand("cat file.txt")
		assert.NoError(t, err)

		err = v.ValidateCommand("echo hello world")
		assert.NoError(t, err)
	})

	t.Run("denies command matching deny list", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"*"},
			Deny:  []string{"rm", "sudo"},
		})

		err := v.ValidateCommand("rm file.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeDenied, valErr.ViolationType)

		err = v.ValidateCommand("sudo ls")
		assert.Error(t, err)
	})

	t.Run("deny takes precedence over allow", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"rm*"},
			Deny:  []string{"rm"},
		})

		// Should be denied because rm is in deny list
		err := v.ValidateCommand("rm file.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeDenied, valErr.ViolationType)
	})

	t.Run("rejects command not in allow list", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{},
		})

		err := v.ValidateCommand("rm file.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeNotAllowed, valErr.ViolationType)
	})

	t.Run("rejects empty command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"*"},
		})

		err := v.ValidateCommand("")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeInvalidCommand, valErr.ViolationType)
	})

	t.Run("handles commands with paths", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{},
		})

		err := v.ValidateCommand("/bin/ls")
		assert.NoError(t, err)

		err = v.ValidateCommand("/usr/bin/cat file.txt")
		assert.NoError(t, err)
	})

	t.Run("handles glob patterns in allow list", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"git*", "npm*"},
			Deny:  []string{},
		})

		err := v.ValidateCommand("git status")
		assert.NoError(t, err)

		err = v.ValidateCommand("git commit -m test")
		assert.NoError(t, err)

		err = v.ValidateCommand("npm install")
		assert.NoError(t, err)
	})

	t.Run("handles glob patterns in deny list", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"*"},
			Deny:  []string{"sudo*", "rm"},
		})

		err := v.ValidateCommand("sudo ls")
		assert.Error(t, err)

		err = v.ValidateCommand("rm -rf /")
		assert.Error(t, err)

		err = v.ValidateCommand("ls file.txt")
		assert.NoError(t, err)
	})

	t.Run("rejects redirects when not allowed", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow:          []string{"*"},
			Deny:           []string{},
			AllowRedirects: false,
		})

		err := v.ValidateCommand("ls > output.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeRedirectNotAllowed, valErr.ViolationType)

		err = v.ValidateCommand("cat file.txt >> output.txt")
		assert.Error(t, err)
	})

	t.Run("allows redirects when permitted", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow:          []string{"*"},
			AllowRedirects: true,
		})

		err := v.ValidateCommand("ls > output.txt")
		assert.NoError(t, err)

		err = v.ValidateCommand("cat file.txt >> output.txt")
		assert.NoError(t, err)
	})

	t.Run("handles compound commands with &&", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat", "echo"},
			Deny:  []string{"rm"},
		})

		// Both allowed
		err := v.ValidateCommand("ls && cat file.txt")
		assert.NoError(t, err)

		// Second denied
		err = v.ValidateCommand("ls && rm file.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, ViolationTypeDenied, valErr.ViolationType)
	})

	t.Run("handles compound commands with ||", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{"rm"},
		})

		err := v.ValidateCommand("ls || cat file.txt")
		assert.NoError(t, err)

		err = v.ValidateCommand("rm file.txt || ls")
		assert.Error(t, err)
	})

	t.Run("handles compound commands with semicolon", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat", "echo"},
			Deny:  []string{"rm"},
		})

		err := v.ValidateCommand("ls ; cat file.txt ; echo done")
		assert.NoError(t, err)

		err = v.ValidateCommand("ls ; rm file.txt")
		assert.Error(t, err)
	})

	t.Run("handles mixed compound commands", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat", "echo", "test"},
			Deny:  []string{"rm", "sudo"},
		})

		err := v.ValidateCommand("ls && cat file.txt || echo fallback")
		assert.NoError(t, err)

		err = v.ValidateCommand("ls ; rm file.txt && echo done")
		assert.Error(t, err)
	})

	t.Run("handles complex compound commands", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"git", "npm", "echo", "ls"},
			Deny:  []string{"rm", "sudo"},
		})

		err := v.ValidateCommand("git status && npm test || echo failed")
		assert.NoError(t, err)

		err = v.ValidateCommand("ls ; sudo su")
		assert.Error(t, err)
	})

	t.Run("validates all segments of compound command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls"},
			Deny:  []string{},
		})

		// First segment allowed, second not
		err := v.ValidateCommand("ls && cat file.txt")
		assert.Error(t, err)
		valErr := err.(*ValidationError)
		assert.Equal(t, "cat file.txt", valErr.Segment)
	})

	t.Run("handles whitespace in compound commands", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{},
		})

		err := v.ValidateCommand("  ls   &&   cat file.txt  ")
		assert.NoError(t, err)
	})

	t.Run("handles single character commands", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"a", "b", "c"},
		})

		err := v.ValidateCommand("a")
		assert.NoError(t, err)

		err = v.ValidateCommand("b arg")
		assert.NoError(t, err)
	})
}

// ==================== ValidateCommandWithDetails Tests ====================

func TestPermissionValidator_ValidateCommandWithDetails(t *testing.T) {
	t.Run("returns detailed results for allowed command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{},
		})

		result := v.ValidateCommandWithDetails("ls -la")
		assert.NotNil(t, result)
		assert.True(t, result.Allowed)
		assert.Equal(t, "ls -la", result.Command)
		assert.Len(t, result.Segments, 1)
		assert.True(t, result.Segments[0].Allowed)
		assert.Equal(t, "ls", result.Segments[0].MatchedAllowPattern)
	})

	t.Run("returns detailed results for denied command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"*"},
			Deny:  []string{"rm"},
		})

		result := v.ValidateCommandWithDetails("rm -rf /")
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Len(t, result.Segments, 1)
		assert.False(t, result.Segments[0].Allowed)
		assert.Equal(t, "rm", result.Segments[0].MatchedDenyPattern)
		assert.NotNil(t, result.Error)
	})

	t.Run("returns detailed results for compound command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"ls", "cat"},
			Deny:  []string{"rm"},
		})

		result := v.ValidateCommandWithDetails("ls && rm file.txt")
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.Len(t, result.Segments, 2)

		// First segment should be allowed
		assert.True(t, result.Segments[0].Allowed)
		assert.Equal(t, "ls", result.Segments[0].Segment)

		// Second segment should be denied
		assert.False(t, result.Segments[1].Allowed)
		assert.Equal(t, "rm file.txt", result.Segments[1].Segment)
		assert.Equal(t, "rm", result.Segments[1].MatchedDenyPattern)
	})

	t.Run("handles empty command", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow: []string{"*"},
		})

		result := v.ValidateCommandWithDetails("")
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.NotNil(t, result.Error)
		assert.Equal(t, ViolationTypeInvalidCommand, result.Error.ViolationType)
	})

	t.Run("handles redirect rejection", func(t *testing.T) {
		v := mustCreateValidator(t, &CommandPermissions{
			Allow:          []string{"*"},
			AllowRedirects: false,
		})

		result := v.ValidateCommandWithDetails("ls > output.txt")
		assert.NotNil(t, result)
		assert.False(t, result.Allowed)
		assert.NotNil(t, result.Error)
		assert.Equal(t, ViolationTypeRedirectNotAllowed, result.Error.ViolationType)
	})
}

// ==================== Glob Pattern Tests ====================

func TestCompileGlobPattern(t *testing.T) {
	t.Run("compiles exact pattern", func(t *testing.T) {
		pattern, err := compileGlobPattern("ls")
		require.NoError(t, err)
		assert.True(t, pattern.isExact)
		assert.True(t, pattern.matches("ls"))
		assert.False(t, pattern.matches("lsa"))
		assert.False(t, pattern.matches("ls -la"))
	})

	t.Run("compiles star wildcard", func(t *testing.T) {
		pattern, err := compileGlobPattern("git*")
		require.NoError(t, err)
		assert.False(t, pattern.isExact)
		assert.True(t, pattern.matches("git"))
		assert.True(t, pattern.matches("gitstatus"))
		assert.True(t, pattern.matches("git-commit"))
		assert.False(t, pattern.matches("mygit"))
	})

	t.Run("compiles question wildcard", func(t *testing.T) {
		pattern, err := compileGlobPattern("a?c")
		require.NoError(t, err)
		assert.True(t, pattern.matches("abc"))
		assert.True(t, pattern.matches("acc"))
		assert.False(t, pattern.matches("ac"))
		assert.False(t, pattern.matches("abbc"))
	})

	t.Run("compiles double star pattern", func(t *testing.T) {
		pattern, err := compileGlobPattern("**")
		require.NoError(t, err)
		assert.True(t, pattern.matches("anything"))
		assert.True(t, pattern.matches("a/b/c"))
	})

	t.Run("escapes regex special characters", func(t *testing.T) {
		pattern, err := compileGlobPattern("git.status")
		require.NoError(t, err)
		assert.True(t, pattern.matches("git.status"))
		assert.False(t, pattern.matches("gitXstatus"))
	})

	t.Run("returns error for empty pattern", func(t *testing.T) {
		pattern, err := compileGlobPattern("")
		assert.Error(t, err)
		assert.Nil(t, pattern)
	})

	t.Run("handles complex patterns", func(t *testing.T) {
		pattern, err := compileGlobPattern("npm-*")
		require.NoError(t, err)
		assert.True(t, pattern.matches("npm-install"))
		assert.True(t, pattern.matches("npm-test"))
		assert.False(t, pattern.matches("npm"))
	})
}

// ==================== ParseCommandSegments Tests ====================

func TestPermissionValidator_parseCommandSegments(t *testing.T) {
	v := mustCreateValidator(t, &CommandPermissions{
		Allow:          []string{"*"},
		AllowPipes:     false,
		AllowRedirects: true,
	})

	t.Run("parses simple command", func(t *testing.T) {
		segments := v.parseCommandSegments("ls")
		assert.Equal(t, []string{"ls"}, segments)
	})

	t.Run("parses command with &&", func(t *testing.T) {
		segments := v.parseCommandSegments("ls && cat file.txt")
		assert.Equal(t, []string{"ls", "cat file.txt"}, segments)
	})

	t.Run("parses command with ||", func(t *testing.T) {
		segments := v.parseCommandSegments("test || echo failed")
		assert.Equal(t, []string{"test", "echo failed"}, segments)
	})

	t.Run("parses command with semicolon", func(t *testing.T) {
		segments := v.parseCommandSegments("ls ; cat file.txt")
		assert.Equal(t, []string{"ls", "cat file.txt"}, segments)
	})

	t.Run("parses command with pipe", func(t *testing.T) {
		// Pipes should be parsed as separators when AllowPipes is false
		segments := v.parseCommandSegments("ls | cat")
		assert.Equal(t, []string{"ls", "cat"}, segments)
	})

	t.Run("parses complex compound command", func(t *testing.T) {
		segments := v.parseCommandSegments("ls && cat file.txt || echo done")
		assert.Equal(t, []string{"ls", "cat file.txt", "echo done"}, segments)
	})

	t.Run("handles whitespace", func(t *testing.T) {
		segments := v.parseCommandSegments("  ls   &&   cat file.txt  ")
		assert.Equal(t, []string{"ls", "cat file.txt"}, segments)
	})

	t.Run("handles empty segments", func(t *testing.T) {
		segments := v.parseCommandSegments("ls && && cat")
		// Empty segments should be filtered out
		assert.Equal(t, []string{"ls", "cat"}, segments)
	})

	t.Run("handles trailing operator", func(t *testing.T) {
		segments := v.parseCommandSegments("ls &&")
		assert.Equal(t, []string{"ls"}, segments)
	})

	t.Run("handles leading operator", func(t *testing.T) {
		segments := v.parseCommandSegments("&& ls")
		// This is edge case - && is at position 0
		assert.Equal(t, []string{"ls"}, segments)
	})
}

func TestPermissionValidator_parseCommandSegments_WithPipes(t *testing.T) {
	v := mustCreateValidator(t, &CommandPermissions{
		Allow:          []string{"*"},
		AllowPipes:     true,
		AllowRedirects: true,
	})

	t.Run("preserves pipes when allowed", func(t *testing.T) {
		segments := v.parseCommandSegments("ls | cat | grep test")
		// When pipes are allowed, they should not split segments
		assert.Equal(t, []string{"ls | cat | grep test"}, segments)
	})

	t.Run("still splits on && when pipes allowed", func(t *testing.T) {
		segments := v.parseCommandSegments("ls | cat && echo done")
		assert.Equal(t, []string{"ls | cat", "echo done"}, segments)
	})
}

// ==================== ExtractBaseCommand Tests ====================

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ls", "ls"},
		{"ls -la", "ls"},
		{"cat file.txt", "cat"},
		{"/bin/ls", "ls"},
		{"/usr/bin/cat file.txt", "cat"},
		{"  ls  ", "ls"},
		{"git status", "git"},
		{"/path/to/git commit", "git"},
		{`C:\Windows\System32\cmd.exe`, "cmd.exe"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractBaseCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== HasRedirect Tests ====================

func TestHasRedirect(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"ls > file.txt", true},
		{"ls >> file.txt", true},
		{"cat file.txt > output.txt", true},
		{"echo hello > /dev/null", true},
		{"echo hello 2> error.log", true},
		{"cmd &> output.log", true},
		{"ls", false},
		{"cat file.txt", false},
		{"echo hello world", false},
		{`echo ">"`, false},   // > in quotes should not count
		{`echo '>'`, false},   // > in single quotes should not count
		{`echo "hello >"`, false},
		{`echo ">" > file.txt`, true}, // > outside quotes
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := hasRedirect(tt.command)
			assert.Equal(t, tt.expected, result, "Command: %s", tt.command)
		})
	}
}

// ==================== IsAllowed Tests ====================

func TestPermissionValidator_IsAllowed(t *testing.T) {
	v := mustCreateValidator(t, &CommandPermissions{
		Allow: []string{"ls", "cat"},
		Deny:  []string{"rm"},
	})

	tests := []struct {
		command string
		allowed bool
	}{
		{"ls", true},
		{"ls -la", true},
		{"cat file.txt", true},
		{"rm file.txt", false},
		{"echo hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := v.IsAllowed(tt.command)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

// ==================== GetPermissions/SetPermissions Tests ====================

func TestPermissionValidator_GetPermissions(t *testing.T) {
	perms := &CommandPermissions{
		Allow:          []string{"ls"},
		Deny:           []string{"rm"},
		AllowRedirects: true,
		AllowPipes:     false,
	}

	v, err := NewPermissionValidator(perms)
	require.NoError(t, err)

	retrieved := v.GetPermissions()
	assert.Equal(t, perms, retrieved)
}

func TestPermissionValidator_SetPermissions(t *testing.T) {
	v := mustCreateValidator(t, &CommandPermissions{
		Allow: []string{"ls"},
		Deny:  []string{},
	})

	// Initially ls is allowed, cat is not
	assert.True(t, v.IsAllowed("ls"))
	assert.False(t, v.IsAllowed("cat"))

	// Update permissions
	newPerms := &CommandPermissions{
		Allow: []string{"cat"},
		Deny:  []string{"ls"},
	}

	err := v.SetPermissions(newPerms)
	require.NoError(t, err)

	// Now cat is allowed, ls is denied
	assert.False(t, v.IsAllowed("ls"))
	assert.True(t, v.IsAllowed("cat"))

	// Verify GetPermissions returns new permissions
	assert.Equal(t, newPerms, v.GetPermissions())
}

func TestPermissionValidator_SetPermissions_Invalid(t *testing.T) {
	v := mustCreateValidator(t, &CommandPermissions{
		Allow: []string{"ls"},
	})

	err := v.SetPermissions(&CommandPermissions{
		Allow: []string{""},
	})
	assert.Error(t, err)
}

// ==================== Default Validators Tests ====================

func TestDefaultDenyValidator(t *testing.T) {
	v := DefaultDenyValidator()
	require.NotNil(t, v)

	assert.False(t, v.IsAllowed("ls"))
	assert.False(t, v.IsAllowed("cat"))
	assert.False(t, v.IsAllowed("anything"))

	perms := v.GetPermissions()
	assert.Empty(t, perms.Allow)
	assert.Equal(t, []string{"*"}, perms.Deny)
	assert.False(t, perms.AllowRedirects)
	assert.False(t, perms.AllowPipes)
}

func TestDefaultAllowValidator(t *testing.T) {
	v := DefaultAllowValidator()
	require.NotNil(t, v)

	assert.True(t, v.IsAllowed("ls"))
	assert.True(t, v.IsAllowed("cat"))
	assert.True(t, v.IsAllowed("anything"))

	perms := v.GetPermissions()
	assert.Equal(t, []string{"*"}, perms.Allow)
	assert.Empty(t, perms.Deny)
	assert.True(t, perms.AllowRedirects)
	assert.True(t, perms.AllowPipes)
}

// ==================== ValidationError Tests ====================

func TestValidationError_Error(t *testing.T) {
	t.Run("error with segment different from command", func(t *testing.T) {
		err := &ValidationError{
			Command:       "ls && rm file.txt",
			Segment:       "rm file.txt",
			Reason:        "command \"rm\" matches denied pattern \"rm\"",
			ViolationType: ViolationTypeDenied,
		}

		msg := err.Error()
		assert.Contains(t, msg, "segment \"rm file.txt\"")
		assert.Contains(t, msg, "denied")
		assert.Contains(t, msg, "command \"rm\" matches denied pattern \"rm\"")
	})

	t.Run("error without segment", func(t *testing.T) {
		err := &ValidationError{
			Command:       "ls > file.txt",
			Reason:        "output redirects are not permitted",
			ViolationType: ViolationTypeRedirectNotAllowed,
		}

		msg := err.Error()
		assert.Contains(t, msg, "ls > file.txt")
		assert.Contains(t, msg, "redirect_not_allowed")
	})

	t.Run("error with same segment and command", func(t *testing.T) {
		err := &ValidationError{
			Command:       "rm file.txt",
			Segment:       "rm file.txt",
			Reason:        "command denied",
			ViolationType: ViolationTypeDenied,
		}

		msg := err.Error()
		assert.Contains(t, msg, "\"rm file.txt\"")
		assert.NotContains(t, msg, "segment") // Should not have "segment" prefix when they're the same
	})
}

// ==================== Integration Tests ====================

func TestPermissionValidator_Integration(t *testing.T) {
	t.Run("complex real-world scenario", func(t *testing.T) {
		// Simulate a CI/CD environment that only allows specific safe commands
		perms := &CommandPermissions{
			Allow: []string{
				"git*",
				"npm*",
				"echo",
				"cat",
				"ls",
				"mkdir",
				"cp",
				"mv",
			},
			Deny: []string{
				"sudo*",
				"rm -rf*",
				"rm -f /",
				"*> /dev/sda",
				"*> /dev/null", // Not actually dangerous, but for testing
				"dd if=*",
			},
			AllowRedirects: false,
			AllowPipes:     false,
		}

		v, err := NewPermissionValidator(perms)
		require.NoError(t, err)

		// Should allow
		assert.NoError(t, v.ValidateCommand("git status"))
		assert.NoError(t, v.ValidateCommand("git commit -m \"test\""))
		assert.NoError(t, v.ValidateCommand("npm install"))
		assert.NoError(t, v.ValidateCommand("npm run build"))
		assert.NoError(t, v.ValidateCommand("echo hello world"))
		assert.NoError(t, v.ValidateCommand("ls -la"))
		assert.NoError(t, v.ValidateCommand("cat file.txt"))

		// Should deny - not in allow list
		assert.Error(t, v.ValidateCommand("python script.py"))

		// Should deny - matches deny pattern
		assert.Error(t, v.ValidateCommand("sudo ls"))
		assert.Error(t, v.ValidateCommand("rm -rf /"))

		// Should deny - has redirect
		assert.Error(t, v.ValidateCommand("echo hello > file.txt"))

		// Should deny - compound command with denied segment
		assert.Error(t, v.ValidateCommand("git status && sudo ls"))
	})

	t.Run("development environment with permissive settings", func(t *testing.T) {
		perms := &CommandPermissions{
			Allow:          []string{"*"},
			Deny:           []string{"sudo*"},
			AllowRedirects: true,
			AllowPipes:     true,
		}

		v, err := NewPermissionValidator(perms)
		require.NoError(t, err)

		// Should allow most things
		assert.NoError(t, v.ValidateCommand("ls"))
		assert.NoError(t, v.ValidateCommand("cat file.txt | grep test"))
		assert.NoError(t, v.ValidateCommand("echo hello > file.txt"))
		assert.NoError(t, v.ValidateCommand("complex_command with args"))

		// Should still deny sudo
		assert.Error(t, v.ValidateCommand("sudo su"))
	})
}

// ==================== Helper Functions ====================

func mustCreateValidator(t *testing.T, perms *CommandPermissions) *PermissionValidator {
	v, err := NewPermissionValidator(perms)
	require.NoError(t, err)
	return v
}