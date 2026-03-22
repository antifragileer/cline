# Authentication & Provider Configuration

## Epic ID
EPIC-DEV-AUTH-004

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L463 - Epic 4: Authentication & Provider Configuration]

## Target Persona
Developer User

## Epic Overview
This epic encompasses all authentication and provider configuration functionality for the Cline CLI GoLang migration. It provides secure credential management, OAuth authentication flows, API key handling, and provider configuration wizards. The epic ensures users can securely authenticate with various AI providers (OpenAI, Anthropic, OpenRouter, etc.) while maintaining enterprise-grade security through encrypted secrets storage and flexible configuration options.

The authentication system must support both interactive wizard-based setup for new users and quick command-line configuration for automation scenarios. All credentials must be stored securely using OS-native keyring integration with fallback to encrypted file storage.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]

## Vision & Objectives
Enable secure, flexible authentication with multiple AI providers while maintaining a seamless user experience. The system should support both individual developers and enterprise users with varying security requirements, providing easy provider switching and robust credential protection.

**Core Value Proposition:**
- Secure credential storage with OS-level encryption
- Multiple authentication methods (OAuth, API keys)
- Easy provider configuration and switching
- Enterprise-ready security compliance

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]

## IAOOI System Components

### Inputs
1. API keys provided via command-line flags (`-k`, `--key`)
2. API keys provided via interactive secure input prompts
3. OAuth provider selection and authorization codes
4. Provider configuration settings (provider ID, model preferences)
5. Existing credentials from `~/.cline/data/secrets.json`
6. OS keyring integration for encryption keys
7. Environment variables for configuration overrides
8. User preferences for default providers and models

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L400-L403]

### Activities
1. **Secure Credential Storage**: Encrypt and store API keys using OS-native keyring or file-based encryption
2. **OAuth Flow Management**: Initiate OAuth flows, start local callback servers, capture and exchange tokens
3. **API Key Validation**: Test API keys against provider endpoints to verify validity before storage
4. **Provider Configuration**: Save provider settings including default models and preferences
5. **Interactive Wizard**: Guide users through provider selection, authentication method, and configuration
6. **Quick Configuration**: Support non-interactive setup via command-line flags for automation
7. **Credential Retrieval**: Decrypt and retrieve stored credentials for API requests
8. **Configuration Persistence**: Save and load provider configurations from `~/.cline/data/globalState.json`
9. **Model Discovery**: Fetch available models from configured providers
10. **Provider Switching**: Enable seamless switching between configured providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L401-L403]

### Outputs
1. Encrypted credentials stored in `~/.cline/data/secrets.json`
2. Provider configurations in `~/.cline/data/globalState.json`
3. Authenticated sessions with valid API tokens
4. Available models lists for each configured provider
5. Validation success/failure status for credentials
6. OAuth tokens with refresh capabilities (where supported)
7. Configuration confirmation messages
8. Error messages for invalid credentials or connection failures

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L402-L404]

### Outcomes
1. Users can securely authenticate with any supported AI provider
2. Credentials are protected with enterprise-grade encryption
3. Provider switching is seamless and immediate
4. New users can configure authentication in under 60 seconds
5. Automation users can configure providers without interactive prompts
6. No plaintext credentials are stored or logged
7. Authentication state persists across CLI invocations
8. Users can view and manage their configured providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L403-L405]

### Impacts
1. **Enterprise Adoption**: Secure credential handling meets enterprise security requirements
2. **Security Compliance**: Encryption and secure storage satisfy compliance standards (SOC2, ISO 27001)
3. **User Trust**: Robust security practices build user confidence
4. **Operational Efficiency**: Quick provider setup reduces time-to-first-task
5. **Automation Enablement**: Non-interactive configuration supports CI/CD and scripting workflows
6. **Multi-Provider Flexibility**: Easy switching enables cost optimization and feature access across providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]

## Key Features
- EPIC-DEV-AUTH-004-OAUTH-001: OAuth Authentication Flows - Browser-based OAuth with local callback server
- EPIC-DEV-AUTH-004-KEY-002: API Key Management - Secure storage and validation of API keys
- EPIC-DEV-AUTH-004-PROV-003: Provider Configuration Wizard - Interactive setup and quick configuration via flags

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L463]

## Business Value & Requirements
This epic directly addresses the following original requirements:

| Requirement ID | Description | Coverage |
|----------------|-------------|----------|
| REQ-009 | Support all existing API providers (OpenAI, Anthropic, OpenRouter, etc.) | Full - All providers supported via unified authentication interface |
| REQ-016 | Configuration management (global and workspace) | Full - Provider configs stored in global state with model preferences |
| REQ-017 | Secure secrets storage | Full - OS keyring integration with encrypted file fallback |

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L406-L407]

## User Journeys & Scenarios

### Primary User Journey: First-Time Setup
1. User installs Cline CLI and runs `cline auth`
2. Interactive wizard displays available providers
3. User selects provider (e.g., Anthropic)
4. User chooses authentication method (OAuth or API key)
5. For API key: Secure prompt collects key, validation confirms validity
6. User selects default model from available options
7. Configuration saved, confirmation displayed
8. User can immediately start using `cline "prompt"`

### Automation User Journey: CI/CD Configuration
1. DevOps engineer exports `CLINE_API_KEY` environment variable
2. Runs `cline auth -p anthropic -k $CLINE_API_KEY -m claude-sonnet`
3. No interactive prompts, immediate configuration
4. Configuration persists for subsequent CLI calls
5. Scripting and automation workflows proceed without human interaction

### Provider Switching Journey
1. User has multiple providers configured
2. Runs `cline auth` to view configured providers
3. Selects different provider to activate
4. Model list updates automatically
5. Next task uses newly selected provider

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463 - Feature Gherkin scenarios]

## BDD Scenarios

### EPIC-DEV-AUTH-004-OAUTH-001: OAuth Authentication Flows

```gherkin
Scenario: OAuth authentication flow
  Given the user selects OAuth provider
  When authentication starts
  Then a browser should open for authorization
  And local callback server should start
  And token should capture and store securely

Scenario: Handle OAuth callback
  Given OAuth flow is in progress
  When the callback receives authorization code
  Then token should exchange
  And configuration should update
```

### EPIC-DEV-AUTH-004-KEY-002: API Key Management

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

### EPIC-DEV-AUTH-004-PROV-003: Provider Configuration Wizard

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]

## Technical Considerations

### Existing Code References
This epic interfaces with the existing Cline core extension via gRPC for provider configuration updates. The storage layer uses the existing file-based storage system:

- State storage: `~/.cline/data/globalState.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
- Secrets storage: `~/.cline/data/secrets.json` (mode 0o600) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L995-L1011]
- gRPC integration for provider state updates: EPIC-INFRA-CORE-011

### Proposed New Components

**Proposed: OAuth Callback Server**
- Local HTTP server listening on ephemeral port (localhost:0)
- Handles OAuth authorization code callback
- Secure token exchange with provider endpoints
- Auto-shutdown after token capture or timeout

**Proposed: OS Keyring Integration**
- macOS: Keychain Services via `security` command or CGO bindings
- Linux: Secret Service API (D-Bus) or `pass` integration
- Windows: Windows Credential Manager via Win32 API
- Fallback: AES-256-GCM encrypted file storage with key derived from machine fingerprint

**Proposed: Provider Validation**
- HTTP client for testing API keys against provider endpoints
- Validation before storage to prevent invalid configurations
- Provider-specific endpoint detection and testing

**Proposed: Secure Input Components**
- Bubble Tea-based secure input (masked characters)
- Support for pasting multi-line keys
- Visual feedback for input validation

### Go Dependencies (Proposed)
- `github.com/zalando/go-keyring` - Cross-platform keyring access
- `golang.org/x/oauth2` - OAuth2 flow implementation
- Standard library: `crypto/aes`, `crypto/cipher` for fallback encryption

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463]

## Implementation Priority
**Priority: HIGH**

This epic is foundational for all AI functionality. Without authentication, users cannot execute tasks. It should be implemented in Phase 4 alongside core task management, as it enables the gRPC communication with AI providers.

**Recommended Sequence:**
1. EPIC-DEV-AUTH-004-KEY-002 (API Key Management) - Quick win, most common use case
2. EPIC-DEV-AUTH-004-PROV-003 (Provider Configuration Wizard) - User experience
3. EPIC-DEV-AUTH-004-OAUTH-001 (OAuth Authentication Flows) - Complex, provider-specific

**Dependencies:**
- Requires EPIC-INFRA-STORAGE-012 (State & Storage Layer) for secrets persistence
- Blocks EPIC-INFRA-API-013 (API Provider Integrations) - cannot call APIs without auth
- Blocks EPIC-DEV-TASK-003 (Task Management) - tasks require authenticated provider

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175 - AI Execution Plan]

## Success Metrics
- **Setup Time**: New users can authenticate and run first task in under 60 seconds
- **Validation Rate**: >95% of entered API keys pass validation on first attempt
- **Security Audit**: No plaintext credentials in logs, memory, or files
- **Provider Coverage**: 100% of existing TypeScript CLI providers supported
- **Automation Support**: 100% of auth scenarios support non-interactive flag-based configuration

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L406]

## Dependencies
**Upstream Dependencies (This epic requires):**
- EPIC-INFRA-STORAGE-012: State & Storage Layer - Required for secrets.json and globalState.json persistence
- EPIC-INFRA-CORE-011: Core Extension Integration - Required for gRPC communication of provider changes

**Downstream Dependencies (This epic blocks):**
- EPIC-INFRA-API-013: API Provider Integrations - Cannot make API calls without authentication
- EPIC-DEV-TASK-003: Task Management - Tasks require authenticated provider to execute

**Parallel Dependencies:**
- EPIC-DEV-CLI-001: Command Line Interface Foundation - Auth commands built on CLI framework

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175]

## Integration Points
**Integration with EPIC-INFRA-STORAGE-012:**
- Reads/writes encrypted credentials to `~/.cline/data/secrets.json`
- Reads/writes provider configuration to `~/.cline/data/globalState.json`
- Uses StateManager for configuration persistence

**Integration with EPIC-INFRA-CORE-011:**
- Sends provider configuration updates to core extension via gRPC
- Receives provider availability status from core

**Integration with EPIC-DEV-CLI-001:**
- Implements `auth` subcommand with all flags
- Provides command handlers for Cobra framework

**Integration with EPIC-INFRA-API-013:**
- Provides authenticated credentials for API calls
- Shares provider configuration with API client layer

**Integration with EPIC-DEV-UI-002:**
- Welcome screen detects unauthenticated state and prompts for auth
- Provider selection UI displays available/configured providers

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L463, L995-L1011]

## Dual Testing Requirements
Per the PRD dual testing mandate, all authentication functionality MUST be tested in BOTH the existing TypeScript CLI AND the new GoLang CLI:

**Required Dual Tests:**
- [ ] API key configuration via flags comparison
- [ ] Interactive wizard flow comparison
- [ ] OAuth flow callback handling comparison
- [ ] Secrets encryption/retrieval comparison
- [ ] Provider switching behavior comparison
- [ ] Error message format comparison
- [ ] Configuration file format compatibility

**Parity Verification:**
- Output format for `cline auth` commands must match byte-for-byte (except timestamps)
- Exit codes must be identical for all scenarios
- State files must be mutually readable between CLIs

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L650-L730 - Dual Testing Strategy]

## Security Considerations
**Critical Security Requirements:**
1. Never log API keys or OAuth tokens
2. Never display secrets in terminal output (except masked)
3. Use OS keyring when available, never store keys in plaintext
4. File-based fallback must use AES-256-GCM encryption
5. Memory should zero sensitive data after use
6. OAuth callback server must bind to localhost only
7. OAuth state parameter must be cryptographically random
8. Validate TLS certificates for all OAuth and API calls

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463, L850-L930]