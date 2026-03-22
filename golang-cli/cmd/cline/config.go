package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// configFlags holds the parsed flag values for config command
var configFlags struct {
	json   bool
	edit   bool
	global bool
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

func init() {
	rootCmd.AddCommand(configCmd)

	// Add flags to config command
	configCmd.Flags().BoolVarP(&configFlags.json, "json", "j", false, "Output in JSON format")
	configCmd.Flags().BoolVarP(&configFlags.edit, "edit", "e", false, "Open configuration in editor")
	configCmd.Flags().BoolVarP(&configFlags.global, "global", "g", false, "Show only global configuration")
}

// runConfig executes the config command
func runConfig(cmd *cobra.Command, args []string) error {
	// Initialize storage context
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Handle edit mode
	if configFlags.edit {
		return editConfig(ctx)
	}

	// Display configuration
	return displayConfig(ctx, configFlags.json, configFlags.global)
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

// displayConfig displays the configuration in the requested format
func displayConfig(ctx *storage.StorageContext, asJSON bool, globalOnly bool) error {
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

	// Output based on format
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(configData)
	}

	// Human-readable output
	return displayConfigHuman(configData)
}

// displayConfigHuman displays configuration in human-readable format
func displayConfigHuman(configData map[string]interface{}) error {
	// Display global configuration
	if globalData, ok := configData["global"].(map[string]interface{}); ok && len(globalData) > 0 {
		fmt.Println("=== Global Configuration ===")
		printConfigSection(globalData, "")
		fmt.Println()
	}

	// Display workspace configuration
	if workspaceData, ok := configData["workspace"].(map[string]interface{}); ok && len(workspaceData) > 0 {
		fmt.Println("=== Workspace Configuration ===")
		printConfigSection(workspaceData, "")
		fmt.Println()
	}

	// If no configuration found
	if len(configData) == 0 {
		fmt.Println("No configuration found.")
		fmt.Println("\nGlobal config location: ~/.cline/data/globalState.json")
		fmt.Println("Workspace config location: ~/.cline/data/workspaces/<hash>/workspaceState.json")
	}

	return nil
}

// printConfigSection prints a configuration section recursively
func printConfigSection(data map[string]interface{}, indent string) {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indent, key)
			printConfigSection(v, indent+"  ")
		case []interface{}:
			fmt.Printf("%s%s: [", indent, key)
			for i, item := range v {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%v", item)
			}
			fmt.Println("]")
		default:
			// Mask sensitive values
			displayValue := maskSensitiveValue(key, fmt.Sprintf("%v", v))
			fmt.Printf("%s%s: %s\n", indent, key, displayValue)
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

// ConfigStorage interface for testing
type ConfigStorage interface {
	GetAll() map[string]interface{}
	FilePath() string
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