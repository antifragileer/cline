# Distribution & Packaging

## Epic ID
EPIC-INFRA-DIST-014

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Epic Section]

## Target Persona
Infrastructure (Internal)

## Epic Overview
This epic covers the complete distribution and packaging infrastructure for the Cline CLI GoLang migration. It ensures the GoLang CLI can be built, packaged, and distributed across all supported platforms (Linux, macOS, Windows) through multiple distribution channels including direct binary downloads, Homebrew for macOS/Linux, and npm for Node.js ecosystem compatibility. The epic also includes critical independence verification to ensure the GoLang CLI is a true standalone implementation with zero dependencies on the existing TypeScript CLI code.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Vision & Objectives
Enable broad adoption of the Cline CLI by providing easy installation across all major platforms through familiar package managers. Deliver a true single-binary distribution that leverages Go's advantages: faster startup, smaller footprint, and better cross-platform support without requiring Node.js runtime.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## IAOOI System Components

### Inputs
- Go source code from the migrated CLI implementation
- Build targets (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Version tags and release metadata
- Platform-specific packaging requirements
- Code signing certificates
- Package manager specifications (Homebrew formula schema, npm package.json)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

### Activities
- Cross-compile Go binaries for all target platforms
- Code sign binaries for platform security requirements
- Generate Homebrew formula with platform-specific URLs and checksums
- Create npm wrapper package with postinstall binary download
- Publish packages to respective repositories
- Verify independence from existing TypeScript CLI (no embedded JS/TS, no Node.js dependencies)
- Validate standalone execution in clean environments

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

### Outputs
- Platform-specific binaries (Linux AMD64/ARM64, macOS AMD64/ARM64, Windows AMD64)
- Signed binaries for macOS and Windows
- Homebrew formula file for tap distribution
- Published npm package with platform detection
- Independence verification reports
- Release artifacts with checksums

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

### Outcomes
- Users can install Cline CLI via `brew install cline` on macOS/Linux
- Users can install via `npm install -g cline` for Node.js ecosystem familiarity
- Direct binary downloads available for all platforms
- Zero Node.js runtime dependency for end users
- Professional distribution appearance comparable to modern CLI tools

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

### Impacts
- Broader adoption due to easier installation across platforms
- Better CI/CD integration for enterprises (single binary, no Node.js requirement)
- Reduced resource overhead compared to Node.js-based CLI
- Cross-platform consistency in behavior and installation experience
- Foundation for future native integrations and partnerships

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Key Features
- FEAT-INFRA-DIST-014-IND-01: Independence Verification - Verify no JavaScript/TypeScript code, no npm dependencies, no Node.js runtime requirements
- FEAT-INFRA-DIST-014-BLD-01: Cross-platform Build Scripts - Compile for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) with code signing
- FEAT-INFRA-DIST-014-HBR-01: Homebrew Formula - Generate and publish Homebrew formula for macOS/Linux installation
- FEAT-INFRA-DIST-014-NPM-01: NPM Wrapper Package - Create npm package that downloads correct platform binary

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Business Value & Requirements
This epic addresses the following original requirements:
- REQ-014: Cross-platform distribution (Linux, macOS, Windows)
- REQ-015: Homebrew and npm distribution

Additional requirements covered:
- REQ-019: No dependencies on existing TypeScript CLI code (via Independence Verification)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## User Journeys & Scenarios

### Developer Installation Journey
1. Developer discovers Cline CLI and wants to install
2. On macOS/Linux: Runs `brew install cline` or `npm install -g cline`
3. On Windows: Downloads binary or uses npm install
4. Binary automatically downloaded for correct platform/architecture
5. CLI available in PATH immediately after installation
6. No Node.js setup or version management required

### DevOps CI/CD Integration Journey
1. DevOps engineer needs Cline CLI in CI pipeline
2. Uses npm install for consistency with Node.js projects
3. Or downloads direct binary for minimal dependencies
4. CLI works immediately without runtime setup
5. Single binary simplifies container image creation

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## BDD Scenarios

### Independence Verification Feature

```gherkin
Scenario: Verify no JavaScript code in binary
  Given the GoLang CLI binary is built
  When scanning for embedded JS/TS code
  Then no JavaScript or TypeScript code should be found
  And no Node.js runtime references should exist

Scenario: Verify no npm dependencies
  Given the GoLang CLI project
  When examining go.mod and imports
  Then no npm packages should be referenced
  And no Node.js modules should be required

Scenario: Verify standalone execution
  Given a clean environment without Node.js
  When running the GoLang CLI binary
  Then it should execute successfully
  And all features should work without Node.js
```

### Cross-platform Build Scripts Feature

```gherkin
Scenario: Build for Linux AMD64
  Given Go source code
  When build script runs with GOOS=linux GOARCH=amd64
  Then binary should compile
  And be executable on Linux AMD64

Scenario: Build for macOS ARM64
  Given Go source code
  When build script runs with GOOS=darwin GOARCH=arm64
  Then binary should compile
  And be executable on Apple Silicon Macs

Scenario: Build for Windows
  Given Go source code
  When build script runs with GOOS=windows
  Then binary should compile with .exe extension
```

### Homebrew Formula Feature

```gherkin
Scenario: Generate Homebrew formula
  Given new release binaries
  When formula generation runs
  Then formula should include URLs for all platforms
  And checksums should be correct

Scenario: Install via Homebrew
  Given Homebrew tap is configured
  When user runs "brew install cline"
  Then cline should install correctly
  And be available in PATH
```

### NPM Wrapper Package Feature

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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Technical Considerations

### Existing Code References
- Current CLI location: `cli/` directory contains TypeScript/React Ink implementation
- New Go CLI location: `golang-cli/` directory (existing structure visible)
- Build scripts: `scripts/build-cli-artifact.sh` [UNVERIFIED - requires confirmation]
- Current npm package: `cli/package.json` [Source: cli/package.json:L1]

### Proposed New Components
- Proposed: `golang-cli/scripts/build-cross-platform.sh` - Cross-platform build automation
- Proposed: `golang-cli/scripts/generate-homebrew-formula.sh` - Homebrew formula generation
- Proposed: `golang-cli/scripts/verify-independence.sh` - Independence verification checks
- Proposed: `npm-package/` directory for npm wrapper (separate from golang-cli)
- Proposed: `.github/workflows/release.yml` - GitHub Actions release workflow

### Cross-Platform Build Requirements
- Linux: AMD64, ARM64 (for cloud and Raspberry Pi deployments)
- macOS: AMD64 (Intel), ARM64 (Apple Silicon)
- Windows: AMD64 (primary desktop architecture)
- Code signing required for macOS (notarization) and Windows (certificate)

### Package Manager Integration
- Homebrew: Requires tap repository or official homebrew-core submission
- NPM: Wrapper package downloads binary during postinstall based on `process.platform` and `process.arch`

### Independence Verification Strategy
- Static analysis of Go binary to detect embedded JavaScript
- Dependency graph analysis to ensure no npm/Node.js imports
- CI/CD test in clean Docker container without Node.js
- Compare binary size and composition against baseline

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Implementation Priority
This epic is scheduled for **Phase 8** (final phase) of the GoLang migration. It depends on:
- All core functionality being complete and tested (Phases 1-7)
- Final binaries being ready for packaging
- Version tagging and release process defined

The epic can be worked on in parallel with final testing phases, but binaries must be frozen before final packaging.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Success Metrics
- Binaries successfully built for all 5 target platform/arch combinations
- Homebrew formula installs correctly on macOS (Intel and Apple Silicon)
- Homebrew formula installs correctly on Linux (AMD64 and ARM64)
- npm install works on all platforms without Node.js runtime errors
- Independence verification passes (no JS/TS code detected, runs without Node.js)
- Binary size under 50MB for all platforms
- Installation time under 30 seconds on broadband connection

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Dependencies
- Completion of all Phase 1-7 epics (core functionality)
- Build system and CI/CD infrastructure
- Code signing certificates for macOS and Windows
- Access to npm registry for package publishing
- Homebrew tap repository setup (if using custom tap)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Integration Points
- **Integration with CI/CD**: Release workflow triggers after successful dual testing
- **Integration with Core Extension**: gRPC/protobuf compatibility ensures CLI works with existing core
- **Integration with State Storage**: CLI uses same `~/.cline/data/` directory as existing implementation
- **Integration with Existing CLI**: npm package may conflict with existing `cli/` package, needs coordination

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]