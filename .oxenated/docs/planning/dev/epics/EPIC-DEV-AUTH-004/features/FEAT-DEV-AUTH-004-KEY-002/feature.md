# API Key Management

## Feature ID
FEAT-DEV-AUTH-004-KEY-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L415-L425 - Feature 2: API Key Management]

## Epic Context
**Parent Epic:** EPIC-DEV-AUTH-004 - Authentication & Provider Configuration [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]
**Target Persona:** Developer User
**Epic Objective:** Enable secure, flexible authentication with multiple AI providers while maintaining a seamless user experience. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]
**Business Impact:** Secure credential handling meets enterprise security requirements; encryption and secure storage satisfy compliance standards (SOC2, ISO 27001). [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]

## Feature Overview
**Purpose:** Provide secure storage and validation of API keys for AI providers, enabling users to configure API key-based authentication through both command-line flags and interactive prompts. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L415-L425]

**Scope:** 
- API key input via command-line flags (`-k`, `--key`)
- Secure interactive input prompts with masked characters
- API key validation against provider endpoints before storage
- Encryption and storage in OS keyring or file-based encrypted storage
- Support for all API key-based providers (OpenAI, Anthropic, OpenRouter, etc.)

**PRD References:** REQ-009, REQ-016, REQ-017 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410, L1159-L1175]
**PRD Feature ID:** EPIC-DEV-AUTH-004-KEY-002
**Dependencies:** 
- EPIC-INFRA-STORAGE-012 (State & Storage Layer) - Required for secrets.json persistence [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175]
- Proposed: OS Keyring Integration component for encryption
- Proposed: Provider Validation component for API key testing

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L415-L425]

**Inputs:**
1. API keys provided via command-line flags (`-k`, `--key`)
2. API keys provided via interactive secure input prompts
3. Provider identification (provider ID for validation endpoint selection)
4. Existing credentials from `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
5. OS keyring integration for encryption keys

**Activities:**
1. **API Key Validation**: Test API keys against provider endpoints to verify validity before storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L401-L403]
2. **Secure Credential Storage**: Encrypt and store API keys using OS-native keyring or file-based encryption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L401-L403]
3. **Interactive Input Handling**: Capture API keys via secure prompts with masked character display
4. **Provider Configuration Update**: Save provider settings including the validated API key
5. **Credential Retrieval**: Decrypt and retrieve stored credentials for API requests
6. **Validation Status Reporting**: Return success/failure status for credential validation

**Outputs:**
1. Encrypted credentials stored in `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
2. Validation success/failure status for credentials [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L402-L404]
3. Configuration confirmation messages
4. Error messages for invalid credentials or connection failures

**Outcomes:**
1. Users can securely authenticate with API key-based providers
2. Credentials are protected with enterprise-grade encryption
3. No plaintext credentials are stored or logged
4. Invalid API keys are detected before storage
5. Authentication state persists across CLI invocations

**Impacts:**
1. **Enterprise Adoption**: Secure credential handling meets enterprise security requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]
2. **Security Compliance**: Encryption and secure storage satisfy compliance standards (SOC2, ISO 27001) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]
3. **User Trust**: Robust security practices build user confidence [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]
4. **Operational Efficiency**: Quick provider setup reduces time-to-first-task [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]

## Technical Requirements

**Architecture Layer:** Application/Infrastructure Layer

**Integration Points:**
- Proposed: OS Keyring Integration - Cross-platform keyring access via `github.com/zalando/go-keyring` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- Proposed: Provider Validation - HTTP client for testing API keys against provider endpoints [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- EPIC-INFRA-STORAGE-012: Reads/writes encrypted credentials to `~/.cline/data/secrets.json` (mode 0o600) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
- EPIC-INFRA-CORE-011: Sends provider configuration updates to core extension via gRPC [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]

**Data Requirements:**
- Proposed: Secrets storage schema in `~/.cline/data/secrets.json` with encrypted API key entries
- Existing storage location: `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]

**Performance Requirements:**
- Key validation should complete within 5 seconds for responsive user experience
- Encryption/decryption operations should not add perceptible delay (<100ms)

**Security Requirements:**
- Never log API keys or display in terminal output (except masked) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]
- Use OS keyring when available, never store keys in plaintext [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]
- File-based fallback must use AES-256-GCM encryption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]
- Memory should zero sensitive data after use [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]
- Validate TLS certificates for all validation API calls [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]

## User Experience

**User Personas:** Developer User, DevOps/Automation User

**User Actions:**
1. Quick configuration via flags: `cline auth -p openai -k sk-xxxxx`
2. Interactive secure input: `cline auth` → select provider → enter key at secure prompt
3. Key validation feedback: Success/failure messages after validation attempt

**UI Components:**
- Proposed: Bubble Tea-based secure input component with masked characters [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- Proposed: Support for pasting multi-line keys [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- Proposed: Visual feedback for input validation [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463 - BDD scenarios for EPIC-DEV-AUTH-004-KEY-002]

```gherkin
Scenario: Configure API key via flag
  Given the user runs "cline auth -p openai -k sk-xxxxx"
  When the command executes
  Then the key should validate
  And store encrypted in secrets.json
  And configuration should update

Scenario: Interactive API key input
  Given the user runs "cline auth" interactively
  When they select API key provider
  Then secure input prompt should display
  And key should store encrypted
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L406, L414-L463]

**Functional:**
- API keys can be configured via `-k` flag without interactive prompts
- Interactive secure prompts mask input characters
- Keys are validated against provider endpoints before storage
- Valid keys are encrypted and stored securely
- Invalid keys are rejected with clear error messages

**Performance:**
- Key validation completes within 5 seconds
- Encryption/decryption adds <100ms overhead

**Quality:**
- No plaintext credentials in logs or memory dumps
- Secure memory handling (zeroing after use)
- Proper error handling for network failures during validation

**Integration:**
- Credentials stored in format compatible with existing Cline core extension
- Works with all API key-based providers (OpenAI, Anthropic, OpenRouter, etc.)

**Business Value:**
- New users can authenticate in under 60 seconds
- >95% of entered API keys pass validation on first attempt
- Meets enterprise security requirements for credential storage

## Testing Strategy

**Unit Testing:**
- Encryption/decryption logic with test keys
- Input masking and secure prompt handling
- Provider validation mock responses

**Integration Testing:**
- End-to-end key storage and retrieval
- Cross-platform keyring integration tests
- Provider API validation with mock servers

**User Acceptance:**
- Manual verification of flag-based configuration
- Interactive prompt usability testing
- Security audit for credential handling

**Dual Testing Requirements:**
Per the PRD dual testing mandate, all API key management functionality MUST be tested in BOTH the existing TypeScript CLI AND the new GoLang CLI: [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L650-L730]

- [ ] API key configuration via flags comparison
- [ ] Interactive wizard flow comparison
- [ ] Secrets encryption/retrieval comparison
- [ ] Provider switching behavior comparison
- [ ] Error message format comparison
- [ ] Configuration file format compatibility

**Parity Verification:**
- Output format for `cline auth` commands must match byte-for-byte (except timestamps)
- Exit codes must be identical for all scenarios
- State files must be mutually readable between CLIs

## Tasks Overview
- Task 1: Implement secure input component with masking
- Task 2: Implement OS keyring integration with fallback encryption
- Task 3: Implement API key validation against provider endpoints
- Task 4: Integrate with storage layer for secrets persistence
- Task 5: Implement command-line flag handling for `-k`/`--key`
- Task 6: Add dual testing scenarios for API key management

## Implementation Notes

**Go Dependencies (Proposed):**
- `github.com/zalando/go-keyring` - Cross-platform keyring access [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]
- Standard library: `crypto/aes`, `crypto/cipher` for fallback encryption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]

**Implementation Priority:**
This feature is the highest priority within EPIC-DEV-AUTH-004 as it represents the "quick win, most common use case" [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]. It should be implemented before OAuth flows.

**Dependencies:**
- Requires EPIC-INFRA-STORAGE-012 (State & Storage Layer) for secrets.json persistence [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]
- Blocks EPIC-INFRA-API-013 (API Provider Integrations) - cannot make API calls without authentication [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines where applicable
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Dual testing requirements included with PRD source
- [x] Security requirements traceable to PRD security section