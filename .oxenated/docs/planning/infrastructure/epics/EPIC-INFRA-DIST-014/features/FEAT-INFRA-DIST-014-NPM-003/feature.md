# NPM Wrapper Package

## Feature ID
FEAT-INFRA-DIST-014-NPM-003

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Epic 14 Feature 3]

## Epic Context
**Parent Epic:** EPIC-INFRA-DIST-014 - Distribution & Packaging [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Enable broad adoption of the Cline CLI by providing easy installation across all major platforms through familiar package managers, delivering a true single-binary distribution without Node.js runtime dependency. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]
**Business Impact:** Broader adoption due to easier installation, better CI/CD integration for enterprises, reduced resource overhead, cross-platform consistency. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Feature Overview
**Purpose:** Create an npm wrapper package that downloads the correct platform-specific Go binary during installation, allowing Node.js ecosystem users to install the GoLang CLI via `npm install -g cline` while maintaining zero Node.js runtime dependency for the actual CLI execution. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Feature 3 description]
**Scope:** 
- **Included:** npm package structure, postinstall binary download script, platform detection logic, binary placement in PATH
- **Excluded:** Actual Go binary compilation (handled by FEAT-INFRA-DIST-014-BUILD-001), Homebrew formula (handled by FEAT-INFRA-DIST-014-HOMEBREW-002)

**PRD References:** REQ-015 (Homebrew and npm distribution) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Traceability Matrix]
**PRD Feature ID:** EPIC-INFRA-DIST-014-NPM-003
**Dependencies:** 
- FEAT-INFRA-DIST-014-BUILD-001 (Cross-platform build scripts must produce binaries)
- FEAT-INFRA-DIST-014-INDEPENDENCE-001 (Independence verification must pass)
- Proposed: Release artifacts hosted on GitHub Releases or similar CDN

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Feature IAOOI section]

**Inputs:**
- Platform binaries for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- npm package.json template with metadata
- Target platform info from `process.platform` and `process.arch` during installation
- Binary download URLs from release artifacts
- Checksums for binary verification

**Activities:**
- Create npm package structure separate from golang-cli directory
- Implement postinstall script to detect platform (`process.platform`, `process.arch`)
- Download correct binary from release URLs based on platform detection
- Verify binary checksums after download
- Place binary in accessible location (node_modules/.bin or global bin)
- Set executable permissions on Unix systems
- Handle installation errors gracefully with helpful messages

**Outputs:**
- Published npm package in npm registry
- Installed cline binary available in PATH after `npm install -g cline`
- Platform-specific binary downloaded and ready for execution
- Installation success/failure feedback

**Outcomes:**
- Node.js ecosystem users can install GoLang CLI via familiar `npm install -g cline` command
- No Node.js runtime required for CLI execution (pure Go binary)
- Binary is immediately available in PATH after installation
- Installation works on all supported platforms (Linux, macOS, Windows)

**Impacts:**
- Familiar installation path for JavaScript/Node.js developers
- Seamless integration into Node.js-based CI/CD pipelines
- Reduced friction for adoption in JavaScript-heavy organizations
- Consistent installation experience across platforms via npm

## Technical Requirements
**Architecture Layer:** Distribution/Packaging Infrastructure
**Integration Points:** 
- Proposed: GitHub Releases API for binary downloads (or similar artifact hosting)
- Proposed: npm registry for package publishing
- Proposed: Platform detection via Node.js `process` module [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - BDD Scenario: Platform detection]

**Data Requirements:** 
- Existing: Release versioning scheme from epic [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]
- Proposed: Binary manifest mapping platforms to download URLs and checksums
- Proposed: npm package metadata (name, version, description, keywords)

**Performance Requirements:** 
- Binary download should complete within 30 seconds on broadband (per epic success metrics) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]
- Installation time under 30 seconds total (including download) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]
- Support resumable downloads or caching for interrupted installations

**Security Requirements:** 
- Verify binary checksums after download to prevent tampering
- Use HTTPS for all binary downloads
- npm package must not contain embedded secrets or credentials
- Postinstall script should validate binary integrity before installation

## User Experience
**User Personas:** 
- Developer User (Individual Developer) - familiar with npm
- DevOps/Automation User (CI/CD) - uses npm in pipelines
- Enterprise User - may have npm registry proxies

**User Actions:**
1. Run `npm install -g cline` to install globally
2. Run `npx cline` for one-off usage without global install
3. Use in CI/CD: `npm install -g cline && cline --json "task"`
4. Verify installation: `cline version`

**UI Components:** 
- Proposed: Console output during postinstall (download progress, platform detection)
- Proposed: Error messages for unsupported platforms
- Proposed: Success confirmation after binary download

**CI/CD Considerations:** 
- Non-interactive installation must work without prompts
- Exit codes must indicate success/failure clearly
- Should work in containerized environments (Docker, Kubernetes)
- Must respect `npm_config_global` and other npm environment variables

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Feature BDD Scenarios]

```gherkin
Scenario: Install via npm
  Given npm package is published
  When user runs "npm install -g cline"
  Then correct binary for platform should download
  And cline should be available in PATH

Scenario: Platform detection in npm install
  Given npm postinstall script runs
  When detecting platform
  Then correct binary URL should be selected
  Based on process.platform and process.arch
```

**Additional Scenarios to Consider:**

```gherkin
Scenario: Install on macOS ARM64 (Apple Silicon)
  Given user is on macOS with ARM64 architecture
  When running "npm install -g cline"
  Then the darwin-arm64 binary should download
  And binary should be executable

Scenario: Install on Windows AMD64
  Given user is on Windows with AMD64 architecture
  When running "npm install -g cline"
  Then the windows-amd64 binary should download
  And cline.exe should be available in PATH

Scenario: Install on Linux AMD64
  Given user is on Linux with AMD64 architecture
  When running "npm install -g cline"
  Then the linux-amd64 binary should download
  And binary should have executable permissions

Scenario: Verify binary checksum
  Given binary download completes
  When postinstall script verifies checksum
  Then installation should proceed only if checksum matches
  And error should display if verification fails

Scenario: Handle unsupported platform gracefully
  Given user is on unsupported platform (e.g., FreeBSD)
  When running npm install
  Then a clear error message should display
  And suggest alternative installation methods

Scenario: npx execution without global install
  Given cline is not installed globally
  When user runs "npx cline 'task'"
  Then binary should download to npx cache
  And task should execute
```

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Traceability Matrix and Success Metrics]

**Functional:**
- `npm install -g cline` successfully downloads and installs the correct platform binary [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - BDD Scenario]
- Binary is immediately available in PATH after installation
- Platform detection correctly identifies `process.platform` and `process.arch` [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - BDD Scenario]
- All 5 target platform/arch combinations work correctly (Linux AMD64/ARM64, macOS AMD64/ARM64, Windows AMD64) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]

**Performance:**
- Installation time under 30 seconds on broadband connection [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]
- Binary download completes within 30 seconds [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]

**Quality:**
- Binary checksum verification passes
- No Node.js runtime errors during installation
- Graceful error handling with helpful messages
- Zero dependencies on Node.js for CLI execution (pure Go binary)

**Integration:**
- Works seamlessly with npm registry
- Compatible with npm registry mirrors/proxies
- Works in CI/CD environments without interaction
- Independence verification passes (no JS/TS code in downloaded binary) [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md:L1]

**Business Value:**
- Provides familiar installation path for Node.js developers
- Enables CI/CD integration without additional tooling
- Maintains zero Node.js runtime dependency (key migration goal)

## Testing Strategy
**Unit Testing:**
- Platform detection logic tests (mock `process.platform` and `process.arch`)
- Binary URL selection tests for all platform combinations
- Checksum verification tests
- Error handling tests for network failures

**Integration Testing:**
- Test npm install in clean Docker containers for each platform
- Verify binary execution after installation
- Test with npm registry mirrors
- Verify independence (no Node.js required to run installed binary)

**User Acceptance:**
- Manual test on macOS (Intel and Apple Silicon)
- Manual test on Windows
- Manual test on Linux (AMD64 and ARM64)
- Test in CI/CD pipeline scenarios

**Performance Testing:**
- Measure installation time across different network conditions
- Verify download resumption/caching behavior
- Test concurrent installations

## Tasks Overview
- **Task 1:** Create npm package directory structure and package.json
- **Task 2:** Implement platform detection utility (process.platform/arch mapping)
- **Task 3:** Implement binary download script with progress reporting
- **Task 4:** Implement checksum verification logic
- **Task 5:** Implement binary installation and PATH setup
- **Task 6:** Create error handling and user messaging
- **Task 7:** Create CI/CD test harness for npm install scenarios
- **Task 8:** Document npm installation in README and CLI docs

## Implementation Notes
**Separation from TypeScript CLI:**
- The npm wrapper for the GoLang CLI MUST be a completely separate package from the existing TypeScript CLI in `cli/` directory [Source: cli/package.json:L1 - Existing CLI package]
- The existing CLI uses `"name": "cline"` at version `2.9.0` [Source: cli/package.json:L2-L3]
- The GoLang CLI npm wrapper should use a distinct package name or coordinate version handoff strategy

**Package Naming Considerations:**
- Option A: Use scoped package `@cline/cli` for Go version
- Option B: Coordinate major version bump with migration announcement
- Option C: Use `cline-go` temporarily during transition period

**Existing npm Packaging Script:**
- There is an existing npm packaging script at `scripts/package-npm.mjs` [Source: scripts/package-npm.mjs:L1-L10]
- This script currently packages the TypeScript CLI from `cli/` directory
- It will need to be replaced or updated to package the Go binary wrapper instead

**Directory Structure Proposed:**
```
npm-package/
├── package.json           # npm package metadata
├── postinstall.js         # Binary download and installation script
├── lib/
│   ├── platform.js        # Platform detection utilities
│   ├── download.js        # Binary download with progress
│   └── verify.js          # Checksum verification
├── bin/
│   └── cline              # Stub script that calls downloaded binary
└── README.md              # Installation instructions
```

**Platform Mapping:**
| process.platform | process.arch | Binary Target |
|------------------|--------------|---------------|
| darwin           | arm64        | cline-darwin-arm64 |
| darwin           | x64          | cline-darwin-amd64 |
| linux            | arm64        | cline-linux-arm64 |
| linux            | x64          | cline-linux-amd64 |
| win32            | x64          | cline-windows-amd64.exe |

**Independence Verification:**
- The downloaded binary MUST pass the same independence verification as FEAT-INFRA-DIST-014-INDEPENDENCE-001 [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]
- npm postinstall script should NOT bundle any TypeScript/JavaScript code from `cli/src/`
- The wrapper should be a thin download/install mechanism only

**Dependency on Release Artifacts:**
- This feature requires release binaries to be published to a URL-accessible location (GitHub Releases recommended)
- Binary URLs and checksums should be versioned and immutable
- Consider using a manifest file that maps versions to binary URLs

## Citation Verification
- [x] All PRD content includes source line numbers (PRD uses L1-L1 as it's a single document section)
- [x] Existing code references cite actual file paths and lines
  - [x] cli/package.json:L1-L3 - Existing CLI package metadata
  - [x] scripts/package-npm.mjs:L1-L10 - Existing npm packaging script
- [x] New functionality clearly marked as "Proposed:"
  - [x] Proposed: GitHub Releases API for binary downloads
  - [x] Proposed: npm package directory structure
  - [x] Proposed: Binary manifest mapping
- [x] Integration points cite existing interfaces or mark as new
  - [x] Existing: Release versioning scheme from epic
  - [x] Proposed: GitHub Releases API integration
- [x] Citation Verification checklist completed