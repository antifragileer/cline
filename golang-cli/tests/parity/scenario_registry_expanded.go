// Package parity provides extended test scenarios for comprehensive parity testing.
package parity

import (
	"fmt"
	"time"
)

// ExtendedRegistry provides additional test scenarios beyond the base registry
type ExtendedRegistry struct {
	*Registry
	extendedScenarios []Scenario
}

// NewExtendedRegistry creates a new extended registry with 50+ scenarios
func NewExtendedRegistry() *ExtendedRegistry {
	r := &ExtendedRegistry{
		Registry: NewRegistry(),
	}
	r.registerExtendedScenarios()
	return r
}

// GetAll returns all scenarios including extended ones
func (r *ExtendedRegistry) GetAll() []Scenario {
	base := r.Registry.GetAll()
	return append(base, r.extendedScenarios...)
}

// GetExtended returns only extended scenarios
func (r *ExtendedRegistry) GetExtended() []Scenario {
	return r.extendedScenarios
}

// GetByCategory returns scenarios filtered by category (including extended)
func (r *ExtendedRegistry) GetByCategory(cat Category) []Scenario {
	all := r.GetAll()
	var filtered []Scenario
	for _, s := range all {
		if s.Category == cat {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// Count returns total number of scenarios
func (r *ExtendedRegistry) Count() int {
	return len(r.GetAll())
}

func (r *ExtendedRegistry) registerExtendedScenarios() {
	r.addTUIScenarios()
	r.addSettingsScenarios()
	r.addApprovalScenarios()
	r.addOAuthScenarios()
	r.addStreamingScenarios()
	r.addSecurityScenarios()
	r.addMCPScenarios()
	r.addCheckpointScenarios()
	r.addBrowserScenarios()
	r.addDiffScenarios()
	r.addModeScenarios()
	r.addConfigScenarios()
	r.addHistoryScenarios()
	r.addTaskManagementScenarios()
}

func (r *ExtendedRegistry) addTUIScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "tui_interactive_mode",
			Description:    "TUI interactive mode startup",
			Category:       CategoryTUI,
			Args:           []string{"--interactive"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeInteractive,
			ExpectExitCode: 0,
			ExpectBaseline: false, // TUI is visual
		},
		{
			Name:           "tui_welcome_screen",
			Description:    "TUI welcome screen display",
			Category:       CategoryTUI,
			Args:           []string{"tui", "welcome"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "tui_chat_mode",
			Description:    "TUI chat interface",
			Category:       CategoryTUI,
			Args:           []string{"tui", "chat"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeInteractive,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "tui_with_prompt",
			Description:    "TUI with initial prompt",
			Category:       CategoryTUI,
			Args:           []string{"--interactive", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModeInteractive,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "tui_mouse_support",
			Description:    "TUI with mouse support",
			Category:       CategoryTUI,
			Args:           []string{"--interactive", "--mouse"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeInteractive,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "tui_alt_screen",
			Description:    "TUI alternate screen buffer",
			Category:       CategoryTUI,
			Args:           []string{"--interactive", "--alt-screen"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeInteractive,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addSettingsScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "settings_show",
			Description:    "Show all settings",
			Category:       CategorySettings,
			Args:           []string{"settings"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "settings_json",
			Description:    "Settings in JSON format",
			Category:       CategorySettings,
			Args:           []string{"settings", "--json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "settings_get",
			Description:    "Get specific setting",
			Category:       CategorySettings,
			Args:           []string{"settings", "get", "provider"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "settings_set",
			Description:    "Set a setting value",
			Category:       CategorySettings,
			Args:           []string{"settings", "set", "test.key", "test.value"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "settings_reset",
			Description:    "Reset settings to default",
			Category:       CategorySettings,
			Args:           []string{"settings", "reset"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "settings_provider",
			Description:    "Configure API provider",
			Category:       CategorySettings,
			Args:           []string{"settings", "provider"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "settings_model",
			Description:    "Configure model settings",
			Category:       CategorySettings,
			Args:           []string{"settings", "model"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *ExtendedRegistry) addApprovalScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "approval_auto_approve",
			Description:    "Auto-approve all actions",
			Category:       CategoryTask,
			Args:           []string{"-y", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "approval_read_only",
			Description:    "Read-only mode (no approvals)",
			Category:       CategoryTask,
			Args:           []string{"--read-only", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "approval_dangerous_requires_confirm",
			Description:    "Dangerous actions require confirmation",
			Category:       CategoryTask,
			Args:           []string{"task", "delete all files"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "approval_list_auto_approved",
			Description:    "List auto-approved tools",
			Category:       CategoryConfig,
			Args:           []string{"config", "auto-approve", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "approval_add_auto_approve",
			Description:    "Add tool to auto-approve list",
			Category:       CategoryConfig,
			Args:           []string{"config", "auto-approve", "add", "read_file"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addOAuthScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "oauth_list_providers",
			Description:    "List OAuth providers",
			Category:       CategoryAuth,
			Args:           []string{"auth", "oauth", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "oauth_status",
			Description:    "Check OAuth status",
			Category:       CategoryAuth,
			Args:           []string{"auth", "oauth", "status"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "oauth_login",
			Description:    "OAuth login flow",
			Category:       CategoryAuth,
			Args:           []string{"auth", "oauth", "login"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "oauth_logout",
			Description:    "OAuth logout",
			Category:       CategoryAuth,
			Args:           []string{"auth", "oauth", "logout"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "oauth_token_refresh",
			Description:    "Refresh OAuth token",
			Category:       CategoryAuth,
			Args:           []string{"auth", "oauth", "refresh"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addStreamingScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "stream_json_lines",
			Description:    "Streaming JSON lines output",
			Category:       CategoryTask,
			Args:           []string{"--jsonl", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "stream_events",
			Description:    "Event stream output",
			Category:       CategoryTask,
			Args:           []string{"--events", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "stream_sse",
			Description:    "Server-sent events output",
			Category:       CategoryTask,
			Args:           []string{"--sse", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "stream_raw",
			Description:    "Raw streaming output",
			Category:       CategoryTask,
			Args:           []string{"--raw", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "stream_with_progress",
			Description:    "Stream with progress indicators",
			Category:       CategoryTask,
			Args:           []string{"--progress", "task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addSecurityScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "security_validate_command",
			Description:    "Validate command safety",
			Category:       CategoryConfig,
			Args:           []string{"security", "validate", "rm -rf /"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 1, // Should reject dangerous command
			ExpectBaseline: false,
		},
		{
			Name:           "security_check_permissions",
			Description:    "Check file permissions",
			Category:       CategoryConfig,
			Args:           []string{"security", "permissions"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "security_audit_log",
			Description:    "View security audit log",
			Category:       CategoryConfig,
			Args:           []string{"security", "audit"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "security_dangerous_patterns",
			Description:    "List dangerous command patterns",
			Category:       CategoryConfig,
			Args:           []string{"security", "patterns"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *ExtendedRegistry) addMCPScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "mcp_list_servers",
			Description:    "List MCP servers",
			Category:       CategoryConfig,
			Args:           []string{"mcp", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "mcp_add_server",
			Description:    "Add MCP server",
			Category:       CategoryConfig,
			Args:           []string{"mcp", "add", "test-server"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "mcp_remove_server",
			Description:    "Remove MCP server",
			Category:       CategoryConfig,
			Args:           []string{"mcp", "remove", "test-server"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "mcp_test_connection",
			Description:    "Test MCP server connection",
			Category:       CategoryConfig,
			Args:           []string{"mcp", "test"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addCheckpointScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "checkpoint_list",
			Description:    "List checkpoints",
			Category:       CategoryHistory,
			Args:           []string{"checkpoint", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "checkpoint_create",
			Description:    "Create checkpoint",
			Category:       CategoryHistory,
			Args:           []string{"checkpoint", "create", "test-checkpoint"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "checkpoint_restore",
			Description:    "Restore checkpoint",
			Category:       CategoryHistory,
			Args:           []string{"checkpoint", "restore", "abc123"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "checkpoint_delete",
			Description:    "Delete checkpoint",
			Category:       CategoryHistory,
			Args:           []string{"checkpoint", "delete", "abc123"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addBrowserScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "browser_launch",
			Description:    "Launch browser session",
			Category:       CategoryTask,
			Args:           []string{"browser", "launch"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "browser_navigate",
			Description:    "Navigate to URL",
			Category:       CategoryTask,
			Args:           []string{"browser", "navigate", "https://example.com"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "browser_screenshot",
			Description:    "Take browser screenshot",
			Category:       CategoryTask,
			Args:           []string{"browser", "screenshot"},
			Timeout:        30 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "browser_close",
			Description:    "Close browser session",
			Category:       CategoryTask,
			Args:           []string{"browser", "close"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addDiffScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "diff_view",
			Description:    "View diff",
			Category:       CategoryTask,
			Args:           []string{"diff", "view"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "diff_apply",
			Description:    "Apply diff",
			Category:       CategoryTask,
			Args:           []string{"diff", "apply"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "diff_revert",
			Description:    "Revert diff",
			Category:       CategoryTask,
			Args:           []string{"diff", "revert"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addModeScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "mode_switch_act",
			Description:    "Switch to act mode",
			Category:       CategoryConfig,
			Args:           []string{"mode", "act"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "mode_switch_plan",
			Description:    "Switch to plan mode",
			Category:       CategoryConfig,
			Args:           []string{"mode", "plan"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "mode_show_current",
			Description:    "Show current mode",
			Category:       CategoryConfig,
			Args:           []string{"mode"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "mode_with_prompt",
			Description:    "Mode with immediate prompt",
			Category:       CategoryTask,
			Args:           []string{"-m", "act", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addConfigScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "config_edit",
			Description:    "Edit configuration",
			Category:       CategoryConfig,
			Args:           []string{"config", "edit"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "config_path",
			Description:    "Show config file path",
			Category:       CategoryConfig,
			Args:           []string{"config", "path"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "config_backup",
			Description:    "Backup configuration",
			Category:       CategoryConfig,
			Args:           []string{"config", "backup"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "config_restore",
			Description:    "Restore configuration",
			Category:       CategoryConfig,
			Args:           []string{"config", "restore"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "config_import",
			Description:    "Import configuration",
			Category:       CategoryConfig,
			Args:           []string{"config", "import", "config.json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "config_export",
			Description:    "Export configuration",
			Category:       CategoryConfig,
			Args:           []string{"config", "export"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addHistoryScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "history_search",
			Description:    "Search history",
			Category:       CategoryHistory,
			Args:           []string{"history", "search", "test"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "history_clear",
			Description:    "Clear history",
			Category:       CategoryHistory,
			Args:           []string{"history", "clear"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "history_export",
			Description:    "Export history",
			Category:       CategoryHistory,
			Args:           []string{"history", "export", "history.json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "history_import",
			Description:    "Import history",
			Category:       CategoryHistory,
			Args:           []string{"history", "import", "history.json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "history_show_task",
			Description:    "Show specific task from history",
			Category:       CategoryHistory,
			Args:           []string{"history", "show", "task-123"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "history_delete_task",
			Description:    "Delete task from history",
			Category:       CategoryHistory,
			Args:           []string{"history", "delete", "task-123"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *ExtendedRegistry) addTaskManagementScenarios() {
	r.extendedScenarios = append(r.extendedScenarios, []Scenario{
		{
			Name:           "task_resume",
			Description:    "Resume previous task",
			Category:       CategoryTask,
			Args:           []string{"task", "resume"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_resume_specific",
			Description:    "Resume specific task",
			Category:       CategoryTask,
			Args:           []string{"task", "resume", "task-123"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_cancel",
			Description:    "Cancel current task",
			Category:       CategoryTask,
			Args:           []string{"task", "cancel"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_status",
			Description:    "Show task status",
			Category:       CategoryTask,
			Args:           []string{"task", "status"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_list",
			Description:    "List running tasks",
			Category:       CategoryTask,
			Args:           []string{"task", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "task_attach",
			Description:    "Attach to running task",
			Category:       CategoryTask,
			Args:           []string{"task", "attach", "task-123"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_detach",
			Description:    "Detach from task",
			Category:       CategoryTask,
			Args:           []string{"task", "detach"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_with_file",
			Description:    "Task with file input",
			Category:       CategoryTask,
			Args:           []string{"task", "-f", "input.txt"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_with_image",
			Description:    "Task with image input",
			Category:       CategoryTask,
			Args:           []string{"task", "-i", "image.png", "describe this"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_with_context",
			Description:    "Task with context directory",
			Category:       CategoryTask,
			Args:           []string{"task", "-C", "/tmp", "list files"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

// GetTotalScenarios returns the total count of all scenarios
func GetTotalScenarios() int {
	registry := NewExtendedRegistry()
	return registry.Count()
}

// PrintScenarioSummary prints a summary of all test scenarios
func PrintScenarioSummary() {
	registry := NewExtendedRegistry()
	all := registry.GetAll()

	fmt.Printf("Total Scenarios: %d\n", len(all))
	fmt.Printf("Categories:\n")

	categories := make(map[Category]int)
	for _, s := range all {
		categories[s.Category]++
	}

	for cat, count := range categories {
		fmt.Printf("  - %s: %d\n", cat, count)
	}
}