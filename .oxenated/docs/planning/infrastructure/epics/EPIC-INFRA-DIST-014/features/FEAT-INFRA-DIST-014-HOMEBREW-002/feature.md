# Homebrew Formula Generation and Distribution

## Feature ID
FEAT-INFRA-DIST-014-HOMEBREW-002

## Source
**PRD Source:** [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Epic 14, Feature 2]

## Epic Context
**Parent Epic:** EPIC-INFRA-DIST-014 - Distribution & Packaging [Source: .oxenated/docs/planning/infrastructure/epics/EPIC-INFRA-DIST-014/epic.md]
**Target Persona:** Infrastructure (Internal)
**Epic Objective:** Enable broad adoption of the Cline CLI by providing easy installation across all major platforms through familiar package managers [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]
**Business Impact:** Professional distribution appearance comparable to modern CLI tools, broader adoption due to easier installation on macOS/Linux [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

## Feature Overview
**Purpose:** Enable macOS and Linux users to install the Cline CLI using Homebrew, the de facto standard package manager for macOS and popular on Linux. This feature automates the generation and publication of Homebrew formula files that reference the cross-compiled platform binaries. [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]

**Scope:** 
- **Included:** Automated formula generation with platform-specific URLs and checksums, support for macOS (Intel/Apple Silicon) and Linux (AMD64/ARM64), integration with release workflow
- **Excluded:** Windows package managers (covered separately), direct binary downloads without package manager

**PRD References:** REQ-015 (Homebrew and npm distribution) [Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1]
**PRD Feature ID:** EPIC-INFRA-DIST-014-HOMEBREW-002

**Dependencies:**
- FEAT-INFRA-DIST-014-BUILD-001: Cross-platform build scripts must produce signed binaries with checksums
- GitHub release infrastructure for hosting binaries
- Homebrew tap repository (cline/homebrew-tap or official homebrew-core submission)

## IAOOI Components
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Feature 2: Homebrew Formula]

**Inputs:**
- Release binaries for all supported platforms (macOS AMD64/ARM64, Linux AMD64/ARM64)
- Version tags and release metadata
- SHA256 checksums for each binary
- Download URLs from GitHub releases
- Homebrew formula template/schema

**Activities:**
- Generate Homebrew formula Ruby file with platform-specific `url` and `sha256` stanzas
- Calculate or retrieve SHA256 checksums for each platform binary
- Structure formula for multi-platform support (macOS Intel, macOS Apple Silicon, Linux AMD64, Linux ARM64)
- Update version and metadata in formula
- Publish formula to Homebrew tap repository
- Test formula installation on target platforms

**Outputs:**
- Homebrew formula Ruby file (`cline.rb`)
- Published formula in tap repository
- Installation verification reports
- Formula metadata (version, URLs, checksums)

**Outcomes:**
- macOS users can install Cline CLI via `brew install cline`
- Linux users can install via Homebrew on Linux (WSL and native)
- Single-command installation without manual binary download
- Automatic updates via `brew upgrade`
- Proper PATH integration

**Impacts:**
- Significantly reduced installation friction for macOS/Linux developers
- Familiar installation experience for developers using Homebrew
- Automatic update notifications through Homebrew
- Professional distribution comparable to other modern CLI tools (gh, kubectl, terraform)
- Foundation for potential future homebrew-core inclusion

## Technical Requirements

**Architecture Layer:** Infrastructure/Distribution

**Integration Points:**
- **Proposed:** `golang-cli/scripts/generate-homebrew-formula.sh` - Automated formula generation script
- **Proposed:** `.github/workflows/release.yml` - GitHub Actions workflow to trigger formula generation on release
- **Proposed:** `cline/homebrew-tap` repository - Custom Homebrew tap for formula distribution
- **Proposed:** GitHub Releases API - Source for binary URLs and release metadata
- [Existing: `cli/scripts/update-brew-formula.mts`] [Source: cli/scripts/update-brew-formula.mts:L1] - Reference existing TypeScript CLI formula update script for patterns

**Data Requirements:**
- Release version (semantic versioning)
- Platform-architecture mapping:
  - `darwin-amd64` → macOS Intel
  - `darwin-arm64` → macOS Apple Silicon
  - `linux-amd64` → Linux AMD64
  - `linux-arm64` → Linux ARM64
- SHA256 checksums for each binary
- GitHub release download URLs following pattern: `https://github.com/cline/cline/releases/download/v{VERSION}/cline_{VERSION}_{OS}_{ARCH}.tar.gz`

**Performance Requirements:**
- Formula generation must complete within 30 seconds of release
- Installation time under 30 seconds on broadband connection
- Binary download should use GitHub's CDN for optimal speed

**Security Requirements:**
- SHA256 checksum verification mandatory for all binaries
- HTTPS-only download URLs
- Formula file integrity through git commit signing (optional but recommended)
- No sensitive data in formula (API keys, credentials)

## User Experience

**User Personas:** 
- Developer User (macOS/Linux developers who prefer Homebrew)
- DevOps/Automation User (setting up CI environments with Homebrew)

**User Actions:**
1. Add Cline tap: `brew tap cline/tap`
2. Install Cline: `brew install cline`
3. Upgrade Cline: `brew upgrade cline`
4. Verify installation: `cline version`

**UI Components:** N/A (command-line only)

**Mobile Considerations:** N/A (Infrastructure feature)

## BDD Scenarios
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Homebrew Formula Feature BDD Scenarios]

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

## Success Criteria
[Source: .oxenated/docs/planning/cline-cli-golang-migration-prd.md:L1-L1 - Epic Success Metrics]

**Functional:**
- Homebrew formula generates correctly with all platform URLs and SHA256 checksums
- Formula includes proper macOS Intel, macOS Apple Silicon, Linux AMD64, and Linux ARM64 support
- `brew install cline` successfully installs the CLI on all supported platforms

**Performance:**
- Installation time under 30 seconds on broadband connection
- Formula generation completes within CI/CD pipeline timeout (5 minutes)

**Quality:**
- All SHA256 checksums validate correctly
- No broken URLs in formula
- Formula passes `brew audit --strict` validation

**Integration:**
- Formula automatically updates on each release
- Integration with GitHub Actions release workflow
- Compatible with both custom tap and potential future homebrew-core submission

**Business Value:**
- Users can install via familiar `brew install` command
- Reduces installation friction compared to manual binary download
- Enables automatic updates through Homebrew

## Testing Strategy

**Unit Testing:**
- Formula generation script produces valid Ruby syntax
- Checksum calculation matches expected SHA256 values
- URL generation follows correct pattern for each platform/architecture

**Integration Testing:**
- Test formula installation in clean macOS VM (Intel and Apple Silicon)
- Test formula installation in clean Linux environment (AMD64 and ARM64)
- Verify binary executes correctly after installation
- Verify PATH integration works

**User Acceptance:**
- Manual installation test on macOS with Intel processor
- Manual installation test on macOS with Apple Silicon (M1/M2/M3)
- Manual installation test on Ubuntu/Debian Linux
- Verify `cline --help` works immediately after installation

**Performance Testing:**
- Measure installation time from `brew install` to executable availability
- Verify download speed meets requirements

## Tasks Overview

**Task 1: Create Homebrew Formula Generation Script**
- Implement shell script or Go program to generate formula from template
- Integrate checksum calculation for each platform binary
- Support version substitution and URL generation

**Task 2: Define Formula Template Structure**
- Create Ruby formula template with multi-platform support
- Define `url` and `sha256` stanzas for each platform
- Include proper class name, homepage, and license metadata

**Task 3: Integrate with Release Workflow**
- Add formula generation step to GitHub Actions release workflow
- Automate tap repository updates on release
- Handle versioning and git commits for formula updates

**Task 4: Setup Homebrew Tap Repository**
- Create `cline/homebrew-tap` repository if using custom tap
- Configure repository structure for Homebrew compatibility
- Setup access controls and automation permissions

**Task 5: Create Installation Documentation**
- Document `brew tap` and `brew install` commands
- Include troubleshooting for common installation issues
- Document upgrade process

**Task 6: Implement Formula Validation**
- Add `brew audit --strict` validation to CI pipeline
- Verify formula syntax before publishing
- Test formula in clean environments

## Implementation Notes

### Formula Structure
The Homebrew formula should follow this structure for multi-platform support:

```ruby
class Cline < Formula
  desc "AI-powered coding assistant CLI"
  homepage "https://github.com/cline/cline"
  version "X.Y.Z"
  
  on_macos do
    on_intel do
      url "https://github.com/cline/cline/releases/download/vX.Y.Z/cline_X.Y.Z_darwin_amd64.tar.gz"
      sha256 "..."
    end
    on_arm do
      url "https://github.com/cline/cline/releases/download/vX.Y.Z/cline_X.Y.Z_darwin_arm64.tar.gz"
      sha256 "..."
    end
  end
  
  on_linux do
    on_intel do
      url "https://github.com/cline/cline/releases/download/vX.Y.Z/cline_X.Y.Z_linux_amd64.tar.gz"
      sha256 "..."
    end
    on_arm do
      url "https://github.com/cline/cline/releases/download/vX.Y.Z/cline_X.Y.Z_linux_arm64.tar.gz"
      sha256 "..."
    end
  end
  
  def install
    bin.install "cline"
  end
  
  test do
    system "#{bin}/cline", "version"
  end
end
```

### Automation Approach
- Use GitHub Actions to trigger formula generation on release publish
- Script should download release assets, calculate SHA256 checksums, and generate formula
- Commit and push updated formula to tap repository automatically
- Consider using `brew bump-formula-pr` pattern for future homebrew-core submission

### Tap vs. Core
- **Phase 1:** Use custom tap (`cline/homebrew-tap`) for rapid iteration
- **Phase 2:** Consider submitting to `homebrew-core` after stability is proven
- Custom tap allows faster updates without waiting for Homebrew maintainers

## Citation Verification
- [x] All PRD content includes source line numbers
- [x] Existing code references cite actual file paths and lines
- [x] New functionality clearly marked as "Proposed:" or "To be created:"
- [x] Integration points cite existing interfaces or mark as new
- [x] Citation Verification checklist completed in each feature.md