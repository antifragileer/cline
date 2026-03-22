# Config, Auth, Update, Version, Dev Subcommands

## Feature ID
FEAT-DEV-CLI-001-CMD-004

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L310-L341]

## Epic Context
**Parent Epic:** EPIC-DEV-CLI-001 - Command Line Interface Foundation [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-CLI-001/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Establish the core CLI infrastructure for the GoLang Cline CLI migration, providing foundational command parsing, routing, and subcommand structure using the Cobra CLI framework [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L225-L241]
**Business Impact:** Enables complete CLI functionality for configuration management, authentication, updates, and debugging, providing self-service management capabilities that reduce support requests and improve developer productivity [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L239]

## Feature Overview
**Purpose:** Implements the management and utility subcommands (config, auth, update, version, dev) that enable users to manage CLI configuration, authenticate with AI providers, check for updates, view version information, and access debugging tools. These commands provide essential self-service capabilities for CLI management without requiring direct file manipulation or external tools [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L310-L315].

**Scope:** 
- **Included:** Config subcommand for viewing/managing configuration, Auth subcommand for provider authentication, Update subcommand for version checking, Version subcommand for version display, Dev subcommand for debugging utilities
- **Excluded:** Task execution (handled by FEAT-DEV-CLI-001-CMD-002), History browsing (handled by FEAT-DEV-CLI-001-CMD-003), Interactive TUI mode (handled by EPIC-DEV-UI-002)

**PRD References:** REQ-003 (Implement all existing CLI commands) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1253-L1255]
**PRD Feature ID:** EPIC-DEV-CLI-001-CMD-004 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L310]
**Dependencies:** 
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for config command to read/write state [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L933-L1011]
- EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) - Required for auth subcommand workflows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L519-L589]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L310-L315 - IAOOI section for this feature]

**Inputs:**
- Subcommand specific flags and arguments (--provider, --key for auth; --config for global config path)
- Environment variables (CLINE_DIR for custom data directory)
- Existing configuration from ~/.cline/data/globalState.json and secrets.json
- npm registry for update checking
- Log file paths for dev command

**Activities:**
- Route to appropriate subcommand handler based on command name
- Execute config display logic (read and format global/workspace state)
- Execute auth workflows (interactive wizard or quick flag-based setup)
- Execute update check (query npm registry, compare versions)
- Execute version display (show current CLI version)
- Execute dev utilities (open log files, debug info)

**Outputs:**
- Subcommand results:
  - Config: Formatted display of global and workspace state
  - Auth: Authentication tokens stored securely, provider configuration updated
  - Update: Version comparison result, update availability status
  - Version: Version string output
  - Dev: Log file opened in default editor or debug info displayed

**Outcomes:**
- Users can view and understand current CLI configuration
- Users can authenticate with AI providers interactively or via flags
- Users can check for and install CLI updates
- Users can verify installed CLI version
- Users can access debugging tools and logs for troubleshooting

**Impacts:**
- Complete CLI functionality enabling self-service management
- Reduced support requests due to clear configuration visibility
- Streamlined onboarding through interactive authentication wizard
- Easy troubleshooting via dev utilities and logs
- Consistent management experience across all platforms

## Technical Requirements
**Architecture Layer:** Application/CLI Layer

**Integration Points:**
- **Proposed:** gRPC client to communicate with core extension for configuration validation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L343-L420]
- **Proposed:** HTTP client for npm registry queries (update command)
- **Proposed:** OS keyring integration for secure credential storage (auth command) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L933-L1011]
- **Existing Storage:** Reads from ~/.cline/data/globalState.json, ~/.cline/data/secrets.json [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]

**Data Requirements:**
- **Existing Model:** GlobalState structure including provider configurations, model preferences [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72]
- **Existing Model:** Secrets storage for API keys with OS keyring integration [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L933-L1011]
- **Existing Model:** Task history for recent task display in config

**Performance Requirements:**
- Config/Version commands: <50ms response time
- Auth command: OAuth flow completion within 60 seconds
- Update command: <5 seconds for npm registry query
- Dev command: <100ms to open log file

**Security Requirements:**
- API keys must be encrypted at rest using OS keyring [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L933-L1011]
- Auth tokens must never be logged or displayed in plaintext
- Config command must mask sensitive values (show *** for API keys)

## User Experience
**User Personas:** 
- Developer User: Uses all commands for CLI management
- Enterprise User: Uses auth/config for SSO and policy compliance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L519-L589]

**User Actions:**
1. **Config workflow**: Run `cline config` → View formatted global and workspace state → Optionally modify settings
2. **Auth workflow (interactive)**: Run `cline auth` → Select provider from list → Enter credentials → Confirm success
3. **Auth workflow (quick)**: Run `cline auth -p <provider> -k <key>` → Immediate configuration
4. **Update workflow**: Run `cline update` → Check npm registry → Display update availability → Offer install option
5. **Version workflow**: Run `cline version` → Display current version string
6. **Dev workflow**: Run `cline dev log` → Open log file in default system editor

**UI Components:**
- **Proposed:** Config display formatter (table/key-value output)
- **Proposed:** Interactive auth wizard (provider selection, secure input prompts)
- **Proposed:** Update availability indicator with semantic versioning comparison
- **Proposed:** Log file opener (system default editor integration)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L326-L341 - BDD scenarios for this feature]

```gherkin
Scenario: Display configuration
  Given the user wants to see current config
  When the user runs "cline config"
  Then current global and workspace state should display

Scenario: Interactive authentication
  Given the user needs to authenticate
  When the user runs "cline auth"
  Then an interactive wizard should guide through provider selection and API key input

Scenario: Quick authentication with flags
  Given the user has an API key
  When the user runs "cline auth -p anthropic -k sk-ant-xxxxx"
  Then the provider should be configured immediately

Scenario: Check for updates
  Given the user wants latest version
  When the user runs "cline update"
  Then it should check npm for newer versions
  And offer to install if available

Scenario: Show version
  Given the user wants version info
  When the user runs "cline version"
  Then the current CLI version should display

Scenario: Open dev logs
  Given the user needs to debug
  When the user runs "cline dev log"
  Then the log file should open in default editor
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L310-L315 - Success criteria derived from PRD]

**Functional:**
- Config command displays formatted global and workspace state
- Auth command supports both interactive wizard and flag-based quick setup
- Auth command validates API keys before storage
- Auth command encrypts credentials using OS keyring
- Update command queries npm registry and displays version comparison
- Version command outputs correct semantic version
- Dev command opens log files in system default editor

**Performance:**
- Config/Version commands complete in <50ms
- Update check completes in <5 seconds
- Auth wizard initializes in <100ms

**Quality:**
- All sensitive data masked in config output
- Auth failures provide clear error messages
- Update check handles network failures gracefully

**Integration:**
- Config command integrates with State & Storage layer
- Auth command integrates with Authentication epic
- All commands work with existing ~/.cline/data/ directory structure

**Business Value:**
- Users can self-manage CLI configuration
- Authentication onboarding time reduced through interactive wizard
- Debugging capabilities reduce support burden

## Testing Strategy
**Unit Testing:**
- Config parsing and display formatting
- Auth flag validation and provider detection
- Version string formatting
- Update version comparison logic

**Integration Testing:**
- Config command reads from actual state files
- Auth command writes to secrets storage
- Update command queries npm registry (mocked)
- Dev command opens files (mocked system calls)

**User Acceptance:**
- Interactive auth wizard guides users through complete flow
- Config output is readable and complete
- Version output matches expected format

**Dual Testing Requirements:**
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L813-L833, L1385-L1400]

All subcommands MUST pass dual testing against existing TypeScript CLI:
- `cline config` output format must match exactly
- `cline auth` behavior must be identical
- `cline update` version checking must produce same results
- `cline version` output format must match
- `cline dev` log opening must work identically
- Exit codes must match for all scenarios

## Tasks Overview
The following tasks will implement this feature:

1. **TASK-001**: Implement `config` subcommand with state display
2. **TASK-002**: Implement `auth` subcommand with interactive wizard and flag support
3. **TASK-003**: Implement `update` subcommand with npm registry checking
4. **TASK-004**: Implement `version` subcommand
5. **TASK-005**: Implement `dev` subcommand with log opening
6. **TASK-006**: Add comprehensive tests for all subcommands
7. **TASK-007**: Dual testing verification against existing CLI

## Implementation Notes
**Key Implementation Details:**

1. **Cobra Framework Usage**: All subcommands use Cobra for consistent help generation, flag parsing, and command routing [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L236]

2. **State Storage Path**: Default path is ~/.cline/data/, overridable via CLINE_DIR environment variable or --config flag [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L64-L72, L235]

3. **Security Considerations**: 
   - API keys must be encrypted at rest
   - Config display must mask sensitive values
   - Auth tokens must be handled securely

4. **Independence Requirements**: 
   - MUST NOT import from existing cli/src/ TypeScript code
   - MUST NOT depend on Node.js runtime
   - Pure Go implementation only [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L34-L53]

5. **Provider Support**: Auth command must support all providers from EPIC-INFRA-API-013 (OpenAI, Anthropic, OpenRouter, Gemini, Bedrock, etc.)

6. **Flag Reference**:
   | Flag | Short | Description | Applies To |
   |------|-------|-------------|------------|
   | --provider | -p | Provider ID | auth |
   | --key | -k | API key | auth |
   | --config | | Custom config path | global |

   [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L273-L295]

**Architectural Decisions:**
- Use Bubble Tea for interactive auth wizard (leverages EPIC-DEV-UI-002 work)
- Use standard Go os/exec for opening log files (cross-platform)
- Use Go's encoding/json for state file reading

**Dependencies on Other Epics:**
- EPIC-INFRA-STORAGE-012 for state file access
- EPIC-DEV-AUTH-004 for authentication workflows
- EPIC-INFRA-API-013 for provider list and validation

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths (where applicable)
- [x] New functionality clearly marked as "Proposed:" where appropriate
- [x] Integration points cite existing interfaces or mark as new