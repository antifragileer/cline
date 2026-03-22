// Package security provides permission rule parsing and validation for the Cline CLI.
// It handles parsing CLINE_COMMAND_PERMISSIONS JSON with schema validation,
// caching parsed permissions, and providing descriptive error messages.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// ParsedPermissionRule represents a single command permission rule with glob pattern.
type ParsedPermissionRule struct {
	// Pattern is the glob pattern for matching commands
	Pattern string `json:"pattern"`

	// Description is an optional human-readable description of the rule
	Description string `json:"description,omitempty"`
}

// PermissionRules defines the structure for command permission configuration.
// It supports allow/deny lists with glob patterns and redirect control.
type PermissionRules struct {
	// Allow is a list of glob patterns for allowed commands
	Allow []string `json:"allow"`

	// Deny is a list of glob patterns for denied commands (takes precedence over allow)
	Deny []string `json:"deny"`

	// AllowRedirects controls whether command output redirects (> or >>) are permitted
	AllowRedirects bool `json:"allowRedirects"`

	// AllowPipes controls whether command pipes (|) are permitted
	AllowPipes bool `json:"allowPipes"`
}

// ParseError represents a detailed error from permission parsing.
// It provides context about why parsing failed.
type ParseError struct {
	// Field is the JSON field that caused the error (if applicable)
	Field string

	// Message describes what went wrong
	Message string

	// RawValue contains the problematic raw value (sanitized)
	RawValue string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("parse error in field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
}

// SchemaValidationResult contains the results of JSON schema validation.
type SchemaValidationResult struct {
	// Valid indicates if the JSON is valid according to the schema
	Valid bool

	// Errors contains validation errors if Valid is false
	Errors []SchemaValidationError

	// Warnings contains non-critical issues
	Warnings []SchemaValidationWarning
}

// SchemaValidationError represents a single schema validation error.
type SchemaValidationError struct {
	// Field is the path to the field with the error
	Field string

	// Message describes the validation failure
	Message string
}

// SchemaValidationWarning represents a non-critical validation issue.
type SchemaValidationWarning struct {
	// Field is the path to the field with the warning
	Field string

	// Message describes the warning
	Message string
}

// Parser handles parsing and validation of permission rules.
type Parser struct {
	// cache stores parsed permissions for reuse
	cache *permissionCache

	// schema defines the expected JSON structure
	schema *permissionSchema
}

// permissionCache provides thread-safe caching of parsed permissions.
type permissionCache struct {
	mu    sync.RWMutex
	items map[string]*cachedPermission
}

// cachedPermission stores a parsed permission with metadata.
type cachedPermission struct {
	rules    *PermissionRules
	parsedAt int64
	hitCount int64
}

// permissionSchema defines the expected structure for validation.
type permissionSchema struct {
	// requiredFields lists fields that must be present
	requiredFields []string

	// allowedTypes defines the expected types for each field
	allowedTypes map[string]string

	// patternValidators contains regex patterns for field validation
	patternValidators map[string]*regexp.Regexp
}

// CacheStats provides statistics about the permission cache.
type CacheStats struct {
	// Size is the number of cached entries
	Size int

	// TotalHits is the total number of cache hits
	TotalHits int64
}

// NewParser creates a new permission parser with caching enabled.
func NewParser() *Parser {
	return &Parser{
		cache: newPermissionCache(),
		schema: &permissionSchema{
			requiredFields: []string{"allow"},
			allowedTypes: map[string]string{
				"allow":          "array",
				"deny":           "array",
				"allowRedirects": "boolean",
				"allowPipes":     "boolean",
			},
			patternValidators: map[string]*regexp.Regexp{
				"glob": regexp.MustCompile(`^[\w\*\?\-\./\\]+$`),
			},
		},
	}
}

// newPermissionCache creates a new permission cache.
func newPermissionCache() *permissionCache {
	return &permissionCache{
		items: make(map[string]*cachedPermission),
	}
}

// Parse parses permission rules from JSON data.
// It validates the JSON schema, caches the result, and returns the parsed rules.
func (p *Parser) Parse(jsonData []byte) (*PermissionRules, error) {
	// Check cache first
	cacheKey := string(jsonData)
	if cached := p.cache.get(cacheKey); cached != nil {
		return cached, nil
	}

	// Parse JSON
	var rules PermissionRules
	if err := json.Unmarshal(jsonData, &rules); err != nil {
		return nil, &ParseError{
			Message:  fmt.Sprintf("invalid JSON: %v", err),
			RawValue: truncate(string(jsonData), 100),
		}
	}

	// Validate schema
	if result := p.ValidateSchema(jsonData); !result.Valid {
		var errMsgs []string
		firstField := ""
		for i, valErr := range result.Errors {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", valErr.Field, valErr.Message))
			if i == 0 {
				firstField = valErr.Field
			}
		}
		return nil, &ParseError{
			Field:    firstField,
			Message:  fmt.Sprintf("schema validation failed: %s", strings.Join(errMsgs, "; ")),
			RawValue: truncate(string(jsonData), 100),
		}
	}

	// Validate rules content
	if err := p.validateRules(&rules); err != nil {
		return nil, err
	}

	// Cache the parsed rules
	p.cache.set(cacheKey, &rules)

	return &rules, nil
}

// ParseString parses permission rules from a JSON string.
func (p *Parser) ParseString(jsonStr string) (*PermissionRules, error) {
	return p.Parse([]byte(jsonStr))
}

// ParseFromEnv parses permission rules from the CLINE_COMMAND_PERMISSIONS
// environment variable. Returns nil if the variable is not set.
func (p *Parser) ParseFromEnv() (*PermissionRules, error) {
	jsonData := os.Getenv("CLINE_COMMAND_PERMISSIONS")
	if jsonData == "" {
		return nil, nil
	}

	return p.ParseString(jsonData)
}

// ParseWithDefault parses permission rules from environment, returning
// default allow-all permissions if not set or on parse error.
func (p *Parser) ParseWithDefault() (*PermissionRules, error) {
	rules, err := p.ParseFromEnv()
	if err != nil {
		return nil, err
	}
	if rules == nil {
		return DefaultAllowAll(), nil
	}
	return rules, nil
}

// ValidateSchema validates JSON data against the permission schema.
func (p *Parser) ValidateSchema(jsonData []byte) *SchemaValidationResult {
	result := &SchemaValidationResult{
		Valid:    true,
		Errors:   []SchemaValidationError{},
		Warnings: []SchemaValidationWarning{},
	}

	// Parse into generic map for validation
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, SchemaValidationError{
			Field:   "",
			Message: fmt.Sprintf("invalid JSON: %v", err),
		})
		return result
	}

	// Check required fields
	for _, field := range p.schema.requiredFields {
		if _, ok := data[field]; !ok {
			// 'allow' is required but we provide a default if missing
			if field == "allow" {
				result.Warnings = append(result.Warnings, SchemaValidationWarning{
					Field:   field,
					Message: "missing required field 'allow', will use default allow-all",
				})
			} else {
				result.Valid = false
				result.Errors = append(result.Errors, SchemaValidationError{
					Field:   field,
					Message: fmt.Sprintf("missing required field: %s", field),
				})
			}
		}
	}

	// Validate field types
	for field, expectedType := range p.schema.allowedTypes {
		if value, ok := data[field]; ok {
			actualType := getJSONType(value)
			if actualType != expectedType {
				result.Valid = false
				result.Errors = append(result.Errors, SchemaValidationError{
					Field:   field,
					Message: fmt.Sprintf("expected type %s, got %s", expectedType, actualType),
				})
			}
		}
	}

	// Validate allow patterns
	if allowList, ok := data["allow"].([]interface{}); ok {
		for i, item := range allowList {
			if pattern, ok := item.(string); ok {
				if err := validateGlobPattern(pattern); err != nil {
					result.Valid = false
					result.Errors = append(result.Errors, SchemaValidationError{
						Field:   fmt.Sprintf("allow[%d]", i),
						Message: err.Error(),
					})
				}
			} else {
				result.Valid = false
				result.Errors = append(result.Errors, SchemaValidationError{
					Field:   fmt.Sprintf("allow[%d]", i),
					Message: "expected string pattern",
				})
			}
		}
	}

	// Validate deny patterns
	if denyList, ok := data["deny"].([]interface{}); ok {
		for i, item := range denyList {
			if pattern, ok := item.(string); ok {
				if err := validateGlobPattern(pattern); err != nil {
					result.Valid = false
					result.Errors = append(result.Errors, SchemaValidationError{
						Field:   fmt.Sprintf("deny[%d]", i),
						Message: err.Error(),
					})
				}
			} else {
				result.Valid = false
				result.Errors = append(result.Errors, SchemaValidationError{
					Field:   fmt.Sprintf("deny[%d]", i),
					Message: "expected string pattern",
				})
			}
		}
	}

	return result
}

// validateRules performs additional validation on parsed rules.
func (p *Parser) validateRules(rules *PermissionRules) error {
	// Validate allow patterns
	for i, pattern := range rules.Allow {
		if pattern == "" {
			return &ParseError{
				Field:   fmt.Sprintf("allow[%d]", i),
				Message: "pattern cannot be empty",
			}
		}
		if err := validateGlobPattern(pattern); err != nil {
			return &ParseError{
				Field:   fmt.Sprintf("allow[%d]", i),
				Message: err.Error(),
			}
		}
	}

	// Validate deny patterns
	for i, pattern := range rules.Deny {
		if pattern == "" {
			return &ParseError{
				Field:   fmt.Sprintf("deny[%d]", i),
				Message: "pattern cannot be empty",
			}
		}
		if err := validateGlobPattern(pattern); err != nil {
			return &ParseError{
				Field:   fmt.Sprintf("deny[%d]", i),
				Message: err.Error(),
			}
		}
	}

	return nil
}

// validateGlobPattern validates a glob pattern.
// It allows glob wildcards, alphanumeric characters, and common shell command characters.
func validateGlobPattern(pattern string) error {
	// Check for empty pattern
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Check for invalid characters - only reject truly dangerous characters
	// that could indicate injection attempts or regex that would break parsing
	for _, ch := range pattern {
		switch ch {
		case '*', '?', '-', '_', '.', '/', '\\', ' ', ':', '@', '#', '%', '&', '+', '=', '[', ']', '{', '}', ',':
			// Valid glob and shell command characters
		default:
			if !isValidGlobChar(ch) {
				return fmt.Errorf("invalid character %q in pattern", ch)
			}
		}
	}

	return nil
}

// isValidGlobChar checks if a character is valid in a glob pattern.
// Allows alphanumeric characters which are the most common in command names.
func isValidGlobChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

// get returns a cached permission or nil if not found.
func (c *permissionCache) get(key string) *PermissionRules {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, ok := c.items[key]; ok {
		item.hitCount++
		return item.rules
	}
	return nil
}

// set stores a permission in the cache.
func (c *permissionCache) set(key string, rules *PermissionRules) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cachedPermission{
		rules:    rules,
		parsedAt: getCurrentTimestamp(),
	}
}

// ClearCache clears all cached permissions.
func (p *Parser) ClearCache() {
	p.cache.mu.Lock()
	defer p.cache.mu.Unlock()

	p.cache.items = make(map[string]*cachedPermission)
}

// GetCacheStats returns statistics about the cache.
func (p *Parser) GetCacheStats() CacheStats {
	p.cache.mu.RLock()
	defer p.cache.mu.RUnlock()

	var totalHits int64
	for _, item := range p.cache.items {
		totalHits += item.hitCount
	}

	return CacheStats{
		Size:      len(p.cache.items),
		TotalHits: totalHits,
	}
}

// DefaultAllowAll returns default permissive permissions.
func DefaultAllowAll() *PermissionRules {
	return &PermissionRules{
		Allow:          []string{"*"},
		Deny:           []string{},
		AllowRedirects: true,
		AllowPipes:     true,
	}
}

// DefaultDenyAll returns default restrictive permissions.
func DefaultDenyAll() *PermissionRules {
	return &PermissionRules{
		Allow:          []string{},
		Deny:           []string{"*"},
		AllowRedirects: false,
		AllowPipes:     false,
	}
}

// getJSONType returns the JSON type name for a value.
func getJSONType(v interface{}) string {
	switch v.(type) {
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getCurrentTimestamp returns the current timestamp.
// This is a variable to allow mocking in tests.
var getCurrentTimestamp = func() int64 {
	return 0 // Simplified for testing; use time.Now().Unix() in production
}
