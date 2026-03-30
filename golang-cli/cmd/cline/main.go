package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cline/cline/golang-cli/internal/exit"
)

func main() {
	// Execute the CLI and get the exit code
	code := runCLI()
	os.Exit(int(code))
}

// runCLI executes the CLI and returns the appropriate exit code
func runCLI() exit.Code {
	// Execute the root command directly
	err := Execute()

	// Map any error to appropriate exit code
	if err != nil {
		code := extractExitCode(err)
		return code
	}

	return exit.Success
}

// extractExitCode extracts the exit code from an error
func extractExitCode(err error) exit.Code {
	if err == nil {
		return exit.Success
	}

	// Check if it's an exitError
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}

	// Check for Cobra unknown command or flag error
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "unknown command") {
		// Print error to stderr for better UX
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return exit.CommandNotFound
	}
	if strings.Contains(errStr, "unknown flag") || strings.Contains(errStr, "flag provided but not defined") {
		// Print error to stderr for better UX
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		return exit.GeneralError
	}

	// Fall back to mapping based on error content
	return exit.MapErrorToCode(err)
}

// stringsContains checks if a string contains a substring (case-insensitive)
func stringsContains(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}
