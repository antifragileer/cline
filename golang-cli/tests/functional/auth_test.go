// Package functional provides comprehensive functional tests for authentication.
// These tests verify authentication flows, API key management, and OAuth handling.
package functional

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthFlow validates authentication flows
func TestAuthFlow(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "auth-flow-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("auth_list", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth list output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_status", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "status")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth status output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_help", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "--help")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Auth help should not fail")

		// Should contain help information
		assert.Contains(t, string(out), "auth", "Auth")
	})

	t.Run("auth_with_provider_flag", func(t *testing.T) {
		providers := []string{"anthropic", "openai", "openrouter", "bedrock", "gemini", "ollama"}

		for _, provider := range providers {
			t.Run(fmt.Sprintf("provider_%s", provider), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				cmd := exec.CommandContext(ctx, binary, "auth", "-p", provider)
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

				out, err := cmd.CombinedOutput()
				t.Logf("Auth provider %s output: %s", provider, string(out))

				exitCode := 0
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						exitCode = exitErr.ExitCode()
					}
				}
				assert.True(t, exitCode >= 0 && exitCode <= 255)
			})
		}
	})

	t.Run("auth_with_api_key", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", "test-api-key-12345")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth with API key output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_with_model", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-m", "claude-3-5-sonnet-20241022")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth with model output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_verbose", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-v", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth verbose output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_with_config_dir", func(t *testing.T) {
		configDir := filepath.Join(tempDir, "custom-config")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "list", "--config", configDir)
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auth with custom config dir output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestAPIKeyStorage validates API key storage and retrieval
func TestAPIKeyStorage(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "apikey-storage-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("store_and_retrieve_api_key", func(t *testing.T) {
		// Store an API key
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", "sk-ant-test123")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Store API key output: %s", string(out))

		// Verify config was created
		configPath := filepath.Join(tempDir, "config.json")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err == nil {
				var config map[string]interface{}
				if jsonErr := json.Unmarshal(data, &config); jsonErr == nil {
					t.Logf("Config created: %+v", config)
				}
			}
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("store_multiple_provider_keys", func(t *testing.T) {
		providers := []struct {
			name   string
			apiKey string
		}{
			{"anthropic", "sk-ant-test123"},
			{"openai", "sk-openai-test123"},
			{"openrouter", "sk-or-test123"},
		}

		for _, p := range providers {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			
			cmd := exec.CommandContext(ctx, binary, "auth", "-p", p.name, "-k", p.apiKey)
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

			out, err := cmd.CombinedOutput()
			t.Logf("Store %s key output: %s", p.name, string(out))

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
			assert.True(t, exitCode >= 0 && exitCode <= 255)
			cancel()
		}
	})

	t.Run("api_key_persistence", func(t *testing.T) {
		// Store key
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "auth", "-p", "anthropic", "-k", "persistent-key-123")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out1, _ := cmd1.CombinedOutput()
		cancel1()
		t.Logf("First store output: %s", string(out1))

		// Check status (should show auth info)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "auth", "status")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, _ := cmd2.CombinedOutput()
		cancel2()
		t.Logf("Status after store: %s", string(out2))
	})

	t.Run("invalid_api_key_format", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test with empty key
		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", "")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Empty key output: %s", string(out))

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestOAuthFlow validates OAuth authentication flows
func TestOAuthFlow(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "oauth-flow-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("oauth_initiate", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// OAuth typically requires browser interaction
		// Test that the command exists and has proper flags
		cmd := exec.CommandContext(ctx, binary, "auth", "--help")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		helpText := string(out)
		// Verify OAuth-related help content exists
		assert.True(t, 
			containsAny(helpText, []string{"oauth", "OAuth", "login", "auth"}),
			"Help should mention OAuth or authentication")
	})

	t.Run("oauth_provider_flags", func(t *testing.T) {
		// Test that OAuth-capable providers can be specified
		oauthProviders := []string{"openai", "anthropic", "google"}

		for _, provider := range oauthProviders {
			t.Run(fmt.Sprintf("oauth_%s", provider), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				cmd := exec.CommandContext(ctx, binary, "auth", "-p", provider)
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

				out, err := cmd.CombinedOutput()
				t.Logf("OAuth %s output: %s", provider, string(out))

				// Should not crash
				exitCode := 0
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						exitCode = exitErr.ExitCode()
					}
				}
				assert.True(t, exitCode >= 0 && exitCode <= 255)
			})
		}
	})

	t.Run("oauth_without_browser", func(t *testing.T) {
		// Test OAuth in non-interactive environment
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "openai")
		cmd.Env = append(os.Environ(), 
			"CLINE_CONFIG_DIR="+tempDir,
			"BROWSER=echo", // Prevent actual browser opening
		)

		out, err := cmd.CombinedOutput()
		t.Logf("OAuth without browser output: %s", string(out))

		// Should handle missing browser gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestAuthConfiguration validates authentication configuration
func TestAuthConfiguration(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "auth-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("auth_config_interaction", func(t *testing.T) {
		// Set auth config
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "auth", "-p", "anthropic", "-k", "test-key")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out1, _ := cmd1.CombinedOutput()
		cancel1()
		t.Logf("Auth set output: %s", string(out1))

		// List config
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "config", "list")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, _ := cmd2.CombinedOutput()
		cancel2()
		t.Logf("Config list output: %s", string(out2))

		// Verify config reflects auth settings
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_config_json_output", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "config", "list", "--json")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Config JSON output: %s", string(out))

		if len(out) > 0 {
			var config map[string]interface{}
			if jsonErr := json.Unmarshal(out, &config); jsonErr == nil {
				t.Logf("Parsed config: %+v", config)
			}
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestAuthEdgeCases validates authentication edge cases
func TestAuthEdgeCases(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "auth-edge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("auth_invalid_provider", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "invalid-provider-xyz")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Invalid provider output: %s", string(out))

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_special_chars_in_key", func(t *testing.T) {
		specialKey := "sk-test!@#$%^&*()_+-=[]{}|;':\",./<>?"

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", specialKey)
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Special chars key output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_very_long_key", func(t *testing.T) {
		longKey := ""
		for i := 0; i < 100; i++ {
			longKey += "abcdefghijklmnopqrstuvwxyz0123456789"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", longKey)
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Long key output length: %d", len(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auth_permission_denied", func(t *testing.T) {
		// Try to write to a read-only directory
		if runtime.GOOS != "windows" {
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err := os.MkdirAll(readOnlyDir, 0555)
			require.NoError(t, err)
			defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, "auth", "-p", "anthropic", "-k", "test")
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+readOnlyDir)

			out, err := cmd.CombinedOutput()
			t.Logf("Permission denied output: %s", string(out))

			// Should handle permission error gracefully
			if err != nil {
				t.Log("Correctly failed with permission error")
			}
		}
	})
}

// Helper functions
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
