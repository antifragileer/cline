// Package security provides command permission control for the Cline CLI.
// This file implements the CommandPermissionController that integrates
// permission validation with command execution.
package security

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// PermissionValidationResult represents the result of a permission validation check.
type PermissionValidationResult struct {
	// Allowed indicates if the command is permitted
	Allowed bool `json:"allowed"`
	// MatchedPattern is the pattern that matched (for error messages)
	MatchedPattern string `json:"matched_pattern,omitempty"`
	// Reason explains why the command was allowed or denied
	Reason string `json:"reason"`
	// DetectedOperator is the shell operator that was detected (for error messages)
	DetectedOperator string `json:"detected_operator,omitempty"`
	// FailedSegment is the command segment that failed validation (for chained commands)
	FailedSegment string `json:"failed_segment,omitempty"`
}

// CommandPermissionController controls command execution permissions based on
// environment variable configuration. It uses glob pattern matching to
// allow/deny specific commands.
//
// Configuration is read from the CLINE_COMMAND_PERMISSIONS environment variable.
// Format: {"allow": ["pattern1", "pattern2"], "deny": ["pattern3"], "allowRedirects": true}
type CommandPermissionController struct {
	config *PermissionRules
	parser *Parser
}

// NewCommandPermissionController creates a new controller and parses the
// CLINE_COMMAND_PERMISSIONS environment variable.
func NewCommandPermissionController() *CommandPermissionController {
	parser := NewParser()
	config, _ := parser.ParseFromEnv()

	return &CommandPermissionController{
		config: config,
		parser: parser,
	}
}

// NewCommandPermissionControllerWithConfig creates a controller with explicit config.
func NewCommandPermissionControllerWithConfig(config *PermissionRules) *CommandPermissionController {
	return &CommandPermissionController{
		config: config,
		parser: NewParser(),
	}
}

// ValidateCommand validates if a command is allowed to execute based on
// configured permissions. For chained commands (using &&, ||, |, ;),
// each segment is validated separately.
func (c *CommandPermissionController) ValidateCommand(command string) PermissionValidationResult {
	// No config = allow everything (backward compatibility)
	if c.config == nil {
		return PermissionValidationResult{
			Allowed: true,
			Reason:  "no_config",
		}
	}

	// Check for dangerous characters first
	if dangerous := c.detectDangerousCharsOutsideQuotes(command); dangerous != nil {
		return PermissionValidationResult{
			Allowed:          false,
			Reason:           "shell_operator_detected",
			DetectedOperator: dangerous.Type.String(),
		}
	}

	// Check for redirects if not allowed
	if !c.config.AllowRedirects && c.hasRedirect(command) {
		return PermissionValidationResult{
			Allowed: false,
			Reason:  "redirect_detected",
		}
	}

	// Parse command into segments
	segments := c.parseCommandSegments(command)

	// If no segments parsed, treat as single command
	if len(segments) == 0 {
		segments = []string{command}
	}

	// Validate each segment
	isMultiSegment := len(segments) > 1
	for _, segment := range segments {
		result := c.validateSingleCommand(segment)
		if !result.Allowed {
			// Only use segment-specific reasons for multi-segment commands
			if isMultiSegment {
				return PermissionValidationResult{
					Allowed:       false,
					MatchedPattern: result.MatchedPattern,
					Reason:        c.mapSegmentReason(result.Reason),
					FailedSegment: segment,
				}
			}
			return result
		}
	}

	return PermissionValidationResult{
		Allowed: true,
		Reason:  "allowed",
	}
}

// validateSingleCommand validates a single command (no operators) against allow/deny rules.
func (c *CommandPermissionController) validateSingleCommand(command string) PermissionValidationResult {
	// Check deny rules first (deny takes precedence)
	for _, pattern := range c.config.Deny {
		if c.matchesPattern(command, pattern) {
			return PermissionValidationResult{
				Allowed:        false,
				MatchedPattern: pattern,
				Reason:         "denied",
			}
		}
	}

	// Check allow rules
	if len(c.config.Allow) > 0 {
		for _, pattern := range c.config.Allow {
			if c.matchesPattern(command, pattern) {
				return PermissionValidationResult{
					Allowed:        true,
					MatchedPattern: pattern,
					Reason:         "allowed",
				}
			}
		}
		// Allow rules defined but no match = deny by default
		return PermissionValidationResult{
			Allowed: false,
			Reason:  "no_match_deny_default",
		}
	}

	// No allow rules defined, and no deny matched = allow
	return PermissionValidationResult{
		Allowed: true,
		Reason:  "no_config",
	}
}

// mapSegmentReason maps a single command reason to a segment-specific reason.
func (c *CommandPermissionController) mapSegmentReason(reason string) string {
	switch reason {
	case "denied":
		return "segment_denied"
	case "no_match_deny_default":
		return "segment_no_match"
	default:
		return reason
	}
}

// parseCommandSegments splits a compound command into individual segments.
// It handles operators: &&, ||, ;, and | (if pipes are treated as separators).
func (c *CommandPermissionController) parseCommandSegments(command string) []string {
	// Define operators in order of precedence
	operators := []string{"&&", "||", ";", "|"}

	var segments []string
	remaining := command

	for remaining != "" {
		// Find the earliest operator
		earliestPos := -1
		earliestOp := ""

		for _, op := range operators {
			// Skip pipe operator if pipes are allowed
			if op == "|" && c.config.AllowPipes {
				continue
			}

			pos := strings.Index(remaining, op)
			if pos != -1 && (earliestPos == -1 || pos < earliestPos) {
				earliestPos = pos
				earliestOp = op
			}
		}

		if earliestPos == -1 {
			// No more operators found
			trimmed := strings.TrimSpace(remaining)
			if trimmed != "" {
				segments = append(segments, trimmed)
			}
			break
		}

		// Extract segment before operator
		segment := strings.TrimSpace(remaining[:earliestPos])
		if segment != "" {
			segments = append(segments, segment)
		}

		// Move past the operator
		remaining = strings.TrimSpace(remaining[earliestPos+len(earliestOp):])
	}

	return segments
}

// matchesPattern checks if a command matches a wildcard pattern.
//
// Uses simple wildcard matching where `*` matches any characters.
// Supported patterns:
// - `*` matches any sequence of characters
// - `?` matches exactly one character
func (c *CommandPermissionController) matchesPattern(command, pattern string) bool {
	// Convert glob pattern to regex
	regexPattern := "^" +
		regexp.QuoteMeta(pattern) +
		"$"

	// Replace glob wildcards with regex equivalents
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "\\?", ".")

	// Match with case insensitivity for command names
	matched, err := regexp.MatchString("(?i)"+regexPattern, command)
	if err != nil {
		return false
	}

	return matched
}

// detectDangerousCharsOutsideQuotes detects dangerous characters outside of quoted strings.
// This includes newlines, carriage returns, and backticks.
func (c *CommandPermissionController) detectDangerousCharsOutsideQuotes(command string) *Danger {
	detector := NewDetector(command)
	dangers := detector.Detect()

	if len(dangers) > 0 {
		return &dangers[0] // Return first danger found
	}

	return nil
}

// hasRedirect checks if a command contains output redirection operators.
func (c *CommandPermissionController) hasRedirect(command string) bool {
	// Check for >, >>, <, etc. operators
	// This is a simplified check - look for redirection patterns
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

		// Handle input redirection
		if !inQuote && ch == '<' {
			return true
		}
	}

	return false
}

// IsEnabled returns true if permission checking is enabled (config exists).
func (c *CommandPermissionController) IsEnabled() bool {
	return c.config != nil
}

// GetConfig returns the current permission configuration.
func (c *CommandPermissionController) GetConfig() *PermissionRules {
	return c.config
}

// ReloadConfig reloads the configuration from the environment variable.
func (c *CommandPermissionController) ReloadConfig() error {
	config, err := c.parser.ParseFromEnv()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	c.config = config
	return nil
}

// FormatErrorMessage formats a user-friendly error message for a denied command.
func (c *CommandPermissionController) FormatErrorMessage(result PermissionValidationResult, command string) string {
	switch result.Reason {
	case "shell_operator_detected":
		return fmt.Sprintf(
			`Command execution blocked by CLINE_COMMAND_PERMISSIONS: Detected %s. You must try a different approach or ask the user to update the permission settings.`,
			result.DetectedOperator,
		)
	case "redirect_detected":
		return `Command execution blocked by CLINE_COMMAND_PERMISSIONS: Redirect operators (>, >>, <) are not allowed. You must try a different approach or ask the user to update the permission settings.`
	case "denied", "segment_denied":
		if result.MatchedPattern != "" {
			return fmt.Sprintf(
				`Command "%s" was denied by CLINE_COMMAND_PERMISSIONS. Segment "%s" matched deny pattern "%s".`,
				command, result.FailedSegment, result.MatchedPattern,
			)
		}
		return fmt.Sprintf(
			`Command "%s" was denied by CLINE_COMMAND_PERMISSIONS. Segment "%s" was explicitly denied.`,
			command, result.FailedSegment,
		)
	case "no_match_deny_default", "segment_no_match":
		if result.MatchedPattern != "" {
			return fmt.Sprintf(
				`Command "%s" was denied by CLINE_COMMAND_PERMISSIONS. Segment "%s" did not match any allow pattern.`,
				command, result.FailedSegment,
			)
		}
		return fmt.Sprintf(
			`Command "%s" was denied by CLINE_COMMAND_PERMISSIONS. Reason: %s`,
			command, result.Reason,
		)
	default:
		return fmt.Sprintf(
			`Command "%s" was denied by CLINE_COMMAND_PERMISSIONS. Reason: %s`,
			command, result.Reason,
		)
	}
}

// Global instance for convenience
var defaultController *CommandPermissionController

// init initializes the default controller.
func init() {
	defaultController = NewCommandPermissionController()
}

// IsPermissionControlEnabled checks if the default controller is enabled.
func IsPermissionControlEnabled() bool {
	return defaultController.IsEnabled()
}

// CheckAndLoadEnv loads permissions from environment and returns a controller.
func CheckAndLoadEnv() (*CommandPermissionController, error) {
	jsonData := os.Getenv(EnvVarName)
	if jsonData == "" {
		return nil, nil // No config set
	}

	parser := NewParser()
	config, err := parser.ParseString(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", EnvVarName, err)
	}

	return NewCommandPermissionControllerWithConfig(config), nil
}
