#!/usr/bin/env node

/**
 * Platform detection and binary management for Cline CLI
 */

const os = require("os")
const fs = require("fs")
const path = require("path")
const { execSync } = require("child_process")

// Platform mappings
const PLATFORM_MAP = {
	darwin: "darwin",
	linux: "linux",
	win32: "windows",
}

const ARCH_MAP = {
	x64: "amd64",
	arm64: "arm64",
	ia32: "386", // 32-bit x86 (limited support)
}

/**
 * Get the current platform and architecture
 * @returns {Object} Object with platform and arch properties
 */
function getPlatform() {
	const platform = process.platform
	const arch = process.arch

	if (!PLATFORM_MAP[platform]) {
		throw new Error(`Unsupported platform: ${platform}\n` + `Supported platforms: macOS (darwin), Linux, Windows`)
	}

	if (!ARCH_MAP[arch]) {
		throw new Error(`Unsupported architecture: ${arch}\n` + `Supported architectures: x64, arm64`)
	}

	return {
		platform: PLATFORM_MAP[platform],
		arch: ARCH_MAP[arch],
		rawPlatform: platform,
		rawArch: arch,
	}
}

/**
 * Get the binary name for the current platform
 * @param {boolean} includeExtension Whether to include the file extension
 * @returns {string} The binary name
 */
function getBinaryName(includeExtension = true) {
	const { platform } = getPlatform()
	const ext = platform === "windows" && includeExtension ? ".exe" : ""
	return `cline${ext}`
}

/**
 * Get the full binary name including platform and arch
 * @param {string} version - The version string
 * @returns {string} The full binary name
 */
function getFullBinaryName(version) {
	const { platform, arch } = getPlatform()
	const ext = platform === "windows" ? ".exe" : ""
	return `cline_${version}_${platform}_${arch}${ext}`
}

/**
 * Get the tarball name for the current platform
 * @param {string} version - The version string
 * @returns {string} The tarball name
 */
function getTarballName(version) {
	const { platform, arch } = getPlatform()
	return `cline_${version}_${platform}_${arch}.tar.gz`
}

/**
 * Get the binary download URL
 * @param {string} version - The version string
 * @param {string} baseUrl - The base URL for downloads
 * @returns {string} The download URL
 */
function getDownloadUrl(version, baseUrl = "https://github.com/cline/cline/releases/download") {
	const tarball = getTarballName(version)
	const tag = version.startsWith("v") ? version : `v${version}`
	return `${baseUrl}/${tag}/${tarball}`
}

/**
 * Get the binary path within the package
 * @returns {string} The path to the binary directory
 */
function getBinaryDir() {
	return path.join(__dirname, "bin")
}

/**
 * Get the full path to the binary
 * @returns {string} The path to the binary
 */
function getBinaryPath() {
	const binaryDir = getBinaryDir()
	const binaryName = getBinaryName()
	return path.join(binaryDir, binaryName)
}

/**
 * Check if the binary exists
 * @returns {boolean} Whether the binary exists
 */
function binaryExists() {
	return fs.existsSync(getBinaryPath())
}

/**
 * Check if a binary exists in the system PATH
 * @param {string} binaryName - The name of the binary to check
 * @returns {string|null} The path to the binary if found, null otherwise
 */
function findInPath(binaryName) {
	try {
		const cmd = process.platform === "win32" ? "where" : "which"
		const result = execSync(`${cmd} ${binaryName}`, {
			encoding: "utf8",
			stdio: ["pipe", "pipe", "ignore"],
		})
			.toString()
			.trim()

		if (result) {
			const paths = result.split("\n")
			// Return the first non-empty path
			for (const p of paths) {
				const trimmed = p.trim()
				if (trimmed && fs.existsSync(trimmed)) {
					return trimmed
				}
			}
		}
	} catch (e) {
		// Binary not found in PATH
	}

	return null
}

/**
 * Find the Cline binary
 * Tries multiple locations in order:
 * 1. Package bin directory
 * 2. System PATH
 * 3. Common installation directories
 * @returns {string|null} The path to the binary if found, null otherwise
 */
function findBinary() {
	// 1. Check package bin directory
	const packageBinary = getBinaryPath()
	if (fs.existsSync(packageBinary)) {
		return packageBinary
	}

	// 2. Check PATH
	const pathBinary = findInPath("cline")
	if (pathBinary) {
		return pathBinary
	}

	// 3. Check alternative names
	const altBinary = findInPath("cline-go")
	if (altBinary) {
		return altBinary
	}

	// 4. Check common installation directories
	const commonPaths = getCommonInstallPaths()
	for (const commonPath of commonPaths) {
		if (fs.existsSync(commonPath)) {
			return commonPath
		}
	}

	return null
}

/**
 * Get common installation paths to check
 * @returns {string[]} Array of common paths
 */
function getCommonInstallPaths() {
	const { platform, rawPlatform } = getPlatform()
	const homeDir = os.homedir()
	const paths = []

	if (rawPlatform === "darwin" || rawPlatform === "linux") {
		// Unix-like systems
		paths.push(
			"/usr/local/bin/cline",
			"/usr/bin/cline",
			"/opt/cline/bin/cline",
			path.join(homeDir, ".local", "bin", "cline"),
			path.join(homeDir, "bin", "cline"),
			path.join(homeDir, ".cline", "bin", "cline"),
		)
	} else if (rawPlatform === "win32") {
		// Windows
		const programFiles = process.env.ProgramFiles || "C:\\Program Files"
		const localAppData = process.env.LOCALAPPDATA || path.join(homeDir, "AppData", "Local")

		paths.push(
			path.join(programFiles, "Cline", "cline.exe"),
			path.join(localAppData, "Cline", "cline.exe"),
			path.join(homeDir, "cline", "cline.exe"),
		)
	}

	return paths
}

/**
 * Get platform information as a string
 * @returns {string} Platform information
 */
function getPlatformInfo() {
	const { platform, arch, rawPlatform, rawArch } = getPlatform()
	return {
		platform,
		arch,
		rawPlatform,
		rawArch,
		nodeVersion: process.version,
	}
}

/**
 * Check if the current platform is supported
 * @returns {boolean} Whether the platform is supported
 */
function isSupported() {
	try {
		getPlatform()
		return true
	} catch (e) {
		return false
	}
}

/**
 * Get the npm cache directory for storing downloaded binaries
 * @returns {string} The cache directory path
 */
function getCacheDir() {
	const homeDir = os.homedir()

	// Use npm's cache if available
	if (process.env.npm_config_cache) {
		return path.join(process.env.npm_config_cache, "cline-cli")
	}

	// Fall back to platform-specific cache
	if (process.platform === "win32") {
		return path.join(process.env.LOCALAPPDATA || path.join(homeDir, "AppData", "Local"), "cline-cli", "Cache")
	}
	if (process.platform === "darwin") {
		return path.join(homeDir, "Library", "Caches", "cline-cli")
	}
	// Linux and others
	return path.join(process.env.XDG_CACHE_HOME || path.join(homeDir, ".cache"), "cline-cli")
}

/**
 * Ensure the binary directory exists
 */
function ensureBinaryDir() {
	const binaryDir = getBinaryDir()
	if (!fs.existsSync(binaryDir)) {
		fs.mkdirSync(binaryDir, { recursive: true })
	}
}

/**
 * Get the expected checksum file name
 * @param {string} version - The version string
 * @returns {string} The checksum file name
 */
function getChecksumFileName(version) {
	const tarball = getTarballName(version)
	return `${tarball}.sha256`
}

/**
 * Get the checksum download URL
 * @param {string} version - The version string
 * @param {string} baseUrl - The base URL for downloads
 * @returns {string} The checksum URL
 */
function getChecksumUrl(version, baseUrl = "https://github.com/cline/cline/releases/download") {
	const checksumFile = getChecksumFileName(version)
	const tag = version.startsWith("v") ? version : `v${version}`
	return `${baseUrl}/${tag}/${checksumFile}`
}

module.exports = {
	getPlatform,
	getBinaryName,
	getFullBinaryName,
	getTarballName,
	getDownloadUrl,
	getBinaryDir,
	getBinaryPath,
	binaryExists,
	findInPath,
	findBinary,
	getCommonInstallPaths,
	getPlatformInfo,
	isSupported,
	getCacheDir,
	ensureBinaryDir,
	getChecksumFileName,
	getChecksumUrl,
}
