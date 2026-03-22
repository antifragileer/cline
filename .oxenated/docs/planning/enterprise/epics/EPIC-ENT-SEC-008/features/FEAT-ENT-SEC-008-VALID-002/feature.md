# Command Validation Engine

## Feature ID
FEAT-ENT-SEC-008-VALID-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L1000 - Epic 8: Security & Permissions Section, Feature 2: Command Validation Engine]

## Epic Context
**Parent Epic:** EPIC-ENT-SEC-008 - Security & Permissions [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-SEC-008/epic.md:L1]
**Target Persona:** Enterprise User
**Epic Objective:** Deliver enterprise-grade security controls for the Cline CLI, enabling organizations to enforce command execution policies and maintain compliance in regulated environments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L925]
**Business Impact:** Commands only execute if explicitly permitted by enterprise policy, with deny rules taking precedence over allow rules for security-first enforcement [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L940-L945]

## Feature Overview
**Purpose:** Validate command strings against configured allow/deny permission lists with deny-precedence logic and segment validation, ensuring only approved commands execute in enterprise environments [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L960]
**Scope:** Core validation engine that checks commands against permission rules, handles compound command segmentation, and returns allow/deny decisions with detailed reasoning
**PRD References:** REQ-008 (Implement command permission validation) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1465-L1480 - Traceability Matrix]
**PRD Feature ID:** EPIC-ENT-SEC-008-VALID-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L960]
**Dependencies:** 
- FEAT-ENT-SEC-008-PARSE-001 (Permission Rule Parsing) - Provides parsed permission rules with compiled glob patterns
- EPIC-AUTO-EXEC-006 (Yolo Mode) - Permission validation integrates with auto-approval flows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1688-L1720 - Phase 6 Dependencies]

## IAOOI Components

**Inputs:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L925-L930 - Inputs Section]
1. Command strings submitted by Cline agent for execution
2. Parsed permission rules from `CLINE_COMMAND_PERMISSIONS` (allow list, deny list, allowRedirects)
3. Compound command strings with multiple segments (e.g., `npm install && npm test`)
4. Redirect settings (`allowRedirects` boolean flag)

**Activities:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L930-L935 - Activities Section]
1. Validate command segments against deny list (first priority - deny rules take precedence)
2. Validate command segments against allow list (if deny check passes)
3. Parse compound commands into individual segments for separate validation
4. Check redirects and piping against `allowRedirects` policy
5. Return detailed allow/deny decision with reasoning

**Outputs:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L935-L940 - Outputs Section]
1. Permission decisions: ALLOW or DENY with detailed reason
2. Security warnings for policy violations
3. Blocked command notifications with explanation
4. Audit log entries for all validation events

**Outcomes:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L940-L945 - Outcomes Section]
1. Commands only execute if explicitly permitted by enterprise policy
2. Deny rules take precedence over allow rules, ensuring security-first enforcement
3. Administrators have granular control over approved command patterns
4. Failed validations provide clear feedback about policy violations

**Impacts:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L945-L950 - Impacts Section]
1. **Security Compliance**: Meets enterprise security governance and regulatory requirements
2. **Risk Reduction**: Prevents accidental or malicious execution of dangerous commands
3. **Enterprise Adoption**: Enables deployment in regulated industries (finance, healthcare, government)
4. **Trust & Confidence**: Builds enterprise confidence in Cline as a secure automation tool

## Technical Requirements
**Architecture Layer:** Infrastructure/Security Domain
**Integration Points:**
- **Proposed:** `internal/security/validator.go` - Command validation engine with glob matching
- **Proposed:** Receives parsed permission rules from `internal/security/permissions.go` (FEAT-ENT-SEC-008-PARSE-001)
- **Proposed:** Provides validation decisions to yolo mode execution flow (EPIC-AUTO-EXEC-006)
- **Proposed:** Logs validation decisions to audit system (EPIC-ENT-AUDIT-010)

**Data Requirements:**
- **Input:** PermissionRules struct containing:
  - `allow []string` - List of allowed glob patterns
  - `deny []string` - List of denied glob patterns  
  - `allowRedirects bool` - Whether redirects/pipes are permitted
- **Output:** ValidationResult struct containing:
  - `allowed bool` - Whether command is permitted
  - `reason string` - Detailed explanation of decision
  - `violationType string` - Type of violation if denied (deny_list, allow_list, redirects)

**Performance Requirements:**
- Sub-millisecond validation overhead per command [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1688-L1720 - Success Metrics]
- Efficient glob pattern matching using compiled matchers
- No regex injection vulnerabilities (patterns treated as globs, not regex)

**Security Requirements:**
- Deny rules checked before allow rules (security-first precedence)
- All command segments validated separately in compound commands
- No execution of partially validated compound commands
- Clear audit trail for all validation decisions

## User Experience
**User Personas:** Enterprise Administrator (defines policies), Developer (affected by enforcement)
**User Actions:**
1. Administrator configures permission rules via `CLINE_COMMAND_PERMISSIONS` environment variable
2. Cline agent attempts command execution during task
3. Validation engine checks command against policies
4. If denied: User sees clear explanation of policy violation
5. If allowed: Command executes normally

**CLI Integration:**
- Validation occurs automatically before command execution in yolo/automated mode
- Blocked commands display policy violation message with specific rule that triggered denial
- Validation results feed into audit logs for security review

## BDD Scenarios

```gherkin
Feature: FEAT-ENT-SEC-008-VALID-002 - Command Validation Engine

  Scenario: Deny takes precedence over allow
    Given allow=["*"] and deny=["rm *"]
    When validating "rm file.txt"
    Then command should be denied
    Because deny rules take precedence

  Scenario: Allow list restricts to specific commands
    Given allow=["npm *","git *"] and no deny
    When validating "npm install"
    Then command should be allowed
    When validating "python script.py"
    Then command should be denied

  Scenario: Validate command segments in compound commands
    Given allow=["npm *","git *"] and deny=["rm *"]
    When validating "npm install && npm test"
    Then both segments should be validated
    And command should be allowed
    When validating "npm install && rm -rf /"
    Then command should be denied
    Because "rm -rf /" matches deny list

  Scenario: Validate command with arguments
    Given allow=["npm run *","git push origin *"]
    When validating "npm run build"
    Then command should be allowed
    When validating "npm run test --coverage"
    Then command should be allowed
    When validating "git push origin main"
    Then command should be allowed

  Scenario: Deny pattern with wildcard blocks matching commands
    Given deny=["curl *","wget *"]
    When validating "curl https://example.com"
    Then command should be denied
    When validating "wget https://example.com/file.zip"
    Then command should be denied

  Scenario: Empty allow list denies all commands
    Given allow=[] and deny=[]
    When validating any command
    Then command should be denied
    Because empty allow list means no commands permitted

  Scenario: Missing permissions allows all commands
    Given CLINE_COMMAND_PERMISSIONS is not set
    When validating "any command"
    Then command should be allowed
    Because no restrictions apply

  Scenario: Redirect validation with allowRedirects=false
    Given allow=["*"] and allowRedirects=false
    When validating "npm install > output.log"
    Then command should be denied
    Because redirects are not allowed

  Scenario: Pipe validation with allowRedirects=false
    Given allow=["*"] and allowRedirects=false
    When validating "cat file.txt | grep pattern"
    Then command should be denied
    Because pipes are not allowed

  Scenario: Redirect validation with allowRedirects=true
    Given allow=["*"] and allowRedirects=true
    When validating "npm install > output.log"
    Then command should be allowed

  Scenario: Complex compound command validation
    Given allow=["npm *","echo *","cat *"] and deny=["rm *","dd *"]
    When validating "npm install && echo 'done' && cat package.json"
    Then all segments should pass validation
    And command should be allowed

  Scenario: Validation returns detailed reason for denial
    Given deny=["rm -rf *"]
    When validating "rm -rf /important/data"
    Then command should be denied
    And reason should contain "matches deny pattern: rm -rf *"
    And violationType should be "deny_list"

  Scenario: Validation returns success reason for allowance
    Given allow=["npm *"] and deny=[]
    When validating "npm install"
    Then command should be allowed
    And reason should contain "matches allow pattern: npm *"
```
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L962-L1000 - BDD Scenarios]

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L1000 - Success Criteria implied by BDD scenarios]
**Functional:**
- All BDD scenarios pass for command validation with deny/allow lists
- Deny rules consistently take precedence over allow rules
- Compound commands are properly segmented and each segment validated
- 100% of denied commands are blocked before execution

**Performance:**
- Sub-millisecond validation overhead per command
- Efficient glob pattern matching without performance degradation

**Quality:**
- Zero false positives (legitimate commands blocked incorrectly)
- Zero false negatives (unauthorized commands that bypass validation)
- Clear, actionable error messages for policy violations

**Integration:**
- Seamless integration with permission parsing (FEAT-ENT-SEC-008-PARSE-001)
- Validation decisions feed correctly into yolo mode execution flow
- Audit logging captures all validation events

## Testing Strategy
**Unit Testing:**
- Test glob pattern matching with various wildcards (*, ?)
- Test deny-precedence logic with overlapping patterns
- Test compound command segmentation (&&, ||, ;, |)
- Test redirect/pipe detection
- Test edge cases: empty strings, special characters, unicode

**Integration Testing:**
- Integration with permission parsing component
- Integration with yolo mode execution flow
- End-to-end validation with real permission configurations

**Security Testing:**
- Test command injection attempts through pattern bypasses
- Test ReDoS prevention (patterns treated as globs, not regex)
- Test validation of obfuscated compound commands

**Dual Testing (GoLang CLI vs TypeScript CLI):**
- Verify identical validation behavior in both implementations
- Compare validation results for identical command/permission combinations
- Ensure consistent error messages and violation reasons

## Tasks Overview
1. **Implement Validation Engine Core** - Create validator.go with ValidateCommand() function
2. **Implement Glob Pattern Matching** - Build efficient matcher using filepath.Match or equivalent
3. **Implement Compound Command Segmentation** - Parse command strings into segments (&&, ||, ;, |)
4. **Implement Redirect/Pipe Detection** - Check for >, >>, <, | characters
5. **Implement Validation Result Struct** - Define ALLOW/DENY response with detailed reasoning
6. **Implement Deny-First Logic** - Ensure deny list checked before allow list
7. **Implement Audit Logging Integration** - Log all validation decisions
8. **Write Unit Tests** - Comprehensive test coverage for all scenarios
9. **Write Integration Tests** - Test with permission parsing component
10. **Write Dual Tests** - Compare behavior with TypeScript CLI validation

## Implementation Notes

**Security Architecture Notes:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L950-L1000 - Technical Implementation Details]
1. **Deny-First Logic**: Deny rules are checked before allow rules, ensuring security takes precedence
2. **Segment Validation**: Compound commands are split and each segment validated separately
3. **Glob Pattern Matching**: Use efficient glob matching (e.g., `filepath.Match` or dedicated library) for pattern performance
4. **No Regex Injection**: Patterns are treated as globs, not regex, to prevent ReDoS attacks

**Compound Command Delimiters:**
- `&&` - Execute second command only if first succeeds
- `||` - Execute second command only if first fails  
- `;` - Execute commands sequentially
- `|` - Pipe output from first to second (also redirect)

**Redirect Characters to Check (when allowRedirects=false):**
- `>` - Redirect stdout to file (overwrite)
- `>>` - Redirect stdout to file (append)
- `<` - Redirect stdin from file
- `|` - Pipe stdout to another command

**Implementation Priority:** Phase 6 (Security & Enterprise) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1688-L1720 - AI Execution Plan Phase 6]

**Dependencies:**
- FEAT-ENT-SEC-008-PARSE-001 (Permission Rule Parsing) for parsed permission input
- EPIC-AUTO-EXEC-006 (Yolo Mode) for execution flow integration
- EPIC-ENT-AUDIT-010 (Audit & Compliance) for validation logging (optional but recommended)

**Proposed Go Package Structure:**
```
internal/security/
├── permissions.go    # FEAT-ENT-SEC-008-PARSE-001
├── validator.go      # FEAT-ENT-SEC-008-VALID-002 (this feature)
├── dangerous.go      # FEAT-ENT-SEC-008-DANGER-003
└── audit.go          # Shared with EPIC-ENT-AUDIT-010
```

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L###-L###]
- [x] Epic reference cites actual epic.md file [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-SEC-008/epic.md]
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] BDD scenarios extracted verbatim from PRD with proper Gherkin format
- [x] IAOOI framework components extracted from PRD sections
- [x] Requirements mapping includes REQ-008 reference
- [x] Dependencies cite related epics and features