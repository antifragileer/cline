package main

import (
	"context"
	"errors"

	"github.com/cline/cline/golang-cli/internal/exit"
)

func main() {
	// Create exit handler for proper exit code management
	handler := exit.NewHandler()

	// Run the CLI with exit code handling
	err := handler.Run(context.Background(), func(ctx context.Context) error {
		return Execute()
	})

	// Map any error to appropriate exit code
	if err != nil {
		code := extractExitCode(err)
		handler.SetExitCode(code)
	}

	// Exit with the appropriate code
	handler.Exit()
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

	// Fall back to mapping based on error content
	return exit.MapErrorToCode(err)
}
