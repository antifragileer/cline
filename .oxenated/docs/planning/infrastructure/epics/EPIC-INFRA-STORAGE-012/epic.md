# State & Storage Layer

## Epic ID
EPIC-INFRA-STORAGE-012

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1252 - Epic 12: State & Storage Layer]

## Target Persona
Infrastructure (Internal)

## Epic Overview
The State & Storage Layer epic provides the foundational data persistence infrastructure for the Cline CLI GoLang migration. This epic implements reliable state management across CLI invocations, ensuring data integrity and seamless compatibility with the existing Cline storage format (~/.cline/data/). The storage layer handles three critical concerns: file-based JSON storage for configuration and state, secure encryption for secrets and API credentials, and migration capabilities for backward compatibility with legacy state formats.

This epic is foundational and must be completed early in the migration as many other epics depend on it for state persistence, including task management, authentication, and enterprise audit features.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1223]

## Vision & Objectives
The State & Storage Layer enables the GoLang CLI to reliably persist and retrieve all user data, configurations, and credentials while maintaining full compatibility with the existing TypeScript CLI's storage format. This ensures users can seamlessly switch between CLI implementations without data loss or migration friction.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1223]

## IAOOI System Components

### Inputs
- Global state JSON files from `~/.cline/data/globalState.json`
- Workspace state JSON files from `~/.cline/data/workspaces/<hash>/workspaceState.json`
- Secrets JSON files from `~/.cline/data/secrets.json`
- Task history from `~/.cline/data/tasks/taskHistory.json`
- Legacy state formats requiring migration
- Encryption keys from OS keyring/keychain
- File locking requirements for concurrent access
- Configuration paths via `CLINE_DIR` environment variable

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1215-L1217]

### Activities
- Read JSON state files with proper error handling
- Write JSON state files with atomic rename operations for crash safety
- Implement file locking to prevent corruption from concurrent CLI instances
- Encrypt sensitive secrets using OS-native keyring/keychain
- Decrypt secrets on-demand for API operations
- Detect legacy state format versions
- Apply automatic migrations to current format
- Update migration version markers
- Manage workspace-specific state isolation
- Handle cross-platform path resolution

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1216-L1218]

### Outputs
- Loaded configuration state (global and workspace)
- Decrypted API credentials and secrets
- Persisted state changes to JSON files
- Migrated state in current format
- Version markers tracking migration state
- Thread-safe file access via locking mechanisms
- Consistent state across CLI invocations

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1217-L1219]

### Outcomes
- Reliable state persistence across all CLI operations
- Zero data loss during CLI crashes or interruptions
- Seamless compatibility with existing TypeScript CLI state
- Secure storage of API credentials without exposure
- Automatic state migration on version upgrades
- Concurrent CLI instance safety
- Cross-platform state consistency

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1218-L1220]

### Impacts
- User trust in data persistence and reliability
- Smooth migration path from TypeScript to GoLang CLI
- Foundation for enterprise features requiring audit trails
- Reduced support issues related to state corruption
- Compliance with security standards for credential storage
- Enablement of state-dependent features across all personas

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]

## Key Features
1. **FEAT-INFRA-STORAGE-012-FILE-001**: File-based JSON Storage - Atomic file operations with locking
2. **FEAT-INFRA-STORAGE-012-SECRET-002**: Secrets Encryption - OS keyring integration for secure credential storage
3. **FEAT-INFRA-STORAGE-012-MIGRATE-003**: State Migration - Automatic backward compatibility and version management

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1224-L1260]

## Business Value & Requirements
This epic addresses the following original requirements:

| Requirement ID | Requirement Description |
|----------------|------------------------|
| REQ-010 | Maintain state persistence (~/.cline/data/) |
| REQ-016 | Configuration management (global and workspace) |
| REQ-017 | Secure secrets storage |
| REQ-018 | Task history with pagination |

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1221-L1222, L1754-L1760]

## User Journeys & Scenarios
This infrastructure epic primarily serves internal technical workflows, but supports the following user-visible scenarios:

### Developer User Journey: Seamless CLI Switching
As a developer user, I can switch between the TypeScript CLI and GoLang CLI without losing my:
- API provider configurations
- Task history
- Conversation context
- Custom settings

This journey requires the GoLang CLI to read and write state in the exact same format as the existing CLI.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L185-L205 - Developer Persona]

### Enterprise User Journey: Secure Credential Management
As an enterprise user, my API credentials are:
- Stored encrypted using my OS keyring
- Never written in plaintext to disk
- Accessible only to authenticated processes
- Portable across CLI implementations

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L215-L223 - Enterprise Persona]

## BDD Scenarios

### Feature 1: File-based JSON Storage (FEAT-INFRA-STORAGE-012-FILE-001)

```gherkin
Scenario: Read global state
  Given ~/.cline/data/globalState.json exists
  When CLI loads configuration
  Then global state should read from file
  And parse into Go structs

Scenario: Atomic write operation
  Given updated state needs persistence
  When write operation executes
  Then it should write to temp file
  And atomically rename to target
  To prevent corruption

Scenario: Handle concurrent access
  Given multiple CLI instances
  When they access same file
  Then file locking should prevent corruption
```

### Feature 2: Secrets Encryption (FEAT-INFRA-STORAGE-012-SECRET-002)

```gherkin
Scenario: Encrypt API key
  Given API key "sk-xxxxx"
  When storing to secrets.json
  Then it should encrypt before writing
  And not be readable as plaintext

Scenario: Decrypt on read
  Given encrypted secrets file
  When CLI reads API key
  Then it should decrypt using OS keyring
  And return plaintext for use
```

### Feature 3: State Migration (FEAT-INFRA-STORAGE-012-MIGRATE-003)

```gherkin
Scenario: Migrate old state format
  Given state file from older version
  When CLI initializes
  Then migration should run automatically
  And state should be in current format

Scenario: Track migration version
  Given migration completes
  Then version marker should update
  And future runs should skip migration
```

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1226-L1260 - Feature BDD Scenarios]

## Technical Considerations

### Existing Code References
This epic involves porting logic from the existing TypeScript storage implementation. Based on the codebase analysis:

- **Storage Architecture**: The existing TypeScript implementation uses `ClineFileStorage` for synchronous JSON file operations with atomic writes
- **StateManager**: In-memory cache layer on top of file storage, populated during initialization
- **File Layout**: 
  - `~/.cline/data/globalState.json` - Global settings & state
  - `~/.cline/data/secrets.json` - API keys (mode 0o600)
  - `~/.cline/data/workspaces/<hash>/workspaceState.json` - Per-workspace state
  - `~/.cline/data/tasks/taskHistory.json` - Task history (separate file)

**Proposed New Components:**
- `internal/storage/file_storage.go` - Go implementation of atomic JSON file operations
- `internal/storage/state_manager.go` - In-memory cache with debounced disk writes
- `internal/storage/secrets.go` - OS keyring integration for encryption/decryption
- `internal/storage/migration.go` - Automatic state format migration logic
- `internal/storage/paths.go` - Cross-platform path resolution utilities

### Critical Implementation Notes

1. **Atomic Writes**: Must implement write-then-rename pattern to prevent corruption during crashes
2. **File Locking**: Required for concurrent CLI instance safety
3. **OS Keyring Integration**: 
   - macOS: Keychain Services
   - Linux: Secret Service API / GNOME Keyring / KWallet
   - Windows: Windows Credential Manager
4. **Backward Compatibility**: Must read existing TypeScript CLI state without modification
5. **Go-Native Libraries**: Use only Go libraries (no cgo dependencies for portability)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1226-L1260 - Feature descriptions, L40-L55 - Independence Requirements]

## Implementation Priority
**Priority: CRITICAL PATH - Phase 1**

This epic is in the critical path and must be completed in Phase 1 (Foundational Setup). It is a prerequisite for:
- EPIC-DEV-CLI-001 (Command Line Interface Foundation) - requires config persistence
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - requires secure secret storage
- EPIC-DEV-TASK-003 (Task Management) - requires task history persistence
- EPIC-ENT-CONFIG-009 (Enterprise Configuration Management) - requires tiered config loading
- EPIC-ENT-AUDIT-010 (Audit & Compliance) - requires audit log persistence

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd-epic-manifest.json - critical_path and dependencies]

## Success Metrics
- All existing state files readable without data loss
- Atomic write operations prevent corruption in 100% of crash scenarios
- Concurrent CLI instances safely share state without corruption
- Secrets encrypted at rest using OS keyring
- Legacy state formats automatically migrate on first run
- Startup state loading completes in <50ms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221 - Impacts]

## Dependencies
**This epic has no external dependencies** - it is a foundational layer that other epics depend on.

**Epics that depend on this:**
- EPIC-DEV-CLI-001 (Command Line Interface Foundation)
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration)
- EPIC-DEV-TASK-003 (Task Management)
- EPIC-ENT-CONFIG-009 (Enterprise Configuration Management)
- EPIC-ENT-AUDIT-010 (Audit & Compliance)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd-epic-manifest.json - dependency_graph]

## Integration Points

### With EPIC-INFRA-CORE-011 (Core Extension Integration)
- State synchronization between CLI and core extension via gRPC
- Shared understanding of state schema for compatibility

### With EPIC-DEV-AUTH-004 (Authentication & Provider Configuration)
- Secure storage of OAuth tokens and API keys
- Provider configuration persistence

### With EPIC-DEV-TASK-003 (Task Management)
- Task history storage and retrieval
- Conversation state persistence

### With EPIC-ENT-AUDIT-010 (Audit & Compliance)
- Audit log file persistence
- Tamper-resistant storage considerations

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1215-L1217 - Inputs/Activities]

## GoLang Implementation Notes

### Must Use Pure Go Libraries
Per the independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55], the GoLang implementation MUST:

- Use only Go-native libraries (no Node.js dependencies)
- Implement all storage logic in pure Go
- Not import or transpile any TypeScript/JavaScript code
- Use standard library or pure Go modules for JSON handling, file operations, and encryption

### Recommended Go Libraries
- `github.com/zalando/go-keyring` - Cross-platform OS keyring integration
- Standard library `encoding/json` - JSON serialization
- Standard library `sync` - Mutexes for in-memory cache
- Standard library `os` - File operations with atomic rename

### Testing Requirements
Per the dual testing mandate [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L55-L58, L1667-L1720], all storage functionality must be tested in BOTH:
1. The existing TypeScript CLI (compatibility verification)
2. The new GoLang CLI (new implementation)

Tests must verify:
- State file format compatibility between implementations
- Concurrent access safety
- Migration correctness
- Encryption/decryption round-trips

---

## Feature Breakdown Ready

This epic is ready for feature extraction. The three features (FILE-001, SECRET-002, MIGRATE-003) will be detailed in the `features/` subdirectory using the `/prd-feature-extract.md` workflow.