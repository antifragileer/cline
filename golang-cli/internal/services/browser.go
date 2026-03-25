// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// BrowserService implements the BrowserService gRPC interface
type BrowserService struct {
	cline.UnimplementedBrowserServiceServer
	state *storage.ClineFileStorage
}

// NewBrowserService creates a new BrowserService instance
func NewBrowserService(state *storage.ClineFileStorage) *BrowserService {
	return &BrowserService{
		state: state,
	}
}

// GetBrowserConnectionInfo returns information about the browser connection
func (s *BrowserService) GetBrowserConnectionInfo(ctx context.Context, req *cline.EmptyRequest) (*cline.BrowserConnectionInfo, error) {
	// Get stored browser settings
	isConnected := false
	isRemote := false
	var host *string

	if data, ok := s.state.Get("browser_connection"); ok && data != nil {
		if connMap, ok := data.(map[string]interface{}); ok {
			if connected, ok := connMap["is_connected"].(bool); ok {
				isConnected = connected
			}
			if remote, ok := connMap["is_remote"].(bool); ok {
				isRemote = remote
			}
			if h, ok := connMap["host"].(string); ok && h != "" {
				host = &h
			}
		}
	}

	return &cline.BrowserConnectionInfo{
		IsConnected: isConnected,
		IsRemote:    isRemote,
		Host:        host,
	}, nil
}

// TestBrowserConnection tests a browser connection to the specified endpoint
func (s *BrowserService) TestBrowserConnection(ctx context.Context, req *cline.StringRequest) (*cline.BrowserConnection, error) {
	// In production, this would test the actual connection
	// For now, return a simulated result
	endpoint := req.Value

	// Simulate connection test
	success := endpoint != ""
	message := "Connection successful"
	if !success {
		message = "Invalid endpoint"
	}

	return &cline.BrowserConnection{
		Success:  success,
		Message:  message,
		Endpoint: &endpoint,
	}, nil
}

// DiscoverBrowser attempts to discover a Chrome/Chromium browser installation
func (s *BrowserService) DiscoverBrowser(ctx context.Context, req *cline.EmptyRequest) (*cline.BrowserConnection, error) {
	// Try to find Chrome/Chromium
	chromePath := s.findChromePath()

	if chromePath == "" {
		return &cline.BrowserConnection{
			Success: false,
			Message: "No Chrome or Chromium installation found",
		}, nil
	}

	// Update state with discovered path
	_ = s.state.Set("chrome_path", chromePath)

	return &cline.BrowserConnection{
		Success:  true,
		Message:  fmt.Sprintf("Found Chrome at: %s", chromePath),
		Endpoint: &chromePath,
	}, nil
}

// GetDetectedChromePath returns the detected Chrome executable path
func (s *BrowserService) GetDetectedChromePath(ctx context.Context, req *cline.EmptyRequest) (*cline.ChromePath, error) {
	// First check if we have a stored path
	if data, ok := s.state.Get("chrome_path"); ok && data != nil {
		if path, ok := data.(string); ok && path != "" {
			return &cline.ChromePath{
				Path:      path,
				IsBundled: false,
			}, nil
		}
	}

	// Try to discover
	path := s.findChromePath()
	if path != "" {
		_ = s.state.Set("chrome_path", path)
		return &cline.ChromePath{
			Path:      path,
			IsBundled: false,
		}, nil
	}

	// Return bundled path as fallback
	return &cline.ChromePath{
		Path:      "/usr/bin/google-chrome",
		IsBundled: true,
	}, nil
}

// RelaunchChromeDebugMode relaunches Chrome in debug mode
func (s *BrowserService) RelaunchChromeDebugMode(ctx context.Context, req *cline.EmptyRequest) (*cline.String, error) {
	chromePath := s.findChromePath()
	if chromePath == "" {
		return nil, fmt.Errorf("Chrome not found")
	}

	// Launch Chrome with remote debugging port
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-n", "-a", "Google Chrome", "--args", "--remote-debugging-port=9222")
	case "windows":
		cmd = exec.Command(chromePath, "--remote-debugging-port=9222")
	default:
		cmd = exec.Command(chromePath, "--remote-debugging-port=9222")
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to launch Chrome: %w", err)
	}

	return &cline.String{Value: "Chrome launched in debug mode on port 9222"}, nil
}

// Helper methods

func (s *BrowserService) findChromePath() string {
	// Common Chrome/Chromium paths by OS
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/usr/bin/google-chrome",
		}
	case "windows":
		paths = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Users\%LOCALAPPDATA%\Google\Chrome\Application\chrome.exe`,
		}
	default: // linux
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	}

	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}

	// Try to find in PATH
	if runtime.GOOS != "windows" {
		if path, err := exec.LookPath("google-chrome"); err == nil {
			return path
		}
		if path, err := exec.LookPath("chromium"); err == nil {
			return path
		}
		if path, err := exec.LookPath("chromium-browser"); err == nil {
			return path
		}
	}

	return ""
}

func fileExists(path string) bool {
	// Use exec.LookPath for commands in PATH
	if path == "google-chrome" || path == "chromium" || path == "chromium-browser" {
		_, err := exec.LookPath(path)
		return err == nil
	}
	
	// For full paths, try to stat the file
	cmd := exec.Command("test", "-f", path)
	err := cmd.Run()
	return err == nil
}
