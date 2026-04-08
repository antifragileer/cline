// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GitStats holds git repository statistics
type GitStats struct {
	Branch    string
	Files     int
	Additions int
	Deletions int
	IsRepo    bool
	RepoName  string
}

// GitStatsDisplay displays git statistics in the UI
type GitStatsDisplay struct {
	// Styling
	branchStyle    lipgloss.Style
	repoStyle      lipgloss.Style
	filesStyle     lipgloss.Style
	additionsStyle lipgloss.Style
	deletionsStyle lipgloss.Style
	separatorStyle lipgloss.Style
}

// NewGitStatsDisplay creates a new git stats display
func NewGitStatsDisplay() *GitStatsDisplay {
	return &GitStatsDisplay{
		branchStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")),
		repoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),
		filesStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
		additionsStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")),
		deletionsStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")),
		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),
	}
}

// GetGitStats retrieves git statistics for the given directory
func GetGitStats(cwd string) *GitStats {
	stats := &GitStats{
		IsRepo: false,
	}

	// Get repo name from directory
	if cwd == "" {
		cwd = "."
	}
	stats.RepoName = filepath.Base(cwd)

	// Try to get git branch
	branch, err := getGitBranch(cwd)
	if err != nil {
		return stats
	}

	stats.IsRepo = true
	stats.Branch = branch

	// Get diff stats
	diffStats, err := getGitDiffStats(cwd)
	if err == nil && diffStats != nil {
		stats.Files = diffStats.Files
		stats.Additions = diffStats.Additions
		stats.Deletions = diffStats.Deletions
	}

	return stats
}

// Render renders the git stats display
func (d *GitStatsDisplay) Render(stats *GitStats) string {
	if !stats.IsRepo {
		return d.repoStyle.Render(stats.RepoName)
	}

	var parts []string

	// Repo name
	parts = append(parts, d.repoStyle.Render(stats.RepoName))

	// Branch
	if stats.Branch != "" {
		parts = append(parts, d.branchStyle.Render(fmt.Sprintf("(%s)", stats.Branch)))
	}

	// Diff stats
	if stats.Files > 0 {
		filesStr := d.filesStyle.Render(fmt.Sprintf("%d file%s", stats.Files, pluralize(stats.Files)))
		parts = append(parts, filesStr)

		if stats.Additions > 0 {
			addStr := d.additionsStyle.Render(fmt.Sprintf("+%d", stats.Additions))
			parts = append(parts, addStr)
		}

		if stats.Deletions > 0 {
			delStr := d.deletionsStyle.Render(fmt.Sprintf("-%d", stats.Deletions))
			parts = append(parts, delStr)
		}
	}

	return strings.Join(parts, " ")
}

// RenderCompact renders a compact version of git stats
func (d *GitStatsDisplay) RenderCompact(stats *GitStats) string {
	if !stats.IsRepo {
		return d.repoStyle.Render(stats.RepoName)
	}

	var parts []string

	// Just repo name and branch
	parts = append(parts, d.repoStyle.Render(stats.RepoName))

	if stats.Branch != "" {
		parts = append(parts, d.branchStyle.Render(fmt.Sprintf("(%s)", stats.Branch)))
	}

	return strings.Join(parts, " ")
}

// getGitBranch gets the current git branch
func getGitBranch(cwd string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	if cwd != "" {
		cmd.Dir = cwd
	}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// DiffStats holds git diff statistics
type DiffStats struct {
	Files     int
	Additions int
	Deletions int
}

// getGitDiffStats gets the git diff statistics
func getGitDiffStats(cwd string) (*DiffStats, error) {
	cmd := exec.Command("git", "diff", "--shortstat")
	if cwd != "" {
		cmd.Dir = cwd
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return &DiffStats{}, nil
	}

	// Parse output like "2 files changed, 10 insertions(+), 5 deletions(-)"
	stats := &DiffStats{}

	// Extract files changed
	filesMatch := strings.Split(line, " file")
	if len(filesMatch) > 0 {
		// Get the number before " file"
		parts := strings.Fields(filesMatch[0])
		if len(parts) > 0 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				stats.Files = n
			}
		}
	}

	// Extract insertions
	insertionsIdx := strings.Index(line, "insertion")
	if insertionsIdx > 0 {
		before := line[:insertionsIdx]
		parts := strings.Fields(before)
		if len(parts) > 0 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				stats.Additions = n
			}
		}
	}

	// Extract deletions
	deletionsIdx := strings.Index(line, "deletion")
	if deletionsIdx > 0 {
		before := line[:deletionsIdx]
		parts := strings.Fields(before)
		if len(parts) > 0 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				stats.Deletions = n
			}
		}
	}

	return stats, nil
}

// GitStatsMsg is sent when git stats are updated
type GitStatsMsg struct {
	Stats *GitStats
}
