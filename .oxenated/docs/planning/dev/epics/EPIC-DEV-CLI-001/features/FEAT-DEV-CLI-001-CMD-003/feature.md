# History Subcommand with Pagination

## Feature ID
FEAT-DEV-CLI-001-CMD-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L341 - Epic Section, L279-L293 - Feature Definition]

## Epic Context
**Parent Epic:** EPIC-DEV-CLI-001 - Command Line Interface Foundation [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-CLI-001/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Establish the core CLI infrastructure for the GoLang Cline CLI migration, providing foundational command parsing, routing, and subcommand structure using the Cobra CLI framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L241]
**Business Impact:** This feature enables users to browse and reference previous tasks, providing task continuity and referenceability that supports workflow efficiency and debugging [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]

## Feature Overview
**Purpose:** Provide a command-line interface for users to view and browse their task history with pagination support, enabling quick access to previous AI coding sessions for reference or resumption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]
**Scope:** Implements the `cline history` subcommand with pagination flags (-n for limit, -p for page) and integration with the existing task storage system (~/.cline/data/)
**PRD References:** REQ-018 (Task history with pagination) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1253-L1255 - Traceability Matrix]
**PRD Feature ID:** EPIC-DEV-CLI-001-CMD-003 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]
**Dependencies:** 
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for reading task history from storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L933-L1011]
- EPIC-DEV-TASK-003 (Task Management) - Provides task metadata structure [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L422-L517]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293 - Feature IAOOI section]

**Inputs:**
- Limit flag (`-n`, `--limit`): Number of tasks to display per page
- Page flag (`-p`, `--page`): Page number for pagination
- Config path flag (`--config`): Custom configuration directory path
- Environment variable `CLINE_DIR`: Base directory for Cline data storage
- Task history data from `~/.cline/data/taskHistory.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]

**Activities:**
- Parse and validate pagination flags with Cobra CLI framework
- Read task history from file-based JSON storage
- Sort tasks by timestamp (most recent first)
- Calculate pagination offsets based on limit and page
- Format task entries for display (ID, timestamp, summary)
- Render paginated output with navigation hints
- Handle empty state (no tasks in history)

**Outputs:**
- Paginated task list displaying task ID, timestamp, and summary
- Pagination metadata (current page, total pages, total tasks)
- Error messages for invalid page numbers or storage access issues
- Plain text formatted output (default) or JSON formatted output (with --json flag)

**Outcomes:**
- Users can browse previous tasks chronologically
- Users can navigate through large task histories efficiently
- Users can quickly identify and reference specific past tasks
- Users can view recent tasks without pagination for quick access

**Impacts:**
- Task continuity: Users can easily find and resume previous work
- Referenceability: Past solutions and conversations are accessible
- Workflow efficiency: Reduces time spent searching for previous tasks
- Debugging support: Access to historical task context for troubleshooting

## Technical Requirements
**Architecture Layer:** Application Layer (CLI Command Handler) with Infrastructure Layer (Storage Access)

**Integration Points:**
- **Existing Storage API:** Reads from `~/.cline/data/taskHistory.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]
- **Proposed:** `internal/storage/history.go` - History reader interface
- **Proposed:** `internal/cli/history.go` - Cobra command implementation
- **Existing State Structure:** Compatible with existing TypeScript CLI task history format [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]

**Data Requirements:**
- **Existing Schema:** Task history entries with fields: taskId, timestamp, task (summary), tokensIn, tokensOut, cacheWrites, cacheReads, totalCost [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]
- **Proposed:** Go struct mapping to existing JSON structure for compatibility

**Performance Requirements:**
- Read operation: <50ms for history files up to 10MB
- Pagination calculation: <10ms regardless of history size
- Display rendering: <100ms for 50 tasks
- Support history files with 10,000+ tasks

**Security Requirements:**
- Respect file permissions on `~/.cline/data/` directory (mode 0o700)
- No exposure of sensitive task content in summary view
- Secure handling of task metadata only (not full conversation content)

## User Experience
**User Personas:** Developer User (primary), DevOps/Automation User (scripting with --json)

**User Actions:**
1. View recent tasks: `cline history` (shows last 10 tasks by default)
2. Limit results: `cline history -n 5` (shows only 5 most recent)
3. Paginate through history: `cline history -n 20 -p 2` (shows tasks 21-40)
4. JSON output for scripting: `cline history --json`
5. Custom config location: `cline history --config /custom/path`

**UI Components:**
- **Proposed:** `internal/cli/history.go` - Command definition with Cobra
- **Proposed:** `internal/history/formatter.go` - Plain text table formatter
- **Proposed:** `internal/history/json_formatter.go` - JSON output formatter
- **Proposed:** Terminal table output with columns: ID, Date/Time, Summary

**Output Format (Plain Text):**
```
ID        DATE                SUMMARY
abc123    2024-01-15 14:30    Create hello world function
def456    2024-01-15 13:15    Fix bug in main.go
ghi789    2024-01-14 09:00    Refactor authentication module

Page 1 of 5 (50 total tasks)
```

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293 - Gherkin BDD Scenarios section]

```gherkin
Scenario: List recent tasks
  Given the user has completed tasks in history
  When the user runs "cline history"
  Then the last 10 tasks should display
  And each task should show ID, timestamp, and summary

Scenario: Paginate through history
  Given the user has many tasks in history
  When the user runs "cline history -n 20 -p 2"
  Then tasks 21-40 should display
  And pagination info should show

Scenario: Limit history results
  Given the user wants to see specific count
  When the user runs "cline history -n 5"
  Then only 5 most recent tasks should display
```

**Additional Edge Case Scenarios:**

```gherkin
Scenario: Empty history
  Given the user has no task history
  When the user runs "cline history"
  Then a message should display "No tasks in history"
  And exit code should be 0

Scenario: Invalid page number
  Given the user has 5 pages of history
  When the user runs "cline history -p 10"
  Then an error should display "Page 10 does not exist"
  And exit code should be 1

Scenario: JSON output format
  Given the user wants machine-readable output
  When the user runs "cline history --json"
  Then output should be valid JSON array
  And each entry should have id, timestamp, and summary fields

Scenario: Large history pagination performance
  Given the user has 10,000 tasks in history
  When the user runs "cline history -n 50 -p 100"
  Then results should display within 200ms
  And correct page should show (tasks 4951-5000)
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239, L1413-L1422 - Success Metrics]

**Functional:**
- `cline history` command displays last 10 tasks by default [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]
- `-n` flag limits results to specified count [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L341 - Flag Reference Table]
- `-p` flag paginates to specified page number [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L341 - Flag Reference Table]
- Task ID, timestamp, and summary display for each entry [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]
- Pagination info displays current page and total [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L279-L293]

**Performance:**
- History loading completes within 100ms for files up to 10MB
- Pagination works efficiently with 10,000+ task histories

**Quality:**
- Help text auto-generated by Cobra matches existing CLI coverage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239, L1413-L1422]
- Error messages are helpful and actionable [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239, L1413-L1422]
- Exit codes match existing CLI (0 for success, 1 for errors) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L813-L833]

**Integration:**
- Works seamlessly with existing task storage format (~/.cline/data/) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]
- Compatible with `--json` global flag for scripting [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L341]
- Respects `--config` flag for custom data directory [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L341]

**Business Value:**
- Users can reference previous tasks without remembering IDs
- Task continuity is supported through easy history browsing

## Testing Strategy
**Unit Testing:**
- Flag parsing validation (limit > 0, page > 0)
- Pagination calculation logic (offset, page count)
- Formatter output validation (plain text and JSON)
- Error handling for missing/invalid history files

**Integration Testing:**
- Integration with file storage layer (read operations)
- Integration with existing task history JSON format
- End-to-end command execution with temporary storage

**User Acceptance:**
- Help text displays correctly: `cline history --help`
- Default behavior shows last 10 tasks
- Pagination navigation works as expected
- JSON output is parseable and complete

**Dual Testing (Critical):**
- Execute `cline history` in both TypeScript and GoLang CLIs
- Compare output byte-for-byte (except timestamps)
- Verify identical exit codes
- Test all flag combinations produce identical results
- Verify task history created by one CLI is readable by the other [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L813-L833, L1385-L1400]

## Tasks Overview
1. **Task 1:** Create Cobra command definition for `history` subcommand with flag definitions
2. **Task 2:** Implement history storage reader interface and file-based implementation
3. **Task 3:** Implement pagination logic (offset calculation, page validation)
4. **Task 4:** Implement plain text table formatter for terminal output
5. **Task 5:** Implement JSON formatter for `--json` flag support
6. **Task 6:** Write unit tests for pagination and formatting logic
7. **Task 7:** Write integration tests with temporary storage
8. **Task 8:** Dual testing verification against existing TypeScript CLI

## Implementation Notes
**Implementation Priority:** Phase 2 - Core CLI Foundation, Third in sequence (after Root Command and Task Subcommand) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1310-L1330]

**Flag Reference:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| --limit | -n | Number of tasks per page | 10 |
| --page | -p | Page number | 1 |
| --config | | Custom config path | ~/.cline/data/ |
| --json | | Output as JSON | false |

**Storage Path Resolution:**
1. Check `--config` flag
2. Check `CLINE_DIR` environment variable
3. Default to `~/.cline/data/`
4. Read `taskHistory.json` from resolved path [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]

**Independence Requirements:**
- MUST NOT import any code from `cli/src/` directory
- MUST NOT depend on Node.js or npm
- MUST be pure Go implementation using only Go-native libraries [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L34-L53]

**Proposed File Structure:**
- `golang-cli/internal/cli/history.go` - Cobra command definition
- `golang-cli/internal/history/reader.go` - History storage interface
- `golang-cli/internal/history/file_reader.go` - File-based implementation
- `golang-cli/internal/history/formatter.go` - Plain text formatter
- `golang-cli/internal/history/json_formatter.go` - JSON formatter
- `golang-cli/internal/history/pagination.go` - Pagination logic

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L##-L##]
- [x] Existing code references cite actual file paths and lines [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]
- [x] New functionality clearly marked as "Proposed:" when it doesn't exist yet
- [x] Integration points cite existing interfaces or marked as new
- [x] Citation Verification checklist completed in feature.md