package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestIsValidModel(t *testing.T) {
	provider := SupportedProviders["anthropic"]
	
	tests := []struct {
		model    string
		expected bool
	}{
		{"claude-3-5-sonnet-20241022", true},
		{"claude-3-opus-20240229", true},
		{"invalid-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := isValidModel(provider, tt.model)
			if result != tt.expected {
				t.Errorf("isValidModel(%q) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestSupportedProviders(t *testing.T) {
	// Test that all providers have required fields
	for name, provider := range SupportedProviders {
		t.Run(name, func(t *testing.T) {
			if provider.Name == "" {
				t.Errorf("Provider %s has no name", name)
			}
			if provider.Description == "" {
				t.Errorf("Provider %s has no description", name)
			}
			if provider.DefaultModel == "" {
				t.Errorf("Provider %s has no default model", name)
			}
			if len(provider.Models) == 0 {
				t.Errorf("Provider %s has no models", name)
			}
		})
	}
}

func TestSelectProviderInteractive(t *testing.T) {
	// Test with valid input
	input := bytes.NewBufferString("1\n")
	output := &bytes.Buffer{}

	provider, err := selectProviderInteractive(input, output)
	if err != nil {
		t.Errorf("selectProviderInteractive returned error: %v", err)
	}

	// Should return the first provider
	found := false
	for name := range SupportedProviders {
		if provider == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected valid provider, got: %s", provider)
	}

	// Test with invalid input
	input = bytes.NewBufferString("invalid\n")
	output = &bytes.Buffer{}

	_, err = selectProviderInteractive(input, output)
	if err == nil {
		t.Error("Expected error for invalid input")
	}

	// Test with out of range input
	input = bytes.NewBufferString("999\n")
	output = &bytes.Buffer{}

	_, err = selectProviderInteractive(input, output)
	if err == nil {
		t.Error("Expected error for out of range input")
	}
}

func TestPromptForKey(t *testing.T) {
	// Test with valid input
	input := bytes.NewBufferString("sk-test123\n")
	output := &bytes.Buffer{}

	key, err := promptForKey(input, output, "openai")
	if err != nil {
		t.Errorf("promptForKey returned error: %v", err)
	}
	if key != "sk-test123" {
		t.Errorf("Expected key 'sk-test123', got: %s", key)
	}

	// Test with empty input
	input = bytes.NewBufferString("\n")
	output = &bytes.Buffer{}

	_, err = promptForKey(input, output, "openai")
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestSelectModelInteractive(t *testing.T) {
	provider := SupportedProviders["anthropic"]

	// Test with number input
	input := bytes.NewBufferString("1\n")
	output := &bytes.Buffer{}

	model, err := selectModelInteractive(input, output, provider)
	if err != nil {
		t.Errorf("selectModelInteractive returned error: %v", err)
	}
	if model != provider.Models[0] {
		t.Errorf("Expected model %s, got: %s", provider.Models[0], model)
	}

	// Test with model name input
	input = bytes.NewBufferString("claude-3-opus-20240229\n")
	output = &bytes.Buffer{}

	model, err = selectModelInteractive(input, output, provider)
	if err != nil {
		t.Errorf("selectModelInteractive returned error: %v", err)
	}
	if model != "claude-3-opus-20240229" {
		t.Errorf("Expected model 'claude-3-opus-20240229', got: %s", model)
	}

	// Test with empty input (should use default)
	input = bytes.NewBufferString("\n")
	output = &bytes.Buffer{}

	model, err = selectModelInteractive(input, output, provider)
	if err != nil {
		t.Errorf("selectModelInteractive returned error: %v", err)
	}
	if model != provider.DefaultModel {
		t.Errorf("Expected default model %s, got: %s", provider.DefaultModel, model)
	}
}

func TestTestProviderAuth(t *testing.T) {
	// Test with each supported provider
	for providerName := range SupportedProviders {
		t.Run(providerName, func(t *testing.T) {
			err := testProviderAuth(providerName, "test-key")
			if err != nil {
				t.Errorf("testProviderAuth(%s) returned error: %v", providerName, err)
			}
		})
	}

	// Test with unknown provider
	err := testProviderAuth("unknown", "test-key")
	if err == nil {
		t.Error("Expected error for unknown provider")
	}
}

func TestProviderInfoStructure(t *testing.T) {
	// Test that ollama doesn't require key
	ollama := SupportedProviders["ollama"]
	if ollama.RequiresKey {
		t.Error("Ollama should not require API key")
	}

	// Test that other providers require key
	anthropic := SupportedProviders["anthropic"]
	if !anthropic.RequiresKey {
		t.Error("Anthropic should require API key")
	}
}

func TestSelectProviderInteractiveOutput(t *testing.T) {
	input := bytes.NewBufferString("1\n")
	output := &bytes.Buffer{}

	_, err := selectProviderInteractive(input, output)
	if err != nil {
		t.Fatalf("selectProviderInteractive returned error: %v", err)
	}

	outputStr := output.String()
	
	// Check that output contains expected elements
	if !strings.Contains(outputStr, "Select an AI provider") {
		t.Error("Output should contain 'Select an AI provider'")
	}

	// Check that all providers are listed
	for name, info := range SupportedProviders {
		if !strings.Contains(outputStr, info.Name) {
			t.Errorf("Output should contain provider name %s", name)
		}
		if !strings.Contains(outputStr, info.Description) {
			t.Errorf("Output should contain provider description for %s", name)
		}
	}
}

func TestSelectModelInteractiveOutput(t *testing.T) {
	provider := SupportedProviders["openai"]
	input := bytes.NewBufferString("\n") // Empty input to select default
	output := &bytes.Buffer{}

	model, err := selectModelInteractive(input, output, provider)
	if err != nil {
		t.Fatalf("selectModelInteractive returned error: %v", err)
	}

	if model != provider.DefaultModel {
		t.Errorf("Expected default model %s, got %s", provider.DefaultModel, model)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Select a model") {
		t.Error("Output should contain 'Select a model'")
	}

	// Check that all models are listed
	for _, m := range provider.Models {
		if !strings.Contains(outputStr, m) {
			t.Errorf("Output should contain model %s", m)
		}
	}
}

func TestPromptForKeyOutput(t *testing.T) {
	input := bytes.NewBufferString("test-api-key\n")
	output := &bytes.Buffer{}

	key, err := promptForKey(input, output, "anthropic")
	if err != nil {
		t.Fatalf("promptForKey returned error: %v", err)
	}

	if key != "test-api-key" {
		t.Errorf("Expected key 'test-api-key', got %s", key)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "anthropic API key") {
		t.Error("Output should prompt for anthropic API key")
	}
}

func TestReadInputErrors(t *testing.T) {
	// Test with a reader that returns an error
	errorReader := &errorReader{}
	output := &bytes.Buffer{}

	_, err := selectProviderInteractive(errorReader, output)
	if err == nil {
		t.Error("Expected error from failing reader")
	}
}

// errorReader is a helper that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}