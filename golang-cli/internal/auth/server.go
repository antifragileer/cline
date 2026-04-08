// Package auth provides OAuth 2.0 authentication flows with PKCE support
// for the Cline CLI.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// successHTML is the HTML page shown after successful authorization
	successHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
        }
        .container {
            text-align: center;
            padding: 40px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            max-width: 400px;
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #2c3e50;
            margin-bottom: 10px;
            font-size: 24px;
        }
        p {
            color: #7f8c8d;
            line-height: 1.6;
            margin-bottom: 20px;
        }
        .close-info {
            font-size: 14px;
            color: #95a5a6;
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #ecf0f1;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✅</div>
        <h1>Authorization Successful</h1>
        <p>You have successfully authorized Cline CLI. You can close this window and return to your terminal.</p>
        <div class="close-info">This window will close automatically in 5 seconds...</div>
    </div>
    <script>
        setTimeout(function() {
            window.close();
        }, 5000);
    </script>
</body>
</html>`

	// errorHTML is the HTML page shown after authorization error
	errorHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Failed</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
        }
        .container {
            text-align: center;
            padding: 40px;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            max-width: 400px;
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #e74c3c;
            margin-bottom: 10px;
            font-size: 24px;
        }
        p {
            color: #7f8c8d;
            line-height: 1.6;
            margin-bottom: 20px;
        }
        .error-details {
            background: #fdf2f2;
            border: 1px solid #fee2e2;
            border-radius: 4px;
            padding: 15px;
            margin-top: 20px;
            color: #991b1b;
            font-family: monospace;
            font-size: 12px;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">❌</div>
        <h1>Authorization Failed</h1>
        <p>We couldn't complete the authorization. Please close this window and try again in your terminal.</p>
        <div class="error-details">%s</div>
    </div>
</body>
</html>`

	// shutdownTimeout is the timeout for gracefully shutting down the server
	shutdownTimeout = 5 * time.Second
	// serverStartTimeout is the timeout for starting the server
	serverStartTimeout = 10 * time.Second
)

// callbackHandler is a function that handles authorization codes
type callbackHandler func(code, state string)

// errorHandler is a function that handles authorization errors
type errorHandler func(error, description string)

// callbackServerConfig holds the configuration for the callback server
type callbackServerConfig struct {
	// Port is the port to listen on (0 = any available port)
	Port int
	// Path is the callback path
	Path string
	// OnCode is called when an authorization code is received
	OnCode callbackHandler
	// OnError is called when an authorization error occurs
	OnError errorHandler
	// Timeout is the maximum time to wait for a callback
	Timeout time.Duration
}

// callbackServer is a local HTTP server that handles OAuth callbacks
type callbackServer struct {
	config     callbackServerConfig
	server     *http.Server
	listener   net.Listener
	url        string
	mu         sync.RWMutex
	started    bool
	stopped    bool
	resultChan chan struct{}
}

// newCallbackServer creates a new callback server with the given configuration
func newCallbackServer(config callbackServerConfig) (*callbackServer, error) {
	if config.Path == "" {
		return nil, errors.New("callback path is required")
	}
	if config.OnCode == nil {
		return nil, errors.New("code handler is required")
	}
	if config.OnError == nil {
		return nil, errors.New("error handler is required")
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	return &callbackServer{
		config:     config,
		resultChan: make(chan struct{}, 1),
	}, nil
}

// Start starts the callback server
func (s *callbackServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New("server already started")
	}

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Build the callback URL
	port := listener.Addr().(*net.TCPAddr).Port
	s.url = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleCallback)
	mux.HandleFunc("/", s.handleDefault)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.started = true

	// Start server in a goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Log error but don't fail - the main flow will timeout
			select {
			case s.resultChan <- struct{}{}:
			default:
			}
		}
	}()

	return nil
}

// Stop gracefully stops the callback server
func (s *callbackServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.stopped {
		return nil
	}

	s.stopped = true

	// Close the result channel to prevent goroutine leaks
	close(s.resultChan)

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		// Force close if graceful shutdown fails
		s.server.Close()
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return nil
}

// GetURL returns the callback URL
func (s *callbackServer) GetURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.url
}

// GetPort returns the port the server is listening on
func (s *callbackServer) GetPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return 0
	}
	return s.listener.Addr().(*net.TCPAddr).Port
}

// IsStarted returns true if the server has been started
func (s *callbackServer) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// handleCallback handles OAuth callback requests
func (s *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Check for error
	if errMsg := query.Get("error"); errMsg != "" {
		description := query.Get("error_description")
		s.config.OnError(errMsg, description)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, errorHTML, htmlEscape(fmt.Sprintf("%s: %s", errMsg, description)))
		return
	}

	// Check for authorization code
	code := query.Get("code")
	if code == "" {
		s.config.OnError("missing_code", "No authorization code received")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, errorHTML, htmlEscape("No authorization code received"))
		return
	}

	// Get state parameter
	state := query.Get("state")

	// Call the code handler
	s.config.OnCode(code, state)

	// Return success page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successHTML))

	// Signal that we have a result
	select {
	case s.resultChan <- struct{}{}:
	default:
	}
}

// handleDefault handles requests to paths other than the callback
func (s *callbackServer) handleDefault(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found. This server only handles OAuth callbacks."))
}

// Wait waits for the callback to be received or the server to stop
func (s *callbackServer) Wait() error {
	select {
	case <-s.resultChan:
		return nil
	case <-time.After(s.config.Timeout):
		return ErrAuthorizationTimeout
	}
}

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "\x26amp;")
	s = strings.ReplaceAll(s, "<", "\x26lt;")
	s = strings.ReplaceAll(s, ">", "\x26gt;")
	s = strings.ReplaceAll(s, "\"", "\x26quot;")
	s = strings.ReplaceAll(s, "'", "\x26#39;")
	return s
}

func validateCallbackURL(callbackURL string) error {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("invalid callback URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("callback URL must use http or https scheme")
	}

	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return errors.New("callback URL must use localhost or loopback address")
	}

	return nil
}

// findAvailablePort finds an available TCP port on localhost
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// isPortAvailable checks if a port is available
func isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// normalizePath normalizes the callback path to start with /
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

// buildRedirectURL builds a redirect URL with the given port and path
func buildRedirectURL(port int, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", port, normalizePath(path))
}

// parseCallbackURL parses a callback URL and extracts port and path
func parseCallbackURL(callbackURL string) (port int, path string, err error) {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return 0, "", fmt.Errorf("invalid callback URL: %w", err)
	}

	// Extract port
	if parsed.Port() != "" {
		var port64 int64
		_, err = fmt.Sscanf(parsed.Port(), "%d", &port64)
		if err != nil {
			return 0, "", fmt.Errorf("invalid port: %w", err)
		}
		port = int(port64)
	} else {
		// Use default ports based on scheme
		if parsed.Scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	// Extract path
	path = parsed.Path
	if path == "" {
		path = "/"
	}

	return port, path, nil
}

// ServerManager manages multiple callback servers
type ServerManager struct {
	servers map[string]*callbackServer
	mu      sync.RWMutex
}

// NewServerManager creates a new server manager
func NewServerManager() *ServerManager {
	return &ServerManager{
		servers: make(map[string]*callbackServer),
	}
}

// Register registers a callback server with a name
func (m *ServerManager) Register(name string, server *callbackServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[name] = server
}

// Unregister removes a callback server
func (m *ServerManager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.servers, name)
}

// Get returns a registered server by name
func (m *ServerManager) Get(name string) (*callbackServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[name]
	return server, ok
}

// StopAll stops all registered servers
func (m *ServerManager) StopAll() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, server := range m.servers {
		if err := server.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop server %s: %w", name, err))
		}
	}
	m.servers = make(map[string]*callbackServer)

	return errs
}

// Count returns the number of registered servers
func (m *ServerManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.servers)
}
