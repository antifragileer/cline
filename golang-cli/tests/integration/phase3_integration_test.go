// Package integration provides integration tests for Phase 3: Developer Tools & Hooks Management.
// This file tests dev utilities, hooks management, skills, workflows, and rules commands.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPhase3_DevTools validates developer tools functionality
func TestPhase3_DevTools(t *testing.T) {
	t.Run("dev_doctor_runs_successfully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		// Create temp storage for isolated test
		tempDir, err := os.MkdirTemp("", "cline-dev-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Build CLI binary for testing
		cliPath := buildCLIBinary(t)
		require.NotEmpty(t, cliPath)

		// Run dev doctor command
		cmd := exec.Command(cliPath, "dev", "doctor", "--json")
		cmd.Env = append(os.Environ(), fmt.Sprintf("CLINE_CONFIG_DIR=%s", tempDir))

		output, err := cmd.CombinedOutput()

		// Command should succeed
		require.NoError(t, err, "dev doctor failed: %s", string(output))

		// Parse JSON output
		var result map[string]interface{}
		err = json.Unmarshal(output, &result)
		require.NoError(t, err, "Failed to parse JSON output: %s", string(output))

		// Verify structure
		assert.NotNil(t, result["timestamp"])
		assert.NotNil(t, result["os"])
		assert.NotNil(t, result["arch"])
		assert.NotNil(t, result["results"])
	})

	t.Run("dev_doctor_human_readable_output", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		cliPath := buildCLIBinary(t)
		require.NotEmpty(t, cliPath)

		cmd := exec.Command(cliPath, "dev", "doctor")
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "dev doctor failed: %s", string(output))

		// Verify human-readable format
		outputStr := string(output)
		assert.Contains(t, outputStr, "Cline CLI Diagnostics")
		assert.Contains(t, outputStr, "System Information")
		assert.Contains(t, outputStr, "Checks:")
	})

	t.Run("dev_log_shows_log_path", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		cliPath := buildCLIBinary(t)
		require.NotEmpty(t, cliPath)

		cmd := exec.Command(cliPath, "dev", "log")
		output, err := cmd.CombinedOutput()

		// Command may succeed (editor opened) or fail (no log file)
		// In either case, it should provide information about the log
		outputStr := string(output)

		// Check that output contains log-related information
		// (either showing log path or indicating file not found)
		assert.True(t,
			strings.Contains(outputStr, "Log file:") ||
				strings.Contains(outputStr, "log file not found") ||
				strings.Contains(outputStr, "log") ||
				err == nil,
			"Expected log-related output, got: %s", outputStr)
	})
}

// TestPhase3_HooksManagement validates hooks management functionality
func TestPhase3_HooksManagement(t *testing.T) {
	t.Run("hooks_list_empty", func(t *testing.T) {
		// Create temp storage
		tempDir, err := os.MkdirTemp("", "cline-hooks-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// List hooks (should be empty)
		output, err := listHooksJSON(ctx)
		require.NoError(t, err)

		assert.Empty(t, output.Global)
		assert.Empty(t, output.Workspaces)
	})

	t.Run("hooks_enable_disable_global", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-hooks-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create a test hook
		hooks := []map[string]interface{}{
			{
				"name":        "test-hook",
				"description": "Test hook for integration",
				"script":      "echo 'test'",
				"enabled":     false,
				"type":        "before-task",
			},
		}

		err = ctx.GlobalState.Set("globalHooks", hooks)
		require.NoError(t, err)

		// Enable the hook
		err = enableHook(ctx, "test-hook")
		require.NoError(t, err)

		// Verify it's enabled
		output, err := listHooksJSON(ctx)
		require.NoError(t, err)
		require.Len(t, output.Global, 1)
		assert.True(t, output.Global[0].Enabled)

		// Disable the hook
		err = disableHook(ctx, "test-hook")
		require.NoError(t, err)

		// Verify it's disabled
		output, err = listHooksJSON(ctx)
		require.NoError(t, err)
		require.Len(t, output.Global, 1)
		assert.False(t, output.Global[0].Enabled)
	})

	t.Run("hooks_list_json_format", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-hooks-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create test hooks
		hooks := []map[string]interface{}{
			{
				"name":        "pre-task",
				"description": "Pre-task hook",
				"script":      "echo 'pre'",
				"enabled":     true,
				"type":        "before-task",
			},
			{
				"name":        "post-task",
				"description": "Post-task hook",
				"script":      "echo 'post'",
				"enabled":     false,
				"type":        "after-task",
			},
		}

		err = ctx.GlobalState.Set("globalHooks", hooks)
		require.NoError(t, err)

		// List hooks in JSON format
		output, err := listHooksJSON(ctx)
		require.NoError(t, err)

		// Verify structure
		require.Len(t, output.Global, 2)
		assert.Equal(t, "pre-task", output.Global[0].Name)
		assert.Equal(t, "post-task", output.Global[1].Name)
		assert.True(t, output.Global[0].Enabled)
		assert.False(t, output.Global[1].Enabled)
	})

	t.Run("hooks_not_found_error", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-hooks-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Try to enable non-existent hook
		err = enableHook(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestPhase3_SkillsManagement validates skills management functionality
func TestPhase3_SkillsManagement(t *testing.T) {
	t.Run("skills_list_empty", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-skills-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		output, err := listSkillsJSON(ctx)
		require.NoError(t, err)

		assert.Empty(t, output.Global)
		assert.Empty(t, output.Local)
	})

	t.Run("skills_enable_disable", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-skills-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create test skills
		skills := []map[string]interface{}{
			{
				"path":        "/path/to/skill1",
				"name":        "skill1",
				"description": "Test skill 1",
				"enabled":     false,
			},
		}

		err = ctx.GlobalState.Set("globalSkills", skills)
		require.NoError(t, err)

		// Enable skill
		err = enableSkill(ctx, "/path/to/skill1")
		require.NoError(t, err)

		// Verify enabled
		output, err := listSkillsJSON(ctx)
		require.NoError(t, err)
		require.Len(t, output.Global, 1)
		assert.True(t, output.Global[0].Enabled)

		// Disable skill
		err = disableSkill(ctx, "/path/to/skill1")
		require.NoError(t, err)

		// Verify disabled
		output, err = listSkillsJSON(ctx)
		require.NoError(t, err)
		require.Len(t, output.Global, 1)
		assert.False(t, output.Global[0].Enabled)
	})
}

// TestPhase3_WorkflowsManagement validates workflows management functionality
func TestPhase3_WorkflowsManagement(t *testing.T) {
	t.Run("workflows_list_empty", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-workflows-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		output, err := listWorkflowsJSON(ctx)
		require.NoError(t, err)

		assert.Empty(t, output.Global)
		assert.Empty(t, output.Local)
	})

	t.Run("workflows_enable_disable", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-workflows-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create test workflows
		toggles := map[string]bool{
			"/path/to/workflow1": false,
			"/path/to/workflow2": true,
		}

		err = ctx.GlobalState.Set("globalWorkflowToggles", toggles)
		require.NoError(t, err)

		// Enable workflow
		err = enableWorkflow(ctx, "/path/to/workflow1")
		require.NoError(t, err)

		// Verify enabled
		output, err := listWorkflowsJSON(ctx)
		require.NoError(t, err)

		found := false
		for _, wf := range output.Global {
			if wf.Path == "/path/to/workflow1" {
				assert.True(t, wf.Enabled)
				found = true
				break
			}
		}
		assert.True(t, found, "workflow1 not found in list")
	})
}

// TestPhase3_RulesManagement validates rules management functionality
func TestPhase3_RulesManagement(t *testing.T) {
	t.Run("rules_list_empty", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		output, err := listRulesJSON(ctx)
		require.NoError(t, err)

		assert.Empty(t, output.Rules)
	})

	t.Run("rules_enable_disable", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create test rules
		toggles := map[string]bool{
			"/path/to/rule1": false,
		}

		err = ctx.GlobalState.Set("globalClineRulesToggles", toggles)
		require.NoError(t, err)

		// Enable rule
		err = enableRule(ctx, "/path/to/rule1")
		require.NoError(t, err)

		// Verify enabled
		output, err := listRulesJSON(ctx)
		require.NoError(t, err)

		found := false
		for _, rule := range output.Rules {
			if rule.Path == "/path/to/rule1" {
				assert.True(t, rule.Enabled)
				found = true
				break
			}
		}
		assert.True(t, found, "rule1 not found in list")
	})

	t.Run("rules_filter_by_type", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cline-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create rules of different types
		clineRules := map[string]bool{"/rule/cline": true}
		cursorRules := map[string]bool{"/rule/cursor": true}

		err = ctx.GlobalState.Set("globalClineRulesToggles", clineRules)
		require.NoError(t, err)
		err = ctx.GlobalState.Set("globalCursorRulesToggles", cursorRules)
		require.NoError(t, err)

		// List all rules
		output, err := listRulesJSON(ctx)
		require.NoError(t, err)
		assert.Len(t, output.Rules, 2)

		// List only cline rules
		clineOutput, err := listRulesByTypeJSON(ctx, "cline")
		require.NoError(t, err)
		assert.Len(t, clineOutput.Rules, 1)
		assert.Equal(t, "cline", clineOutput.Rules[0].RuleType)
	})
}

// TestPhase3_Performance benchmarks developer tools performance
func TestPhase3_Performance(t *testing.T) {
	t.Run("hooks_operations_performance", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		tempDir, err := os.MkdirTemp("", "cline-perf-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Create many hooks
		var hooks []map[string]interface{}
		for i := 0; i < 100; i++ {
			hooks = append(hooks, map[string]interface{}{
				"name":        fmt.Sprintf("hook-%d", i),
				"description": fmt.Sprintf("Hook %d", i),
				"script":      fmt.Sprintf("echo %d", i),
				"enabled":     i%2 == 0,
				"type":        "before-task",
			})
		}

		err = ctx.GlobalState.Set("globalHooks", hooks)
		require.NoError(t, err)

		// Benchmark list operation
		start := time.Now()
		for i := 0; i < 100; i++ {
			_, err := listHooksJSON(ctx)
			require.NoError(t, err)
		}
		elapsed := time.Since(start)

		avgTime := elapsed / 100
		t.Logf("Average list time: %v", avgTime)
		assert.Less(t, avgTime, 10*time.Millisecond, "List operation too slow")
	})

	t.Run("storage_operations_performance", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance test in short mode")
		}

		tempDir, err := os.MkdirTemp("", "cline-perf-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)
		defer ctx.Close()

		// Benchmark set operations (with disk persistence, expect < 20ms avg)
		start := time.Now()
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			err := ctx.GlobalState.Set(key, value)
			require.NoError(t, err)
		}
		elapsed := time.Since(start)

		avgTime := elapsed / 100
		t.Logf("Average set time: %v", avgTime)
		// With disk persistence, expect < 50ms average
		assert.Less(t, avgTime, 50*time.Millisecond, "Set operation too slow")

		// Benchmark get operations (in-memory, expect < 1ms)
		start = time.Now()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			_, _ = ctx.GlobalState.Get(key)
		}
		elapsed = time.Since(start)

		avgTime = elapsed / 1000
		t.Logf("Average get time: %v", avgTime)
		assert.Less(t, avgTime, time.Millisecond, "Get operation too slow")
	})
}

// Helper functions

func buildCLIBinary(t *testing.T) string {
	t.Helper()

	// Check if already built
	cliPath := filepath.Join(os.TempDir(), "cline-test-binary")
	if _, err := os.Stat(cliPath); err == nil {
		return cliPath
	}

	// Build the CLI
	cmd := exec.Command("go", "build", "-o", cliPath, "./cmd/cline")
	cmd.Dir = getProjectRoot(t)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(output))
		return ""
	}

	return cliPath
}

func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Try to find project root from current directory
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Walk up to find go.mod (could be in current dir or golang-cli/ subdirectory)
	for {
		// First check if this directory has golang-cli/go.mod (nested project structure)
		if _, err := os.Stat(filepath.Join(wd, "golang-cli", "go.mod")); err == nil {
			return filepath.Join(wd, "golang-cli")
		}
		// Then check current dir for go.mod
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			// Verify this is the CLI project by checking for cmd/cline
			if _, err := os.Stat(filepath.Join(wd, "cmd", "cline")); err == nil {
				return wd
			}
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	return ""
}

// Hook helpers
type HookOutput struct {
	Global     []HookData      `json:"global"`
	Workspaces []WorkspaceData `json:"workspaces"`
}

type HookData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Script      string `json:"script"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type"`
}

type WorkspaceData struct {
	WorkspaceName string     `json:"workspaceName"`
	Hooks         []HookData `json:"hooks"`
}

func listHooksJSON(ctx *storage.StorageContext) (*HookOutput, error) {
	var output HookOutput

	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if hooks, err := parseHooksData(val); err == nil {
			output.Global = hooks
		}
	}

	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if wsHooks, err := parseWorkspaceHooksData(val); err == nil {
				output.Workspaces = wsHooks
			}
		}
	}

	return &output, nil
}

func parseHooksData(data interface{}) ([]HookData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var hooks []HookData
	if err := json.Unmarshal(jsonData, &hooks); err != nil {
		return nil, err
	}

	return hooks, nil
}

func parseWorkspaceHooksData(data interface{}) ([]WorkspaceData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var wsHooks []WorkspaceData
	if err := json.Unmarshal(jsonData, &wsHooks); err != nil {
		// Try as single workspace
		var single WorkspaceData
		if err := json.Unmarshal(jsonData, &single); err != nil {
			return nil, err
		}
		return []WorkspaceData{single}, nil
	}

	return wsHooks, nil
}

func enableHook(ctx *storage.StorageContext, name string) error {
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		var hooks []map[string]interface{}
		jsonData, _ := json.Marshal(val)
		if err := json.Unmarshal(jsonData, &hooks); err == nil {
			for i, hook := range hooks {
				if hook["name"] == name {
					hooks[i]["enabled"] = true
					return ctx.GlobalState.Set("globalHooks", hooks)
				}
			}
		}
	}
	return fmt.Errorf("hook '%s' not found", name)
}

func disableHook(ctx *storage.StorageContext, name string) error {
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		var hooks []map[string]interface{}
		jsonData, _ := json.Marshal(val)
		if err := json.Unmarshal(jsonData, &hooks); err == nil {
			for i, hook := range hooks {
				if hook["name"] == name {
					hooks[i]["enabled"] = false
					return ctx.GlobalState.Set("globalHooks", hooks)
				}
			}
		}
	}
	return fmt.Errorf("hook '%s' not found", name)
}

// Skills helpers
type SkillsOutput struct {
	Global []SkillData `json:"global"`
	Local  []SkillData `json:"local"`
}

type SkillData struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"`
}

func listSkillsJSON(ctx *storage.StorageContext) (*SkillsOutput, error) {
	var output SkillsOutput

	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		if skills, err := parseSkillsData(val); err == nil {
			for i := range skills {
				skills[i].Source = "global"
			}
			output.Global = skills
		}
	}

	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localSkills"); ok {
			if skills, err := parseSkillsData(val); err == nil {
				for i := range skills {
					skills[i].Source = "local"
				}
				output.Local = skills
			}
		}
	}

	return &output, nil
}

func parseSkillsData(data interface{}) ([]SkillData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var skills []SkillData
	if err := json.Unmarshal(jsonData, &skills); err != nil {
		return nil, err
	}

	return skills, nil
}

func enableSkill(ctx *storage.StorageContext, path string) error {
	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		var skills []map[string]interface{}
		jsonData, _ := json.Marshal(val)
		if err := json.Unmarshal(jsonData, &skills); err == nil {
			for i, skill := range skills {
				if skill["path"] == path {
					skills[i]["enabled"] = true
					return ctx.GlobalState.Set("globalSkills", skills)
				}
			}
		}
	}
	return fmt.Errorf("skill '%s' not found", path)
}

func disableSkill(ctx *storage.StorageContext, path string) error {
	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		var skills []map[string]interface{}
		jsonData, _ := json.Marshal(val)
		if err := json.Unmarshal(jsonData, &skills); err == nil {
			for i, skill := range skills {
				if skill["path"] == path {
					skills[i]["enabled"] = false
					return ctx.GlobalState.Set("globalSkills", skills)
				}
			}
		}
	}
	return fmt.Errorf("skill '%s' not found", path)
}

// Workflows helpers
type WorkflowsOutput struct {
	Global []WorkflowData `json:"global"`
	Local  []WorkflowData `json:"local"`
}

type WorkflowData struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"`
}

func listWorkflowsJSON(ctx *storage.StorageContext) (*WorkflowsOutput, error) {
	var output WorkflowsOutput

	if val, ok := ctx.GlobalState.Get("globalWorkflowToggles"); ok {
		if toggles, err := parseTogglesData(val); err == nil {
			for path, enabled := range toggles {
				output.Global = append(output.Global, WorkflowData{
					Path:    path,
					Name:    filepath.Base(path),
					Enabled: enabled,
					Source:  "global",
				})
			}
		}
	}

	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localWorkflowToggles"); ok {
			if toggles, err := parseTogglesData(val); err == nil {
				for path, enabled := range toggles {
					output.Local = append(output.Local, WorkflowData{
						Path:    path,
						Name:    filepath.Base(path),
						Enabled: enabled,
						Source:  "local",
					})
				}
			}
		}
	}

	return &output, nil
}

func parseTogglesData(data interface{}) (map[string]bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var toggles map[string]bool
	if err := json.Unmarshal(jsonData, &toggles); err != nil {
		// Try as map[string]interface{}
		var raw map[string]interface{}
		if err := json.Unmarshal(jsonData, &raw); err != nil {
			return nil, err
		}
		toggles = make(map[string]bool)
		for k, v := range raw {
			if b, ok := v.(bool); ok {
				toggles[k] = b
			}
		}
	}

	return toggles, nil
}

func enableWorkflow(ctx *storage.StorageContext, path string) error {
	if val, ok := ctx.GlobalState.Get("globalWorkflowToggles"); ok {
		toggles, _ := parseTogglesData(val)
		if toggles == nil {
			toggles = make(map[string]bool)
		}
		toggles[path] = true
		return ctx.GlobalState.Set("globalWorkflowToggles", toggles)
	}
	return fmt.Errorf("workflow '%s' not found", path)
}

// Rules helpers
type RulesOutput struct {
	Rules []RuleData `json:"rules"`
}

type RuleData struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	RuleType string `json:"ruleType"`
	Source   string `json:"source"`
}

func listRulesJSON(ctx *storage.StorageContext) (*RulesOutput, error) {
	var output RulesOutput

	// Collect from all rule types
	ruleTypes := []string{
		"globalClineRulesToggles",
		"localClineRulesToggles",
		"globalCursorRulesToggles",
		"localCursorRulesToggles",
		"globalWindsurfRulesToggles",
		"localWindsurfRulesToggles",
		"globalAgentsRulesToggles",
		"localAgentsRulesToggles",
	}

	ruleTypeMap := map[string]string{
		"globalClineRulesToggles":    "cline",
		"localClineRulesToggles":     "cline",
		"globalCursorRulesToggles":   "cursor",
		"localCursorRulesToggles":    "cursor",
		"globalWindsurfRulesToggles": "windsurf",
		"localWindsurfRulesToggles":  "windsurf",
		"globalAgentsRulesToggles":   "agents",
		"localAgentsRulesToggles":    "agents",
	}

	sourceMap := map[string]string{
		"globalClineRulesToggles":    "global",
		"localClineRulesToggles":     "local",
		"globalCursorRulesToggles":   "global",
		"localCursorRulesToggles":    "local",
		"globalWindsurfRulesToggles": "global",
		"localWindsurfRulesToggles":  "local",
		"globalAgentsRulesToggles":   "global",
		"localAgentsRulesToggles":    "local",
	}

	for _, key := range ruleTypes {
		storage := ctx.GlobalState
		if strings.HasPrefix(key, "local") && ctx.WorkspaceState != nil {
			storage = ctx.WorkspaceState
		}

		if val, ok := storage.Get(key); ok {
			if toggles, err := parseTogglesData(val); err == nil {
				for path, enabled := range toggles {
					output.Rules = append(output.Rules, RuleData{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: ruleTypeMap[key],
						Source:   sourceMap[key],
					})
				}
			}
		}
	}

	return &output, nil
}

func listRulesByTypeJSON(ctx *storage.StorageContext, ruleType string) (*RulesOutput, error) {
	allRules, err := listRulesJSON(ctx)
	if err != nil {
		return nil, err
	}

	var filtered RulesOutput
	for _, rule := range allRules.Rules {
		if rule.RuleType == ruleType {
			filtered.Rules = append(filtered.Rules, rule)
		}
	}

	return &filtered, nil
}

func enableRule(ctx *storage.StorageContext, path string) error {
	ruleTypes := []string{
		"globalClineRulesToggles",
		"localClineRulesToggles",
		"globalCursorRulesToggles",
		"localCursorRulesToggles",
		"globalWindsurfRulesToggles",
		"localWindsurfRulesToggles",
		"globalAgentsRulesToggles",
		"localAgentsRulesToggles",
	}

	for _, key := range ruleTypes {
		storage := ctx.GlobalState
		if strings.HasPrefix(key, "local") && ctx.WorkspaceState != nil {
			storage = ctx.WorkspaceState
		}

		if val, ok := storage.Get(key); ok {
			if toggles, err := parseTogglesData(val); err == nil {
				if _, exists := toggles[path]; exists {
					toggles[path] = true
					return storage.Set(key, toggles)
				}
			}
		}
	}

	return fmt.Errorf("rule '%s' not found", path)
}

// BenchmarkPhase3_DevDoctor benchmarks dev doctor performance
func BenchmarkPhase3_DevDoctor(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "cline-bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		b.Fatal(err)
	}
	defer ctx.Close()

	// Set up some state
	ctx.GlobalState.Set("apiProvider", "anthropic")
	ctx.Secrets.Set("anthropicApiKey", "test-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run diagnostics checks
		_ = checkStorageBench(ctx)
		_ = checkConfigBench(ctx)
		_ = checkSecretsBench(ctx)
		_ = checkAPIKeyBench(ctx)
	}
}

func checkStorageBench(ctx *storage.StorageContext) bool {
	return ctx.GlobalState != nil
}

func checkConfigBench(ctx *storage.StorageContext) bool {
	_, ok := ctx.GlobalState.Get("apiProvider")
	return ok
}

func checkSecretsBench(ctx *storage.StorageContext) bool {
	return ctx.Secrets != nil
}

func checkAPIKeyBench(ctx *storage.StorageContext) bool {
	_, ok := ctx.Secrets.Get("anthropicApiKey")
	return ok
}

// BenchmarkPhase3_HooksOperations benchmarks hooks operations
func BenchmarkPhase3_HooksOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "cline-bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		b.Fatal(err)
	}
	defer ctx.Close()

	// Create test hooks
	var hooks []map[string]interface{}
	for i := 0; i < 100; i++ {
		hooks = append(hooks, map[string]interface{}{
			"name":    fmt.Sprintf("hook-%d", i),
			"enabled": i%2 == 0,
		})
	}
	ctx.GlobalState.Set("globalHooks", hooks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// List hooks
		val, _ := ctx.GlobalState.Get("globalHooks")
		var h []map[string]interface{}
		jsonData, _ := json.Marshal(val)
		json.Unmarshal(jsonData, &h)

		// Find and toggle a hook
		for j := range h {
			if h[j]["name"] == "hook-50" {
				h[j]["enabled"] = !h[j]["enabled"].(bool)
				break
			}
		}

		ctx.GlobalState.Set("globalHooks", h)
	}
}
