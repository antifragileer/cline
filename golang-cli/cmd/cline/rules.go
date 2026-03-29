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

// Rule represents a Cline rule configuration
type Rule struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	RuleType    string `json:"ruleType,omitempty"` // "cline", "cursor", "windsurf", "agents"
	Source      string `json:"source,omitempty"`   // "global" or "local"
}

// rulesFlags holds flags for rules commands
var rulesFlags struct {
	json     bool
	global   bool
	ruleType string
}

// rulesCmd represents the rules command
var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage Cline rules",
	Long: `Manage Cline rules for custom behavior and constraints.

Rules allow you to define custom constraints and behaviors that Cline
will follow when executing tasks. Rules can be applied globally or
per-workspace.`,
	Example: `  # List all rules
  cline rules list

  # List rules in JSON format
  cline rules list --json

  # List only global rules
  cline rules list --global

  # List only cursor rules
  cline rules list --type cursor`,
}

// rulesListCmd represents the rules list subcommand
var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured rules",
	Long:  `List all configured rules with their status and description.`,
	Example: `  cline rules list
  cline rules list --json
  cline rules list --global
  cline rules list --type cursor`,
	RunE: runRulesList,
}

// rulesEnableCmd represents the rules enable subcommand
var rulesEnableCmd = &cobra.Command{
	Use:   "enable <path>",
	Short: "Enable a rule",
	Long:  `Enable a previously disabled rule.`,
	Example: `  cline rules enable my-rule`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRulesEnable,
}

// rulesDisableCmd represents the rules disable subcommand
var rulesDisableCmd = &cobra.Command{
	Use:   "disable <path>",
	Short: "Disable a rule",
	Long:  `Disable a rule without removing it from configuration.`,
	Example: `  cline rules disable my-rule`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRulesDisable,
}

func init() {
	rootCmd.AddCommand(rulesCmd)

	// Add subcommands
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesEnableCmd)
	rulesCmd.AddCommand(rulesDisableCmd)

	// Add flags
	rulesListCmd.Flags().BoolVarP(&rulesFlags.json, "json", "j", false, "Output in JSON format")
	rulesListCmd.Flags().BoolVarP(&rulesFlags.global, "global", "g", false, "Show only global rules")
	rulesListCmd.Flags().StringVarP(&rulesFlags.ruleType, "type", "t", "", "Filter by rule type (cline, cursor, windsurf, agents)")
}

// runRulesList executes the rules list command
func runRulesList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Collect all rules
	var allRules []Rule

	// Load global cline rules
	if !rulesFlags.global || rulesFlags.ruleType == "" || rulesFlags.ruleType == "cline" {
		if val, ok := ctx.GlobalState.Get("globalClineRulesToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					allRules = append(allRules, Rule{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: "cline",
						Source:   "global",
					})
				}
			}
		}
	}

	// Load local cline rules
	if (!rulesFlags.global || rulesFlags.ruleType == "" || rulesFlags.ruleType == "cline") && ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localClineRulesToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					allRules = append(allRules, Rule{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: "cline",
						Source:   "local",
					})
				}
			}
		}
	}

	// Load cursor rules
	if rulesFlags.ruleType == "" || rulesFlags.ruleType == "cursor" {
		// Global cursor rules
		if val, ok := ctx.GlobalState.Get("globalCursorRulesToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					allRules = append(allRules, Rule{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: "cursor",
						Source:   "global",
					})
				}
			}
		}

		// Local cursor rules
		if !rulesFlags.global && ctx.WorkspaceState != nil {
			if val, ok := ctx.WorkspaceState.Get("localCursorRulesToggles"); ok {
				if toggles, err := parseToggles(val); err == nil {
					for path, enabled := range toggles {
						allRules = append(allRules, Rule{
							Path:     path,
							Name:     filepath.Base(path),
							Enabled:  enabled,
							RuleType: "cursor",
							Source:   "local",
						})
					}
				}
			}
		}
	}

	// Load windsurf rules
	if rulesFlags.ruleType == "" || rulesFlags.ruleType == "windsurf" {
		// Global windsurf rules
		if val, ok := ctx.GlobalState.Get("globalWindsurfRulesToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					allRules = append(allRules, Rule{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: "windsurf",
						Source:   "global",
					})
				}
			}
		}

		// Local windsurf rules
		if !rulesFlags.global && ctx.WorkspaceState != nil {
			if val, ok := ctx.WorkspaceState.Get("localWindsurfRulesToggles"); ok {
				if toggles, err := parseToggles(val); err == nil {
					for path, enabled := range toggles {
						allRules = append(allRules, Rule{
							Path:     path,
							Name:     filepath.Base(path),
							Enabled:  enabled,
							RuleType: "windsurf",
							Source:   "local",
						})
					}
				}
			}
		}
	}

	// Load agents rules
	if rulesFlags.ruleType == "" || rulesFlags.ruleType == "agents" {
		// Global agents rules
		if val, ok := ctx.GlobalState.Get("globalAgentsRulesToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					allRules = append(allRules, Rule{
						Path:     path,
						Name:     filepath.Base(path),
						Enabled:  enabled,
						RuleType: "agents",
						Source:   "global",
					})
				}
			}
		}

		// Local agents rules
		if !rulesFlags.global && ctx.WorkspaceState != nil {
			if val, ok := ctx.WorkspaceState.Get("localAgentsRulesToggles"); ok {
				if toggles, err := parseToggles(val); err == nil {
					for path, enabled := range toggles {
						allRules = append(allRules, Rule{
							Path:     path,
							Name:     filepath.Base(path),
							Enabled:  enabled,
							RuleType: "agents",
							Source:   "local",
						})
					}
				}
			}
		}
	}

	// Output as JSON if requested
	if rulesFlags.json {
		output := map[string]interface{}{
			"rules": allRules,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	if len(allRules) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No rules configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nRules allow you to define custom constraints for Cline.")
		return nil
	}

	// Group by source and type
	globalRules := filterRules(allRules, func(r Rule) bool { return r.Source == "global" })
	localRules := filterRules(allRules, func(r Rule) bool { return r.Source == "local" })

	// Display global rules
	if len(globalRules) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Global Rules ===")
		for _, rule := range globalRules {
			printRule(cmd.OutOrStdout(), rule)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Display local rules
	if len(localRules) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Workspace Rules ===")
		for _, rule := range localRules {
			printRule(cmd.OutOrStdout(), rule)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// filterRules filters rules based on a predicate
func filterRules(rules []Rule, predicate func(Rule) bool) []Rule {
	var filtered []Rule
	for _, rule := range rules {
		if predicate(rule) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

// printRule prints a single rule in human-readable format
func printRule(w io.Writer, rule Rule) {
	status := "✓ enabled"
	if !rule.Enabled {
		status = "✗ disabled"
	}

	name := rule.Name
	if name == "" {
		name = filepath.Base(rule.Path)
	}

	fmt.Fprintf(w, "  %s %s", status, name)
	if rule.RuleType != "" && rule.RuleType != "cline" {
		fmt.Fprintf(w, " [%s]", rule.RuleType)
	}
	fmt.Fprintln(w)

	if rule.Description != "" {
		fmt.Fprintf(w, "    Description: %s\n", rule.Description)
	}
	fmt.Fprintf(w, "    Path: %s\n", rule.Path)
}

// runRulesEnable executes the rules enable command
func runRulesEnable(cmd *cobra.Command, args []string) error {
	rulePath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try to find and enable the rule in various locations
	ruleTypes := []struct {
		key    string
		global bool
	}{
		{"globalClineRulesToggles", true},
		{"localClineRulesToggles", false},
		{"globalCursorRulesToggles", true},
		{"localCursorRulesToggles", false},
		{"globalWindsurfRulesToggles", true},
		{"localWindsurfRulesToggles", false},
		{"globalAgentsRulesToggles", true},
		{"localAgentsRulesToggles", false},
	}

	for _, rt := range ruleTypes {
		var storage *storage.ClineFileStorage
		if rt.global {
			storage = ctx.GlobalState
		} else {
			storage = ctx.WorkspaceState
			if storage == nil {
				continue
			}
		}

		if val, ok := storage.Get(rt.key); ok {
			if toggles, err := parseToggles(val); err == nil {
				if _, exists := toggles[rulePath]; exists {
					toggles[rulePath] = true
					if err := storage.Set(rt.key, toggles); err != nil {
						return fmt.Errorf("failed to enable rule: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Rule '%s' enabled\n", rulePath)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("rule '%s' not found", rulePath)
}

// runRulesDisable executes the rules disable command
func runRulesDisable(cmd *cobra.Command, args []string) error {
	rulePath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try to find and disable the rule in various locations
	ruleTypes := []struct {
		key    string
		global bool
	}{
		{"globalClineRulesToggles", true},
		{"localClineRulesToggles", false},
		{"globalCursorRulesToggles", true},
		{"localCursorRulesToggles", false},
		{"globalWindsurfRulesToggles", true},
		{"localWindsurfRulesToggles", false},
		{"globalAgentsRulesToggles", true},
		{"localAgentsRulesToggles", false},
	}

	for _, rt := range ruleTypes {
		var storage *storage.ClineFileStorage
		if rt.global {
			storage = ctx.GlobalState
		} else {
			storage = ctx.WorkspaceState
			if storage == nil {
				continue
			}
		}

		if val, ok := storage.Get(rt.key); ok {
			if toggles, err := parseToggles(val); err == nil {
				if _, exists := toggles[rulePath]; exists {
					toggles[rulePath] = false
					if err := storage.Set(rt.key, toggles); err != nil {
						return fmt.Errorf("failed to disable rule: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Rule '%s' disabled\n", rulePath)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("rule '%s' not found", rulePath)
}

// GetRulesDir returns the rules directory path
func GetRulesDir(global bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		return filepath.Join(homeDir, ".cline", "rules"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, ".cline", "rules"), nil
}

// ValidateRulePath validates a rule path
func ValidateRulePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("rule path cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(path, "<>:\"|?*") {
		return fmt.Errorf("rule path contains invalid characters")
	}

	return nil
}

// GetValidRuleTypes returns a list of valid rule types
func GetValidRuleTypes() []string {
	return []string{"cline", "cursor", "windsurf", "agents"}
}