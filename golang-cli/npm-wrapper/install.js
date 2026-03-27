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
const crypto = require("crypto")
const { execSync } = require("child_process")

const platform = require("./platform")
const PACKAGE_VERSION = require("./package.json").version
const GITHUB_REPO = "cline/cline"

/**
 * Download a file from a URL to a destination
 * @param {string} url - The URL to download from
 * @param {string} dest - The destination path
 * @param {Object} options - Download options
 * @returns {Promise<void>}
 */
function downloadFile(url, dest, options = {}) {
	return new Promise((resolve, reject) => {
		console.log(`Downloading from: ${url}`)

		const file = fs.createWriteStream(dest)
		const timeout = options.timeout || 5 * 60 * 1000 // 5 minutes default

		const request = https
			.get(url, { timeout }, (response) => {
				// Handle redirects
				if (response.statusCode === 302 || response.statusCode === 301) {
					file.close()
					fs.unlinkSync(dest)
					downloadFile(response.headers.location, dest, options).then(resolve).catch(reject)
					return
				}

				if (response.statusCode !== 200) {
					file.close()
					fs.unlinkSync(dest)
					reject(new Error(`Download failed with status code: ${response.statusCode}`))
					return
				}

				// Track download progress
				const totalBytes = Number.parseInt(response.headers["content-length"], 10)
				let downloadedBytes = 0
				let lastProgress = 0

				response.on("data", (chunk) => {
					downloadedBytes += chunk.length
					if (totalBytes && !options.quiet) {
						const progress = Math.round((downloadedBytes / totalBytes) * 100)
						if (progress - lastProgress >= 10) {
							console.log(`  Download progress: ${progress}%`)
							lastProgress = progress
						}
					}
				})

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

		request.on("timeout", () => {
			request.destroy()
			fs.unlinkSync(dest)
			reject(new Error("Download timeout"))
		})
	})
}

/**
 * Verify SHA256 checksum of a file
 * @param {string} filePath - Path to the file
 * @param {string} expectedChecksum - Expected SHA256 checksum
 * @returns {Promise<boolean>}
 */
async function verifyChecksum(filePath, expectedChecksum) {
	return new Promise((resolve, reject) => {
		const hash = crypto.createHash("sha256")
		const stream = fs.createReadStream(filePath)

		stream.on("error", reject)
		stream.on("data", (chunk) => hash.update(chunk))
		stream.on("end", () => {
			const actualChecksum = hash.digest("hex")
			resolve(actualChecksum.toLowerCase() === expectedChecksum.toLowerCase())
		})
	})
}

/**
 * Download and verify checksum file
 * @param {string} checksumUrl - URL of the checksum file
 * @returns {Promise<string|null>} The expected checksum or null
 */
async function downloadChecksum(checksumUrl) {
	try {
		const tempFile = path.join(require("os").tmpdir(), `cline-checksum-${Date.now()}.txt`)
		await downloadFile(checksumUrl, tempFile, { quiet: true, timeout: 30000 })

		const content = fs.readFileSync(tempFile, "utf8")
		fs.unlinkSync(tempFile)

		// Parse checksum file (format: "<hash>  <filename>")
		const parts = content.trim().split(/\s+/)
		return parts[0] || null
	} catch (err) {
		console.warn(`  Warning: Could not download checksum: ${err.message}`)
		return null
	}
}

/**
 * Extract a tar.gz archive
 * @param {string} archivePath - Path to the archive
 * @param {string} destDir - Destination directory
 */
function extractTarGz(archivePath, destDir) {
	try {
		// Try using tar command first
		execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: "pipe" })
	} catch (err) {
		// Fallback: try using Node.js built-in modules
		console.log("  Using Node.js extraction fallback...")

		const zlib = require("zlib")
		const tar = require("tar") // Note: this would need to be a dependency

		// For now, throw error if tar command fails
		throw new Error(`Failed to extract archive: ${err.message}\n` + `Please ensure 'tar' is installed on your system.`)
	}
}

/**
 * Make a file executable (Unix only)
 * @param {string} filePath - Path to the file
 */
function makeExecutable(filePath) {
	if (process.platform !== "win32") {
		fs.chmodSync(filePath, 0o755)
	}
}

/**
 * Main installation function
 */
async function install() {
	console.log(`\n=== Cline CLI Installation ===\n`)

	try {
		// Check platform support
		if (!platform.isSupported()) {
			throw new Error("Your platform is not supported")
		}

		const { platform: plat, arch } = platform.getPlatform()
		console.log(`Platform: ${plat}/${arch}`)
		console.log(`Version: ${PACKAGE_VERSION}\n`)

		// Ensure binary directory exists
		platform.ensureBinaryDir()

		const binaryPath = platform.getBinaryPath()

		// Check if binary already exists (from npm publish with bundled binary)
		if (platform.binaryExists()) {
			console.log("Binary already exists, verifying...")

			// Make executable and verify
			makeExecutable(binaryPath)

			try {
				const version = execSync(`"${binaryPath}" --version`, {
					encoding: "utf8",
					timeout: 10000,
				}).trim()
				console.log(`✓ Binary verified: ${version}`)
				return
			} catch (err) {
				console.warn("  Warning: Existing binary verification failed, re-downloading...")
			}
		}

		// Download binary
		const downloadUrl = platform.getDownloadUrl(PACKAGE_VERSION)
		const tarballName = platform.getTarballName(PACKAGE_VERSION)
		const tempDir = path.join(require("os").tmpdir(), `cline-install-${Date.now()}`)
		const archivePath = path.join(tempDir, tarballName)

		// Create temp directory
		fs.mkdirSync(tempDir, { recursive: true })

		try {
			// Download tarball
			console.log(`Downloading Cline CLI ${PACKAGE_VERSION}...`)
			await downloadFile(downloadUrl, archivePath)

			// Download and verify checksum if available
			const checksumUrl = platform.getChecksumUrl(PACKAGE_VERSION)
			const expectedChecksum = await downloadChecksum(checksumUrl)

			if (expectedChecksum) {
				console.log("Verifying checksum...")
				const isValid = await verifyChecksum(archivePath, expectedChecksum)
				if (!isValid) {
					throw new Error("Checksum verification failed - download may be corrupted")
				}
				console.log("✓ Checksum verified")
			}

			// Extract archive
			console.log("Extracting binary...")
			extractTarGz(archivePath, tempDir)

			// Find extracted binary
			const extractedBinary = path.join(tempDir, "cline")
			if (!fs.existsSync(extractedBinary)) {
				// Try with extension for Windows
				const extractedBinaryWin = path.join(tempDir, "cline.exe")
				if (!fs.existsSync(extractedBinaryWin)) {
					throw new Error("Binary not found in extracted archive")
				}
			}

			// Move binary to final location
			console.log("Installing binary...")
			const finalBinaryPath = platform.getBinaryPath()

			// Remove existing binary if present
			if (fs.existsSync(finalBinaryPath)) {
				fs.unlinkSync(finalBinaryPath)
			}

			// Copy binary
			fs.copyFileSync(extractedBinary, finalBinaryPath)
			makeExecutable(finalBinaryPath)

			// Verify installation
			console.log("Verifying installation...")
			const version = execSync(`"${finalBinaryPath}" --version`, {
				encoding: "utf8",
				timeout: 10000,
			}).trim()

			console.log(`\n✓ Cline CLI ${PACKAGE_VERSION} installed successfully!`)
			console.log(`  Location: ${finalBinaryPath}`)
			console.log(`  Version: ${version}`)
			console.log(`\nUsage:`)
			console.log(`  cline --help`)
			console.log(`  cline "Your prompt here"`)
			console.log(`  cline auth`)
		} finally {
			// Cleanup temp directory
			try {
				fs.rmSync(tempDir, { recursive: true, force: true })
			} catch (err) {
				// Ignore cleanup errors
			}
		}
	} catch (error) {
		console.error("\n✗ Installation failed:", error.message)
		console.error("\nTroubleshooting:")
		console.error("  1. Check your internet connection")
		console.error("  2. Verify the release exists at:")
		console.error(`     https://github.com/${GITHUB_REPO}/releases`)
		console.error("  3. Try installing manually from the releases page")
		console.error("  4. For proxy issues, set HTTPS_PROXY environment variable")
		console.error("\nYou can also install Cline CLI manually:")
		console.error(`  curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | sh`)

		process.exit(1)
	}
}

// Run installation
install()
