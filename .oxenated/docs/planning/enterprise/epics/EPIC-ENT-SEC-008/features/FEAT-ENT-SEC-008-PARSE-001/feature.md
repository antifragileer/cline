# Permission Rule Parsing

## Feature ID
FEAT-ENT-SEC-008-PARSE-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L962 - Epic 8: Security & Permissions, Feature 1 Section]

## Epic Context
**Parent Epic:** EPIC-ENT-SEC-008 - Security & Permissions [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-SEC-008/epic.md]
**Target Persona:** Enterprise User
**Epic Objective:** Enable secure, policy-compliant deployment of Cline CLI in enterprise environments through robust command permission validation, dangerous character detection, and comprehensive audit capabilities. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L925]
**Business Impact:** This feature is foundational to enterprise security governance, enabling organizations to enforce command execution policies and maintain compliance in regulated environments. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L925]

## Feature Overview
**Purpose:** Parse the `CLINE_COMMAND_PERMISSIONS` environment variable containing JSON permission rules, validate the schema, and compile glob patterns into efficient matchers for allow/deny lists. This feature transforms raw environment-based policy configuration into a structured, validated, and ready-to-use permission rule set. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L962]

**Scope:** 
- Environment variable reading and parsing
- JSON schema validation for permission structure
- Glob pattern compilation for allow/deny lists
- Redirect settings parsing (`allowRedirects` boolean)
- Error handling for malformed configurations

**PRD References:** REQ-008 - Implement command permission validation (CLINE_COMMAND_PERMISSIONS) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1465-L1480 - Traceability Matrix]
**PRD Feature ID:** EPIC-ENT-SEC-008-PARSE-001 (as defined in PRD) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L962]

**Dependencies:** 
- Proposed: `internal/security/permissions.go` - Core permission parsing logic (to be created)
- Proposed: JSON schema validation library or custom validation logic
- Proposed: Glob pattern matching library (e.g., `filepath.Match` or dedicated library like `github.com/gobwas/glob`)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L925-L950 - IAOOI section for Security & Permissions epic]

**Inputs:**
1. `CLINE_COMMAND_PERMISSIONS` environment variable containing JSON permission rules [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L925-L930]
2. JSON schema defining valid permission structure (allow: string[], deny: string[], allowRedirects: boolean)
3. Glob pattern strings from allow/deny arrays
4. Optional: Default values when environment variable is not set

**Activities:**
1. Read `CLINE_COMMAND_PERMISSIONS` from environment at CLI initialization [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L930-L935]
2. Parse JSON configuration with schema validation (validate required fields, data types)
3. Compile glob patterns into efficient matchers for allow/deny lists
4. Validate pattern syntax (ensure valid glob expressions)
5. Handle missing or empty permissions (default to allow-all behavior)
6. Cache parsed permission rules for efficient repeated validation
7. Return structured permission configuration object

**Outputs:**
1. Parsed permission rules struct with compiled glob patterns (allow list, deny list, allowRedirects) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L935-L940]
2. Validation errors for malformed permission configurations
3. Compiled glob matchers ready for command validation
4. Default permission configuration when env var is not set

**Outcomes:**
1. Commands only execute if explicitly permitted by enterprise policy (after validation by EPIC-ENT-SEC-008-VALID-002)
2. Administrators have granular control over approved command patterns
3. Failed validations provide clear feedback about configuration errors
4. Security policies are loaded once at startup and cached for performance

**Impacts:**
1. **Security Compliance**: Foundation for meeting enterprise security governance and regulatory requirements
2. **Risk Reduction**: Enables the deny-first security model that prevents accidental or malicious execution
3. **Enterprise Adoption**: Required capability for deployment in regulated industries (finance, healthcare, government)
4. **Operational Safety**: Supports safe automation by ensuring only approved commands can execute

## Technical Requirements
**Architecture Layer:** Infrastructure/Security Layer

**Integration Points:** 
- Proposed: `internal/security/permissions.go` - Main parsing implementation (to be created)
- Proposed: `internal/security/validator.go` - Consumes parsed permissions for command validation (related to EPIC-ENT-SEC-008-VALID-002)
- Proposed: `cmd/cline/main.go` or initialization code - Triggers permission loading at startup
- Proposed: Environment variable access via `os.Getenv()` or similar

**Data Requirements:** 
- Proposed: PermissionRules struct to be defined:
  ```go
  type PermissionRules struct {
      Allow          []string // Raw glob patterns
      Deny           []string // Raw glob patterns
      AllowRedirects bool
      // Compiled matchers for performance
      compiledAllow []glob.Glob
      compiledDeny  []glob.Glob
  }
  ```

**Performance Requirements:** 
- Parse and compile permissions once at CLI startup (not per-command)
- Sub-millisecond parsing overhead for typical configurations (< 100 patterns)
- Memory efficient compiled glob matchers
- Lazy compilation - only compile when permissions are first needed

**Security Requirements:** 
- Treat permission patterns as globs, not regex, to prevent ReDoS attacks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1000-L1005 - Security Architecture Notes]
- Read permissions from environment variable only (read-only, not from mutable config files)
- Validate all patterns before compilation to prevent injection
- Clear error messages for malformed configs without exposing sensitive data

## User Experience
**User Personas:** Enterprise Administrator, Security Team

**User Actions:** 
1. Administrator sets `CLINE_COMMAND_PERMISSIONS` environment variable with JSON policy
2. CLI parses permissions at startup (transparent to end users)
3. If configuration is invalid, CLI shows clear error and exits
4. Parsed permissions are used by validation engine (EPIC-ENT-SEC-008-VALID-002)

**CLI Components:** 
- Proposed: New `internal/security/` package to be created
- Proposed: Permission parsing service with methods:
  - `ParsePermissions() (*PermissionRules, error)`
  - `ValidateSchema(jsonData []byte) error`
  - `CompilePatterns(patterns []string) ([]glob.Glob, error)`

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L962-L987 - BDD Scenarios for Permission Rule Parsing]

```gherkin
Scenario: Parse permission configuration
  Given CLINE_COMMAND_PERMISSIONS='{"allow":["npm *","git *"],"deny":["rm -rf *"],"allowRedirects":false}'
  When the CLI initializes
  Then allow list should contain "npm *" and "git *"
  And deny list should contain "rm -rf *"
  And allowRedirects should be false

Scenario: Handle missing permissions
  Given CLINE_COMMAND_PERMISSIONS is not set
  When the CLI runs
  Then all commands should be allowed
  And no restrictions should apply

Scenario: Parse complex patterns
  Given patterns with wildcards "npm run *" and "git push origin *"
  When permissions parse
  Then glob patterns should compile correctly
  For matching during validation
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1006-L1010 - Success Metrics]

**Functional:**
- Feature correctly parses valid JSON permission configurations
- All BDD scenarios pass for permission parsing
- Handles missing environment variable gracefully (allow-all mode)
- Rejects malformed JSON with clear error messages
- Compiles glob patterns without errors for valid patterns

**Performance:**
- Parse and compile permissions in under 10ms for configurations with up to 100 patterns
- Memory usage remains under 1MB for typical permission sets

**Quality:**
- No panics or crashes on invalid input
- Clear, actionable error messages for configuration errors
- 100% unit test coverage for parsing logic

**Integration:**
- Parsed permissions are consumable by EPIC-ENT-SEC-008-VALID-002 (Command Validation Engine)
- Compatible with EPIC-ENT-AUDIT-010 for logging permission loading events

## Testing Strategy
**Unit Testing:**
- Test JSON parsing with valid and invalid inputs
- Test glob pattern compilation with various pattern types (`*`, `?`, `[...]`)
- Test schema validation (missing fields, wrong types)
- Test error handling for malformed JSON
- Test default behavior when env var is not set

**Integration Testing:**
- Test integration with command validation engine
- Test end-to-end: set env var → parse → validate command
- Test with realistic enterprise permission configurations

**User Acceptance:**
- Verify BDD scenarios pass
- Verify error messages are clear and helpful
- Verify performance meets requirements

## Tasks Overview
1. **TASK-PARSE-001-001**: Define PermissionRules struct and JSON schema
2. **TASK-PARSE-001-002**: Implement environment variable reading
3. **TASK-PARSE-001-003**: Implement JSON parsing with validation
4. **TASK-PARSE-001-004**: Implement glob pattern compilation
5. **TASK-PARSE-001-005**: Implement error handling and logging
6. **TASK-PARSE-001-006**: Write unit tests for parsing logic
7. **TASK-PARSE-001-007**: Write integration tests with validation engine

## Implementation Notes
- Use a dedicated glob library like `github.com/gobwas/glob` for efficient pattern matching
- Consider using `encoding/json` with custom unmarshaling for validation
- Cache compiled patterns to avoid recompilation during command validation
- Log permission loading events for audit purposes (integrate with EPIC-ENT-AUDIT-010)
- Ensure thread-safety if permissions may be accessed concurrently
- Document the JSON schema clearly for enterprise administrators

## Related Features
- **EPIC-ENT-SEC-008-VALID-002**: Command Validation Engine - Consumes parsed permissions
- **EPIC-ENT-SEC-008-DANGER-003**: Dangerous Character Detection - Parallel security feature
- **EPIC-AUTO-EXEC-006-YOLO-001**: Yolo Mode - Permission validation must execute before auto-approvals

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L###]
- [x] Existing code references cite actual file paths and lines (N/A - this is new functionality)
- [x] New functionality clearly marked as "Proposed:" in Technical Requirements and Integration Points
- [x] Integration points cite existing interfaces or mark as new
- [x] Citation Verification checklist completed in feature.md