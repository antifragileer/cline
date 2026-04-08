package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDangerType_String(t *testing.T) {
	tests := []struct {
		name     string
		dt       DangerType
		expected string
	}{
		{"backtick", DangerTypeBacktick, "backtick_command_substitution"},
		{"subshell", DangerTypeSubshell, "subshell_command_substitution"},
		{"newline", DangerTypeNewline, "unquoted_newline"},
		{"unknown", DangerType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dt.String())
		})
	}
}

func TestDanger_String(t *testing.T) {
	d := Danger{
		Type:        DangerTypeBacktick,
		Start:       5,
		End:         12,
		Content:     "`whoami`",
		Description: "backtick command substitution detected",
	}

	expected := "backtick_command_substitution at [5:12]: `whoami` (backtick command substitution detected)"
	assert.Equal(t, expected, d.String())
}

func TestNewDetector(t *testing.T) {
	t.Run("creates detector with input", func(t *testing.T) {
		input := "test input"
		d := NewDetector(input)

		require.NotNil(t, d)
		assert.Equal(t, input, d.input)
		assert.Equal(t, len(input), d.length)
	})

	t.Run("creates detector with empty input", func(t *testing.T) {
		d := NewDetector("")

		require.NotNil(t, d)
		assert.Equal(t, "", d.input)
		assert.Equal(t, 0, d.length)
	})
}

func TestDetector_Detect_Backticks(t *testing.T) {
	t.Run("detects simple backtick command", func(t *testing.T) {
		input := "`whoami`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, 0, dangers[0].Start)
		assert.Equal(t, 8, dangers[0].End)
		assert.Equal(t, "`whoami`", dangers[0].Content)
	})

	t.Run("detects backtick with spaces", func(t *testing.T) {
		input := "`echo hello world`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, "`echo hello world`", dangers[0].Content)
	})

	t.Run("detects backtick in middle of text", func(t *testing.T) {
		input := "Hello `whoami` there"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, 6, dangers[0].Start)
		assert.Equal(t, 14, dangers[0].End)
		assert.Equal(t, "`whoami`", dangers[0].Content)
	})

	t.Run("detects multiple backticks", func(t *testing.T) {
		input := "`whoami` and `hostname`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, "`whoami`", dangers[0].Content)
		assert.Equal(t, DangerTypeBacktick, dangers[1].Type)
		assert.Equal(t, "`hostname`", dangers[1].Content)
	})

	t.Run("detects backtick without closing", func(t *testing.T) {
		input := "`whoami"
		d := NewDetector(input)
		dangers := d.Detect()

		// Should still detect the opening backtick as dangerous
		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("backticks inside single quotes are safe", func(t *testing.T) {
		input := "'`whoami`'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("backticks inside double quotes are still dangerous", func(t *testing.T) {
		input := "\"`whoami`\""
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("backticks after closing single quote are dangerous", func(t *testing.T) {
		input := "'safe' `dangerous`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, 7, dangers[0].Start)
	})
}

func TestDetector_Detect_Subshells(t *testing.T) {
	t.Run("detects simple subshell", func(t *testing.T) {
		input := "$(whoami)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
		assert.Equal(t, 0, dangers[0].Start)
		assert.Equal(t, 9, dangers[0].End)
		assert.Equal(t, "$(whoami)", dangers[0].Content)
	})

	t.Run("detects subshell with command", func(t *testing.T) {
		input := "$(echo hello world)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "$(echo hello world)", dangers[0].Content)
	})

	t.Run("detects subshell in middle of text", func(t *testing.T) {
		input := "Hello $(whoami) there"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
		assert.Equal(t, 6, dangers[0].Start)
		assert.Equal(t, 15, dangers[0].End)
	})

	t.Run("detects multiple subshells", func(t *testing.T) {
		input := "$(whoami) and $(hostname)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, "$(whoami)", dangers[0].Content)
		assert.Equal(t, "$(hostname)", dangers[1].Content)
	})

	t.Run("detects nested subshells", func(t *testing.T) {
		input := "$(echo $(whoami))"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "$(echo $(whoami))", dangers[0].Content)
	})

	t.Run("detects deeply nested subshells", func(t *testing.T) {
		input := "$(a $(b $(c)))"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "$(a $(b $(c)))", dangers[0].Content)
	})

	t.Run("subshell without closing paren", func(t *testing.T) {
		input := "$(whoami"
		d := NewDetector(input)
		dangers := d.Detect()

		// Should still detect the opening as dangerous
		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("subshell inside single quotes are safe", func(t *testing.T) {
		input := "'$(whoami)'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("subshell inside double quotes are still dangerous", func(t *testing.T) {
		input := "\"$(whoami)\""
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("subshell after closing single quote are dangerous", func(t *testing.T) {
		input := "'safe' $(dangerous)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("dollar sign without paren is not dangerous", func(t *testing.T) {
		input := "$HOME"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("dollar at end of string is not dangerous", func(t *testing.T) {
		input := "test$"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("subshell with single quotes inside", func(t *testing.T) {
		input := "$(echo 'hello world')"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "$(echo 'hello world')", dangers[0].Content)
	})

	t.Run("subshell with escaped parens", func(t *testing.T) {
		input := "$(echo \\))"
		d := NewDetector(input)
		dangers := d.Detect()

		// The escaped paren shouldn't close the subshell
		// But our simple parser will treat it as escaped
		require.Len(t, dangers, 1)
	})
}

func TestDetector_Detect_Newlines(t *testing.T) {
	t.Run("detects unquoted newline", func(t *testing.T) {
		input := "hello\nworld"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
		assert.Equal(t, 5, dangers[0].Start)
		assert.Equal(t, 6, dangers[0].End)
		assert.Equal(t, "\n", dangers[0].Content)
	})

	t.Run("detects multiple unquoted newlines", func(t *testing.T) {
		input := "a\nb\nc"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
		assert.Equal(t, DangerTypeNewline, dangers[1].Type)
	})

	t.Run("newline inside single quotes is safe", func(t *testing.T) {
		input := "'hello\nworld'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("newline inside double quotes is still dangerous", func(t *testing.T) {
		input := "\"hello\nworld\""
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
	})

	t.Run("newline after closing single quote is dangerous", func(t *testing.T) {
		input := "'safe'\ndangerous"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
		assert.Equal(t, 6, dangers[0].Start)
	})

	t.Run("newline at start of string", func(t *testing.T) {
		input := "\nhello"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, 0, dangers[0].Start)
	})

	t.Run("newline at end of string", func(t *testing.T) {
		input := "hello\n"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, 5, dangers[0].Start)
	})

	t.Run("only newline", func(t *testing.T) {
		input := "\n"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
	})
}

func TestDetector_Detect_Combined(t *testing.T) {
	t.Run("backtick and subshell together", func(t *testing.T) {
		input := "`whoami` $(hostname)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, DangerTypeSubshell, dangers[1].Type)
	})

	t.Run("backtick and newline", func(t *testing.T) {
		input := "`whoami`\nnext line"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, DangerTypeNewline, dangers[1].Type)
	})

	t.Run("subshell and newline", func(t *testing.T) {
		input := "$(whoami)\nnext line"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
		assert.Equal(t, DangerTypeNewline, dangers[1].Type)
	})

	t.Run("all three dangers", func(t *testing.T) {
		input := "`whoami` $(hostname)\nnext"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 3)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, DangerTypeSubshell, dangers[1].Type)
		assert.Equal(t, DangerTypeNewline, dangers[2].Type)
	})

	t.Run("all safe inside single quotes", func(t *testing.T) {
		input := "'`whoami` $(hostname)\nnext'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("mixed quoting - partial protection", func(t *testing.T) {
		input := "'safe'`dangerous`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, "`dangerous`", dangers[0].Content)
	})

	t.Run("double quotes do not protect", func(t *testing.T) {
		input := "\"`bt` $(sub) \n nl\""
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 3)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
		assert.Equal(t, DangerTypeSubshell, dangers[1].Type)
		assert.Equal(t, DangerTypeNewline, dangers[2].Type)
	})

	t.Run("single quote re-enables detection", func(t *testing.T) {
		input := "'safe'`dangerous`'safe2'`dangerous2`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, "`dangerous`", dangers[0].Content)
		assert.Equal(t, "`dangerous2`", dangers[1].Content)
	})

	t.Run("empty single quotes", func(t *testing.T) {
		input := "''`dangerous`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "`dangerous`", dangers[0].Content)
	})

	t.Run("unclosed single quote protects rest", func(t *testing.T) {
		input := "'`dangerous` never closed"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})
}

func TestDetector_HasDangerousCharacters(t *testing.T) {
	t.Run("returns true when dangerous characters present", func(t *testing.T) {
		d := NewDetector("`whoami`")
		assert.True(t, d.HasDangerousCharacters())
	})

	t.Run("returns false when no dangerous characters", func(t *testing.T) {
		d := NewDetector("safe input")
		assert.False(t, d.HasDangerousCharacters())
	})

	t.Run("returns false for empty string", func(t *testing.T) {
		d := NewDetector("")
		assert.False(t, d.HasDangerousCharacters())
	})
}

func TestDetector_Count(t *testing.T) {
	t.Run("returns correct count for multiple dangers", func(t *testing.T) {
		d := NewDetector("`a` `b` `c`")
		assert.Equal(t, 3, d.Count())
	})

	t.Run("returns zero for safe input", func(t *testing.T) {
		d := NewDetector("safe input")
		assert.Equal(t, 0, d.Count())
	})
}

func TestValidateInput(t *testing.T) {
	t.Run("returns nil for safe input", func(t *testing.T) {
		err := ValidateInput("safe input")
		assert.NoError(t, err)
	})

	t.Run("returns error for dangerous input", func(t *testing.T) {
		err := ValidateInput("`whoami`")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous characters detected")
		assert.Contains(t, err.Error(), "backtick_command_substitution")
	})

	t.Run("error includes all dangers", func(t *testing.T) {
		err := ValidateInput("`a` $(b)\nc")
		require.Error(t, err)
		errStr := err.Error()
		assert.Contains(t, errStr, "backtick_command_substitution")
		assert.Contains(t, errStr, "subshell_command_substitution")
		assert.Contains(t, errStr, "unquoted_newline")
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		err := ValidateInput("")
		assert.NoError(t, err)
	})
}

func TestDetectDangers(t *testing.T) {
	t.Run("returns dangers for dangerous input", func(t *testing.T) {
		dangers := DetectDangers("`whoami`")
		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("returns empty slice for safe input", func(t *testing.T) {
		dangers := DetectDangers("safe input")
		assert.Len(t, dangers, 0)
	})
}

func TestIsSafe(t *testing.T) {
	t.Run("returns false for dangerous input", func(t *testing.T) {
		assert.False(t, IsSafe("`whoami`"))
	})

	t.Run("returns true for safe input", func(t *testing.T) {
		assert.True(t, IsSafe("safe input"))
	})

	t.Run("returns true for empty string", func(t *testing.T) {
		assert.True(t, IsSafe(""))
	})

	t.Run("returns true for single-quoted dangerous content", func(t *testing.T) {
		assert.True(t, IsSafe("'`whoami`'"))
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("carriage return is not detected as newline", func(t *testing.T) {
		input := "hello\rworld"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("tab is not detected as dangerous", func(t *testing.T) {
		input := "hello\tworld"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("escaped newline is still detected", func(t *testing.T) {
		input := "hello\\\nworld"
		d := NewDetector(input)
		dangers := d.Detect()

		// The backslash doesn't escape in our simple parser
		// We detect the newline after the backslash
		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
	})

	t.Run("unicode content", func(t *testing.T) {
		input := "`echo 你好世界`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("very long content", func(t *testing.T) {
		input := "`" + string(make([]byte, 10000)) + "`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("single character input", func(t *testing.T) {
		d := NewDetector("`")
		dangers := d.Detect()

		// Just a backtick without closing should still be detected
		require.Len(t, dangers, 1)
	})

	t.Run("just a dollar sign", func(t *testing.T) {
		d := NewDetector("$")
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("dollar with other chars but not paren", func(t *testing.T) {
		d := NewDetector("$VAR")
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run("nested single quotes in subshell", func(t *testing.T) {
		input := "$(echo 'it\\'s working')"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("multiple unclosed single quotes", func(t *testing.T) {
		input := "'a'`b`'c'`d`"
		d := NewDetector(input)
		dangers := d.Detect()

		// Quote state: ' toggles, so pattern is:
		// ' -> in quote, a, ' -> out, `b` -> dangerous, ' -> in, c, ' -> out, `d` -> dangerous
		require.Len(t, dangers, 2)
		assert.Equal(t, "`b`", dangers[0].Content)
		assert.Equal(t, "`d`", dangers[1].Content)
	})

	t.Run("backtick adjacent to text", func(t *testing.T) {
		input := "abc`whoami`def"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, 3, dangers[0].Start)
		assert.Equal(t, 11, dangers[0].End)
	})

	t.Run("subshell adjacent to text", func(t *testing.T) {
		input := "abc$(whoami)def"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, 3, dangers[0].Start)
		assert.Equal(t, 12, dangers[0].End)
	})

	t.Run("consecutive backticks", func(t *testing.T) {
		input := "``"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, "``", dangers[0].Content)
	})

	t.Run("consecutive backticks with content", func(t *testing.T) {
		input := "`a``b`"
		d := NewDetector(input)
		dangers := d.Detect()

		// Two separate backtick commands: `a` and `b`
		require.Len(t, dangers, 2)
		assert.Equal(t, "`a`", dangers[0].Content)
		assert.Equal(t, "`b`", dangers[1].Content)
	})

	t.Run("escaped characters don't affect parsing", func(t *testing.T) {
		input := "\\`not escaped in our parser\\`"
		d := NewDetector(input)
		dangers := d.Detect()

		// Our parser doesn't treat backslash as escape
		// The string is: \`not escaped in our parser\`
		// The backtick at position 1 starts a command that ends at the final backtick
		require.Len(t, dangers, 1)
		assert.Equal(t, "`not escaped in our parser\\`", dangers[0].Content)
	})
}

func TestRealWorldExamples(t *testing.T) {
	t.Run("complex shell command", func(t *testing.T) {
		input := "echo $(curl -s https://example.com | grep password)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("SQL injection-like pattern", func(t *testing.T) {
		input := "'; DROP TABLE users; --"
		d := NewDetector(input)
		dangers := d.Detect()

		// No shell dangers here
		assert.Len(t, dangers, 0)
	})

	t.Run("command with arguments", func(t *testing.T) {
		input := "git commit -m \"$(echo hacked)\""
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("path with backticks in filename", func(t *testing.T) {
		input := "/path/to/`malicious`/file"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("multiline script", func(t *testing.T) {
		input := "echo line1\necho line2\necho line3"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
		assert.Equal(t, DangerTypeNewline, dangers[1].Type)
	})

	t.Run("safe multiline in single quotes", func(t *testing.T) {
		input := "'echo line1\necho line2'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})

	t.Run(" heredoc-like content", func(t *testing.T) {
		input := "cat << EOF\ncontent\nEOF"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 2)
		assert.Equal(t, DangerTypeNewline, dangers[0].Type)
		assert.Equal(t, DangerTypeNewline, dangers[1].Type)
	})

	t.Run("variable assignment with command substitution", func(t *testing.T) {
		input := "USER=`whoami`"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeBacktick, dangers[0].Type)
	})

	t.Run("variable assignment with subshell", func(t *testing.T) {
		input := "USER=$(whoami)"
		d := NewDetector(input)
		dangers := d.Detect()

		require.Len(t, dangers, 1)
		assert.Equal(t, DangerTypeSubshell, dangers[0].Type)
	})

	t.Run("safe variable assignment", func(t *testing.T) {
		input := "USER='john_doe'"
		d := NewDetector(input)
		dangers := d.Detect()

		assert.Len(t, dangers, 0)
	})
}
