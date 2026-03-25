// Package security provides security-related utilities for the Cline CLI.
// It includes dangerous character detection to identify potentially harmful
// command injection patterns in user input.
package security

import (
	"fmt"
	"strings"
)

// DangerType represents the type of dangerous pattern detected.
type DangerType int

const (
	// DangerTypeBacktick indicates command substitution via backticks (`cmd`).
	DangerTypeBacktick DangerType = iota

	// DangerTypeSubshell indicates command substitution via $(...).
	DangerTypeSubshell

	// DangerTypeNewline indicates an unquoted newline character.
	DangerTypeNewline
)

// String returns a human-readable description of the danger type.
func (d DangerType) String() string {
	switch d {
	case DangerTypeBacktick:
		return "backtick_command_substitution"
	case DangerTypeSubshell:
		return "subshell_command_substitution"
	case DangerTypeNewline:
		return "unquoted_newline"
	default:
		return "unknown"
	}
}

// Danger represents a single dangerous pattern found in input.
type Danger struct {
	// Type is the category of dangerous pattern detected.
	Type DangerType

	// Start is the starting byte position of the dangerous pattern.
	Start int

	// End is the ending byte position (exclusive) of the dangerous pattern.
	End int

	// Content is the actual text of the dangerous pattern.
	Content string

	// Description provides a human-readable explanation of the danger.
	Description string
}

// String returns a formatted string representation of the Danger.
func (d Danger) String() string {
	return fmt.Sprintf("%s at [%d:%d]: %s (%s)", d.Type, d.Start, d.End, d.Content, d.Description)
}

// Detector handles dangerous character detection in shell-like input.
// It identifies command substitution patterns and unquoted newlines that
// could lead to unexpected command execution.
//
// The detector respects shell quoting rules:
//   - Single quotes (') disable all special interpretation
//   - Double quotes (") do NOT disable dangerous character detection
type Detector struct {
	// input is the string being analyzed.
	input string

	// length is the cached length of the input.
	length int
}

// NewDetector creates a new dangerous character detector for the given input.
func NewDetector(input string) *Detector {
	return &Detector{
		input:  input,
		length: len(input),
	}
}

// Detect analyzes the input and returns all dangerous patterns found.
// It processes the input character by character, tracking quote state
// to determine whether special characters should be interpreted.
func (d *Detector) Detect() []Danger {
	var dangers []Danger

	inSingleQuote := false

	for i := 0; i < d.length; i++ {
		ch := d.input[i]

		// Handle single quotes - toggle state but don't check content inside
		if ch == '\'' {
			inSingleQuote = !inSingleQuote
			continue
		}

		// If inside single quotes, skip all dangerous character checks
		if inSingleQuote {
			continue
		}

		// Check for newline (outside single quotes)
		if ch == '\n' {
			dangers = append(dangers, Danger{
				Type:        DangerTypeNewline,
				Start:       i,
				End:         i + 1,
				Content:     "\n",
				Description: "unquoted newline character detected",
			})
			continue
		}

		// Check for backtick command substitution (outside single quotes)
		// Note: double quotes do NOT protect against backticks
		if ch == '`' {
			// Find the closing backtick
			end := d.findClosingBacktick(i + 1)
			if end > i {
				content := d.input[i:end]
				dangers = append(dangers, Danger{
					Type:        DangerTypeBacktick,
					Start:       i,
					End:         end,
					Content:     content,
					Description: "backtick command substitution detected",
				})
				i = end - 1 // Skip to the end of the backtick sequence
				continue
			}
		}

		// Check for subshell $() (outside single quotes)
		// Note: double quotes do NOT protect against $()
		if ch == '$' && i+1 < d.length && d.input[i+1] == '(' {
			// Find the closing parenthesis, handling nesting
			end := d.findClosingParenthesis(i + 2)
			if end > i {
				content := d.input[i:end]
				dangers = append(dangers, Danger{
					Type:        DangerTypeSubshell,
					Start:       i,
					End:         end,
					Content:     content,
					Description: "subshell command substitution detected",
				})
				i = end - 1 // Skip to the end of the subshell
				continue
			}
		}
	}

	return dangers
}

// HasDangerousCharacters returns true if any dangerous patterns were detected.
func (d *Detector) HasDangerousCharacters() bool {
	return len(d.Detect()) > 0
}

// Count returns the number of dangerous patterns detected.
func (d *Detector) Count() int {
	return len(d.Detect())
}

// findClosingBacktick finds the position of the closing backtick starting from pos.
// Returns the position after the closing backtick, or pos if no closing backtick found.
func (d *Detector) findClosingBacktick(pos int) int {
	for i := pos; i < d.length; i++ {
		if d.input[i] == '`' {
			return i + 1
		}
	}
	// No closing backtick found - return pos to indicate incomplete pattern
	// Still report it as dangerous since the intent is clear
	return pos
}

// findClosingParenthesis finds the position of the closing parenthesis for a subshell,
// properly handling nested parentheses. Returns the position after the closing parenthesis,
// or pos if no valid closing parenthesis found.
func (d *Detector) findClosingParenthesis(pos int) int {
	depth := 1
	for i := pos; i < d.length; i++ {
		ch := d.input[i]

		// Handle nested parentheses
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				return i + 1
			}
		} else if ch == '\'' {
			// Skip over single-quoted content within subshell
			// (single quotes inside $() are still literal)
			j := i + 1
			for j < d.length && d.input[j] != '\'' {
				j++
			}
			if j < d.length {
				i = j // Continue from the closing quote
			}
		} else if ch == '\\' && i+1 < d.length {
			// Skip escaped character
			i++
		}
	}
	// No closing parenthesis found - return pos to indicate incomplete pattern
	// Still report it as dangerous since the intent is clear
	return pos
}

// ValidateInput checks if the input contains any dangerous characters and returns
// an error if any are found. This is a convenience method for simple validation.
func ValidateInput(input string) error {
	detector := NewDetector(input)
	dangers := detector.Detect()

	if len(dangers) == 0 {
		return nil
	}

	var messages []string
	for _, d := range dangers {
		messages = append(messages, d.String())
	}

	return fmt.Errorf("dangerous characters detected: %s", strings.Join(messages, "; "))
}

// DetectDangers is a convenience function that creates a detector and returns
// all dangerous patterns found in the input.
func DetectDangers(input string) []Danger {
	return NewDetector(input).Detect()
}

// IsSafe is a convenience function that returns true if the input contains
// no dangerous patterns.
func IsSafe(input string) bool {
	return !NewDetector(input).HasDangerousCharacters()
}

// ContainsDangerousCharacters is a convenience function that returns true
// if the input contains any dangerous patterns.
func ContainsDangerousCharacters(input string) bool {
	return NewDetector(input).HasDangerousCharacters()
}
