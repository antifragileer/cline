// Package exit provides enhanced error handling and recovery for the Cline CLI.
// This file extends the basic exit handling with Node.js parity features.
package exit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrorCategory categorizes errors for appropriate handling.
type ErrorCategory int

const (
	// ErrorCategoryUnknown is an uncategorized error
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryValidation is a validation error (invalid input)
	ErrorCategoryValidation
	// ErrorCategoryConfiguration is a configuration error
	ErrorCategoryConfiguration
	// ErrorCategoryConnection is a connection/network error
	ErrorCategoryConnection
	// ErrorCategoryAuthentication is an authentication error
	ErrorCategoryAuthentication
	// ErrorCategoryAuthorization is an authorization/permission error
	ErrorCategoryAuthorization
	// ErrorCategoryTimeout is a timeout error
	ErrorCategoryTimeout
	// ErrorCategoryCancelled is a cancelled operation
	ErrorCategoryCancelled
	// ErrorCategoryTaskFailure is a task execution failure
	ErrorCategoryTaskFailure
	// ErrorCategorySystem is a system/internal error
	ErrorCategorySystem
	// ErrorCategoryUser is a user-induced error
	ErrorCategoryUser
)

// String returns the string representation of the error category.
func (ec ErrorCategory) String() string {
	switch ec {
	case ErrorCategoryValidation:
		return "validation"
	case ErrorCategoryConfiguration:
		return "configuration"
	case ErrorCategoryConnection:
		return "connection"
	case ErrorCategoryAuthentication:
		return "authentication"
	case ErrorCategoryAuthorization:
		return "authorization"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryCancelled:
		return "cancelled"
	case ErrorCategoryTaskFailure:
		return "task_failure"
	case ErrorCategorySystem:
		return "system"
	case ErrorCategoryUser:
		return "user"
	default:
		return "unknown"
	}
}

// CategorizedError represents an error with category and context.
type CategorizedError struct {
	// Original error
	Err error
	// Category of the error
	Category ErrorCategory
	// Exit code to use
	ExitCode Code
	// User-facing message
	Message string
	// Suggestion for resolving the error
	Suggestion string
	// Context information
	Context map[string]interface{}
	// Timestamp when error occurred
	Timestamp time.Time
	// Recoverable indicates if the operation can be retried
	Recoverable bool
	// Retryable indicates if the error is transient and retry may succeed
	Retryable bool
}

// Error implements the error interface.
func (ce *CategorizedError) Error() string {
	if ce.Message != "" {
		return ce.Message
	}
	if ce.Err != nil {
		return ce.Err.Error()
	}
	return "unknown error"
}

// Unwrap returns the original error.
func (ce *CategorizedError) Unwrap() error {
	return ce.Err
}

// Format formats the error with full details.
func (ce *CategorizedError) Format(verbose bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Error: %s\n", ce.Error()))

	if verbose {
		if ce.Category != ErrorCategoryUnknown {
			sb.WriteString(fmt.Sprintf("Category: %s\n", ce.Category))
		}

		if ce.Err != nil && ce.Err.Error() != ce.Error() {
			sb.WriteString(fmt.Sprintf("Details: %s\n", ce.Err.Error()))
		}

		if len(ce.Context) > 0 {
			sb.WriteString("Context:\n")
			for k, v := range ce.Context {
				sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
			}
		}

		sb.WriteString(fmt.Sprintf("Exit Code: %d\n", ce.ExitCode))
	}

	if ce.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\nSuggestion: %s\n", ce.Suggestion))
	}

	return sb.String()
}

// ErrorClassifier classifies errors into categories.
type ErrorClassifier struct {
	// Custom classifiers
	classifiers []ErrorClassifierFunc
	mu          sync.RWMutex
}

// ErrorClassifierFunc is a function that classifies an error.
type ErrorClassifierFunc func(error) (*CategorizedError, bool)

// NewErrorClassifier creates a new error classifier.
func NewErrorClassifier() *ErrorClassifier {
	ec := &ErrorClassifier{
		classifiers: make([]ErrorClassifierFunc, 0),
	}

	// Add default classifiers
	ec.AddClassifier(classifyValidationErrors)
	ec.AddClassifier(classifyConnectionErrors)
	ec.AddClassifier(classifyAuthenticationErrors)
	ec.AddClassifier(classifyTimeoutErrors)
	ec.AddClassifier(classifyConfigurationErrors)
	ec.AddClassifier(classifyPermissionErrors)

	return ec
}

// AddClassifier adds a custom error classifier.
func (ec *ErrorClassifier) AddClassifier(fn ErrorClassifierFunc) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.classifiers = append(ec.classifiers, fn)
}

// Classify classifies an error and returns a categorized error.
func (ec *ErrorClassifier) Classify(err error) *CategorizedError {
	if err == nil {
		return nil
	}

	// Check if already categorized
	if ce, ok := err.(*CategorizedError); ok {
		return ce
	}

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Try each classifier
	for _, classifier := range ec.classifiers {
		if ce, ok := classifier(err); ok {
			return ce
		}
	}

	// Default classification
	return &CategorizedError{
		Err:         err,
		Category:    ErrorCategoryUnknown,
		ExitCode:    GeneralError,
		Message:     err.Error(),
		Timestamp:   time.Now(),
		Recoverable: false,
		Retryable:   false,
	}
}

// Default classifier implementations

func classifyValidationErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"invalid argument",
		"invalid flag",
		"flag provided but not defined",
		"required flag",
		"missing required",
		"invalid value",
		"parse error",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryValidation,
				ExitCode:    InvalidArguments,
				Message:     "Invalid command arguments",
				Suggestion:  "Check the command syntax and try again. Use --help for usage information.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   false,
			}, true
		}
	}

	return nil, false
}

func classifyConnectionErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"no such host",
		"network is unreachable",
		"timeout awaiting response",
		"cannot connect",
		"grpc",
		"rpc error",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryConnection,
				ExitCode:    ConnectionError,
				Message:     "Connection error - unable to reach Cline core extension",
				Suggestion:  "Ensure the Cline extension is running and the gRPC endpoint is accessible.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   true,
				Context: map[string]interface{}{
					"error_details": err.Error(),
				},
			}, true
		}
	}

	return nil, false
}

func classifyAuthenticationErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"authentication failed",
		"unauthorized",
		"invalid token",
		"expired token",
		"api key invalid",
		"401",
		"403",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryAuthentication,
				ExitCode:    PermissionDenied,
				Message:     "Authentication failed",
				Suggestion:  "Check your API credentials with 'cline auth status' or run 'cline auth login'.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   false,
			}, true
		}
	}

	return nil, false
}

func classifyTimeoutErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"timeout",
		"deadline exceeded",
		"context deadline",
		"operation timed out",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryTimeout,
				ExitCode:    Timeout,
				Message:     "Operation timed out",
				Suggestion:  "The operation took too long to complete. Try again or increase the timeout with --timeout.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   true,
			}, true
		}
	}

	return nil, false
}

func classifyConfigurationErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"configuration",
		"config file",
		"invalid config",
		"missing config",
		"parse config",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryConfiguration,
				ExitCode:    ConfigurationError,
				Message:     "Configuration error",
				Suggestion:  "Check your configuration file or run 'cline config' to review settings.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   false,
			}, true
		}
	}

	return nil, false
}

func classifyPermissionErrors(err error) (*CategorizedError, bool) {
	errStr := strings.ToLower(err.Error())

	patterns := []string{
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
		"cannot access",
		"operation not permitted",
	}

	for _, pattern := range patterns {
		if strings.Contains(errStr, pattern) {
			return &CategorizedError{
				Err:         err,
				Category:    ErrorCategoryAuthorization,
				ExitCode:    PermissionDenied,
				Message:     "Permission denied",
				Suggestion:  "Check file permissions or run with appropriate privileges.",
				Timestamp:   time.Now(),
				Recoverable: true,
				Retryable:   false,
			}, true
		}
	}

	return nil, false
}

// ErrorRecovery provides error recovery mechanisms.
type ErrorRecovery struct {
	classifier *ErrorClassifier
	maxRetries int
	retryDelay time.Duration
}

// NewErrorRecovery creates a new error recovery handler.
func NewErrorRecovery() *ErrorRecovery {
	return &ErrorRecovery{
		classifier: NewErrorClassifier(),
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

// ExecuteWithRetry executes a function with retry logic.
func (er *ErrorRecovery) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= er.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(er.retryDelay * time.Duration(attempt)):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		ce := er.classifier.Classify(err)

		// Don't retry if not retryable
		if !ce.Retryable {
			return err
		}

		// Log retry attempt (in real implementation)
		if attempt < er.maxRetries {
			// fmt.Fprintf(os.Stderr, "Attempt %d failed, retrying... (%s)\n", attempt+1, ce.Message)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// SetRetryDelay sets the delay between retry attempts.
func (er *ErrorRecovery) SetRetryDelay(delay time.Duration) {
	er.retryDelay = delay
}

// SetMaxRetries sets the maximum number of retry attempts.
func (er *ErrorRecovery) SetMaxRetries(retries int) {
	er.maxRetries = retries
}

// HandleError handles an error with appropriate user feedback.
func (er *ErrorRecovery) HandleError(err error, writer io.Writer, verbose bool) Code {
	if err == nil {
		return Success
	}

	ce := er.classifier.Classify(err)

	// Write error message
	fmt.Fprintln(writer, ce.Format(verbose))

	return ce.ExitCode
}

// EnhancedExitHandler provides enhanced exit handling with error recovery.
type EnhancedExitHandler struct {
	*Handler
	classifier   *ErrorClassifier
	recovery     *ErrorRecovery
	errorLog     []ErrorLogEntry
	errorLogMu   sync.RWMutex
	maxErrorLog  int
	verbose      bool
	errorWriter  io.Writer
}

// ErrorLogEntry represents a logged error.
type ErrorLogEntry struct {
	Timestamp time.Time
	Error     string
	Category  string
	ExitCode  int
	Recovered bool
}

// NewEnhancedExitHandler creates a new enhanced exit handler.
func NewEnhancedExitHandler() *EnhancedExitHandler {
	return &EnhancedExitHandler{
		Handler:     NewHandler(),
		classifier:  NewErrorClassifier(),
		recovery:    NewErrorRecovery(),
		errorLog:    make([]ErrorLogEntry, 0),
		maxErrorLog: 100,
		verbose:     false,
		errorWriter: os.Stderr,
	}
}

// SetVerbose sets verbose mode for error output.
func (eh *EnhancedExitHandler) SetVerbose(verbose bool) {
	eh.verbose = verbose
}

// SetErrorWriter sets the writer for error output.
func (eh *EnhancedExitHandler) SetErrorWriter(w io.Writer) {
	eh.errorWriter = w
}

// HandleError handles an error and logs it.
func (eh *EnhancedExitHandler) HandleError(err error) Code {
	code := eh.recovery.HandleError(err, eh.errorWriter, eh.verbose)

	// Log the error
	ce := eh.classifier.Classify(err)
	eh.logError(err, ce, code, false)

	// Set exit code
	eh.SetExitCode(code)

	return code
}

// HandleErrorWithRecovery attempts to recover from an error.
func (eh *EnhancedExitHandler) HandleErrorWithRecovery(ctx context.Context, fn func() error) error {
	err := eh.recovery.ExecuteWithRetry(ctx, fn)
	if err != nil {
		eh.HandleError(err)
		return err
	}
	return nil
}

// logError logs an error entry.
func (eh *EnhancedExitHandler) logError(err error, ce *CategorizedError, code Code, recovered bool) {
	eh.errorLogMu.Lock()
	defer eh.errorLogMu.Unlock()

	entry := ErrorLogEntry{
		Timestamp: time.Now(),
		Error:     err.Error(),
		Category:  ce.Category.String(),
		ExitCode:  int(code),
		Recovered: recovered,
	}

	eh.errorLog = append(eh.errorLog, entry)

	// Trim log if too large
	if len(eh.errorLog) > eh.maxErrorLog {
		eh.errorLog = eh.errorLog[len(eh.errorLog)-eh.maxErrorLog:]
	}
}

// GetErrorLog returns the error log.
func (eh *EnhancedExitHandler) GetErrorLog() []ErrorLogEntry {
	eh.errorLogMu.RLock()
	defer eh.errorLogMu.RUnlock()

	// Return a copy
	log := make([]ErrorLogEntry, len(eh.errorLog))
	copy(log, eh.errorLog)
	return log
}

// ClearErrorLog clears the error log.
func (eh *EnhancedExitHandler) ClearErrorLog() {
	eh.errorLogMu.Lock()
	defer eh.errorLogMu.Unlock()
	eh.errorLog = make([]ErrorLogEntry, 0)
}

// GetErrorStats returns error statistics.
func (eh *EnhancedExitHandler) GetErrorStats() ErrorStats {
	eh.errorLogMu.RLock()
	defer eh.errorLogMu.RUnlock()

	stats := ErrorStats{
		TotalErrors:   len(eh.errorLog),
		ByCategory:    make(map[string]int),
		ByExitCode:    make(map[int]int),
		RecoveredErrors: 0,
	}

	for _, entry := range eh.errorLog {
		stats.ByCategory[entry.Category]++
		stats.ByExitCode[entry.ExitCode]++
		if entry.Recovered {
			stats.RecoveredErrors++
		}
	}

	return stats
}

// ErrorStats contains error statistics.
type ErrorStats struct {
	TotalErrors     int
	ByCategory      map[string]int
	ByExitCode      map[int]int
	RecoveredErrors int
}

// WrapError wraps an error with additional context.
func WrapError(err error, category ErrorCategory, message string) *CategorizedError {
	if err == nil {
		return nil
	}

	var code Code = GeneralError
	switch category {
	case ErrorCategoryValidation:
		code = InvalidArguments
	case ErrorCategoryConfiguration:
		code = ConfigurationError
	case ErrorCategoryConnection:
		code = ConnectionError
	case ErrorCategoryAuthentication, ErrorCategoryAuthorization:
		code = PermissionDenied
	case ErrorCategoryTimeout:
		code = Timeout
	case ErrorCategoryTaskFailure:
		code = TaskFailed
	}

	return &CategorizedError{
		Err:         err,
		Category:    category,
		ExitCode:    code,
		Message:     message,
		Timestamp:   time.Now(),
		Recoverable: false,
		Retryable:   false,
	}
}

// IsCategorizedError checks if an error is a categorized error.
func IsCategorizedError(err error) (*CategorizedError, bool) {
	var ce *CategorizedError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}

// CategorizeError categorizes an error if not already categorized.
func CategorizeError(err error) *CategorizedError {
	if err == nil {
		return nil
	}

	if ce, ok := IsCategorizedError(err); ok {
		return ce
	}

	classifier := NewErrorClassifier()
	return classifier.Classify(err)
}

// ExitWithError exits with an error, categorizing it if needed.
func ExitWithError(err error, verbose bool) {
	if err == nil {
		os.Exit(int(Success))
	}

	ce := CategorizeError(err)

	if verbose {
		fmt.Fprintf(os.Stderr, "Error: %s\n", ce.Format(true))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", ce.Error())
		if ce.Suggestion != "" {
			fmt.Fprintf(os.Stderr, "\n%s\n", ce.Suggestion)
		}
	}

	os.Exit(int(ce.ExitCode))
}

// RecoverFromPanic recovers from a panic and returns an error.
func RecoverFromPanic(r interface{}) error {
	if r == nil {
		return nil
	}

	var err error
	switch v := r.(type) {
	case error:
		err = v
	case string:
		err = errors.New(v)
	default:
		err = fmt.Errorf("panic: %v", v)
	}

	return WrapError(err, ErrorCategorySystem, fmt.Sprintf("Internal error occurred: %v", err))
}

// SafeExecute executes a function with panic recovery.
func SafeExecute(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = RecoverFromPanic(r)
		}
	}()

	return fn()
}