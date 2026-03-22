// Package security provides command validation and security enforcement
// for the Cline CLI. This file contains comprehensive tests for the
// command validation engine.
package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewCommandValidator tests validator creation
func TestNewCommandValidator(t *testing.T) {
	t.Run("creates validator with default options", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		v, err := NewCommandValidator(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if v == nil {
			t.Fatal("expected validator to not be nil")
		}
		defer v.Close()
	})

	t.Run("creates validator with disabled cache", func(t *testing.T) {
		opts := ValidatorOptions{
			EnableCache:   false,
			EnableLogging: false,
		}
		v, err := NewCommandValidator(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if v.cache != nil {
			t.Error("expected cache to be nil when disabled")
		}
		defer v.Close()
	})

	t.Run("creates validator with log file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "validation.log")

		opts := ValidatorOptions{
			EnableLogging: true,
			LogPath:       logPath,
		}
		v, err := NewCommandValidator(opts)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer v.Close()

		if v.logger == nil {
			t.Error("expected logger to be initialized")
		}

		// Verify log file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("expected log file to be created")
		}
	})

	t.Run("fails with invalid log path", func(t *testing.T) {
		opts := ValidatorOptions{
			EnableLogging: true,
			LogPath:       "/nonexistent/directory/log.txt",
		}
		_, err := NewCommandValidator(opts)
		if err == nil {
			t.Error("expected error for invalid log path")
		}
	})
}

// TestAddAllowRule tests adding allow rules
func TestAddAllowRule(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	t.Run("adds valid allow rule", func(t *testing.T) {
		err := v.AddAllowRule("git *", "allow git commands")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		allow, _ := v.GetRules()
		if len(allow) != 1 {
			t.Fatalf("expected 1 allow rule, got %d", len(allow))
		}
		if allow[0].Pattern != "git *" {
			t.Errorf("expected pattern 'git *', got %s", allow[0].Pattern)
		}
		if allow[0].Type != RuleTypeAllow {
			t.Errorf("expected type allow, got %s", allow[0].Type)
		}
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		err := v.AddAllowRule("", "empty pattern")
		if err == nil {
			t.Error("expected error for empty pattern")
		}
	})

	t.Run("rejects invalid glob pattern", func(t *testing.T) {
		err := v.AddAllowRule("test{invalid", "unbalanced braces")
		if err == nil {
			t.Error("expected error for invalid glob pattern")
		}
	})
}

// TestAddDenyRule tests adding deny rules
func TestAddDenyRule(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	t.Run("adds valid deny rule", func(t *testing.T) {
		err := v.AddDenyRule("rm -rf /", "prevent dangerous rm")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, deny := v.GetRules()
		if len(deny) != 1 {
			t.Fatalf("expected 1 deny rule, got %d", len(deny))
		}
		if deny[0].Pattern != "rm -rf /" {
			t.Errorf("expected pattern 'rm -rf /', got %s", deny[0].Pattern)
		}
		if deny[0].Type != RuleTypeDeny {
			t.Errorf("expected type deny, got %s", deny[0].Type)
		}
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		err := v.AddDenyRule("", "empty pattern")
		if err == nil {
			t.Error("expected error for empty pattern")
		}
	})
}

// TestRemoveRule tests rule removal
func TestRemoveRule(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	t.Run("removes allow rule", func(t *testing.T) {
		v.AddAllowRule("test-pattern", "test")
		removed := v.RemoveRule("test-pattern")
		if !removed {
			t.Error("expected rule to be removed")
		}
		allow, _ := v.GetRules()
		if len(allow) != 0 {
			t.Error("expected no allow rules after removal")
		}
	})

	t.Run("removes deny rule", func(t *testing.T) {
		v.AddDenyRule("dangerous-pattern", "test")
		removed := v.RemoveRule("dangerous-pattern")
		if !removed {
			t.Error("expected rule to be removed")
		}
		_, deny := v.GetRules()
		if len(deny) != 0 {
			t.Error("expected no deny rules after removal")
		}
	})

	t.Run("returns false for non-existent rule", func(t *testing.T) {
		removed := v.RemoveRule("non-existent")
		if removed {
			t.Error("expected false for non-existent rule")
		}
	})
}

// TestDenyPrecedesAllow tests that deny rules take precedence
func TestDenyPrecedesAllow(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	// Add allow rule for all git commands
	v.AddAllowRule("git *", "allow all git")
	// Add deny rule for git push specifically
	v.AddDenyRule("git push *", "deny git push")

	tests := []struct {
		name     string
		command  string
		expected bool
		reason   string
	}{
		{"allowed git command", "git status", true, ""},
		{"allowed git clone", "git clone https://example.com/repo", true, ""},
		{"denied git push", "git push origin main", false, "denied"},
		{"denied git push with args", "git push --force", false, "denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.Validate(ctx, tt.command)
			if result.Allowed != tt.expected {
				t.Errorf("expected allowed=%v, got %v (reason: %s)", tt.expected, result.Allowed, result.Reason)
			}
			if !tt.expected && !strings.Contains(strings.ToLower(result.Reason), tt.reason) {
				t.Errorf("expected reason to contain %q, got %q", tt.reason, result.Reason)
			}
			if !tt.expected && result.DeniedBy == nil {
				t.Error("expected DeniedBy to be set when command is denied")
			}
		})
	}
}

// TestGlobMatching tests glob pattern matching
func TestGlobMatching(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	tests := []struct {
		pattern string
		command string
		match   bool
	}{
		// Simple wildcards
		{"*", "any command", true},
		{"git *", "git status", true},
		{"git *", "git clone url", true},
		{"git *", "docker ps", false},
		{"git status", "git status", true},
		{"git status", "git clone", false},

		// Character wildcards
		{"git statu?", "git status", true},
		{"git statu?", "git statu", false},

		// Character classes
		{"git [sc]*", "git status", true},
		{"git [sc]*", "git commit", true},
		{"git [sc]*", "git push", false},

		// Brace expansion
		{"git {status,clone,log}", "git status", true},
		{"git {status,clone,log}", "git clone", true},
		{"git {status,clone,log}", "git push", false},

		// Multiple wildcards
		{"git * --*", "git status --short", true},
		{"git * --*", "git log --oneline", true},
		{"git * --*", "git status", false},

		// Case insensitivity
		{"GIT STATUS", "git status", true},
		{"git STATUS", "GIT status", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("pattern=%s cmd=%s", tt.pattern, tt.command), func(t *testing.T) {
			v.ClearRules()
			v.AddAllowRule(tt.pattern, "test")

			result := v.Validate(ctx, tt.command)
			if result.Allowed != tt.match {
				t.Errorf("pattern=%q command=%q expected=%v got=%v reason=%s",
					tt.pattern, tt.command, tt.match, result.Allowed, result.Reason)
			}
		})
	}
}

// TestCompoundCommandValidation tests validation of compound commands
func TestCompoundCommandValidation(t *testing.T) {
	t.Run("validates segments separately", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		// Allow git and docker, deny npm
		v.AddAllowRule("git *", "allow git")
		v.AddAllowRule("docker *", "allow docker")
		v.AddDenyRule("npm *", "deny npm")

		result := v.Validate(ctx, "git status && docker ps")
		if !result.Allowed {
			t.Errorf("expected compound command to be allowed, got: %s", result.Reason)
		}
		if len(result.Segments) == 0 {
			t.Error("expected segments to be validated")
		}
	})

	t.Run("denies if any segment denied", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		v.AddAllowRule("git *", "allow git")
		v.AddDenyRule("npm *", "deny npm")

		result := v.Validate(ctx, "git status && npm install")
		if result.Allowed {
			t.Error("expected compound command with npm to be denied")
		}
	})

	t.Run("validates piped commands", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		v.AddAllowRule("cat *", "allow cat")
		v.AddAllowRule("grep *", "allow grep")

		result := v.Validate(ctx, "cat file.txt | grep pattern")
		if !result.Allowed {
			t.Errorf("expected piped command to be allowed, got: %s", result.Reason)
		}
	})

	t.Run("validates semicolon separated commands", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		v.AddAllowRule("echo *", "allow echo")
		v.AddAllowRule("ls *", "allow ls")

		result := v.Validate(ctx, "echo hello ; ls -la")
		if !result.Allowed {
			t.Errorf("expected semicolon command to be allowed, got: %s", result.Reason)
		}
	})

	t.Run("validates or commands", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		v.AddAllowRule("command1", "allow command1")
		v.AddAllowRule("command2", "allow command2")

		result := v.Validate(ctx, "command1 || command2")
		if !result.Allowed {
			t.Errorf("expected or command to be allowed, got: %s", result.Reason)
		}
	})

	t.Run("disabled compound validation", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = false
		v, _ := NewCommandValidator(opts)
		defer v.Close()

		ctx := context.Background()

		v.AddAllowRule("git status && *", "allow pattern")

		result := v.Validate(ctx, "git status && npm install")
		if !result.Allowed {
			t.Errorf("expected pattern match without segment validation, got: %s", result.Reason)
		}
	})
}

// TestDetailedValidationResults tests that validation results contain expected details
func TestDetailedValidationResults(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git status", "check git status")
	v.AddDenyRule("rm -rf /", "prevent rm -rf /")

	t.Run("contains all fields for allowed command", func(t *testing.T) {
		result := v.Validate(ctx, "git status")

		if !result.Allowed {
			t.Error("expected command to be allowed")
		}
		if result.Command != "git status" {
			t.Errorf("expected Command='git status', got %q", result.Command)
		}
		if result.NormalizedCommand != "git status" {
			t.Errorf("expected NormalizedCommand='git status', got %q", result.NormalizedCommand)
		}
		if result.MatchedRule == nil {
			t.Error("expected MatchedRule to be set")
		}
		if result.Reason == "" {
			t.Error("expected Reason to be set")
		}
		if result.Timestamp.IsZero() {
			t.Error("expected Timestamp to be set")
		}
	})

	t.Run("contains all fields for denied command", func(t *testing.T) {
		result := v.Validate(ctx, "rm -rf /")

		if result.Allowed {
			t.Error("expected command to be denied")
		}
		if result.DeniedBy == nil {
			t.Error("expected DeniedBy to be set")
		}
		if !strings.Contains(result.Reason, "denied") {
			t.Errorf("expected Reason to contain 'denied', got %q", result.Reason)
		}
	})

	t.Run("contains segment validations", func(t *testing.T) {
		opts := DefaultValidatorOptions()
		opts.ValidateCompoundCommands = true
		v2, _ := NewCommandValidator(opts)
		defer v2.Close()

		v2.AddAllowRule("git *", "allow git")
		v2.AddAllowRule("docker *", "allow docker")

		result := v2.Validate(ctx, "git status && docker ps")
		if len(result.Segments) == 0 {
			t.Error("expected Segments to be populated")
		}
	})
}

// TestCaching tests validation result caching
func TestCaching(t *testing.T) {
	opts := DefaultValidatorOptions()
	opts.EnableCache = true
	opts.CacheTTL = 100 * time.Millisecond
	v, _ := NewCommandValidator(opts)
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")

	t.Run("caches results", func(t *testing.T) {
		// First validation
		result1 := v.Validate(ctx, "git status")
		if result1.CacheHit {
			t.Error("first validation should not be cache hit")
		}

		// Second validation should be cached
		result2 := v.Validate(ctx, "git status")
		if !result2.CacheHit {
			t.Error("second validation should be cache hit")
		}
	})

	t.Run("cache respects TTL", func(t *testing.T) {
		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		result := v.Validate(ctx, "git status")
		if result.CacheHit {
			t.Error("validation after TTL should not be cache hit")
		}
	})

	t.Run("cache cleared on rule change", func(t *testing.T) {
		// Validate to populate cache
		v.Validate(ctx, "git status")

		// Add new rule
		v.AddAllowRule("docker *", "allow docker")

		// Should not be cache hit after rule change
		result := v.Validate(ctx, "git status")
		if result.CacheHit {
			t.Error("validation after rule change should not be cache hit")
		}
	})

	t.Run("clears cache manually", func(t *testing.T) {
		// Validate to populate cache
		v.Validate(ctx, "git status")

		// Clear cache
		v.ClearCache()

		// Should not be cache hit
		result := v.Validate(ctx, "git status")
		if result.CacheHit {
			t.Error("validation after cache clear should not be cache hit")
		}
	})

	t.Run("disabled cache", func(t *testing.T) {
		opts2 := DefaultValidatorOptions()
		opts2.EnableCache = false
		v2, _ := NewCommandValidator(opts2)
		defer v2.Close()

		v2.AddAllowRule("git *", "allow git")

		// Multiple validations should never be cache hits
		v2.Validate(ctx, "git status")
		result := v2.Validate(ctx, "git status")
		if result.CacheHit {
			t.Error("cache should be disabled")
		}
	})
}

// TestLogging tests validation logging
func TestLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "validation.log")

	opts := ValidatorOptions{
		EnableLogging:            true,
		LogPath:                  logPath,
		EnableCache:              false,
		ValidateCompoundCommands: true,
	}
	v, _ := NewCommandValidator(opts)
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git commands")
	v.AddDenyRule("rm -rf /", "prevent dangerous rm")

	t.Run("logs validation results", func(t *testing.T) {
		v.Validate(ctx, "git status")
		v.Validate(ctx, "rm -rf /")

		// Force log flush by closing and reopening
		v.Close()

		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logStr := string(content)
		if !strings.Contains(logStr, "VALIDATION") {
			t.Error("expected log to contain VALIDATION entries")
		}
		if !strings.Contains(logStr, "git status") {
			t.Error("expected log to contain validated command")
		}
		if !strings.Contains(logStr, "ALLOWED") {
			t.Error("expected log to contain ALLOWED status")
		}
		if !strings.Contains(logStr, "DENIED") {
			t.Error("expected log to contain DENIED status")
		}
	})

	t.Run("logs rule changes", func(t *testing.T) {
		v.AddAllowRule("test-pattern", "test")
		v.RemoveRule("test-pattern")
		v.ClearRules()

		// Force log flush
		v.Close()

		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		logStr := string(content)
		if !strings.Contains(logStr, "ADD_ALLOW_RULE") {
			t.Error("expected log to contain ADD_ALLOW_RULE")
		}
		if !strings.Contains(logStr, "REMOVE_RULE") {
			t.Error("expected log to contain REMOVE_RULE")
		}
		if !strings.Contains(logStr, "CLEAR_RULES") {
			t.Error("expected log to contain CLEAR_RULES")
		}
	})
}

// TestStrictMode tests strict mode validation
func TestStrictMode(t *testing.T) {
	opts := DefaultValidatorOptions()
	opts.StrictMode = true
	opts.ValidateCompoundCommands = true
	v, _ := NewCommandValidator(opts)
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")
	v.AddAllowRule("docker *", "allow docker")

	t.Run("requires all segments allowed in strict mode", func(t *testing.T) {
		// Only git is allowed, not npm
		result := v.Validate(ctx, "git status && npm install")
		if result.Allowed {
			t.Error("expected command with unallowed segment to be denied in strict mode")
		}
		if !strings.Contains(result.Reason, "strict mode") {
			t.Errorf("expected reason to mention strict mode, got: %s", result.Reason)
		}
	})

	t.Run("allows when all segments allowed", func(t *testing.T) {
		result := v.Validate(ctx, "git status && docker ps")
		if !result.Allowed {
			t.Errorf("expected allowed command, got: %s", result.Reason)
		}
	})
}

// TestCommandNormalization tests command normalization
func TestCommandNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  git   status  ", "git status"},
		{"git\tstatus", "git status"},
		{"git\nstatus", "git status"},
		{"git  status  --short", "git status --short"},
		{"GIT STATUS", "GIT STATUS"}, // Case preserved, matching is case-insensitive
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("normalize_%q", tt.input), func(t *testing.T) {
			opts := DefaultValidatorOptions()
			v, _ := NewCommandValidator(opts)
			defer v.Close()

			ctx := context.Background()
			v.AddAllowRule("git status*", "allow git status")

			result := v.Validate(ctx, tt.input)
			if !result.Allowed {
				t.Errorf("expected normalized command %q to be allowed, got: %s", tt.input, result.Reason)
			}
			if result.NormalizedCommand != tt.expected {
				t.Errorf("expected normalized=%q, got %q", tt.expected, result.NormalizedCommand)
			}
		})
	}
}

// TestCommandLengthLimit tests maximum command length enforcement
func TestCommandLengthLimit(t *testing.T) {
	opts := DefaultValidatorOptions()
	opts.MaxCommandLength = 50
	v, _ := NewCommandValidator(opts)
	defer v.Close()

	ctx := context.Background()

	t.Run("allows short command", func(t *testing.T) {
		v.AddAllowRule("git status", "allow")
		result := v.Validate(ctx, "git status")
		if !result.Allowed {
			t.Errorf("expected short command to be allowed, got: %s", result.Reason)
		}
	})

	t.Run("denies long command", func(t *testing.T) {
		longCommand := strings.Repeat("a", 51)
		result := v.Validate(ctx, longCommand)
		if result.Allowed {
			t.Error("expected long command to be denied")
		}
		if !strings.Contains(result.Reason, "exceeds maximum length") {
			t.Errorf("expected length error, got: %s", result.Reason)
		}
	})
}

// TestValidateBatch tests batch validation
func TestValidateBatch(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")
	v.AddAllowRule("docker *", "allow docker")
	v.AddDenyRule("rm -rf /", "deny dangerous")

	commands := []string{
		"git status",
		"docker ps",
		"rm -rf /",
		"unknown command",
	}

	results := v.ValidateBatch(ctx, commands)

	if len(results) != len(commands) {
		t.Fatalf("expected %d results, got %d", len(commands), len(results))
	}

	expected := []bool{true, true, false, false}
	for i, exp := range expected {
		if results[i].Allowed != exp {
			t.Errorf("command %d: expected allowed=%v, got %v (reason: %s)",
				i, exp, results[i].Allowed, results[i].Reason)
		}
	}
}

// TestLoadFromPermissionSet tests loading rules from permission set
func TestLoadFromPermissionSet(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	set := PermissionSet{
		Allow: []string{"git *", "docker *"},
		Deny:  []string{"rm -rf /", "sudo *"},
	}

	err := v.LoadFromPermissionSet(set)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	allow, deny := v.GetRules()
	if len(allow) != 2 {
		t.Errorf("expected 2 allow rules, got %d", len(allow))
	}
	if len(deny) != 2 {
		t.Errorf("expected 2 deny rules, got %d", len(deny))
	}

	// Test that rules work
	tests := []struct {
		command string
		allowed bool
	}{
		{"git status", true},
		{"docker ps", true},
		{"rm -rf /", false},
		{"sudo ls", false},
		{"npm install", false},
	}

	for _, tt := range tests {
		result := v.Validate(ctx, tt.command)
		if result.Allowed != tt.allowed {
			t.Errorf("command %q: expected allowed=%v, got %v",
				tt.command, tt.allowed, result.Allowed)
		}
	}
}

// TestIsAllowed tests the convenience method
func TestIsAllowed(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")

	if !v.IsAllowed(ctx, "git status") {
		t.Error("expected IsAllowed to return true for allowed command")
	}

	if v.IsAllowed(ctx, "docker ps") {
		t.Error("expected IsAllowed to return false for unallowed command")
	}
}

// TestConcurrency tests thread safety
func TestConcurrency(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	// Add initial rules
	v.AddAllowRule("git *", "allow git")
	v.AddDenyRule("rm -rf /", "deny dangerous")

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent validations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			command := fmt.Sprintf("git status %d", n)
			result := v.Validate(ctx, command)
			if !result.Allowed {
				t.Errorf("goroutine %d: expected allowed, got: %s", n, result.Reason)
			}
		}(i)
	}

	// Concurrent rule additions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pattern := fmt.Sprintf("test-pattern-%d", n)
			v.AddAllowRule(pattern, "test")
		}(i)
	}

	wg.Wait()

	// Verify final state
	allow, _ := v.GetRules()
	if len(allow) < 11 { // 1 initial + 10 added
		t.Errorf("expected at least 11 allow rules, got %d", len(allow))
	}
}

// TestClearRules tests clearing all rules
func TestClearRules(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	v.AddAllowRule("git *", "allow git")
	v.AddDenyRule("rm -rf /", "deny rm")

	v.ClearRules()

	allow, deny := v.GetRules()
	if len(allow) != 0 {
		t.Errorf("expected 0 allow rules, got %d", len(allow))
	}
	if len(deny) != 0 {
		t.Errorf("expected 0 deny rules, got %d", len(deny))
	}
}

// TestValidationEdgeCases tests various edge cases
func TestValidationEdgeCases(t *testing.T) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	t.Run("empty command", func(t *testing.T) {
		v.AddAllowRule("*", "allow everything")
		result := v.Validate(ctx, "")
		// Empty command should be handled gracefully
		if !result.Allowed {
			t.Logf("Empty command result: %s", result.Reason)
		}
	})

	t.Run("only whitespace", func(t *testing.T) {
		v.ClearRules()
		v.AddAllowRule("*", "allow everything")
		result := v.Validate(ctx, "   ")
		// Whitespace-only command should be handled gracefully
		if !result.Allowed {
			t.Logf("Whitespace command result: %s", result.Reason)
		}
	})

	t.Run("special characters in command", func(t *testing.T) {
		v.ClearRules()
		v.AddAllowRule("echo *", "allow echo")
		result := v.Validate(ctx, "echo 'hello world'")
		if !result.Allowed {
			t.Errorf("expected echo with quotes to be allowed, got: %s", result.Reason)
		}
	})

	t.Run("unicode in command", func(t *testing.T) {
		v.ClearRules()
		v.AddAllowRule("echo *", "allow echo")
		result := v.Validate(ctx, "echo こんにちは")
		if !result.Allowed {
			t.Errorf("expected echo with unicode to be allowed, got: %s", result.Reason)
		}
	})
}

// TestGlobToRegex tests the glob to regex conversion
func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		match   bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"?", "a", true},
		{"?", "ab", false},
		{"test*", "testing", true},
		{"test*", "test", true},
		{"test?", "test1", true},
		{"test[0-9]", "test5", true},
		{"test[0-9]", "testa", false},
		{"file.{txt,md}", "file.txt", true},
		{"file.{txt,md}", "file.md", true},
		{"file.{txt,md}", "file.go", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("glob_%s", tt.pattern), func(t *testing.T) {
			regex := globToRegex(tt.pattern)
			matched, err := matchRegex(regex, tt.input)
			if err != nil {
				t.Fatalf("regex error: %v", err)
			}
			if matched != tt.match {
				t.Errorf("pattern=%q input=%q expected=%v got=%v (regex=%s)",
					tt.pattern, tt.input, tt.match, matched, regex)
			}
		})
	}
}

// TestIsValidGlobPattern tests glob pattern validation
func TestIsValidGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		valid   bool
	}{
		{"*", true},
		{"test*", true},
		{"{a,b}", true},
		{"{a,b,c}", true},
		{"{a,{b,c}}", true},
		{"{invalid", false},
		{"invalid}", false},
		{"{a,b}{c,d}", true},
		{"test[abc]", true},
		{"", true}, // Empty is technically valid
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("pattern_%s", tt.pattern), func(t *testing.T) {
			valid := isValidGlobPattern(tt.pattern)
			if valid != tt.valid {
				t.Errorf("pattern=%q expected valid=%v got=%v",
					tt.pattern, tt.valid, valid)
			}
		})
	}
}

// Helper function to match regex
func matchRegex(regex, input string) (bool, error) {
	re, err := regexp.Compile("(?i)^" + regex + "$")
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// BenchmarkValidate benchmarks command validation
func BenchmarkValidate(b *testing.B) {
	v, _ := NewCommandValidator(DefaultValidatorOptions())
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")
	v.AddDenyRule("rm -rf /", "deny dangerous")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(ctx, "git status")
	}
}

// BenchmarkValidateWithCache benchmarks cached validation
func BenchmarkValidateWithCache(b *testing.B) {
	opts := DefaultValidatorOptions()
	opts.EnableCache = true
	v, _ := NewCommandValidator(opts)
	defer v.Close()

	ctx := context.Background()

	v.AddAllowRule("git *", "allow git")

	// Pre-populate cache
	v.Validate(ctx, "git status")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(ctx, "git status")
	}
}

// BenchmarkGlobMatching benchmarks glob pattern matching
func BenchmarkGlobMatching(b *testing.B) {
	pattern := "git {status,clone,pull,push} *"
	input := "git status --short"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchGlob(pattern, input)
	}
}