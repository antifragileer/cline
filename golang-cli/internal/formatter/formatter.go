// Package formatter provides output formatting capabilities for the Cline CLI.
// It supports multiple output formats: plain text, JSON, and JSON Lines (streaming).
package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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

// Message represents a structured output message
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

// ProgressInfo represents progress information
type ProgressInfo struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

// Formatter handles output formatting
type Formatter struct {
	format   Format
	output   io.Writer
	errOut   io.Writer
	useColor bool
	verbose  bool
	encoder  *json.Encoder
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

	// Set up JSON encoder if needed
	if f.format == FormatJSON || f.format == FormatJSONLines {
		f.encoder = json.NewEncoder(f.output)
		if f.format == FormatJSON {
			f.encoder.SetIndent("", "  ")
		}
	}

	return f
}

// FormatSay formats a SAY message
func (f *Formatter) FormatSay(sayType string, text string, partial bool) error {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		msg := Message{
			Type:      "say",
			SubType:   sayType,
			Text:      text,
			Partial:   partial,
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainSay(sayType, text, partial)
		return nil
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatAsk formats an ASK message
func (f *Formatter) FormatAsk(askType string, text string) error {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		msg := Message{
			Type:      "ask",
			SubType:   askType,
			Text:      text,
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainAsk(askType, text)
		return nil
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// FormatError formats an error message
func (f *Formatter) FormatError(err error) error {
	switch f.format {
	case FormatJSON, FormatJSONLines:
		msg := Message{
			Type:      "error",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainError(err)
		return nil
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
		msg := Message{
			Type:      "info",
			Text:      text,
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainInfo(text)
		return nil
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
		msg := Message{
			Type:      "status",
			Text:      status,
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainStatus(status)
		return nil
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
		msg := Message{
			Type:      "progress",
			Progress:  &ProgressInfo{Current: current, Total: total},
			Timestamp: time.Now(),
		}
		return f.encoder.Encode(msg)
	case FormatPlain:
		f.printPlainProgress(current, total)
		return nil
	default:
		return fmt.Errorf("unknown format: %s", f.format)
	}
}

// Flush flushes any buffered output
func (f *Formatter) Flush() error {
	if flusher, ok := f.output.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// printPlainSay prints a SAY message in plain text format
func (f *Formatter) printPlainSay(sayType string, text string, partial bool) {
	if partial {
		// Don't print partial messages in plain text
		return
	}

	switch sayType {
	case "text":
		fmt.Fprintln(f.output, text)
	case "error":
		f.printError(text)
	case "command":
		f.printCommand(text)
	case "command_output":
		fmt.Fprintln(f.output, text)
	case "tool":
		f.printTool(text)
	case "completion_result":
		fmt.Fprintf(f.output, "\n%s\n", f.styleSuccess(text))
	case "thinking":
		if f.verbose {
			f.printThinking(text)
		}
	default:
		if f.verbose {
			fmt.Fprintf(f.output, "[%s] %s\n", sayType, text)
		}
	}
}

// printPlainAsk prints an ASK message in plain text format
func (f *Formatter) printPlainAsk(askType string, text string) {
	fmt.Fprintln(f.output)
	fmt.Fprintln(f.output, f.styleQuestion(text))
	switch askType {
	case "command_approval":
		fmt.Fprintln(f.output, f.styleHint("Approve command execution? (y/n/a): "))
	case "tool_approval":
		fmt.Fprintln(f.output, f.styleHint("Approve tool use? (y/n/a): "))
	case "browser_action_launch":
		fmt.Fprintln(f.output, f.styleHint("Approve browser action? (y/n/a): "))
	default:
		fmt.Fprintln(f.output, f.styleHint("Response (y/n/a): "))
	}
}

// printPlainError prints an error message in plain text format
func (f *Formatter) printPlainError(err error) {
	fmt.Fprintf(f.errOut, "%s: %v\n", f.styleError("Error"), err)
}

// printPlainInfo prints an info message in plain text format
func (f *Formatter) printPlainInfo(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("[INFO]"), text)
}

// printPlainStatus prints a status message in plain text format
func (f *Formatter) printPlainStatus(status string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("[STATUS]"), status)
}

// printPlainProgress prints a progress message in plain text format
func (f *Formatter) printPlainProgress(current, total int) {
	percentage := 0.0
	if total > 0 {
		percentage = float64(current) * 100.0 / float64(total)
	}
	bar := f.renderProgressBar(current, total, 30)
	fmt.Fprintf(f.output, "\r%s %3.0f%% %s", f.styleInfo("[PROGRESS]"), percentage, bar)
	if current >= total {
		fmt.Fprintln(f.output) // New line when complete
	}
}

// renderProgressBar renders a text-based progress bar
func (f *Formatter) renderProgressBar(current, total, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("-", width) + "]"
	}
	filled := int(float64(current) * float64(width) / float64(total))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", empty) + "]"
}

// printCommand prints a command execution message
func (f *Formatter) printCommand(cmd string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("Command:"), f.styleCommand(cmd))
}

// printTool prints a tool use message
func (f *Formatter) printTool(tool string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("Tool:"), f.styleTool(tool))
}

// printThinking prints a thinking message
func (f *Formatter) printThinking(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("Thinking:"), f.styleDim(text))
}

// printError prints an error message
func (f *Formatter) printError(text string) {
	fmt.Fprintf(f.errOut, "%s: %s\n", f.styleError("Error"), text)
}

// Color/style functions
func (f *Formatter) styleError(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[31m%s\033[0m", text) // Red
}

func (f *Formatter) styleSuccess(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[32m%s\033[0m", text) // Green
}

func (f *Formatter) styleInfo(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[36m%s\033[0m", text) // Cyan
}

func (f *Formatter) styleHint(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text) // Yellow
}

func (f *Formatter) styleQuestion(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[1m\033[35m%s\033[0m", text) // Bold Magenta
}

func (f *Formatter) styleCommand(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text) // Yellow
}

func (f *Formatter) styleTool(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[35m%s\033[0m", text) // Magenta
}

func (f *Formatter) styleDim(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[90m%s\033[0m", text) // Gray
}

// GetFormatFromString parses a format string
func GetFormatFromString(s string) Format {
	switch strings.ToLower(s) {
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