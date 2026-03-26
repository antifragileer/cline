#!/usr/bin/env node

/**
 * Cline CLI NPM Wrapper - Install Script
 *
 * This script downloads the appropriate binary for the user's platform
 * during npm install.
 */

const fs = require("fs")
const path = require("path")
const https = require("https")
const { execSync } = require("child_process")

const PACKAGE_VERSION = require("./package.json").version
const BINARY_NAME = "cline"
const GITHUB_REPO = "cline/cline"

// Platform mappings
const PLATFORM_MAP = {
	darwin: "darwin",
	linux: "linux",
	win32: "windows",
}

const ARCH_MAP = {
	x64: "amd64",
	arm64: "arm64",
}

function getPlatform() {
	const platform = process.platform
	const arch = process.arch

	if (!PLATFORM_MAP[platform]) {
		console.error(`Unsupported platform: ${platform}`)
		console.error("Supported platforms: macOS, Linux, Windows")
		process.exit(1)
	}

	if (!ARCH_MAP[arch]) {
		console.error(`Unsupported architecture: ${arch}`)
		console.error("Supported architectures: x64, arm64")
		process.exit(1)
	}

	return {
		platform: PLATFORM_MAP[platform],
		arch: ARCH_MAP[arch],
	}
}

function getBinaryUrl(version, platform, arch) {
	const ext = platform === "windows" ? ".exe" : ""
	const tag = version.startsWith("v") ? version : `v${version}`
	return `https://github.com/${GITHUB_REPO}/releases/download/${tag}/${BINARY_NAME}_${tag}_${platform}_${arch}${ext}`
}

function downloadFile(url, dest) {
	return new Promise((resolve, reject) => {
		console.log(`Downloading from: ${url}`)

		const file = fs.createWriteStream(dest)

		https
			.get(url, (response) => {
				if (response.statusCode === 302 || response.statusCode === 301) {
					// Follow redirects
					file.close()
					fs.unlinkSync(dest)
					downloadFile(response.headers.location, dest).then(resolve).catch(reject)
					return
				}

				if (response.statusCode !== 200) {
					file.close()
					fs.unlinkSync(dest)
					reject(new Error(`Download failed with status code: ${response.statusCode}`))
					return
				}

				response.pipe(file)

				file.on("finish", () => {
					file.close()
					resolve()
				})
			})
			.on("error", (err) => {
				fs.unlinkSync(dest)
				reject(err)
			})
	})
}

async function install() {
	try {
		const { platform, arch } = getPlatform()
		const binDir = path.join(__dirname, "bin")
		const binPath = path.join(binDir, `${BINARY_NAME}-go${platform === "windows" ? ".exe" : ""}`)

		// Create bin directory
		if (!fs.existsSync(binDir)) {
			fs.mkdirSync(binDir, { recursive: true })
		}

		// Check if binary already exists (from npm publish)
		if (fs.existsSync(binPath)) {
			console.log("Binary already exists, skipping download.")

			// Make executable on Unix
			if (platform !== "windows") {
				fs.chmodSync(binPath, 0o755)
			}

			return
		}

		// Download binary
		const url = getBinaryUrl(PACKAGE_VERSION, platform, arch)
		console.log(`Installing Cline CLI ${PACKAGE_VERSION} for ${platform}/${arch}...`)

		try {
			await downloadFile(url, binPath)
		} catch (err) {
			console.error(`Failed to download binary: ${err.message}`)
			console.error("")
			console.error("You can manually download the binary from:")
			console.error(`https://github.com/${GITHUB_REPO}/releases`)
			process.exit(1)
		}

		// Make executable on Unix
		if (platform !== "windows") {
			fs.chmodSync(binPath, 0o755)
		}

		console.log(`✓ Cline CLI installed successfully to ${binPath}`)
		console.log("")
		console.log("Usage:")
		console.log("  cline-go --help")
		console.log('  cline-go "Your prompt here"')
	} catch (error) {
		console.error("Installation failed:", error.message)
		process.exit(1)
	}
}

// Run installation
install()
