#!/usr/bin/env node

/**
 * Cline CLI NPM Wrapper
 *
 * This module provides a Node.js wrapper around the Go CLI binary.
 * It handles platform detection and binary execution.
 */

const { spawn } = require("child_process")
const path = require("path")
const fs = require("fs")

const BINARY_NAME = "cline-go"

function getBinaryPath() {
	const platform = process.platform
	const ext = platform === "win32" ? ".exe" : ""
	const binName = `${BINARY_NAME}${ext}`

	// Check in bin directory
	const binPath = path.join(__dirname, "bin", binName)
	if (fs.existsSync(binPath)) {
		return binPath
	}

	// Check if cline is in PATH
	try {
		const which = require("child_process")
			.execSync(platform === "win32" ? "where cline-go" : "which cline-go", {
				encoding: "utf8",
				stdio: ["pipe", "pipe", "ignore"],
			})
			.trim()
		if (which) {
			return which.split("\n")[0]
		}
	} catch (e) {
		// Not in PATH
	}

	throw new Error(
		`Cline CLI binary not found. Please run 'npm install' to download the binary, ` +
			`or install Cline CLI manually from https://github.com/cline/cline/releases`,
	)
}

function runCLI(args) {
	const binaryPath = getBinaryPath()

	const child = spawn(binaryPath, args, {
		stdio: "inherit",
		env: process.env,
	})

	child.on("error", (err) => {
		console.error("Failed to start Cline CLI:", err.message)
		process.exit(1)
	})

	child.on("exit", (code) => {
		process.exit(code)
	})
}

// If called directly, run the CLI
if (require.main === module) {
	const args = process.argv.slice(2)
	runCLI(args)
}

module.exports = { runCLI, getBinaryPath }
