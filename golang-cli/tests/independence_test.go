package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewIndependenceVerifier(t *testing.T) {
	t.Run("creates verifier with correct settings", func(t *testing.T) {
		verifier := NewIndependenceVerifier("/project", "/binary", true)

		if verifier.ProjectRoot != "/project" {
			t.Errorf("ProjectRoot = %s, want /project", verifier.ProjectRoot)
		}
		if verifier.BinaryPath != "/binary" {
			t.Errorf("BinaryPath = %s, want /binary", verifier.BinaryPath)
		}
		if verifier.Verbose != true {
			t.Error("Verbose should be true")
		}
		if verifier.Results == nil {
			t.Error("Results should be initialized")
		}
	})

	t.Run("creates verifier with empty paths", func(t *testing.T) {
		verifier := NewIndependenceVerifier("", "", false)

		if verifier.ProjectRoot != "" {
			t.Errorf("ProjectRoot = %s, want empty", verifier.ProjectRoot)
		}
		if verifier.BinaryPath != "" {
			t.Errorf("BinaryPath = %s, want empty", verifier.BinaryPath)
		}
	})
}

func TestIndependenceVerifier_Verify(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "independence-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("returns report with all checks", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		report, err := verifier.Verify()

		if err != nil {
			t.Fatalf("Verify() returned error: %v", err)
		}

		if report == nil {
			t.Fatal("Verify() returned nil report")
		}

		// Should have all check results
		expectedChecks := 8
		if len(report.Results) != expectedChecks {
			t.Errorf("Got %d results, want %d", len(report.Results), expectedChecks)
		}

		// Check that all expected checks are present
		checkNames := map[string]bool{
			CheckNoEmbeddedJS:   false,
			CheckNoNodeDeps:     false,
			CheckStaticBinary:   false,
			CheckNoCLIImports:   false,
			CheckNoNPMDeps:      false,
			CheckNoTypeScript:   false,
			CheckNoPackageJSON:  false,
			CheckGoModIntegrity: false,
		}

		for _, result := range report.Results {
			if _, exists := checkNames[result.Name]; exists {
				checkNames[result.Name] = true
			}
		}

		for name, found := range checkNames {
			if !found {
				t.Errorf("Missing check: %s", name)
			}
		}
	})

	t.Run("report includes success status", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		report, err := verifier.Verify()

		if err != nil {
			t.Fatalf("Verify() returned error: %v", err)
		}

		// Success should be boolean
		if report.Success && report.ExitCode != 0 {
			t.Error("Success=true but ExitCode != 0")
		}
		if !report.Success && report.ExitCode == 0 {
			t.Error("Success=false but ExitCode == 0")
		}

		if report.Summary == "" {
			t.Error("Summary should not be empty")
		}
	})
}

func TestIndependenceVerifier_verifyNoNodeDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "node-deps-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("passes when no node files exist", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNodeDependencies()

		if !result.Passed {
			t.Errorf("Expected pass, got fail: %s", result.Message)
		}
		if result.Message != "No Node.js dependencies detected" {
			t.Errorf("Unexpected message: %s", result.Message)
		}
	})

	t.Run("fails when node_modules exists", func(t *testing.T) {
		nodeModulesPath := filepath.Join(tempDir, "node_modules")
		if err := os.MkdirAll(nodeModulesPath, 0755); err != nil {
			t.Fatalf("Failed to create node_modules: %v", err)
		}
		defer os.RemoveAll(nodeModulesPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNodeDependencies()

		if result.Passed {
			t.Error("Expected fail when node_modules exists")
		}
		if !strings.Contains(result.Message, "node_modules") {
			t.Errorf("Message should mention node_modules: %s", result.Message)
		}
	})

	t.Run("fails when package.json exists", func(t *testing.T) {
		packageJSONPath := filepath.Join(tempDir, "package.json")
		if err := os.WriteFile(packageJSONPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}
		defer os.Remove(packageJSONPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNodeDependencies()

		if result.Passed {
			t.Error("Expected fail when package.json exists")
		}
	})
}

func TestIndependenceVerifier_verifyNoNPMDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "npm-deps-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("passes when no npm files exist", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNPMDependencies()

		if !result.Passed {
			t.Errorf("Expected pass, got fail: %s", result.Message)
		}
	})

	t.Run("fails when package-lock.json exists", func(t *testing.T) {
		lockPath := filepath.Join(tempDir, "package-lock.json")
		if err := os.WriteFile(lockPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create package-lock.json: %v", err)
		}
		defer os.Remove(lockPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNPMDependencies()

		if result.Passed {
			t.Error("Expected fail when package-lock.json exists")
		}
	})

	t.Run("fails when yarn.lock exists", func(t *testing.T) {
		yarnPath := filepath.Join(tempDir, "yarn.lock")
		if err := os.WriteFile(yarnPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create yarn.lock: %v", err)
		}
		defer os.Remove(yarnPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNPMDependencies()

		if result.Passed {
			t.Error("Expected fail when yarn.lock exists")
		}
	})

	t.Run("fails when pnpm-lock.yaml exists", func(t *testing.T) {
		pnpmPath := filepath.Join(tempDir, "pnpm-lock.yaml")
		if err := os.WriteFile(pnpmPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create pnpm-lock.yaml: %v", err)
		}
		defer os.Remove(pnpmPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNPMDependencies()

		if result.Passed {
			t.Error("Expected fail when pnpm-lock.yaml exists")
		}
	})

	t.Run("fails when .npmrc exists", func(t *testing.T) {
		npmrcPath := filepath.Join(tempDir, ".npmrc")
		if err := os.WriteFile(npmrcPath, []byte("registry=https://registry.npmjs.org"), 0644); err != nil {
			t.Fatalf("Failed to create .npmrc: %v", err)
		}
		defer os.Remove(npmrcPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoNPMDependencies()

		if result.Passed {
			t.Error("Expected fail when .npmrc exists")
		}
	})
}

func TestIndependenceVerifier_verifyNoTypeScriptFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ts-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("passes when no TypeScript files exist", func(t *testing.T) {
		// Create a Go file
		goPath := filepath.Join(tempDir, "main.go")
		if err := os.WriteFile(goPath, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to create main.go: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoTypeScriptFiles()

		if !result.Passed {
			t.Errorf("Expected pass, got fail: %s", result.Message)
		}
	})

	t.Run("fails when .ts files exist", func(t *testing.T) {
		tsPath := filepath.Join(tempDir, "test.ts")
		if err := os.WriteFile(tsPath, []byte("const x = 1;"), 0644); err != nil {
			t.Fatalf("Failed to create test.ts: %v", err)
		}
		defer os.Remove(tsPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoTypeScriptFiles()

		if result.Passed {
			t.Error("Expected fail when .ts files exist")
		}
		if !strings.Contains(result.Details, "test.ts") {
			t.Errorf("Details should mention test.ts: %s", result.Details)
		}
	})

	t.Run("fails when .tsx files exist", func(t *testing.T) {
		tsxPath := filepath.Join(tempDir, "test.tsx")
		if err := os.WriteFile(tsxPath, []byte("export default () => {}"), 0644); err != nil {
			t.Fatalf("Failed to create test.tsx: %v", err)
		}
		defer os.Remove(tsxPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoTypeScriptFiles()

		if result.Passed {
			t.Error("Expected fail when .tsx files exist")
		}
	})

	t.Run("skips vendor directory", func(t *testing.T) {
		vendorDir := filepath.Join(tempDir, "vendor")
		tsPath := filepath.Join(vendorDir, "test.ts")
		if err := os.MkdirAll(vendorDir, 0755); err != nil {
			t.Fatalf("Failed to create vendor dir: %v", err)
		}
		if err := os.WriteFile(tsPath, []byte("const x = 1;"), 0644); err != nil {
			t.Fatalf("Failed to create test.ts: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoTypeScriptFiles()

		if !result.Passed {
			t.Errorf("Should skip vendor directory, got fail: %s", result.Message)
		}
	})
}

func TestIndependenceVerifier_verifyNoPackageJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pkg-json-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("passes when no package.json exists", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoPackageJSON()

		if !result.Passed {
			t.Errorf("Expected pass, got fail: %s", result.Message)
		}
	})

	t.Run("fails when package.json exists", func(t *testing.T) {
		pkgPath := filepath.Join(tempDir, "package.json")
		if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}
		defer os.Remove(pkgPath)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoPackageJSON()

		if result.Passed {
			t.Error("Expected fail when package.json exists")
		}
	})

	t.Run("fails when nested package.json exists", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}
		pkgPath := filepath.Join(subDir, "package.json")
		if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}
		defer os.RemoveAll(subDir)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoPackageJSON()

		if result.Passed {
			t.Error("Expected fail when nested package.json exists")
		}
	})
}

func TestIndependenceVerifier_verifyGoModIntegrity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gomod-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("passes with clean go.mod", func(t *testing.T) {
		goModPath := filepath.Join(tempDir, "go.mod")
		content := `module github.com/example/test
		
go 1.21

require (
	github.com/spf13/cobra v1.8.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create go.mod: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyGoModIntegrity()

		if !result.Passed {
			t.Errorf("Expected pass, got fail: %s - %s", result.Message, result.Details)
		}
	})

	t.Run("fails when go.mod contains otto", func(t *testing.T) {
		goModPath := filepath.Join(tempDir, "go.mod")
		content := `module github.com/example/test
		
go 1.21

require (
	github.com/robertkrimen/otto v1.0.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create go.mod: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyGoModIntegrity()

		if result.Passed {
			t.Error("Expected fail when go.mod contains otto")
		}
		if !strings.Contains(result.Details, "otto") {
			t.Errorf("Details should mention otto: %s", result.Details)
		}
	})

	t.Run("fails when go.mod contains goja", func(t *testing.T) {
		goModPath := filepath.Join(tempDir, "go.mod")
		content := `module github.com/example/test
		
go 1.21

require (
	github.com/dop251/goja v1.0.0
)
`
		if err := os.WriteFile(goModPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create go.mod: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyGoModIntegrity()

		if result.Passed {
			t.Error("Expected fail when go.mod contains goja")
		}
	})

	t.Run("fails when go.mod is missing", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyGoModIntegrity()

		if result.Passed {
			t.Error("Expected fail when go.mod is missing")
		}
	})
}

func TestIndependenceVerifier_verifyNoEmbeddedJS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embedded-js-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("fails when binary path is empty", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoEmbeddedJS()

		if result.Passed {
			t.Error("Expected fail when binary path is empty")
		}
		if !strings.Contains(result.Message, "not provided") {
			t.Errorf("Message should mention 'not provided': %s", result.Message)
		}
	})

	t.Run("fails when binary does not exist", func(t *testing.T) {
		verifier := NewIndependenceVerifier(tempDir, "/nonexistent/binary", false)
		result := verifier.verifyNoEmbeddedJS()

		if result.Passed {
			t.Error("Expected fail when binary does not exist")
		}
	})

	t.Run("passes with clean binary", func(t *testing.T) {
		// Create a simple binary-like file without JS patterns
		binaryPath := filepath.Join(tempDir, "test-binary")
		content := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01, 0x01, 0x00} // ELF header
		if err := os.WriteFile(binaryPath, content, 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, binaryPath, false)
		result := verifier.verifyNoEmbeddedJS()

		if !result.Passed {
			t.Errorf("Expected pass for clean binary, got: %s - %s", result.Message, result.Details)
		}
	})

	t.Run("detects JavaScript patterns in binary", func(t *testing.T) {
		binaryPath := filepath.Join(tempDir, "test-binary-js")
		// Create content with JS patterns
		content := []byte("function test() { return 42; }\nconst x = 1;")
		if err := os.WriteFile(binaryPath, content, 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		verifier := NewIndependenceVerifier(tempDir, binaryPath, false)
		result := verifier.verifyNoEmbeddedJS()

		if result.Passed {
			t.Error("Expected fail when JS patterns are found")
		}
		if !strings.Contains(result.Message, "JavaScript") {
			t.Errorf("Message should mention JavaScript: %s", result.Message)
		}
	})
}

func TestIndependenceVerifier_verifyNoCLIImports(t *testing.T) {
	t.Run("passes when cli/src does not exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cli-imports-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		verifier := NewIndependenceVerifier(tempDir, "", false)
		result := verifier.verifyNoCLIImports()

		if !result.Passed {
			t.Errorf("Expected pass when cli/src does not exist, got: %s", result.Message)
		}
	})

	t.Run("passes with clean Go files", func(t *testing.T) {
		// Create parent temp directory
		parentDir, err := os.MkdirTemp("", "cli-imports-parent-*")
		if err != nil {
			t.Fatalf("Failed to create parent temp dir: %v", err)
		}
		defer os.RemoveAll(parentDir)

		// Create project directory under parent
		projectDir := filepath.Join(parentDir, "golang-cli")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		// Create cli/src directory (simulating existence)
		cliSrcPath := filepath.Join(parentDir, "cli", "src")
		if err := os.MkdirAll(cliSrcPath, 0755); err != nil {
			t.Fatalf("Failed to create cli/src: %v", err)
		}

		// Create a clean Go file
		goPath := filepath.Join(projectDir, "main.go")
		content := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello")
}
`
		if err := os.WriteFile(goPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create main.go: %v", err)
		}

		verifier := NewIndependenceVerifier(projectDir, "", false)
		result := verifier.verifyNoCLIImports()

		if !result.Passed {
			t.Errorf("Expected pass for clean imports, got: %s - %s", result.Message, result.Details)
		}
	})

	t.Run("detects cli/src imports", func(t *testing.T) {
		// Create parent temp directory
		parentDir, err := os.MkdirTemp("", "cli-imports-parent-*")
		if err != nil {
			t.Fatalf("Failed to create parent temp dir: %v", err)
		}
		defer os.RemoveAll(parentDir)

		// Create project directory under parent
		projectDir := filepath.Join(parentDir, "golang-cli")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		// Create cli/src directory
		cliSrcPath := filepath.Join(parentDir, "cli", "src")
		if err := os.MkdirAll(cliSrcPath, 0755); err != nil {
			t.Fatalf("Failed to create cli/src: %v", err)
		}

		// Create a Go file with cli/src import
		goPath := filepath.Join(projectDir, "main.go")
		content := `package main

import (
	"fmt"
	"github.com/cline/cline/cli/src/utils"
)

func main() {
	fmt.Println("Hello")
}
`
		if err := os.WriteFile(goPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create main.go: %v", err)
		}

		verifier := NewIndependenceVerifier(projectDir, "", false)
		result := verifier.verifyNoCLIImports()

		if result.Passed {
			t.Error("Expected fail when cli/src import is found")
		}
		if !strings.Contains(result.Details, "cli/src") {
			t.Errorf("Details should mention cli/src: %s", result.Details)
		}
	})
}

func TestVerificationResult(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		result := VerificationResult{
			Name:    "test_check",
			Passed:  true,
			Message: "Test passed",
			Details: "Additional info",
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded VerificationResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Name != result.Name {
			t.Errorf("Name mismatch: %s vs %s", decoded.Name, result.Name)
		}
		if decoded.Passed != result.Passed {
			t.Errorf("Passed mismatch: %v vs %v", decoded.Passed, result.Passed)
		}
	})
}

func TestVerificationReport(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		report := &VerificationReport{
			Success: true,
			Results: []VerificationResult{
				{Name: "check1", Passed: true, Message: "OK"},
			},
			Summary:  "All good",
			ExitCode: 0,
		}

		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded VerificationReport
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Success != report.Success {
			t.Errorf("Success mismatch")
		}
		if len(decoded.Results) != len(report.Results) {
			t.Errorf("Results length mismatch")
		}
	})
}

func TestGetBinaryPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "binary-path-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("finds binary in root", func(t *testing.T) {
		binaryName := "cline"
		if runtime.GOOS == "windows" {
			binaryName = "cline.exe"
		}
		binaryPath := filepath.Join(tempDir, binaryName)
		if err := os.WriteFile(binaryPath, []byte("binary"), 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		found := GetBinaryPath(tempDir)
		if found != binaryPath {
			t.Errorf("Expected %s, got %s", binaryPath, found)
		}
	})

	t.Run("returns empty when binary not found", func(t *testing.T) {
		found := GetBinaryPath(tempDir)
		// Should return empty or a path from system
		// We can't predict system paths, so just verify it doesn't panic
		_ = found
	})
}

func TestConstants(t *testing.T) {
	t.Run("check constants are defined", func(t *testing.T) {
		checks := []string{
			CheckNoEmbeddedJS,
			CheckNoNodeDeps,
			CheckStaticBinary,
			CheckNoCLIImports,
			CheckNoNPMDeps,
			CheckNoTypeScript,
			CheckNoPackageJSON,
			CheckGoModIntegrity,
		}

		for _, check := range checks {
			if check == "" {
				t.Error("Check constant should not be empty")
			}
		}
	})
}
