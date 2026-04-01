package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPolicyLevelString tests the PolicyLevel String method
func TestPolicyLevelString(t *testing.T) {
	tests := []struct {
		level    PolicyLevel
		expected string
	}{
		{PolicyLevelSystem, "system"},
		{PolicyLevelOrganization, "organization"},
		{PolicyLevelTeam, "team"},
		{PolicyLevelUser, "user"},
		{PolicyLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

// TestNewPolicyEnforcer tests policy enforcer creation
func TestNewPolicyEnforcer(t *testing.T) {
	t.Run("creates enforcer with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		require.NotNil(t, enforcer)

		defer enforcer.Stop()

		assert.NotEmpty(t, enforcer.GetPolicyDir())
		assert.False(t, enforcer.IsEnforcementEnabled())
		assert.Empty(t, enforcer.ListPolicies())
	})

	t.Run("creates policy directory if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "new", "policy", "dir")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir:          policyDir,
			EnforcementEnabled: true,
		})
		require.NoError(t, err)
		require.NotNil(t, enforcer)

		defer enforcer.Stop()

		// Directory should be created
		_, err = os.Stat(policyDir)
		assert.NoError(t, err)
		assert.True(t, enforcer.IsEnforcementEnabled())
	})

	t.Run("uses custom logger", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		var buf strings.Builder
		logger := log.New(&buf, "[TEST] ", log.LstdFlags)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
			Logger:    logger,
		})
		require.NoError(t, err)
		require.NotNil(t, enforcer)

		defer enforcer.Stop()

		// Logger should be set (we can't easily test output without operations)
		assert.NotNil(t, enforcer.logger)
	})

	t.Run("loads existing policies", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a policy file
		policy := &Policy{
			Name:        "test-policy",
			Level:       PolicyLevelOrganization,
			Description: "Test policy",
			Enabled:     true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"claude-3-sonnet"},
				DeniedModels:  []string{"gpt-4"},
			},
		}
		writePolicyFile(t, policyDir, "test-policy.json", policy)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should have loaded the policy
		policies := enforcer.ListPolicies()
		assert.Contains(t, policies, "test-policy")

		loaded, ok := enforcer.GetPolicy("test-policy")
		require.True(t, ok)
		assert.Equal(t, "test-policy", loaded.Name)
		assert.Equal(t, PolicyLevelOrganization, loaded.Level)
	})

	t.Run("ignores disabled policies", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a disabled policy
		policy := &Policy{
			Name:    "disabled-policy",
			Level:   PolicyLevelUser,
			Enabled: false,
		}
		writePolicyFile(t, policyDir, "disabled-policy.json", policy)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should not have loaded the disabled policy
		policies := enforcer.ListPolicies()
		assert.NotContains(t, policies, "disabled-policy")
	})

	t.Run("ignores non-JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a non-JSON file
		err := os.WriteFile(filepath.Join(policyDir, "readme.txt"), []byte("not a policy"), 0644)
		require.NoError(t, err)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should have no policies
		assert.Empty(t, enforcer.ListPolicies())
	})
}

// TestLoadPolicies tests policy loading
func TestLoadPolicies(t *testing.T) {
	t.Run("handles missing policy directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "nonexistent")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should create the directory
		_, err = os.Stat(policyDir)
		assert.NoError(t, err)
	})

	t.Run("handles corrupted policy file", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create an invalid JSON file
		err := os.WriteFile(filepath.Join(policyDir, "corrupt.json"), []byte("not valid json"), 0644)
		require.NoError(t, err)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should not crash, just skip the invalid file
		assert.Empty(t, enforcer.ListPolicies())
	})

	t.Run("handles policy without name", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a policy without a name
		policy := map[string]interface{}{
			"level":   PolicyLevelUser,
			"enabled": true,
		}
		data, _ := json.Marshal(policy)
		err := os.WriteFile(filepath.Join(policyDir, "noname.json"), data, 0644)
		require.NoError(t, err)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should not load the invalid policy
		assert.Empty(t, enforcer.ListPolicies())
	})

	t.Run("reloads policies", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Initially empty
		assert.Empty(t, enforcer.ListPolicies())

		// Add a policy file
		policy := &Policy{
			Name:    "new-policy",
			Level:   PolicyLevelUser,
			Enabled: true,
		}
		writePolicyFile(t, policyDir, "new-policy.json", policy)

		// Reload
		err = enforcer.Reload()
		require.NoError(t, err)

		// Should now have the policy
		assert.Contains(t, enforcer.ListPolicies(), "new-policy")
	})
}

// TestSavePolicy tests saving policies
func TestSavePolicy(t *testing.T) {
	t.Run("saves valid policy", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		policy := &Policy{
			Name:        "test-save",
			Level:       PolicyLevelTeam,
			Description: "Test saving",
			Enabled:     true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model1", "model2"},
			},
		}

		err = enforcer.SavePolicy(policy)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(filepath.Join(policyDir, "test-save.json"))
		assert.NoError(t, err)

		// Verify it's in memory
		loaded, ok := enforcer.GetPolicy("test-save")
		require.True(t, ok)
		assert.Equal(t, "test-save", loaded.Name)
	})

	t.Run("rejects policy without name", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		policy := &Policy{
			Level:   PolicyLevelUser,
			Enabled: true,
		}

		err = enforcer.SavePolicy(policy)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy name is required")
	})

	t.Run("sets timestamps", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		beforeSave := time.Now().Add(-time.Second)
		policy := &Policy{
			Name:    "timestamp-test",
			Enabled: true,
		}

		err = enforcer.SavePolicy(policy)
		require.NoError(t, err)

		loaded, ok := enforcer.GetPolicy("timestamp-test")
		require.True(t, ok)
		assert.False(t, loaded.CreatedAt.IsZero())
		assert.False(t, loaded.UpdatedAt.IsZero())
		assert.True(t, loaded.CreatedAt.After(beforeSave))
	})
}

// TestDeletePolicy tests policy deletion
func TestDeletePolicy(t *testing.T) {
	t.Run("deletes existing policy", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a policy
		policy := &Policy{
			Name:    "to-delete",
			Level:   PolicyLevelUser,
			Enabled: true,
		}
		writePolicyFile(t, policyDir, "to-delete.json", policy)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Verify it exists
		_, ok := enforcer.GetPolicy("to-delete")
		assert.True(t, ok)

		// Delete it
		err = enforcer.DeletePolicy("to-delete")
		require.NoError(t, err)

		// Verify it's gone
		_, ok = enforcer.GetPolicy("to-delete")
		assert.False(t, ok)

		// Verify file is deleted
		_, err = os.Stat(filepath.Join(policyDir, "to-delete.json"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("handles non-existent policy", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should not error on non-existent policy
		err = enforcer.DeletePolicy("does-not-exist")
		assert.NoError(t, err)
	})
}

// TestValidateModel tests model validation
func TestValidateModel(t *testing.T) {
	t.Run("allows model in allowed list", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels:  []string{"claude-3-sonnet", "claude-3-opus"},
					DeniedModels:   []string{},
					AllowByDefault: false,
				},
			},
		})

		violation := enforcer.ValidateModel("claude-3-sonnet")
		assert.Nil(t, violation)
	})

	t.Run("denies model in denied list", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"claude-3-sonnet"},
					DeniedModels:  []string{"gpt-4"},
				},
			},
		})

		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeDeniedModel, violation.Type)
		assert.Equal(t, "gpt-4", violation.Resource)
	})

	t.Run("denies model not in allowed list", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"claude-3-sonnet"},
					DeniedModels:  []string{},
				},
			},
		})

		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeNotAllowedModel, violation.Type)
	})

	t.Run("allows any model when allowed list is empty and allow by default", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels:  []string{},
					DeniedModels:   []string{},
					AllowByDefault: true,
				},
			},
		})

		violation := enforcer.ValidateModel("any-model")
		assert.Nil(t, violation)
	})

	t.Run("denies all when allowed list is empty and not allow by default", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels:  []string{},
					DeniedModels:   []string{},
					AllowByDefault: false,
				},
			},
		})

		violation := enforcer.ValidateModel("any-model")
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeNotAllowedModel, violation.Type)
	})

	t.Run("denied list takes precedence over allowed list", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "model-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"gpt-4", "claude-3"},
					DeniedModels:  []string{"gpt-4"},
				},
			},
		})

		// Even though gpt-4 is in allowed list, it should be denied
		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeDeniedModel, violation.Type)
	})

	t.Run("ignores disabled policies", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "disabled-policy",
				Level:   PolicyLevelOrganization,
				Enabled: false,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
			},
		})

		// Should be allowed since policy is disabled
		violation := enforcer.ValidateModel("gpt-4")
		assert.Nil(t, violation)
	})

	t.Run("supports wildcard patterns", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "wildcard-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-*"},
				},
			},
		})

		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)
	})

	t.Run("checks multiple policies", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "policy1",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"claude-3"},
				},
			},
			{
				Name:    "policy2",
				Level:   PolicyLevelTeam,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
			},
		})

		// Should pass policy1 but fail policy2
		violation := enforcer.ValidateModel("gpt-4")
		// gpt-4 is not in allowed list of policy1, so it should fail there first
		require.NotNil(t, violation)
	})
}

// TestValidateKeyAge tests key age validation
func TestValidateKeyAge(t *testing.T) {
	t.Run("allows key within age limit", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "key-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					MaxKeyAge:        30 * 24 * time.Hour, // 30 days
					RequireRotation:  true,
					AllowedProviders: []string{"anthropic"},
				},
			},
		})

		// Key created 10 days ago
		keyCreatedAt := time.Now().Add(-10 * 24 * time.Hour)
		violation := enforcer.ValidateKeyAge("anthropic", keyCreatedAt)
		assert.Nil(t, violation)
	})

	t.Run("denies expired key", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "key-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					MaxKeyAge:        30 * 24 * time.Hour, // 30 days
					RequireRotation:  true,
					AllowedProviders: []string{"anthropic"},
				},
			},
		})

		// Key created 60 days ago
		keyCreatedAt := time.Now().Add(-60 * 24 * time.Hour)
		violation := enforcer.ValidateKeyAge("anthropic", keyCreatedAt)
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeKeyExpired, violation.Type)
		assert.Equal(t, "anthropic", violation.Resource)
		assert.Contains(t, violation.Details, "key_age")
		assert.Contains(t, violation.Details, "require_rotation")
	})

	t.Run("allows any age when no max age set", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "key-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					MaxKeyAge: 0, // No limit
				},
			},
		})

		// Key created 1 year ago
		keyCreatedAt := time.Now().Add(-365 * 24 * time.Hour)
		violation := enforcer.ValidateKeyAge("anthropic", keyCreatedAt)
		assert.Nil(t, violation)
	})

	t.Run("denies provider in denied list", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "key-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					DeniedProviders: []string{"openai"},
				},
			},
		})

		violation := enforcer.ValidateKeyAge("openai", time.Now())
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeInvalidConfiguration, violation.Type)
	})

	t.Run("skips policy for non-allowed provider", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "key-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					MaxKeyAge:        30 * 24 * time.Hour,
					AllowedProviders: []string{"anthropic"}, // Only anthropic checked
				},
			},
		})

		// OpenAI key is very old but not checked by this policy
		keyCreatedAt := time.Now().Add(-365 * 24 * time.Hour)
		violation := enforcer.ValidateKeyAge("openai", keyCreatedAt)
		assert.Nil(t, violation)
	})
}

// TestCheckQuota tests quota validation
func TestCheckQuota(t *testing.T) {
	t.Run("allows usage under quota", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxRequestsPerDay:  100,
					MaxTokensPerDay:    1000000,
					MaxCostPerDay:      50.0,
					MaxConcurrentTasks: 5,
				},
			},
		})

		usage := UsageMetrics{
			RequestsToday:   50,
			TokensToday:     500000,
			CostToday:       25.0,
			ConcurrentTasks: 3,
		}

		violation := enforcer.CheckQuota(usage)
		assert.Nil(t, violation)
	})

	t.Run("denies when requests exceeded", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxRequestsPerDay: 100,
				},
			},
		})

		usage := UsageMetrics{
			RequestsToday: 100, // At limit
		}

		violation := enforcer.CheckQuota(usage)
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeQuotaExceeded, violation.Type)
		assert.Contains(t, violation.Details["quota_type"], "requests_per_day")
	})

	t.Run("denies when tokens exceeded", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxTokensPerDay: 1000000,
				},
			},
		})

		usage := UsageMetrics{
			TokensToday: 1000000, // At limit
		}

		violation := enforcer.CheckQuota(usage)
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeQuotaExceeded, violation.Type)
		assert.Contains(t, violation.Details["quota_type"], "tokens_per_day")
	})

	t.Run("denies when cost exceeded", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxCostPerDay: 50.0,
				},
			},
		})

		usage := UsageMetrics{
			CostToday: 50.0, // At limit
		}

		violation := enforcer.CheckQuota(usage)
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeQuotaExceeded, violation.Type)
		assert.Contains(t, violation.Details["quota_type"], "cost_per_day")
	})

	t.Run("denies when concurrent tasks exceeded", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxConcurrentTasks: 5,
				},
			},
		})

		usage := UsageMetrics{
			ConcurrentTasks: 5, // At limit
		}

		violation := enforcer.CheckQuota(usage)
		require.NotNil(t, violation)
		assert.Equal(t, ViolationTypeQuotaExceeded, violation.Type)
		assert.Contains(t, violation.Details["quota_type"], "concurrent_tasks")
	})

	t.Run("ignores disabled quotas", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "quota-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				QuotaPolicy: &QuotaPolicy{
					MaxRequestsPerDay: 0, // Disabled
					MaxTokensPerDay:   0, // Disabled
				},
			},
		})

		usage := UsageMetrics{
			RequestsToday: 1000000,
			TokensToday:   1000000000,
		}

		violation := enforcer.CheckQuota(usage)
		assert.Nil(t, violation)
	})
}

// TestValidateConfiguration tests configuration validation
func TestValidateConfiguration(t *testing.T) {
	t.Run("validates model in config", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "config-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"claude-3-sonnet"},
				},
			},
		})

		tmpDir := t.TempDir()
		config, err := NewLayeredConfig(ConfigOptions{
			BaseDir: tmpDir,
		})
		require.NoError(t, err)

		config.SetCLIFlag("model", "gpt-4")

		violations := enforcer.ValidateConfiguration(config)
		assert.Len(t, violations, 1)
		assert.Equal(t, ViolationTypeNotAllowedModel, violations[0].Type)
	})

	t.Run("validates provider in config", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "config-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					AllowedProviders: []string{"anthropic"},
				},
			},
		})

		tmpDir := t.TempDir()
		config, err := NewLayeredConfig(ConfigOptions{
			BaseDir: tmpDir,
		})
		require.NoError(t, err)

		config.SetCLIFlag("api.provider", "openai")

		violations := enforcer.ValidateConfiguration(config)
		assert.Len(t, violations, 1)
		assert.Equal(t, ViolationTypeInvalidConfiguration, violations[0].Type)
	})

	t.Run("detects denied provider", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "config-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				KeyPolicy: &KeyPolicy{
					DeniedProviders: []string{"openai"},
				},
			},
		})

		tmpDir := t.TempDir()
		config, err := NewLayeredConfig(ConfigOptions{
			BaseDir: tmpDir,
		})
		require.NoError(t, err)

		config.SetCLIFlag("api.provider", "openai")

		violations := enforcer.ValidateConfiguration(config)
		assert.Len(t, violations, 1)
		assert.Equal(t, ViolationTypeInvalidConfiguration, violations[0].Type)
	})

	t.Run("returns multiple violations", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "config-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
				KeyPolicy: &KeyPolicy{
					DeniedProviders: []string{"openai"},
				},
			},
		})

		tmpDir := t.TempDir()
		config, err := NewLayeredConfig(ConfigOptions{
			BaseDir: tmpDir,
		})
		require.NoError(t, err)

		config.SetCLIFlag("model", "gpt-4")
		config.SetCLIFlag("api.provider", "openai")

		violations := enforcer.ValidateConfiguration(config)
		assert.GreaterOrEqual(t, len(violations), 1) // At least model violation
	})
}

// TestEnforce tests the enforcement mechanism
func TestEnforce(t *testing.T) {
	t.Run("allows operation when no violation", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{})

		err := enforcer.Enforce(nil)
		assert.NoError(t, err)
	})

	t.Run("logs but allows when enforcement disabled", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{})

		violation := &PolicyViolation{
			Type:       ViolationTypeDeniedModel,
			Message:    "Test violation",
			Level:      PolicyLevelOrganization,
			Timestamp:  time.Now(),
			PolicyName: "test-policy",
		}

		err := enforcer.Enforce(violation)
		assert.NoError(t, err) // Should not block when enforcement is disabled
	})

	t.Run("blocks when enforcement enabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir:          policyDir,
			EnforcementEnabled: true,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		violation := &PolicyViolation{
			Type:       ViolationTypeDeniedModel,
			Message:    "Test violation",
			Level:      PolicyLevelOrganization,
			Timestamp:  time.Now(),
			PolicyName: "test-policy",
		}

		err = enforcer.Enforce(violation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy enforcement blocked operation")
	})
}

// TestPolicyInheritance tests policy inheritance
func TestPolicyInheritance(t *testing.T) {
	t.Run("inherits from parent policy", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create parent policy
		parent := &Policy{
			Name:    "parent",
			Level:   PolicyLevelOrganization,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels:  []string{"claude-3"},
				DeniedModels:   []string{"gpt-3"},
				AllowByDefault: true,
			},
			KeyPolicy: &KeyPolicy{
				MaxKeyAge:        30 * 24 * time.Hour,
				RequireRotation:  true,
				AllowedProviders: []string{"anthropic", "openai"},
			},
			QuotaPolicy: &QuotaPolicy{
				MaxRequestsPerDay: 100,
				MaxTokensPerDay:   1000000,
			},
		}
		writePolicyFile(t, policyDir, "parent.json", parent)

		// Create child policy that inherits from parent
		child := &Policy{
			Name:         "child",
			Level:        PolicyLevelTeam,
			Enabled:      true,
			InheritsFrom: []string{"parent"},
			ModelPolicy: &ModelPolicy{
				// Child adds to denied models
				DeniedModels: []string{"gpt-4"},
			},
			// KeyPolicy and QuotaPolicy inherited from parent
		}
		writePolicyFile(t, policyDir, "child.json", child)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Get effective policy
		effective, err := enforcer.GetEffectivePolicy("child")
		require.NoError(t, err)

		// Child's denied models should be merged with parent's
		assert.Contains(t, effective.ModelPolicy.DeniedModels, "gpt-3")
		assert.Contains(t, effective.ModelPolicy.DeniedModels, "gpt-4")

		// Parent's allowed models should be inherited
		assert.Contains(t, effective.ModelPolicy.AllowedModels, "claude-3")

		// Parent's key policy should be inherited
		assert.Equal(t, 30*24*time.Hour, effective.KeyPolicy.MaxKeyAge)
		assert.True(t, effective.KeyPolicy.RequireRotation)

		// Parent's quota policy should be inherited
		assert.Equal(t, 100, effective.QuotaPolicy.MaxRequestsPerDay)
	})

	t.Run("child values take precedence over parent", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create parent policy
		parent := &Policy{
			Name:    "parent",
			Level:   PolicyLevelOrganization,
			Enabled: true,
			QuotaPolicy: &QuotaPolicy{
				MaxRequestsPerDay: 100,
			},
		}
		writePolicyFile(t, policyDir, "parent.json", parent)

		// Create child policy that overrides parent value
		child := &Policy{
			Name:         "child",
			Level:        PolicyLevelTeam,
			Enabled:      true,
			InheritsFrom: []string{"parent"},
			QuotaPolicy: &QuotaPolicy{
				MaxRequestsPerDay: 200, // Override parent's 100
			},
		}
		writePolicyFile(t, policyDir, "child.json", child)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		effective, err := enforcer.GetEffectivePolicy("child")
		require.NoError(t, err)

		// Child's value should take precedence
		assert.Equal(t, 200, effective.QuotaPolicy.MaxRequestsPerDay)
	})

	t.Run("detects circular inheritance", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create circular dependency: A -> B -> C -> A
		policyA := &Policy{
			Name:         "policy-a",
			Level:        PolicyLevelOrganization,
			Enabled:      true,
			InheritsFrom: []string{"policy-b"},
		}
		writePolicyFile(t, policyDir, "policy-a.json", policyA)

		policyB := &Policy{
			Name:         "policy-b",
			Level:        PolicyLevelTeam,
			Enabled:      true,
			InheritsFrom: []string{"policy-c"},
		}
		writePolicyFile(t, policyDir, "policy-b.json", policyB)

		policyC := &Policy{
			Name:         "policy-c",
			Level:        PolicyLevelUser,
			Enabled:      true,
			InheritsFrom: []string{"policy-a"},
		}
		writePolicyFile(t, policyDir, "policy-c.json", policyC)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Should detect circular inheritance
		_, err = enforcer.GetEffectivePolicy("policy-a")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular inheritance")
	})

	t.Run("handles missing parent policy", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		child := &Policy{
			Name:         "orphan",
			Level:        PolicyLevelUser,
			Enabled:      true,
			InheritsFrom: []string{"nonexistent"},
		}
		writePolicyFile(t, policyDir, "orphan.json", child)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		_, err = enforcer.GetEffectivePolicy("orphan")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parent policy not found")
	})

	t.Run("supports multi-level inheritance", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Grandparent -> Parent -> Child
		grandparent := &Policy{
			Name:    "grandparent",
			Level:   PolicyLevelSystem,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model-a"},
			},
		}
		writePolicyFile(t, policyDir, "grandparent.json", grandparent)

		parent := &Policy{
			Name:         "parent",
			Level:        PolicyLevelOrganization,
			Enabled:      true,
			InheritsFrom: []string{"grandparent"},
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model-b"},
			},
		}
		writePolicyFile(t, policyDir, "parent.json", parent)

		child := &Policy{
			Name:         "child",
			Level:        PolicyLevelTeam,
			Enabled:      true,
			InheritsFrom: []string{"parent"},
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model-c"},
			},
		}
		writePolicyFile(t, policyDir, "child.json", child)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		effective, err := enforcer.GetEffectivePolicy("child")
		require.NoError(t, err)

		// All models should be merged
		assert.Contains(t, effective.ModelPolicy.AllowedModels, "model-a")
		assert.Contains(t, effective.ModelPolicy.AllowedModels, "model-b")
		assert.Contains(t, effective.ModelPolicy.AllowedModels, "model-c")
	})
}

// TestPolicyViolationRecording tests violation recording
func TestPolicyViolationRecording(t *testing.T) {
	t.Run("records violations", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "recording-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
			},
		})

		// Trigger a violation
		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)

		// Check recorded violations
		violations := enforcer.GetViolations()
		assert.Len(t, violations, 1)
		assert.Equal(t, "gpt-4", violations[0].Resource)
	})

	t.Run("returns violations since time", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "recording-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
			},
		})

		before := time.Now()

		// Trigger violations
		enforcer.ValidateModel("gpt-4")

		violations := enforcer.GetViolationsSince(before)
		assert.Len(t, violations, 1)

		// Should return empty for future time
		future := time.Now().Add(time.Hour)
		violations = enforcer.GetViolationsSince(future)
		assert.Empty(t, violations)
	})

	t.Run("clears violations", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "recording-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					DeniedModels: []string{"gpt-4"},
				},
			},
		})

		// Trigger a violation
		enforcer.ValidateModel("gpt-4")

		// Verify it was recorded
		assert.Len(t, enforcer.GetViolations(), 1)

		// Clear violations
		enforcer.ClearViolations()

		// Verify cleared
		assert.Empty(t, enforcer.GetViolations())
	})

	t.Run("calls onViolation callback", func(t *testing.T) {
		var called bool
		var receivedViolation PolicyViolation

		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create a policy that will be violated
		policy := &Policy{
			Name:    "callback-policy",
			Level:   PolicyLevelOrganization,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				DeniedModels: []string{"gpt-4"},
			},
		}
		writePolicyFile(t, policyDir, "callback-policy.json", policy)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
			OnViolation: func(v PolicyViolation) {
				called = true
				receivedViolation = v
			},
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Trigger a violation
		enforcer.ValidateModel("gpt-4")

		// Verify callback was called
		assert.True(t, called)
		assert.Equal(t, "gpt-4", receivedViolation.Resource)
	})
}

// TestWildcardMatching tests wildcard pattern matching
func TestWildcardMatching(t *testing.T) {
	enforcer := createTestEnforcer(t, []*Policy{})

	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"gpt-4", "gpt-4", true},
		{"gpt-4", "claude-3", false},
		{"gpt-*", "gpt-4", true},
		{"gpt-*", "gpt-4-turbo", true},
		{"gpt-*", "claude-3", false},
		{"claude-3-*", "claude-3-sonnet", true},
		{"claude-3-*", "claude-3-opus-20240229", true},
		{"claude-3-*", "claude-2", false},
		// Middle pattern tests - these test the actual implementation behavior
		{"claude*sonnet", "claude-3-sonnet", true},
		{"claude*sonnet", "claude-sonnet", true},
		{"claude*sonnet", "claude-3-opus", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			result := enforcer.matchesWildcard(tt.value, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEnforcementToggle tests enabling/disabling enforcement
func TestEnforcementToggle(t *testing.T) {
	t.Run("toggles enforcement", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir:          policyDir,
			EnforcementEnabled: false,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		assert.False(t, enforcer.IsEnforcementEnabled())

		enforcer.SetEnforcementEnabled(true)
		assert.True(t, enforcer.IsEnforcementEnabled())

		enforcer.SetEnforcementEnabled(false)
		assert.False(t, enforcer.IsEnforcementEnabled())
	})
}

// TestPolicyEquality tests policy comparison
func TestPolicyEquality(t *testing.T) {
	t.Run("equal policies", func(t *testing.T) {
		p1 := &Policy{
			Name:    "test",
			Level:   PolicyLevelOrganization,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model1"},
			},
		}

		p2 := &Policy{
			Name:    "test",
			Level:   PolicyLevelOrganization,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model1"},
			},
		}

		assert.True(t, p1.Equals(p2))
	})

	t.Run("different names", func(t *testing.T) {
		p1 := &Policy{Name: "test1", Enabled: true}
		p2 := &Policy{Name: "test2", Enabled: true}

		assert.False(t, p1.Equals(p2))
	})

	t.Run("different levels", func(t *testing.T) {
		p1 := &Policy{Name: "test", Level: PolicyLevelSystem, Enabled: true}
		p2 := &Policy{Name: "test", Level: PolicyLevelUser, Enabled: true}

		assert.False(t, p1.Equals(p2))
	})

	t.Run("different model policies", func(t *testing.T) {
		p1 := &Policy{
			Name:    "test",
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model1"},
			},
		}
		p2 := &Policy{
			Name:    "test",
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				AllowedModels: []string{"model2"},
			},
		}

		assert.False(t, p1.Equals(p2))
	})

	t.Run("nil comparison", func(t *testing.T) {
		p1 := &Policy{Name: "test"}
		var p2 *Policy

		assert.False(t, p1.Equals(p2))
		assert.False(t, p2.Equals(p1))
		assert.True(t, p2.Equals(nil))
	})
}

// TestMergeStringSlices tests string slice merging
func TestMergeStringSlices(t *testing.T) {
	tests := []struct {
		name       string
		target     []string
		source     []string
		sourceWins bool
		expected   []string
	}{
		{
			name:       "source wins",
			target:     []string{"a", "b"},
			source:     []string{"c", "d"},
			sourceWins: true,
			expected:   []string{"c", "d"},
		},
		{
			name:       "target wins with merge",
			target:     []string{"a", "b"},
			source:     []string{"b", "c"},
			sourceWins: false,
			expected:   []string{"a", "b", "c"},
		},
		{
			name:       "empty source",
			target:     []string{"a", "b"},
			source:     []string{},
			sourceWins: false,
			expected:   []string{"a", "b"},
		},
		{
			name:       "empty target",
			target:     []string{},
			source:     []string{"a", "b"},
			sourceWins: false,
			expected:   []string{"a", "b"},
		},
		{
			name:       "both empty",
			target:     []string{},
			source:     []string{},
			sourceWins: false,
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeStringSlices(tt.target, tt.source, tt.sourceWins)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// TestGetEffectivePolicy tests getting effective policy
func TestGetEffectivePolicy(t *testing.T) {
	t.Run("returns error for non-existent policy", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{})

		_, err := enforcer.GetEffectivePolicy("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})

	t.Run("returns cloned policy without inheritance", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:        "standalone",
				Level:       PolicyLevelUser,
				Enabled:     true,
				Description: "Test policy",
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"model1"},
				},
			},
		})

		effective, err := enforcer.GetEffectivePolicy("standalone")
		require.NoError(t, err)

		assert.Equal(t, "standalone", effective.Name)
		assert.Equal(t, PolicyLevelUser, effective.Level)
		assert.Equal(t, "Test policy", effective.Description)
		assert.Contains(t, effective.ModelPolicy.AllowedModels, "model1")
	})
}

// Helper functions

// createTestEnforcer creates a PolicyEnforcer with the given policies for testing
func createTestEnforcer(t *testing.T, policies []*Policy) *PolicyEnforcer {
	t.Helper()

	tmpDir := t.TempDir()
	policyDir := filepath.Join(tmpDir, "policies")
	require.NoError(t, os.MkdirAll(policyDir, 0755))

	for _, policy := range policies {
		writePolicyFile(t, policyDir, policy.Name+".json", policy)
	}

	enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
		PolicyDir: policyDir,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		enforcer.Stop()
	})

	return enforcer
}

// writePolicyFile writes a policy to a JSON file
func writePolicyFile(t *testing.T, dir, filename string, policy *Policy) {
	t.Helper()

	path := filepath.Join(dir, filename)
	data, err := json.MarshalIndent(policy, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(path, data, 0644)
	require.NoError(t, err)
}

// TestIntegration tests end-to-end policy enforcement
func TestIntegration(t *testing.T) {
	t.Run("full policy workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")

		// Create enforcer with enforcement enabled
		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir:          policyDir,
			EnforcementEnabled: true,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Create a comprehensive policy
		policy := &Policy{
			Name:        "enterprise-policy",
			Level:       PolicyLevelOrganization,
			Description: "Enterprise-wide policy",
			Enabled:     true,
			ModelPolicy: &ModelPolicy{
				AllowedModels:  []string{"claude-3-*", "gpt-4"},
				DeniedModels:   []string{"gpt-3-*"},
				AllowByDefault: false,
			},
			KeyPolicy: &KeyPolicy{
				MaxKeyAge:        90 * 24 * time.Hour,
				RequireRotation:  true,
				AllowedProviders: []string{"anthropic", "openai"},
				DeniedProviders:  []string{"openrouter"},
			},
			QuotaPolicy: &QuotaPolicy{
				MaxRequestsPerDay:  1000,
				MaxTokensPerDay:    10000000,
				MaxCostPerDay:      100.0,
				MaxConcurrentTasks: 10,
			},
		}

		// Save the policy
		err = enforcer.SavePolicy(policy)
		require.NoError(t, err)

		// Test model validation
		t.Run("model validation", func(t *testing.T) {
			// Allowed models
			assert.Nil(t, enforcer.ValidateModel("claude-3-sonnet"))
			assert.Nil(t, enforcer.ValidateModel("claude-3-opus"))
			assert.Nil(t, enforcer.ValidateModel("gpt-4"))

			// Denied models
			violation := enforcer.ValidateModel("gpt-3-turbo")
			require.NotNil(t, violation)
			assert.Equal(t, ViolationTypeDeniedModel, violation.Type)

			// Not allowed models
			violation = enforcer.ValidateModel("unknown-model")
			require.NotNil(t, violation)
			assert.Equal(t, ViolationTypeNotAllowedModel, violation.Type)
		})

		// Test key age validation
		t.Run("key age validation", func(t *testing.T) {
			// Valid key age
			recentKey := time.Now().Add(-30 * 24 * time.Hour)
			assert.Nil(t, enforcer.ValidateKeyAge("anthropic", recentKey))

			// Expired key
			oldKey := time.Now().Add(-120 * 24 * time.Hour)
			violation := enforcer.ValidateKeyAge("anthropic", oldKey)
			require.NotNil(t, violation)
			assert.Equal(t, ViolationTypeKeyExpired, violation.Type)
		})

		// Test quota validation
		t.Run("quota validation", func(t *testing.T) {
			// Valid usage
			validUsage := UsageMetrics{
				RequestsToday:   500,
				TokensToday:     5000000,
				CostToday:       50.0,
				ConcurrentTasks: 5,
			}
			assert.Nil(t, enforcer.CheckQuota(validUsage))

			// Exceeded requests
			exceededUsage := UsageMetrics{
				RequestsToday: 1000,
			}
			violation := enforcer.CheckQuota(exceededUsage)
			require.NotNil(t, violation)
			assert.Equal(t, ViolationTypeQuotaExceeded, violation.Type)
		})

		// Test enforcement
		t.Run("enforcement", func(t *testing.T) {
			violation := enforcer.ValidateModel("gpt-3-turbo")
			err := enforcer.Enforce(violation)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "policy enforcement blocked operation")
		})

		// Check violations were recorded
		t.Run("violation recording", func(t *testing.T) {
			violations := enforcer.GetViolations()
			assert.GreaterOrEqual(t, len(violations), 2) // At least gpt-3 and old key violations
		})
	})
}

// TestConcurrentAccess tests thread-safe operations
func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent policy reads", func(t *testing.T) {
		enforcer := createTestEnforcer(t, []*Policy{
			{
				Name:    "concurrent-policy",
				Level:   PolicyLevelOrganization,
				Enabled: true,
				ModelPolicy: &ModelPolicy{
					AllowedModels: []string{"model1", "model2", "model3"},
					DeniedModels:  []string{"banned"},
				},
			},
		})

		// Run concurrent validations
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					enforcer.ValidateModel("model1")
					enforcer.ValidateModel("model2")
					enforcer.ValidateModel("banned")
					enforcer.GetPolicy("concurrent-policy")
					enforcer.ListPolicies()
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should have recorded violations without race conditions
		violations := enforcer.GetViolations()
		assert.GreaterOrEqual(t, len(violations), 10) // At least one per goroutine
	})
}

// TestHotReload tests policy hot-reload functionality
func TestHotReload(t *testing.T) {
	t.Run("reloads policies manually", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Initially empty
		assert.Empty(t, enforcer.ListPolicies())

		// Add a new policy file
		policy := &Policy{
			Name:    "new-policy",
			Level:   PolicyLevelUser,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				DeniedModels: []string{"gpt-4"},
			},
		}
		writePolicyFile(t, policyDir, "new-policy.json", policy)

		// Reload
		err = enforcer.Reload()
		require.NoError(t, err)

		// Should now have the policy
		assert.Contains(t, enforcer.ListPolicies(), "new-policy")

		// Validate should work with new policy
		violation := enforcer.ValidateModel("gpt-4")
		require.NotNil(t, violation)
	})

	t.Run("reloads after policy deletion", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyDir := filepath.Join(tmpDir, "policies")
		require.NoError(t, os.MkdirAll(policyDir, 0755))

		// Create initial policy
		policy := &Policy{
			Name:    "to-remove",
			Level:   PolicyLevelUser,
			Enabled: true,
			ModelPolicy: &ModelPolicy{
				DeniedModels: []string{"gpt-4"},
			},
		}
		writePolicyFile(t, policyDir, "to-remove.json", policy)

		enforcer, err := NewPolicyEnforcer(PolicyEnforcerOptions{
			PolicyDir: policyDir,
		})
		require.NoError(t, err)
		defer enforcer.Stop()

		// Verify policy exists
		assert.Contains(t, enforcer.ListPolicies(), "to-remove")

		// Delete the file
		err = os.Remove(filepath.Join(policyDir, "to-remove.json"))
		require.NoError(t, err)

		// Reload
		err = enforcer.Reload()
		require.NoError(t, err)

		// Policy should be gone
		assert.NotContains(t, enforcer.ListPolicies(), "to-remove")
	})
}
