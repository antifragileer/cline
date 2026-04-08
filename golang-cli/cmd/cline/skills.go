package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// Skill represents a Cline skill configuration
type Skill struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source,omitempty"` // "global" or "local"
}

// skillsFlags holds flags for skills commands
var skillsFlags struct {
	json   bool
	global bool
	search string
}

// skillsCmd represents the skills command
var skillsCmd = &cobra.Command{
	Use:    "skills",
	Short:  "Manage Cline skills",
	Hidden: true, // Hidden - not in TypeScript CLI
	Long: `Manage Cline skills for extended functionality.

Skills are modular capabilities that can be added to Cline to extend
its functionality. Skills can be installed globally or per-workspace.`,
	Example: `  # List all skills
  cline skills list

  # List skills in JSON format
  cline skills list --json

  # List only global skills
  cline skills list --global`,
}

// skillsListCmd represents the skills list subcommand
var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	Long:  `List all installed skills with their status and description.`,
	Example: `  cline skills list
  cline skills list --json
  cline skills list --global`,
	RunE: runSkillsList,
}

// skillsEnableCmd represents the skills enable subcommand
var skillsEnableCmd = &cobra.Command{
	Use:     "enable <path>",
	Short:   "Enable a skill",
	Long:    `Enable a previously disabled skill.`,
	Example: `  cline skills enable my-skill`,
	Args:    cobra.ExactArgs(1),
	RunE:    runSkillsEnable,
}

// skillsDisableCmd represents the skills disable subcommand
var skillsDisableCmd = &cobra.Command{
	Use:     "disable <path>",
	Short:   "Disable a skill",
	Long:    `Disable a skill without removing it from configuration.`,
	Example: `  cline skills disable my-skill`,
	Args:    cobra.ExactArgs(1),
	RunE:    runSkillsDisable,
}

// skillsMarketplaceCmd represents the skills marketplace subcommand
var skillsMarketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"mp", "market"},
	Short:   "Browse available skills",
	Long:    `Browse and search the skills marketplace for available skills to install.`,
	Example: `  cline skills marketplace
  cline skills marketplace --search python`,
	RunE: runSkillsMarketplace,
}

func init() {
	rootCmd.AddCommand(skillsCmd)

	// Add subcommands
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsEnableCmd)
	skillsCmd.AddCommand(skillsDisableCmd)
	skillsCmd.AddCommand(skillsMarketplaceCmd)

	// Add flags
	skillsListCmd.Flags().BoolVarP(&skillsFlags.json, "json", "j", false, "Output in JSON format")
	skillsListCmd.Flags().BoolVarP(&skillsFlags.global, "global", "g", false, "Show only global skills")

	skillsMarketplaceCmd.Flags().StringVar(&skillsFlags.search, "search", "", "Search term for skills")
}

// runSkillsList executes the skills list command
func runSkillsList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load global skills
	var globalSkills []Skill
	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		if skills, err := parseSkills(val); err == nil {
			for i := range skills {
				skills[i].Source = "global"
			}
			globalSkills = skills
		}
	}

	// Load local skills
	var localSkills []Skill
	if !skillsFlags.global && ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localSkills"); ok {
			if skills, err := parseSkills(val); err == nil {
				for i := range skills {
					skills[i].Source = "local"
				}
				localSkills = skills
			}
		}
	}

	// Output as JSON if requested
	if skillsFlags.json {
		output := map[string]interface{}{
			"global": globalSkills,
		}
		if !skillsFlags.global {
			output["local"] = localSkills
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	if len(globalSkills) == 0 && len(localSkills) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No skills installed.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'cline skills marketplace' to browse available skills.")
		return nil
	}

	// Display global skills
	if len(globalSkills) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Global Skills ===")
		for _, skill := range globalSkills {
			printSkill(cmd.OutOrStdout(), skill)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Display local skills
	if !skillsFlags.global && len(localSkills) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Workspace Skills ===")
		for _, skill := range localSkills {
			printSkill(cmd.OutOrStdout(), skill)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// printSkill prints a single skill in human-readable format
func printSkill(w io.Writer, skill Skill) {
	status := "✓ enabled"
	if !skill.Enabled {
		status = "✗ disabled"
	}

	name := skill.Name
	if name == "" {
		name = filepath.Base(skill.Path)
	}

	fmt.Fprintf(w, "  %s %s\n", status, name)
	if skill.Description != "" {
		fmt.Fprintf(w, "    Description: %s\n", skill.Description)
	}
	fmt.Fprintf(w, "    Path: %s\n", skill.Path)
}

// runSkillsEnable executes the skills enable command
func runSkillsEnable(cmd *cobra.Command, args []string) error {
	skillPath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try global skills first
	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		if skills, err := parseSkills(val); err == nil {
			for i, skill := range skills {
				if skill.Path == skillPath {
					skills[i].Enabled = true
					if err := ctx.GlobalState.Set("globalSkills", skills); err != nil {
						return fmt.Errorf("failed to enable skill: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Skill '%s' enabled\n", skillPath)
					return nil
				}
			}
		}
	}

	// Try local skills
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localSkills"); ok {
			if skills, err := parseSkills(val); err == nil {
				for i, skill := range skills {
					if skill.Path == skillPath {
						skills[i].Enabled = true
						if err := ctx.WorkspaceState.Set("localSkills", skills); err != nil {
							return fmt.Errorf("failed to enable skill: %w", err)
						}
						fmt.Fprintf(cmd.OutOrStdout(), "✓ Skill '%s' enabled\n", skillPath)
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("skill '%s' not found", skillPath)
}

// runSkillsDisable executes the skills disable command
func runSkillsDisable(cmd *cobra.Command, args []string) error {
	skillPath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try global skills first
	if val, ok := ctx.GlobalState.Get("globalSkills"); ok {
		if skills, err := parseSkills(val); err == nil {
			for i, skill := range skills {
				if skill.Path == skillPath {
					skills[i].Enabled = false
					if err := ctx.GlobalState.Set("globalSkills", skills); err != nil {
						return fmt.Errorf("failed to disable skill: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Skill '%s' disabled\n", skillPath)
					return nil
				}
			}
		}
	}

	// Try local skills
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localSkills"); ok {
			if skills, err := parseSkills(val); err == nil {
				for i, skill := range skills {
					if skill.Path == skillPath {
						skills[i].Enabled = false
						if err := ctx.WorkspaceState.Set("localSkills", skills); err != nil {
							return fmt.Errorf("failed to disable skill: %w", err)
						}
						fmt.Fprintf(cmd.OutOrStdout(), "✓ Skill '%s' disabled\n", skillPath)
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("skill '%s' not found", skillPath)
}

// runSkillsMarketplace executes the skills marketplace command
func runSkillsMarketplace(cmd *cobra.Command, args []string) error {
	// For now, show a curated list of popular skills
	// In the future, this could fetch from an actual marketplace API

	skills := []struct {
		Name        string
		Description string
		Author      string
		Repository  string
		InstallCmd  string
	}{
		{
			Name:        "python-dev",
			Description: "Python development tools and best practices",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-python",
			InstallCmd:  "npx skills add cline/skills-python",
		},
		{
			Name:        "typescript-dev",
			Description: "TypeScript/JavaScript development tools",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-typescript",
			InstallCmd:  "npx skills add cline/skills-typescript",
		},
		{
			Name:        "go-dev",
			Description: "Go development tools and best practices",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-go",
			InstallCmd:  "npx skills add cline/skills-go",
		},
		{
			Name:        "rust-dev",
			Description: "Rust development tools and patterns",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-rust",
			InstallCmd:  "npx skills add cline/skills-rust",
		},
		{
			Name:        "docker-dev",
			Description: "Docker containerization best practices",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-docker",
			InstallCmd:  "npx skills add cline/skills-docker",
		},
		{
			Name:        "kubernetes-dev",
			Description: "Kubernetes deployment and management",
			Author:      "cline",
			Repository:  "https://github.com/cline/skills-kubernetes",
			InstallCmd:  "npx skills add cline/skills-kubernetes",
		},
	}

	// Filter by search term if provided
	searchTerm, _ := cmd.Flags().GetString("search")
	if searchTerm != "" {
		searchTerm = strings.ToLower(searchTerm)
		var filtered []struct {
			Name        string
			Description string
			Author      string
			Repository  string
			InstallCmd  string
		}
		for _, skill := range skills {
			if strings.Contains(strings.ToLower(skill.Name), searchTerm) ||
				strings.Contains(strings.ToLower(skill.Description), searchTerm) {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}

	if len(skills) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No skills found matching your search.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Available Skills:")
	fmt.Fprintln(cmd.OutOrStdout())

	for _, skill := range skills {
		fmt.Fprintf(cmd.OutOrStdout(), "  📦 %s\n", skill.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", skill.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "     Author: %s\n", skill.Author)
		fmt.Fprintf(cmd.OutOrStdout(), "     Install: %s\n", skill.InstallCmd)
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "To install a skill, run the install command shown above.")
	fmt.Fprintln(cmd.OutOrStdout(), "Visit https://skills.sh/ for more skills.")

	return nil
}

// parseSkills parses skill data from storage
func parseSkills(data interface{}) ([]Skill, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	if err := json.Unmarshal(jsonData, &skills); err != nil {
		return nil, err
	}

	return skills, nil
}

// GetSkillsDir returns the skills directory path
func GetSkillsDir(global bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		return filepath.Join(homeDir, ".cline", "skills"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, ".cline", "skills"), nil
}

// InstallSkill installs a skill from a repository
func InstallSkill(repo string, global bool) error {
	// This is a placeholder - actual implementation would:
	// 1. Clone the repository
	// 2. Parse SKILL.md
	// 3. Copy to appropriate directory
	// 4. Update configuration
	return fmt.Errorf("skill installation not yet implemented. Use: npx skills add %s", repo)
}

// UninstallSkill removes an installed skill
func UninstallSkill(name string, global bool) error {
	// This is a placeholder - actual implementation would:
	// 1. Remove from configuration
	// 2. Delete files if needed
	return fmt.Errorf("skill uninstallation not yet implemented")
}

// ValidateSkill validates a skill configuration
func ValidateSkill(skill *Skill) error {
	if skill.Path == "" {
		return fmt.Errorf("skill path cannot be empty")
	}

	// Check if path contains invalid characters
	if strings.ContainsAny(skill.Path, "<>:\"|?*") {
		return fmt.Errorf("skill path contains invalid characters")
	}

	return nil
}
