// Package security provides command validation and security enforcement
// for the Cline CLI.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EnvVarName is the environment variable name for command permissions
const EnvVarName = "CLINE_COMMAND_PERMISSIONS"

// LoadFromEnv loads permission rules from the CLINE_COMMAND_PERMISSIONS
// environment variable.
//
// The environment variable should contain a JSON object with "allow" and "deny"
// arrays of glob patterns:
//
//	CLINE_COMMAND_PERMISSIONS='{"allow":["git *","npm *"],"deny":["rm -rf *","sudo *"]}'
//
// Returns nil, nil if the environment variable is not set.
func LoadFromEnv() (*CommandValidator, error) {
	envValue := os.Getenv(EnvVarName)
	if envValue == "" {
		return nil, nil
	}

	return LoadFromString(envValue)
}

// LoadFromString loads permission rules from a JSON string.
//
// The JSON should be an object with "allow" and "deny" arrays:
//
//	{
//	  "allow": ["git *", "npm *", "cargo *"],
//	  "deny": ["rm -rf *", "sudo *", "mkfs.*"]
//	}
func LoadFromString(jsonStr string) (*CommandValidator, error) {
	var permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	}

	// Trim whitespace and quotes
	jsonStr = strings.TrimSpace(jsonStr)
	jsonStr = strings.Trim(jsonStr, "'\"")

	if err := json.Unmarshal([]byte(jsonStr), &permissions); err != nil {
		return nil, fmt.Errorf("invalid %s format: %w", EnvVarName, err)
	}

	// Create validator with default options
	opts := DefaultValidatorOptions()
	opts.EnableLogging = true
	opts.LogPath = "" // Use stderr

	validator, err := NewCommandValidator(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	// Add allow rules
	for _, pattern := range permissions.Allow {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			if err := validator.AddAllowRule(pattern, fmt.Sprintf("from %s", EnvVarName)); err != nil {
				return nil, fmt.Errorf("failed to add allow rule %q: %w", pattern, err)
			}
		}
	}

	// Add deny rules
	for _, pattern := range permissions.Deny {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			if err := validator.AddDenyRule(pattern, fmt.Sprintf("from %s", EnvVarName)); err != nil {
				return nil, fmt.Errorf("failed to add deny rule %q: %w", pattern, err)
			}
		}
	}

	return validator, nil
}

// ValidateCommand validates a single command against environment permissions.
// Returns true if allowed, false if denied or if validation fails.
func ValidateCommand(command string) (bool, string) {
	validator, err := LoadFromEnv()
	if err != nil {
		return false, fmt.Sprintf("failed to load permissions: %v", err)
	}

	if validator == nil {
		// No permissions set, allow by default
		return true, "no permissions configured"
	}

	defer validator.Close()

	result := validator.Validate(nil, command)
	return result.Allowed, result.Reason
}

// MustValidateCommand validates a command and returns an error if denied.
func MustValidateCommand(command string) error {
	allowed, reason := ValidateCommand(command)
	if !allowed {
		return fmt.Errorf("command not allowed: %s", reason)
	}
	return nil
}
