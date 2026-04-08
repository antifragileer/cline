// Package formatter provides output formatting capabilities for the Cline CLI.
// It supports multiple output formats: plain text, JSON, and JSON Lines (streaming).
package formatter

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cline/cline/golang-cli/internal/exit"
	"github.com/cline/cline/golang-cli/internal/task"
)

// Format represents the output format type
type Format string

const (
	// FormatPlain outputs human-readable text
	FormatPlain Format = "plain"
	// FormatJSON outputs a single JSON object (non-streaming)
	FormatJSON Format = "json"
	// FormatJSONLines outputs JSON lines for streaming (one object per line)
	FormatJSONLines Format = "jsonl"
)

// Message represents a structured output message (legacy, kept for compatibility)
type Message struct {
	// Type is the message type (say, ask, error, info, status, progress)
	Type string `json:"type"`
	// SubType is the specific subtype (sayType, askType)
	SubType string `json:"subType,omitempty"`
	// Text is the message content
	Text string `json:"text,omitempty"`
	// Partial indicates if this is a partial/streaming message
	Partial bool `json:"partial,omitempty"`
	// Timestamp is when the message was created
	Timestamp time.Time `json:"timestamp,omitempty"`
	// Error contains error information
	Error string `json:"error,omitempty"`
	// Progress contains progress information
	Progress *ProgressInfo `json:"progress,omitempty"`
	// Metadata contains additional type-specific data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProgressInfo represents progress information (legacy)
type ProgressInfo struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

// Formatter handles output formatting (legacy interface, kept for compatibility)
type Formatter struct {
	format   Format
	output   io.Writer
	errOut   io.Writer
	useColor bool
	verbose  bool

	// New formatters
	jsonFormatter  *JSONFormatter
	plainFormatter *PlainFormatter
}

// FormatterOption configures a Formatter
type FormatterOption func(*Formatter)

// WithFormat sets the output format
func WithFormat(format Format) FormatterOption {
	return func(f *Formatter) {
		f.format = format
	}
}

// WithOutput sets the output writer
func WithOutput(w io.Writer) FormatterOption {
	return func(f *Formatter) {
		f.output = w
	}
}

// WithErrorOutput sets the error output writer
func WithErrorOutput(w io.Writer) FormatterOption {
	return func(f *Formatter) {
		f.errOut = w
	}
}

// WithColor enables or disables color output
func WithColor(enabled bool) FormatterOption {
	return func(f *Formatter) {
		f.useColor = enabled
	}
}

// WithVerbose enables verbose output
func WithVerbose(enabled bool) FormatterOption {
	return func(f *Formatter) {
		f.verbose = enabled
	}
}

// NewFormatter creates a new Formatter with the specified options
func NewFormatter(opts ...FormatterOption) *Formatter {
	f := &Formatter{
		format:   FormatPlain,
		output:   os.Stdout,
		errOut:   os.Stderr,
		useColor: true,
		verbose:  false,
	}

	for _, opt := range opts {
		opt(f)
	}

	// Initialize sub-formatters
	f.jsonFormatter = NewJSONFormatter(f.output, f.errOut, f.format == FormatJSONLines)
	f.plainFormatter = NewPlainFormatter(f.output, f.errOut, f.useColor, f.verbose)

	return f
}

// SetExitHandler sets the exit handler for all formatters
func (f *Formatter) SetExitHandler(handler *exit.Handler) {
	if f.plainFormatter != nil {
		f.plainFormatter.SetExitHandler(handler)
	}
}

// FormatSay formats a SAY message
func (f *Formatter) FormatSay(sayType string, text string, partial bool) error {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		return f.jsonFormatter.FormatSayMessage(sayType, text, partial, nil)
	case FormatPlain:
		return f.plainFormatter.FormatSayMessage(sayType, text, partial)
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatAsk formats an ASK message
func (f *Formatter) FormatAsk(askType string, text string) (string, error) {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		// JSON mode auto-approves for scripting
		f.jsonFormatter.FormatAskMessage(askType, text, nil)
		return "yesButtonClicked", nil
	case FormatPlain:
		return f.plainFormatter.FormatAskMessage(askType, text)
	default:
		return "", fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatError formats an error message
func (f *Formatter) FormatError(err error) error {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		return f.jsonFormatter.FormatError(err)
	case FormatPlain:
		return f.plainFormatter.FormatError(err)
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatInfo formats an informational message
func (f *Formatter) FormatInfo(text string) error {
	if !f.verbose {
		return nil
	}

	switch f.format {
	case FormatJSON, FormatJSONLines:
		return f.jsonFormatter.FormatSayMessage("info", text, false, nil)
	case FormatPlain:
		return f.plainFormatter.FormatStatus(text)
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatStatus formats a status update
func (f *Formatter) FormatStatus(status string) error {
	if !f.verbose {
		return nil
	}

	switch f.format {
	case FormatJSON, FormatJSONLines:
		return f.jsonFormatter.FormatSayMessage("info", status, false, nil)
	case FormatPlain:
		return f.plainFormatter.FormatStatus(status)
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatProgress formats a progress update
func (f *Formatter) FormatProgress(current, total int) error {
	if !f.verbose {
		return nil
	}

	switch f.format {
	case FormatJSON, FormatJSONLines:
		return f.jsonFormatter.FormatProgress(current, total, fmt.Sprintf("%d/%d", current, total))
	case FormatPlain:
		return f.plainFormatter.FormatProgress(current, total, "")
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// Flush flushes any buffered output
func (f *Formatter) Flush() error {
	if f.jsonFormatter != nil {
		f.jsonFormatter.Flush()
	}
	if f.plainFormatter != nil {
		f.plainFormatter.Flush()
	}
	return nil
}

// GetFormatFromString parses a format string
func GetFormatFromString(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "jsonl", "jsonlines":
		return FormatJSONLines
	case "plain", "text":
		return FormatPlain
	default:
		return FormatPlain
	}
}

// CreateHandler creates a MessageHandler based on the format and options
func CreateHandler(format Format, output io.Writer, verbose, autoApprove bool) task.MessageHandler {
	switch format {
	case FormatJSON, FormatJSONLines:
		return NewJSONHandler(output, verbose)
	case FormatPlain:
		return NewPlainHandler(output, verbose, autoApprove)
	default:
		return NewPlainHandler(output, verbose, autoApprove)
	}
}

// CreateScriptingHandler creates a handler optimized for scripting/automation
func CreateScriptingHandler(output io.Writer, verbose bool) *ScriptingHandler {
	return NewScriptingHandler(output, verbose)
}
