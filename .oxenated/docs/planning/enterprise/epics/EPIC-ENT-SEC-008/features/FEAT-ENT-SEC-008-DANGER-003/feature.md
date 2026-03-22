# Dangerous Character Detection

## Feature ID
FEAT-ENT-SEC-008-DANGER-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L997-L1000 - Feature 3: Dangerous Character Detection]

## Epic Context
**Parent Epic:** EPIC-ENT-SEC-008 - Security & Permissions [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L1000]
**Target Persona:** Enterprise User
**Epic Objective:** Deliver enterprise-grade security controls for the Cline CLI, enabling organizations to enforce command execution policies and maintain compliance in regulated environments. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L916-L925]
**Business Impact:** Prevents command injection attacks by detecting dangerous character patterns, protecting against malicious input that could compromise system security. This feature is critical for enterprise adoption in regulated industries where command injection vulnerabilities pose unacceptable risks. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L945-L950]

## Feature Overview
**Purpose:** Detect and flag dangerous characters in command strings to prevent command injection attacks. This feature analyzes command input to identify potentially malicious patterns including backticks outside of single quotes, unquoted newlines, and subshell constructs that could be exploited for code injection. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L997-L1000]

**Scope:** 
- **Included:** Detection of backticks outside single quotes, unquoted newlines, subshell patterns ($(...), `...`)
- **Excluded:** Content inside properly quoted strings (single quotes block dangerous character detection)
- **Integration:** Works in conjunction with permission validation (EPIC-ENT-SEC-008-VALID-002) as a secondary security layer

**PRD References:** REQ-008 (Implement command permission validation including dangerous character detection) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1465-L1480]
**Dependencies:** 
- Proposed: `internal/security/validator.go` - Command validation engine (from EPIC-ENT-SEC-008-VALID-002)
- Proposed: `internal/security/dangerous.go` - New file for dangerous character detection utilities

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L997-L1000 - Feature 3 IAOOI]

**Inputs:**
1. Command strings submitted by Cline agent for execution
2. Commands containing special characters: backticks, newlines, subshells
3. Quote context (single quotes, double quotes) to determine if characters are protected

**Activities:**
1. Scan command strings for dangerous character patterns
2. Detect backticks outside single quotes (command substitution)
3. Detect unquoted newlines (potential command injection)
4. Detect subshell constructs: `$()` and backtick command substitution
5. Analyze quote context to determine if characters are protected
6. Allow backticks/newlines inside single quotes (literal interpretation)
7. Flag commands containing dangerous patterns for blocking

**Outputs:**
1. Dangerous character detection results (boolean: safe or dangerous)
2. Detailed report of detected dangerous patterns with locations
3. Security warnings for policy violations
4. Blocked command notifications with specific pattern explanations

**Outcomes:**
1. Commands containing injection patterns are detected before execution
2. Backticks inside single quotes are correctly identified as safe
3. Subshell attempts are flagged regardless of quoting context (except single quotes)
4. Clear feedback is provided about why commands were flagged

**Impacts:**
1. **Security Hardening:** Prevents command injection attacks that could compromise systems
2. **Risk Reduction:** Blocks potentially malicious commands before they execute
3. **Compliance Support:** Meets security audit requirements for command validation
4. **Enterprise Confidence:** Enables safe deployment in production environments

## Technical Requirements
**Architecture Layer:** Infrastructure/Security Layer

**Integration Points:** 
- **Proposed:** `internal/security/dangerous.go` - New utility module for dangerous character detection
- **Proposed:** `internal/security/validator.go` - Command validation engine calls dangerous character detection [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1027-L1030]
- **Integration:** Results feed into command execution decision in EPIC-ENT-SEC-008-VALID-002

**Data Requirements:** 
- No persistent storage required - operates on transient command strings
- Configuration: Dangerous character patterns are hardcoded for security (not configurable)

**Performance Requirements:** 
- Sub-millisecond detection overhead per command
- Linear time complexity O(n) with command length
- No external dependencies that could impact performance

**Security Requirements:** 
- Must correctly handle complex quoting scenarios
- Must not be bypassable through encoding or escaping tricks
- Must treat all content outside single quotes as potentially dangerous
- Detection must be deterministic and consistent

## User Experience
**User Personas:** Enterprise User, DevOps/Automation User (indirect - benefits from security)

**User Actions:** 
- Commands are automatically scanned before execution
- Dangerous commands are blocked with clear explanation
- No user configuration required - detection is automatic

**UI Components:** 
- **Proposed:** Security warning display in plain text mode
- **Proposed:** Error message output showing detected dangerous pattern

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L997-L1000 - Feature 3 BDD Scenarios]

```gherkin
Feature: Dangerous Character Detection

  Scenario: Detect backticks outside single quotes
    Given command "echo `whoami`"
    When dangerous character check runs
    Then it should detect backticks
    And command should be flagged as dangerous

  Scenario: Allow backticks inside single quotes
    Given command "echo '`whoami`'"
    When dangerous character check runs
    Then it should not flag as dangerous
    Because backticks are inside single quotes

  Scenario: Detect unquoted newlines
    Given command with embedded newline
    When validation runs
    Then it should detect unquoted newline
    And flag as dangerous

  Scenario: Detect subshell with $() outside quotes
    Given command "echo $(cat /etc/passwd)"
    When dangerous character check runs
    Then it should detect subshell pattern
    And command should be flagged as dangerous

  Scenario: Allow subshell inside single quotes
    Given command "echo '$(not executed)'"
    When dangerous character check runs
    Then it should not flag as dangerous
    Because subshell is inside single quotes

  Scenario: Detect backticks in compound commands
    Given command "npm install && echo `date`"
    When dangerous character check runs
    Then it should detect backticks in second segment
    And entire command should be flagged as dangerous

  Scenario: Allow complex quoted strings
    Given command "echo 'backtick: ` not dangerous'"
    When dangerous character check runs
    Then it should not flag as dangerous
    Because content is protected by single quotes
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1045-L1050 - Success Metrics]

**Functional:**
- 100% detection of backticks outside single quotes
- 100% detection of unquoted newlines
- 100% detection of subshell patterns outside single quotes
- Zero false positives for content inside single quotes

**Performance:**
- Sub-millisecond detection overhead per command
- No measurable impact on command execution time

**Quality:**
- Comprehensive test coverage for edge cases
- Deterministic behavior across all input types
- Clear error messages for blocked commands

**Integration:**
- Seamless integration with command validation engine (EPIC-ENT-SEC-008-VALID-002)
- Consistent behavior across all execution paths (interactive, yolo, JSON modes)

**Security:**
- Zero bypass techniques succeed
- All documented dangerous patterns are detected
- No false negatives for command injection attempts

## Testing Strategy
**Unit Testing:**
- Test individual detection functions with various input patterns
- Test quote state tracking logic
- Test edge cases: nested quotes, escaped quotes, empty strings

**Integration Testing:**
- Test integration with command validation engine
- Test with real-world command patterns
- Test performance with large command strings

**Security Testing:**
- Attempt to bypass detection with encoding tricks
- Test with obfuscated injection attempts
- Validate all BDD scenarios pass

**Regression Testing:**
- Ensure detection doesn't break legitimate commands
- Verify no false positives for common valid patterns

## Tasks Overview
1. **Task 1:** Create `internal/security/dangerous.go` with core detection logic
2. **Task 2:** Implement quote state parser (single quote tracking)
3. **Task 3:** Implement backtick detection algorithm
4. **Task 4:** Implement newline detection algorithm
5. **Task 5:** Implement subshell detection algorithm ($() and ``)
6. **Task 6:** Write comprehensive unit tests (100% coverage)
7. **Task 7:** Integrate with command validation engine
8. **Task 8:** Implement error/warning message formatting
9. **Task 9:** Performance benchmarking and optimization
10. **Task 10:** Security testing and bypass validation

## Implementation Notes
**Key Design Decisions:**
1. **Single Quote Protection:** Content inside single quotes is treated as literal and not scanned for dangerous characters. This is because single quotes prevent shell interpretation of special characters.
2. **Double Quote Limitation:** Double quotes do NOT protect against backticks or $() - they only prevent word splitting and glob expansion. Therefore, backticks inside double quotes are still detected as dangerous.
3. **Order of Detection:** Scan character by character, tracking quote state, then apply detection rules only when outside single quotes.
4. **Deterministic Rules:** No configuration options - dangerous patterns are hardcoded to prevent policy bypass through configuration manipulation.

**Go Implementation Considerations:**
- Use `utf8.RuneCountInString` for proper Unicode handling
- Implement as a state machine with quote tracking
- Return detailed error types for different detection categories
- Avoid regex for performance (use character iteration instead)

**Cross-Platform Considerations:**
- Newline detection should handle both `\n` (Unix) and `\r\n` (Windows) patterns
- Quote handling is consistent across POSIX and Windows shells for single quotes

**Integration with Permission System:**
This feature operates as a secondary security layer. Even if a command passes the allow/deny list validation, it can still be blocked by dangerous character detection. The recommended integration flow:
1. Parse command into segments
2. Check against deny list (EPIC-ENT-SEC-008-VALID-002)
3. Check against allow list (EPIC-ENT-SEC-008-VALID-002)
4. **Run dangerous character detection (this feature)**
5. Only execute if all checks pass

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines (none exist - this is new functionality)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new