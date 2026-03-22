#!/usr/bin/env node

/**
 * Cline Go CLI - NPM Entry Point
 *
 * This module provides a programmatic interface to the Cline Go CLI binary.
 * It handles spawning the correct binary for the current platform.
 */

const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");
const os = require("os");

const BINARY_NAME = "cline";
const PACKAGE_NAME = "@cline/golang-cli";

/**
 * Get the binary path for the current platform
 * @returns {string} Path to the binary
 * @throws {Error} If binary is not found
 */
function getBinaryPath() {
	const binDir = path.join(__dirname, "bin");
	const ext = process.platform === "win32" ? ".exe" : "";
	const binaryName = `${BINARY_NAME}${ext}`;
	const binaryPath = path.join(binDir, binaryName);

	if (!fs.existsSync(binaryPath)) {
		throw new Error(
			`Binary not found at ${binaryPath}. ` +
			`Please run 'npm install' to download the binary for your platform.`
		);
	}

	return binaryPath;
}

/**
 * Run the Cline CLI with the given arguments
 * @param {string[]} args - Command line arguments
 * @param {object} options - Spawn options (cwd, env, stdio, etc.)
 * @returns {ChildProcess} The spawned process
 */
function run(args = [], options = {}) {
	const binaryPath = getBinaryPath();
	const defaultOptions = {
		stdio: "inherit",
		cwd: process.cwd(),
		env: process.env,
	};

	return spawn(binaryPath, args, { ...defaultOptions, ...options });
}

/**
 * Run the Cline CLI synchronously
 * @param {string[]} args - Command line arguments
 * @param {object} options - Spawn options
 * @returns {object} Result with status, stdout, stderr
 */
function runSync(args = [], options = {}) {
	const binaryPath = getBinaryPath();
	const { execFileSync } = require("child_process");
	const defaultOptions = {
		cwd: process.cwd(),
		env: process.env,
		encoding: "utf8",
	};

	try {
		const stdout = execFileSync(binaryPath, args, { ...defaultOptions, ...options });
		return {
			status: 0,
			stdout,
			stderr: "",
			success: true,
		};
	} catch (error) {
		return {
			status: error.status || 1,
			stdout: error.stdout || "",
			stderr: error.stderr || "",
			success: false,
			error,
		};
	}
}

/**
 * Get the version of the installed binary
 * @returns {string|null} Version string or null if not available
 */
function getVersion() {
	try {
		const result = runSync(["--version"]);
		if (result.success) {
			return result.stdout.trim();
		}
	} catch (error) {
		// Ignore errors
	}
	return null;
}

/**
 * Check if the binary is installed
 * @returns {boolean} True if binary exists
 */
function isInstalled() {
	try {
		getBinaryPath();
		return true;
	} catch {
		return false;
	}
}

/**
 * Get platform information
 * @returns {object} Platform details
 */
function getPlatform() {
	return {
		platform: process.platform,
		arch: process.arch,
		isWindows: process.platform === "win32",
		isMac: process.platform === "darwin",
		isLinux: process.platform === "linux",
		nodeVersion: process.version,
	};
}

module.exports = {
	getBinaryPath,
	run,
	runSync,
	getVersion,
	isInstalled,
	getPlatform,
	PACKAGE_NAME,
	BINARY_NAME,
};

// If this file is run directly, execute the CLI
if (require.main === module) {
	const binaryPath = getBinaryPath();
	const args = process.argv.slice(2);
	const child = run(args);

	child.on("exit", (code) => {
		process.exit(code || 0);
	});

	child.on("error", (error) => {
		console.error(`Failed to start ${PACKAGE_NAME}:`, error.message);
		process.exit(1);
	});
}