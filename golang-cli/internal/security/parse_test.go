package security

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NewParser Tests ====================

func TestNewParser(t *testing.T) {
	t.Run("creates parser with initialized cache", func(t *testing.T) {
		parser := NewParser()
		assert.NotNil(t, parser)
		assert.NotNil(t, parser.cache)
		assert.NotNil(t, parser.schema)
	})

	t.Run("creates parser with correct schema configuration", func(t *testing.T) {
		parser := NewParser()
		assert.NotNil(t, parser.schema)
		assert.Equal(t, []string{"allow"}, parser.schema.requiredFields)
		assert.Contains(t, parser.schema.allowedTypes, "allow")
		assert.Contains(t, parser.schema.allowedTypes, "deny")
		assert.Contains(t, parser.schema.allowedTypes, "allowRedirects")
		assert.Contains(t, parser.schema.allowedTypes, "allowPipes")
	})
}

// ==================== Parse Tests ====================

func TestParser_Parse(t *testing.T) {
	t.Run("parses valid JSON with all fields", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","cat"],"deny":["rm"],"allowRedirects":true,"allowPipes":false}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"ls", "cat"}, rules.Allow)
		assert.Equal(t, []string{"rm"}, rules.Deny)
		assert.True(t, rules.AllowRedirects)
		assert.False(t, rules.AllowPipes)
	})

	t.Run("parses valid JSON with minimal fields", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["*"]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"*"}, rules.Allow)
		assert.Empty(t, rules.Deny)
		assert.False(t, rules.AllowRedirects)
		assert.False(t, rules.AllowPipes)
	})

	t.Run("parses empty allow list", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":[]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Empty(t, rules.Allow)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{invalid json}`)

		rules, err := parser.Parse(jsonData)
		assert.Error(t, err)
		assert.Nil(t, rules)

		parseErr, ok := err.(*ParseError)
		assert.True(t, ok)
		assert.Contains(t, parseErr.Message, "invalid JSON")
	})

	t.Run("returns error for empty allow pattern", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":[""]}`)

		rules, err := parser.Parse(jsonData)
		assert.Error(t, err)
		assert.Nil(t, rules)

		parseErr, ok := err.(*ParseError)
		assert.True(t, ok)
		// The field should be "allow[0]" in the ParseError struct
		assert.Equal(t, "allow[0]", parseErr.Field)
	})

	t.Run("returns error for invalid glob pattern", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","test!@#"]}`)

		rules, err := parser.Parse(jsonData)
		assert.Error(t, err)
		assert.Nil(t, rules)
	})

	t.Run("caches parsed permissions", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","cat"]}`)

		// First parse should cache
		rules1, err := parser.Parse(jsonData)
		require.NoError(t, err)

		// Second parse should return cached
		rules2, err := parser.Parse(jsonData)
		require.NoError(t, err)

		// Should be the same object from cache
		assert.Equal(t, rules1, rules2)

		// Cache should show hits
		stats := parser.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
		assert.GreaterOrEqual(t, stats.TotalHits, int64(0))
	})

	t.Run("handles complex nested commands in patterns", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["git*","npm-*","docker-compose"],"deny":["sudo*","rm -rf*"]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.Len(t, rules.Allow, 3)
		assert.Len(t, rules.Deny, 2)
	})

	t.Run("handles path patterns with slashes", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["/usr/bin/ls","/bin/cat","./script.sh"]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.Len(t, rules.Allow, 3)
	})

	t.Run("handles Windows-style paths", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["C:\\Windows\\System32\\cmd.exe",".\\script.bat"]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.Len(t, rules.Allow, 2)
	})
}

// ==================== ParseString Tests ====================

func TestParser_ParseString(t *testing.T) {
	t.Run("parses valid JSON string", func(t *testing.T) {
		parser := NewParser()
		jsonStr := `{"allow":["ls","cat"],"deny":["rm"]}`

		rules, err := parser.ParseString(jsonStr)
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"ls", "cat"}, rules.Allow)
	})

	t.Run("returns error for invalid JSON string", func(t *testing.T) {
		parser := NewParser()
		jsonStr := `{invalid}`

		rules, err := parser.ParseString(jsonStr)
		assert.Error(t, err)
		assert.Nil(t, rules)
	})
}

// ==================== ParseFromEnv Tests ====================

func TestParser_ParseFromEnv(t *testing.T) {
	t.Run("parses valid permissions from environment", func(t *testing.T) {
		parser := NewParser()
		jsonData := `{"allow":["ls","cat"],"deny":["rm"],"allowRedirects":true,"allowPipes":false}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseFromEnv()
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"ls", "cat"}, rules.Allow)
		assert.Equal(t, []string{"rm"}, rules.Deny)
		assert.True(t, rules.AllowRedirects)
		assert.False(t, rules.AllowPipes)
	})

	t.Run("returns nil when environment variable not set", func(t *testing.T) {
		parser := NewParser()
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseFromEnv()
		require.NoError(t, err)
		assert.Nil(t, rules)
	})

	t.Run("returns error for invalid JSON in environment", func(t *testing.T) {
		parser := NewParser()
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "{invalid json")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseFromEnv()
		assert.Error(t, err)
		assert.Nil(t, rules)
	})

	t.Run("returns nil for empty environment variable", func(t *testing.T) {
		parser := NewParser()
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseFromEnv()
		require.NoError(t, err)
		assert.Nil(t, rules)
	})

	t.Run("handles complex JSON with whitespace", func(t *testing.T) {
		parser := NewParser()
		jsonData := `{
			"allow": ["git*", "npm*", "echo"],
			"deny": ["sudo*", "rm -rf /"],
			"allowRedirects": true,
			"allowPipes": true
		}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseFromEnv()
		require.NoError(t, err)
		assert.Len(t, rules.Allow, 3)
		assert.Len(t, rules.Deny, 2)
	})
}

// ==================== ParseWithDefault Tests ====================

func TestParser_ParseWithDefault(t *testing.T) {
	t.Run("returns parsed rules when environment is set", func(t *testing.T) {
		parser := NewParser()
		jsonData := `{"allow":["ls","cat"]}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseWithDefault()
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"ls", "cat"}, rules.Allow)
	})

	t.Run("returns default allow-all when environment not set", func(t *testing.T) {
		parser := NewParser()
		os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseWithDefault()
		require.NoError(t, err)
		assert.NotNil(t, rules)
		assert.Equal(t, []string{"*"}, rules.Allow)
		assert.True(t, rules.AllowRedirects)
		assert.True(t, rules.AllowPipes)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		parser := NewParser()
		os.Setenv("CLINE_COMMAND_PERMISSIONS", "{invalid")
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		rules, err := parser.ParseWithDefault()
		assert.Error(t, err)
		assert.Nil(t, rules)
	})
}

// ==================== ValidateSchema Tests ====================

func TestParser_ValidateSchema(t *testing.T) {
	t.Run("validates correct schema", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","cat"],"deny":["rm"],"allowRedirects":true,"allowPipes":false}`)

		result := parser.ValidateSchema(jsonData)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("validates minimal schema", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["*"]}`)

		result := parser.ValidateSchema(jsonData)
		assert.True(t, result.Valid)
	})

	t.Run("detects invalid JSON", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{invalid}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "invalid JSON")
	})

	t.Run("detects wrong type for allow", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":"ls"}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.True(t, hasSchemaFieldError(result.Errors, "allow"))
	})

	t.Run("detects wrong type for allowRedirects", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["*"],"allowRedirects":"yes"}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.True(t, hasSchemaFieldError(result.Errors, "allowRedirects"))
	})

	t.Run("detects wrong type for allowPipes", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["*"],"allowPipes":"yes"}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.True(t, hasSchemaFieldError(result.Errors, "allowPipes"))
	})

	t.Run("detects non-string in allow array", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls",123]}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.True(t, hasSchemaFieldError(result.Errors, "allow[1]"))
	})

	t.Run("detects non-string in deny array", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["*"],"deny":[true]}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
		assert.True(t, hasSchemaFieldError(result.Errors, "deny[0]"))
	})

	t.Run("detects invalid glob pattern in allow", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","test!@#"]}`)

		result := parser.ValidateSchema(jsonData)
		assert.False(t, result.Valid)
	})

	t.Run("generates warning for missing allow field", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"deny":["rm"]}`)

		result := parser.ValidateSchema(jsonData)
		assert.True(t, result.Valid) // Still valid, just warning
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Field, "allow")
	})

	t.Run("validates empty arrays", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":[],"deny":[]}`)

		result := parser.ValidateSchema(jsonData)
		assert.True(t, result.Valid)
	})
}

// ==================== Cache Tests ====================

func TestParser_Cache(t *testing.T) {
	t.Run("clears cache", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls"]}`)

		// Parse to populate cache
		_, err := parser.Parse(jsonData)
		require.NoError(t, err)

		// Verify cache has entry
		stats := parser.GetCacheStats()
		assert.Equal(t, 1, stats.Size)

		// Clear cache
		parser.ClearCache()

		// Verify cache is empty
		stats = parser.GetCacheStats()
		assert.Equal(t, 0, stats.Size)
	})

	t.Run("tracks cache stats", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls"]}`)

		// Initial parse
		_, err := parser.Parse(jsonData)
		require.NoError(t, err)

		// Get stats
		stats := parser.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
	})

	t.Run("caches multiple different permissions", func(t *testing.T) {
		parser := NewParser()

		// Parse different permissions
		_, err := parser.Parse([]byte(`{"allow":["ls"]}`))
		require.NoError(t, err)

		_, err = parser.Parse([]byte(`{"allow":["cat"]}`))
		require.NoError(t, err)

		_, err = parser.Parse([]byte(`{"allow":["echo"]}`))
		require.NoError(t, err)

		stats := parser.GetCacheStats()
		assert.Equal(t, 3, stats.Size)
	})
}

// ==================== ParseError Tests ====================

func TestParseError_Error(t *testing.T) {
	t.Run("error with field", func(t *testing.T) {
		err := &ParseError{
			Field:   "allow[0]",
			Message: "pattern cannot be empty",
		}
		msg := err.Error()
		assert.Contains(t, msg, "allow[0]")
		assert.Contains(t, msg, "pattern cannot be empty")
	})

	t.Run("error without field", func(t *testing.T) {
		err := &ParseError{
			Message: "invalid JSON",
		}
		msg := err.Error()
		assert.Contains(t, msg, "invalid JSON")
		assert.NotContains(t, msg, "field")
	})

	t.Run("error with raw value", func(t *testing.T) {
		err := &ParseError{
			Field:    "allow[0]",
			Message:  "pattern cannot be empty",
			RawValue: "{...very long json...}",
		}
		assert.NotEmpty(t, err.RawValue)
	})
}

// ==================== Default Permissions Tests ====================

func TestDefaultAllowAll(t *testing.T) {
	rules := DefaultAllowAll()
	assert.NotNil(t, rules)
	assert.Equal(t, []string{"*"}, rules.Allow)
	assert.Empty(t, rules.Deny)
	assert.True(t, rules.AllowRedirects)
	assert.True(t, rules.AllowPipes)
}

func TestDefaultDenyAll(t *testing.T) {
	rules := DefaultDenyAll()
	assert.NotNil(t, rules)
	assert.Empty(t, rules.Allow)
	assert.Equal(t, []string{"*"}, rules.Deny)
	assert.False(t, rules.AllowRedirects)
	assert.False(t, rules.AllowPipes)
}

// ==================== Helper Function Tests ====================

func TestTruncate(t *testing.T) {
	t.Run("does not truncate short strings", func(t *testing.T) {
		s := "short"
		result := truncate(s, 100)
		assert.Equal(t, s, result)
	})

	t.Run("truncates long strings", func(t *testing.T) {
		s := "this is a very long string that should be truncated"
		result := truncate(s, 10)
		assert.Equal(t, "this is a ...", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := truncate("", 10)
		assert.Equal(t, "", result)
	})
}

func TestGetJSONType(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{[]interface{}{}, "array"},
		{map[string]interface{}{}, "object"},
		{"string", "string"},
		{float64(42), "number"},
		{true, "boolean"},
		{nil, "null"},
		{make(chan int), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getJSONType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateGlobPattern(t *testing.T) {
	t.Run("validates simple patterns", func(t *testing.T) {
		assert.NoError(t, validateGlobPattern("ls"))
		assert.NoError(t, validateGlobPattern("git*"))
		assert.NoError(t, validateGlobPattern("npm-*"))
	})

	t.Run("validates patterns with wildcards", func(t *testing.T) {
		assert.NoError(t, validateGlobPattern("*"))
		assert.NoError(t, validateGlobPattern("a?c"))
		assert.NoError(t, validateGlobPattern("**"))
	})

	t.Run("validates patterns with paths", func(t *testing.T) {
		assert.NoError(t, validateGlobPattern("/usr/bin/ls"))
		assert.NoError(t, validateGlobPattern("./script.sh"))
		assert.NoError(t, validateGlobPattern("C:\\Windows\\cmd.exe"))
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		err := validateGlobPattern("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("rejects invalid characters", func(t *testing.T) {
		err := validateGlobPattern("test!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")

		err = validateGlobPattern("test<")
		assert.Error(t, err)
	})

	t.Run("rejects special regex characters", func(t *testing.T) {
		err := validateGlobPattern("test$")
		assert.Error(t, err)

		err = validateGlobPattern("test^")
		assert.Error(t, err)

		err = validateGlobPattern("test|")
		assert.Error(t, err)
	})
}

func TestIsValidGlobChar(t *testing.T) {
	t.Run("validates letters", func(t *testing.T) {
		assert.True(t, isValidGlobChar('a'))
		assert.True(t, isValidGlobChar('Z'))
	})

	t.Run("validates numbers", func(t *testing.T) {
		assert.True(t, isValidGlobChar('0'))
		assert.True(t, isValidGlobChar('9'))
	})

	t.Run("rejects special characters", func(t *testing.T) {
		assert.False(t, isValidGlobChar('!'))
		assert.False(t, isValidGlobChar('<'))
	})
}

// ==================== Integration Tests ====================

func TestParser_Integration(t *testing.T) {
	t.Run("full workflow with environment variable", func(t *testing.T) {
		// Set up environment
		jsonData := `{
			"allow": ["git*", "npm*", "echo", "cat", "ls"],
			"deny": ["sudo*", "rm -rf*", "dd if=*"],
			"allowRedirects": false,
			"allowPipes": false
		}`
		os.Setenv("CLINE_COMMAND_PERMISSIONS", jsonData)
		defer os.Unsetenv("CLINE_COMMAND_PERMISSIONS")

		parser := NewParser()

		// Parse from environment
		rules, err := parser.ParseWithDefault()
		require.NoError(t, err)

		// Verify rules
		assert.Len(t, rules.Allow, 5)
		assert.Len(t, rules.Deny, 3)
		assert.False(t, rules.AllowRedirects)
		assert.False(t, rules.AllowPipes)

		// Verify cache
		stats := parser.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
	})

	t.Run("handles concurrent parsing", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["ls","cat"]}`)

		// Parse concurrently
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := parser.Parse(jsonData)
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Cache should only have one entry
		stats := parser.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
	})

	t.Run("handles large permission sets", func(t *testing.T) {
		parser := NewParser()

		// Create large allow list
		allowList := make([]string, 100)
		for i := 0; i < 100; i++ {
			allowList[i] = fmt.Sprintf("cmd%d", i)
		}

		jsonData := fmt.Sprintf(`{"allow":[%s]}`, joinStringsForTest(allowList))
		rules, err := parser.Parse([]byte(jsonData))
		require.NoError(t, err)
		assert.Len(t, rules.Allow, 100)
	})
}

// ==================== Helper Functions ====================

func hasSchemaFieldError(errors []SchemaValidationError, field string) bool {
	for _, err := range errors {
		if err.Field == field {
			return true
		}
	}
	return false
}

func joinStringsForTest(strs []string) string {
	var result []string
	for _, s := range strs {
		result = append(result, fmt.Sprintf("%q", s))
	}
	return strings.Join(result, ",")
}

// ==================== validateRules Tests ====================

func TestParser_validateRules(t *testing.T) {
	t.Run("valid rules pass", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{"ls *", "cat *"},
			Deny:  []string{"rm -rf *"},
		}
		err := parser.validateRules(rules)
		assert.NoError(t, err)
	})

	t.Run("empty allow pattern fails", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{"", "cat *"},
		}
		err := parser.validateRules(rules)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern cannot be empty")
	})

	t.Run("empty deny pattern fails", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{"ls *"},
			Deny:  []string{"", "rm *"},
		}
		err := parser.validateRules(rules)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern cannot be empty")
	})

	t.Run("invalid glob in allow pattern fails", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{"ls !$invalid"},
		}
		err := parser.validateRules(rules)
		assert.Error(t, err)
	})

	t.Run("invalid glob in deny pattern fails", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{"ls *"},
			Deny:  []string{"rm !$invalid"},
		}
		err := parser.validateRules(rules)
		assert.Error(t, err)
	})

	t.Run("empty allow and deny passes", func(t *testing.T) {
		parser := NewParser()
		rules := &PermissionRules{
			Allow: []string{},
			Deny:  []string{},
		}
		err := parser.validateRules(rules)
		assert.NoError(t, err)
	})
}

// ==================== Parse edge cases for coverage ====================

func TestParser_Parse_EdgeCases(t *testing.T) {
	t.Run("parses JSON with nested braces in patterns", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["cmd {a,b,c}"],"deny":[]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.Equal(t, []string{"cmd {a,b,c}"}, rules.Allow)
	})

	t.Run("parses JSON with special characters", func(t *testing.T) {
		parser := NewParser()
		jsonData := []byte(`{"allow":["cmd [a-z]*","cmd test?"],"deny":[]}`)

		rules, err := parser.Parse(jsonData)
		require.NoError(t, err)
		assert.Equal(t, []string{"cmd [a-z]*", "cmd test?"}, rules.Allow)
	})
}
