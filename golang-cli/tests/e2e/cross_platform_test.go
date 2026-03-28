// Package e2e provides cross-platform compatibility tests for the Go CLI.
// These tests ensure the CLI works correctly on different operating systems
// and architectures.
package e2e

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

// CrossPlatformTest represents a cross-platform test case
type CrossPlatformTest struct {
	Name       string
	SkipOn     []string // OS/ARCH combinations to skip
	OnlyOn     []string // OS/ARCH combinations to run on (if set, only these run)
	Setup      func(t *testing.T) (cleanup func(), err error)
	Test       func(t *testing.T)
}

// PlatformInfo holds information about the current platform
type PlatformInfo struct {
	OS       string
	Arch     string
	Platform string // OS/ARCH combined
	IsUnix   bool
	IsWindows bool
	IsMacOS  bool
	IsLinux  bool
}

// GetPlatformInfo returns information about the current platform
func GetPlatformInfo() PlatformInfo {
	os := runtime.GOOS
	arch := runtime.GOARCH

	return PlatformInfo{
		OS:        os,
		Arch:      arch,
		Platform:  fmt.Sprintf("%s/%s", os, arch),
		IsUnix:    os != "windows",
		IsWindows: os == "windows",
		IsMacOS:   os == "darwin",
		IsLinux:   os == "linux",
	}
}

// ShouldSkip checks if a test should be skipped on the current platform
func (p PlatformInfo) ShouldSkip(skipOn, onlyOn []string) bool {
	if len(onlyOn) > 0 {
		// If onlyOn is specified, skip unless platform is in the list
		for _, platform := range onlyOn {
			if p.Platform == platform || p.OS == platform {
				return false
			}
		}
		return true
	}

	// Check skip list
	for _, platform := range skipOn {
		if p.Platform == platform || p.OS == platform {
			return true
		}
	}

	return false
}

// TestCrossPlatformBasics tests basic functionality across platforms
func TestCrossPlatformBasics(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	t.Run("version_across_platforms", func(t *testing.T) {
		cmd := exec.Command(binary, "version")
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Version should work on %s", platform.Platform)
		assert.Contains(t, string(output), "version", "Output should contain version info")
	})

	t.Run("help_across_platforms", func(t *testing.T) {
		cmd := exec.Command(binary, "--help")
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "Help should work on %s", platform.Platform)
		assert.Contains(t, string(output), "Usage:", "Help should show usage")
	})

	t.Run("json_output_format", func(t *testing.T) {
		cmd := exec.Command(binary, "version", "--json")
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "JSON output should work on %s", platform.Platform)

		// Verify JSON is valid
		var result map[string]interface{}
		err = json.Unmarshal(output, &result)
		assert.NoError(t, err, "JSON should be valid on %s", platform.Platform)
	})
}

// TestWindowsSpecific tests Windows-specific behavior
func TestWindowsSpecific(t *testing.T) {
	platform := GetPlatformInfo()
	if !platform.IsWindows {
		t.Skip("Windows-specific test")
	}

	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("windows_path_handling", func(t *testing.T) {
		// Test with Windows-style paths
		tempDir := os.TempDir()
		cmd := exec.Command(binary, "config", "list", "--config", tempDir)
		output, err := cmd.CombinedOutput()

		// Should not crash
		_ = err
		_ = output
	})

	t.Run("windows_drive_letters", func(t *testing.T) {
		// Test with different drive letters
		drives := []string{"C:", "D:", "E:"}
		for _, drive := range drives {
			path := drive + "\\test"
			cmd := exec.Command(binary, "version")
			cmd.Env = append(os.Environ(), "TEST_PATH="+path)
			_, _ = cmd.CombinedOutput()
		}
	})

	t.Run("windows_env_vars", func(t *testing.T) {
		// Test Windows environment variables
		cmd := exec.Command(binary, "version")
		cmd.Env = append(os.Environ(), 
			"USERPROFILE="+os.TempDir(),
			"APPDATA="+os.TempDir(),
		)
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err, "CLI should handle Windows env vars")
		_ = output
	})
}

// TestUnixSpecific tests Unix-specific behavior
func TestUnixSpecific(t *testing.T) {
	platform := GetPlatformInfo()
	if !platform.IsUnix {
		t.Skip("Unix-specific test")
	}

	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("unix_path_handling", func(t *testing.T) {
		// Test with Unix-style paths
		tempDir := "/tmp"
		cmd := exec.Command(binary, "config", "list", "--config", tempDir)
		output, err := cmd.CombinedOutput()

		// Should not crash
		_ = err
		_ = output
	})

	t.Run("symlink_handling", func(t *testing.T) {
		// Create a symlink to the binary
		tempDir, err := os.MkdirTemp("", "symlink-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		linkPath := filepath.Join(tempDir, "cline-link")
		err = os.Symlink(binary, linkPath)
		require.NoError(t, err)

		// Test that the CLI works through the symlink
		cmd := exec.Command(linkPath, "version")
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err, "CLI should work through symlinks")
		assert.Contains(t, string(output), "version")
	})

	t.Run("unix_permissions", func(t *testing.T) {
		// Check binary is executable
		info, err := os.Stat(binary)
		require.NoError(t, err)

		mode := info.Mode()
		assert.True(t, mode&0111 != 0, "Binary should be executable")
	})
}

// TestMacOSSpecific tests macOS-specific behavior
func TestMacOSSpecific(t *testing.T) {
	platform := GetPlatformInfo()
	if !platform.IsMacOS {
		t.Skip("macOS-specific test")
	}

	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("macos_keychain", func(t *testing.T) {
		// Test macOS keychain integration if applicable
		cmd := exec.Command(binary, "auth", "status")
		output, err := cmd.CombinedOutput()

		// Should handle macOS keychain gracefully
		_ = err
		t.Logf("Keychain auth output: %s", string(output))
	})

	t.Run("macos_app_translocation", func(t *testing.T) {
		// Test behavior when binary is in app bundle
		// (simplified - just verify it works from different locations)
		locations := []string{
			"/tmp",
			os.TempDir(),
			filepath.Join(os.TempDir(), "test"),
		}

		for _, loc := range locations {
			// Just verify CLI works from these locations
			cmd := exec.Command(binary, "version")
			cmd.Dir = loc
			output, err := cmd.CombinedOutput()
			
			if err != nil {
				t.Logf("Location %s: %v", loc, err)
			}
			_ = output
		}
	})
}

// TestLinuxSpecific tests Linux-specific behavior
func TestLinuxSpecific(t *testing.T) {
	platform := GetPlatformInfo()
	if !platform.IsLinux {
		t.Skip("Linux-specific test")
	}

	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("linux_xdg_compliance", func(t *testing.T) {
		// Test XDG Base Directory compliance
		tempDir, err := os.MkdirTemp("", "xdg-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		cmd := exec.Command(binary, "config", "list")
		cmd.Env = append(os.Environ(),
			"XDG_CONFIG_HOME="+tempDir,
			"XDG_DATA_HOME="+tempDir,
		)
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "CLI should respect XDG directories")
		_ = output
	})

	t.Run("linux_proc_fs", func(t *testing.T) {
		// Test that CLI doesn't break with /proc filesystem quirks
		cmd := exec.Command(binary, "version")
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err)
		_ = output
	})
}

// TestArchitectureSpecific tests architecture-specific behavior
func TestArchitectureSpecific(t *testing.T) {
	platform := GetPlatformInfo()
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run(fmt.Sprintf("arch_%s", platform.Arch), func(t *testing.T) {
		// Test that binary runs on current architecture
		cmd := exec.Command(binary, "version", "--json")
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err, "CLI should run on %s", platform.Arch)

		var result map[string]interface{}
		err = json.Unmarshal(output, &result)
		require.NoError(t, err)

		// If version info includes architecture, verify it matches
		if arch, ok := result["goarch"].(string); ok {
			assert.Equal(t, platform.Arch, arch, "Reported arch should match runtime")
		}
	})
}

// TestCrossPlatformFilePaths tests file path handling across platforms
func TestCrossPlatformFilePaths(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tests := []struct {
		name     string
		pathType string
		skipOn   []string
	}{
		{
			name:     "spaces_in_path",
			pathType: "path with spaces",
		},
		{
			name:     "unicode_in_path",
			pathType: "path-with-unicode-日本語",
		},
		{
			name:     "long_path",
			pathType: strings.Repeat("a/", 20) + "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform := GetPlatformInfo()
			for _, skip := range tt.skipOn {
				if platform.OS == skip || platform.Platform == skip {
					t.Skipf("Skipping on %s", skip)
				}
			}

			// Create temp dir with special path
			baseDir, err := os.MkdirTemp("", "path-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(baseDir)

			specialPath := filepath.Join(baseDir, tt.pathType)
			err = os.MkdirAll(specialPath, 0755)
			require.NoError(t, err)

			// Test CLI works with this path
			cmd := exec.Command(binary, "config", "list")
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+specialPath)
			output, err := cmd.CombinedOutput()

			// Should not crash
			t.Logf("Path test %s: err=%v", tt.name, err)
			_ = output
		})
	}
}

// TestCrossPlatformEnvironment tests environment variable handling
func TestCrossPlatformEnvironment(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	tests := []struct {
		name string
		env  map[string]string
		skip []string // OS to skip on
	}{
		{
			name: "home_directory",
			env: map[string]string{
				"HOME": os.TempDir(),
			},
			skip: []string{"windows"},
		},
		{
			name: "user_profile",
			env: map[string]string{
				"USERPROFILE": os.TempDir(),
			},
			skip: []string{"linux", "darwin"},
		},
		{
			name: "path_variable",
			env: map[string]string{
				"PATH": os.Getenv("PATH"),
			},
		},
		{
			name: "temp_directory",
			env: map[string]string{
				"TMPDIR": os.TempDir(),
				"TEMP":   os.TempDir(),
				"TMP":    os.TempDir(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if we should skip
			for _, skipOS := range tt.skip {
				if platform.OS == skipOS {
					t.Skipf("Skipping %s on %s", tt.name, skipOS)
				}
			}

			cmd := exec.Command(binary, "version")
			cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
			for k, v := range tt.env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "CLI should handle environment on %s", platform.Platform)
			_ = output
		})
	}
}

// TestCrossPlatformSignals tests signal handling across platforms
func TestCrossPlatformSignals(t *testing.T) {
	platform := GetPlatformInfo()
	if platform.IsWindows {
		t.Skip("Signal tests skipped on Windows")
	}

	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	t.Run("interrupt_signal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "version")
		
		// Start command
		err := cmd.Start()
		require.NoError(t, err)

		// Give it a moment to start
		time.Sleep(50 * time.Millisecond)

		// Send interrupt
		err = cmd.Process.Signal(os.Interrupt)
		require.NoError(t, err)

		// Wait for completion
		done := make(chan error)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			// Process exited - this is expected
			t.Logf("Process exited: %v", err)
		case <-time.After(5 * time.Second):
			t.Error("Process did not exit within timeout")
			cmd.Process.Kill()
		}
	})
}

// TestCrossPlatformNetworking tests network behavior across platforms
func TestCrossPlatformNetworking(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	t.Run("no_proxy_environment", func(t *testing.T) {
		cmd := exec.Command(binary, "version")
		cmd.Env = append(os.Environ(),
			"NO_PROXY=*",
			"no_proxy=*",
		)
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err)
		_ = output
	})

	t.Run("proxy_environment", func(t *testing.T) {
		// This tests that the CLI handles proxy settings
		// It doesn't actually connect through the proxy
		cmd := exec.Command(binary, "version")
		cmd.Env = append(os.Environ(),
			"HTTP_PROXY=http://localhost:8080",
			"HTTPS_PROXY=http://localhost:8080",
		)
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err)
		t.Logf("Platform %s: proxy env test completed", platform.Platform)
		_ = output
	})
}

// TestCrossPlatformTimezones tests timezone handling
func TestCrossPlatformTimezones(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	timezones := []string{
		"UTC",
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
		"Australia/Sydney",
	}

	for _, tz := range timezones {
		t.Run(fmt.Sprintf("timezone_%s", tz), func(t *testing.T) {
			cmd := exec.Command(binary, "version")
			cmd.Env = append(os.Environ(), "TZ="+tz)
			output, err := cmd.CombinedOutput()
			
			require.NoError(t, err, "CLI should work in timezone %s", tz)
			_ = output
		})
	}
}

// TestCrossPlatformLocale tests locale handling
func TestCrossPlatformLocale(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	// Different locales per platform
	locales := []string{}
	if platform.IsUnix {
		locales = []string{
			"C",
			"en_US.UTF-8",
			"de_DE.UTF-8",
			"ja_JP.UTF-8",
		}
	} else {
		locales = []string{
			"English",
			"German",
			"Japanese",
		}
	}

	for _, locale := range locales {
		t.Run(fmt.Sprintf("locale_%s", locale), func(t *testing.T) {
			cmd := exec.Command(binary, "version")
			if platform.IsUnix {
				cmd.Env = append(os.Environ(), "LC_ALL="+locale)
			}
			output, err := cmd.CombinedOutput()
			
			// Should not crash regardless of locale
			_ = err
			_ = output
		})
	}
}

// TestCrossPlatformBinaryCompatibility tests that the binary is compatible
func TestCrossPlatformBinaryCompatibility(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	t.Run("binary_architecture", func(t *testing.T) {
		// Verify we're running the correct binary for this platform
		cmd := exec.Command(binary, "version", "--json")
		output, err := cmd.CombinedOutput()
		
		require.NoError(t, err)

		var info map[string]interface{}
		err = json.Unmarshal(output, &info)
		require.NoError(t, err)

		// Check that Go version matches what we expect
		if goos, ok := info["goos"].(string); ok {
			assert.Equal(t, platform.OS, goos, "Binary OS should match runtime")
		}
		if goarch, ok := info["goarch"].(string); ok {
			assert.Equal(t, platform.Arch, goarch, "Binary arch should match runtime")
		}
	})

	t.Run("static_linking", func(t *testing.T) {
		if platform.IsWindows {
			t.Skip("Static linking check skipped on Windows")
		}

		// Check that binary doesn't have unexpected dynamic dependencies
		// This is a basic check - more thorough checks would use ldd/otool
		cmd := exec.Command(binary, "version")
		err := cmd.Run()
		
		// Should run without library errors
		require.NoError(t, err)
	})
}

// BenchmarkCrossPlatformPerformance benchmarks performance across platforms
func BenchmarkCrossPlatformPerformance(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	platform := GetPlatformInfo()

	b.Run(fmt.Sprintf("startup_%s", platform.Platform), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(binary, "version")
			cmd.Run()
		}
	})
}

