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

// hooksAddCmd represents the hooks add subcommand
var hooksAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new hook",
	Long: `Add a new hook to Cline's configuration.

Hooks are custom scripts that run at specific points during Cline's execution.
Available hook types:
  - before-task: Runs before a task starts
  - after-task: Runs after a task completes
  - before-tool: Runs before a tool is executed
  - after-tool: Runs after a tool is executed
  - on-error: Runs when an error occurs`,
	Example: `  # Add a hook that runs before each task
  cline hooks add pre-check --type before-task --script "echo 'Starting task'"

  # Add a hook from a file
  cline hooks add validate --type before-tool --file ./validate.sh`,
	Args: cobra.ExactArgs(1),
	RunE: runHooksAdd,
}

// hooksRemoveCmd represents the hooks remove subcommand
var hooksRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a hook",
	Long:    `Remove a hook from Cline's configuration permanently.`,
	Example: `  cline hooks remove my-hook
  cline hooks rm my-hook`,
	Args: cobra.ExactArgs(1),
	RunE: runHooksRemove,
}

func init() {
	rootCmd.AddCommand(hooksCmd)

	// Add subcommands
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksEnableCmd)
	hooksCmd.AddCommand(hooksDisableCmd)
	hooksCmd.AddCommand(hooksAddCmd)
	hooksCmd.AddCommand(hooksRemoveCmd)

	// Add flags
	hooksListCmd.Flags().BoolVarP(&hooksFlags.json, "json", "j", false, "Output in JSON format")
	hooksListCmd.Flags().BoolVarP(&hooksFlags.global, "global", "g", false, "Show only global hooks")

	// Add command flags
	hooksAddCmd.Flags().String("type", "before-task", "Hook type (before-task, after-task, before-tool, after-tool, on-error)")
	hooksAddCmd.Flags().String("script", "", "Hook script content")
	hooksAddCmd.Flags().String("file", "", "Path to script file")
	hooksAddCmd.Flags().String("description", "", "Hook description")
	hooksAddCmd.Flags().BoolVarP(&hooksFlags.global, "global", "g", false, "Add as global hook")
}

// runHooksList executes the hooks list command
func runHooksList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Reload to get latest data from disk
	if err := ctx.GlobalState.Reload(); err != nil && verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: failed to reload: %v\n", err)
	}

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: storage path: %s\n", ctx.GlobalState.FilePath())
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: all keys: %v\n", getKeys(ctx.GlobalState.GetAll()))
	}

	// Load global hooks
	var globalHooks []Hook
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: found globalHooks, type=%T\n", val)
		}
		if hooks, err := parseHooks(val); err == nil {
			globalHooks = hooks
		} else if verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: failed to parse hooks: %v\n", err)
		}
	} else if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: globalHooks not found in storage\n")
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

	// Reload to get latest data from disk
	ctx.GlobalState.Reload()

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

	// Reload to get latest data from disk
	ctx.GlobalState.Reload()

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Looking for hook '%s'\n", hookName)
	}

	// Try to find and disable the hook in global hooks first
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Found globalHooks in storage\n")
		}
		if hooks, err := parseHooks(val); err == nil {
			if verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Parsed %d hooks\n", len(hooks))
			}
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
		} else if verbose {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Failed to parse hooks: %v\n", err)
		}
	} else if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: globalHooks not found in storage\n")
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

// getKeys returns all keys from a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// runHooksAdd executes the hooks add command
func runHooksAdd(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	// Get flags
	hookType, _ := cmd.Flags().GetString("type")
	script, _ := cmd.Flags().GetString("script")
	scriptFile, _ := cmd.Flags().GetString("file")
	description, _ := cmd.Flags().GetString("description")
	isGlobal, _ := cmd.Flags().GetBool("global")

	// If script file is provided, read it
	if scriptFile != "" {
		content, err := os.ReadFile(scriptFile)
		if err != nil {
			return fmt.Errorf("failed to read script file: %w", err)
		}
		script = string(content)
	}

	// Validate that we have a script
	if strings.TrimSpace(script) == "" {
		return fmt.Errorf("hook script is required (use --script or --file)")
	}

	// Create the hook
	hook, err := CreateHook(hookName, script, hookType, isGlobal)
	if err != nil {
		return err
	}

	if description != "" {
		hook.Description = description
	}

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Add to appropriate storage
	if isGlobal {
		// Load existing hooks
		var hooks []Hook
		if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
			if existingHooks, err := parseHooks(val); err == nil {
				hooks = existingHooks
			}
		}

		// Check for duplicate
		for _, h := range hooks {
			if h.Name == hookName {
				return fmt.Errorf("hook '%s' already exists", hookName)
			}
		}

		// Add new hook
		hooks = append(hooks, *hook)
		if err := ctx.GlobalState.Set("globalHooks", hooks); err != nil {
			return fmt.Errorf("failed to save hook: %w", err)
		}
	} else {
		// Load existing workspace hooks
		var wsHooks []WorkspaceHooks
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if existing, err := parseWorkspaceHooks(val); err == nil {
				wsHooks = existing
			}
		}

		// Find or create workspace entry
		workspaceName := getWorkspaceHash()
		found := false
		for i, ws := range wsHooks {
			if ws.WorkspaceName == workspaceName {
				// Check for duplicate
				for _, h := range ws.Hooks {
					if h.Name == hookName {
						return fmt.Errorf("hook '%s' already exists in this workspace", hookName)
					}
				}
				wsHooks[i].Hooks = append(wsHooks[i].Hooks, *hook)
				found = true
				break
			}
		}

		if !found {
			wsHooks = append(wsHooks, WorkspaceHooks{
				WorkspaceName: workspaceName,
				Hooks:         []Hook{*hook},
			})
		}

		if err := ctx.WorkspaceState.Set("workspaceHooks", wsHooks); err != nil {
			return fmt.Errorf("failed to save hook: %w", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' added successfully\n", hookName)
	return nil
}

// runHooksRemove executes the hooks remove command
func runHooksRemove(cmd *cobra.Command, args []string) error {
	hookName := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Reload to get latest data from disk
	ctx.GlobalState.Reload()

	// Try to remove from global hooks first
	if val, ok := ctx.GlobalState.Get("globalHooks"); ok {
		if hooks, err := parseHooks(val); err == nil {
			for i, hook := range hooks {
				if hook.Name == hookName {
					// Remove this hook
					hooks = append(hooks[:i], hooks[i+1:]...)
					if err := ctx.GlobalState.Set("globalHooks", hooks); err != nil {
						return fmt.Errorf("failed to remove hook: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' removed\n", hookName)
					return nil
				}
			}
		}
	}

	// Try to remove from workspace hooks
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("workspaceHooks"); ok {
			if wsHooks, err := parseWorkspaceHooks(val); err == nil {
				for wi, ws := range wsHooks {
					for hi, hook := range ws.Hooks {
						if hook.Name == hookName {
							// Remove this hook
							wsHooks[wi].Hooks = append(ws.Hooks[:hi], ws.Hooks[hi+1:]...)
							if err := ctx.WorkspaceState.Set("workspaceHooks", wsHooks); err != nil {
								return fmt.Errorf("failed to remove hook: %w", err)
							}
							fmt.Fprintf(cmd.OutOrStdout(), "✓ Hook '%s' removed\n", hookName)
							return nil
						}
					}
				}
			}
		}
	}

	return fmt.Errorf("hook '%s' not found", hookName)
}
