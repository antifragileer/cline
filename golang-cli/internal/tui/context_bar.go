// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ContextBar displays token usage and context window information
type ContextBar struct {
	// Styling
	 filledStyle      lipgloss.Style
	 emptyStyle       lipgloss.Style
	 labelStyle       lipgloss.Style
	 valueStyle       lipgloss.Style
	 costStyle        lipgloss.Style
	 separatorStyle   lipgloss.Style
}

// NewContextBar creates a new context bar
func NewContextBar() *ContextBar {
	return &ContextBar{
		filledStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")),
		emptyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333")),
		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
		valueStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		costStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")),
		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),
	}
}

// ContextInfo holds context window information
type ContextInfo struct {
	UsedTokens   int
	TotalTokens  int
	ContextSize  int
	TotalCost    float64
	ModelID      string
}

// Render renders the context bar with model info, token usage, and cost
func (cb *ContextBar) Render(info *ContextInfo) string {
	var parts []string

	// Model ID
	if info.ModelID != "" {
		parts = append(parts, cb.valueStyle.Render(info.ModelID))
	}

	// Token usage bar
	if info.ContextSize > 0 {
		bar := cb.createTokenBar(info.UsedTokens, info.ContextSize)
		parts = append(parts, bar)
	}

	// Token count
	if info.UsedTokens > 0 {
		parts = append(parts, cb.labelStyle.Render(fmt.Sprintf("(%s)", formatNumber(info.UsedTokens))))
	}

	// Cost
	if info.TotalCost > 0 {
		parts = append(parts, cb.separatorStyle.Render("|"))
		parts = append(parts, cb.costStyle.Render(fmt.Sprintf("$%.3f", info.TotalCost)))
	}

	return strings.Join(parts, " ")
}

// RenderCompact renders a compact version with just the essentials
func (cb *ContextBar) RenderCompact(info *ContextInfo) string {
	var parts []string

	// Model ID (shortened)
	modelID := info.ModelID
	if len(modelID) > 20 {
		modelID = modelID[:17] + "..."
	}
	parts = append(parts, cb.valueStyle.Render(modelID))

	// Mini token bar
	if info.ContextSize > 0 {
		bar := cb.createMiniTokenBar(info.UsedTokens, info.ContextSize)
		parts = append(parts, bar)
	}

	// Cost
	if info.TotalCost > 0 {
		parts = append(parts, cb.costStyle.Render(fmt.Sprintf("$%.2f", info.TotalCost)))
	}

	return strings.Join(parts, " ")
}

// createTokenBar creates a visual bar showing token usage
func (cb *ContextBar) createTokenBar(used, total int) string {
	width := 8
	if used < 0 {
		used = 0
	}
	if total <= 0 {
		return cb.emptyStyle.Render(strings.Repeat("█", width))
	}

	ratio := float64(used) / float64(total)
	if ratio > 1 {
		ratio = 1
	}

	filledCount := 0
	if used > 0 {
		filledCount = max(1, int(ratio*float64(width)))
	}
	emptyCount := width - filledCount

	filled := cb.filledStyle.Render(strings.Repeat("█", filledCount))
	empty := cb.emptyStyle.Render(strings.Repeat("█", emptyCount))

	return filled + empty
}

// createMiniTokenBar creates a smaller token bar (4 chars)
func (cb *ContextBar) createMiniTokenBar(used, total int) string {
	width := 4
	if used < 0 {
		used = 0
	}
	if total <= 0 {
		return cb.emptyStyle.Render("░░░░")
	}

	ratio := float64(used) / float64(total)
	if ratio > 1 {
		ratio = 1
	}

	filledCount := int(ratio * float64(width))
	emptyCount := width - filledCount

	bar := ""
	for i := 0; i < filledCount; i++ {
		bar += "█"
	}
	for i := 0; i < emptyCount; i++ {
		bar += "░"
	}

	return cb.filledStyle.Render(bar[:filledCount]) + cb.emptyStyle.Render(bar[filledCount:])
}

// formatNumber formats a number with commas for thousands
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n%1000000)/1000, n%1000)
}

// DefaultContextSize returns the default context window size for a model
func DefaultContextSize(modelID string) int {
	// Common model context sizes
	contextSizes := map[string]int{
		"claude-sonnet-4":    200000,
		"claude-opus-4":      200000,
		"claude-haiku-3":     200000,
		"gpt-4o":             128000,
		"gpt-4o-mini":        128000,
		"gpt-4":              8192,
		"gpt-4-turbo":        128000,
		"gemini-1.5-pro":     1000000,
		"gemini-1.5-flash":   1000000,
	}

	if size, ok := contextSizes[modelID]; ok {
		return size
	}

	// Default to 128k for unknown models
	return 128000
}

// CalculateTokensEstimate estimates token count from text
// This is a rough estimate (approx 4 chars per token)
func CalculateTokensEstimate(text string) int {
	return len(text) / 4
}

// ContextBarMsg is sent to update the context bar
type ContextBarMsg struct {
	Info *ContextInfo
}