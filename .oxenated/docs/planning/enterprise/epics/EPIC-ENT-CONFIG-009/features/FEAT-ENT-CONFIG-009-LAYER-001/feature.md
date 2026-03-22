# Configuration Layering

## Feature ID
FEAT-ENT-CONFIG-009-LAYER-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 section]

## Epic Context
**Parent Epic:** EPIC-ENT-CONFIG-009 - Enterprise Configuration Management [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L1]
**Target Persona:** Enterprise User
**Epic Objective:** Provide enterprise users with robust configuration management that balances organizational control with developer flexibility, including hierarchical configuration loading, policy enforcement mechanisms, and compliance features. [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L10-L15]
**Business Impact:** Enables IT administrators to define organization-wide policies while allowing developers to maintain workspace-specific settings. Supports hierarchical configuration with proper precedence rules and enforcement of organizational standards. [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L12-L15]

## Feature Overview
**Purpose:** Implement tiered configuration loading with proper precedence rules (environment variables > workspace > global > defaults), enabling enterprise users to maintain consistent baseline configurations while allowing workspace-specific overrides. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 IAOOI]

**Scope:** 
- Configuration loading from multiple sources (global, workspace, environment)
- Hierarchical merging with defined precedence
- Support for `CLINE_DIR` environment variable override
- File-based JSON storage compatibility with existing TypeScript CLI
- GoLang implementation with zero dependencies on TypeScript CLI code

**PRD References:** REQ-016 (Configuration management - global and workspace) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Traceability Matrix]

**PRD Feature ID:** EPIC-ENT-CONFIG-009-LAYER-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Features section]

**Dependencies:** 
- EPIC-INFRA-STORAGE-012: State & Storage Layer (provides file storage foundation) [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L165]
- EPIC-DEV-AUTH-004: Authentication & Provider Configuration (uses configuration system) [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L166]

## IAOOI Components

**Inputs:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 IAOOI]
1. Global configuration files (`~/.cline/data/globalState.json`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]
2. Workspace-specific configuration (`~/.cline/data/workspaces/<hash>/workspaceState.json`) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]
3. Environment variables (`CLINE_DIR`, `CLINE_COMMAND_PERMISSIONS`, and other `CLINE_*` variables) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]
4. Default configuration values and fallbacks [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]

**Activities:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 IAOOI]
1. **Configuration Loading**: Read and parse global state from JSON files [Source: src/shared/storage/ClineFileStorage.ts:L40-L47 - readFromDisk()]
2. **Workspace Detection**: Identify current workspace and load workspace-specific settings [Source: src/shared/storage/storage-context.ts:L67-L75 - hashString() and workspace path resolution]
3. **Environment Variable Parsing**: Extract `CLINE_*` environment variables [Source: src/shared/storage/storage-context.ts:L70 - process.env.CLINE_DIR]
4. **Configuration Merging**: Apply hierarchical merging with proper precedence (env vars > workspace > global > defaults) [Source: src/core/storage/StateManager.ts:L394-L405 - getSettingWithOverride() precedence logic]
5. **Settings Persistence**: Save configuration updates to appropriate tier [Source: src/core/storage/StateManager.ts:L120-L130 - setGlobalState(), setWorkspaceState()]
6. **Default Application**: Apply sensible defaults for unset values [Source: src/core/storage/StateManager.ts:L394-L405 - fallback to global settings]

**Outputs:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 IAOOI]
1. **Effective Configuration**: Merged configuration with all tiers applied in correct precedence [Source: src/core/storage/StateManager.ts:L394-L405]
2. **Configuration Warnings**: Alerts for deprecated settings or required updates
3. **Error Messages**: Clear feedback when configuration is invalid
4. **Help Documentation**: Context-sensitive configuration guidance

**Outcomes:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 IAOOI Outcomes]
1. **Consistent Team Configuration**: All team members operate with baseline enterprise settings
2. **Flexible Workspace Overrides**: Developers can customize non-restricted settings per project
3. **Simplified Onboarding**: New developers inherit enterprise configuration automatically
4. **Reduced Configuration Drift**: Centralized management prevents divergence from standards

**Impacts:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 IAOOI Impacts]
1. **Enterprise Adoption**: Meets security and compliance requirements for regulated industries
2. **IT Governance**: Enables centralized management of AI tooling across the organization
3. **Operational Efficiency**: Reduces support burden from configuration issues
4. **Regulatory Compliance**: Supports audit requirements and data residency policies

## Technical Requirements

**Architecture Layer:** Infrastructure (Configuration Management)

**Integration Points:**
- **Existing Storage Layer**: Uses `ClineFileStorage` for synchronous file-backed JSON storage [Source: src/shared/storage/ClineFileStorage.ts:L1-L90]
- **Existing StorageContext**: Entry point for storage operations creating `StorageContext` with global, secrets, and workspace state [Source: src/shared/storage/storage-context.ts:L1-L90]
- **Existing StateManager**: In-memory cache with debounced persistence to disk [Source: src/core/storage/StateManager.ts:L1-L500]
- **Proposed: GoLang Configuration Package**: Pure Go implementation at `internal/config` with hierarchical loading logic
- **Proposed: Configuration Merger**: Merge strategy interface with type-safe merging for different configuration value types

**Data Requirements:**
- **Existing Schema**: `globalState.json` - Global settings & state [Source: src/shared/storage/storage-context.ts:L28-L29]
- **Existing Schema**: `secrets.json` - API keys with mode 0o600 [Source: src/shared/storage/storage-context.ts:L28-L29]
- **Existing Schema**: `workspaceState.json` - Per-workspace toggles [Source: src/shared/storage/storage-context.ts:L28-L29]
- **Existing Directory Structure**: 
  ```
  ~/.cline/
    data/
      globalState.json
      secrets.json
      workspaces/
        <hash>/
          workspaceState.json
  ```
  [Source: src/shared/storage/storage-context.ts:L28-L29 and src/core/storage/StateManager.ts:L1-L50]

**Precedence Rules (Highest to Lowest):** [Source: src/core/storage/StateManager.ts:L394-L405]
1. Remote config (organization-level settings)
2. Session overrides (CLI flags like `--yolo`)
3. Task-specific settings
4. Global settings
5. Default values

**Environment Variable Support:** [Source: src/shared/storage/storage-context.ts:L70]
- `CLINE_DIR`: Override default `~/.cline` directory [Source: src/shared/storage/storage-context.ts:L70]
- `CLINE_COMMAND_PERMISSIONS`: Command permission rules [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]
- Other `CLINE_*` variables for configuration overrides

**Performance Requirements:**
- Configuration loading: <50ms [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L180]
- In-memory cache for fast reads [Source: src/core/storage/StateManager.ts:L35-L50]
- Debounced persistence (500ms delay) [Source: src/core/storage/StateManager.ts:L58]

**Security Requirements:**
- Secrets stored with file mode 0o600 (owner read/write only) [Source: src/shared/storage/storage-context.ts:L77-L79]
- Atomic file writes using temp file + rename pattern [Source: src/shared/storage/ClineFileStorage.ts:L93-L108]
- No plaintext secrets in logs or error messages

## User Experience

**User Personas:** Enterprise User, Developer User

**User Actions:**
1. **First-time Setup**: CLI automatically loads enterprise global configuration on first run
2. **Project Work**: Workspace-specific settings overlay global config automatically
3. **Override Configuration**: Set `CLINE_DIR` environment variable for custom config location
4. **View Configuration**: Run `cline config` to display current effective configuration

**Configuration Workflow:** [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L95-L105]
1. IT administrator updates global policy
2. All team members automatically receive updated configuration on next CLI invocation
3. Developer overrides non-restricted setting in workspace config
4. CLI merges settings correctly with workspace taking precedence over global

**GoLang CLI Specific Behavior:**
- Pure Go implementation with no Node.js dependencies
- Single binary distribution
- File-compatible with existing TypeScript CLI storage
- gRPC integration for core extension communication

## BDD Scenarios

**Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Epic 9 Feature 1 Gherkin Scenarios]

```gherkin
Scenario: Load tiered configuration
  Given global config has setting A=1
  And workspace config has setting A=2
  When configuration loads
  Then effective A should be 2
  Because workspace overrides global

Scenario: Environment variable override
  Given CLINE_DIR is set to "/custom/path"
  When configuration loads
  Then config directory should be "/custom/path"
  Overriding any file-based setting

Scenario: Global configuration loading
  Given ~/.cline/data/globalState.json exists with provider settings
  When CLI initializes
  Then global state should load from file
  And parse into Go structs

Scenario: Workspace configuration loading
  Given a workspace at /project with workspaceState.json
  When CLI initializes in /project
  Then workspace-specific settings should load
  And overlay global settings

Scenario: Configuration precedence
  Given global setting X=1
  And workspace setting X=2
  And environment variable CLINE_X=3
  When configuration resolves
  Then X should be 3 (environment wins)

Scenario: Atomic write operation
  Given updated state needs persistence
  When write operation executes
  Then it should write to temp file
  And atomically rename to target
  To prevent corruption
```

## Success Criteria

**Functional:** [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L178-L185]
- Successfully loads and merges global, workspace, and environment configurations
- Environment variables override workspace settings, which override global settings
- File storage compatible with existing TypeScript CLI (`~/.cline/data/` structure)
- GoLang CLI achieves exact functional parity with TypeScript CLI configuration behavior

**Performance:**
- Configuration loading completes in <50ms [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L180]
- No blocking I/O during configuration reads (in-memory cache)
- Atomic writes prevent configuration corruption

**Quality:**
- 100% functional parity with existing TypeScript CLI (verified through dual testing)
- No regression in configuration behavior
- Proper error handling for missing or corrupted files

**Integration:**
- Works seamlessly with EPIC-INFRA-STORAGE-012 (State & Storage Layer)
- Compatible with EPIC-DEV-AUTH-004 (Authentication configuration)
- Integrates with EPIC-ENT-SEC-008 (Security permissions)

**Business Value:**
- Enterprise teams can share baseline configurations
- Developers maintain flexibility for project-specific settings
- IT can enforce organizational standards through global config

## Testing Strategy

**Unit Testing:**
- Configuration loading from each tier (global, workspace, env)
- Precedence resolution logic
- Merge strategy implementations
- Error handling for malformed JSON
- File permission validation (secrets mode 0o600)

**Integration Testing:**
- File storage integration with ClineFileStorage equivalent [Source: src/shared/storage/ClineFileStorage.ts:L1-L90]
- Cross-platform path handling (Linux, macOS, Windows)
- Concurrent access handling (file locking)
- StateManager cache synchronization [Source: src/core/storage/StateManager.ts:L1-L500]

**User Acceptance:**
- Configuration commands work identically in TypeScript and GoLang CLIs
- Environment variable overrides behave correctly
- Workspace detection works for various project structures

**Dual Testing (Required):** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Dual Testing Strategy]
- Side-by-side comparison of configuration loading
- Output format verification (JSON structure)
- State file compatibility (tasks created in one CLI can be resumed in the other)
- Flag behavior equivalence testing
- Exit code matching

**Performance Testing:**
- Benchmark configuration loading time
- Memory usage comparison between implementations
- Binary size verification

## Tasks Overview

1. **Implement ClineFileStorage equivalent in Go** - Port synchronous file-backed JSON storage with atomic writes
2. **Implement StorageContext in Go** - Create storage context with global, secrets, and workspace state
3. **Implement workspace hash calculation** - Port deterministic hash function for workspace directory naming
4. **Implement StateManager cache in Go** - In-memory cache with debounced persistence
5. **Implement configuration precedence logic** - Remote > Session > Task > Global > Defaults
6. **Implement environment variable parsing** - Support CLINE_DIR and other CLINE_* variables
7. **Implement atomic file writes** - Temp file + rename pattern for corruption prevention
8. **Add dual testing framework integration** - Verify parity with TypeScript implementation

## Implementation Notes

**Critical Independence Requirements:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND - Critical Independence Requirements]
- The GoLang implementation MUST NOT import, transpile, bundle, or execute any TypeScript/JavaScript code from `cli/src/`
- The GoLang implementation MUST NOT depend on `cli/package.json` or any npm packages
- All configuration functionality MUST be implemented in pure Go using Go-native libraries
- The ONLY permitted connection to existing Cline code is via gRPC/protobuf

**Storage Compatibility:** [Source: .oxenated/docs/planning/enterprise/epics/EPIC-ENT-CONFIG-009/epic.md:L118-L130]
- Configuration file formats (`globalState.json`, `workspaceState.json`) must remain compatible with existing TypeScript CLI
- Storage file layout must match exactly:
  ```
  ~/.cline/
    data/
      globalState.json
      secrets.json
      workspaces/<hash>/workspaceState.json
  ```

**Existing Code Patterns to Follow:** [Source: src/shared/storage/ClineFileStorage.ts:L1-L108]
- Use atomic write pattern (temp file + rename) for data integrity
- Use file mode 0o600 for secrets.json
- Use hash-based workspace isolation (deterministic 8-character hex)
- Use debounced persistence (500ms) to batch writes

**GoLang-Specific Considerations:**
- Use standard library `os` and `path/filepath` for cross-platform compatibility
- Use `sync.RWMutex` for thread-safe cache access
- Use `encoding/json` for JSON serialization
- Consider using `fsnotify` for file watching (task history sync)

## Citation Verification

- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-LEND]
- [x] Existing code references cite actual file paths and lines:
  - [Source: src/shared/storage/ClineFileStorage.ts:L1-L108] - File storage implementation
  - [Source: src/shared/storage/storage-context.ts:L1-L90] - Storage context and path resolution
  - [Source: src/core/storage/StateManager.ts:L1-L500] - State manager with caching
- [x] New functionality clearly marked as "Proposed:" or "To be created:"
  - Proposed: GoLang Configuration Package at `internal/config`
  - Proposed: Configuration Merger with merge strategy interface
- [x] Integration points cite existing interfaces or mark as new
  - Existing: ClineFileStorage, StorageContext, StateManager
  - New: GoLang implementations of above