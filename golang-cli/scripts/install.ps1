#!/usr/bin/env powershell
#Requires -Version 5.0

<#
.SYNOPSIS
    Cline CLI Installation Script for Windows

.DESCRIPTION
    This script downloads and installs the Cline CLI on Windows systems.
    It automatically detects the platform architecture and downloads the
    appropriate binary.

.PARAMETER Version
    Specific version to install (e.g., "1.0.0"). Defaults to latest.

.PARAMETER InstallDir
    Directory to install the binary. Defaults to "C:\Program Files\Cline".

.PARAMETER AddToPath
    Add the installation directory to the system PATH.

.PARAMETER NoPrompt
    Skip confirmation prompts.

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -Version "1.0.0" -InstallDir "C:\Tools\Cline"

.EXAMPLE
    .\install.ps1 -AddToPath -NoPrompt

.NOTES
    Run with Administrator privileges if installing to system directories
    or adding to system PATH.
#>

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\Cline",
    [switch]$AddToPath,
    [switch]$NoPrompt
)

# Error handling
$ErrorActionPreference = "Stop"

# Colors for output (fallback for older PowerShell)
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    
    $colorMap = @{
        "Red" = "Red"
        "Green" = "Green"
        "Yellow" = "Yellow"
        "Blue" = "Cyan"
        "White" = "White"
    }
    
    Write-Host $Message -ForegroundColor $colorMap[$Color]
}

function Write-Info {
    param([string]$Message)
    Write-ColorOutput "[INFO] $Message" "Blue"
}

function Write-Success {
    param([string]$Message)
    Write-ColorOutput "[SUCCESS] $Message" "Green"
}

function Write-Warning {
    param([string]$Message)
    Write-ColorOutput "[WARNING] $Message" "Yellow"
}

function Write-Error {
    param([string]$Message)
    Write-ColorOutput "[ERROR] $Message" "Red"
}

# Configuration
$GitHubRepo = "cline/cline"
$BinaryName = "cline.exe"

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" { 
            Write-Warning "x86 (32-bit) architecture detected. Cline CLI requires 64-bit Windows."
            exit 1
        }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest version from GitHub API
function Get-LatestVersion {
    try {
        $apiUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"
        $response = Invoke-RestMethod -Uri $apiUrl -TimeoutSec 30
        
        # Remove 'v' prefix if present
        $version = $response.tag_name -replace '^v', ''
        return $version
    }
    catch {
        Write-Error "Failed to get latest version from GitHub API: $_"
        exit 1
    }
}

# Download file with progress
function Download-File {
    param(
        [string]$Url,
        [string]$Destination
    )
    
    try {
        $ProgressPreference = "Continue"
        
        Invoke-WebRequest -Uri $Url -OutFile $Destination -TimeoutSec 300 -UseBasicParsing
        
        if (-not (Test-Path $Destination)) {
            throw "Download failed - file not created"
        }
        
        $fileSize = (Get-Item $Destination).Length
        if ($fileSize -eq 0) {
            throw "Download failed - file is empty"
        }
        
        return $true
    }
    catch {
        Write-Error "Failed to download from $Url : $_"
        return $false
    }
}

# Verify checksum
function Verify-Checksum {
    param(
        [string]$FilePath,
        [string]$ExpectedChecksum
    )
    
    if (-not $ExpectedChecksum) {
        Write-Warning "No checksum provided, skipping verification"
        return $true
    }
    
    try {
        $actualChecksum = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()
        $expectedLower = $ExpectedChecksum.ToLower()
        
        if ($actualChecksum -eq $expectedLower) {
            return $true
        }
        else {
            Write-Error "Checksum mismatch!"
            Write-Error "  Expected: $expectedLower"
            Write-Error "  Actual:   $actualChecksum"
            return $false
        }
    }
    catch {
        Write-Error "Failed to verify checksum: $_"
        return $false
    }
}

# Add to PATH
function Add-ToPath {
    param([string]$Directory)
    
    try {
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        
        if ($currentPath -like "*$Directory*") {
            Write-Info "Directory already in PATH"
            return $true
        }
        
        $newPath = "$currentPath;$Directory"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        
        # Also update current session
        $env:PATH = "$env:PATH;$Directory"
        
        Write-Success "Added to PATH: $Directory"
        return $true
    }
    catch {
        Write-Error "Failed to add to PATH: $_"
        return $false
    }
}

# Main installation
function Install-Cline {
    Write-ColorOutput ""
    Write-ColorOutput "========================================" "Blue"
    Write-ColorOutput "  Cline CLI Installation for Windows" "Blue"
    Write-ColorOutput "========================================" "Blue"
    Write-ColorOutput ""
    
    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Error "PowerShell 5.0 or later is required"
        exit 1
    }
    
    # Check for admin rights if installing to Program Files
    if ($InstallDir -like "$env:ProgramFiles*") {
        $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        
        if (-not $isAdmin) {
            Write-Warning "Administrator privileges required to install to $InstallDir"
            Write-Info "Please run PowerShell as Administrator and try again"
            exit 1
        }
    }
    
    # Get version
    if ($Version -eq "latest") {
        Write-Info "Detecting latest version..."
        $Version = Get-LatestVersion
    }
    
    Write-Info "Installing Cline CLI version $Version"
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"
    
    # Set up paths
    $assetName = "cline-v$Version-windows-$arch.zip"
    $downloadUrl = "https://github.com/$GitHubRepo/releases/download/v$Version/$assetName"
    $tempDir = Join-Path $env:TEMP "cline-install-$(Get-Random)"
    $zipPath = Join-Path $tempDir $assetName
    
    Write-Info "Download URL: $downloadUrl"
    
    # Confirm installation
    if (-not $NoPrompt) {
        Write-ColorOutput ""
        $confirm = Read-Host "Install Cline CLI v$Version to $InstallDir? (Y/n)"
        if ($confirm -and $confirm -notmatch '^[Yy]') {
            Write-Info "Installation cancelled"
            exit 0
        }
    }
    
    # Create directories
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    
    try {
        # Download binary
        Write-Info "Downloading Cline CLI..."
        if (-not (Download-File -Url $downloadUrl -Destination $zipPath)) {
            exit 1
        }
        Write-Success "Download complete"
        
        # Download checksum file
        $checksumUrl = "$downloadUrl.sha256"
        $checksumPath = "$zipPath.sha256"
        $expectedChecksum = $null
        
        try {
            Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -TimeoutSec 30 -UseBasicParsing -ErrorAction SilentlyContinue
            $expectedChecksum = (Get-Content $checksumPath -Raw).Trim().Split()[0]
            Write-Info "Downloaded checksum: $expectedChecksum"
        }
        catch {
            Write-Warning "Could not download checksum file"
        }
        
        # Verify checksum
        if ($expectedChecksum) {
            Write-Info "Verifying checksum..."
            if (-not (Verify-Checksum -FilePath $zipPath -ExpectedChecksum $expectedChecksum)) {
                exit 1
            }
            Write-Success "Checksum verified"
        }
        
        # Extract archive
        Write-Info "Extracting archive..."
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
        
        # Find extracted binary
        $extractedBinary = Get-ChildItem -Path $tempDir -Recurse -Filter $BinaryName | Select-Object -First 1
        
        if (-not $extractedBinary) {
            Write-Error "Binary not found in extracted archive"
            exit 1
        }
        
        # Install binary
        $destinationPath = Join-Path $InstallDir $BinaryName
        
        # Remove existing binary if present
        if (Test-Path $destinationPath) {
            Write-Info "Removing existing installation..."
            Remove-Item $destinationPath -Force
        }
        
        Write-Info "Installing binary to $destinationPath..."
        Move-Item -Path $extractedBinary.FullName -Destination $destinationPath -Force
        
        # Verify installation
        Write-Info "Verifying installation..."
        $installedVersion = & $destinationPath --version 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Installation successful!"
            Write-ColorOutput ""
            Write-ColorOutput "Installed: $destinationPath" "Green"
            Write-ColorOutput "Version: $installedVersion" "Green"
        }
        else {
            Write-Error "Installation verification failed"
            exit 1
        }
        
        # Add to PATH if requested
        if ($AddToPath) {
            Write-Info "Adding to PATH..."
            Add-ToPath -Directory $InstallDir
        }
        
        # Output usage information
        Write-ColorOutput ""
        Write-ColorOutput "========================================" "Blue"
        Write-ColorOutput "  Installation Complete!" "Green"
        Write-ColorOutput "========================================" "Blue"
        Write-ColorOutput ""
        Write-ColorOutput "Get started with:" "White"
        Write-ColorOutput "  cline --help          Show help" "White"
        Write-ColorOutput "  cline --version       Show version" "White"
        Write-ColorOutput "  cline 'Hello'         Run your first task" "White"
        Write-ColorOutput ""
        
        if (-not $AddToPath) {
            Write-ColorOutput "To add to your PATH, run:" "Yellow"
            Write-ColorOutput "  [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$InstallDir', 'User')" "White"
            Write-ColorOutput ""
        }
        
        Write-ColorOutput "Documentation: https://docs.cline.bot" "Blue"
        Write-ColorOutput ""
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Handle script execution
try {
    Install-Cline
}
catch {
    Write-Error "Installation failed: $_"
    Write-ColorOutput ""
    Write-ColorOutput "Troubleshooting:" "Yellow"
    Write-ColorOutput "  1. Check your internet connection" "White"
    Write-ColorOutput "  2. Verify the release exists: https://github.com/$GitHubRepo/releases" "White"
    Write-ColorOutput "  3. Try manual installation from the releases page" "White"
    Write-ColorOutput "  4. For proxy issues, check your proxy configuration" "White"
    exit 1
}