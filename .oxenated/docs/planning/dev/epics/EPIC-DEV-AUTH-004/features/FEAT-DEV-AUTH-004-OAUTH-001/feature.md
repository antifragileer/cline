# OAuth Authentication Flows

## Feature ID
FEAT-DEV-AUTH-004-OAUTH-001

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L463]

## Epic Context
**Parent Epic:** EPIC-DEV-AUTH-004 - Authentication & Provider Configuration [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1]
**Target Persona:** Developer User [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]
**Epic Objective:** Enable secure, flexible authentication with multiple AI providers while maintaining a seamless user experience [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L410]
**Business Impact:** Enhanced security through OAuth flows eliminates API key exposure and supports enterprise SSO requirements, building user trust and enabling enterprise adoption [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L404-L406]

## Feature Overview
**Purpose:** Implement browser-based OAuth authentication flows with local callback server for providers that support OAuth (e.g., OpenAI Codex, some OpenRouter providers). This feature eliminates the need for users to manually copy/paste API keys while maintaining enterprise-grade security through token-based authentication. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L414]

**Scope:** 
- **Included:** OAuth flow initiation, local callback server, token exchange, secure token storage, provider configuration updates
- **Excluded:** API key-based authentication (covered by FEAT-DEV-AUTH-004-KEY-002), provider discovery UI (covered by FEAT-DEV-AUTH-004-PROV-003)

**PRD References:** REQ-009 (Support all existing API providers), REQ-017 (Secure secrets storage) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L406-L407]
**PRD Feature ID:** EPIC-DEV-AUTH-004-OAUTH-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408]

**Dependencies:**
- EPIC-INFRA-STORAGE-012: State & Storage Layer - Required for persisting OAuth tokens to `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1159-L1175]
- EPIC-INFRA-CORE-011: Core Extension Integration - Required for notifying core extension of provider configuration changes via gRPC [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L1159-L1175]

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L408-L414 - OAuth Feature IAOOI]

**Inputs:**
1. Provider selection from user (OAuth-capable provider like OpenAI Codex)
2. OAuth endpoint URLs (authorization endpoint, token endpoint)
3. Client ID and client secret (if applicable for the provider)
4. Scopes/requirements for the OAuth flow
5. Callback URL configuration (localhost with ephemeral port)
6. Authorization code from browser redirect
7. Existing OAuth tokens (for refresh flows)
8. State parameter for CSRF protection

**Activities:**
1. **OAuth Flow Initiation**: Start OAuth flow by opening system browser with authorization URL including client ID, scopes, state parameter, and callback URL
2. **Local Callback Server**: Start ephemeral HTTP server on localhost (port 0 for system-assigned) to receive authorization callback
3. **Authorization Code Capture**: Handle incoming HTTP request from browser, extract authorization code and state parameter
4. **State Validation**: Verify state parameter matches the one sent to prevent CSRF attacks
5. **Token Exchange**: Exchange authorization code for access token and refresh token via provider's token endpoint
6. **Token Validation**: Test the obtained token against provider API to verify validity
7. **Secure Storage**: Encrypt and store access token and refresh token using OS keyring or file-based encryption
8. **Configuration Update**: Update provider configuration in global state with OAuth indicator
9. **Server Shutdown**: Gracefully shutdown callback server after token capture or timeout
10. **Core Extension Notification**: Send provider configuration update to core extension via gRPC

**Outputs:**
1. Encrypted OAuth tokens (access token, refresh token) stored in `~/.cline/data/secrets.json` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L402-L404]
2. Provider configuration updated in `~/.cline/data/globalState.json` with OAuth authentication method indicator
3. Authenticated session ready for API calls
4. Success/failure status message displayed to user
5. Error messages for failed flows (user denied, timeout, invalid state)
6. Token refresh capability (where provider supports it)

**Outcomes:**
1. Users can authenticate without manually handling API keys
2. Browser-based OAuth provides familiar authentication experience
3. No plaintext tokens stored or displayed
4. Automatic token refresh extends session without re-authentication (where supported)
5. Seamless integration with provider's native auth flows

**Impacts:**
1. **Security Enhancement**: OAuth tokens are short-lived and refreshable, reducing risk compared to long-lived API keys
2. **Enterprise SSO**: Supports corporate identity providers through OAuth/OIDC integration
3. **User Experience**: Single-click authentication without copy/paste of credentials
4. **Compliance**: Token-based auth satisfies stricter security compliance requirements (SOC2, ISO 27001)

## Technical Requirements

**Architecture Layer:** Application/Infrastructure
**Integration Points:**
- **Proposed:** OAuth Callback Server - Local HTTP server component [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L414-L463]
- **Existing gRPC:** Provider configuration updates via `cline.Auth` or similar service [Source: EPIC-INFRA-CORE-011]
- **Existing Storage:** `~/.cline/data/secrets.json` for token storage [Source: .clinerules/storage.md:L1-L10]

**Data Requirements:**
- **Proposed:** OAuth token schema with fields: access_token, refresh_token, expires_at, token_type
- **Existing:** Provider configuration schema in global state [Source: .clinerules/storage.md:L1-L10]

**Performance Requirements:**
- OAuth flow completion: < 60 seconds (browser open to token stored)
- Callback server startup: < 500ms
- Token exchange: < 5 seconds
- Server shutdown: immediate after token capture or timeout

**Security Requirements:**
- State parameter must be cryptographically random (minimum 32 bytes)
- State validation must reject mismatched or missing state parameters
- Callback server must bind to localhost only (127.0.0.1, not 0.0.0.0)
- TLS certificate validation required for all OAuth provider communications
- Tokens must be encrypted at rest using OS keyring or AES-256-GCM
- Memory must clear sensitive data after use
- No tokens logged or displayed in terminal output

## User Experience

**User Personas:** Developer User (individual developers), Enterprise User (corporate environments with SSO)

**User Actions:**
1. Run `cline auth` and select OAuth-capable provider
2. Browser automatically opens to provider's authorization page
3. User authenticates with provider (may include SSO/MFA)
4. User approves Cline CLI access to their account
5. Browser redirects to localhost callback
6. CLI captures token and displays success message
7. User can immediately start using authenticated commands

**UI Components:**
- **Proposed:** OAuth initiation prompt with provider selection
- **Proposed:** Browser launch notification with callback URL
- **Proposed:** Progress indicator during token exchange
- **Proposed:** Success/error message with next steps

**Mobile Considerations:** Not applicable for this CLI feature

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L414-L463 - OAuth BDD Scenarios]

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

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L395-L406]

**Functional:**
- OAuth flow completes successfully for all supported OAuth providers
- Browser opens automatically on all platforms (macOS, Linux, Windows)
- Callback server receives and processes authorization code correctly
- State parameter validates to prevent CSRF attacks
- Tokens exchange and store securely without manual user intervention

**Performance:**
- OAuth flow completes within 60 seconds under normal conditions
- Callback server responds within 1 second of browser redirect
- Token exchange completes within 5 seconds

**Quality:**
- No plaintext tokens in logs, memory dumps, or terminal output
- Graceful handling of user cancellation (browser closed, denied access)
- Proper timeout handling (callback server shuts down after 5 minutes)
- Clear error messages for all failure scenarios

**Integration:**
- Works seamlessly with existing provider configuration system
- Tokens compatible with existing API client implementations
- Provider switching works correctly with OAuth-authenticated providers

**Business Value:**
- Users can authenticate in under 60 seconds without API key copy/paste
- Supports enterprise SSO requirements for corporate adoption
- Reduces support requests related to API key configuration

## Testing Strategy

**Unit Testing:**
- OAuth URL generation with proper encoding
- State parameter generation (cryptographic randomness)
- State validation logic
- Token exchange request formatting
- Callback server request parsing

**Integration Testing:**
- End-to-end OAuth flow with mock OAuth provider
- Callback server startup/shutdown lifecycle
- Token storage and retrieval encryption
- gRPC notification to core extension

**User Acceptance:**
- Manual testing with real OAuth providers (OpenAI Codex)
- Browser integration testing on all platforms
- Timeout and error scenario validation

**Security Testing:**
- CSRF attack prevention (invalid state rejection)
- localhost binding verification (no external access)
- TLS certificate validation
- Memory clearing verification
- Log scanning for token exposure

**Dual Testing Requirements:**
Per the PRD dual testing mandate, all OAuth functionality MUST be tested in BOTH the existing TypeScript CLI AND the new GoLang CLI:
- OAuth flow initiation comparison
- Callback server handling comparison
- Token storage format compatibility
- Error message format comparison
- Configuration file format compatibility

## Tasks Overview
1. **OAuth URL Generation**: Implement authorization URL construction with state parameter
2. **Callback Server**: Implement ephemeral HTTP server for authorization code capture
3. **Token Exchange**: Implement OAuth token endpoint communication
4. **Secure Storage**: Integrate with storage layer for encrypted token persistence
5. **Browser Integration**: Implement cross-platform browser launching
6. **Error Handling**: Implement timeout, cancellation, and failure scenarios
7. **Core Extension Integration**: Send provider updates via gRPC

## Implementation Notes

**Go Dependencies (Proposed):**
- `golang.org/x/oauth2` - OAuth2 flow implementation and token management [Source: .oxenated/docs/planning/dev/epics/EPIC-DEV-AUTH-004/epic.md:L414-L463]
- Standard library: `net/http` for callback server
- Standard library: `crypto/rand` for state parameter generation

**Key Implementation Details:**
- Use `oauth2.Config` for provider configuration
- Implement custom HTTP server with timeout context
- Support PKCE (Proof Key for Code Exchange) for enhanced security where providers support it
- Handle platform-specific browser launching (`open` on macOS, `xdg-open` on Linux, `start` on Windows)

**Security Considerations:**
- State parameter must be stored temporarily (in-memory) during flow
- Callback server must validate state before exchanging code
- Server must shutdown immediately after successful capture or timeout
- All OAuth communications over HTTPS with certificate validation

**Timeout Configuration:**
- Callback server timeout: 5 minutes
- Token exchange timeout: 30 seconds
- Browser launch timeout: 10 seconds

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:"
- [x] Integration points cite existing interfaces or mark as new