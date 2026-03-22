#!/usr/bin/env node

/**
 * NPM Install Script for Cline Go CLI
 *
 * This script downloads the appropriate platform-specific binary
 * during npm install. It handles:
 * - Platform detection (process.platform, process.arch)
 * - Binary download from GitHub releases
 * - Checksum verification
 * - Executable permission setting
 * - Proxy configuration support
 * - Error handling and cleanup
 */

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const https = require("https");
const http = require("http");
const { execSync } = require("child_process");

// Configuration
const BINARY_NAME = "cline";
const GITHUB_OWNER = "cline";
const GITHUB_REPO = "cline";
const VERSION = require("./package.json").version;

// Platform mappings
const PLATFORM_MAPPINGS = {
	darwin: "darwin",
	linux: "linux",
	win32: "windows",
};

const ARCH_MAPPINGS = {
	x64: "amd64",
	arm64: "arm64",
};

// Exit codes
const EXIT_CODES = {
	SUCCESS: 0,
	PLATFORM_UNSUPPORTED: 1,
	ARCH_UNSUPPORTED: 2,
	DOWNLOAD_FAILED: 3,
	CHECKSUM_FAILED: 4,
	INSTALL_FAILED: 5,
	NETWORK_ERROR: 6,
};

/**
 * Logger utility
 */
const logger = {
	info: (msg) => console.log(`[cline-install] ${msg}`),
	error: (msg) => console.error(`[cline-install] ERROR: ${msg}`),
	warn: (msg) => console.warn(`[cline-install] WARNING: ${msg}`),
	debug: (msg) => {
		if (process.env.DEBUG) {
			console.log(`[cline-install] DEBUG: ${msg}`);
		}
	},
};

/**
 * Get platform and architecture
 */
function getPlatformInfo() {
	const platform = process.platform;
	const arch = process.arch;

	logger.debug(`Detected platform: ${platform}, arch: ${arch}`);

	const mappedPlatform = PLATFORM_MAPPINGS[platform];
	const mappedArch = ARCH_MAPPINGS[arch];

	if (!mappedPlatform) {
		logger.error(`Unsupported platform: ${platform}`);
		logger.error(`Supported platforms: ${Object.keys(PLATFORM_MAPPINGS).join(", ")}`);
		process.exit(EXIT_CODES.PLATFORM_UNSUPPORTED);
	}

	if (!mappedArch) {
		logger.error(`Unsupported architecture: ${arch}`);
		logger.error(`Supported architectures: ${Object.keys(ARCH_MAPPINGS).join(", ")}`);
		process.exit(EXIT_CODES.ARCH_UNSUPPORTED);
	}

	return {
		platform: mappedPlatform,
		arch: mappedArch,
		isWindows: platform === "win32",
	};
}

/**
 * Get proxy configuration from environment
 */
function getProxyConfig() {
	const proxyUrl =
		process.env.HTTPS_PROXY ||
		process.env.https_proxy ||
		process.env.HTTP_PROXY ||
		process.env.http_proxy ||
		null;

	if (proxyUrl) {
		logger.debug(`Using proxy: ${proxyUrl}`);
	}

	return proxyUrl;
}

/**
 * Parse proxy URL
 */
function parseProxyUrl(proxyUrl) {
	try {
		const url = new URL(proxyUrl);
		return {
			host: url.hostname,
			port: parseInt(url.port, 10) || (url.protocol === "https:" ? 443 : 80),
			protocol: url.protocol,
			auth: url.username
				? {
						username: url.username,
						password: url.password,
					}
				: null,
		};
	} catch (error) {
		logger.warn(`Failed to parse proxy URL: ${error.message}`);
		return null;
	}
}

/**
 * Download file with proxy support
 */
function downloadFile(url, destPath, proxyUrl) {
	return new Promise((resolve, reject) => {
		const file = fs.createWriteStream(destPath);
		const urlObj = new URL(url);

		let requestOptions = {
			hostname: urlObj.hostname,
			port: urlObj.port || 443,
			path: urlObj.pathname + urlObj.search,
			method: "GET",
			headers: {
				"User-Agent": `cline-npm-install/${VERSION}`,
			},
		};

		// Configure proxy if provided
		let client = https;
		if (proxyUrl) {
			const proxyConfig = parseProxyUrl(proxyUrl);
			if (proxyConfig) {
				requestOptions = {
					hostname: proxyConfig.host,
					port: proxyConfig.port,
					path: url,
					method: "GET",
					headers: {
						"User-Agent": `cline-npm-install/${VERSION}`,
						Host: urlObj.hostname,
					},
				};

				if (proxyConfig.auth) {
					const auth = Buffer.from(
						`${proxyConfig.auth.username}:${proxyConfig.auth.password}`,
					).toString("base64");
					requestOptions.headers["Proxy-Authorization"] = `Basic ${auth}`;
				}

				client = proxyConfig.protocol === "https:" ? https : http;
			}
		}

		const request = client.request(requestOptions, (response) => {
			if (response.statusCode === 301 || response.statusCode === 302) {
				// Follow redirects
				logger.debug(`Following redirect to: ${response.headers.location}`);
				downloadFile(response.headers.location, destPath, proxyUrl)
					.then(resolve)
					.catch(reject);
				return;
			}

			if (response.statusCode !== 200) {
				reject(
					new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`),
				);
				return;
			}

			const totalSize = parseInt(response.headers["content-length"], 10) || 0;
			let downloadedSize = 0;
			let lastProgress = 0;

			response.on("data", (chunk) => {
				downloadedSize += chunk.length;
				if (totalSize > 0) {
					const progress = Math.round((downloadedSize / totalSize) * 100);
					if (progress >= lastProgress + 10) {
						logger.debug(`Download progress: ${progress}%`);
						lastProgress = progress;
					}
				}
			});

			response.pipe(file);

			file.on("finish", () => {
				file.close();
				resolve();
			});
		});

		request.on("error", (error) => {
			fs.unlink(destPath, () => {}); // Clean up on error
			reject(error);
		});

		request.setTimeout(30000, () => {
			request.destroy();
			fs.unlink(destPath, () => {}); // Clean up on timeout
			reject(new Error("Download timeout after 30 seconds"));
		});

		request.end();
	});
}

/**
 * Calculate SHA256 checksum of a file
 */
function calculateChecksum(filePath) {
	return new Promise((resolve, reject) => {
		const hash = crypto.createHash("sha256");
		const stream = fs.createReadStream(filePath);

		stream.on("error", reject);
		stream.on("data", (chunk) => hash.update(chunk));
		stream.on("end", () => resolve(hash.digest("hex")));
	});
}

/**
 * Download checksum file and verify
 */
async function verifyChecksum(binaryPath, checksumUrl, proxyUrl) {
	logger.info("Verifying checksum...");

	const checksumPath = `${binaryPath}.sha256`;

	try {
		await downloadFile(checksumUrl, checksumPath, proxyUrl);

		const checksumContent = fs.readFileSync(checksumPath, "utf8").trim();
		const expectedChecksum = checksumContent.split(/\s+/)[0];

		const actualChecksum = await calculateChecksum(binaryPath);

		fs.unlinkSync(checksumPath); // Clean up checksum file

		if (actualChecksum !== expectedChecksum) {
			throw new Error(
				`Checksum mismatch: expected ${expectedChecksum}, got ${actualChecksum}`,
			);
		}

		logger.debug("Checksum verified successfully");
		return true;
	} catch (error) {
		fs.unlink(checksumPath, () => {}); // Clean up on error
		throw new Error(`Checksum verification failed: ${error.message}`);
	}
}

/**
 * Set executable permissions
 */
function setExecutablePermissions(filePath, isWindows) {
	if (isWindows) {
		// Windows doesn't need executable permissions
		return;
	}

	try {
		fs.chmodSync(filePath, 0o755);
		logger.debug("Set executable permissions (755)");
	} catch (error) {
		throw new Error(
			`Failed to set executable permissions: ${error.message}`,
		);
	}
}

/**
 * Ensure directory exists
 */
function ensureDir(dirPath) {
	if (!fs.existsSync(dirPath)) {
		fs.mkdirSync(dirPath, { recursive: true });
	}
}

/**
 * Get binary download URL
 */
function getBinaryUrl(platform, arch, version) {
	const ext = platform === "windows" ? ".exe" : "";
	const filename = `${BINARY_NAME}_${version}_${platform}_${arch}${ext}`;
	return `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/v${version}/${filename}`;
}

/**
 * Get checksum URL
 */
function getChecksumUrl(platform, arch, version) {
	return `${getBinaryUrl(platform, arch, version)}.sha256`;
}

/**
 * Main installation function
 */
async function install() {
	logger.info(`Installing Cline Go CLI v${VERSION}...`);

	const { platform, arch, isWindows } = getPlatformInfo();
	const proxyUrl = getProxyConfig();

	const binDir = path.join(__dirname, "bin");
	const ext = isWindows ? ".exe" : "";
	const binaryName = `${BINARY_NAME}${ext}`;
	const binaryPath = path.join(binDir, binaryName);

	// Ensure bin directory exists
	ensureDir(binDir);

	// Check if binary already exists
	if (fs.existsSync(binaryPath)) {
		logger.info("Binary already exists, skipping download");
		console.log(`[cline-install] Installed at: ${binaryPath}`);
		process.exit(EXIT_CODES.SUCCESS);
	}

	const binaryUrl = getBinaryUrl(platform, arch, VERSION);
	const checksumUrl = getChecksumUrl(platform, arch, VERSION);

	logger.info(`Downloading binary for ${platform}/${arch}...`);
	logger.debug(`URL: ${binaryUrl}`);

	try {
		await downloadFile(binaryUrl, binaryPath, proxyUrl);
		logger.info("Binary downloaded successfully");
	} catch (error) {
		logger.error(`Failed to download binary: ${error.message}`);
		fs.unlink(binaryPath, () => {}); // Clean up

		// Provide helpful error message for common issues
		if (error.message.includes("404")) {
			logger.error("Binary not found for your platform/architecture");
			logger.error("This may mean:");
			logger.error("  1. The release hasn't been published yet");
			logger.error("  2. Your platform/architecture is not supported");
			logger.error(`  Platform: ${platform}, Architecture: ${arch}`);
		}

		process.exit(EXIT_CODES.DOWNLOAD_FAILED);
	}

	// Verify checksum
	try {
		await verifyChecksum(binaryPath, checksumUrl, proxyUrl);
	} catch (error) {
		logger.error(error.message);
		fs.unlink(binaryPath, () => {}); // Clean up
		process.exit(EXIT_CODES.CHECKSUM_FAILED);
	}

	// Set executable permissions
	try {
		setExecutablePermissions(binaryPath, isWindows);
	} catch (error) {
		logger.error(error.message);
		fs.unlink(binaryPath, () => {}); // Clean up
		process.exit(EXIT_CODES.INSTALL_FAILED);
	}

	logger.info("Installation complete!");
	console.log(`[cline-install] Installed at: ${binaryPath}`);

	// Verify installation
	try {
		const version = execSync(`"${binaryPath}" --version`, { encoding: "utf8" });
		logger.info(`Verified: ${version.trim()}`);
	} catch (error) {
		logger.warn("Could not verify binary execution, but installation appears successful");
	}

	process.exit(EXIT_CODES.SUCCESS);
}

// Handle uncaught errors
process.on("uncaughtException", (error) => {
	logger.error(`Unexpected error: ${error.message}`);
	process.exit(EXIT_CODES.INSTALL_FAILED);
});

process.on("unhandledRejection", (reason, promise) => {
	logger.error(`Unhandled rejection at: ${promise}, reason: ${reason}`);
	process.exit(EXIT_CODES.INSTALL_FAILED);
});

// Run installation
install().catch((error) => {
	logger.error(`Installation failed: ${error.message}`);
	process.exit(EXIT_CODES.INSTALL_FAILED);
});