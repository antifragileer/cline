package main

import (
	"context"
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
	// Create exit handler for proper exit code management
	handler := exit.NewHandler()

	// Run the CLI with exit code handling
	err := handler.Run(context.Background(), func(ctx context.Context) error {
		return Execute()
	})

	// Map any error to appropriate exit code
	if err != nil {
		code := extractExitCode(err)
		return code
	}

	return handler.GetExitCode()
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

	// Check for Cobra unknown command error
	errStr := err.Error()
	if stringsContains(errStr, "unknown command") || stringsContains(errStr, "unknown flag") {
		// Print error to stderr for better UX
		fmt.Fprintf(os.Stderr, "Error: %s\n", errStr)
		return exit.CommandNotFound
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
