package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/cline/cline/golang-cli/internal/tui"
)

// configFlags holds the parsed flag values for config command
var configFlags struct {
	json   bool
	edit   bool
	global bool
	list   bool
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display and manage Cline configuration",
	Long: `Display and manage Cline configuration state.

This command allows you to view and edit global and workspace-specific
configuration settings. Configuration is stored in JSON files and can
be displayed in human-readable or JSON format.`,
	Example: `  # Display all configuration
  cline config

  # Display configuration as JSON
  cline config --json

  # Display only global configuration
  cline config --global

  # Edit configuration in default editor
  cline config --edit`,
	RunE: runConfig,
}

// configListCmd represents the config list subcommand
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration settings",
	Long:  `List all configuration settings in human-readable or JSON format.`,
	Example: `  cline config list
  cline config list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFlags.list = true
		return runConfig(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configDeleteCmd)

	// Add flags to config list subcommand
	configListCmd.Flags().BoolVarP(&configFlags.json, "json", "j", false, "Output in JSON format")
	configListCmd.Flags().BoolVarP(&configFlags.global, "global", "g", false, "Show only global configuration")

	// Add flags to config set subcommand
	configSetCmd.Flags().BoolVarP(&configFlags.global, "global", "g", false, "Set in global configuration")

	// Add flags to config get subcommand
	configGetCmd.Flags().BoolVarP(&configFlags.global, "global", "g", false, "Get from global configuration")

	// Add flags to config delete subcommand
	configDeleteCmd.Flags().BoolVarP(&configFlags.global, "global", "g", false, "Delete from global configuration")
}

// configSetCmd represents the config set subcommand
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  `Set a configuration value in global or workspace state.`,
	Example: `  # Set a global config value
  cline config set apiProvider anthropic

  # Set a workspace config value
  cline config set customModel gpt-4 --global`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

// configGetCmd represents the config get subcommand
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  `Get a configuration value from global or workspace state.`,
	Example: `  # Get a config value
  cline config get apiProvider

  # Get a global config value
  cline config get apiProvider --global`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

// configDeleteCmd represents the config delete subcommand
var configDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a configuration value",
	Long:  `Delete a configuration value from global or workspace state.`,
	Example: `  # Delete a config value
  cline config delete customModel

  # Delete a global config value
  cline config delete customModel --global`,
	Aliases: []string{"del", "rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runConfigDelete,
}

// runConfigSet executes the config set subcommand
func runConfigSet(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Get the --global flag value
	global, _ := cmd.Flags().GetBool("global")
	
	return setConfigValue(ctx, args[0], args[1], global)
}

// runConfigGet executes the config get subcommand
func runConfigGet(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Get the --global flag value
	global, _ := cmd.Flags().GetBool("global")
	
	return getConfigValue(ctx, args[0], global)
}

// runConfigDelete executes the config delete subcommand
func runConfigDelete(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Get the --global flag value
	global, _ := cmd.Flags().GetBool("global")

	storage := ctx.GlobalState
	if !global && ctx.WorkspaceState != nil {
		storage = ctx.WorkspaceState
	}

	if err := storage.Delete(args[0]); err != nil {
		return fmt.Errorf("failed to delete config value: %w", err)
	}

	fmt.Printf("Deleted %s\n", args[0])
	return nil
}

// runConfig executes the config command
func runConfig(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Handle list subcommand (from configListCmd)
	if configFlags.list {
		if configFlags.json {
			return displayConfigJSON(ctx, configFlags.global)
		}
		return displayConfigHuman(ctx, configFlags.global)
	}

	// Handle edit mode
	if configFlags.edit {
		return editConfig(ctx)
	}

	// Handle JSON output mode
	if configFlags.json {
		return displayConfigJSON(ctx, configFlags.global)
	}

	// Check if we should use interactive TUI mode
	// TUI is used when: no flags are provided AND stdout is a terminal
	if shouldUseInteractiveMode() {
		return runInteractiveConfig(ctx)
	}

	// Display configuration in human-readable format
	return displayConfigHuman(ctx, configFlags.global)
}

// setConfigValue sets a configuration value
func setConfigValue(ctx *storage.StorageContext, key, value string, global bool) error {
	storage := ctx.GlobalState
	if !global && ctx.WorkspaceState != nil {
		storage = ctx.WorkspaceState
	}

	// Try to parse as JSON first
	var parsedValue interface{}
	if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
		// Not valid JSON, store as string
		parsedValue = value
	}

	if err := storage.Set(key, parsedValue); err != nil {
		return fmt.Errorf("failed to set config value: %w", err)
	}

	fmt.Printf("Set %s = %v\n", key, parsedValue)
	return nil
}

// getConfigValue gets a configuration value
func getConfigValue(ctx *storage.StorageContext, key string, global bool) error {
	storage := ctx.GlobalState
	if !global && ctx.WorkspaceState != nil {
		storage = ctx.WorkspaceState
	}

	value, ok := storage.Get(key)
	if !ok {
		// Try to find in global if not found in workspace
		if !global {
			value, ok = ctx.GlobalState.Get(key)
		}
		if !ok {
			return fmt.Errorf("key not found: %s", key)
		}
	}

	// Output the value
	switch v := value.(type) {
	case string:
		fmt.Println(v)
	case map[string]interface{}, []interface{}:
		// Pretty print complex values
		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("%v\n", v)
		} else {
			fmt.Println(string(jsonBytes))
		}
	default:
		fmt.Printf("%v\n", v)
	}

	return nil
}

// shouldUseInteractiveMode returns true if we should use the interactive TUI
func shouldUseInteractiveMode() bool {
	// Use TUI if no flags are set
	if configFlags.json || configFlags.edit || configFlags.global {
		return false
	}

	// Only use TUI if stdout is a terminal
	return tui.SupportsTUI()
}

// runInteractiveConfig runs the interactive TUI config mode
func runInteractiveConfig(ctx *storage.StorageContext) error {
	// Get data directory
	dataDir := getDataDir()

	// Create the config TUI model
	configModel := tui.NewConfigModel(ctx, dataDir)

	// Set up callbacks
	configModel.OnUpdateGlobal = func(key string, value interface{}) error {
		return ctx.GlobalState.Set(key, value)
	}

	configModel.OnUpdateWorkspace = func(key string, value interface{}) error {
		if ctx.WorkspaceState == nil {
			return fmt.Errorf("no workspace state available")
		}
		return ctx.WorkspaceState.Set(key, value)
	}

	configModel.OnToggleRule = func(isGlobal bool, rulePath string, enabled bool, ruleType string) error {
		return handleToggleRule(ctx, isGlobal, rulePath, enabled, ruleType)
	}

	configModel.OnToggleWorkflow = func(isGlobal bool, workflowPath string, enabled bool) error {
		return handleToggleWorkflow(ctx, isGlobal, workflowPath, enabled)
	}

	configModel.OnToggleHook = func(isGlobal bool, hookName string, enabled bool, workspaceName string) error {
		return handleToggleHook(ctx, isGlobal, hookName, enabled, workspaceName)
	}

	configModel.OnToggleSkill = func(isGlobal bool, skillPath string, enabled bool) error {
		return handleToggleSkill(ctx, isGlobal, skillPath, enabled)
	}

	configModel.OnQuit = func() {
		// Cleanup if needed
	}

	// Run the TUI program
	p := tea.NewProgram(configModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		// If TUI fails (e.g., no TTY available), fall back to human-readable output
		if isTTYError(err) {
			return displayConfigHuman(ctx, configFlags.global)
		}
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// isTTYError checks if the error is related to TTY unavailability
func isTTYError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "tty") ||
		contains(errStr, "terminal") ||
		contains(errStr, "device not configured") ||
		contains(errStr, "inappropriate ioctl") ||
		contains(errStr, "input/output error")
}

// getDataDir returns the data directory path
func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.cline/data"
	}
	return filepath.Join(homeDir, ".cline", "data")
}

// getWorkspaceHash returns the current workspace hash or empty string
func getWorkspaceHash() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	hash, _ := storage.GetWorkspaceHash(cwd)
	return hash
}

// handleToggleRule handles toggling a rule
func handleToggleRule(ctx *storage.StorageContext, isGlobal bool, rulePath string, enabled bool, ruleType string) error {
	var key string
	if isGlobal {
		key = "globalClineRulesToggles"
	} else {
		switch ruleType {
		case "cline":
			key = "localClineRulesToggles"
		case "cursor":
			key = "localCursorRulesToggles"
		case "windsurf":
			key = "localWindsurfRulesToggles"
		case "agents":
			key = "localAgentsRulesToggles"
		default:
			key = "localClineRulesToggles"
		}
	}

	storage := ctx.GlobalState
	if !isGlobal {
		storage = ctx.WorkspaceState
		if storage == nil {
			return fmt.Errorf("no workspace state available")
		}
	}

	// Get current toggles
	var toggles map[string]interface{}
	if val, ok := storage.Get(key); ok {
		if t, ok := val.(map[string]interface{}); ok {
			toggles = t
		} else {
			toggles = make(map[string]interface{})
		}
	} else {
		toggles = make(map[string]interface{})
	}

	// Update toggle
	toggles[rulePath] = enabled

	return storage.Set(key, toggles)
}

// handleToggleWorkflow handles toggling a workflow
func handleToggleWorkflow(ctx *storage.StorageContext, isGlobal bool, workflowPath string, enabled bool) error {
	var key string
	if isGlobal {
		key = "globalWorkflowToggles"
	} else {
		key = "localWorkflowToggles"
	}

	storage := ctx.GlobalState
	if !isGlobal {
		storage = ctx.WorkspaceState
		if storage == nil {
			return fmt.Errorf("no workspace state available")
		}
	}

	// Get current toggles
	var toggles map[string]interface{}
	if val, ok := storage.Get(key); ok {
		if t, ok := val.(map[string]interface{}); ok {
			toggles = t
		} else {
			toggles = make(map[string]interface{})
		}
	} else {
		toggles = make(map[string]interface{})
	}

	// Update toggle
	toggles[workflowPath] = enabled

	return storage.Set(key, toggles)
}

// handleToggleHook handles toggling a hook
func handleToggleHook(ctx *storage.StorageContext, isGlobal bool, hookName string, enabled bool, workspaceName string) error {
	if isGlobal {
		// Update global hooks
		val, ok := ctx.GlobalState.Get("globalHooks")
		if !ok {
			return nil
		}

		hooks, ok := val.([]interface{})
		if !ok {
			return nil
		}

		// Find and update the hook
		for _, h := range hooks {
			if hookMap, ok := h.(map[string]interface{}); ok {
				if name, ok := hookMap["name"].(string); ok && name == hookName {
					hookMap["enabled"] = enabled
					break
				}
			}
		}

		return ctx.GlobalState.Set("globalHooks", hooks)
	}

	// Update workspace hooks
	if ctx.WorkspaceState == nil {
		return fmt.Errorf("no workspace state available")
	}

	val, ok := ctx.WorkspaceState.Get("workspaceHooks")
	if !ok {
		return nil
	}

	wsHooks, ok := val.([]interface{})
	if !ok {
		return nil
	}

	// Find and update the hook in the appropriate workspace
	for _, wh := range wsHooks {
		if wsHookMap, ok := wh.(map[string]interface{}); ok {
			wsName := ""
			if name, ok := wsHookMap["workspaceName"].(string); ok {
				wsName = name
			}

			if wsName == workspaceName {
				if hooks, ok := wsHookMap["hooks"].([]interface{}); ok {
					for _, h := range hooks {
						if hookMap, ok := h.(map[string]interface{}); ok {
							if name, ok := hookMap["name"].(string); ok && name == hookName {
								hookMap["enabled"] = enabled
								break
							}
						}
					}
					break
				}
			}
		}
	}

	return ctx.WorkspaceState.Set("workspaceHooks", wsHooks)
}

// handleToggleSkill handles toggling a skill
func handleToggleSkill(ctx *storage.StorageContext, isGlobal bool, skillPath string, enabled bool) error {
	var key string
	if isGlobal {
		key = "globalSkills"
	} else {
		key = "localSkills"
	}

	storage := ctx.GlobalState
	if !isGlobal {
		storage = ctx.WorkspaceState
		if storage == nil {
			return fmt.Errorf("no workspace state available")
		}
	}

	// Get current skills
	val, ok := storage.Get(key)
	if !ok {
		return nil
	}

	skills, ok := val.([]interface{})
	if !ok {
		return nil
	}

	// Find and update the skill
	for _, s := range skills {
		if skillMap, ok := s.(map[string]interface{}); ok {
			if path, ok := skillMap["path"].(string); ok && path == skillPath {
				skillMap["enabled"] = enabled
				break
			}
		}
	}

	return storage.Set(key, skills)
}

// editConfig opens the configuration file in the default editor
func editConfig(ctx *storage.StorageContext) error {
	// Determine which file to edit
	var configFile string

	if configFlags.global || ctx.WorkspaceState == nil {
		configFile = ctx.GlobalState.FilePath()
	} else {
		configFile = ctx.WorkspaceState.FilePath()
	}

	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, ed := range []string{"vim", "vi", "nano", "emacs", "code"} {
			if _, err := exec.LookPath(ed); err == nil {
				editor = ed
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set EDITOR environment variable")
	}

	// Open file in editor
	cmd := exec.Command(editor, configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "Opening %s in %s...\n", configFile, editor)
	return cmd.Run()
}

// displayConfigJSON displays the configuration as JSON
func displayConfigJSON(ctx *storage.StorageContext, globalOnly bool) error {
	configData := make(map[string]interface{})

	// Add global state
	globalData := ctx.GlobalState.GetAll()
	if len(globalData) > 0 {
		configData["global"] = globalData
	}

	// Add workspace state if available and not global-only
	if !globalOnly && ctx.WorkspaceState != nil {
		workspaceData := ctx.WorkspaceState.GetAll()
		if len(workspaceData) > 0 {
			configData["workspace"] = workspaceData
		}
	}

	// Output as JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(configData)
}

// displayConfigHuman displays configuration in human-readable format
func displayConfigHuman(ctx *storage.StorageContext, globalOnly bool) error {
	// Get global state
	globalData := ctx.GlobalState.GetAll()
	
	// Get workspace state if available and not global-only
	var workspaceData map[string]interface{}
	if !globalOnly && ctx.WorkspaceState != nil {
		workspaceData = ctx.WorkspaceState.GetAll()
	}

	// Filter and format the data
	filteredGlobal := filterConfigData(globalData)
	filteredWorkspace := filterConfigData(workspaceData)

	// Display global configuration
	if len(filteredGlobal) > 0 {
		fmt.Println("=== Global Configuration ===")
		printConfigSectionFormatted(filteredGlobal, "")
		fmt.Println()
	}

	// Display workspace configuration
	if len(filteredWorkspace) > 0 {
		fmt.Println("=== Workspace Configuration ===")
		printConfigSectionFormatted(filteredWorkspace, "")
		fmt.Println()
	}

	// If no configuration found
	if len(filteredGlobal) == 0 && len(filteredWorkspace) == 0 {
		fmt.Println("No configuration found.")
		fmt.Println("\nGlobal config location: ~/.cline/data/globalState.json")
		fmt.Println("Workspace config location: ~/.cline/data/workspaces/<hash>/workspaceState.json")
	}

	return nil
}

// filterConfigData filters out sensitive and internal fields from config data
func filterConfigData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	filtered := make(map[string]interface{})
	for key, value := range data {
		// Skip internal/toggle keys
		if shouldExcludeConfigKey(key) {
			continue
		}

		// Mask sensitive values
		if isSensitiveConfigKey(key) {
			if strVal, ok := value.(string); ok && strVal != "" {
				filtered[key] = maskSensitiveValue(key, strVal)
			} else {
				filtered[key] = value
			}
		} else {
			filtered[key] = value
		}
	}

	return filtered
}

// shouldExcludeConfigKey returns true if the key should be excluded from display
func shouldExcludeConfigKey(key string) bool {
	excludedSuffixes := []string{
		"Toggles",
		"RulesToggles",
		"WorkflowToggles",
	}
	excludedPrefixes := []string{
		"apiConfig_",
	}
	excludedKeys := []string{
		"taskHistory",
	}

	lowerKey := strings.ToLower(key)

	// Check excluded keys
	for _, excluded := range excludedKeys {
		if strings.EqualFold(key, excluded) {
			return true
		}
	}

	// Check excluded suffixes
	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}

	// Check excluded prefixes
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(lowerKey, strings.ToLower(prefix)) {
			return true
		}
	}

	return false
}

// isSensitiveConfigKey returns true if the key contains sensitive data
func isSensitiveConfigKey(key string) bool {
	sensitivePatterns := []string{
		"apiKey",
		"api_key",
		"secret",
		"password",
		"token",
		"auth",
		"credential",
		"private",
	}

	lowerKey := strings.ToLower(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// printConfigSectionFormatted prints configuration with better formatting
func printConfigSectionFormatted(data map[string]interface{}, indent string) {
	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := data[key]
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indent, key)
			printConfigSectionFormatted(v, indent+"  ")
		case []interface{}:
			// Format arrays more nicely
			if len(v) == 0 {
				fmt.Printf("%s%s: []\n", indent, key)
			} else {
				fmt.Printf("%s%s:\n", indent, key)
				for i, item := range v {
					switch itemVal := item.(type) {
					case map[string]interface{}:
						fmt.Printf("%s  [%d]:\n", indent, i)
						printConfigSectionFormatted(itemVal, indent+"    ")
					default:
						fmt.Printf("%s  - %v\n", indent, item)
					}
				}
			}
		case string:
			if v == "" {
				fmt.Printf("%s%s: (empty)\n", indent, key)
			} else {
				fmt.Printf("%s%s: %s\n", indent, key, v)
			}
		case nil:
			fmt.Printf("%s%s: (null)\n", indent, key)
		case bool:
			fmt.Printf("%s%s: %t\n", indent, key, v)
		case float64:
			// Print integers without decimal
			if v == float64(int64(v)) {
				fmt.Printf("%s%s: %.0f\n", indent, key, v)
			} else {
				fmt.Printf("%s%s: %g\n", indent, key, v)
			}
		default:
			fmt.Printf("%s%s: %v\n", indent, key, v)
		}
	}
}

// maskSensitiveValue masks sensitive configuration values
func maskSensitiveValue(key, value string) string {
	sensitiveKeys := []string{
		"apiKey", "api_key", "secret", "password", "token", "auth",
		"key", "credential", "private",
	}

	lowerKey := ""
	for _, r := range key {
		if r >= 'A' && r <= 'Z' {
			lowerKey += string(r - 'A' + 'a')
		} else {
			lowerKey += string(r)
		}
	}

	for _, sensitive := range sensitiveKeys {
		if contains(lowerKey, sensitive) {
			if len(value) <= 8 {
				return "****"
			}
			return value[:4] + "****" + value[len(value)-4:]
		}
	}

	return value
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath(global bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		return filepath.Join(homeDir, ".cline", "data", "globalState.json"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	hash, err := storage.GetWorkspaceHash(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace hash: %w", err)
	}

	return filepath.Join(homeDir, ".cline", "data", "workspaces", hash, "workspaceState.json"), nil
}