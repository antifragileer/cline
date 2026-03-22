// Package config provides policy enforcement functionality for the Cline CLI.
// It implements enterprise policy controls including allowed/denied model lists,
// API key age limits, usage quotas, and policy inheritance with hot-reload support.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/fsnotify/fsnotify"
)

// PolicyLevel defines the hierarchy level of a policy
type PolicyLevel int

const (
	// PolicyLevelSystem represents system-wide policies (highest priority)
	PolicyLevelSystem PolicyLevel = iota
	// PolicyLevelOrganization represents organization-level policies
	PolicyLevelOrganization
	// PolicyLevelTeam represents team-level policies
	PolicyLevelTeam
	// PolicyLevelUser represents user-level policies (lowest priority)
	PolicyLevelUser
)

// String returns a human-readable name for the policy level
func (p PolicyLevel) String() string {
	switch p {
	case PolicyLevelSystem:
		return "system"
	case PolicyLevelOrganization:
		return "organization"
	case PolicyLevelTeam:
		return "team"
	case PolicyLevelUser:
		return "user"
	default:
		return "unknown"
	}
}

// PolicyViolation represents a detected policy violation
type PolicyViolation struct {
	// Type is the category of violation
	Type ViolationType `json:"type"`
	// Message describes the violation
	Message string `json:"message"`
	// Level is the policy level that triggered the violation
	Level PolicyLevel `json:"level"`
	// Timestamp when the violation was detected
	Timestamp time.Time `json:"timestamp"`
	// PolicyName identifies which policy was violated
	PolicyName string `json:"policy_name"`
	// Resource identifies the resource involved (model, key, etc.)
	Resource string `json:"resource,omitempty"`
	// Details contains additional violation-specific information
	Details map[string]interface{} `json:"details,omitempty"`
}

// ViolationType categorizes policy violations
type ViolationType string

const (
	// ViolationTypeDeniedModel indicates a model is in the denied list
	ViolationTypeDeniedModel ViolationType = "denied_model"
	// ViolationTypeNotAllowedModel indicates a model is not in the allowed list
	ViolationTypeNotAllowedModel ViolationType = "not_allowed_model"
	// ViolationTypeKeyExpired indicates an API key has exceeded its age limit
	ViolationTypeKeyExpired ViolationType = "key_expired"
	// ViolationTypeQuotaExceeded indicates a usage quota has been exceeded
	ViolationTypeQuotaExceeded ViolationType = "quota_exceeded"
	// ViolationTypeInvalidConfiguration indicates a configuration violates policy
	ViolationTypeInvalidConfiguration ViolationType = "invalid_configuration"
)

// Policy defines enterprise policy constraints
type Policy struct {
	// Name identifies the policy
	Name string `json:"name"`
	// Level indicates the policy hierarchy level
	Level PolicyLevel `json:"level"`
	// Description provides human-readable context
	Description string `json:"description,omitempty"`
	// Enabled indicates if the policy is active
	Enabled bool `json:"enabled"`
	// CreatedAt is when the policy was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the policy was last modified
	UpdatedAt time.Time `json:"updated_at"`
	// InheritsFrom specifies parent policies to inherit
	InheritsFrom []string `json:"inherits_from,omitempty"`

	// ModelPolicy controls model access
	ModelPolicy *ModelPolicy `json:"model_policy,omitempty"`
	// KeyPolicy controls API key constraints
	KeyPolicy *KeyPolicy `json:"key_policy,omitempty"`
	// QuotaPolicy controls usage limits
	QuotaPolicy *QuotaPolicy `json:"quota_policy,omitempty"`
}

// ModelPolicy defines model access controls
type ModelPolicy struct {
	// AllowedModels is a list of permitted models (empty = all allowed)
	AllowedModels []string `json:"allowed_models,omitempty"`
	// DeniedModels is a list of explicitly prohibited models
	DeniedModels []string `json:"denied_models,omitempty"`
	// AllowByDefault determines behavior when AllowedModels is empty
	AllowByDefault bool `json:"allow_by_default"`
}

// KeyPolicy defines API key management constraints
type KeyPolicy struct {
	// MaxKeyAge is the maximum age of an API key before rotation is required
	MaxKeyAge time.Duration `json:"max_key_age,omitempty"`
	// RequireRotation indicates if key rotation is mandatory
	RequireRotation bool `json:"require_rotation"`
	// AllowedProviders restricts which providers can be used
	AllowedProviders []string `json:"allowed_providers,omitempty"`
	// DeniedProviders explicitly prohibits specific providers
	DeniedProviders []string `json:"denied_providers,omitempty"`
}

// QuotaPolicy defines usage limits
type QuotaPolicy struct {
	// MaxRequestsPerDay limits daily API requests
	MaxRequestsPerDay int `json:"max_requests_per_day,omitempty"`
	// MaxTokensPerDay limits daily token consumption
	MaxTokensPerDay int `json:"max_tokens_per_day,omitempty"`
	// MaxCostPerDay limits daily spending
	MaxCostPerDay float64 `json:"max_cost_per_day,omitempty"`
	// MaxConcurrentTasks limits parallel task execution
	MaxConcurrentTasks int `json:"max_concurrent_tasks,omitempty"`
}

// PolicyEnforcer manages policy enforcement and hot-reload
type PolicyEnforcer struct {
	// policies holds all loaded policies by name
	policies map[string]*Policy
	// policiesMu protects the policies map
	policiesMu sync.RWMutex
	// violations records detected violations
	violations []PolicyViolation
	// violationsMu protects the violations slice
	violationsMu sync.RWMutex
	// storage provides access to persistent state
	storage *storage.StorageContext
	// policyDir is the directory containing policy files
	policyDir string
	// watcher monitors policy files for changes
	watcher *fsnotify.Watcher
	// stopWatcher signals the watcher to stop
	stopWatcher chan struct{}
	// logger for policy events
	logger *log.Logger
	// enforcementEnabled controls whether policies are enforced
	enforcementEnabled bool
	// onViolation is called when a violation is detected
	onViolation func(violation PolicyViolation)
}

// PolicyEnforcerOptions provides configuration for creating a PolicyEnforcer
type PolicyEnforcerOptions struct {
	// PolicyDir is the directory containing policy JSON files
	PolicyDir string
	// Storage provides access to persistent state
	Storage *storage.StorageContext
	// Logger for policy events (defaults to stdout)
	Logger *log.Logger
	// EnforcementEnabled controls whether violations prevent actions
	EnforcementEnabled bool
	// OnViolation is called when a violation is detected
	OnViolation func(violation PolicyViolation)
}

// NewPolicyEnforcer creates a new policy enforcer with the given options
func NewPolicyEnforcer(opts PolicyEnforcerOptions) (*PolicyEnforcer, error) {
	if opts.PolicyDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.PolicyDir = filepath.Join(homeDir, ".cline", "policies")
	}

	if opts.Logger == nil {
		opts.Logger = log.New(os.Stdout, "[POLICY] ", log.LstdFlags)
	}

	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(opts.PolicyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create policy directory: %w", err)
	}

	enforcer := &PolicyEnforcer{
		policies:           make(map[string]*Policy),
		violations:         make([]PolicyViolation, 0),
		storage:            opts.Storage,
		policyDir:          opts.PolicyDir,
		logger:             opts.Logger,
		enforcementEnabled: opts.EnforcementEnabled,
		onViolation:        opts.OnViolation,
		stopWatcher:        make(chan struct{}),
	}

	// Load initial policies
	if err := enforcer.LoadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	// Start file watcher for hot-reload
	if err := enforcer.startWatcher(); err != nil {
		enforcer.logger.Printf("Warning: failed to start policy watcher: %v", err)
	}

	return enforcer, nil
}

// LoadPolicies loads all policy files from the policy directory
func (pe *PolicyEnforcer) LoadPolicies() error {
	pe.policiesMu.Lock()
	defer pe.policiesMu.Unlock()

	// Clear existing policies
	pe.policies = make(map[string]*Policy)

	entries, err := os.ReadDir(pe.policyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process JSON files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(pe.policyDir, entry.Name())
		policy, err := pe.loadPolicyFile(path)
		if err != nil {
			pe.logger.Printf("Warning: failed to load policy from %s: %v", path, err)
			continue
		}

		if policy.Enabled {
			pe.policies[policy.Name] = policy
			pe.logger.Printf("Loaded policy: %s (level: %s)", policy.Name, policy.Level.String())
		}
	}

	return nil
}

// loadPolicyFile loads a single policy file
func (pe *PolicyEnforcer) loadPolicyFile(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
	}

	// Set default timestamps if not provided
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}
	policy.UpdatedAt = time.Now()

	// Validate policy
	if policy.Name == "" {
		return nil, fmt.Errorf("policy name is required")
	}

	return &policy, nil
}

// startWatcher initializes the file system watcher for hot-reload
func (pe *PolicyEnforcer) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	pe.watcher = watcher

	// Watch the policy directory
	if err := watcher.Add(pe.policyDir); err != nil {
		return fmt.Errorf("failed to watch policy directory: %w", err)
	}

	go pe.watchLoop()

	return nil
}

// watchLoop monitors for file changes and reloads policies
func (pe *PolicyEnforcer) watchLoop() {
	for {
		select {
		case event, ok := <-pe.watcher.Events:
			if !ok {
				return
			}

			// Handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if filepath.Ext(event.Name) == ".json" {
					pe.logger.Printf("Policy file changed: %s", event.Name)
					if err := pe.LoadPolicies(); err != nil {
						pe.logger.Printf("Error reloading policies: %v", err)
					}
				}
			}

			// Handle delete events
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				pe.logger.Printf("Policy file removed: %s", event.Name)
				if err := pe.LoadPolicies(); err != nil {
					pe.logger.Printf("Error reloading policies: %v", err)
				}
			}

		case err, ok := <-pe.watcher.Errors:
			if !ok {
				return
			}
			pe.logger.Printf("Policy watcher error: %v", err)

		case <-pe.stopWatcher:
			return
		}
	}
}

// Stop shuts down the policy enforcer and releases resources
func (pe *PolicyEnforcer) Stop() error {
	close(pe.stopWatcher)

	if pe.watcher != nil {
		if err := pe.watcher.Close(); err != nil {
			return fmt.Errorf("failed to close watcher: %w", err)
		}
	}

	return nil
}

// GetPolicy returns a policy by name
func (pe *PolicyEnforcer) GetPolicy(name string) (*Policy, bool) {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	policy, ok := pe.policies[name]
	return policy, ok
}

// ListPolicies returns all loaded policy names
func (pe *PolicyEnforcer) ListPolicies() []string {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	names := make([]string, 0, len(pe.policies))
	for name := range pe.policies {
		names = append(names, name)
	}
	return names
}

// GetEffectivePolicy computes the merged policy for a given level and inheritance chain
func (pe *PolicyEnforcer) GetEffectivePolicy(startingPolicy string) (*Policy, error) {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	startPolicy, ok := pe.policies[startingPolicy]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", startingPolicy)
	}

	// Start with the base policy
	effective := pe.clonePolicy(startPolicy)

	// Apply inheritance if specified
	visited := make(map[string]bool)
	visited[startingPolicy] = true

	for _, parentName := range startPolicy.InheritsFrom {
		if err := pe.applyInheritance(effective, parentName, visited); err != nil {
			return nil, err
		}
	}

	return effective, nil
}

// applyInheritance recursively applies parent policies
func (pe *PolicyEnforcer) applyInheritance(target *Policy, parentName string, visited map[string]bool) error {
	if visited[parentName] {
		return fmt.Errorf("circular inheritance detected: %s", parentName)
	}
	visited[parentName] = true

	parent, ok := pe.policies[parentName]
	if !ok {
		return fmt.Errorf("parent policy not found: %s", parentName)
	}

	// Merge parent policies with lower priority (parent values only apply if not set in child)
	pe.mergePolicy(target, parent, false)

	// Recursively apply grandparent policies
	for _, grandparentName := range parent.InheritsFrom {
		if err := pe.applyInheritance(target, grandparentName, visited); err != nil {
			return err
		}
	}

	return nil
}

// mergePolicy combines two policies, with control over which takes precedence
func (pe *PolicyEnforcer) mergePolicy(target, source *Policy, sourceWins bool) {
	// Merge model policies
	if source.ModelPolicy != nil {
		if target.ModelPolicy == nil {
			target.ModelPolicy = &ModelPolicy{}
		}
		pe.mergeModelPolicy(target.ModelPolicy, source.ModelPolicy, sourceWins)
	}

	// Merge key policies
	if source.KeyPolicy != nil {
		if target.KeyPolicy == nil {
			target.KeyPolicy = &KeyPolicy{}
		}
		pe.mergeKeyPolicy(target.KeyPolicy, source.KeyPolicy, sourceWins)
	}

	// Merge quota policies
	if source.QuotaPolicy != nil {
		if target.QuotaPolicy == nil {
			target.QuotaPolicy = &QuotaPolicy{}
		}
		pe.mergeQuotaPolicy(target.QuotaPolicy, source.QuotaPolicy, sourceWins)
	}
}

// mergeModelPolicy combines model policies
func (pe *PolicyEnforcer) mergeModelPolicy(target, source *ModelPolicy, sourceWins bool) {
	// Always merge lists - denied models from both parent and child should apply
	target.AllowedModels = mergeStringSlices(target.AllowedModels, source.AllowedModels, sourceWins)
	target.DeniedModels = mergeStringSlices(target.DeniedModels, source.DeniedModels, sourceWins)
	if sourceWins {
		target.AllowByDefault = source.AllowByDefault
	}
}

// mergeKeyPolicy combines key policies
func (pe *PolicyEnforcer) mergeKeyPolicy(target, source *KeyPolicy, sourceWins bool) {
	if sourceWins || target.MaxKeyAge == 0 {
		target.MaxKeyAge = source.MaxKeyAge
	}
	// For booleans, we need to check if the target has been explicitly set
	// Since we can't distinguish between false and unset, we always merge when sourceWins
	// or when target is false and source is true
	if sourceWins || (!target.RequireRotation && source.RequireRotation) {
		target.RequireRotation = source.RequireRotation
	}
	// Always merge provider lists - denied providers from both should apply
	target.AllowedProviders = mergeStringSlices(target.AllowedProviders, source.AllowedProviders, sourceWins)
	target.DeniedProviders = mergeStringSlices(target.DeniedProviders, source.DeniedProviders, sourceWins)
}

// mergeQuotaPolicy combines quota policies
func (pe *PolicyEnforcer) mergeQuotaPolicy(target, source *QuotaPolicy, sourceWins bool) {
	if sourceWins || target.MaxRequestsPerDay == 0 {
		target.MaxRequestsPerDay = source.MaxRequestsPerDay
	}
	if sourceWins || target.MaxTokensPerDay == 0 {
		target.MaxTokensPerDay = source.MaxTokensPerDay
	}
	if sourceWins || target.MaxCostPerDay == 0 {
		target.MaxCostPerDay = source.MaxCostPerDay
	}
	if sourceWins || target.MaxConcurrentTasks == 0 {
		target.MaxConcurrentTasks = source.MaxConcurrentTasks
	}
}

// mergeStringSlices combines two string slices with conflict resolution
func mergeStringSlices(target, source []string, sourceWins bool) []string {
	if sourceWins {
		return source
	}
	// If target wins, keep target values and append unique source values
	seen := make(map[string]bool)
	for _, s := range target {
		seen[s] = true
	}
	result := append([]string{}, target...)
	for _, s := range source {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

// ValidateModel checks if a model is allowed by policy
func (pe *PolicyEnforcer) ValidateModel(model string) *PolicyViolation {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	for _, policy := range pe.policies {
		if !policy.Enabled || policy.ModelPolicy == nil {
			continue
		}

		modelPolicy := policy.ModelPolicy

		// Check denied list first (explicit denials take precedence)
		for _, denied := range modelPolicy.DeniedModels {
			if denied == model || pe.matchesWildcard(model, denied) {
				violation := PolicyViolation{
					Type:       ViolationTypeDeniedModel,
					Message:    fmt.Sprintf("Model '%s' is in the denied list", model),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   model,
				}
				pe.recordViolation(violation)
				return &violation
			}
		}

		// Check allowed list
		if len(modelPolicy.AllowedModels) > 0 {
			allowed := false
			for _, allowedModel := range modelPolicy.AllowedModels {
				if allowedModel == model || pe.matchesWildcard(model, allowedModel) {
					allowed = true
					break
				}
			}
			if !allowed {
				violation := PolicyViolation{
					Type:       ViolationTypeNotAllowedModel,
					Message:    fmt.Sprintf("Model '%s' is not in the allowed list", model),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   model,
				}
				pe.recordViolation(violation)
				return &violation
			}
		} else if !modelPolicy.AllowByDefault {
			// Empty allowed list with AllowByDefault=false means deny all
			violation := PolicyViolation{
				Type:       ViolationTypeNotAllowedModel,
				Message:    fmt.Sprintf("Model '%s' is not explicitly allowed", model),
				Level:      policy.Level,
				Timestamp:  time.Now(),
				PolicyName: policy.Name,
				Resource:   model,
			}
			pe.recordViolation(violation)
			return &violation
		}
	}

	return nil
}

// ValidateKeyAge checks if an API key's age is within policy limits
func (pe *PolicyEnforcer) ValidateKeyAge(provider string, keyCreatedAt time.Time) *PolicyViolation {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	for _, policy := range pe.policies {
		if !policy.Enabled || policy.KeyPolicy == nil {
			continue
		}

		keyPolicy := policy.KeyPolicy

		// Check if provider is allowed
		if len(keyPolicy.AllowedProviders) > 0 {
			allowed := false
			for _, allowedProvider := range keyPolicy.AllowedProviders {
				if allowedProvider == provider {
					allowed = true
					break
				}
			}
			if !allowed {
				continue // Skip this policy for non-allowed providers
			}
		}

		// Check denied providers
		for _, deniedProvider := range keyPolicy.DeniedProviders {
			if deniedProvider == provider {
				violation := PolicyViolation{
					Type:       ViolationTypeInvalidConfiguration,
					Message:    fmt.Sprintf("Provider '%s' is denied by policy", provider),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   provider,
				}
				pe.recordViolation(violation)
				return &violation
			}
		}

		// Check key age
		if keyPolicy.MaxKeyAge > 0 {
			age := time.Since(keyCreatedAt)
			if age > keyPolicy.MaxKeyAge {
				violation := PolicyViolation{
					Type:       ViolationTypeKeyExpired,
					Message:    fmt.Sprintf("API key for '%s' has exceeded maximum age of %v", provider, keyPolicy.MaxKeyAge),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   provider,
					Details: map[string]interface{}{
						"key_age":          age.String(),
						"max_age":          keyPolicy.MaxKeyAge.String(),
						"created_at":       keyCreatedAt.Format(time.RFC3339),
						"require_rotation": keyPolicy.RequireRotation,
					},
				}
				pe.recordViolation(violation)
				return &violation
			}
		}
	}

	return nil
}

// CheckQuota verifies if an operation would exceed quota limits
func (pe *PolicyEnforcer) CheckQuota(usage UsageMetrics) *PolicyViolation {
	pe.policiesMu.RLock()
	defer pe.policiesMu.RUnlock()

	for _, policy := range pe.policies {
		if !policy.Enabled || policy.QuotaPolicy == nil {
			continue
		}

		quotaPolicy := policy.QuotaPolicy

		// Check requests per day
		if quotaPolicy.MaxRequestsPerDay > 0 && usage.RequestsToday >= quotaPolicy.MaxRequestsPerDay {
			violation := PolicyViolation{
				Type:       ViolationTypeQuotaExceeded,
				Message:    fmt.Sprintf("Daily request quota exceeded: %d of %d", usage.RequestsToday, quotaPolicy.MaxRequestsPerDay),
				Level:      policy.Level,
				Timestamp:  time.Now(),
				PolicyName: policy.Name,
				Details: map[string]interface{}{
					"quota_type": "requests_per_day",
					"limit":      quotaPolicy.MaxRequestsPerDay,
					"used":       usage.RequestsToday,
				},
			}
			pe.recordViolation(violation)
			return &violation
		}

		// Check tokens per day
		if quotaPolicy.MaxTokensPerDay > 0 && usage.TokensToday >= quotaPolicy.MaxTokensPerDay {
			violation := PolicyViolation{
				Type:       ViolationTypeQuotaExceeded,
				Message:    fmt.Sprintf("Daily token quota exceeded: %d of %d", usage.TokensToday, quotaPolicy.MaxTokensPerDay),
				Level:      policy.Level,
				Timestamp:  time.Now(),
				PolicyName: policy.Name,
				Details: map[string]interface{}{
					"quota_type": "tokens_per_day",
					"limit":      quotaPolicy.MaxTokensPerDay,
					"used":       usage.TokensToday,
				},
			}
			pe.recordViolation(violation)
			return &violation
		}

		// Check cost per day
		if quotaPolicy.MaxCostPerDay > 0 && usage.CostToday >= quotaPolicy.MaxCostPerDay {
			violation := PolicyViolation{
				Type:       ViolationTypeQuotaExceeded,
				Message:    fmt.Sprintf("Daily cost quota exceeded: $%.2f of $%.2f", usage.CostToday, quotaPolicy.MaxCostPerDay),
				Level:      policy.Level,
				Timestamp:  time.Now(),
				PolicyName: policy.Name,
				Details: map[string]interface{}{
					"quota_type": "cost_per_day",
					"limit":      quotaPolicy.MaxCostPerDay,
					"used":       usage.CostToday,
				},
			}
			pe.recordViolation(violation)
			return &violation
		}

		// Check concurrent tasks
		if quotaPolicy.MaxConcurrentTasks > 0 && usage.ConcurrentTasks >= quotaPolicy.MaxConcurrentTasks {
			violation := PolicyViolation{
				Type:       ViolationTypeQuotaExceeded,
				Message:    fmt.Sprintf("Concurrent task limit reached: %d of %d", usage.ConcurrentTasks, quotaPolicy.MaxConcurrentTasks),
				Level:      policy.Level,
				Timestamp:  time.Now(),
				PolicyName: policy.Name,
				Details: map[string]interface{}{
					"quota_type": "concurrent_tasks",
					"limit":      quotaPolicy.MaxConcurrentTasks,
					"used":       usage.ConcurrentTasks,
				},
			}
			pe.recordViolation(violation)
			return &violation
		}
	}

	return nil
}

// UsageMetrics tracks current usage for quota validation
type UsageMetrics struct {
	RequestsToday   int
	TokensToday     int
	CostToday       float64
	ConcurrentTasks int
}

// ValidateConfiguration checks if a configuration violates any policies
func (pe *PolicyEnforcer) ValidateConfiguration(config *LayeredConfig) []PolicyViolation {
	violations := make([]PolicyViolation, 0)

	// Validate model setting
	model := config.GetString("model")
	if violation := pe.ValidateModel(model); violation != nil {
		violations = append(violations, *violation)
	}

	// Validate provider
	provider := config.GetString("api.provider")
	pe.policiesMu.RLock()
	for _, policy := range pe.policies {
		if !policy.Enabled || policy.KeyPolicy == nil {
			continue
		}

		// Check denied providers
		for _, deniedProvider := range policy.KeyPolicy.DeniedProviders {
			if deniedProvider == provider {
				violations = append(violations, PolicyViolation{
					Type:       ViolationTypeInvalidConfiguration,
					Message:    fmt.Sprintf("Provider '%s' is denied by policy", provider),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   provider,
				})
			}
		}

		// Check allowed providers
		if len(policy.KeyPolicy.AllowedProviders) > 0 {
			allowed := false
			for _, allowedProvider := range policy.KeyPolicy.AllowedProviders {
				if allowedProvider == provider {
					allowed = true
					break
				}
			}
			if !allowed {
				violations = append(violations, PolicyViolation{
					Type:       ViolationTypeInvalidConfiguration,
					Message:    fmt.Sprintf("Provider '%s' is not in the allowed providers list", provider),
					Level:      policy.Level,
					Timestamp:  time.Now(),
					PolicyName: policy.Name,
					Resource:   provider,
				})
			}
		}
	}
	pe.policiesMu.RUnlock()

	return violations
}

// Enforce determines if an operation should be allowed based on policy violations
func (pe *PolicyEnforcer) Enforce(violation *PolicyViolation) error {
	if violation == nil {
		return nil
	}

	pe.logger.Printf("Policy violation: [%s] %s (level: %s, policy: %s)",
		violation.Type, violation.Message, violation.Level.String(), violation.PolicyName)

	if pe.enforcementEnabled {
		return fmt.Errorf("policy enforcement blocked operation: %s", violation.Message)
	}

	return nil
}

// recordViolation logs a policy violation
func (pe *PolicyEnforcer) recordViolation(violation PolicyViolation) {
	pe.violationsMu.Lock()
	defer pe.violationsMu.Unlock()

	pe.violations = append(pe.violations, violation)

	// Log the violation
	pe.logger.Printf("POLICY VIOLATION: [%s] %s (level: %s, policy: %s)",
		violation.Type, violation.Message, violation.Level.String(), violation.PolicyName)

	// Call the onViolation callback if set
	if pe.onViolation != nil {
		pe.onViolation(violation)
	}

	// Persist violation to storage if available
	if pe.storage != nil && pe.storage.GlobalState != nil {
		pe.persistViolation(violation)
	}
}

// persistViolation saves a violation to persistent storage
func (pe *PolicyEnforcer) persistViolation(violation PolicyViolation) {
	// Load existing violations
	var storedViolations []PolicyViolation
	if data, ok := pe.storage.GlobalState.Get("policy_violations"); ok {
		if jsonData, err := json.Marshal(data); err == nil {
			_ = json.Unmarshal(jsonData, &storedViolations)
		}
	}

	// Append new violation
	storedViolations = append(storedViolations, violation)

	// Keep only recent violations (last 1000)
	if len(storedViolations) > 1000 {
		storedViolations = storedViolations[len(storedViolations)-1000:]
	}

	// Save back to storage
	_ = pe.storage.GlobalState.Set("policy_violations", storedViolations)
}

// GetViolations returns all recorded violations
func (pe *PolicyEnforcer) GetViolations() []PolicyViolation {
	pe.violationsMu.RLock()
	defer pe.violationsMu.RUnlock()

	// Return a copy
	result := make([]PolicyViolation, len(pe.violations))
	copy(result, pe.violations)
	return result
}

// GetViolationsSince returns violations after a specific time
func (pe *PolicyEnforcer) GetViolationsSince(since time.Time) []PolicyViolation {
	pe.violationsMu.RLock()
	defer pe.violationsMu.RUnlock()

	var result []PolicyViolation
	for _, v := range pe.violations {
		if v.Timestamp.After(since) {
			result = append(result, v)
		}
	}
	return result
}

// ClearViolations clears the in-memory violation history
func (pe *PolicyEnforcer) ClearViolations() {
	pe.violationsMu.Lock()
	defer pe.violationsMu.Unlock()

	pe.violations = make([]PolicyViolation, 0)
}

// matchesWildcard checks if a string matches a pattern with wildcards
func (pe *PolicyEnforcer) matchesWildcard(value, pattern string) bool {
	// Handle exact match
	if pattern == value {
		return true
	}

	// Handle wildcard patterns
	if pattern == "*" {
		return true
	}

	// Simple prefix/suffix matching for patterns with *
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(value, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(value, suffix)
	}

	if strings.Contains(pattern, "*") {
		// Split pattern by * and check if all parts match in order
		parts := strings.Split(pattern, "*")
		currentPos := 0
		for i, part := range parts {
			if part == "" {
				continue // Skip empty parts from consecutive * or leading/trailing *
			}

			if i == 0 {
				// First part must be at the beginning
				if !strings.HasPrefix(value, part) {
					return false
				}
				currentPos = len(part)
			} else if i == len(parts)-1 {
				// Last part must be at the end
				if !strings.HasSuffix(value[currentPos:], part) {
					return false
				}
			} else {
				// Middle parts can be anywhere after current position
				idx := strings.Index(value[currentPos:], part)
				if idx == -1 {
					return false
				}
				currentPos += idx + len(part)
			}
		}
		return true
	}

	return false
}

// clonePolicy creates a deep copy of a policy
func (pe *PolicyEnforcer) clonePolicy(policy *Policy) *Policy {
	// Serialize and deserialize to create a deep copy
	data, _ := json.Marshal(policy)
	var cloned Policy
	_ = json.Unmarshal(data, &cloned)
	return &cloned
}

// SavePolicy saves a policy to a file
func (pe *PolicyEnforcer) SavePolicy(policy *Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	policy.UpdatedAt = time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}

	filename := policy.Name + ".json"
	path := filepath.Join(pe.policyDir, filename)

	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	// Update in-memory cache
	pe.policiesMu.Lock()
	pe.policies[policy.Name] = policy
	pe.policiesMu.Unlock()

	pe.logger.Printf("Saved policy: %s", policy.Name)
	return nil
}

// DeletePolicy removes a policy file and from memory
func (pe *PolicyEnforcer) DeletePolicy(name string) error {
	filename := name + ".json"
	path := filepath.Join(pe.policyDir, filename)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete policy file: %w", err)
	}

	pe.policiesMu.Lock()
	delete(pe.policies, name)
	pe.policiesMu.Unlock()

	pe.logger.Printf("Deleted policy: %s", name)
	return nil
}

// IsEnforcementEnabled returns whether policy enforcement is active
func (pe *PolicyEnforcer) IsEnforcementEnabled() bool {
	return pe.enforcementEnabled
}

// SetEnforcementEnabled controls whether violations block operations
func (pe *PolicyEnforcer) SetEnforcementEnabled(enabled bool) {
	pe.enforcementEnabled = enabled
	pe.logger.Printf("Policy enforcement %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// GetPolicyDir returns the policy directory path
func (pe *PolicyEnforcer) GetPolicyDir() string {
	return pe.policyDir
}

// Reload forces a reload of all policies from disk
func (pe *PolicyEnforcer) Reload() error {
	return pe.LoadPolicies()
}

// Equals checks if two policies are equal
func (p *Policy) Equals(other *Policy) bool {
	if p == nil || other == nil {
		return p == other
	}
	return p.Name == other.Name &&
		p.Level == other.Level &&
		p.Enabled == other.Enabled &&
		reflect.DeepEqual(p.ModelPolicy, other.ModelPolicy) &&
		reflect.DeepEqual(p.KeyPolicy, other.KeyPolicy) &&
		reflect.DeepEqual(p.QuotaPolicy, other.QuotaPolicy)
}
