package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "API key masking",
			key:      "apiKey",
			value:    "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "Secret masking",
			key:      "secretToken",
			value:    "my-secret-value",
			expected: "my-s****alue",
		},
		{
			name:     "Short value masking",
			key:      "apiKey",
			value:    "short",
			expected: "****",
		},
		{
			name:     "Non-sensitive key",
			key:      "username",
			value:    "john_doe",
			expected: "john_doe",
		},
		{
			name:     "Password masking",
			key:      "password",
			value:    "supersecretpassword123",
			expected: "supe****d123",
		},
		{
			name:     "Token masking",
			key:      "authToken",
			value:    "bearer_token_xyz",
			expected: "bear****_xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("maskSensitiveValue(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"test", "", true},
		{"test", "test", true},
		{"test", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// Test global config path
	globalPath, err := GetConfigPath(true)
	if err != nil {
		t.Errorf("GetConfigPath(true) returned error: %v", err)
	}
	expectedGlobal := filepath.Join(homeDir, ".cline", "data", "globalState.json")
	if globalPath != expectedGlobal {
		t.Errorf("GetConfigPath(true) = %q, want %q", globalPath, expectedGlobal)
	}

	// Test workspace config path
	wsPath, err := GetConfigPath(false)
	if err != nil {
		t.Errorf("GetConfigPath(false) returned error: %v", err)
	}
	// Should contain the workspaces directory
	if !strings.Contains(wsPath, "workspaces") {
		t.Errorf("GetConfigPath(false) should contain 'workspaces', got: %s", wsPath)
	}
}

func TestPrintConfigSectionFormatted(t *testing.T) {
	data := map[string]interface{}{
		"apiProvider": "anthropic",
		"nested": map[string]interface{}{
			"key": "value",
		},
		"list": []interface{}{"a", "b", "c"},
	}

	// Just verify it doesn't panic - function writes to stdout
	printConfigSectionFormatted(data, "")
}

func TestConfigJSONOutput(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cline-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test storage files
	globalStatePath := filepath.Join(tempDir, "globalState.json")
	workspaceDir := filepath.Join(tempDir, "workspaces", "test")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}
	workspaceStatePath := filepath.Join(workspaceDir, "workspaceState.json")

	// Write test data
	globalData := map[string]interface{}{
		"apiProvider":  "anthropic",
		"defaultModel": "claude-3-5-sonnet",
	}
	globalJSON, _ := json.Marshal(globalData)
	if err := os.WriteFile(globalStatePath, globalJSON, 0644); err != nil {
		t.Fatalf("Failed to write global state: %v", err)
	}

	workspaceData := map[string]interface{}{
		"workspaceSetting": "test-value",
	}
	workspaceJSON, _ := json.Marshal(workspaceData)
	if err := os.WriteFile(workspaceStatePath, workspaceJSON, 0644); err != nil {
		t.Fatalf("Failed to write workspace state: %v", err)
	}

	// Test would require mocking storage context, so we just verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(globalJSON, &result); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if result["apiProvider"] != "anthropic" {
		t.Errorf("Expected apiProvider to be 'anthropic', got %v", result["apiProvider"])
	}
}
