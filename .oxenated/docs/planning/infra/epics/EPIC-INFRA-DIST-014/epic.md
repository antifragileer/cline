# Distribution & Packaging

## Epic ID
EPIC-INFRA-DIST-014

## Source Reference
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1180 - Epic 14: Distribution & Packaging]

## Target Persona
Infrastructure (Internal)

## Epic Overview
This epic covers the complete distribution and packaging infrastructure for the GoLang Cline CLI. It includes cross-platform build scripts for Linux, macOS, and Windows (AMD64 and ARM64), binary signing, Homebrew formula generation, and npm wrapper package creation. A critical component is the independence verification system that ensures the GoLang CLI contains no embedded JavaScript/TypeScript code and has zero dependencies on the existing TypeScript CLI. The goal is to achieve easy installation across all platforms through familiar package managers while maintaining a professional, enterprise-ready distribution model.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1117]

## Vision & Objectives
Leverage Go's advantages to deliver single binary distribution, faster startup, smaller footprint, and better cross-platform support. The migrated CLI will maintain full compatibility with existing state storage (~/.cline/data/) and the existing Cline core extension via gRPC/protobuf, while achieving broader adoption through easier installation and better CI/CD integration for enterprises.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L15]

## IAOOI System Components

### Inputs
- Go source code from the golang-cli/ directory
- Build targets (Linux, macOS, Windows)
- Architecture targets (AMD64, ARM64)
- Version tags and release information
- Package manager requirements (Homebrew, npm, scoop)
- Existing CLI comparison requirements for independence verification

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1108]

### Activities
- Cross-compile binaries for all supported platforms (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Binary signing for security and verification
- Generate and publish Homebrew formula to tap
- Create npm wrapper package with platform-specific binary downloads
- Verify binary contains no embedded JavaScript/TypeScript code
- Validate no Node.js dependencies exist in the binary
- Confirm single static binary output for all platforms
- Create independence verification reports

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1108, L1118-L1180]

### Outputs
- Platform-specific binaries (Linux, macOS, Windows; AMD64, ARM64)
- Signed binaries for security verification
- Homebrew formula file for macOS/Linux installation
- Published npm package for Node.js ecosystem users
- Independence verification report
- Installable packages for all supported platforms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1108]

### Outcomes
- Users can install Cline CLI via `brew install cline` on macOS/Linux
- Users can install via `npm install -g cline` for familiar Node.js workflow
- Single binary distribution without Node.js dependency
- Professional, enterprise-ready distribution model
- Easy installation across all platforms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1104-L1108, L1125-L1129, L1154-L1157]

### Impacts
- Broader adoption due to easier installation
- Better CI/CD integration for enterprises
- Reduced resource overhead
- Cross-platform consistency
- Foundation for future native integrations
- Professional appearance and enterprise credibility

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L44-L48, L1109-L1113]

## Key Features
- FEAT-INFRA-DIST-014-INDEPENDENCE-001: Independence Verification - Verify binary contains no embedded JavaScript/TypeScript code and has zero dependencies on existing TypeScript CLI
- FEAT-INFRA-DIST-014-BUILD-001: Cross-platform Build Scripts - Compile for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64), with binary signing
- FEAT-INFRA-DIST-014-HOMEBREW-002: Homebrew Formula - Generate/update Homebrew formula and publish to tap for macOS/Linux installation
- FEAT-INFRA-DIST-014-NPM-003: NPM Wrapper Package - Create npm package that downloads correct binary for platform with postinstall script

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1118-L1157]

## Business Value & Requirements
This epic directly addresses the following requirements:
- REQ-014: Cross-platform distribution (Linux, macOS, Windows)
- REQ-015: Homebrew and npm distribution

It also supports the critical independence requirement:
- REQ-019: No dependencies on existing TypeScript CLI code (verified via FEAT-INFRA-DIST-014-INDEPENDENCE-001)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1274-L1282, L1286]

## User Journeys & Scenarios

### Developer Installation Journey
1. Developer discovers Cline CLI and wants to install
2. On macOS/Linux: Runs `brew install cline`
3. On Windows or with Node.js: Runs `npm install -g cline`
4. Binary downloads automatically for correct platform
5. CLI is available in PATH and ready to use

### CI/CD Integration Journey
1. DevOps engineer wants to use Cline in CI pipeline
2. Uses npm install for consistency with existing Node.js projects
3. Or downloads binary directly for minimal dependencies
4. Binary runs without Node.js runtime requirement
5. Scriptable automation works reliably across platforms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L62-L73, L74-L84]

## BDD Scenarios

### Independence Verification
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

### Cross-platform Build
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

### Homebrew Distribution
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

### NPM Distribution
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

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1118-L1180]

## Technical Considerations

### Existing Code References
- Existing CLI packaging scripts: `cli/scripts/` directory [Source: cli/scripts/ - packaging and distribution scripts]
- Existing Homebrew formula: `cli/cline.rb` [Source: cli/cline.rb:L1-L40]
- Existing npm package structure: `cli/package.json` [Source: cli/package.json:L1-L50]
- Build configuration: `cli/esbuild.mts` [Source: cli/esbuild.mts:L1-L30]

### Proposed New Components
- **Proposed:** `golang-cli/scripts/build.sh` - Cross-platform build script with GOOS/GOARCH matrix
- **Proposed:** `golang-cli/scripts/sign.sh` - Binary signing script for security
- **Proposed:** `golang-cli/scripts/verify-independence.sh` - Scans binary for JS/TS code and Node.js references
- **Proposed:** `golang-cli/scripts/generate-homebrew-formula.sh` - Generates Homebrew formula from release binaries
- **Proposed:** `golang-cli/npm/` - NPM wrapper package directory with package.json and postinstall script
- **Proposed:** `golang-cli/.goreleaser.yml` - GoReleaser configuration for automated releases
- **Proposed:** GitHub Actions workflow for release automation

### Build Architecture
The GoLang CLI must produce a single static binary for each platform/architecture combination. No shared libraries or external dependencies should be required at runtime. The build process should:

1. Compile with CGO_ENABLED=0 for static linking
2. Use ldflags to embed version information
3. Strip debug symbols for smaller binaries
4. Sign binaries with appropriate certificates per platform
5. Generate checksums for verification

### Independence Verification Architecture
The independence verification is a critical security and compliance requirement:

1. **Static Analysis**: Scan source code for any imports from `cli/src/` or references to npm packages
2. **Binary Analysis**: Scan compiled binary for embedded JavaScript/TypeScript strings or Node.js runtime references
3. **Dependency Verification**: Ensure go.mod contains only Go modules, no Node.js dependencies
4. **Execution Test**: Run binary in clean environment without Node.js installed
5. **CI/CD Integration**: Independence verification runs on every build before release

### Distribution Channels
1. **GitHub Releases**: Primary distribution with binaries for all platforms
2. **Homebrew Tap**: `brew install cline` for macOS/Linux users
3. **NPM Registry**: `npm install -g cline` for Node.js ecosystem users
4. **Scoop (Future)**: Windows package manager support (post-MVP)
5. **AUR (Future)**: Arch Linux user repository (post-MVP)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1118-L1180, L1375-L1383]

## Implementation Priority
This epic is scheduled for **Phase 8** of the GoLang migration plan - the final phase before release. It depends on all other epics being complete, as it requires:

- Functional CLI implementation (Phases 1-7)
- Independence verification needs the complete binary to scan
- Build scripts need all source code available
- Package managers need working binaries to distribute

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1406-L1414]

## Success Metrics
- Binaries successfully build for all 5 platform/architecture combinations (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Binary size under 50MB per platform (compressed)
- Installation success rate >95% via Homebrew and npm
- Independence verification passes with zero JS/TS/Node.js references
- Binary startup time <100ms on all platforms

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1109-L1113, L1120-L1123]

## Dependencies
This epic depends on:
- EPIC-INFRA-CORE-011: Core Extension Integration (gRPC communication must work before distribution)
- EPIC-INFRA-STORAGE-012: State & Storage Layer (state persistence must function)
- All other epics providing functional CLI features

No other epics depend on this one; it is the final delivery phase.

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1406-L1414]

## Integration Points
- Integrates with existing Cline core extension via gRPC (protobuf definitions in `proto/` directory)
- Uses existing state storage at `~/.cline/data/` for seamless migration
- Compatible with existing VSCode extension state (users can switch between CLI implementations)
- NPM package integrates with Node.js ecosystem for familiar installation experience
- Homebrew integrates with macOS/Linux package management standards

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L6-L11, L1517-L1521]

## Dual Testing Requirements
Per the dual testing mandate, all distribution features must be tested against the existing TypeScript CLI:

1. **Installation Parity**: Both CLIs must install successfully via their respective package managers
2. **Version Output**: `cline version` must produce identical output format (excluding version numbers)
3. **Help Output**: `cline --help` must produce identical command and flag documentation
4. **State Compatibility**: Tasks created in one CLI must be resumable in the other (via shared `~/.cline/data/`)

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1319-L1383]

## Independence Verification Checklist
Before this epic completes, verify:
- [ ] No imports from `cli/src/` or `cli/package.json`
- [ ] No Node.js runtime dependencies
- [ ] No TypeScript/JavaScript code bundled in binary
- [ ] All dependencies are pure Go modules
- [ ] Build produces single static binary
- [ ] Binary runs in clean environment without Node.js

[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1426-L1432]