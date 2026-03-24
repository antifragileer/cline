package tui

import (
	"testing"

	"github.com/cline/cline/golang-cli/internal/storage"
)

func TestNewConfigModel(t *testing.T) {
	// Create a temporary storage context for testing
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	dataDir := "/tmp/test-data"
	m := NewConfigModel(storageCtx, dataDir)

	if m.Title != "⚙️  Cline Configuration" {
		t.Errorf("Expected title '⚙️  Cline Configuration', got %q", m.Title)
	}

	if m.CurrentTab != TabSettings {
		t.Errorf("Expected initial tab TabSettings, got %v", m.CurrentTab)
	}

	if m.DataDir != dataDir {
		t.Errorf("Expected dataDir %q, got %q", dataDir, m.DataDir)
	}

	// Check initial state
	if m.ViewState != ViewStateList {
		t.Errorf("Expected initial state %v, got %v", ViewStateList, m.ViewState)
	}

	if m.SelectedIndex != 0 {
		t.Errorf("Expected initial selection 0, got %d", m.SelectedIndex)
	}

	// Check tabs
	if len(m.Tabs) != 5 {
		t.Errorf("Expected 5 tabs, got %d", len(m.Tabs))
	}

	expectedTabs := []TabView{TabSettings, TabRules, TabWorkflows, TabHooks, TabSkills}
	for i, expected := range expectedTabs {
		if m.Tabs[i].Key != expected {
			t.Errorf("Expected tab %v at index %d, got %v", expected, i, m.Tabs[i].Key)
		}
	}

	// Check that entries are built
	if len(m.ConfigEntries) == 0 {
		t.Errorf("Expected config entries to be built, got none")
	}
}

func TestTabViewString(t *testing.T) {
	tests := []struct {
		tab      TabView
		expected string
	}{
		{TabSettings, "Settings"},
		{TabRules, "Rules"},
		{TabWorkflows, "Workflows"},
		{TabHooks, "Hooks"},
		{TabSkills, "Skills"},
		{TabView(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.tab.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestConfigModelCallbacks(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// Test OnUpdateGlobal callback
	updateGlobalCalled := false
	var updateGlobalKey string
	var updateGlobalValue interface{}
	m.OnUpdateGlobal = func(key string, value interface{}) error {
		updateGlobalCalled = true
		updateGlobalKey = key
		updateGlobalValue = value
		return nil
	}

	// Simulate update
	if m.OnUpdateGlobal != nil {
		m.OnUpdateGlobal("testKey", "testValue")
	}

	if !updateGlobalCalled {
		t.Error("Expected OnUpdateGlobal to be called")
	}

	if updateGlobalKey != "testKey" {
		t.Errorf("Expected update key 'testKey', got %q", updateGlobalKey)
	}

	if updateGlobalValue != "testValue" {
		t.Errorf("Expected update value 'testValue', got %v", updateGlobalValue)
	}

	// Test OnQuit callback
	quitCalled := false
	m.OnQuit = func() {
		quitCalled = true
	}

	if m.OnQuit != nil {
		m.OnQuit()
	}

	if !quitCalled {
		t.Error("Expected OnQuit to be called")
	}

	// Test toggle callbacks
	toggleRuleCalled := false
	m.OnToggleRule = func(isGlobal bool, rulePath string, enabled bool, ruleType string) error {
		toggleRuleCalled = true
		return nil
	}

	if m.OnToggleRule != nil {
		m.OnToggleRule(true, "/path/to/rule", true, "cline")
	}

	if !toggleRuleCalled {
		t.Error("Expected OnToggleRule to be called")
	}

	toggleWorkflowCalled := false
	m.OnToggleWorkflow = func(isGlobal bool, workflowPath string, enabled bool) error {
		toggleWorkflowCalled = true
		return nil
	}

	if m.OnToggleWorkflow != nil {
		m.OnToggleWorkflow(true, "/path/to/workflow", true)
	}

	if !toggleWorkflowCalled {
		t.Error("Expected OnToggleWorkflow to be called")
	}

	toggleHookCalled := false
	m.OnToggleHook = func(isGlobal bool, hookName string, enabled bool, workspaceName string) error {
		toggleHookCalled = true
		return nil
	}

	if m.OnToggleHook != nil {
		m.OnToggleHook(true, "test-hook", true, "")
	}

	if !toggleHookCalled {
		t.Error("Expected OnToggleHook to be called")
	}

	toggleSkillCalled := false
	m.OnToggleSkill = func(isGlobal bool, skillPath string, enabled bool) error {
		toggleSkillCalled = true
		return nil
	}

	if m.OnToggleSkill != nil {
		m.OnToggleSkill(true, "/path/to/skill", true)
	}

	if !toggleSkillCalled {
		t.Error("Expected OnToggleSkill to be called")
	}
}

func TestConfigModelInit(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	cmd := m.Init()

	// Init should return nil for the main config model
	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}

func TestConfigModelGetFilteredConfigEntries(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// Test without search query
	allEntries := m.getFilteredConfigEntries()
	if len(allEntries) != len(m.ConfigEntries) {
		t.Errorf("Expected %d entries without filter, got %d", len(m.ConfigEntries), len(allEntries))
	}

	// Test with search query
	m.SearchQuery = "nonexistent"
	filtered := m.getFilteredConfigEntries()
	if len(filtered) != 0 {
		t.Errorf("Expected 0 entries with non-matching filter, got %d", len(filtered))
	}

	// Test with matching query
	if len(m.ConfigEntries) > 0 {
		m.SearchQuery = m.ConfigEntries[0].Key[:3] // First 3 chars of first key
		filtered = m.getFilteredConfigEntries()
		if len(filtered) == 0 {
			t.Error("Expected some entries with matching filter")
		}
	}
}

func TestConfigModelGetCurrentListLength(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// Test settings tab
	m.CurrentTab = TabSettings
	settingsLen := m.getCurrentListLength()
	if settingsLen != len(m.ConfigEntries) {
		t.Errorf("Expected settings length %d, got %d", len(m.ConfigEntries), settingsLen)
	}

	// Test rules tab
	m.CurrentTab = TabRules
	rulesLen := m.getCurrentListLength()
	if rulesLen != len(m.RuleEntries) {
		t.Errorf("Expected rules length %d, got %d", len(m.RuleEntries), rulesLen)
	}

	// Test workflows tab
	m.CurrentTab = TabWorkflows
	workflowsLen := m.getCurrentListLength()
	if workflowsLen != len(m.WorkflowEntries) {
		t.Errorf("Expected workflows length %d, got %d", len(m.WorkflowEntries), workflowsLen)
	}

	// Test hooks tab
	m.CurrentTab = TabHooks
	hooksLen := m.getCurrentListLength()
	if hooksLen != len(m.HookEntries) {
		t.Errorf("Expected hooks length %d, got %d", len(m.HookEntries), hooksLen)
	}

	// Test skills tab
	m.CurrentTab = TabSkills
	skillsLen := m.getCurrentListLength()
	if skillsLen != len(m.SkillEntries) {
		t.Errorf("Expected skills length %d, got %d", len(m.SkillEntries), skillsLen)
	}
}

func TestConfigModelGetAvailableTabs(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// With all features enabled, should have 5 tabs
	available := m.getAvailableTabs()
	if len(available) != 5 {
		t.Errorf("Expected 5 available tabs with all features enabled, got %d", len(available))
	}

	// Disable hooks
	m.HooksEnabled = false
	available = m.getAvailableTabs()
	if len(available) != 4 {
		t.Errorf("Expected 4 available tabs with hooks disabled, got %d", len(available))
	}

	// Disable skills
	m.SkillsEnabled = false
	available = m.getAvailableTabs()
	if len(available) != 3 {
		t.Errorf("Expected 3 available tabs with hooks and skills disabled, got %d", len(available))
	}
}

func TestConfigModelSetError(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	testErr := &testError{msg: "test error"}
	m.SetError(testErr)

	if m.Err != testErr {
		t.Errorf("Expected error to be set, got %v", m.Err)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestConfigModelGetSelectedItem(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// Test settings tab
	m.CurrentTab = TabSettings
	m.SelectedIndex = 0
	item := m.GetSelectedItem()
	if item == nil {
		t.Error("Expected GetSelectedItem to return an item for settings")
	}

	// Test out of bounds
	m.SelectedIndex = 10000
	item = m.GetSelectedItem()
	if item != nil {
		t.Error("Expected GetSelectedItem to return nil for out of bounds")
	}

	// Test rules tab
	m.CurrentTab = TabRules
	m.SelectedIndex = 0
	if len(m.RuleEntries) > 0 {
		item = m.GetSelectedItem()
		if item == nil {
			t.Error("Expected GetSelectedItem to return an item for rules")
		}
	} else {
		item = m.GetSelectedItem()
		if item != nil {
			t.Error("Expected GetSelectedItem to return nil when no rules")
		}
	}
}

func TestFormatConfigKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"apiProvider", "apiProvider"},
		{"actModeApiProvider", "act Mode Api Provider"},
		{"planModeThinkingBudgetTokens", "plan Mode Thinking Budget Tokens"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatConfigKey(tt.input)
			if result != tt.expected {
				t.Errorf("formatConfigKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectValueType(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{true, "boolean"},
		{false, "boolean"},
		{42.0, "number"},
		{42, "number"},
		{"hello", "string"},
		{map[string]interface{}{}, "object"},
		{[]interface{}{}, "array"},
		{nil, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := detectValueType(tt.value)
			if result != tt.expected {
				t.Errorf("detectValueType(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestIsEditableKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"apiProvider", true},
		{"actModeModelId", true},
		{"version", false},
		{"installationId", false},
		{"lastShownAnnouncementId", false},
		{"taskHistory", false},
		{"checkpointStorage", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isEditableKey(tt.key)
			if result != tt.expected {
				t.Errorf("isEditableKey(%q) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetBoolValue(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected bool
	}{
		{true, true},
		{false, false},
		{"true", false},
		{1, false},
		{nil, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getBoolValue(tt.value)
			if result != tt.expected {
				t.Errorf("getBoolValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestGetStringValue(t *testing.T) {
	tests := []struct {
		value    interface{}
		expected string
	}{
		{"hello", "hello"},
		{"", ""},
		{123, ""},
		{nil, ""},
		{true, ""},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getStringValue(tt.value)
			if result != tt.expected {
				t.Errorf("getStringValue(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestFormatValueForDisplay(t *testing.T) {
	tests := []struct {
		entry    ConfigEntry
		expected string
	}{
		{ConfigEntry{Type: "boolean", Value: true}, "enabled"},
		{ConfigEntry{Type: "boolean", Value: false}, "disabled"},
		{ConfigEntry{Type: "string", Value: "hello"}, "hello"},
		{ConfigEntry{Type: "string", Value: "a very long string that exceeds fifty characters"}, "a very long string that exceeds fifty chara..."},
		{ConfigEntry{Type: "number", Value: 42.5}, "42.5"},
	}

	for _, tt := range tests {
		t.Run(tt.entry.Type, func(t *testing.T) {
			result := formatValueForDisplay(tt.entry)
			if result != tt.expected {
				t.Errorf("formatValueForDisplay() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfigStyles(t *testing.T) {
	styles := DefaultConfigStyles()

	// Just ensure it returns valid styles without panicking
	if styles.TitleStyle.GetForeground() == nil {
		t.Error("Expected TitleStyle to have a foreground color")
	}

	if styles.SelectedStyle.GetForeground() == nil {
		t.Error("Expected SelectedStyle to have a foreground color")
	}

	if styles.ErrorStyle.GetForeground() == nil {
		t.Error("Expected ErrorStyle to have a foreground color")
	}
}

func TestConfigProgram(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	prog := NewConfigProgram(storageCtx, "/tmp/test")

	if prog == nil {
		t.Fatal("Expected NewConfigProgram to return a non-nil program")
	}

	if prog.configModel.StorageCtx != storageCtx {
		t.Error("Expected program to use the provided storage context")
	}

	// Test SetCallbacks
	onUpdateGlobalCalled := false
	onUpdateWorkspaceCalled := false
	onQuitCalled := false

	prog.SetCallbacks(
		func(key string, value interface{}) error {
			onUpdateGlobalCalled = true
			return nil
		},
		func(key string, value interface{}) error {
			onUpdateWorkspaceCalled = true
			return nil
		},
		func() {
			onQuitCalled = true
		},
	)

	if prog.configModel.OnUpdateGlobal == nil {
		t.Error("Expected OnUpdateGlobal callback to be set")
	}

	if prog.configModel.OnUpdateWorkspace == nil {
		t.Error("Expected OnUpdateWorkspace callback to be set")
	}

	if prog.configModel.OnQuit == nil {
		t.Error("Expected OnQuit callback to be set")
	}

	// Test SetToggleCallbacks
	prog.SetToggleCallbacks(
		func(isGlobal bool, rulePath string, enabled bool, ruleType string) error { return nil },
		func(isGlobal bool, workflowPath string, enabled bool) error { return nil },
		func(isGlobal bool, hookName string, enabled bool, workspaceName string) error { return nil },
		func(isGlobal bool, skillPath string, enabled bool) error { return nil },
	)

	if prog.configModel.OnToggleRule == nil {
		t.Error("Expected OnToggleRule callback to be set")
	}
}

func TestExportConfig(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	// Set a test value
	storageCtx.GlobalState.Set("testKey", "testValue")

	m := NewConfigModel(storageCtx, "/tmp/test")

	config, err := m.ExportConfig()
	if err != nil {
		t.Errorf("ExportConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if global, ok := config["global"].(map[string]interface{}); ok {
		if val, ok := global["testKey"]; !ok || val != "testValue" {
			t.Errorf("Expected global.testKey to be 'testValue', got %v", val)
		}
	} else {
		t.Error("Expected global config to be present")
	}
}

func TestGetHelpText(t *testing.T) {
	storageCtx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Skipf("Storage initialization failed: %v", err)
	}
	defer storageCtx.Close()

	m := NewConfigModel(storageCtx, "/tmp/test")

	// Test settings tab help
	m.CurrentTab = TabSettings
	help := m.getHelpText()
	if help == "" {
		t.Error("Expected non-empty help text for settings tab")
	}
	if !contains(help, "search") {
		t.Error("Expected settings help to mention search")
	}

	// Test other tabs help
	m.CurrentTab = TabRules
	help = m.getHelpText()
	if help == "" {
		t.Error("Expected non-empty help text for rules tab")
	}
}

func TestGetProviderModelKey(t *testing.T) {
	tests := []struct {
		provider string
		mode     string
		expected string
	}{
		{"anthropic", "act", "actModeAnthropicModelId"},
		{"openai", "plan", "planModeOpenAiModelId"},
		{"openrouter", "act", "actModeOpenRouterModelId"},
		{"unknown", "act", "actModeUnknownModelId"},
		{"", "act", "actModeModelId"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.mode, func(t *testing.T) {
			result := getProviderModelKey(tt.provider, tt.mode)
			if result != tt.expected {
				t.Errorf("getProviderModelKey(%q, %q) = %q, want %q", tt.provider, tt.mode, result, tt.expected)
			}
		})
	}
}