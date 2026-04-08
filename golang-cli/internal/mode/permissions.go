// Package mode provides command permission validation for the Cline CLI.
// It implements allow/deny lists with glob matching, compound command validation,
// and configurable redirect handling.
package mode

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CommandPermissions defines the structure for command permission configuration.
// It supports allow/deny lists with glob patterns and redirect control.
type CommandPermissions struct {
	// Allow is a list of glob patterns for allowed commands
	Allow []string `json:"allow"`

	// Deny is a list of glob patterns for denied commands (takes precedence over allow)
	Deny []string `json:"deny"`

	// AllowRedirects controls whether command output redirects (> or >>) are permitted
	AllowRedirects bool `json:"allowRedirects"`

	// AllowPipes controls whether command pipes (|) are permitted
	AllowPipes bool `json:"allowPipes"`
}

// ValidationError represents a detailed error from command validation.
// It provides context about why a command was rejected.
type ValidationError struct {
	// Command is the command that failed validation
	Command string

	// Segment is the specific segment of a compound command that failed
	Segment string

	// Reason describes why the command was rejected
	Reason string

	// ViolationType categorizes the type of violation
	ViolationType ViolationType
}

// ViolationType categorizes the type of permission violation.
type ViolationType string

const (
	// ViolationTypeDenied indicates the command matched a deny pattern
	ViolationTypeDenied ViolationType = "denied"

	// ViolationTypeNotAllowed indicates the command didn't match any allow pattern
	ViolationTypeNotAllowed ViolationType = "not_allowed"

	// ViolationTypeRedirectNotAllowed indicates redirects are not permitted
	ViolationTypeRedirectNotAllowed ViolationType = "redirect_not_allowed"

	// ViolationTypePipeNotAllowed indicates pipes are not permitted
	ViolationTypePipeNotAllowed ViolationType = "pipe_not_allowed"

	// ViolationTypeInvalidCommand indicates the command is malformed
	ViolationTypeInvalidCommand ViolationType = "invalid_command"

	// ViolationTypeParseError indicates the compound command could not be parsed
	ViolationTypeParseError ViolationType = "parse_error"
)

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Segment != "" && e.Segment != e.Command {
		return fmt.Sprintf("command validation failed: segment %q (%s): %s", e.Segment, e.ViolationType, e.Reason)
	}
	return fmt.Sprintf("command validation failed: %q (%s): %s", e.Command, e.ViolationType, e.Reason)
}

// PermissionValidator validates commands against configured permissions.
// It supports glob pattern matching and compound command validation.
type PermissionValidator struct {
	permissions *CommandPermissions

	// compiledAllow contains compiled glob patterns for allowed commands
	compiledAllow []*globPattern

	// compiledDeny contains compiled glob patterns for denied commands
	compiledDeny []*globPattern

	// segmentParsers defines the operators that split compound commands
	segmentParsers []segmentParser
}

// globPattern represents a compiled glob pattern for matching.
type globPattern struct {
	// original is the original pattern string
	original string

	// regex is the compiled regular expression
	regex *regexp.Regexp

	// isExact indicates if the pattern is an exact match (no wildcards)
	isExact bool
}

// segmentParser defines how to split compound commands.
type segmentParser struct {
	// operator is the string that separates command segments
	operator string

	// trimTrailing indicates if the operator should be trimmed from the end
	trimTrailing bool
}

// DefaultSegmentParsers defines the standard command separators.
// These are used to split compound commands into individual segments.
var DefaultSegmentParsers = []segmentParser{
	{operator: "&&", trimTrailing: true},
	{operator: "||", trimTrailing: true},
	{operator: ";", trimTrailing: true},
	{operator: "|", trimTrailing: true},
}

// NewPermissionValidator creates a new validator with the given permissions.
// It compiles all glob patterns for efficient matching.
func NewPermissionValidator(permissions *CommandPermissions) (*PermissionValidator, error) {
	if permissions == nil {
		return nil, fmt.Errorf("permissions cannot be nil")
	}

	v := &PermissionValidator{
		permissions:    permissions,
		segmentParsers: DefaultSegmentParsers,
	}

	// Compile allow patterns
	for _, pattern := range permissions.Allow {
		compiled, err := compileGlobPattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid allow pattern %q: %w", pattern, err)
		}
		v.compiledAllow = append(v.compiledAllow, compiled)
	}

	// Compile deny patterns
	for _, pattern := range permissions.Deny {
		compiled, err := compileGlobPattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid deny pattern %q: %w", pattern, err)
		}
		v.compiledDeny = append(v.compiledDeny, compiled)
	}

	return v, nil
}

// LoadPermissionsFromEnv loads command permissions from the CLINE_COMMAND_PERMISSIONS
// environment variable. Returns nil if the variable is not set.
func LoadPermissionsFromEnv() (*CommandPermissions, error) {
	jsonData := os.Getenv("CLINE_COMMAND_PERMISSIONS")
	if jsonData == "" {
		return nil, nil
	}

	var permissions CommandPermissions
	if err := json.Unmarshal([]byte(jsonData), &permissions); err != nil {
		return nil, fmt.Errorf("failed to parse CLINE_COMMAND_PERMISSIONS: %w", err)
	}

	return &permissions, nil
}

// MustLoadPermissionsFromEnv loads permissions from environment or returns defaults.
// If parsing fails, it returns an error rather than panicking.
func MustLoadPermissionsFromEnv() (*CommandPermissions, error) {
	perms, err := LoadPermissionsFromEnv()
	if err != nil {
		return nil, err
	}
	if perms == nil {
		// Return permissive defaults when no permissions are configured
		return &CommandPermissions{
			Allow:          []string{"*"},
			Deny:           []string{},
			AllowRedirects: true,
			AllowPipes:     true,
		}, nil
	}
	return perms, nil
}

// ValidateCommand validates a complete command string against permissions.
// It handles compound commands by validating each segment separately.
func (v *PermissionValidator) ValidateCommand(command string) error {
	if command == "" {
		return &ValidationError{
			Command:       command,
			Reason:        "command cannot be empty",
			ViolationType: ViolationTypeInvalidCommand,
		}
	}

	// First, check for redirects if not allowed
	if !v.permissions.AllowRedirects {
		if hasRedirect(command) {
			return &ValidationError{
				Command:       command,
				Reason:        "output redirects (> or >>) are not permitted",
				ViolationType: ViolationTypeRedirectNotAllowed,
			}
		}
	}

	// Parse compound command into segments
	segments := v.parseCommandSegments(command)

	// Validate each segment
	for _, segment := range segments {
		if err := v.validateSegment(segment, command); err != nil {
			return err
		}
	}

	return nil
}

// ValidateCommandWithDetails validates a command and returns detailed results
// for each segment, useful for debugging or logging.
func (v *PermissionValidator) ValidateCommandWithDetails(command string) *ValidationResult {
	result := &ValidationResult{
		Command:  command,
		Segments: []SegmentValidation{},
		Allowed:  true,
	}

	if command == "" {
		result.Allowed = false
		result.Error = &ValidationError{
			Command:       command,
			Reason:        "command cannot be empty",
			ViolationType: ViolationTypeInvalidCommand,
		}
		return result
	}

	// Check for redirects
	if !v.permissions.AllowRedirects && hasRedirect(command) {
		result.Allowed = false
		result.Error = &ValidationError{
			Command:       command,
			Reason:        "output redirects (> or >>) are not permitted",
			ViolationType: ViolationTypeRedirectNotAllowed,
		}
		return result
	}

	// Parse and validate each segment
	segments := v.parseCommandSegments(command)
	for _, segment := range segments {
		segResult := v.validateSegmentWithDetails(segment)
		result.Segments = append(result.Segments, segResult)
		if !segResult.Allowed {
			result.Allowed = false
			if result.Error == nil {
				result.Error = segResult.Error
			}
		}
	}

	return result
}

// ValidationResult contains the detailed results of command validation.
type ValidationResult struct {
	// Command is the original command string
	Command string

	// Segments contains validation results for each command segment
	Segments []SegmentValidation

	// Allowed indicates if the entire command is allowed
	Allowed bool

	// Error contains the first validation error if validation failed
	Error *ValidationError
}

// SegmentValidation contains validation results for a single command segment.
type SegmentValidation struct {
	// Segment is the command segment text
	Segment string

	// Allowed indicates if this segment is allowed
	Allowed bool

	// MatchedAllowPattern is the allow pattern that matched (if any)
	MatchedAllowPattern string

	// MatchedDenyPattern is the deny pattern that matched (if any)
	MatchedDenyPattern string

	// Error contains the validation error if the segment was rejected
	Error *ValidationError
}

// validateSegment validates a single command segment.
func (v *PermissionValidator) validateSegment(segment, fullCommand string) error {
	// Extract the base command (first word, before any arguments)
	baseCmd := extractBaseCommand(segment)

	// Check deny patterns first (deny takes precedence)
	for _, pattern := range v.compiledDeny {
		if pattern.matches(baseCmd) {
			return &ValidationError{
				Command:       fullCommand,
				Segment:       segment,
				Reason:        fmt.Sprintf("command %q matches denied pattern %q", baseCmd, pattern.original),
				ViolationType: ViolationTypeDenied,
			}
		}
	}

	// Check allow patterns
	for _, pattern := range v.compiledAllow {
		if pattern.matches(baseCmd) {
			return nil // Allowed
		}
	}

	// Not in allow list
	return &ValidationError{
		Command:       fullCommand,
		Segment:       segment,
		Reason:        fmt.Sprintf("command %q does not match any allowed pattern", baseCmd),
		ViolationType: ViolationTypeNotAllowed,
	}
}

// validateSegmentWithDetails validates a segment and returns detailed results.
func (v *PermissionValidator) validateSegmentWithDetails(segment string) SegmentValidation {
	result := SegmentValidation{
		Segment: segment,
		Allowed: false,
	}

	baseCmd := extractBaseCommand(segment)

	// Check deny patterns first
	for _, pattern := range v.compiledDeny {
		if pattern.matches(baseCmd) {
			result.MatchedDenyPattern = pattern.original
			result.Error = &ValidationError{
				Command:       segment,
				Segment:       segment,
				Reason:        fmt.Sprintf("command %q matches denied pattern %q", baseCmd, pattern.original),
				ViolationType: ViolationTypeDenied,
			}
			return result
		}
	}

	// Check allow patterns
	for _, pattern := range v.compiledAllow {
		if pattern.matches(baseCmd) {
			result.Allowed = true
			result.MatchedAllowPattern = pattern.original
			return result
		}
	}

	// Not allowed
	result.Error = &ValidationError{
		Command:       segment,
		Segment:       segment,
		Reason:        fmt.Sprintf("command %q does not match any allowed pattern", baseCmd),
		ViolationType: ViolationTypeNotAllowed,
	}
	return result
}

// parseCommandSegments splits a compound command into individual segments.
// It handles operators: &&, ||, ;, and | (if pipes are treated as separators).
func (v *PermissionValidator) parseCommandSegments(command string) []string {
	var segments []string
	current := command

	for current != "" {
		// Find the earliest operator
		earliestPos := -1
		earliestOp := ""

		for _, parser := range v.segmentParsers {
			// Skip pipe operator if pipes are allowed (treat as part of command)
			if parser.operator == "|" && v.permissions.AllowPipes {
				continue
			}

			pos := strings.Index(current, parser.operator)
			if pos != -1 && (earliestPos == -1 || pos < earliestPos) {
				earliestPos = pos
				earliestOp = parser.operator
			}
		}

		if earliestPos == -1 {
			// No more operators found
			trimmed := strings.TrimSpace(current)
			if trimmed != "" {
				segments = append(segments, trimmed)
			}
			break
		}

		// Extract segment before operator
		segment := strings.TrimSpace(current[:earliestPos])
		if segment != "" {
			segments = append(segments, segment)
		}

		// Move past the operator
		current = strings.TrimSpace(current[earliestPos+len(earliestOp):])
	}

	return segments
}

// extractBaseCommand extracts the base command from a command string.
// It handles commands with paths (e.g., /bin/ls -> ls) and arguments.
func extractBaseCommand(command string) string {
	// Trim whitespace
	command = strings.TrimSpace(command)

	// Find the first space to separate command from arguments
	if idx := strings.IndexAny(command, " \t"); idx != -1 {
		command = command[:idx]
	}

	// Extract just the command name from a path
	if idx := strings.LastIndex(command, "/"); idx != -1 {
		command = command[idx+1:]
	}

	// Handle Windows-style paths
	if idx := strings.LastIndex(command, "\\"); idx != -1 {
		command = command[idx+1:]
	}

	return command
}

// hasRedirect checks if a command contains output redirection operators.
func hasRedirect(command string) bool {
	// Check for > and >> operators (but not as part of strings)
	// This is a simplified check - a full implementation would need proper shell parsing
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range command {
		// Handle quotes
		if ch == '"' || ch == '\'' {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
				quoteChar = 0
			}
			continue
		}

		if !inQuote && ch == '>' {
			// Check for >> (append) or > (overwrite)
			return true
		}

		// Handle 2>, &>, etc.
		if !inQuote && i > 0 && ch == '>' {
			prev := command[i-1]
			if prev == '2' || prev == '&' || prev == '1' {
				return true
			}
		}
	}

	return false
}

// compileGlobPattern compiles a glob pattern into a regex pattern.
// Supports: * (match any chars), ? (match single char), ** (match across dirs)
func compileGlobPattern(pattern string) (*globPattern, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	// Check if pattern is exact (no wildcards)
	isExact := !strings.ContainsAny(pattern, "*?[]")

	// Convert glob to regex
	var regexStr strings.Builder
	regexStr.WriteString("^")

	i := 0
	for i < len(pattern) {
		ch := pattern[i]

		switch ch {
		case '*':
			// Check for ** (match across directory boundaries)
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				regexStr.WriteString(".*")
				i += 2
			} else {
				// Single * - match any except path separator
				regexStr.WriteString("[^/]*")
				i++
			}
		case '?':
			regexStr.WriteString(".")
			i++
		case '.':
			regexStr.WriteString(`\.`)
			i++
		case '+', '(', ')', '^', '$', '[', ']', '{', '}', '|', '\\':
			// Escape regex special chars
			regexStr.WriteString(`\`)
			regexStr.WriteByte(ch)
			i++
		default:
			regexStr.WriteByte(ch)
			i++
		}
	}

	regexStr.WriteString("$")

	regex, err := regexp.Compile(regexStr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern: %w", err)
	}

	return &globPattern{
		original: pattern,
		regex:    regex,
		isExact:  isExact,
	}, nil
}

// matches checks if a string matches the glob pattern.
func (p *globPattern) matches(s string) bool {
	return p.regex.MatchString(s)
}

// IsAllowed checks if a command would be allowed without returning detailed errors.
// This is a convenience method for simple permission checks.
func (v *PermissionValidator) IsAllowed(command string) bool {
	return v.ValidateCommand(command) == nil
}

// GetPermissions returns the current permission configuration.
func (v *PermissionValidator) GetPermissions() *CommandPermissions {
	return v.permissions
}

// SetPermissions updates the validator with new permissions.
// This recompiles all patterns.
func (v *PermissionValidator) SetPermissions(permissions *CommandPermissions) error {
	newValidator, err := NewPermissionValidator(permissions)
	if err != nil {
		return err
	}

	v.permissions = newValidator.permissions
	v.compiledAllow = newValidator.compiledAllow
	v.compiledDeny = newValidator.compiledDeny
	return nil
}

// DefaultDenyValidator creates a validator that denies all commands.
// Useful for creating restrictive default configurations.
func DefaultDenyValidator() *PermissionValidator {
	v, _ := NewPermissionValidator(&CommandPermissions{
		Allow:          []string{},
		Deny:           []string{"*"},
		AllowRedirects: false,
		AllowPipes:     false,
	})
	return v
}

// DefaultAllowValidator creates a validator that allows all commands.
// Useful for permissive configurations during development.
func DefaultAllowValidator() *PermissionValidator {
	v, _ := NewPermissionValidator(&CommandPermissions{
		Allow:          []string{"*"},
		Deny:           []string{},
		AllowRedirects: true,
		AllowPipes:     true,
	})
	return v
}
