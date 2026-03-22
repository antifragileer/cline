# Secrets Encryption

## Feature ID
FEAT-INFRA-STORAGE-012-SECRET-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1248-L1252]

## Epic Context
**Parent Epic:** EPIC-INFRA-STORAGE-012 - State & Storage Layer [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Provide foundational data persistence infrastructure for the Cline CLI GoLang migration, ensuring reliable state management across CLI invocations with data integrity and seamless compatibility with existing Cline storage format [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1207-L1223]
**Business Impact:** Enables secure credential storage without exposure, compliance with security standards, and user trust in data persistence and reliability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]

## Feature Overview
**Purpose:** Implement secure encryption for API credentials and secrets using OS-native keyring/keychain integration, ensuring secrets are never stored in plaintext on disk and are accessible only to authenticated processes [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1216-L1218, L1249]
**Scope:** 
- IN SCOPE: OS keyring integration for macOS, Linux, and Windows; encryption/decryption of secrets; secure storage of API keys and OAuth tokens
- OUT OF SCOPE: Custom encryption schemes, cloud-based secret storage, secret rotation logic
**PRD References:** REQ-017 (Secure secrets storage) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1754-L1760]
**PRD Feature ID:** EPIC-INFRA-STORAGE-012-SECRET-002 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1248-L1252]
**Dependencies:** 
- FEAT-INFRA-STORAGE-012-FILE-001 (File-based JSON Storage) - for atomic file operations on secrets.json
- Existing TypeScript storage format compatibility

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1248-L1252]

**Inputs:**
- Plaintext secrets (API keys, OAuth tokens) from authentication flows [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1249]
- Encryption keys from OS keyring/keychain [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1215-L1217]
- Storage paths via `CLINE_DIR` environment variable or default ~/.cline/data/ [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1215-L1217]
- Platform detection (macOS, Linux, Windows) for keyring selection

**Activities:**
- Encrypt secrets before writing to disk using OS-provided encryption mechanisms [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1216-L1218, L1249]
- Decrypt secrets on-demand for API operations using OS keyring [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1216-L1218, L1249]
- Integrate with OS-native keyring services:
  - macOS: Keychain Services
  - Linux: Secret Service API / GNOME Keyring / KWallet
  - Windows: Windows Credential Manager [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Critical Implementation Notes]
- Handle cross-platform path resolution for secret storage
- Manage encryption key lifecycle (creation, retrieval, validation)
- Ensure secrets are never written in plaintext to disk [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L215-L223 - Enterprise User Journey]

**Outputs:**
- Encrypted secrets stored in ~/.cline/data/secrets.json [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1217-L1219]
- Decrypted plaintext secrets returned for API operations
- Consistent state across CLI invocations with secure credential access
- Zero plaintext exposure of sensitive data at rest

**Outcomes:**
- Secure storage of API credentials without exposure [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1218-L1220]
- Credentials accessible only to authenticated processes
- Compliance with security standards for credential storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]
- User trust in data security and reliability

**Impacts:**
- User trust in data persistence and reliability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]
- Compliance with security standards for credential storage [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]
- Enablement of enterprise features requiring secure credential management
- Foundation for enterprise adoption requiring policy compliance

## Technical Requirements
**Architecture Layer:** Infrastructure/Security Layer

**Integration Points:**
- Proposed: New `internal/storage/secrets.go` module for OS keyring integration [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Proposed New Components]
- Proposed: Integration with `internal/storage/file_storage.go` for atomic writes to secrets.json
- Proposed: Integration with authentication module (EPIC-DEV-AUTH-004) for OAuth token storage
- Proposed: Integration with API provider module (EPIC-INFRA-API-013) for credential retrieval

**Data Requirements:**
- Existing storage location: `~/.cline/data/secrets.json` (mode 0o600) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Existing Code References]
- Proposed: Go struct for secrets storage compatible with existing JSON format
- Proposed: Encryption metadata (algorithm version, key ID) stored alongside encrypted data

**Performance Requirements:**
- Encryption/decryption operations should complete in <10ms per secret
- Secret retrieval should not block UI during API operations
- Lazy decryption: only decrypt when needed

**Security Requirements:**
- Use only OS-native keyring/keychain services (no custom encryption) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Critical Implementation Notes]
- Secrets must never be written in plaintext to disk [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L215-L223]
- File permissions on secrets.json must be 0o600 (owner read/write only) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Existing Code References]
- Encryption keys must be stored only in OS keyring, not in application memory longer than necessary
- Clear sensitive variables from memory after use [Reference: .clinerules/bash_coding_best_practices.md - Security]

**Platform-Specific Requirements:**
- macOS: Use Keychain Services API via `github.com/zalando/go-keyring` [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Recommended Go Libraries]
- Linux: Use Secret Service API / D-Bus via `github.com/zalando/go-keyring`
- Windows: Use Windows Credential Manager via `github.com/zalando/go-keyring`

## User Experience
**User Personas:** 
- Developer User: Benefits from seamless authentication without security concerns
- Enterprise User: Requires secure credential management for compliance [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L215-L223]

**User Actions:**
1. Configure API key via `cline auth -p <provider> -k <key>` - key is encrypted before storage
2. Authenticate via OAuth - tokens are encrypted and stored securely
3. Execute tasks requiring API access - credentials are decrypted on-demand
4. Resume previous sessions - encrypted credentials remain secure across invocations

**Integration Flow:**
```
User enters API key → Key is encrypted using OS keyring → 
Encrypted blob written to secrets.json → On API call, decrypt using OS keyring → 
Use plaintext temporarily → Clear from memory
```

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1250-L1252]

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

**Additional Test Scenarios:**

```gherkin
Scenario: Cross-platform keyring integration
  Given the CLI is running on macOS
  When storing a secret
  Then it should use macOS Keychain Services
  
  Given the CLI is running on Linux
  When storing a secret
  Then it should use Secret Service API
  
  Given the CLI is running on Windows
  When storing a secret
  Then it should use Windows Credential Manager

Scenario: Secrets file permissions
  Given a new secrets file is created
  When the file is written
  Then permissions should be 0o600
  And only the owner can read/write

Scenario: Graceful fallback on keyring unavailable
  Given the OS keyring is not accessible
  When attempting to store a secret
  Then an appropriate error should display
  And no plaintext should be written

Scenario: Multiple secret storage
  Given API keys for multiple providers (OpenAI, Anthropic, OpenRouter)
  When storing each key
  Then each should be encrypted independently
  And retrieval should return the correct decrypted key for each provider
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1219-L1221]

**Functional:**
- All API credentials are encrypted at rest using OS keyring [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Success Metrics]
- Decryption works seamlessly across CLI invocations
- Compatible with existing TypeScript CLI secrets format

**Performance:**
- Encryption/decryption operations complete in <10ms
- No perceptible delay during authentication or API calls

**Quality:**
- 100% test coverage for encryption/decryption round-trips
- No plaintext secrets in memory dumps or logs
- Secure handling of all credential types (API keys, OAuth tokens)

**Integration:**
- Works seamlessly with File-based JSON Storage (FEAT-INFRA-STORAGE-012-FILE-001)
- Integrates correctly with Authentication module (EPIC-DEV-AUTH-004)
- Compatible with all API providers (EPIC-INFRA-API-013)

**Business Value:**
- Meets enterprise security compliance requirements
- Enables secure team configuration sharing (without sharing actual credentials)
- Maintains user trust through transparent security practices

## Testing Strategy
**Unit Testing:**
- Test encryption/decryption round-trips for each supported platform
- Test error handling when keyring is unavailable
- Test with various secret sizes and character encodings

**Integration Testing:**
- Test integration with file storage for atomic write operations
- Test cross-CLI compatibility: encrypt in TypeScript CLI, decrypt in GoLang CLI
- Test concurrent access scenarios

**User Acceptance:**
- Verify enterprise user journey: credentials never exposed in plaintext [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L215-L223]
- Verify developer user journey: seamless authentication experience
- Validate file permissions are correct on all platforms

**Security Testing:**
- Verify no plaintext in memory after credential use
- Verify file permissions are 0o600 on secrets.json
- Test keyring isolation (one user cannot access another's credentials)

**Dual Testing Mandate:**
Per REQ-020, all functionality must be tested in BOTH:
1. Existing TypeScript CLI (verify compatibility)
2. New GoLang CLI (verify new implementation) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L55-L58, L1667-L1720]

## Tasks Overview
1. **Implement OS Keyring Integration**: Create `internal/storage/secrets.go` with cross-platform keyring support using `github.com/zalando/go-keyring`
2. **Implement Encryption Layer**: Add encrypt/decrypt functions that wrap secrets before storage
3. **Integrate with File Storage**: Connect with atomic file operations for secure writes to secrets.json
4. **Add Platform-Specific Testing**: Create platform detection and testing for macOS, Linux, Windows
5. **Implement Dual CLI Compatibility**: Ensure secrets encrypted by TypeScript CLI can be decrypted by GoLang CLI and vice versa
6. **Add Security Hardening**: Implement secure memory handling, file permission enforcement, and error handling

## Implementation Notes
**Go-Native Library Requirement:**
Per independence requirements [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55], MUST use pure Go libraries only.

**Recommended Library:**
- `github.com/zalando/go-keyring` - Cross-platform OS keyring integration without CGO [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Recommended Go Libraries]

**Critical Implementation Requirements:**
1. **OS Keyring Integration**: Use only OS-native keyring services, no custom encryption schemes [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Critical Implementation Notes]
2. **Backward Compatibility**: Must read and write secrets compatible with existing TypeScript CLI format [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-STORAGE-012/epic.md - Critical Implementation Notes]
3. **Pure Go Implementation**: No CGO dependencies for portability [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L40-L55]

**Proposed File Structure:**
```
golang-cli/
└── internal/
    └── storage/
        ├── secrets.go          # OS keyring integration, encrypt/decrypt
        ├── file_storage.go     # Atomic file operations (from FILE-001)
        ├── state_manager.go    # In-memory cache with debounced writes
        └── paths.go            # Cross-platform path resolution
```

**Dependency on FILE-001:**
This feature depends on FEAT-INFRA-STORAGE-012-FILE-001 for atomic file operations when writing encrypted secrets to disk. The encryption layer should wrap the file storage layer, ensuring encrypted data is written atomically.

**Integration with Authentication:**
EPIC-DEV-AUTH-004 (Authentication & Provider Configuration) depends on this feature for secure storage of OAuth tokens and API keys. The secrets module must provide a clean API for storing and retrieving provider credentials.

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and context
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Citation Verification checklist completed