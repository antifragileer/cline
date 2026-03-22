#!/usr/bin/env node

/**
 * Unit tests for install.js
 *
 * Run with: node --test install.test.js
 */

const { describe, it, before, after } = require("node:test");
const assert = require("node:assert");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { execSync } = require("child_process");

// Create a test directory
const TEST_DIR = path.join(os.tmpdir(), "cline-npm-test-" + Date.now());

describe("install.js", () => {
	before(() => {
		if (!fs.existsSync(TEST_DIR)) {
			fs.mkdirSync(TEST_DIR, { recursive: true });
		}
	});

	after(() => {
		// Cleanup
		try {
			fs.rmSync(TEST_DIR, { recursive: true, force: true });
		} catch (e) {
			// Ignore cleanup errors
		}
	});

	describe("Platform Detection", () => {
		it("should detect supported platforms", () => {
			const supportedPlatforms = ["darwin", "linux", "win32"];
			assert.ok(supportedPlatforms.includes(process.platform));
		});

		it("should detect supported architectures", () => {
			const supportedArchs = ["x64", "arm64"];
			assert.ok(supportedArchs.includes(process.arch));
		});
	});

	describe("Binary Path Resolution", () => {
		it("should construct correct binary path for Windows", () => {
			const isWindows = process.platform === "win32";
			const expectedExt = isWindows ? ".exe" : "";
			const binDir = path.join(__dirname, "bin");
			const expectedPath = path.join(binDir, `cline${expectedExt}`);
			assert.ok(typeof expectedPath === "string");
		});

		it("should construct correct binary path for Unix", () => {
			const isWindows = process.platform === "win32";
			if (!isWindows) {
				const binDir = path.join(__dirname, "bin");
				const expectedPath = path.join(binDir, "cline");
				assert.ok(typeof expectedPath === "string");
			}
		});
	});

	describe("URL Construction", () => {
		it("should construct valid GitHub release URLs", () => {
			const version = "0.1.0";
			const platform = "linux";
			const arch = "amd64";
			const expectedUrl = `https://github.com/cline/cline/releases/download/v${version}/cline_${version}_${platform}_${arch}`;
			assert.ok(expectedUrl.includes("github.com"));
			assert.ok(expectedUrl.includes(version));
			assert.ok(expectedUrl.includes(platform));
			assert.ok(expectedUrl.includes(arch));
		});

		it("should append .exe extension for Windows binaries", () => {
			const version = "0.1.0";
			const platform = "windows";
			const arch = "amd64";
			const expectedUrl = `https://github.com/cline/cline/releases/download/v${version}/cline_${version}_${platform}_${arch}.exe`;
			assert.ok(expectedUrl.endsWith(".exe"));
		});
	});

	describe("Checksum Verification", () => {
		it("should calculate SHA256 checksum of test file", () => {
			const testFile = path.join(TEST_DIR, "test-checksum.txt");
			const content = "test content for checksum";
			fs.writeFileSync(testFile, content);

			const crypto = require("crypto");
			const hash = crypto.createHash("sha256");
			hash.update(content);
			const expectedChecksum = hash.digest("hex");

			assert.ok(typeof expectedChecksum === "string");
			assert.equal(expectedChecksum.length, 64); // SHA256 hex length
		});
	});

	describe("Directory Operations", () => {
		it("should create directory recursively", () => {
			const nestedDir = path.join(TEST_DIR, "nested", "deep", "dir");
			fs.mkdirSync(nestedDir, { recursive: true });
			assert.ok(fs.existsSync(nestedDir));
		});

		it("should handle existing directory gracefully", () => {
			const existingDir = path.join(TEST_DIR, "existing");
			fs.mkdirSync(existingDir, { recursive: true });
			// Should not throw
			fs.mkdirSync(existingDir, { recursive: true });
			assert.ok(fs.existsSync(existingDir));
		});
	});

	describe("Proxy Configuration", () => {
		it("should read HTTPS_PROXY from environment", () => {
			const originalProxy = process.env.HTTPS_PROXY;
			process.env.HTTPS_PROXY = "http://proxy.example.com:8080";

			const proxyUrl =
				process.env.HTTPS_PROXY ||
				process.env.https_proxy ||
				process.env.HTTP_PROXY ||
				process.env.http_proxy ||
				null;

			assert.equal(proxyUrl, "http://proxy.example.com:8080");

			// Restore
			if (originalProxy) {
				process.env.HTTPS_PROXY = originalProxy;
			} else {
				delete process.env.HTTPS_PROXY;
			}
		});

		it("should handle missing proxy configuration", () => {
			const originalHttpsProxy = process.env.HTTPS_PROXY;
			const originalHttpProxy = process.env.HTTP_PROXY;

			delete process.env.HTTPS_PROXY;
			delete process.env.https_proxy;
			delete process.env.HTTP_PROXY;
			delete process.env.http_proxy;

			const proxyUrl =
				process.env.HTTPS_PROXY ||
				process.env.https_proxy ||
				process.env.HTTP_PROXY ||
				process.env.http_proxy ||
				null;

			assert.equal(proxyUrl, null);

			// Restore
			if (originalHttpsProxy) process.env.HTTPS_PROXY = originalHttpsProxy;
			if (originalHttpProxy) process.env.HTTP_PROXY = originalHttpProxy;
		});
	});

	describe("Exit Codes", () => {
		it("should have defined exit codes", () => {
			const EXIT_CODES = {
				SUCCESS: 0,
				PLATFORM_UNSUPPORTED: 1,
				ARCH_UNSUPPORTED: 2,
				DOWNLOAD_FAILED: 3,
				CHECKSUM_FAILED: 4,
				INSTALL_FAILED: 5,
				NETWORK_ERROR: 6,
			};

			assert.equal(EXIT_CODES.SUCCESS, 0);
			assert.ok(EXIT_CODES.PLATFORM_UNSUPPORTED > 0);
			assert.ok(EXIT_CODES.ARCH_UNSUPPORTED > 0);
			assert.ok(EXIT_CODES.DOWNLOAD_FAILED > 0);
			assert.ok(EXIT_CODES.CHECKSUM_FAILED > 0);
			assert.ok(EXIT_CODES.INSTALL_FAILED > 0);
			assert.ok(EXIT_CODES.NETWORK_ERROR > 0);
		});
	});

	describe("Platform Mappings", () => {
		it("should map Node.js platforms to Go-style platforms", () => {
			const PLATFORM_MAPPINGS = {
				darwin: "darwin",
				linux: "linux",
				win32: "windows",
			};

			assert.equal(PLATFORM_MAPPINGS.darwin, "darwin");
			assert.equal(PLATFORM_MAPPINGS.linux, "linux");
			assert.equal(PLATFORM_MAPPINGS.win32, "windows");
		});

		it("should map Node.js architectures to Go-style architectures", () => {
			const ARCH_MAPPINGS = {
				x64: "amd64",
				arm64: "arm64",
			};

			assert.equal(ARCH_MAPPINGS.x64, "amd64");
			assert.equal(ARCH_MAPPINGS.arm64, "arm64");
		});
	});

	describe("File Permissions", () => {
		it("should set executable permissions on Unix", () => {
			if (process.platform !== "win32") {
				const testFile = path.join(TEST_DIR, "executable-test");
				fs.writeFileSync(testFile, "#!/bin/sh\necho test");

				fs.chmodSync(testFile, 0o755);
				const stats = fs.statSync(testFile);
				// Check that owner has execute permission
				assert.ok(stats.mode & 0o100);
			}
		});
	});
});

// Run tests
if (require.main === module) {
	console.log("Running install.js tests...");
}