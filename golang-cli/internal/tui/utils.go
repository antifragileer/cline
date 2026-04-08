// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
)

// formatString is a helper function for formatting strings.
func formatString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
