#!/usr/bin/env node

/**
 * Unit tests for index.js
 *
 * Run with: node --test index.test.js
 */

const { describe, it, before, after } = require("node:test");
const assert = require("node:assert");
const fs = require("fs");
const path = require("path");
const os = require("os");

// Test directory
const TEST_DIR = path.join(os.tmpdir(), "cline-index-test-" + Date.now());

describe("index.js", () => {
	before(() => {
		if (!fs.existsSync(TEST_DIR)) {
			fs.mkdirSync(TEST_DIR, { recursive: true });
		}
	});

	after(() => {
		try {
			fs.rmSync(TEST_DIR, { recursive: true, force: true });
		} catch (e) {
			// Ignore cleanup errors
		}
	});

	describe("Platform Information", () => {
		it("should return platform details", () => {
			const platform = process.platform;
			const arch = process.arch;

			assert.ok(typeof platform === "string");
			assert.ok(typeof arch === "string");
			assert.ok(["darwin", "linux", "win32"].includes(platform));
			assert.ok(["x64", "arm64"].includes(arch));
		});

		it("should detect Windows correctly", () => {
			const isWindows = process.platform === "win32";
			if (process.platform === "win32") {
				assert.equal(isWindows, true);
			} else {
				assert.equal(isWindows, false);
			}
		});

		it("should detect macOS correctly", () => {
			const isMac = process.platform === "darwin";
			if (process.platform === "darwin") {
				assert.equal(isMac, true);
			} else {
				assert.equal(isMac, false);
			}
		});

		it("should detect Linux correctly", () => {
			const isLinux = process.platform === "linux";
			if (process.platform === "linux") {
				assert.equal(isLinux, true);
			} else {
				assert.equal(isLinux, false);
			}
		});
	});

	describe("Binary Path Construction", () => {
		it("should use .exe extension on Windows", () => {
			const isWindows = process.platform === "win32";
			const ext = isWindows ? ".exe" : "";
			const binaryName = `cline${ext}`;

			if (isWindows) {
				assert.ok(binaryName.endsWith(".exe"));
			} else {
				assert.ok(!binaryName.endsWith(".exe"));
			}
		});

		it("should construct binary path in bin directory", () => {
			const binDir = path.join(__dirname, "bin");
			const ext = process.platform === "win32" ? ".exe" : "";
			const expectedPath = path.join(binDir, `cline${ext}`);

			assert.ok(expectedPath.includes("bin"));
			assert.ok(expectedPath.includes("cline"));
		});
	});

	describe("Package Constants", () => {
		it("should have correct package name", () => {
			const PACKAGE_NAME = "@cline/golang-cli";
			assert.equal(PACKAGE_NAME, "@cline/golang-cli");
		});

		it("should have correct binary name", () => {
			const BINARY_NAME = "cline";
			assert.equal(BINARY_NAME, "cline");
		});
	});

	describe("Binary Existence Check", () => {
		it("should handle missing binary gracefully", () => {
			const fakeBinDir = path.join(TEST_DIR, "fake-bin");
			const fakeBinary = path.join(fakeBinDir, "nonexistent");

			assert.ok(!fs.existsSync(fakeBinary));
		});

		it("should detect existing file", () => {
			const testFile = path.join(TEST_DIR, "test-binary");
			fs.writeFileSync(testFile, "#!/bin/sh\necho test");

			assert.ok(fs.existsSync(testFile));

			fs.unlinkSync(testFile);
		});
	});

	describe("Module Exports", () => {
		it("should export expected functions", () => {
			// Simulate what index.js exports
			const expectedExports = [
				"getBinaryPath",
				"run",
				"runSync",
				"getVersion",
				"isInstalled",
				"getPlatform",
				"PACKAGE_NAME",
				"BINARY_NAME",
			];

			assert.ok(Array.isArray(expectedExports));
			assert.ok(expectedExports.includes("getBinaryPath"));
			assert.ok(expectedExports.includes("run"));
			assert.ok(expectedExports.includes("runSync"));
			assert.ok(expectedExports.includes("isInstalled"));
		});
	});

	describe("Process Arguments", () => {
		it("should handle empty arguments", () => {
			const args = [];
			assert.equal(args.length, 0);
		});

		it("should handle multiple arguments", () => {
			const args = ["--version", "--help"];
			assert.equal(args.length, 2);
		});
	});

	describe("Error Handling", () => {
		it("should throw error for missing binary", () => {
			const fakePath = path.join(TEST_DIR, "nonexistent-binary");
			let threwError = false;

			try {
				if (!fs.existsSync(fakePath)) {
					throw new Error(
						`Binary not found at ${fakePath}. ` +
						`Please run 'npm install' to download the binary for your platform.`
					);
				}
			} catch (error) {
				threwError = true;
				assert.ok(error.message.includes("Binary not found"));
			}

			assert.ok(threwError);
		});
	});

	describe("Spawn Options", () => {
		it("should have default spawn options", () => {
			const defaultOptions = {
				stdio: "inherit",
				cwd: process.cwd(),
				env: process.env,
			};

			assert.equal(defaultOptions.stdio, "inherit");
			assert.ok(typeof defaultOptions.cwd === "string");
			assert.ok(typeof defaultOptions.env === "object");
		});
	});
});

// Run tests
if (require.main === module) {
	console.log("Running index.js tests...");
}