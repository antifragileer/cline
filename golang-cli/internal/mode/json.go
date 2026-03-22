// Package mode provides JSON output formatting capabilities for the Cline CLI.
// It handles serialization of messages with deterministic output and optional
// pretty-printing.
package mode

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
)

// OutputType represents the type of output message.
type OutputType string

const (
	// OutputTypeSay represents a message from the AI assistant.
	OutputTypeSay OutputType = "say"
	// OutputTypeAsk represents a question from the AI assistant to the user.
	OutputTypeAsk OutputType = "ask"
	// OutputTypeText represents plain text content.
	OutputTypeText OutputType = "text"
	// OutputTypeToolUse represents a tool invocation.
	OutputTypeToolUse OutputType = "tool_use"
	// OutputTypeToolResult represents the result of a tool invocation.
	OutputTypeToolResult OutputType = "tool_result"
	// OutputTypeError represents an error message.
	OutputTypeError OutputType = "error"
	// OutputTypeUser represents a message from the user.
	OutputTypeUser OutputType = "user"
	// OutputTypeReasoning represents a reasoning/thinking message.
	OutputTypeReasoning OutputType = "reasoning"
)

// JSONOutput represents a JSON-serializable message output.
// All fields are sorted alphabetically for deterministic output.
type JSONOutput struct {
	// Ask is the ask type for ask messages (optional).
	Ask string `json:"ask,omitempty"`
	// Files is a list of file paths referenced in the message (optional).
	Files []string `json:"files,omitempty"`
	// Images is a list of image data/URIs referenced in the message (optional).
	Images []string `json:"images,omitempty"`
	// Partial indicates if this is a partial/streaming message (optional).
	Partial bool `json:"partial,omitempty"`
	// Reasoning contains the AI's reasoning/thinking process (optional).
	Reasoning string `json:"reasoning,omitempty"`
	// Say is the say type for say messages (optional).
	Say string `json:"say,omitempty"`
	// Text is the main message content (required).
	Text string `json:"text"`
	// Timestamp is the Unix timestamp in milliseconds (required).
	Timestamp int64 `json:"ts"`
	// Type is the message type (required).
	Type OutputType `json:"type"`
}

// JSONFormatter handles JSON output formatting with configurable options.
type JSONFormatter struct {
	// pretty enables pretty-printed output with indentation.
	pretty bool
	// writer is the output destination.
	writer io.Writer
}

// JSONOption configures a JSONFormatter.
type JSONOption func(*JSONFormatter)

// WithPretty enables pretty-printed JSON output.
func WithPretty(pretty bool) JSONOption {
	return func(f *JSONFormatter) {
		f.pretty = pretty
	}
}

// WithWriter sets the output writer for the formatter.
func WithWriter(w io.Writer) JSONOption {
	return func(f *JSONFormatter) {
		f.writer = w
	}
}

// NewJSONFormatter creates a new JSON formatter with the provided options.
func NewJSONFormatter(opts ...JSONOption) *JSONFormatter {
	f := &JSONFormatter{
		pretty: false,
		writer: nil,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Format serializes a JSONOutput to JSON with configured options.
func (f *JSONFormatter) Format(output *JSONOutput) ([]byte, error) {
	if output == nil {
		return nil, fmt.Errorf("cannot format nil output")
	}

	// Ensure timestamp is set
	if output.Timestamp == 0 {
		output.Timestamp = time.Now().UnixMilli()
	}

	var data []byte
	var err error

	if f.pretty {
		data, err = json.MarshalIndent(output, "", "  ")
	} else {
		data, err = f.marshalDeterministic(output)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON output: %w", err)
	}

	return data, nil
}

// FormatToWriter serializes and writes output directly to the configured writer.
func (f *JSONFormatter) FormatToWriter(output *JSONOutput) error {
	if f.writer == nil {
		return fmt.Errorf("no writer configured")
	}

	data, err := f.Format(output)
	if err != nil {
		return err
	}

	_, err = f.writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}

	_, err = f.writer.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// FormatError creates and formats an error message.
func (f *JSONFormatter) FormatError(errMsg string) ([]byte, error) {
	output := &JSONOutput{
		Type:      OutputTypeError,
		Text:      errMsg,
		Timestamp: time.Now().UnixMilli(),
	}

	return f.Format(output)
}

// FormatErrorToWriter creates and writes an error message directly.
func (f *JSONFormatter) FormatErrorToWriter(errMsg string) error {
	if f.writer == nil {
		return fmt.Errorf("no writer configured")
	}

	data, err := f.FormatError(errMsg)
	if err != nil {
		return err
	}

	_, err = f.writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write error output: %w", err)
	}

	_, err = f.writer.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// marshalDeterministic produces JSON with sorted keys for deterministic output.
func (f *JSONFormatter) marshalDeterministic(output *JSONOutput) ([]byte, error) {
	// Build a map with only non-zero values for deterministic output
	m := make(map[string]interface{})

	// Always include required fields
	m["text"] = output.Text
	m["ts"] = output.Timestamp
	m["type"] = output.Type

	// Include optional fields only when set
	if output.Ask != "" {
		m["ask"] = output.Ask
	}
	if len(output.Files) > 0 {
		m["files"] = output.Files
	}
	if len(output.Images) > 0 {
		m["images"] = output.Images
	}
	if output.Partial {
		m["partial"] = output.Partial
	}
	if output.Reasoning != "" {
		m["reasoning"] = output.Reasoning
	}
	if output.Say != "" {
		m["say"] = output.Say
	}

	return marshalSorted(m)
}

// marshalSorted produces JSON with alphabetically sorted keys.
func marshalSorted(v map[string]interface{}) ([]byte, error) {
	// Get sorted keys
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build JSON manually for guaranteed ordering
	var result []byte
	result = append(result, '{')

	for i, k := range keys {
		if i > 0 {
			result = append(result, ',')
		}

		// Marshal key
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		result = append(result, keyJSON...)
		result = append(result, ':')

		// Marshal value
		valJSON, err := json.Marshal(v[k])
		if err != nil {
			return nil, err
		}
		result = append(result, valJSON...)
	}

	result = append(result, '}')

	return result, nil
}

// NewJSONOutput creates a new JSONOutput with the required fields.
func NewJSONOutput(msgType OutputType, text string) *JSONOutput {
	return &JSONOutput{
		Type:      msgType,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewSayOutput creates a new say message output.
func NewSayOutput(text, sayType string) *JSONOutput {
	return &JSONOutput{
		Type:      OutputTypeSay,
		Text:      text,
		Say:       sayType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewAskOutput creates a new ask message output.
func NewAskOutput(text, askType string) *JSONOutput {
	return &JSONOutput{
		Type:      OutputTypeAsk,
		Text:      text,
		Ask:       askType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewTextOutput creates a new text message output.
func NewTextOutput(text string) *JSONOutput {
	return &JSONOutput{
		Type:      OutputTypeText,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewErrorOutput creates a new error message output.
func NewErrorOutput(text string) *JSONOutput {
	return &JSONOutput{
		Type:      OutputTypeError,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewReasoningOutput creates a new reasoning message output.
func NewReasoningOutput(text string) *JSONOutput {
	return &JSONOutput{
		Type:      OutputTypeReasoning,
		Text:      text,
		Timestamp: time.Now().UnixMilli(),
	}
}

// SetReasoning sets the reasoning field and returns the output for chaining.
func (o *JSONOutput) SetReasoning(reasoning string) *JSONOutput {
	o.Reasoning = reasoning
	return o
}

// SetSay sets the say field and returns the output for chaining.
func (o *JSONOutput) SetSay(say string) *JSONOutput {
	o.Say = say
	return o
}

// SetAsk sets the ask field and returns the output for chaining.
func (o *JSONOutput) SetAsk(ask string) *JSONOutput {
	o.Ask = ask
	return o
}

// SetPartial sets the partial field and returns the output for chaining.
func (o *JSONOutput) SetPartial(partial bool) *JSONOutput {
	o.Partial = partial
	return o
}

// SetImages sets the images field and returns the output for chaining.
func (o *JSONOutput) SetImages(images []string) *JSONOutput {
	o.Images = images
	return o
}

// SetFiles sets the files field and returns the output for chaining.
func (o *JSONOutput) SetFiles(files []string) *JSONOutput {
	o.Files = files
	return o
}

// SetTimestamp sets a specific timestamp and returns the output for chaining.
func (o *JSONOutput) SetTimestamp(ts int64) *JSONOutput {
	o.Timestamp = ts
	return o
}