# File-based JSON Storage

## Feature ID
FEAT-INFRA-STORAGE-012-FILE-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1260 - Epic 12: State & Storage Layer]

## Epic Context
**Parent Epic:** EPIC-INFRA-STORAGE-012 - State & Storage Layer [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Provide foundational data persistence infrastructure for the Cline CLI GoLang migration, ensuring reliable state management across CLI invocations with full compatibility with existing Cline storage format (~/.cline/data/) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1223]
**Business Impact:** Enables seamless switching between TypeScript and GoLang CLI implementations without data loss; foundation for all state-dependent features across all personas [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]

## Feature Overview
**Purpose:** Implement atomic file-based JSON storage operations for the GoLang CLI that read and write state files in the exact same format as the existing TypeScript CLI, ensuring data integrity through atomic rename operations and file locking for concurrent access safety [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1224-L1226]

**Scope:** 
- **Included:** JSON file reading with error handling, atomic write operations using temp-then-rename pattern, file locking for concurrent CLI instance safety, cross-platform path resolution, workspace-specific state isolation
- **Excluded:** Secrets encryption (handled by FEAT-INFRA-STORAGE-012-SECRET-002), state migration logic (handled by FEAT-INFRA-STORAGE-012-MIGRATE-003), in-memory caching layer

**PRD References:** REQ-010 (Maintain state persistence), REQ-016 (Configuration management), REQ-018 (Task history with pagination) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1754-L1760]
**PRD Feature ID:** EPIC-INFRA-STORAGE-012-FILE-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1224-L1226]
**Dependencies:** 
- Protobuf definitions for state schema [Source: proto/cline/state.proto]
- Proposed: Go module initialization (EPIC-INFRA-CORE-011 project structure)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1224-L1226 - Feature 1: File-based JSON Storage]

**Inputs:**
- Storage paths: `~/.cline/data/globalState.json`, `~/.cline/data/workspaces/<hash>/workspaceState.json`, `~/.cline/data/tasks/taskHistory.json`
- JSON data structures for global state, workspace state, and task history
- File locking requirements for concurrent access
- Configuration paths via `CLINE_DIR` environment variable
- Existing TypeScript CLI state files in compatible format

**Activities:**
- Read JSON state files with proper error handling and validation
- Write JSON state files with atomic rename operations (write to temp file, then rename to target) for crash safety
- Implement advisory file locking to prevent corruption from concurrent CLI instances
- Manage workspace-specific state isolation through path hashing
- Handle cross-platform path resolution (Windows, macOS, Linux)
- Parse and validate JSON schema compatibility

**Outputs:**
- Loaded configuration state (global and workspace) as Go structs
- Persisted state changes to JSON files with atomic guarantees
- Thread-safe file access via locking mechanisms
- Consistent state across CLI invocations
- Error reports for file access failures or corruption detection

**Outcomes:**
- Reliable state persistence across all CLI operations with zero data loss
- Zero data loss during CLI crashes or interruptions due to atomic writes
- Seamless compatibility with existing TypeScript CLI state format
- Concurrent CLI instance safety through file locking
- Cross-platform state consistency

**Impacts:**
- User trust in data persistence and reliability
- Smooth migration path from TypeScript to GoLang CLI
- Foundation for enterprise features requiring audit trails
- Reduced support issues related to state corruption
- Enablement of state-dependent features across all personas

## Technical Requirements
**Architecture Layer:** Infrastructure/Storage Layer

**Integration Points:**
- Proposed: `internal/storage/file_storage.go` - Core atomic file operations implementation
- Proposed: `internal/storage/paths.go` - Cross-platform path resolution utilities
- Existing: gRPC state synchronization with core extension via `cline/state.proto` [Source: proto/cline/state.proto]
- Proposed: StateManager in-memory cache layer (reads from file storage)

**Data Requirements:**
- Existing state schema defined in TypeScript: `GlobalState`, `WorkspaceState`, `TaskHistory` types [Source: src/shared/storage/state-keys.ts]
- Proposed: Go struct definitions mirroring TypeScript interfaces for JSON serialization compatibility
- JSON format must remain identical to existing TypeScript CLI output

**Performance Requirements:**
- State file read operations must complete in <10ms for files <1MB
- State file write operations must complete in <20ms with atomic guarantee
- File locking must not block indefinitely (implement timeouts)
- Support for state files up to 10MB without performance degradation

**Security Requirements:**
- File permissions: State files should be created with 0o600 permissions (user read/write only)
- Directory permissions: `~/.cline/data/` should be 0o700
- No plaintext secrets in state files (secrets handled by separate feature)

## User Experience
**User Personas:** All personas (Developer, DevOps, Enterprise) - this is foundational infrastructure

**User Actions:**
- CLI startup triggers automatic state loading from `~/.cline/data/globalState.json`
- Task creation/updates trigger state persistence
- Task history queries read from `~/.cline/data/tasks/taskHistory.json`
- Workspace-specific operations load/save to `~/.cline/data/workspaces/<hash>/workspaceState.json`
- Multiple CLI instances can run concurrently without data corruption

**UI Components:**
- N/A - This is a backend infrastructure feature with no direct UI

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1226-L1260 - Feature 1 BDD Scenarios]

```gherkin
Scenario: Read global state
  Given ~/.cline/data/globalState.json exists with valid JSON
  When CLI loads configuration
  Then global state should read from file
  And parse into Go structs matching TypeScript schema
  And all configuration values should be accessible

Scenario: Atomic write operation
  Given updated state needs persistence
  When write operation executes
  Then it should write to temp file first
  And atomically rename to target file
  To prevent corruption during crashes or interruptions

Scenario: Handle concurrent access
  Given multiple CLI instances are running
  When they attempt to access the same state file simultaneously
  Then file locking should prevent corruption
  And writes should be serialized safely
  And reads should not block indefinitely

Scenario: Create state file if missing
  Given ~/.cline/data/globalState.json does not exist
  When CLI attempts to load configuration
  Then it should create the file with default values
  And parent directories should be created if needed

Scenario: Handle corrupted state file
  Given ~/.cline/data/globalState.json contains invalid JSON
  When CLI attempts to load configuration
  Then it should return a clear error message
  And offer to reset to defaults or restore from backup

Scenario: Workspace state isolation
  Given the user is in a workspace at /project/path
  When workspace state is saved
  Then it should be stored in ~/.cline/data/workspaces/<hash>/workspaceState.json
  Where <hash> is a deterministic hash of the workspace path

Scenario: Cross-platform path resolution
  Given the CLI runs on Windows, macOS, or Linux
  When resolving state file paths
  Then it should use platform-appropriate path separators
  And respect CLINE_DIR environment variable if set
  And fall back to default ~/.cline/data/ if not set
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221 - Success Metrics]

**Functional:**
- All existing state files readable without data loss or format errors
- Atomic write operations prevent corruption in 100% of crash scenarios
- Concurrent CLI instances safely share state without corruption
- State file format matches TypeScript CLI output exactly (verified through dual testing)

**Performance:**
- State file read operations complete in <10ms for typical files
- State file write operations complete in <20ms
- File locking does not cause indefinite blocking

**Quality:**
- Zero data loss during power failures or process crashes
- Clear error messages for file access failures
- Proper file permissions (0o600 for files, 0o700 for directories)

**Integration:**
- GoLang CLI can read state files created by TypeScript CLI
- TypeScript CLI can read state files created by GoLang CLI
- State synchronization works correctly with core extension via gRPC

**Business Value:**
- Foundation for all other epics that require state persistence
- Enables seamless migration between CLI implementations
- Reduces support burden from state corruption issues

## Testing Strategy
**Unit Testing:**
- Test atomic write operations with simulated crashes
- Test file locking behavior with concurrent goroutines
- Test JSON serialization/deserialization round-trips
- Test path resolution on all target platforms
- Test error handling for corrupted/missing files

**Integration Testing:**
- Test reading state files created by existing TypeScript CLI
- Test state file compatibility between GoLang and TypeScript implementations
- Test concurrent access with multiple CLI processes
- Test cross-platform state file portability

**Dual Testing (Critical):**
- Execute identical state operations in both TypeScript and GoLang CLIs
- Compare resulting state files byte-for-byte (excluding timestamps)
- Verify both CLIs can read each other's state files
- Test concurrent access scenarios with mixed CLI implementations

**Performance Testing:**
- Benchmark read/write operations against performance requirements
- Test with large state files (up to 10MB)
- Measure file locking overhead under concurrent load

## Tasks Overview
1. **Implement atomic file operations** - Write-then-rename pattern for crash-safe writes
2. **Implement file locking** - Advisory locking for concurrent CLI instance safety
3. **Implement path resolution** - Cross-platform path handling with CLINE_DIR support
4. **Implement JSON serialization** - Go structs matching TypeScript schema
5. **Implement error handling** - Graceful handling of corrupted/missing files
6. **Create comprehensive tests** - Unit, integration, and dual-testing verification

## Implementation Notes

### Go-Native Implementation Requirements
Per the independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55], this feature MUST:
- Use only Go standard library or pure Go modules
- Not import or transpile any TypeScript/JavaScript code
- Implement all logic in pure Go without Node.js dependencies

### Recommended Go Libraries
- Standard library `encoding/json` - JSON serialization/deserialization
- Standard library `os` - File operations with atomic rename via `os.Rename()`
- Standard library `sync` - For any in-process synchronization (file locking needs external package)
- `github.com/gofrs/flock` or similar - Cross-platform advisory file locking (pure Go)

### Critical Implementation Details

1. **Atomic Write Pattern:**
   ```go
   // Write to temp file, then rename
   tempFile := targetPath + ".tmp"
   // ... write to tempFile ...
   os.Rename(tempFile, targetPath)  // Atomic on POSIX and Windows
   ```

2. **File Locking Strategy:**
   - Use advisory locking (not mandatory) to allow compatibility with other tools
   - Implement timeout to prevent indefinite blocking
   - Support both shared locks (for reads) and exclusive locks (for writes)

3. **Path Resolution:**
   - Default: `~/.cline/data/` (respecting $HOME)
   - Override: `$CLINE_DIR/data/` if environment variable set
   - Workspace hash: SHA256 of absolute workspace path, truncated to 16 characters

4. **TypeScript Compatibility:**
   - Go struct field names must match TypeScript interface property names
   - JSON tags must preserve exact serialization format
   - Handle TypeScript `undefined` as Go zero values or omitted fields

### Existing TypeScript Reference Implementation
The existing TypeScript implementation uses [Source: src/shared/storage/ClineFileStorage.ts]:
- Synchronous file operations for simplicity
- Write-then-rename atomic pattern
- No file locking (relies on Node.js single-threaded nature)
- JSON serialization with `JSON.stringify()` and `JSON.parse()`

The Go implementation must add file locking since Go programs can run truly concurrent goroutines and multiple OS processes.

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or marked as new
- [x] BDD scenarios extracted verbatim from PRD
- [x] IAOOI framework extracted from PRD source