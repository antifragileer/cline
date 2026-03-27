#!/usr/bin/env node

/**
 * Cline CLI NPM Wrapper
 *
 * This module provides a Node.js wrapper around the Go CLI binary.
 * It handles platform detection, binary execution, and error handling.
 */

const { spawn } = require("child_process")
const path = require("path")
const fs = require("fs")
const platform = require("./platform")

const BINARY_NAME = "cline"

/**
 * Get the binary path for the Cline CLI
 * Tries multiple locations in order:
 * 1. Package bin directory
 * 2. System PATH
 * 3. Common installation directories
 * @returns {string} The path to the binary
 * @throws {Error} If the binary cannot be found
 */
function getBinaryPath() {
	// Try to find the binary using the platform module
	const binaryPath = platform.findBinary()

	if (binaryPath) {
		return binaryPath
	}

	// If not found, provide helpful error message with platform info
	const platformInfo = platform.getPlatformInfo()

	throw new Error(
		`Cline CLI binary not found.\n\n` +
			`Platform: ${platformInfo.platform}/${platformInfo.arch}\n` +
			`Node.js: ${platformInfo.nodeVersion}\n\n` +
			`The binary should be located at:\n` +
			`  ${platform.getBinaryPath()}\n\n` +
			`Please try one of the following:\n` +
			`  1. Reinstall the package: npm install -g @cline/golang-cli\n` +
			`  2. Download manually from: https://github.com/cline/cline/releases\n` +
			`  3. Install using the install script:\n` +
			`     curl -fsSL https://raw.githubusercontent.com/cline/cline/main/install.sh | sh\n\n` +
			`For more help, visit: https://github.com/cline/cline#readme`,
	)
}

/**
 * Run the Cline CLI with the given arguments
 * @param {string[]} args - Arguments to pass to the CLI
 * @param {Object} options - Options for spawning the process
 * @returns {Object} The child process
 */
function runCLI(args, options = {}) {
	const binaryPath = getBinaryPath()

	// Prepare environment
	const env = {
		...process.env,
		CLINE_NPM_WRAPPER: "true",
		CLINE_NPM_VERSION: require("./package.json").version,
		...options.env,
	}

	// Spawn the child process
	const child = spawn(binaryPath, args, {
		stdio: options.stdio || "inherit",
		env: env,
		cwd: options.cwd || process.cwd(),
		windowsHide: process.platform === "win32", // Hide window on Windows
	})

	// Handle process errors
	child.on("error", (err) => {
		if (err.code === "ENOENT") {
			console.error("Error: Cline CLI binary not found at:", binaryPath)
			console.error("Please reinstall the package.")
		} else {
			console.error("Failed to start Cline CLI:", err.message)
		}
		process.exit(1)
	})

	// Handle process exit
	child.on("exit", (code, signal) => {
		if (signal) {
			process.exit(1)
		} else {
			process.exit(code || 0)
		}
	})

	return child
}

/**
 * Check if the Cline CLI is installed
 * @returns {boolean} Whether the CLI is installed
 */
function isInstalled() {
	try {
		return platform.findBinary() !== null
	} catch (e) {
		return false
	}
}

/**
 * Get the installed version of the Cline CLI
 * @returns {string|null} The version string or null if not installed
 */
function getVersion() {
	try {
		const binaryPath = getBinaryPath()
		const { execSync } = require("child_process")
		const version = execSync(`"${binaryPath}" --version`, {
			encoding: "utf8",
			timeout: 5000,
		}).trim()
		return version
	} catch (e) {
		return null
	}
}

/**
 * Get platform information
 * @returns {Object} Platform information
 */
function getPlatformInfo() {
	return platform.getPlatformInfo()
}

// If called directly, run the CLI
if (require.main === module) {
	const args = process.argv.slice(2)

	// Handle special wrapper commands
	if (args[0] === "--wrapper-version") {
		console.log(require("./package.json").version)
		process.exit(0)
	}

	if (args[0] === "--wrapper-check") {
		const installed = isInstalled()
		const version = getVersion()
		const platformInfo = getPlatformInfo()

		console.log("Cline CLI Wrapper Check")
		console.log("=======================")
		console.log(`Wrapper version: ${require("./package.json").version}`)
		console.log(`Platform: ${platformInfo.platform}/${platformInfo.arch}`)
		console.log(`Installed: ${installed ? "Yes" : "No"}`)
		if (version) {
			console.log(`CLI version: ${version}`)
		}
		console.log(`Binary path: ${platform.findBinary() || "Not found"}`)
		process.exit(installed ? 0 : 1)
	}

	runCLI(args)
}

module.exports = {
	runCLI,
	getBinaryPath,
	isInstalled,
	getVersion,
	getPlatformInfo,
}
