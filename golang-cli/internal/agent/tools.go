// Package agent provides the core agent functionality for the Cline CLI.
package agent

import (
	"context"
	"fmt"
	"sync"
)

// Tool represents a tool that can be executed by the agent
type Tool interface {
	// GetName returns the unique name of the tool
	GetName() string

	// GetDescription returns a description of what the tool does
	GetDescription() string

	// GetUsage returns usage instructions for the tool
	GetUsage() string

	// GetParameters returns the parameters the tool accepts
	GetParameters() []ToolParameter

	// Execute executes the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (string, error)

	// Validate validates the parameters before execution
	Validate(params map[string]interface{}) error

	// IsDangerous returns true if the tool can perform dangerous operations
	IsDangerous() bool
}

// ToolParameter represents a parameter for a tool
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool in the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.GetName()] = tool
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// IsToolAvailable checks if a tool is registered
func (r *ToolRegistry) IsToolAvailable(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// GetAllTools returns all registered tools
func (r *ToolRegistry) GetAllTools() map[string]Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	tools := make(map[string]Tool, len(r.tools))
	for k, v := range r.tools {
		tools[k] = v
	}
	return tools
}

// GetToolNames returns a list of all tool names
func (r *ToolRegistry) GetToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Unregister removes a tool from the registry
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// BaseTool provides common functionality for tools
type BaseTool struct {
	Name        string
	Description string
	Usage       string
	Parameters  []ToolParameter
	Dangerous   bool
}

// GetName returns the tool name
func (t *BaseTool) GetName() string {
	return t.Name
}

// GetDescription returns the tool description
func (t *BaseTool) GetDescription() string {
	return t.Description
}

// GetUsage returns the tool usage
func (t *BaseTool) GetUsage() string {
	return t.Usage
}

// GetParameters returns the tool parameters
func (t *BaseTool) GetParameters() []ToolParameter {
	return t.Parameters
}

// IsDangerous returns true if the tool is dangerous
func (t *BaseTool) IsDangerous() bool {
	return t.Dangerous
}

// ValidateRequiredParams validates that all required parameters are present
func ValidateRequiredParams(params map[string]interface{}, required []string) error {
	for _, param := range required {
		if _, ok := params[param]; !ok {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}

// GetStringParam extracts a string parameter with a default value
func GetStringParam(params map[string]interface{}, name, defaultValue string) string {
	if val, ok := params[name]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetBoolParam extracts a boolean parameter with a default value
func GetBoolParam(params map[string]interface{}, name string, defaultValue bool) bool {
	if val, ok := params[name]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetIntParam extracts an int parameter with a default value
func GetIntParam(params map[string]interface{}, name string, defaultValue int) int {
	if val, ok := params[name]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}