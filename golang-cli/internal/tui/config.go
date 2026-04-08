// Package tui provides a Bubble Tea based terminal user interface for Cline configuration.
// This file implements the interactive config view that displays and manages configuration settings.
package tui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// TabView represents the different tabs in the config view
type TabView int

const (
	// TabSettings shows global and workspace settings
	TabSettings TabView = iota
	// TabRules shows cline/cursor/windsurf/agents rules toggles
	TabRules
	// TabWorkflows shows workflow toggles
	TabWorkflows
	// TabHooks shows hook toggles
	TabHooks
	// TabSkills shows skill toggles
	TabSkills
)

// String returns the display name for the tab
func (t TabView) String() string {
	switch t {
	case TabSettings:
		return "Settings"
	case TabRules:
		return "Rules"
	case TabWorkflows:
		return "Workflows"
	case TabHooks:
		return "Hooks"
	case TabSkills:
		return "Skills"
	default:
		return "Unknown"
	}
}

// TabInfo holds information about a tab
type TabInfo struct {
	Key          TabView
	Label        string
	RequiresFlag string // "hooks", "skills", or empty
}

// ConfigEntry represents a single configuration entry for the settings tab
type ConfigEntry struct {
	Key        string
	Label      string
	Value      interface{}
	Type       string // "string", "boolean", "number", "object", "array"
	Source     string // "global" or "workspace"
	IsEditable bool
}

// ToggleEntry represents a toggleable item (rules, workflows, hooks, skills)
type ToggleEntry struct {
	Key           string
	Label         string
	Enabled       bool
	Source        string // "global" or "workspace"
	Type          string // "cline", "cursor", "windsurf", "agents" for rules
	WorkspaceName string // for workspace-specific hooks
	Description   string
}

// HookInfo represents hook information
type HookInfo struct {
	Name        string
	Description string
	Enabled     bool
}

// SkillInfo represents skill information
type SkillInfo struct {
	Name        string
	Path        string
	Description string
	Enabled     bool
}

// ConfigModel represents the state of the config TUI
type ConfigModel struct {
	// Title of the configuration view
	Title string

	// Current tab
	CurrentTab TabView

	// Available tabs
	Tabs []TabInfo

	// Settings tab data
	ConfigEntries []ConfigEntry
	SearchQuery   string

	// Rules tab data
	RuleEntries []ToggleEntry

	// Workflows tab data
	WorkflowEntries []ToggleEntry

	// Hooks tab data
	HookEntries []struct {
		Hook          HookInfo
		IsGlobal      bool
		WorkspaceName string
	}

	// Skills tab data
	SkillEntries []struct {
		Skill    SkillInfo
		IsGlobal bool
	}

	// Selected item index
	SelectedIndex int

	// Current view state
	ViewState ConfigViewState

	// Text input for editing values and search
	TextInput textinput.Model

	// Width and height of the terminal
	Width  int
	Height int

	// Configuration data
	Provider  string
	Model     string
	APIKeySet bool
	DataDir   string

	// Feature flags
	HooksEnabled  bool
	SkillsEnabled bool

	// Storage context
	StorageCtx *storage.StorageContext

	// Callbacks
	OnUpdateGlobal    func(key string, value interface{}) error
	OnUpdateWorkspace func(key string, value interface{}) error
	OnToggleRule      func(isGlobal bool, rulePath string, enabled bool, ruleType string) error
	OnToggleWorkflow  func(isGlobal bool, workflowPath string, enabled bool) error
	OnToggleHook      func(isGlobal bool, hookName string, enabled bool, workspaceName string) error
	OnToggleSkill     func(isGlobal bool, skillPath string, enabled bool) error
	OnOpenFolder      func(folderType string, isGlobal bool) error
	OnQuit            func()

	// Error state
	Err error
}

// ConfigViewState represents the current view state
type ConfigViewState int

const (
	// ViewStateList shows the list of settings
	ViewStateList ConfigViewState = iota
	// ViewStateEditing shows the edit input
	ViewStateEditing
	// ViewStateConfirm shows a confirmation dialog
	ViewStateConfirm
)

// ConfigStyles holds the styling for the config TUI
type ConfigStyles struct {
	TitleStyle         lipgloss.Style
	SubtitleStyle      lipgloss.Style
	TabStyle           lipgloss.Style
	SelectedTabStyle   lipgloss.Style
	SelectedStyle      lipgloss.Style
	ItemStyle          lipgloss.Style
	DescriptionStyle   lipgloss.Style
	HelpStyle          lipgloss.Style
	ErrorStyle         lipgloss.Style
	HeaderStyle        lipgloss.Style
	SectionHeaderStyle lipgloss.Style
	BoxStyle           lipgloss.Style
	DisabledStyle      lipgloss.Style
	EnabledStyle       lipgloss.Style
}

// DefaultConfigStyles returns the default styling configuration
func DefaultConfigStyles() ConfigStyles {
	return ConfigStyles{
		TitleStyle:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87CEEB")).MarginBottom(1),
		SubtitleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		TabStyle:           lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		SelectedTabStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#87CEEB")),
		SelectedStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB")).Bold(true),
		ItemStyle:          lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0")),
		DescriptionStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")).Italic(true),
		HelpStyle:          lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		ErrorStyle:         lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Bold(true),
		HeaderStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true),
		SectionHeaderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true).MarginTop(1),
		BoxStyle:           lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#87CEEB")).Padding(1),
		DisabledStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")),
		EnabledStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#4ECB71")),
	}
}

// NewConfigModel creates a new configuration TUI model
func NewConfigModel(storageCtx *storage.StorageContext, dataDir string) ConfigModel {
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.Focus()

	tabs := []TabInfo{
		{Key: TabSettings, Label: "Settings"},
		{Key: TabRules, Label: "Rules"},
		{Key: TabWorkflows, Label: "Workflows"},
		{Key: TabHooks, Label: "Hooks", RequiresFlag: "hooks"},
		{Key: TabSkills, Label: "Skills", RequiresFlag: "skills"},
	}

	// Get current configuration
	provider, model, apiKeySet := getCurrentConfigFromStorage(storageCtx)

	m := ConfigModel{
		Title:         "⚙️  Cline Configuration",
		CurrentTab:    TabSettings,
		Tabs:          tabs,
		ViewState:     ViewStateList,
		TextInput:     ti,
		Width:         80,
		Height:        24,
		Provider:      provider,
		Model:         model,
		APIKeySet:     apiKeySet,
		DataDir:       dataDir,
		StorageCtx:    storageCtx,
		HooksEnabled:  true, // Default to enabled
		SkillsEnabled: true, // Default to enabled
	}

	m.buildAllEntries()
	return m
}

// getCurrentConfigFromStorage retrieves the current configuration from storage
func getCurrentConfigFromStorage(ctx *storage.StorageContext) (provider, model string, apiKeySet bool) {
	// Get provider from global state
	if val, ok := ctx.GlobalState.Get("actModeApiProvider"); ok {
		if str, ok := val.(string); ok && str != "" {
			provider = str
		}
	}
	if provider == "" {
		if val, ok := ctx.GlobalState.Get("apiProvider"); ok {
			if str, ok := val.(string); ok && str != "" {
				provider = str
			}
		}
	}
	if provider == "" {
		provider = "not configured"
	}

	// Get model from global state based on provider
	modelKey := getProviderModelKey(provider, "act")
	if val, ok := ctx.GlobalState.Get(modelKey); ok {
		if str, ok := val.(string); ok && str != "" {
			model = str
		}
	}
	if model == "" {
		model = "not configured"
	}

	// Check if API key is set
	apiKeyFields := []string{
		"apiKey", "openRouterApiKey", "anthropicApiKey", "openAiApiKey",
		"geminiApiKey", "bedrockAccessKey", "ollamaApiKey",
	}

	for _, key := range apiKeyFields {
		if val, ok := ctx.Secrets.Get(key); ok {
			if str, ok := val.(string); ok && str != "" {
				apiKeySet = true
				break
			}
		}
	}

	return provider, model, apiKeySet
}

// getProviderModelKey returns the state key for a provider's model
func getProviderModelKey(provider, mode string) string {
	providerKeyMap := map[string]string{
		"anthropic":     "anthropic",
		"openai":        "openAi",
		"openai-native": "openAiNative",
		"openrouter":    "openRouter",
		"gemini":        "gemini",
		"bedrock":       "bedrock",
		"ollama":        "ollama",
		"lmstudio":      "lmStudio",
		"cline":         "cline",
	}

	prefix := providerKeyMap[provider]
	if prefix == "" {
		prefix = provider
	}

	if len(prefix) == 0 {
		return fmt.Sprintf("%sMode%sModelId", mode, prefix)
	}
	return fmt.Sprintf("%sMode%sModelId", mode, strings.ToUpper(prefix[:1])+prefix[1:])
}

// buildAllEntries builds all entries for all tabs
func (m *ConfigModel) buildAllEntries() {
	m.buildConfigEntries()
	m.buildRuleEntries()
	m.buildWorkflowEntries()
	m.buildHookEntries()
	m.buildSkillEntries()
}

// buildConfigEntries builds configuration entries from storage
func (m *ConfigModel) buildConfigEntries() {
	var entries []ConfigEntry

	// Build from global state
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		globalData := m.StorageCtx.GlobalState.GetAll()
		for key, value := range globalData {
			entry := ConfigEntry{
				Key:        key,
				Label:      formatConfigKey(key),
				Value:      value,
				Type:       detectValueType(value),
				Source:     "global",
				IsEditable: isEditableKey(key),
			}
			entries = append(entries, entry)
		}
	}

	// Sort global entries
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	// Build from workspace state
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		workspaceData := m.StorageCtx.WorkspaceState.GetAll()
		var workspaceEntries []ConfigEntry
		for key, value := range workspaceData {
			entry := ConfigEntry{
				Key:        key,
				Label:      formatConfigKey(key),
				Value:      value,
				Type:       detectValueType(value),
				Source:     "workspace",
				IsEditable: isEditableKey(key),
			}
			workspaceEntries = append(workspaceEntries, entry)
		}
		// Sort workspace entries
		sort.Slice(workspaceEntries, func(i, j int) bool {
			return workspaceEntries[i].Key < workspaceEntries[j].Key
		})
		entries = append(entries, workspaceEntries...)
	}

	m.ConfigEntries = entries
}

// buildRuleEntries builds rule toggle entries
func (m *ConfigModel) buildRuleEntries() {
	var entries []ToggleEntry

	// Global cline rules
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		if val, ok := m.StorageCtx.GlobalState.Get("globalClineRulesToggles"); ok {
			if toggles, ok := val.(map[string]interface{}); ok {
				for path, enabled := range toggles {
					entries = append(entries, ToggleEntry{
						Key:     path,
						Label:   filepath.Base(path),
						Enabled: getBoolValue(enabled),
						Source:  "global",
						Type:    "cline",
					})
				}
			}
		}
	}

	// Workspace cline rules
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		if val, ok := m.StorageCtx.WorkspaceState.Get("localClineRulesToggles"); ok {
			if toggles, ok := val.(map[string]interface{}); ok {
				for path, enabled := range toggles {
					entries = append(entries, ToggleEntry{
						Key:     path,
						Label:   filepath.Base(path),
						Enabled: getBoolValue(enabled),
						Source:  "workspace",
						Type:    "cline",
					})
				}
			}
		}
	}

	// Sort entries by source then key
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source == "global"
		}
		return entries[i].Key < entries[j].Key
	})

	m.RuleEntries = entries
}

// buildWorkflowEntries builds workflow toggle entries
func (m *ConfigModel) buildWorkflowEntries() {
	var entries []ToggleEntry

	// Global workflows
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		if val, ok := m.StorageCtx.GlobalState.Get("globalWorkflowToggles"); ok {
			if toggles, ok := val.(map[string]interface{}); ok {
				for path, enabled := range toggles {
					entries = append(entries, ToggleEntry{
						Key:     path,
						Label:   filepath.Base(path),
						Enabled: getBoolValue(enabled),
						Source:  "global",
					})
				}
			}
		}
	}

	// Workspace workflows
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		if val, ok := m.StorageCtx.WorkspaceState.Get("localWorkflowToggles"); ok {
			if toggles, ok := val.(map[string]interface{}); ok {
				for path, enabled := range toggles {
					entries = append(entries, ToggleEntry{
						Key:     path,
						Label:   filepath.Base(path),
						Enabled: getBoolValue(enabled),
						Source:  "workspace",
					})
				}
			}
		}
	}

	// Sort entries
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source == "global"
		}
		return entries[i].Key < entries[j].Key
	})

	m.WorkflowEntries = entries
}

// buildHookEntries builds hook entries
func (m *ConfigModel) buildHookEntries() {
	var entries []struct {
		Hook          HookInfo
		IsGlobal      bool
		WorkspaceName string
	}

	// Global hooks
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		if val, ok := m.StorageCtx.GlobalState.Get("globalHooks"); ok {
			if hooks, ok := val.([]interface{}); ok {
				for _, h := range hooks {
					if hookMap, ok := h.(map[string]interface{}); ok {
						name := getStringValue(hookMap["name"])
						entries = append(entries, struct {
							Hook          HookInfo
							IsGlobal      bool
							WorkspaceName string
						}{
							Hook: HookInfo{
								Name:        name,
								Description: getStringValue(hookMap["description"]),
								Enabled:     getBoolValue(hookMap["enabled"]),
							},
							IsGlobal: true,
						})
					}
				}
			}
		}
	}

	// Workspace hooks
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		if val, ok := m.StorageCtx.WorkspaceState.Get("workspaceHooks"); ok {
			if wsHooks, ok := val.([]interface{}); ok {
				for _, wh := range wsHooks {
					if wsHookMap, ok := wh.(map[string]interface{}); ok {
						wsName := getStringValue(wsHookMap["workspaceName"])
						if hooks, ok := wsHookMap["hooks"].([]interface{}); ok {
							for _, h := range hooks {
								if hookMap, ok := h.(map[string]interface{}); ok {
									name := getStringValue(hookMap["name"])
									entries = append(entries, struct {
										Hook          HookInfo
										IsGlobal      bool
										WorkspaceName string
									}{
										Hook: HookInfo{
											Name:        name,
											Description: getStringValue(hookMap["description"]),
											Enabled:     getBoolValue(hookMap["enabled"]),
										},
										IsGlobal:      false,
										WorkspaceName: wsName,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	// Sort entries
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Hook.Name < entries[j].Hook.Name
	})

	m.HookEntries = entries
}

// buildSkillEntries builds skill entries
func (m *ConfigModel) buildSkillEntries() {
	var entries []struct {
		Skill    SkillInfo
		IsGlobal bool
	}

	// Global skills
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		if val, ok := m.StorageCtx.GlobalState.Get("globalSkills"); ok {
			if skills, ok := val.([]interface{}); ok {
				for _, s := range skills {
					if skillMap, ok := s.(map[string]interface{}); ok {
						entries = append(entries, struct {
							Skill    SkillInfo
							IsGlobal bool
						}{
							Skill: SkillInfo{
								Name:        getStringValue(skillMap["name"]),
								Path:        getStringValue(skillMap["path"]),
								Description: getStringValue(skillMap["description"]),
								Enabled:     getBoolValue(skillMap["enabled"]),
							},
							IsGlobal: true,
						})
					}
				}
			}
		}
	}

	// Local skills
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		if val, ok := m.StorageCtx.WorkspaceState.Get("localSkills"); ok {
			if skills, ok := val.([]interface{}); ok {
				for _, s := range skills {
					if skillMap, ok := s.(map[string]interface{}); ok {
						entries = append(entries, struct {
							Skill    SkillInfo
							IsGlobal bool
						}{
							Skill: SkillInfo{
								Name:        getStringValue(skillMap["name"]),
								Path:        getStringValue(skillMap["path"]),
								Description: getStringValue(skillMap["description"]),
								Enabled:     getBoolValue(skillMap["enabled"]),
							},
							IsGlobal: false,
						})
					}
				}
			}
		}
	}

	// Sort entries
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Skill.Name < entries[j].Skill.Name
	})

	m.SkillEntries = entries
}

// Helper functions for type conversion
func getBoolValue(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func getStringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// formatConfigKey converts a config key to a human-readable label
func formatConfigKey(key string) string {
	// Add spaces before capital letters and uppercase the first letter
	var result strings.Builder
	for i, r := range key {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		if i == 0 {
			result.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// detectValueType detects the type of a value
func detectValueType(value interface{}) string {
	switch v := value.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case int, int64:
		return "number"
	case string:
		return "string"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		_ = v
		return "string"
	}
}

// isEditableKey determines if a key is editable
func isEditableKey(key string) bool {
	// Non-editable keys (system/internal)
	nonEditable := []string{
		"version", "installationId", "lastShownAnnouncementId",
		"taskHistory", "checkpointStorage",
	}
	for _, ne := range nonEditable {
		if key == ne {
			return false
		}
	}
	return true
}

// getFilteredConfigEntries returns config entries filtered by search query
func (m *ConfigModel) getFilteredConfigEntries() []ConfigEntry {
	if m.SearchQuery == "" {
		return m.ConfigEntries
	}

	var filtered []ConfigEntry
	query := strings.ToLower(m.SearchQuery)
	for _, entry := range m.ConfigEntries {
		if strings.Contains(strings.ToLower(entry.Key), query) ||
			strings.Contains(strings.ToLower(fmt.Sprintf("%v", entry.Value)), query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// getCurrentListLength returns the length of the current tab's list
func (m *ConfigModel) getCurrentListLength() int {
	switch m.CurrentTab {
	case TabSettings:
		return len(m.getFilteredConfigEntries())
	case TabRules:
		return len(m.RuleEntries)
	case TabWorkflows:
		return len(m.WorkflowEntries)
	case TabHooks:
		return len(m.HookEntries)
	case TabSkills:
		return len(m.SkillEntries)
	default:
		return 0
	}
}

// getAvailableTabs returns the list of available tabs based on feature flags
func (m *ConfigModel) getAvailableTabs() []TabInfo {
	var available []TabInfo
	for _, tab := range m.Tabs {
		if tab.RequiresFlag == "hooks" && !m.HooksEnabled {
			continue
		}
		if tab.RequiresFlag == "skills" && !m.SkillsEnabled {
			continue
		}
		available = append(available, tab)
	}
	return available
}

// Init implements tea.Model
func (m ConfigModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case ConfigErrorMsg:
		m.Err = msg.Err
		return m, nil
	}

	if m.ViewState == ViewStateEditing {
		m.TextInput, cmd = m.TextInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// ConfigErrorMsg represents an error message for the config TUI
type ConfigErrorMsg struct {
	Err error
}

// handleKeyMsg processes keyboard input
func (m *ConfigModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle quit keys in any state
	if key == "ctrl+c" {
		if m.OnQuit != nil {
			m.OnQuit()
		}
		return m, tea.Quit
	}

	switch m.ViewState {
	case ViewStateEditing:
		return m.handleEditKeys(key, msg)
	case ViewStateList:
		return m.handleListKeys(key, msg)
	}

	return m, nil
}

// handleEditKeys handles keys when in edit mode
func (m *ConfigModel) handleEditKeys(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.ViewState = ViewStateList
		m.TextInput.SetValue("")
		m.Err = nil
		return m, nil

	case "enter":
		return m.handleEditSave()
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

// handleEditSave saves the edited value
func (m *ConfigModel) handleEditSave() (tea.Model, tea.Cmd) {
	if m.CurrentTab != TabSettings {
		m.ViewState = ViewStateList
		m.TextInput.SetValue("")
		return m, nil
	}

	filtered := m.getFilteredConfigEntries()
	if m.SelectedIndex >= len(filtered) {
		m.ViewState = ViewStateList
		m.TextInput.SetValue("")
		return m, nil
	}

	entry := filtered[m.SelectedIndex]
	if !entry.IsEditable {
		m.ViewState = ViewStateList
		m.TextInput.SetValue("")
		return m, nil
	}

	value := m.TextInput.Value()
	var parsedValue interface{}
	var err error

	// Parse value based on type
	switch entry.Type {
	case "boolean":
		parsedValue = value == "true" || value == "yes" || value == "1"
	case "number":
		var f float64
		_, err = fmt.Sscanf(value, "%f", &f)
		if err == nil {
			parsedValue = f
		} else {
			parsedValue = value
		}
	default:
		parsedValue = value
	}

	// Save to storage
	if entry.Source == "global" && m.OnUpdateGlobal != nil {
		err = m.OnUpdateGlobal(entry.Key, parsedValue)
	} else if entry.Source == "workspace" && m.OnUpdateWorkspace != nil {
		err = m.OnUpdateWorkspace(entry.Key, parsedValue)
	}

	if err != nil {
		m.Err = err
		return m, nil
	}

	// Update local state
	if entry.Key == "actModeApiProvider" || entry.Key == "apiProvider" {
		m.Provider = value
	}

	// Refresh entries
	m.buildConfigEntries()

	m.ViewState = ViewStateList
	m.TextInput.SetValue("")
	m.Err = nil
	return m, nil
}

// handleListKeys handles keys when in list mode
func (m *ConfigModel) handleListKeys(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	listLen := m.getCurrentListLength()
	availableTabs := m.getAvailableTabs()

	switch key {
	case "q", "esc":
		if m.OnQuit != nil {
			m.OnQuit()
		}
		return m, tea.Quit

	case "up", "k":
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		} else {
			m.SelectedIndex = listLen - 1
		}

	case "down", "j":
		if m.SelectedIndex < listLen-1 {
			m.SelectedIndex++
		} else {
			m.SelectedIndex = 0
		}

	case "left", "h":
		// Navigate tabs
		currentTabIdx := -1
		for i, tab := range availableTabs {
			if tab.Key == m.CurrentTab {
				currentTabIdx = i
				break
			}
		}
		if currentTabIdx >= 0 {
			newIdx := currentTabIdx - 1
			if newIdx < 0 {
				newIdx = len(availableTabs) - 1
			}
			m.CurrentTab = availableTabs[newIdx].Key
			m.SelectedIndex = 0
		}

	case "right", "l":
		// Navigate tabs
		currentTabIdx := -1
		for i, tab := range availableTabs {
			if tab.Key == m.CurrentTab {
				currentTabIdx = i
				break
			}
		}
		if currentTabIdx >= 0 {
			newIdx := currentTabIdx + 1
			if newIdx >= len(availableTabs) {
				newIdx = 0
			}
			m.CurrentTab = availableTabs[newIdx].Key
			m.SelectedIndex = 0
		}

	case "1", "2", "3", "4", "5":
		// Direct tab selection
		idx := int(key[0] - '1')
		if idx < len(availableTabs) {
			m.CurrentTab = availableTabs[idx].Key
			m.SelectedIndex = 0
		}

	case "enter", " ":
		return m.handleToggleAction()

	case "backspace":
		if m.CurrentTab == TabSettings {
			if len(m.SearchQuery) > 0 {
				m.SearchQuery = m.SearchQuery[:len(m.SearchQuery)-1]
				m.SelectedIndex = 0
			}
		}

	default:
		// Handle character input for search in settings tab
		if m.CurrentTab == TabSettings && len(key) == 1 {
			m.SearchQuery += key
			m.SelectedIndex = 0
		}
	}

	return m, nil
}

// handleToggleAction handles toggle/edit actions
func (m *ConfigModel) handleToggleAction() (tea.Model, tea.Cmd) {
	switch m.CurrentTab {
	case TabSettings:
		return m.handleSettingsAction()
	case TabRules:
		return m.handleRuleToggle()
	case TabWorkflows:
		return m.handleWorkflowToggle()
	case TabHooks:
		return m.handleHookToggle()
	case TabSkills:
		return m.handleSkillToggle()
	}
	return m, nil
}

// handleSettingsAction handles actions in the settings tab
func (m *ConfigModel) handleSettingsAction() (tea.Model, tea.Cmd) {
	filtered := m.getFilteredConfigEntries()
	if m.SelectedIndex >= len(filtered) {
		return m, nil
	}

	entry := filtered[m.SelectedIndex]
	if !entry.IsEditable {
		return m, nil
	}

	// For booleans, toggle directly
	if entry.Type == "boolean" {
		currentValue := getBoolValue(entry.Value)
		newValue := !currentValue

		var err error
		if entry.Source == "global" && m.OnUpdateGlobal != nil {
			err = m.OnUpdateGlobal(entry.Key, newValue)
		} else if entry.Source == "workspace" && m.OnUpdateWorkspace != nil {
			err = m.OnUpdateWorkspace(entry.Key, newValue)
		}

		if err != nil {
			m.Err = err
			return m, nil
		}

		// Refresh entries
		m.buildConfigEntries()
		return m, nil
	}

	// For other types, enter edit mode
	m.TextInput.SetValue(fmt.Sprintf("%v", entry.Value))
	m.TextInput.Focus()
	m.ViewState = ViewStateEditing
	return m, nil
}

// handleRuleToggle handles rule toggle
func (m *ConfigModel) handleRuleToggle() (tea.Model, tea.Cmd) {
	if m.SelectedIndex >= len(m.RuleEntries) {
		return m, nil
	}

	entry := m.RuleEntries[m.SelectedIndex]
	newValue := !entry.Enabled

	if m.OnToggleRule != nil {
		err := m.OnToggleRule(entry.Source == "global", entry.Key, newValue, entry.Type)
		if err != nil {
			m.Err = err
			return m, nil
		}
	}

	// Refresh entries
	m.buildRuleEntries()
	return m, nil
}

// handleWorkflowToggle handles workflow toggle
func (m *ConfigModel) handleWorkflowToggle() (tea.Model, tea.Cmd) {
	if m.SelectedIndex >= len(m.WorkflowEntries) {
		return m, nil
	}

	entry := m.WorkflowEntries[m.SelectedIndex]
	newValue := !entry.Enabled

	if m.OnToggleWorkflow != nil {
		err := m.OnToggleWorkflow(entry.Source == "global", entry.Key, newValue)
		if err != nil {
			m.Err = err
			return m, nil
		}
	}

	// Refresh entries
	m.buildWorkflowEntries()
	return m, nil
}

// handleHookToggle handles hook toggle
func (m *ConfigModel) handleHookToggle() (tea.Model, tea.Cmd) {
	if m.SelectedIndex >= len(m.HookEntries) {
		return m, nil
	}

	entry := m.HookEntries[m.SelectedIndex]
	newValue := !entry.Hook.Enabled

	if m.OnToggleHook != nil {
		err := m.OnToggleHook(entry.IsGlobal, entry.Hook.Name, newValue, entry.WorkspaceName)
		if err != nil {
			m.Err = err
			return m, nil
		}
	}

	// Refresh entries
	m.buildHookEntries()
	return m, nil
}

// handleSkillToggle handles skill toggle
func (m *ConfigModel) handleSkillToggle() (tea.Model, tea.Cmd) {
	if m.SelectedIndex >= len(m.SkillEntries) {
		return m, nil
	}

	entry := m.SkillEntries[m.SelectedIndex]
	newValue := !entry.Skill.Enabled

	if m.OnToggleSkill != nil {
		err := m.OnToggleSkill(entry.IsGlobal, entry.Skill.Path, newValue)
		if err != nil {
			m.Err = err
			return m, nil
		}
	}

	// Refresh entries
	m.buildSkillEntries()
	return m, nil
}

// View implements tea.Model
func (m ConfigModel) View() string {
	var b strings.Builder
	styles := DefaultConfigStyles()

	switch m.ViewState {
	case ViewStateEditing:
		b.WriteString(m.renderEditView(styles))
	case ViewStateList:
		b.WriteString(m.renderListView(styles))
	}

	return b.String()
}

// renderListView renders the list view
func (m *ConfigModel) renderListView(styles ConfigStyles) string {
	var sections []string

	// Header
	sections = append(sections, styles.TitleStyle.Render(m.Title))
	sections = append(sections, "")

	// Tabs
	tabsLine := m.renderTabs(styles)
	sections = append(sections, tabsLine)
	sections = append(sections, strings.Repeat("─", m.Width))
	sections = append(sections, "")

	// Tab content
	content := m.renderTabContent(styles)
	sections = append(sections, content)

	// Help text
	sections = append(sections, "")
	helpText := m.getHelpText()
	if m.Err != nil {
		sections = append(sections, styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	} else {
		sections = append(sections, styles.HelpStyle.Render(helpText))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTabs renders the tab bar
func (m *ConfigModel) renderTabs(styles ConfigStyles) string {
	var parts []string
	availableTabs := m.getAvailableTabs()

	for i, tab := range availableTabs {
		label := fmt.Sprintf(" %d:%s ", i+1, tab.Label)
		if tab.Key == m.CurrentTab {
			parts = append(parts, styles.SelectedTabStyle.Render(label))
		} else {
			parts = append(parts, styles.TabStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderTabContent renders content for the current tab
func (m *ConfigModel) renderTabContent(styles ConfigStyles) string {
	switch m.CurrentTab {
	case TabSettings:
		return m.renderSettingsTab(styles)
	case TabRules:
		return m.renderRulesTab(styles)
	case TabWorkflows:
		return m.renderWorkflowsTab(styles)
	case TabHooks:
		return m.renderHooksTab(styles)
	case TabSkills:
		return m.renderSkillsTab(styles)
	default:
		return ""
	}
}

// renderSettingsTab renders the settings tab
func (m *ConfigModel) renderSettingsTab(styles ConfigStyles) string {
	var items []string

	// Search and data dir info
	items = append(items, fmt.Sprintf("Search: %s", m.SearchQuery))
	items = append(items, fmt.Sprintf("Data directory: %s", m.DataDir))
	items = append(items, "")

	filtered := m.getFilteredConfigEntries()
	if len(filtered) == 0 {
		items = append(items, styles.DescriptionStyle.Render("No settings found."))
		return lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	// Group by source
	var currentSource string
	for i, entry := range filtered {
		// Show section header when source changes
		if entry.Source != currentSource {
			currentSource = entry.Source
			var header string
			if entry.Source == "global" {
				header = "Global Settings:"
			} else {
				header = "Workspace Settings:"
			}
			items = append(items, styles.SectionHeaderStyle.Render(header))
		}

		line := m.renderConfigEntry(entry, i == m.SelectedIndex, styles)
		items = append(items, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderConfigEntry renders a single config entry
func (m *ConfigModel) renderConfigEntry(entry ConfigEntry, selected bool, styles ConfigStyles) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, styles.SelectedStyle.Render("❯ "))
	} else {
		parts = append(parts, "  ")
	}

	// Label
	label := fmt.Sprintf("%s: ", entry.Label)
	if selected {
		parts = append(parts, styles.SelectedStyle.Render(label))
	} else {
		parts = append(parts, styles.ItemStyle.Render(label))
	}

	// Value
	valueStr := formatValueForDisplay(entry)
	if selected {
		parts = append(parts, styles.SelectedStyle.Render(valueStr))
	} else {
		if entry.Type == "boolean" {
			if getBoolValue(entry.Value) {
				parts = append(parts, styles.EnabledStyle.Render("✓ enabled"))
			} else {
				parts = append(parts, styles.DisabledStyle.Render("✗ disabled"))
			}
		} else {
			parts = append(parts, styles.ItemStyle.Render(valueStr))
		}
	}

	// Editable indicator
	if entry.IsEditable && selected {
		parts = append(parts, styles.HelpStyle.Render(" [edit]"))
	}

	line := lipgloss.JoinHorizontal(lipgloss.Left, parts...)

	// Description (on new line if selected)
	if selected && !entry.IsEditable {
		desc := styles.DescriptionStyle.Render("    (read-only)")
		line = lipgloss.JoinVertical(lipgloss.Left, line, desc)
	}

	return line
}

// formatValueForDisplay formats a value for display
func formatValueForDisplay(entry ConfigEntry) string {
	switch entry.Type {
	case "boolean":
		if getBoolValue(entry.Value) {
			return "enabled"
		}
		return "disabled"
	case "object", "array":
		data, _ := json.MarshalIndent(entry.Value, "", "  ")
		return string(data)
	default:
		str := fmt.Sprintf("%v", entry.Value)
		// Truncate long strings
		if len(str) > 50 {
			return str[:47] + "..."
		}
		return str
	}
}

// renderRulesTab renders the rules tab
func (m *ConfigModel) renderRulesTab(styles ConfigStyles) string {
	var items []string

	if len(m.RuleEntries) == 0 {
		items = append(items, styles.DescriptionStyle.Render("No rules configured. Add .clinerules files to your workspace or global config."))
		return lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	var currentSource string
	for i, entry := range m.RuleEntries {
		// Show section header when source changes
		if entry.Source != currentSource {
			currentSource = entry.Source
			var header string
			if entry.Source == "global" {
				header = "Global Rules:"
			} else {
				header = "Workspace Rules:"
			}
			items = append(items, styles.SectionHeaderStyle.Render(header))
		}

		line := m.renderToggleEntry(entry, i == m.SelectedIndex, styles, true)
		items = append(items, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderWorkflowsTab renders the workflows tab
func (m *ConfigModel) renderWorkflowsTab(styles ConfigStyles) string {
	var items []string

	if len(m.WorkflowEntries) == 0 {
		items = append(items, styles.DescriptionStyle.Render("No workflows configured. Add workflow files to enable this feature."))
		return lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	var currentSource string
	for i, entry := range m.WorkflowEntries {
		// Show section header when source changes
		if entry.Source != currentSource {
			currentSource = entry.Source
			var header string
			if entry.Source == "global" {
				header = "Global Workflows:"
			} else {
				header = "Workspace Workflows:"
			}
			items = append(items, styles.SectionHeaderStyle.Render(header))
		}

		line := m.renderToggleEntry(entry, i == m.SelectedIndex, styles, false)
		items = append(items, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderHooksTab renders the hooks tab
func (m *ConfigModel) renderHooksTab(styles ConfigStyles) string {
	var items []string

	if len(m.HookEntries) == 0 {
		items = append(items, styles.DescriptionStyle.Render("No hooks configured. Add hook scripts to enable automation."))
		return lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	var currentSection string
	for i, entry := range m.HookEntries {
		// Determine section
		var section string
		if entry.IsGlobal {
			section = "Global Hooks:"
		} else {
			section = fmt.Sprintf("%s Hooks:", entry.WorkspaceName)
		}

		// Show section header when changes
		if section != currentSection {
			currentSection = section
			items = append(items, styles.SectionHeaderStyle.Render(section))
		}

		line := m.renderHookEntry(entry.Hook, i == m.SelectedIndex, styles)
		items = append(items, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderSkillsTab renders the skills tab
func (m *ConfigModel) renderSkillsTab(styles ConfigStyles) string {
	var items []string

	if len(m.SkillEntries) == 0 {
		items = append(items, styles.DescriptionStyle.Render("No skills configured. Add SKILL.md files to enable skills."))
		return lipgloss.JoinVertical(lipgloss.Left, items...)
	}

	var currentSection string
	for i, entry := range m.SkillEntries {
		// Determine section
		var section string
		if entry.IsGlobal {
			section = "Global Skills:"
		} else {
			section = "Workspace Skills:"
		}

		// Show section header when changes
		if section != currentSection {
			currentSection = section
			items = append(items, styles.SectionHeaderStyle.Render(section))
		}

		line := m.renderSkillEntry(entry.Skill, i == m.SelectedIndex, styles)
		items = append(items, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderToggleEntry renders a toggle entry (rules, workflows)
func (m *ConfigModel) renderToggleEntry(entry ToggleEntry, selected bool, styles ConfigStyles, showType bool) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, styles.SelectedStyle.Render("❯ "))
	} else {
		parts = append(parts, "  ")
	}

	// Label
	if selected {
		parts = append(parts, styles.SelectedStyle.Render(entry.Label))
	} else {
		parts = append(parts, styles.ItemStyle.Render(entry.Label))
	}

	// Type indicator (for rules)
	if showType && entry.Type != "" {
		parts = append(parts, styles.HelpStyle.Render(fmt.Sprintf(" (%s)", entry.Type)))
	}

	// Status
	if entry.Enabled {
		parts = append(parts, styles.EnabledStyle.Render(" ✓"))
	} else {
		parts = append(parts, styles.DisabledStyle.Render(" ✗"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderHookEntry renders a hook entry
func (m *ConfigModel) renderHookEntry(hook HookInfo, selected bool, styles ConfigStyles) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, styles.SelectedStyle.Render("❯ "))
	} else {
		parts = append(parts, "  ")
	}

	// Name
	if selected {
		parts = append(parts, styles.SelectedStyle.Render(hook.Name))
	} else {
		parts = append(parts, styles.ItemStyle.Render(hook.Name))
	}

	// Status
	if hook.Enabled {
		parts = append(parts, styles.EnabledStyle.Render(" ✓"))
	} else {
		parts = append(parts, styles.DisabledStyle.Render(" ✗"))
	}

	// Description
	if selected && hook.Description != "" {
		line := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
		desc := styles.DescriptionStyle.Render(fmt.Sprintf("    %s", hook.Description))
		return lipgloss.JoinVertical(lipgloss.Left, line, desc)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderSkillEntry renders a skill entry
func (m *ConfigModel) renderSkillEntry(skill SkillInfo, selected bool, styles ConfigStyles) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, styles.SelectedStyle.Render("❯ "))
	} else {
		parts = append(parts, "  ")
	}

	// Name
	if selected {
		parts = append(parts, styles.SelectedStyle.Render(skill.Name))
	} else {
		parts = append(parts, styles.ItemStyle.Render(skill.Name))
	}

	// Status
	if skill.Enabled {
		parts = append(parts, styles.EnabledStyle.Render(" ✓"))
	} else {
		parts = append(parts, styles.DisabledStyle.Render(" ✗"))
	}

	// Description
	if selected && skill.Description != "" {
		line := lipgloss.JoinHorizontal(lipgloss.Left, parts...)
		desc := styles.DescriptionStyle.Render(fmt.Sprintf("    %s", skill.Description))
		return lipgloss.JoinVertical(lipgloss.Left, line, desc)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderEditView renders the edit view
func (m *ConfigModel) renderEditView(styles ConfigStyles) string {
	var sections []string

	// Header
	sections = append(sections, styles.TitleStyle.Render("✏️  Edit Configuration"))
	sections = append(sections, "")

	// Current item info
	filtered := m.getFilteredConfigEntries()
	if m.SelectedIndex < len(filtered) {
		entry := filtered[m.SelectedIndex]
		sections = append(sections, styles.HeaderStyle.Render(fmt.Sprintf("Editing: %s", entry.Label)))
		sections = append(sections, styles.DescriptionStyle.Render(fmt.Sprintf("Type: %s", entry.Type)))
		sections = append(sections, "")
	}

	// Input field
	inputBox := styles.BoxStyle.Width(m.Width - 4).Render(m.TextInput.View())
	sections = append(sections, inputBox)

	// Help text
	sections = append(sections, "")
	helpText := "Enter to save • Esc to cancel"
	if m.Err != nil {
		sections = append(sections, styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	} else {
		sections = append(sections, styles.HelpStyle.Render(helpText))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// getHelpText returns the appropriate help text for the current tab
func (m *ConfigModel) getHelpText() string {
	base := "↑/↓ Navigate • ←/→ Switch Tab • 1-5 Select Tab • q/Esc Exit"

	switch m.CurrentTab {
	case TabSettings:
		return base + " • Type to search • Enter Edit (booleans toggle) • Backspace clear search"
	default:
		return base + " • Enter/Space Toggle"
	}
}

// SetError sets an error message in the model
func (m *ConfigModel) SetError(err error) {
	m.Err = err
}

// GetSelectedItem returns the currently selected item based on current tab
func (m ConfigModel) GetSelectedItem() interface{} {
	switch m.CurrentTab {
	case TabSettings:
		filtered := m.getFilteredConfigEntries()
		if m.SelectedIndex < len(filtered) {
			return &filtered[m.SelectedIndex]
		}
	case TabRules:
		if m.SelectedIndex < len(m.RuleEntries) {
			return &m.RuleEntries[m.SelectedIndex]
		}
	case TabWorkflows:
		if m.SelectedIndex < len(m.WorkflowEntries) {
			return &m.WorkflowEntries[m.SelectedIndex]
		}
	case TabHooks:
		if m.SelectedIndex < len(m.HookEntries) {
			return &m.HookEntries[m.SelectedIndex]
		}
	case TabSkills:
		if m.SelectedIndex < len(m.SkillEntries) {
			return &m.SkillEntries[m.SelectedIndex]
		}
	}
	return nil
}

// ConfigProgram wraps the config TUI program
type ConfigProgram struct {
	configModel ConfigModel
	prog        *tea.Program
}

// NewConfigProgram creates a new config TUI program
func NewConfigProgram(storageCtx *storage.StorageContext, dataDir string) *ConfigProgram {
	m := NewConfigModel(storageCtx, dataDir)
	return &ConfigProgram{
		configModel: m,
	}
}

// SetCallbacks sets the callbacks for the config program
func (p *ConfigProgram) SetCallbacks(
	onUpdateGlobal func(key string, value interface{}) error,
	onUpdateWorkspace func(key string, value interface{}) error,
	onQuit func(),
) {
	p.configModel.OnUpdateGlobal = onUpdateGlobal
	p.configModel.OnUpdateWorkspace = onUpdateWorkspace
	p.configModel.OnQuit = onQuit
}

// SetToggleCallbacks sets the toggle callbacks for the config program
func (p *ConfigProgram) SetToggleCallbacks(
	onToggleRule func(isGlobal bool, rulePath string, enabled bool, ruleType string) error,
	onToggleWorkflow func(isGlobal bool, workflowPath string, enabled bool) error,
	onToggleHook func(isGlobal bool, hookName string, enabled bool, workspaceName string) error,
	onToggleSkill func(isGlobal bool, skillPath string, enabled bool) error,
) {
	p.configModel.OnToggleRule = onToggleRule
	p.configModel.OnToggleWorkflow = onToggleWorkflow
	p.configModel.OnToggleHook = onToggleHook
	p.configModel.OnToggleSkill = onToggleSkill
}

// Run runs the config TUI program
func (p *ConfigProgram) Run() error {
	p.prog = tea.NewProgram(p.configModel, tea.WithAltScreen())
	_, err := p.prog.Run()
	return err
}

// ExportConfig exports the configuration as JSON
func (m ConfigModel) ExportConfig() (map[string]interface{}, error) {
	configData := make(map[string]interface{})

	// Add global state
	if m.StorageCtx != nil && m.StorageCtx.GlobalState != nil {
		configData["global"] = m.StorageCtx.GlobalState.GetAll()
	}

	// Add workspace state if available
	if m.StorageCtx != nil && m.StorageCtx.WorkspaceState != nil {
		configData["workspace"] = m.StorageCtx.WorkspaceState.GetAll()
	}

	return configData, nil
}
