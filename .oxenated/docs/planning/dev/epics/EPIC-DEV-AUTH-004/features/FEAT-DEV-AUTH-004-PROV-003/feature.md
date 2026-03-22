# Provider Configuration Wizard

## Feature ID
FEAT-DEV-AUTH-004-PROV-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L463, L414-L463]

## Epic Context
**Parent Epic:** EPIC-DEV-AUTH-004 - Authentication & Provider Configuration [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md]
**Target Persona:** Developer User
**Epic Objective:** Enable secure, flexible authentication with multiple AI providers while maintaining a seamless user experience. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]
**Business Impact:** Reduced onboarding friction for new users, enabling configuration in under 60 seconds with both interactive wizard and quick flag-based setup for automation scenarios. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410, L414-L463]

## Feature Overview
**Purpose:** Provide an interactive provider configuration wizard that guides users through selecting AI providers, authentication methods, and default models. Also supports quick non-interactive configuration via command-line flags for automation scenarios. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
**Scope:** 
- Interactive wizard for first-time setup (`cline auth`)
- Quick configuration via flags (`cline auth -p <provider> -k <key> -m <model>`)
- Provider selection and model discovery
- Configuration persistence to state storage
**PRD References:** REQ-009, REQ-016, REQ-017 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]
**PRD Feature ID:** EPIC-DEV-AUTH-004-PROV-003 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
**Dependencies:** 
- EPIC-INFRA-STORAGE-012: State & Storage Layer - Required for persisting configurations [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175]
- EPIC-INFRA-CORE-011: Core Extension Integration - Required for gRPC communication of provider changes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L400-L403, L414-L463]

**Inputs:**
1. User selections via interactive prompts (provider choice, auth method, model preference)
2. Command-line flags: `-p` (provider), `-k` (api-key), `-m` (model), `--config` (config path)
3. Available providers list from core extension via gRPC
4. Model lists fetched from provider endpoints
5. Existing configuration from `~/.cline/data/globalState.json`
6. Environment variables for override (`CLINE_DIR`)

**Activities:**
1. **Interactive Wizard Flow**: Guide users through provider selection, authentication method choice, and model selection using Bubble Tea TUI components
2. **Flag-based Quick Setup**: Parse and validate command-line flags for non-interactive configuration
3. **Provider Discovery**: Fetch available providers from core extension via gRPC
4. **Model Discovery**: Query provider endpoints to fetch available models list
5. **Configuration Validation**: Validate provider ID, API key format, and model availability before saving
6. **Configuration Persistence**: Save provider settings including default model to `~/.cline/data/globalState.json`
7. **Secrets Storage**: Delegate API key storage to EPIC-DEV-AUTH-004-KEY-002 for encryption
8. **gRPC State Update**: Notify core extension of provider configuration changes
9. **Mode Detection**: Determine interactive vs non-interactive mode based on flags and TTY state

**Outputs:**
1. Provider configuration saved to `~/.cline/data/globalState.json` with fields: `providerId`, `defaultModel`, `modelPreferences`
2. Confirmation message showing configured provider and model
3. Error messages for invalid provider, failed validation, or storage errors
4. Available models list displayed during interactive setup
5. Configuration summary for user review

**Outcomes:**
1. New users can configure their first provider in under 60 seconds
2. Automation users can configure providers without interactive prompts
3. Provider configurations persist across CLI invocations
4. Users can view and switch between multiple configured providers
5. Default model selection optimizes user experience per provider

**Impacts:**
1. **Reduced Time-to-First-Task**: Quick setup increases user activation rates
2. **Automation Enablement**: Flag-based configuration supports CI/CD workflows
3. **User Experience**: Interactive wizard reduces configuration errors
4. **Operational Efficiency**: Self-service configuration reduces support burden
5. **Multi-Provider Flexibility**: Easy provider switching enables cost optimization

## Technical Requirements
**Architecture Layer:** Application Layer (CLI command handler + TUI components)
**Integration Points:** 
- Proposed: gRPC client to core extension for provider state updates [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- Proposed: Integration with storage layer for `globalState.json` persistence [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
- Existing: Storage path `~/.cline/data/globalState.json` [Source: .clinerules/storage.md]

**Data Requirements:** 
- Proposed: Configuration schema for provider settings (providerId: string, defaultModel: string, modelPreferences: object)
- Existing: `globalState.json` storage format [Source: .clinerules/storage.md]
- Existing: Provider enum from protobuf definitions in `proto/cline/models.proto`

**Performance Requirements:**
- Wizard initialization: <500ms to display available providers
- Model list fetching: <2 seconds for provider endpoint response
- Configuration save: <100ms to persist to storage

**Security Requirements:**
- Never log API keys passed via flags (delegate to secure storage)
- Mask API keys in terminal output (show only last 4 characters)
- Validate all user inputs before processing

## User Experience
**User Personas:** Developer User (individual developers), DevOps/Automation User (CI/CD scenarios)

**User Actions:**
1. **First-Time Setup**: Run `cline auth` → select provider → choose auth method → select model → confirm
2. **Quick Configuration**: Run `cline auth -p anthropic -k sk-ant-xxxxx -m claude-sonnet-4` → immediate configuration
3. **View Configured Providers**: Run `cline auth` with existing config → display list → select to switch
4. **Update Default Model**: Run `cline auth` → select existing provider → choose different model

**UI Components:**
- Proposed: Bubble Tea list component for provider selection
- Proposed: Bubble Tea text input with masking for API key entry
- Proposed: Bubble Tea list component for model selection
- Proposed: Confirmation view with configuration summary
- Proposed: Progress indicator during model fetching

**Mobile Considerations:** N/A - CLI tool for desktop/server environments only

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]

```gherkin
Scenario: Interactive provider wizard
  Given the user runs "cline auth"
  When the wizard starts
  Then available providers should list
  And model selection should follow
  And configuration should save

Scenario: Quick setup with all flags
  Given the user runs "cline auth -p anthropic -k key -m claude-sonnet"
  When the command executes
  Then provider should configure immediately
  Without interactive prompts
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L395-L406]

**Functional:**
- Interactive wizard guides users through complete provider setup
- Flag-based configuration works without interactive prompts
- Provider configurations persist correctly to `globalState.json`
- Model lists display accurately from provider endpoints
- Configuration changes notify core extension via gRPC

**Performance:**
- Wizard displays within 500ms of command execution
- Model list fetches within 2 seconds
- Configuration saves within 100ms

**Quality:**
- Input validation prevents invalid configurations
- Clear error messages for all failure scenarios
- No plaintext credentials in output or logs

**Integration:**
- Seamless integration with storage layer (EPIC-INFRA-STORAGE-012)
- Correct gRPC communication with core extension (EPIC-INFRA-CORE-011)
- Compatible with API key management (EPIC-DEV-AUTH-004-KEY-002)

**Business Value:**
- New users complete setup in under 60 seconds
- 100% of automation scenarios support non-interactive flags
- Zero configuration data loss across CLI restarts

## Testing Strategy
**Unit Testing:**
- Flag parsing validation for all combinations
- Provider ID validation logic
- Model list filtering and sorting
- Configuration schema validation

**Integration Testing:**
- End-to-end wizard flow with mock TTY
- gRPC communication with mock core extension
- Storage layer read/write operations
- Provider endpoint mocking for model discovery

**User Acceptance:**
- First-time user can complete setup without documentation
- DevOps engineer can script configuration
- Configuration persists across CLI invocations
- Dual CLI parity: TypeScript CLI and GoLang CLI produce identical state files

**Dual Testing Requirements (Per PRD Mandate):**
- [ ] Interactive wizard flow comparison between TypeScript and GoLang CLIs
- [ ] Flag-based configuration output format comparison
- [ ] Configuration file format compatibility verification
- [ ] Exit code matching for all scenarios
- [ ] Error message format comparison [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L650-L730]

## Tasks Overview
1. **Task 1**: Implement command-line flag parsing for `auth` subcommand (-p, -k, -m, --config)
2. **Task 2**: Create Bubble Tea TUI wizard components (provider list, model list, confirmation)
3. **Task 3**: Implement gRPC client calls to fetch available providers from core extension
4. **Task 4**: Implement model discovery by querying provider endpoints
5. **Task 5**: Create configuration validation logic (provider ID, model availability)
6. **Task 6**: Implement configuration persistence to `globalState.json` via storage layer
7. **Task 7**: Add gRPC notification to core extension on provider configuration change
8. **Task 8**: Create unit tests for flag parsing and validation logic
9. **Task 9**: Create integration tests for wizard flow and storage operations
10. **Task 10**: Implement dual testing verification with TypeScript CLI

## Implementation Notes

### Go Dependencies (Proposed)
- `github.com/charmbracelet/bubbletea` - TUI framework for interactive wizard
- `github.com/charmbracelet/bubbles` - Pre-built components (list, text input)
- `github.com/spf13/cobra` - CLI framework (already used for command structure)
- Existing gRPC client from EPIC-INFRA-CORE-011

### Configuration Schema
```json
{
  "providerId": "anthropic",
  "defaultModel": "claude-sonnet-4-20250514",
  "modelPreferences": {
    "preferredModels": ["claude-sonnet-4", "claude-opus-4"],
    "fallbackEnabled": true
  }
}
```

### Mode Detection Logic
- If any auth flags provided (-p, -k, -m): Non-interactive mode
- If TTY not available: Error (interactive mode requires TTY)
- If no flags and TTY available: Interactive wizard mode

### Security Considerations
- API keys passed via `-k` flag are immediately passed to secure storage layer
- Never display full API keys in terminal output
- Log only provider ID and model ID (never keys)
- Validate provider ID against allowed list from core extension

### Dual Testing Parity Points
- Output format for `cline auth` help text must match byte-for-byte
- Configuration JSON structure must be identical between CLIs
- Exit codes: 0 for success, 1 for validation errors, 2 for storage errors

## Citation Verification
- [x] All PRD content includes source line numbers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L###]
- [x] Existing code references cite actual file paths (.clinerules/storage.md)
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Citation Verification checklist completed