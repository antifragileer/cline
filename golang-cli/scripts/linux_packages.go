// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements APT (Debian/Ubuntu) and YUM (RHEL/CentOS/Fedora) package
// generation for distributing the CLI on Linux systems.
package scripts

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// LinuxPackageType represents the type of Linux package.
type LinuxPackageType string

const (
	// PackageTypeDEB represents Debian/Ubuntu .deb packages
	PackageTypeDEB LinuxPackageType = "deb"
	// PackageTypeRPM represents RHEL/CentOS/Fedora .rpm packages
	PackageTypeRPM LinuxPackageType = "rpm"
)

// LinuxPackageConfig holds configuration for building Linux packages.
type LinuxPackageConfig struct {
	// PackageType is the type of package to build
	PackageType LinuxPackageType

	// Name is the package name
	Name string

	// Version is the package version
	Version string

	// Release is the package release number (for RPM)
	Release string

	// Architecture is the target architecture (amd64, arm64)
	Architecture string

	// Maintainer is the package maintainer
	Maintainer string

	// Description is a short description
	Description string

	// LongDescription is a longer description
	LongDescription string

	// Homepage is the project homepage
	Homepage string

	// License is the software license
	License string

	// Section is the package section (for DEB)
	Section string

	// Priority is the package priority (for DEB)
	Priority string

	// SourcePath is the path to the binary to package
	SourcePath string

	// OutputDir is the directory for output packages
	OutputDir string

	// BinaryName is the name of the binary
	BinaryName string

	// InstallPath is where to install the binary
	InstallPath string

	// Depends are package dependencies
	Depends []string

	// Suggests are suggested packages
	Suggests []string

	// Conflicts are conflicting packages
	Conflicts []string

	// Replaces are packages this package replaces
	Replaces []string

	// Provides are virtual packages this package provides
	Provides []string
}

// DefaultLinuxPackageConfig returns a LinuxPackageConfig with sensible defaults.
func DefaultLinuxPackageConfig(pkgType LinuxPackageType, version, arch string) *LinuxPackageConfig {
	return &LinuxPackageConfig{
		PackageType:     pkgType,
		Name:            "cline",
		Version:         version,
		Release:         "1",
		Architecture:    arch,
		Maintainer:      "Cline Bot Inc. <support@cline.bot>",
		Description:     "AI-powered coding assistant CLI",
		LongDescription: "Cline is an autonomous coding agent that runs in your terminal.\nIt can create, edit, and manage files, execute commands, and more.",
		Homepage:        "https://github.com/cline/cline",
		License:         "Apache-2.0",
		Section:         "devel",
		Priority:        "optional",
		BinaryName:      "cline",
		InstallPath:     "/usr/bin",
		OutputDir:       "dist",
		Depends:         []string{"git"},
		Suggests:        []string{"ripgrep"},
	}
}

// LinuxPackageBuilder handles the creation of Linux packages.
type LinuxPackageBuilder struct {
	config *LinuxPackageConfig
}

// NewLinuxPackageBuilder creates a new package builder with the given configuration.
func NewLinuxPackageBuilder(config *LinuxPackageConfig) *LinuxPackageBuilder {
	return &LinuxPackageBuilder{config: config}
}

// Build creates the Linux package according to the configuration.
func (b *LinuxPackageBuilder) Build() (string, error) {
	switch b.config.PackageType {
	case PackageTypeDEB:
		return b.buildDEB()
	case PackageTypeRPM:
		return b.buildRPM()
	default:
		return "", fmt.Errorf("unsupported package type: %s", b.config.PackageType)
	}
}

// buildDEB creates a Debian .deb package.
func (b *LinuxPackageBuilder) buildDEB() (string, error) {
	// Validate inputs
	if err := b.validate(); err != nil {
		return "", err
	}

	// Map architecture names for Debian
	debArch := b.mapDebianArch(b.config.Architecture)

	// Create package filename
	pkgName := fmt.Sprintf("%s_%s_%s.deb", b.config.Name, b.config.Version, debArch)
	outputPath := filepath.Join(b.config.OutputDir, pkgName)

	// Create output directory
	if err := os.MkdirAll(b.config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary directory for package contents
	tempDir, err := os.MkdirTemp("", "cline-deb-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	if err := b.createDEBStructure(tempDir); err != nil {
		return "", err
	}

	// Copy binary
	if err := b.copyBinary(tempDir); err != nil {
		return "", err
	}

	// Create control file
	if err := b.createDEBControl(tempDir); err != nil {
		return "", err
	}

	// Create the .deb package (ar archive format)
	if err := b.createARArchive(tempDir, outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}

// buildRPM creates an RPM package.
func (b *LinuxPackageBuilder) buildRPM() (string, error) {
	// Validate inputs
	if err := b.validate(); err != nil {
		return "", err
	}

	// Map architecture names for RPM
	rpmArch := b.mapRPMArch(b.config.Architecture)

	// Create package filename
	pkgName := fmt.Sprintf("%s-%s-%s.%s.rpm", b.config.Name, b.config.Version, b.config.Release, rpmArch)
	outputPath := filepath.Join(b.config.OutputDir, pkgName)

	// Create output directory
	if err := os.MkdirAll(b.config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary directory for package contents
	tempDir, err := os.MkdirTemp("", "cline-rpm-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create RPM build structure
	buildRoot := filepath.Join(tempDir, "BUILDROOT")
	if err := os.MkdirAll(buildRoot, 0755); err != nil {
		return "", fmt.Errorf("failed to create build root: %w", err)
	}

	// Create directory structure
	installDir := filepath.Join(buildRoot, b.config.InstallPath)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create install directory: %w", err)
	}

	// Copy binary
	src, err := os.Open(b.config.SourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source binary: %w", err)
	}
	defer src.Close()

	dstPath := filepath.Join(installDir, b.config.BinaryName)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy binary: %w", err)
	}

	// Generate spec file
	specPath := filepath.Join(tempDir, "cline.spec")
	if err := b.createRPMSpec(specPath); err != nil {
		return "", err
	}

	// For now, return a placeholder. Full RPM creation requires rpmbuild tool.
	// In a real implementation, we would call rpmbuild here.
	// For this implementation, we'll create a simple cpio-based RPM structure.

	// Note: Full RPM creation is complex and typically requires the rpmbuild tool.
	// This implementation creates the basic structure. For production use,
	// consider using tools like `rpmbuild` or libraries like `github.com/google/rpmpack`.

	return outputPath, fmt.Errorf("RPM creation requires rpmbuild tool - structure prepared at %s", tempDir)
}

// validate checks if the configuration is valid.
func (b *LinuxPackageBuilder) validate() error {
	if b.config.Name == "" {
		return fmt.Errorf("package name is required")
	}
	if b.config.Version == "" {
		return fmt.Errorf("package version is required")
	}
	if b.config.Architecture == "" {
		return fmt.Errorf("package architecture is required")
	}
	if b.config.SourcePath == "" {
		return fmt.Errorf("source binary path is required")
	}
	if _, err := os.Stat(b.config.SourcePath); err != nil {
		return fmt.Errorf("source binary not found: %w", err)
	}
	return nil
}

// mapDebianArch maps Go architecture names to Debian architecture names.
func (b *LinuxPackageBuilder) mapDebianArch(arch string) string {
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "386":
		return "i386"
	case "arm":
		return "armhf"
	default:
		return arch
	}
}

// mapRPMArch maps Go architecture names to RPM architecture names.
func (b *LinuxPackageBuilder) mapRPMArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "386":
		return "i386"
	case "arm":
		return "armhfp"
	default:
		return arch
	}
}

// createDEBStructure creates the Debian package directory structure.
func (b *LinuxPackageBuilder) createDEBStructure(tempDir string) error {
	// Create usr/bin directory
	binDir := filepath.Join(tempDir, b.config.InstallPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create DEBIAN control directory
	controlDir := filepath.Join(tempDir, "DEBIAN")
	if err := os.MkdirAll(controlDir, 0755); err != nil {
		return fmt.Errorf("failed to create DEBIAN directory: %w", err)
	}

	return nil
}

// copyBinary copies the binary to the package directory.
func (b *LinuxPackageBuilder) copyBinary(tempDir string) error {
	src, err := os.Open(b.config.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer src.Close()

	dstPath := filepath.Join(tempDir, b.config.InstallPath, b.config.BinaryName)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	return nil
}

// createDEBControl creates the Debian control file.
func (b *LinuxPackageBuilder) createDEBControl(tempDir string) error {
	controlPath := filepath.Join(tempDir, "DEBIAN", "control")

	// Build depends line
	depends := ""
	if len(b.config.Depends) > 0 {
		depends = strings.Join(b.config.Depends, ", ")
	}

	// Build suggests line
	suggests := ""
	if len(b.config.Suggests) > 0 {
		suggests = strings.Join(b.config.Suggests, ", ")
	}

	debArch := b.mapDebianArch(b.config.Architecture)

	data := struct {
		*LinuxPackageConfig
		DependsStr   string
		SuggestsStr  string
		DebianArch   string
		InstalledSize int64
	}{
		LinuxPackageConfig: b.config,
		DependsStr:         depends,
		SuggestsStr:        suggests,
		DebianArch:         debArch,
	}

	// Calculate installed size
	stat, err := os.Stat(b.config.SourcePath)
	if err == nil {
		data.InstalledSize = (stat.Size() + 1023) / 1024 // Convert to KB, round up
	}

	tmpl, err := template.New("control").Parse(debianControlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse control template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute control template: %w", err)
	}

	if err := os.WriteFile(controlPath, []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("failed to write control file: %w", err)
	}

	return nil
}

// createARArchive creates an ar archive for the .deb package.
func (b *LinuxPackageBuilder) createARArchive(tempDir, outputPath string) error {
	// Create the ar archive file
	arFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create ar archive: %w", err)
	}
	defer arFile.Close()

	// Write ar magic header
	if _, err := arFile.WriteString("!<arch>\n"); err != nil {
		return fmt.Errorf("failed to write ar magic: %w", err)
	}

	// Create debian-binary file
	debianBinary := "2.0\n"
	debianBinaryContent := []byte(debianBinary)

	// Write debian-binary
	if err := b.writeAREntry(arFile, "debian-binary", time.Now(), debianBinaryContent); err != nil {
		return fmt.Errorf("failed to write debian-binary: %w", err)
	}

	// Create control.tar.gz
	controlData, err := b.createControlTarGz(tempDir)
	if err != nil {
		return fmt.Errorf("failed to create control.tar.gz: %w", err)
	}
	if err := b.writeAREntry(arFile, "control.tar.gz", time.Now(), controlData); err != nil {
		return fmt.Errorf("failed to write control.tar.gz: %w", err)
	}

	// Create data.tar.gz
	dataData, err := b.createDataTarGz(tempDir)
	if err != nil {
		return fmt.Errorf("failed to create data.tar.gz: %w", err)
	}
	if err := b.writeAREntry(arFile, "data.tar.gz", time.Now(), dataData); err != nil {
		return fmt.Errorf("failed to write data.tar.gz: %w", err)
	}

	return nil
}

// writeAREntry writes a single entry to an ar archive.
func (b *LinuxPackageBuilder) writeAREntry(w io.Writer, name string, modTime time.Time, data []byte) error {
	// Format ar header
	header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n",
		name,
		modTime.Unix(),
		0,    // Owner ID
		0,    // Group ID
		0644, // Mode
		len(data),
	)

	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	// Pad to 2-byte boundary
	if len(data)%2 != 0 {
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}

	return nil
}

// createControlTarGz creates the control.tar.gz file for Debian packages.
func (b *LinuxPackageBuilder) createControlTarGz(tempDir string) ([]byte, error) {
	var buf strings.Builder
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	controlFile := filepath.Join(tempDir, "DEBIAN", "control")
	data, err := os.ReadFile(controlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read control file: %w", err)
	}

	header := &tar.Header{
		Name:     "control",
		Size:     int64(len(data)),
		Mode:     0644,
		ModTime:  time.Now(),
		Typeflag: tar.TypeReg,
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// createDataTarGz creates the data.tar.gz file for Debian packages.
func (b *LinuxPackageBuilder) createDataTarGz(tempDir string) ([]byte, error) {
	var buf strings.Builder
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Walk through the temp directory, excluding DEBIAN
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip DEBIAN directory
		if strings.Contains(path, "DEBIAN") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// createRPMSpec creates the RPM spec file.
func (b *LinuxPackageBuilder) createRPMSpec(specPath string) error {
	data := struct {
		*LinuxPackageConfig
		BuildRoot string
		BuildArch string
	}{
		LinuxPackageConfig: b.config,
		BuildRoot:          "%{buildroot}",
		BuildArch:          b.mapRPMArch(b.config.Architecture),
	}

	tmpl, err := template.New("spec").Parse(rpmSpecTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse spec template: %w", err)
	}

	file, err := os.Create(specPath)
	if err != nil {
		return fmt.Errorf("failed to create spec file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute spec template: %w", err)
	}

	return nil
}

// PackageInfo holds information about a built package.
type PackageInfo struct {
	// Path is the path to the package file
	Path string

	// Type is the package type (deb or rpm)
	Type LinuxPackageType

	// Name is the package name
	Name string

	// Version is the package version
	Version string

	// Architecture is the package architecture
	Architecture string

	// Size is the package size in bytes
	Size int64

	// SHA256 is the package SHA256 checksum
	SHA256 string
}

// GeneratePackageInfo generates a PackageInfo from a built package.
func GeneratePackageInfo(pkgPath string, pkgType LinuxPackageType) (*PackageInfo, error) {
	stat, err := os.Stat(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat package: %w", err)
	}

	sha256, err := CalculateSHA256FromFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Parse filename to extract info
	filename := filepath.Base(pkgPath)

	info := &PackageInfo{
		Path:         pkgPath,
		Type:         pkgType,
		Size:         stat.Size(),
		SHA256:       sha256,
	}

	// Try to parse the filename
	// For DEB: cline_1.0.0_amd64.deb
	// For RPM: cline-1.0.0-1.x86_64.rpm
	if pkgType == PackageTypeDEB {
		parts := strings.Split(filename, "_")
		if len(parts) >= 2 {
			info.Name = parts[0]
			versionArch := strings.Split(parts[1], ".")
			if len(versionArch) >= 2 {
				info.Version = versionArch[0]
				info.Architecture = strings.Join(versionArch[1:len(versionArch)-1], ".")
			}
		}
	} else {
		parts := strings.Split(filename, "-")
		if len(parts) >= 3 {
			info.Name = parts[0]
			info.Version = parts[1]
			releaseArch := strings.Split(parts[2], ".")
			if len(releaseArch) >= 2 {
				info.Architecture = releaseArch[len(releaseArch)-2]
			}
		}
	}

	return info, nil
}

// LinuxPackageMetadata contains metadata for generating repository indexes.
type LinuxPackageMetadata struct {
	// Packages is the list of packages
	Packages []*PackageInfo

	// Distribution is the distribution name (e.g., "stable")
	Distribution string

	// Component is the component name (e.g., "main")
	Component string

	// Architecture is the architecture
	Architecture string
}

// GenerateAPTRepository generates an APT repository structure.
func GenerateAPTRepository(rootPath string, metadata *LinuxPackageMetadata) error {
	// Create pool directory
	poolDir := filepath.Join(rootPath, "pool", metadata.Component)
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

	// Create dists directory structure
	distsDir := filepath.Join(rootPath, "dists", metadata.Distribution)
	componentDir := filepath.Join(distsDir, metadata.Component, "binary-"+metadata.Architecture)
	if err := os.MkdirAll(componentDir, 0755); err != nil {
		return fmt.Errorf("failed to create component directory: %w", err)
	}

	// Copy packages to pool
	for _, pkg := range metadata.Packages {
		destPath := filepath.Join(poolDir, filepath.Base(pkg.Path))
		if err := copyFile(pkg.Path, destPath); err != nil {
			return fmt.Errorf("failed to copy package %s: %w", pkg.Path, err)
		}
	}

	// Generate Packages file
	packagesPath := filepath.Join(componentDir, "Packages")
	if err := generatePackagesFile(packagesPath, metadata.Packages); err != nil {
		return fmt.Errorf("failed to generate Packages file: %w", err)
	}

	// Generate compressed Packages.gz
	if err := compressPackagesFile(packagesPath); err != nil {
		return fmt.Errorf("failed to compress Packages file: %w", err)
	}

	// Generate Release file
	releasePath := filepath.Join(distsDir, "Release")
	if err := generateReleaseFile(releasePath, metadata); err != nil {
		return fmt.Errorf("failed to generate Release file: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// generatePackagesFile generates the Packages file for APT.
func generatePackagesFile(path string, packages []*PackageInfo) error {
	var content strings.Builder

	for _, pkg := range packages {
		content.WriteString(fmt.Sprintf("Package: %s\n", pkg.Name))
		content.WriteString(fmt.Sprintf("Version: %s\n", pkg.Version))
		content.WriteString(fmt.Sprintf("Architecture: %s\n", pkg.Architecture))
		content.WriteString(fmt.Sprintf("Maintainer: Cline Bot Inc. <support@cline.bot>\n"))
		content.WriteString(fmt.Sprintf("Filename: pool/main/%s\n", filepath.Base(pkg.Path)))
		content.WriteString(fmt.Sprintf("Size: %d\n", pkg.Size))
		content.WriteString(fmt.Sprintf("SHA256: %s\n", pkg.SHA256))
		content.WriteString(fmt.Sprintf("Description: AI-powered coding assistant CLI\n"))
		content.WriteString("\n")
	}

	return os.WriteFile(path, []byte(content.String()), 0644)
}

// compressPackagesFile compresses the Packages file to Packages.gz.
func compressPackagesFile(packagesPath string) error {
	data, err := os.ReadFile(packagesPath)
	if err != nil {
		return err
	}

	gzPath := packagesPath + ".gz"
	file, err := os.Create(gzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	_, err = gzw.Write(data)
	return err
}

// generateReleaseFile generates the Release file for APT.
func generateReleaseFile(path string, metadata *LinuxPackageMetadata) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("Origin: Cline\n"))
	content.WriteString(fmt.Sprintf("Label: Cline CLI\n"))
	content.WriteString(fmt.Sprintf("Suite: %s\n", metadata.Distribution))
	content.WriteString(fmt.Sprintf("Codename: %s\n", metadata.Distribution))
	content.WriteString(fmt.Sprintf("Date: %s\n", time.Now().UTC().Format(time.RFC1123)))
	content.WriteString(fmt.Sprintf("Architectures: %s\n", metadata.Architecture))
	content.WriteString(fmt.Sprintf("Components: %s\n", metadata.Component))
	content.WriteString(fmt.Sprintf("Description: Cline CLI APT Repository\n"))

	return os.WriteFile(path, []byte(content.String()), 0644)
}

const debianControlTemplate = `Package: {{.Name}}
Version: {{.Version}}
Section: {{.Section}}
Priority: {{.Priority}}
Architecture: {{.DebianArch}}
Maintainer: {{.Maintainer}}
Installed-Size: {{.InstalledSize}}
{{if .DependsStr}}Depends: {{.DependsStr}}
{{end}}{{if .SuggestsStr}}Suggests: {{.SuggestsStr}}
{{end}}Homepage: {{.Homepage}}
Description: {{.Description}}
{{.LongDescription}}
`

const rpmSpecTemplate = `Name:           {{.Name}}
Version:        {{.Version}}
Release:        {{.Release}}%{?dist}
Summary:        {{.Description}}

License:        {{.License}}
URL:            {{.Homepage}}
Source0:        %{name}-%{version}.tar.gz

BuildArch:      {{.BuildArch}}

{{range .Depends}}Requires:       {{.}}
{{end}}

%description
{{.LongDescription}}

%prep
# No prep needed for binary package

%build
# No build needed for binary package

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}}/{{.InstallPath}}
install -m 755 {{.BinaryName}} %{buildroot}}/{{.InstallPath}}/{{.BinaryName}}

%files
{{.InstallPath}}/{{.BinaryName}}

%changelog
* {{.BuildTime}} {{.Maintainer}} - {{.Version}}-{{.Release}}
- Initial package release
`