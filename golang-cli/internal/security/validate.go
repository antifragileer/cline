// Package security provides command validation and security enforcement
// for the Cline CLI. It implements permission-based command validation
// with glob matching, caching, and detailed logging.
package security

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ValidationResult represents the outcome of a command validation
type ValidationResult struct {
	// Allowed indicates if the command is permitted
	Allowed bool `json:"allowed"`

	// Command is the original command that was validated
	Command string `json:"command"`

	// NormalizedCommand is the command after normalization
	NormalizedCommand string `json:"normalized_command"`

	// MatchedRule is the rule that matched (if any)
	MatchedRule *PermissionRule `json:"matched_rule,omitempty"`

	// DeniedBy is the deny rule that blocked the command (if denied)
	DeniedBy *PermissionRule `json:"denied_by,omitempty"`

	// Reason provides a human-readable explanation
	Reason string `json:"reason"`

	// Segments contains validation results for individual command segments
	Segments []SegmentValidation `json:"segments,omitempty"`

	// Timestamp when the validation occurred
	Timestamp time.Time `json:"timestamp"`

	// CacheHit indicates if this result was retrieved from cache
	CacheHit bool `json:"cache_hit"`

	// ValidationDuration is the time taken to validate
	ValidationDuration time.Duration `json:"validation_duration"`
}

// SegmentValidation represents validation result for a command segment
type SegmentValidation struct {
	// Segment is the command segment (e.g., "git", "clone")
	Segment string `json:"segment"`

	// Index is the position in the command
	Index int `json:"index"`

	// Allowed indicates if this segment is permitted
	Allowed bool `json:"allowed"`

	// MatchedPatterns lists patterns that matched this segment
	MatchedPatterns []string `json:"matched_patterns,omitempty"`

	// DeniedBy lists deny patterns that blocked this segment
	DeniedBy []string `json:"denied_by,omitempty"`
}

// PermissionRule represents a single permission rule
type PermissionRule struct {
	// Pattern is the glob pattern for matching commands
	Pattern string `json:"pattern"`

	// Type indicates if this is an allow or deny rule
	Type RuleType `json:"type"`

	// Description provides context for the rule
	Description string `json:"description,omitempty"`

	// Priority determines rule precedence (higher = more important)
	Priority int `json:"priority"`

	// CreatedAt when the rule was created
	CreatedAt time.Time `json:"created_at"`

	// Metadata contains additional rule data
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RuleType represents the type of permission rule
type RuleType string

const (
	// RuleTypeAllow permits matching commands
	RuleTypeAllow RuleType = "allow"

	// RuleTypeDeny blocks matching commands
	RuleTypeDeny RuleType = "deny"
)

// CommandValidator is the main validation engine
type CommandValidator struct {
	// allowRules contains permission rules that allow commands
	allowRules []PermissionRule

	// denyRules contains permission rules that deny commands
	denyRules []PermissionRule

	// cache stores validation results for performance
	cache *validationCache

	// logger for validation events
	logger *log.Logger

	// mu protects the rules slices
	mu sync.RWMutex

	// options contains configuration options
	options ValidatorOptions
}

// ValidatorOptions configures the command validator
type ValidatorOptions struct {
	// EnableCache enables result caching
	EnableCache bool

	// CacheTTL is how long cached results remain valid
	CacheTTL time.Duration

	// EnableLogging enables validation logging
	EnableLogging bool

	// LogPath is the path to the validation log file
	LogPath string

	// StrictMode requires all segments to be explicitly allowed
	StrictMode bool

	// MaxCommandLength limits the maximum command length
	MaxCommandLength int

	// NormalizeWhitespace collapses multiple whitespace characters
	NormalizeWhitespace bool

	// ValidateCompoundCommands validates each segment of compound commands
	ValidateCompoundCommands bool
}

// DefaultValidatorOptions returns sensible default options
func DefaultValidatorOptions() ValidatorOptions {
	return ValidatorOptions{
		EnableCache:              true,
		CacheTTL:                 5 * time.Minute,
		EnableLogging:            true,
		LogPath:                  "",
		StrictMode:               false,
		MaxCommandLength:         8192,
		NormalizeWhitespace:      true,
		ValidateCompoundCommands: true,
	}
}

// NewCommandValidator creates a new command validator with the given options
func NewCommandValidator(opts ValidatorOptions) (*CommandValidator, error) {
	v := &CommandValidator{
		allowRules: make([]PermissionRule, 0),
		denyRules:  make([]PermissionRule, 0),
		options:    opts,
	}

	if opts.EnableCache {
		v.cache = newValidationCache(opts.CacheTTL)
	}

	if opts.EnableLogging {
		if opts.LogPath != "" {
			file, err := os.OpenFile(opts.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to open log file: %w", err)
			}
			v.logger = log.New(file, "SECURITY: ", log.LstdFlags|log.Lmicroseconds)
		} else {
			v.logger = log.New(os.Stderr, "SECURITY: ", log.LstdFlags|log.Lmicroseconds)
		}
	}

	return v, nil
}

// AddAllowRule adds an allow rule to the validator
func (v *CommandValidator) AddAllowRule(pattern, description string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	if !isValidGlobPattern(pattern) {
		return fmt.Errorf("invalid glob pattern: %s", pattern)
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	rule := PermissionRule{
		Pattern:     pattern,
		Type:        RuleTypeAllow,
		Description: description,
		Priority:    len(v.allowRules),
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}

	v.allowRules = append(v.allowRules, rule)
	v.invalidateCache()

	v.logEvent("ADD_ALLOW_RULE", fmt.Sprintf("pattern=%s", pattern))
	return nil
}

// AddDenyRule adds a deny rule to the validator
func (v *CommandValidator) AddDenyRule(pattern, description string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	if !isValidGlobPattern(pattern) {
		return fmt.Errorf("invalid glob pattern: %s", pattern)
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	rule := PermissionRule{
		Pattern:     pattern,
		Type:        RuleTypeDeny,
		Description: description,
		Priority:    len(v.denyRules),
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}

	v.denyRules = append(v.denyRules, rule)
	v.invalidateCache()

	v.logEvent("ADD_DENY_RULE", fmt.Sprintf("pattern=%s", pattern))
	return nil
}

// RemoveRule removes a rule by pattern
func (v *CommandValidator) RemoveRule(pattern string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	removed := false

	// Check allow rules
	for i, rule := range v.allowRules {
		if rule.Pattern == pattern {
			v.allowRules = append(v.allowRules[:i], v.allowRules[i+1:]...)
			removed = true
			break
		}
	}

	// Check deny rules if not found in allow
	if !removed {
		for i, rule := range v.denyRules {
			if rule.Pattern == pattern {
				v.denyRules = append(v.denyRules[:i], v.denyRules[i+1:]...)
				removed = true
				break
			}
		}
	}

	if removed {
		v.invalidateCache()
		v.logEvent("REMOVE_RULE", fmt.Sprintf("pattern=%s", pattern))
	}

	return removed
}

// Validate checks if a command is allowed
func (v *CommandValidator) Validate(ctx context.Context, command string) ValidationResult {
	start := time.Now()
	result := ValidationResult{
		Command:            command,
		Timestamp:          time.Now(),
		ValidationDuration: 0,
	}

	// Check command length
	if v.options.MaxCommandLength > 0 && len(command) > v.options.MaxCommandLength {
		result.Allowed = false
		result.Reason = fmt.Sprintf("command exceeds maximum length of %d characters", v.options.MaxCommandLength)
		result.ValidationDuration = time.Since(start)
		v.logValidationResult(result)
		return result
	}

	// Normalize command
	normalized := v.normalizeCommand(command)
	result.NormalizedCommand = normalized

	// Check cache
	if v.options.EnableCache && v.cache != nil {
		if cached, hit := v.cache.get(normalized); hit {
			cached.Timestamp = result.Timestamp
			cached.CacheHit = true
			cached.ValidationDuration = time.Since(start)
			v.logEvent("CACHE_HIT", fmt.Sprintf("command=%s", normalized))
			return cached
		}
	}

	// Validate compound command segments if enabled
	if v.options.ValidateCompoundCommands {
		segments := v.parseCommandSegments(normalized)
		result.Segments = v.validateSegments(segments)
	}

	// Check deny rules first (deny precedes allow)
	v.mu.RLock()
	denyRules := make([]PermissionRule, len(v.denyRules))
	copy(denyRules, v.denyRules)
	allowRules := make([]PermissionRule, len(v.allowRules))
	copy(allowRules, v.allowRules)
	v.mu.RUnlock()

	// Check deny rules - any match means denied
	for _, rule := range denyRules {
		if matchGlob(rule.Pattern, normalized) {
			result.Allowed = false
			result.DeniedBy = &rule
			result.Reason = fmt.Sprintf("denied by rule: %s", rule.Description)
			if result.Reason == "denied by rule: " {
				result.Reason = fmt.Sprintf("denied by pattern: %s", rule.Pattern)
			}
			result.ValidationDuration = time.Since(start)
			v.cacheResult(result)
			v.logValidationResult(result)
			return result
		}
	}

	// Check allow rules - must match at least one
	allowed := false
	var matchedRule *PermissionRule
	for _, rule := range allowRules {
		if matchGlob(rule.Pattern, normalized) {
			allowed = true
			matchedRule = &rule
			break
		}
	}

	// If strict mode, check all segments are allowed
	if v.options.StrictMode && len(result.Segments) > 0 {
		for _, seg := range result.Segments {
			if !seg.Allowed {
				allowed = false
				result.Reason = fmt.Sprintf("segment '%s' not explicitly allowed in strict mode", seg.Segment)
				break
			}
		}
	}

	result.Allowed = allowed
	if allowed {
		result.MatchedRule = matchedRule
		result.Reason = "command allowed"
		if matchedRule != nil && matchedRule.Description != "" {
			result.Reason = fmt.Sprintf("allowed by rule: %s", matchedRule.Description)
		}
	} else {
		result.Reason = "no matching allow rule found"
	}

	result.ValidationDuration = time.Since(start)
	v.cacheResult(result)
	v.logValidationResult(result)
	return result
}

// ValidateBatch validates multiple commands efficiently
func (v *CommandValidator) ValidateBatch(ctx context.Context, commands []string) []ValidationResult {
	results := make([]ValidationResult, len(commands))
	for i, cmd := range commands {
		results[i] = v.Validate(ctx, cmd)
	}
	return results
}

// GetRules returns all configured rules
func (v *CommandValidator) GetRules() ([]PermissionRule, []PermissionRule) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	allowCopy := make([]PermissionRule, len(v.allowRules))
	denyCopy := make([]PermissionRule, len(v.denyRules))
	copy(allowCopy, v.allowRules)
	copy(denyCopy, v.denyRules)

	return allowCopy, denyCopy
}

// ClearRules removes all rules
func (v *CommandValidator) ClearRules() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.allowRules = v.allowRules[:0]
	v.denyRules = v.denyRules[:0]
	v.invalidateCache()

	v.logEvent("CLEAR_RULES", "all rules removed")
}

// ClearCache clears the validation cache
func (v *CommandValidator) ClearCache() {
	if v.cache != nil {
		v.cache.clear()
		v.logEvent("CLEAR_CACHE", "validation cache cleared")
	}
}

// normalizeCommand normalizes the command for validation
func (v *CommandValidator) normalizeCommand(command string) string {
	// Trim leading/trailing whitespace
	normalized := strings.TrimSpace(command)

	// Normalize whitespace if enabled
	if v.options.NormalizeWhitespace {
		// Replace multiple whitespace with single space
		re := regexp.MustCompile(`\s+`)
		normalized = re.ReplaceAllString(normalized, " ")
	}

	// Remove shell escaping for validation
	normalized = strings.ReplaceAll(normalized, "\\ ", " ")

	return normalized
}

// parseCommandSegments breaks a compound command into segments
func (v *CommandValidator) parseCommandSegments(command string) []string {
	// Split by common compound command operators
	operators := []string{" && ", " || ", " ; ", " | ", "`", "$("}
	segments := []string{command}

	for _, op := range operators {
		var newSegments []string
		for _, seg := range segments {
			parts := strings.Split(seg, op)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					newSegments = append(newSegments, part)
				}
			}
		}
		segments = newSegments
	}

	// Further split by the first word (command name)
	var result []string
	for _, seg := range segments {
		parts := strings.Fields(seg)
		if len(parts) > 0 {
			// Add the full segment
			result = append(result, seg)
			// Also add the base command
			result = append(result, parts[0])
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, seg := range result {
		if !seen[seg] {
			seen[seg] = true
			unique = append(unique, seg)
		}
	}

	return unique
}

// validateSegments validates individual command segments
func (v *CommandValidator) validateSegments(segments []string) []SegmentValidation {
	validations := make([]SegmentValidation, len(segments))

	v.mu.RLock()
	denyRules := make([]PermissionRule, len(v.denyRules))
	allowRules := make([]PermissionRule, len(v.allowRules))
	copy(denyRules, v.denyRules)
	copy(allowRules, v.allowRules)
	v.mu.RUnlock()

	for i, seg := range segments {
		val := SegmentValidation{
			Segment:         seg,
			Index:           i,
			Allowed:         false,
			MatchedPatterns: []string{},
			DeniedBy:        []string{},
		}

		// Check deny rules first
		for _, rule := range denyRules {
			if matchGlob(rule.Pattern, seg) {
				val.DeniedBy = append(val.DeniedBy, rule.Pattern)
				val.Allowed = false
			}
		}

		// If not denied, check allow rules
		if len(val.DeniedBy) == 0 {
			for _, rule := range allowRules {
				if matchGlob(rule.Pattern, seg) {
					val.MatchedPatterns = append(val.MatchedPatterns, rule.Pattern)
					val.Allowed = true
					break
				}
			}
		}

		validations[i] = val
	}

	return validations
}

// invalidateCache clears the cache when rules change
func (v *CommandValidator) invalidateCache() {
	if v.cache != nil {
		v.cache.clear()
	}
}

// cacheResult stores a validation result in cache
func (v *CommandValidator) cacheResult(result ValidationResult) {
	if v.options.EnableCache && v.cache != nil && !result.CacheHit {
		v.cache.set(result.NormalizedCommand, result)
	}
}

// logValidationResult logs a validation result
func (v *CommandValidator) logValidationResult(result ValidationResult) {
	if !v.options.EnableLogging || v.logger == nil {
		return
	}

	status := "ALLOWED"
	if !result.Allowed {
		status = "DENIED"
	}

	v.logger.Printf("VALIDATION: status=%s command=%q allowed=%v reason=%q duration=%v cache_hit=%v",
		status,
		result.Command,
		result.Allowed,
		result.Reason,
		result.ValidationDuration,
		result.CacheHit,
	)
}

// logEvent logs a security event
func (v *CommandValidator) logEvent(eventType, message string) {
	if !v.options.EnableLogging || v.logger == nil {
		return
	}

	v.logger.Printf("EVENT: type=%s %s", eventType, message)
}

// Close cleans up resources
func (v *CommandValidator) Close() error {
	if v.cache != nil {
		v.cache.clear()
	}
	return nil
}

// isValidGlobPattern checks if a pattern is a valid glob
func isValidGlobPattern(pattern string) bool {
	// Basic validation - check for unbalanced braces
	openCount := 0
	for _, ch := range pattern {
		switch ch {
		case '{':
			openCount++
		case '}':
			openCount--
			if openCount < 0 {
				return false
			}
		}
	}
	return openCount == 0
}

// matchGlob matches a string against a glob pattern
func matchGlob(pattern, s string) bool {
	// Handle empty cases
	if pattern == "" {
		return s == ""
	}
	if pattern == "*" {
		return true
	}

	// Convert glob pattern to regex
	regexPattern := globToRegex(pattern)

	// Match with case insensitivity for command names
	matched, err := regexp.MatchString("(?i)^"+regexPattern+"$", s)
	if err != nil {
		// Fallback to filepath.Glob-style matching for simple patterns
		matched, _ = filepath.Match(pattern, s)
		return matched
	}

	return matched
}

// globToRegex converts a glob pattern to a regex pattern
func globToRegex(pattern string) string {
	var result strings.Builder

	// Track if we're in a brace expansion
	inBrace := false
	braceContent := &strings.Builder{}

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]

		switch ch {
		case '*':
			if inBrace {
				braceContent.WriteByte(ch)
			} else {
				// Check for ** (match across path separators)
				if i+1 < len(pattern) && pattern[i+1] == '*' {
					result.WriteString(".*")
					i++ // Skip next *
				} else {
					result.WriteString("[^/]*")
				}
			}
		case '?':
			if inBrace {
				braceContent.WriteByte(ch)
			} else {
				result.WriteString(".")
			}
		case '[':
			if inBrace {
				braceContent.WriteByte(ch)
			} else {
				// Handle character class
				j := i + 1
				if j < len(pattern) && pattern[j] == '!' {
					j++
					result.WriteString("[^")
				} else {
					result.WriteString("[")
				}
				for j < len(pattern) && pattern[j] != ']' {
					result.WriteByte(pattern[j])
					j++
				}
				if j < len(pattern) {
					result.WriteByte(']')
					i = j
				}
			}
		case '{':
			if inBrace {
				braceContent.WriteByte(ch)
			} else {
				inBrace = true
				braceContent.Reset()
			}
		case '}':
			if inBrace {
				inBrace = false
				// Convert brace content to regex alternation
				options := strings.Split(braceContent.String(), ",")
				result.WriteString("(")
				for i, opt := range options {
					if i > 0 {
						result.WriteString("|")
					}
					result.WriteString(regexp.QuoteMeta(opt))
				}
				result.WriteString(")")
			} else {
				result.WriteString("\\}")
			}
		case '\\':
			// Escape next character
			if i+1 < len(pattern) {
				if inBrace {
					braceContent.WriteByte(pattern[i+1])
				} else {
					result.WriteString(regexp.QuoteMeta(string(pattern[i+1])))
				}
				i++
			}
		default:
			if inBrace {
				braceContent.WriteByte(ch)
			} else {
				result.WriteString(regexp.QuoteMeta(string(ch)))
			}
		}
	}

	return result.String()
}

// validationCache provides thread-safe caching of validation results
type validationCache struct {
	entries map[string]cacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type cacheEntry struct {
	result    ValidationResult
	timestamp time.Time
}

func newValidationCache(ttl time.Duration) *validationCache {
	cache := &validationCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
	go cache.cleanup()
	return cache
}

func (c *validationCache) get(key string) (ValidationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return ValidationResult{}, false
	}

	// Check if entry is expired
	if time.Since(entry.timestamp) > c.ttl {
		return ValidationResult{}, false
	}

	return entry.result, true
}

func (c *validationCache) set(key string, result ValidationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
}

func (c *validationCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]cacheEntry)
}

func (c *validationCache) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.timestamp) > c.ttl {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// PermissionSet represents a set of allow/deny rules for easy configuration
type PermissionSet struct {
	Allow []string `json:"allow" yaml:"allow"`
	Deny  []string `json:"deny" yaml:"deny"`
}

// LoadFromPermissionSet loads rules from a permission set
func (v *CommandValidator) LoadFromPermissionSet(set PermissionSet) error {
	for _, pattern := range set.Allow {
		if err := v.AddAllowRule(pattern, ""); err != nil {
			return fmt.Errorf("failed to add allow rule: %w", err)
		}
	}

	for _, pattern := range set.Deny {
		if err := v.AddDenyRule(pattern, ""); err != nil {
			return fmt.Errorf("failed to add deny rule: %w", err)
		}
	}

	return nil
}

// IsAllowed is a convenience method that returns only the allowed status
func (v *CommandValidator) IsAllowed(ctx context.Context, command string) bool {
	result := v.Validate(ctx, command)
	return result.Allowed
}