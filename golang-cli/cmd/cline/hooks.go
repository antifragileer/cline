package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// Hook represents a Cline hook configuration
type Hook struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Script      string `json:"script"`
	Enabled     bool   `json:"enabled"`
	Type        string `json:"type,omitempty"`
}

// WorkspaceHooks represents hooks for a specific workspace
type WorkspaceHooks struct {
	WorkspaceName string `json:"workspaceName"`
	Hooks         []Hook `json:"hooks"`
}

// hooksFlags holds flags for hooks commands
var hooksFlags struct {
	json    bool
	global  bool
	enabled bool
}

// hooksCmd represents the hooks command
var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Cline hooks",
	Long: `Manage Cline hooks for custom automation and extensions.

Hooks allow you to run custom scripts at various points during Cline's
execution, enabling powerful automation and customization.`,
	Example: `  # List all hooks
  cline hooks list

  # List hooks in JSON format
  cline hooks list --json

  # List global hooks
  cline hooks list --global`,
}

// hooksListCmd represents the hooks list subcommand
var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured hooks",
	Long:  `List all configured hooks with their status and description.`,
	Example: `  cline hooks list
  cline hooks list --json
  cline hooks list --global`,
	RunE: runHooksList,
}

// hooksEnableCmd represents the hooks enable subcommand
var hooksEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a hook",
	Long:  `Enable a previously disabled hook.`,
	Example: `  cline hooks enable my-hook`,
	Args:  cobra.ExactArgs(1),
	RunE:  runHooksEnable,
}

// hooksDisableCmd represents the hooks disable subcommand
var hooksDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a hook",
	Long:  `Disable a hook without removing it from configuration.`,
	Example: `  cline hooks disable my-hook`,
	Args:  cobra.ExactArgs(1),
	RunE:  runHooksDisable,
}

func init() {
	rootCmd.AddCommand(hooksCmd)

	// Add subcommands
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksEnableCmd)
	hooksCmd.AddCommand(hooksDisableCmd)

	// Add flags
	hooksListCmd.Flags().BoolVarP(&hooksFlags.json, "json", "j", false, "Output in JSON format")
	hooksListCmd.Flags().BoolVarP(&hooksFlags.global, "global", "g", false, "Show only global hooks")
}

// runHooksList executes the hooks list command
func runHooksList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load global hooks
	var globalHooks []Hook
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if hooks, err := parseHooks(val); err == nil {
			globalHooks = hooks
		}
	}

	// Load workspace hooks
	var workspaceHooks []WorkspaceHooks
	if !hooksFlags.global && ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if wh, err := parseWorkspaceHooks(val); err == nil {
				workspaceHooks = wh
			}
		}
	}

	// Output as JSON if requested
	if hooksFlags.json {
		output := map[string]interface{}{
			"global": globalHooks,
		}
		if !hooksFlags.global {
			output["workspaces"] = workspaceHooks
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	if len(globalHooks) == 0 && len(workspaceHooks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No hooks configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nHooks allow you to run custom scripts during Cline's execution.")
		return nil
	}

	// Display global hooks
	if len(globalHooks) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Global Hooks ===")
		for _, hook := range globalHooks {
			printHook(cmd.OutOrStdout(), hook)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Display workspace hooks
	if !hooksFlags.global && len(workspaceHooks) > 0 {
		for _, ws := range workspaceHooks {
			fmt.Fprintf(cmd.OutOrStdout(), "=== Workspace: %s ===\n", ws.WorkspaceName)
			for _, hook := range ws.Hooks {
				printHook(cmd.OutOrStdout(), hook)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}

// printHook prints a single hook in human-readable format
func printHook(w io.Writer, hook Hook) {
	status := "✓ enabled"
	if !hook.Enabled {
		status = "✗ disabled"
	}

	fmt.Fprintf(w, "  %s %s\n", status, hook.Name)
	if hook.Description != "" {
		fmt.Fprintf(w, "    Description: %s\n", hook.Description)
	}
	if hook.Type != "" {
		fmt.Fprintf(w, "    Type: %s\n", hook.Type)
	}
	if hook.Script != "" {
		// Truncate long scripts
		script := hook.Script
		if len(script) > 60 {
			script = script[:57] + "..."
		}
		fmt.Fprintf(w, "    Script: %s\n", script)
	}
}

// runHooksEnable executes the hooks enable command
func runHooksEnable(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try to find and enable the hook in global hooks first
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if hooks, err := parseHooks(val); err == nil {
			for i, hook := range hooks {
				if hook.Name == hookName {
					hooks[i].Enabled = true
					if err := ctx.GlobalState.Set("globalHooks", hooks); err != nil {
						return fmt.Errorf("failed to enable hook: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' enabled\n", hookName)
					return nil
				}
			}
		}
	}

	// Try workspace hooks
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if wsHooks, err := parseWorkspaceHooks(val); err == nil {
				updated := false
				for wi, ws := range wsHooks {
					for hi, hook := range ws.Hooks {
						if hook.Name == hookName {
							wsHooks[wi].Hooks[hi].Enabled = true
							updated = true
							break
						}
					}
				}
				if updated {
					if err := ctx.WorkspaceState.Set("workspaceHooks", wsHooks); err != nil {
						return fmt.Errorf("failed to enable hook: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' enabled\n", hookName)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("hook '%s' not found", hookName)
}

// runHooksDisable executes the hooks disable command
func runHooksDisable(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try to find and disable the hook in global hooks first
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if hooks, err := parseHooks(val); err == nil {
			for i, hook := range hooks {
				if hook.Name == hookName {
					hooks[i].Enabled = false
					if err := ctx.GlobalState.Set("globalHooks", hooks); err != nil {
						return fmt.Errorf("failed to disable hook: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' disabled\n", hookName)
					return nil
				}
			}
		}
	}

	// Try workspace hooks
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if wsHooks, err := parseWorkspaceHooks(val); err == nil {
				updated := false
				for wi, ws := range wsHooks {
					for hi, hook := range ws.Hooks {
						if hook.Name == hookName {
							wsHooks[wi].Hooks[hi].Enabled = false
							updated = true
							break
						}
					}
				}
				if updated {
					if err := ctx.WorkspaceState.Set("workspaceHooks", wsHooks); err != nil {
						return fmt.Errorf("failed to disable hook: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' disabled\n", hookName)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("hook '%s' not found", hookName)
}

// parseHooks parses hook data from storage
func parseHooks(data interface{}) ([]Hook, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var hooks []Hook
	if err := json.Unmarshal(jsonData, &hooks); err != nil {
		return nil, err
	}

	return hooks, nil
}

// parseWorkspaceHooks parses workspace hooks data from storage
func parseWorkspaceHooks(data interface{}) ([]WorkspaceHooks, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var wsHooks []WorkspaceHooks
	if err := json.Unmarshal(jsonData, &wsHooks); err != nil {
		return nil, err
	}

	return wsHooks, nil
}

// GetHooksDir returns the hooks directory path
func GetHooksDir(global bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		return filepath.Join(homeDir, ".cline", "hooks"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, ".cline", "hooks"), nil
}

// ListAvailableHooks returns a list of available hook types
func ListAvailableHooks() []string {
	return []string{
		"before-task",
		"after-task",
		"before-tool",
		"after-tool",
		"on-error",
	}
}

// ValidateHookScript validates a hook script
func ValidateHookScript(script string) error {
	if strings.TrimSpace(script) == "" {
		return fmt.Errorf("hook script cannot be empty")
	}

	// Check for common issues
	if len(script) > 10000 {
		return fmt.Errorf("hook script is too long (max 10000 characters)")
	}

	return nil
}

// CreateHook creates a new hook
func CreateHook(name, script, hookType string, global bool) (*Hook, error) {
	// Validate inputs
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("hook name cannot be empty")
	}

	if err := ValidateHookScript(script); err != nil {
		return nil, err
	}

	// Validate hook type
	validTypes := ListAvailableHooks()
	valid := false
	for _, t := range validTypes {
		if t == hookType {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid hook type: %s (valid types: %s)", hookType, strings.Join(validTypes, ", "))
	}

	hook := &Hook{
		Name:        name,
		Script:      script,
		Type:        hookType,
		Enabled:     true,
		Description: fmt.Sprintf("%s hook", hookType),
	}

	return hook, nil
}