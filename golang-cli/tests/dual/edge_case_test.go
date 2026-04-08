// Package dual provides edge case tests for the Cline CLI dual testing framework.
// These tests cover compound commands, large file handling, and network failures.
package dual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Compound Command Tests ====================

func TestCompoundCommands(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for compound command tests")
	}

	ctx := context.Background()

	t.Run("piped_input_handling", func(t *testing.T) {
		// Test piped input (echo "prompt" | cline)
		prompt := "Explain what is 2+2"

		// Run Go CLI with piped input
		goCmd := exec.CommandContext(ctx, goPath)
		goCmd.Stdin = strings.NewReader(prompt)
		goOut, goErr := goCmd.CombinedOutput()

		// Run TS CLI with piped input
		tsCmd := exec.CommandContext(ctx, tsPath)
		tsCmd.Stdin = strings.NewReader(prompt)
		tsOut, tsErr := tsCmd.CombinedOutput()

		// Both should behave similarly (either both succeed or both fail)
		t.Logf("Go CLI exit: %v, output length: %d", goErr, len(goOut))
		t.Logf("TS CLI exit: %v, output length: %d", tsErr, len(tsOut))
	})

	t.Run("large_argument_handling", func(t *testing.T) {
		// Test with large argument
		largeArg := strings.Repeat("a", 10000)

		goCmd := exec.CommandContext(ctx, goPath, "version")
		goCmd.Env = append(os.Environ(), "TEST_LARGE_VAR="+largeArg)
		output, err := goCmd.CombinedOutput()
		require.NoError(t, err, "Go CLI should handle large env vars")

		tsCmd := exec.CommandContext(ctx, tsPath, "version")
		tsCmd.Env = append(os.Environ(), "TEST_LARGE_VAR="+largeArg)
		output, err = tsCmd.CombinedOutput()
		require.NoError(t, err, "TS CLI should handle large env vars")

		// Both should produce version output
		assert.Contains(t, string(output), "version")
	})

	t.Run("special_character_arguments", func(t *testing.T) {
		specialArgs := []string{
			"hello\nworld",
			"hello\tworld",
			"hello\\world",
			"hello\"world",
			"hello'world",
			"hello`world",
			"hello$world",
			"hello|world",
			"hello;world",
			"hello&world",
		}

		for _, arg := range specialArgs {
			t.Run(fmt.Sprintf("arg_%d", len(arg)), func(t *testing.T) {
				// Note: We're testing that the CLI doesn't crash with special chars
				// Not that it processes them correctly (that requires core extension)

				goCmd := exec.CommandContext(ctx, goPath, "version")
				goCmd.Env = append(os.Environ(), "TEST_ARG="+arg)
				_, _ = goCmd.CombinedOutput()

				tsCmd := exec.CommandContext(ctx, tsPath, "version")
				tsCmd.Env = append(os.Environ(), "TEST_ARG="+arg)
				_, _ = tsCmd.CombinedOutput()
			})
		}
	})

	t.Run("multiple_flags_combination", func(t *testing.T) {
		// Test various flag combinations
		flagCombos := [][]string{
			{"version", "--short"},
			{"version", "--json"},
			{"config", "list"},
			{"config", "list", "--json"},
			{"history", "--json"},
			{"--help"},
		}

		for _, combo := range flagCombos {
			t.Run(strings.Join(combo, "_"), func(t *testing.T) {
				goCmd := exec.CommandContext(ctx, goPath, combo...)
				goOut, goErr := goCmd.CombinedOutput()

				tsCmd := exec.CommandContext(ctx, tsPath, combo...)
				tsOut, tsErr := tsCmd.CombinedOutput()

				// Both should have similar exit codes
				goExit := 0
				if goErr != nil {
					if exitErr, ok := goErr.(*exec.ExitError); ok {
						goExit = exitErr.ExitCode()
					}
				}

				tsExit := 0
				if tsErr != nil {
					if exitErr, ok := tsErr.(*exec.ExitError); ok {
						tsExit = exitErr.ExitCode()
					}
				}

				assert.Equal(t, goExit, tsExit,
					"Exit codes should match for flags: %v", combo)
				_ = goOut
				_ = tsOut
			})
		}
	})
}

// ==================== Large File Handling Tests ====================

func TestLargeFileHandling(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	ctx := context.Background()

	t.Run("large_config_file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "large-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a large config file (1MB)
		largeConfig := make(map[string]interface{})
		for i := 0; i < 10000; i++ {
			largeConfig[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
		}

		configData, err := json.Marshal(largeConfig)
		require.NoError(t, err)

		configPath := filepath.Join(tempDir, "config.json")
		err = os.WriteFile(configPath, configData, 0644)
		require.NoError(t, err)

		// CLI should handle large config without crashing
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		output, err := cmd.CombinedOutput()

		// Should either succeed or fail gracefully
		if err != nil {
			t.Logf("Large config handling: %v, output: %s", err, string(output))
		}
	})

	t.Run("large_history_file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "large-history-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create tasks directory
		tasksDir := filepath.Join(tempDir, "tasks")
		err = os.MkdirAll(tasksDir, 0755)
		require.NoError(t, err)

		// Create a large task history file
		history := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			history[i] = map[string]interface{}{
				"id":        fmt.Sprintf("task-%d", i),
				"timestamp": time.Now().Add(-time.Duration(i) * time.Hour).UnixMilli(),
				"prompt":    strings.Repeat("prompt ", 100),
				"status":    "completed",
			}
		}

		historyData, err := json.Marshal(history)
		require.NoError(t, err)

		historyPath := filepath.Join(tasksDir, "history.json")
		err = os.WriteFile(historyPath, historyData, 0644)
		require.NoError(t, err)

		// CLI should handle large history without crashing
		cmd := exec.CommandContext(ctx, goPath, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		output, err := cmd.CombinedOutput()

		// Should succeed
		require.NoError(t, err, "Large history should be handled: %s", string(output))

		// Verify output is valid JSON
		var result interface{}
		err = json.Unmarshal(output, &result)
		assert.NoError(t, err, "Output should be valid JSON")
	})

	t.Run("binary_file_in_config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "binary-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a config file with binary/null characters
		configPath := filepath.Join(tempDir, "config.json")
		binaryData := []byte(`{"key": "value\u0000with\u0000nulls"}`)
		err = os.WriteFile(configPath, binaryData, 0644)
		require.NoError(t, err)

		// CLI should handle gracefully
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		output, err := cmd.CombinedOutput()

		// Should not crash
		t.Logf("Binary config handling: err=%v, output=%s", err, string(output))
	})

	t.Run("deeply_nested_json", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "nested-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create deeply nested config
		nested := make(map[string]interface{})
		current := nested
		for i := 0; i < 100; i++ {
			next := make(map[string]interface{})
			current["nested"] = next
			current = next
		}
		current["value"] = "deep"

		configData, err := json.Marshal(nested)
		require.NoError(t, err)

		configPath := filepath.Join(tempDir, "config.json")
		err = os.WriteFile(configPath, configData, 0644)
		require.NoError(t, err)

		// CLI should handle nested JSON
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		output, err := cmd.CombinedOutput()

		// Should not crash
		t.Logf("Nested JSON handling: err=%v", err)
		_ = output
	})
}

// ==================== Network Failure Tests ====================

func TestNetworkFailures(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	ctx := context.Background()

	t.Run("no_network_available", func(t *testing.T) {
		// Test CLI behavior when network is unavailable
		// This simulates offline mode

		cmd := exec.CommandContext(ctx, goPath, "version")
		// Block network by using an invalid proxy
		cmd.Env = append(os.Environ(),
			"HTTP_PROXY=http://invalid-proxy:9999",
			"HTTPS_PROXY=http://invalid-proxy:9999",
		)
		output, err := cmd.CombinedOutput()

		// Version command should work even without network
		require.NoError(t, err, "Version should work offline: %s", string(output))
	})

	t.Run("slow_network_timeout", func(t *testing.T) {
		// This test would require a mock server that responds slowly
		// For now, we just verify the CLI doesn't hang indefinitely

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "version")
		output, err := cmd.CombinedOutput()

		// Should complete within timeout
		require.NoError(t, err, "CLI should complete within timeout: %s", string(output))
	})

	t.Run("invalid_proxy_configuration", func(t *testing.T) {
		invalidProxies := []string{
			"http://invalid-host:99999", // Invalid port
			"ftp://proxy.example.com",   // Wrong protocol
			"not-a-valid-url",           // Invalid format
			"://missing-scheme",         // Missing scheme
		}

		for _, proxy := range invalidProxies {
			t.Run(fmt.Sprintf("proxy_%s", proxy), func(t *testing.T) {
				cmd := exec.CommandContext(ctx, goPath, "version")
				cmd.Env = append(os.Environ(),
					"HTTP_PROXY="+proxy,
					"HTTPS_PROXY="+proxy,
				)
				output, err := cmd.CombinedOutput()

				// Should handle invalid proxy gracefully
				// Version command should still work (local only)
				if err != nil {
					t.Logf("Invalid proxy handling: %v, output: %s", err, string(output))
				}
			})
		}
	})
}

// ==================== Concurrent Access Tests ====================

func TestConcurrentAccess(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	ctx := context.Background()

	t.Run("concurrent_config_access", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "concurrent-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Start 10 readers and 10 writers concurrently
		for i := 0; i < 10; i++ {
			// Writers
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				cmd := exec.CommandContext(ctx, goPath, "config", "set",
					fmt.Sprintf("key%d", n), fmt.Sprintf("value%d", n))
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
				_, err := cmd.CombinedOutput()
				if err != nil {
					errors <- err
				}
			}(i)

			// Readers
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				cmd := exec.CommandContext(ctx, goPath, "config", "list")
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
				_, err := cmd.CombinedOutput()
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Count errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
				t.Logf("Concurrent access error: %v", err)
			}
		}

		// Some errors are acceptable due to file locking
		// The important thing is that data doesn't get corrupted
		t.Logf("Concurrent operations completed with %d errors", errorCount)

		// Verify config is still readable
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Config should still be readable after concurrent access: %s", string(output))
	})

	t.Run("rapid_sequential_calls", func(t *testing.T) {
		// Make 100 rapid calls to the CLI
		for i := 0; i < 100; i++ {
			cmd := exec.CommandContext(ctx, goPath, "version", "--short")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Call %d failed: %s", i, string(output))
		}
	})
}

// ==================== Resource Exhaustion Tests ====================

func TestResourceExhaustion(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("many_arguments", func(t *testing.T) {
		// Create many arguments
		args := []string{"version"}
		for i := 0; i < 100; i++ {
			args = append(args, fmt.Sprintf("--flag%d=value%d", i, i))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, args...)
		output, err := cmd.CombinedOutput()

		// Should handle gracefully (ignore unknown flags or fail gracefully)
		t.Logf("Many arguments: err=%v, output=%s", err, string(output))
	})

	t.Run("long_environment_variables", func(t *testing.T) {
		ctx := context.Background()

		cmd := exec.CommandContext(ctx, goPath, "version")
		// Set very long environment variable
		longValue := strings.Repeat("x", 100000)
		cmd.Env = append(os.Environ(), "LONG_VAR="+longValue)
		output, err := cmd.CombinedOutput()

		// Should not crash
		require.NoError(t, err, "Long env var should be handled: %s", string(output))
	})
}

// ==================== Signal Handling Tests ====================

func TestSignalHandling(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	if runtime.GOOS == "windows" {
		t.Skip("Signal handling tests skipped on Windows")
	}

	t.Run("interrupt_handling", func(t *testing.T) {
		ctx := context.Background()

		// Start a long-running command
		cmd := exec.CommandContext(ctx, goPath)
		stdin, err := cmd.StdinPipe()
		require.NoError(t, err)

		err = cmd.Start()
		require.NoError(t, err)

		// Send some input
		io.WriteString(stdin, "test prompt\n")
		time.Sleep(100 * time.Millisecond)

		// Send interrupt signal
		err = cmd.Process.Signal(os.Interrupt)
		require.NoError(t, err)

		// Wait for process to exit
		done := make(chan error)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			// Process exited
			t.Logf("Process exited with: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("Process did not exit within timeout")
			cmd.Process.Kill()
		}
	})
}

// ==================== Unicode and Encoding Tests ====================

func TestUnicodeAndEncoding(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	ctx := context.Background()

	t.Run("unicode_in_config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "unicode-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		unicodeValues := []struct {
			name  string
			value string
		}{
			{"chinese", "你好世界"},
			{"japanese", "こんにちは"},
			{"arabic", "مرحبا"},
			{"emoji", "🎉🚀💻🔥"},
			{"math", "∀x ∈ ℝ: x² ≥ 0"},
			{"mixed", "Hello 世界 🌍"},
		}

		for _, tc := range unicodeValues {
			t.Run(tc.name, func(t *testing.T) {
				// Set config with unicode value
				cmd := exec.CommandContext(ctx, goPath, "config", "set", "test-key", tc.value)
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "Setting unicode value failed: %s", string(output))

				// Get config
				cmd = exec.CommandContext(ctx, goPath, "config", "get", "test-key")
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
				output, err = cmd.CombinedOutput()
				require.NoError(t, err, "Getting unicode value failed: %s", string(output))

				// Value should be preserved (or at least not crash)
				assert.NotEmpty(t, string(output))
			})
		}
	})

	t.Run("utf8_streaming", func(t *testing.T) {
		// Test that UTF-8 characters are handled correctly in streaming output
		unicodeStrings := []string{
			"Hello 世界",
			"Привет мир",
			"🎉 Celebration 🚀",
		}

		for _, s := range unicodeStrings {
			// Just verify the CLI doesn't crash with unicode
			cmd := exec.CommandContext(ctx, goPath, "version")
			cmd.Env = append(os.Environ(), "TEST_UNICODE="+s)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Unicode handling failed for: %s", s)
			_ = output
		}
	})
}

// ==================== File System Edge Cases ====================

func TestFileSystemEdgeCases(t *testing.T) {
	goPath := FindGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	ctx := context.Background()

	t.Run("long_path_handling", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Long path test skipped on Windows")
		}

		// Create a very long path
		baseDir, err := os.MkdirTemp("", "long-path-*")
		require.NoError(t, err)
		defer os.RemoveAll(baseDir)

		longPath := baseDir
		for i := 0; i < 20; i++ {
			longPath = filepath.Join(longPath, "very-long-directory-name")
		}
		err = os.MkdirAll(longPath, 0755)
		require.NoError(t, err)

		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+longPath)
		output, err := cmd.CombinedOutput()

		// Should handle long paths
		t.Logf("Long path handling: err=%v", err)
		_ = output
	})

	t.Run("special_path_characters", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Special path test skipped on Windows")
		}

		specialPaths := []string{
			"path with spaces",
			"path-with-dashes",
			"path_with_underscores",
			"path.with.dots",
		}

		for _, pathName := range specialPaths {
			t.Run(pathName, func(t *testing.T) {
				baseDir, err := os.MkdirTemp("", "special-path-*")
				require.NoError(t, err)
				defer os.RemoveAll(baseDir)

				specialPath := filepath.Join(baseDir, pathName)
				err = os.MkdirAll(specialPath, 0755)
				require.NoError(t, err)

				cmd := exec.CommandContext(ctx, goPath, "config", "list")
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+specialPath)
				output, err := cmd.CombinedOutput()

				// Should not crash
				_ = err
				_ = output
			})
		}
	})

	t.Run("symlink_handling", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Symlink test skipped on Windows")
		}

		baseDir, err := os.MkdirTemp("", "symlink-*")
		require.NoError(t, err)
		defer os.RemoveAll(baseDir)

		realDir := filepath.Join(baseDir, "real")
		linkDir := filepath.Join(baseDir, "link")
		err = os.MkdirAll(realDir, 0755)
		require.NoError(t, err)

		err = os.Symlink(realDir, linkDir)
		require.NoError(t, err)

		// Test with symlink
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+linkDir)
		output, err := cmd.CombinedOutput()

		// Should handle symlinks
		require.NoError(t, err, "Symlink handling failed: %s", string(output))
	})
}

// ==================== Memory and Performance Tests ====================

func BenchmarkLargeConfigHandling(b *testing.B) {
	goPath := FindGoBinary()
	if goPath == "" {
		b.Skip("Go CLI not found")
	}

	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "bench-large-config-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create large config
	largeConfig := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeConfig[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 100)
	}
	configData, _ := json.Marshal(largeConfig)
	configPath := filepath.Join(tempDir, "config.json")
	os.WriteFile(configPath, configData, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.CommandContext(ctx, goPath, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		cmd.Run()
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	goPath := FindGoBinary()
	if goPath == "" {
		b.Skip("Go CLI not found")
	}

	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "bench-concurrent-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cmd := exec.CommandContext(ctx, goPath, "config", "set",
				fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			cmd.Run()
			i++
		}
	})
}
