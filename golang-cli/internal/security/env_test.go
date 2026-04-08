package security

import (
	"os"
	"testing"
)

func TestLoadFromEnv_NotSet(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv(EnvVarName)

	validator, err := LoadFromEnv()
	if err != nil {
		t.Errorf("Expected no error when env not set, got: %v", err)
	}
	if validator != nil {
		t.Error("Expected nil validator when env not set")
	}
}

func TestLoadFromEnv_ValidConfig(t *testing.T) {
	// Set valid config
	os.Setenv(EnvVarName, `{"allow":["git *","npm *"],"deny":["rm -rf *","sudo *"]}`)
	defer os.Unsetenv(EnvVarName)

	validator, err := LoadFromEnv()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()

	// Test that rules were loaded
	result := validator.Validate(nil, "git status")
	if !result.Allowed {
		t.Error("Expected 'git status' to be allowed")
	}

	result = validator.Validate(nil, "rm -rf /")
	if result.Allowed {
		t.Error("Expected 'rm -rf /' to be denied")
	}
}

func TestLoadFromEnv_InvalidJSON(t *testing.T) {
	// Set invalid config
	os.Setenv(EnvVarName, `not valid json`)
	defer os.Unsetenv(EnvVarName)

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadFromEnv_EmptyAllowDeny(t *testing.T) {
	// Set config with empty arrays
	os.Setenv(EnvVarName, `{"allow":[],"deny":[]}`)
	defer os.Unsetenv(EnvVarName)

	validator, err := LoadFromEnv()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()
}

func TestLoadFromEnv_QuotedString(t *testing.T) {
	// Set config with surrounding quotes (should be trimmed)
	os.Setenv(EnvVarName, `'{"allow":["git *"],"deny":[]}'`)
	defer os.Unsetenv(EnvVarName)

	validator, err := LoadFromEnv()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()
}

func TestLoadFromString_Valid(t *testing.T) {
	jsonStr := `{"allow":["git *","npm *"],"deny":["rm -rf *"]}`

	validator, err := LoadFromString(jsonStr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()

	// Test allowed command
	result := validator.Validate(nil, "git status")
	if !result.Allowed {
		t.Error("Expected 'git status' to be allowed")
	}

	// Test denied command
	result = validator.Validate(nil, "rm -rf /home")
	if result.Allowed {
		t.Error("Expected 'rm -rf /home' to be denied")
	}
}

func TestLoadFromString_InvalidJSON(t *testing.T) {
	jsonStr := `not valid json`

	_, err := LoadFromString(jsonStr)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadFromString_Empty(t *testing.T) {
	jsonStr := ``

	_, err := LoadFromString(jsonStr)
	if err == nil {
		t.Error("Expected error for empty string")
	}
}

func TestLoadFromString_Whitespace(t *testing.T) {
	jsonStr := `   {"allow":["git *"],"deny":[]}   `

	validator, err := LoadFromString(jsonStr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()
}

func TestLoadFromString_InvalidPattern(t *testing.T) {
	// The glob library accepts most patterns including unusual characters
	// Just verify it doesn't panic and processes the pattern
	jsonStr := `{"allow":["[invalid"],"deny":[]}`

	validator, err := LoadFromString(jsonStr)
	// The validator may or may not return an error depending on glob implementation
	// Just ensure it doesn't panic
	_ = err
	_ = validator
	if validator != nil {
		validator.Close()
	}
}

func TestValidateCommand_Allowed(t *testing.T) {
	os.Setenv(EnvVarName, `{"allow":["git *"],"deny":[]}`)
	defer os.Unsetenv(EnvVarName)

	allowed, reason := ValidateCommand("git status")
	if !allowed {
		t.Errorf("Expected command to be allowed, reason: %s", reason)
	}
}

func TestValidateCommand_Denied(t *testing.T) {
	os.Setenv(EnvVarName, `{"allow":["git *"],"deny":["rm -rf *"]}`)
	defer os.Unsetenv(EnvVarName)

	allowed, reason := ValidateCommand("rm -rf /")
	if allowed {
		t.Error("Expected command to be denied")
	}
	if reason == "" {
		t.Error("Expected non-empty reason for denied command")
	}
}

func TestValidateCommand_NoEnv(t *testing.T) {
	os.Unsetenv(EnvVarName)

	allowed, reason := ValidateCommand("any command")
	if !allowed {
		t.Error("Expected command to be allowed when no env set")
	}
	if reason != "no permissions configured" {
		t.Errorf("Expected 'no permissions configured', got: %s", reason)
	}
}

func TestValidateCommand_InvalidEnv(t *testing.T) {
	os.Setenv(EnvVarName, `invalid json`)
	defer os.Unsetenv(EnvVarName)

	allowed, reason := ValidateCommand("any command")
	if allowed {
		t.Error("Expected command to be denied when env is invalid")
	}
	if reason == "" {
		t.Error("Expected error reason")
	}
}

func TestMustValidateCommand_Success(t *testing.T) {
	os.Setenv(EnvVarName, `{"allow":["git *"],"deny":[]}`)
	defer os.Unsetenv(EnvVarName)

	err := MustValidateCommand("git status")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMustValidateCommand_Failure(t *testing.T) {
	os.Setenv(EnvVarName, `{"allow":[],"deny":["*"]}`)
	defer os.Unsetenv(EnvVarName)

	err := MustValidateCommand("any command")
	if err == nil {
		t.Error("Expected error for denied command")
	}
}

func TestMustValidateCommand_NoEnv(t *testing.T) {
	os.Unsetenv(EnvVarName)

	err := MustValidateCommand("any command")
	if err != nil {
		t.Errorf("Unexpected error when no env set: %v", err)
	}
}

func TestLoadFromEnv_MultipleRules(t *testing.T) {
	os.Setenv(EnvVarName, `{
		"allow": [
			"git *",
			"npm *",
			"yarn *",
			"cargo *",
			"go *"
		],
		"deny": [
			"rm -rf *",
			"sudo *",
			"mkfs.*",
			"dd *",
			"> *"
		]
	}`)
	defer os.Unsetenv(EnvVarName)

	validator, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}
	defer validator.Close()

	// Test various commands
	tests := []struct {
		command string
		allowed bool
	}{
		{"git status", true},
		{"git push origin main", true},
		{"npm install", true},
		{"cargo build", true},
		{"go test ./...", true},
		{"rm -rf /", false},
		{"sudo apt-get update", false},
		{"dd if=/dev/zero of=/dev/sda", false},
		{"echo hello > /etc/passwd", false},
	}

	for _, tt := range tests {
		result := validator.Validate(nil, tt.command)
		if result.Allowed != tt.allowed {
			t.Errorf("Command %q: expected allowed=%v, got %v (reason: %s)",
				tt.command, tt.allowed, result.Allowed, result.Reason)
		}
	}
}
