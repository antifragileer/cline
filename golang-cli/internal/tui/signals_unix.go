//go:build darwin || linux

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandling sets up signal handling for Unix platforms
func SetupSignalHandling() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGWINCH)
	return sigChan
}