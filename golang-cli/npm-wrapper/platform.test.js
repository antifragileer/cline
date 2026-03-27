#!/usr/bin/env node

/**
 * Tests for the platform detection module
 */

const platform = require("./platform")
const assert = require("assert")

console.log("Running platform detection tests...\n")

// Test 1: getPlatform
console.log("Test 1: getPlatform()")
try {
	const info = platform.getPlatform()
	assert(info.platform, "platform should be defined")
	assert(info.arch, "arch should be defined")
	assert(info.rawPlatform, "rawPlatform should be defined")
	assert(info.rawArch, "rawArch should be defined")
	console.log("  ✓ Returns platform info:", info)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 2: getBinaryName
console.log("\nTest 2: getBinaryName()")
try {
	const name = platform.getBinaryName()
	assert(typeof name === "string", "binary name should be a string")
	assert(name.startsWith("cline"), "binary name should start with 'cline'")
	console.log("  ✓ Binary name:", name)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 3: getTarballName
console.log("\nTest 3: getTarballName()")
try {
	const name = platform.getTarballName("1.0.0")
	assert(name.includes("1.0.0"), "tarball should include version")
	assert(name.endsWith(".tar.gz"), "tarball should end with .tar.gz")
	console.log("  ✓ Tarball name:", name)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 4: getDownloadUrl
console.log("\nTest 4: getDownloadUrl()")
try {
	const url = platform.getDownloadUrl("1.0.0")
	assert(url.includes("github.com"), "URL should point to GitHub")
	assert(url.includes("1.0.0"), "URL should include version")
	console.log("  ✓ Download URL:", url)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 5: getBinaryPath
console.log("\nTest 5: getBinaryPath()")
try {
	const path = platform.getBinaryPath()
	assert(typeof path === "string", "path should be a string")
	console.log("  ✓ Binary path:", path)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 6: getPlatformInfo
console.log("\nTest 6: getPlatformInfo()")
try {
	const info = platform.getPlatformInfo()
	assert(info.platform, "platform should be defined")
	assert(info.arch, "arch should be defined")
	assert(info.nodeVersion, "nodeVersion should be defined")
	console.log("  ✓ Platform info:", info)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 7: isSupported
console.log("\nTest 7: isSupported()")
try {
	const supported = platform.isSupported()
	assert(typeof supported === "boolean", "should return boolean")
	console.log("  ✓ Platform supported:", supported)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 8: getCacheDir
console.log("\nTest 8: getCacheDir()")
try {
	const dir = platform.getCacheDir()
	assert(typeof dir === "string", "cache dir should be a string")
	console.log("  ✓ Cache directory:", dir)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 9: getChecksumUrl
console.log("\nTest 9: getChecksumUrl()")
try {
	const url = platform.getChecksumUrl("1.0.0")
	assert(url.includes("sha256"), "checksum URL should include sha256")
	console.log("  ✓ Checksum URL:", url)
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 10: findInPath
console.log("\nTest 10: findInPath()")
try {
	// node should always be in PATH
	const nodePath = platform.findInPath("node")
	if (nodePath) {
		console.log("  ✓ Found 'node' in PATH:", nodePath)
	} else {
		console.log("  ℹ 'node' not found in PATH (this may be expected)")
	}
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

// Test 11: getCommonInstallPaths
console.log("\nTest 11: getCommonInstallPaths()")
try {
	const paths = platform.getCommonInstallPaths()
	assert(Array.isArray(paths), "should return array")
	console.log("  ✓ Common install paths:", paths.length, "paths found")
} catch (e) {
	console.error("  ✗ Failed:", e.message)
	process.exit(1)
}

console.log("\n✓ All tests passed!")
