//go:build windows

package tui

import (
	"os"
	"os/signal"
)

// SetupSignalHandling sets up signal handling for Windows
func SetupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	return sigChan
}
