// Package tests provides build verification utilities for the Go CLI.
// This package ensures the Go CLI remains independent of Node.js/TypeScript dependencies.
package tests

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// VerificationResult represents the result of a single verification check
type VerificationResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// VerificationReport contains all verification results
type VerificationReport struct {
	Success  bool                 `json:"success"`
	Results  []VerificationResult `json:"results"`
	Summary  string               `json:"summary"`
	ExitCode int                  `json:"exitCode"`
}

// Constants for verification checks
const (
	CheckNoEmbeddedJS      = "no_embedded_js"
	CheckNoNodeDeps        = "no_node_dependencies"
	CheckStaticBinary      = "static_binary"
	CheckNoCLIImports      = "no_cli_src_imports"
	CheckNoNPMDeps         = "no_npm_dependencies"
	CheckNoTypeScript      = "no_typescript_files"
	CheckNoPackageJSON     = "no_package_json"
	CheckGoModIntegrity    = "go_mod_integrity"
)

// IndependenceVerifier performs all independence verification checks
type IndependenceVerifier struct {
	ProjectRoot   string
	BinaryPath    string
	Verbose       bool
	Results       []VerificationResult
}

// NewIndependenceVerifier creates a new verifier instance
func NewIndependenceVerifier(projectRoot, binaryPath string, verbose bool) *IndependenceVerifier {
	return &IndependenceVerifier{
		ProjectRoot: projectRoot,
		BinaryPath:  binaryPath,
		Verbose:     verbose,
		Results:     make([]VerificationResult, 0),
	}
}

// Verify runs all independence checks and returns a report
func (v *IndependenceVerifier) Verify() (*VerificationReport, error) {
	v.Results = make([]VerificationResult, 0)

	// Run all verification checks
	checks := []func() VerificationResult{
		v.verifyNoEmbeddedJS,
		v.verifyNoNodeDependencies,
		v.verifyStaticBinary,
		v.verifyNoCLIImports,
		v.verifyNoNPMDependencies,
		v.verifyNoTypeScriptFiles,
		v.verifyNoPackageJSON,
		v.verifyGoModIntegrity,
	}

	allPassed := true
	for _, check := range checks {
		result := check()
		v.Results = append(v.Results, result)
		if !result.Passed {
			allPassed = false
		}
	}

	exitCode := 0
	summary := "All independence checks passed"
	if !allPassed {
		exitCode = 1
		summary = "Independence verification failed"
	}

	report := &VerificationReport{
		Success:  allPassed,
		Results:  v.Results,
		Summary:  summary,
		ExitCode: exitCode,
	}

	return report, nil
}

// verifyNoEmbeddedJS checks for embedded JavaScript/TypeScript content in the binary
func (v *IndependenceVerifier) verifyNoEmbeddedJS() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoEmbeddedJS,
		Passed: true,
	}

	if v.BinaryPath == "" {
		result.Passed = false
		result.Message = "Binary path not provided"
		result.Details = "Cannot check for embedded JS without a binary path"
		return result
	}

	// Check if binary exists
	if _, err := os.Stat(v.BinaryPath); os.IsNotExist(err) {
		result.Passed = false
		result.Message = "Binary not found"
		result.Details = fmt.Sprintf("Binary path does not exist: %s", v.BinaryPath)
		return result
	}

	// Read binary content
	data, err := os.ReadFile(v.BinaryPath)
	if err != nil {
		result.Passed = false
		result.Message = "Failed to read binary"
		result.Details = err.Error()
		return result
	}

	// Check for common JS/TS patterns
	jsPatterns := []string{
		`function\s+\w+\s*\(`,           // function declarations
		`const\s+\w+\s*=`,                // const declarations
		`let\s+\w+\s*=`,                  // let declarations
		`var\s+\w+\s*=`,                  // var declarations
		`module\.exports`,                // CommonJS
		`require\s*\(`,                   // require calls
		`import\s+.*\s+from\s+`,          // ES6 imports
		`export\s+(default\s+)?`,         // ES6 exports
		`console\.(log|error|warn)`,      // console usage
		`process\.env`,                   // process.env
		`__dirname`,                      // __dirname
		`__filename`,                     // __filename
		`=>`,                             // arrow functions
		`async\s+function`,               // async functions
		`await\s+`,                       // await keyword
	}

	content := string(data)
	foundPatterns := make([]string, 0)

	for _, pattern := range jsPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(content) {
			foundPatterns = append(foundPatterns, pattern)
		}
	}

	if len(foundPatterns) > 0 {
		result.Passed = false
		result.Message = "Potential JavaScript/TypeScript content detected in binary"
		result.Details = fmt.Sprintf("Found %d JS/TS patterns: %v", len(foundPatterns), foundPatterns)
	} else {
		result.Message = "No embedded JavaScript/TypeScript detected"
	}

	return result
}

// verifyNoNodeDependencies checks for Node.js runtime dependencies
func (v *IndependenceVerifier) verifyNoNodeDependencies() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoNodeDeps,
		Passed: true,
	}

	// Check for node_modules in project
	nodeModulesPath := filepath.Join(v.ProjectRoot, "node_modules")
	if info, err := os.Stat(nodeModulesPath); err == nil && info.IsDir() {
		result.Passed = false
		result.Message = "node_modules directory found in project"
		result.Details = fmt.Sprintf("node_modules exists at: %s", nodeModulesPath)
		return result
	}

	// Check for package.json in golang-cli directory
	packageJSONPath := filepath.Join(v.ProjectRoot, "package.json")
	if _, err := os.Stat(packageJSONPath); err == nil {
		result.Passed = false
		result.Message = "package.json found in Go CLI project"
		result.Details = fmt.Sprintf("package.json exists at: %s", packageJSONPath)
		return result
	}

	// Check for any Node.js shebangs in scripts
	scriptsDir := filepath.Join(v.ProjectRoot, "scripts")
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(scriptsDir, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			if strings.HasPrefix(string(content), "#!/usr/bin/env node") ||
				strings.HasPrefix(string(content), "#!/usr/bin/node") {
				result.Passed = false
				result.Message = "Node.js script found"
				result.Details = fmt.Sprintf("Script %s has Node.js shebang", filePath)
				return result
			}
		}
	}

	result.Message = "No Node.js dependencies detected"
	return result
}

// verifyStaticBinary checks if the binary is statically linked
func (v *IndependenceVerifier) verifyStaticBinary() VerificationResult {
	result := VerificationResult{
		Name:   CheckStaticBinary,
		Passed: true,
	}

	if v.BinaryPath == "" {
		result.Passed = false
		result.Message = "Binary path not provided"
		result.Details = "Cannot check static linking without a binary path"
		return result
	}

	if _, err := os.Stat(v.BinaryPath); os.IsNotExist(err) {
		result.Passed = false
		result.Message = "Binary not found"
		result.Details = fmt.Sprintf("Binary path does not exist: %s", v.BinaryPath)
		return result
	}

	// Check static linking based on OS
	switch runtime.GOOS {
	case "linux":
		return v.checkStaticLinux()
	case "darwin":
		return v.checkStaticDarwin()
	case "windows":
		return v.checkStaticWindows()
	default:
		result.Passed = true
		result.Message = fmt.Sprintf("Static binary check skipped on %s", runtime.GOOS)
		return result
	}
}

// checkStaticLinux checks if Linux binary is statically linked
func (v *IndependenceVerifier) checkStaticLinux() VerificationResult {
	result := VerificationResult{
		Name:   CheckStaticBinary,
		Passed: true,
	}

	file, err := elf.Open(v.BinaryPath)
	if err != nil {
		result.Passed = false
		result.Message = "Failed to open ELF binary"
		result.Details = err.Error()
		return result
	}
	defer file.Close()

	// Check for dynamic section
	if file.Section(".dynamic") != nil {
		// Has dynamic section, check if it's only using standard libraries
		libs, err := file.ImportedLibraries()
		if err != nil {
			result.Passed = false
			result.Message = "Failed to read imported libraries"
			result.Details = err.Error()
			return result
		}

		// Allow only standard system libraries
		allowedLibs := map[string]bool{
			"libc.so":         true,
			"libpthread.so":   true,
			"libdl.so":        true,
			"libm.so":         true,
			"libresolv.so":    true,
			"libc.so.6":       true,
			"libpthread.so.0": true,
			"libdl.so.2":      true,
			"libm.so.6":       true,
			"libresolv.so.2":  true,
		}

		unallowedLibs := make([]string, 0)
		for _, lib := range libs {
			found := false
			for allowed := range allowedLibs {
				if strings.HasPrefix(lib, allowed) {
					found = true
					break
				}
			}
			if !found {
				unallowedLibs = append(unallowedLibs, lib)
			}
		}

		if len(unallowedLibs) > 0 {
			result.Passed = false
			result.Message = "Binary has non-standard dynamic library dependencies"
			result.Details = fmt.Sprintf("Unexpected libraries: %v", unallowedLibs)
			return result
		}
	}

	result.Message = "Binary is statically linked (or uses only standard system libraries)"
	return result
}

// checkStaticDarwin checks if macOS binary is statically linked
func (v *IndependenceVerifier) checkStaticDarwin() VerificationResult {
	result := VerificationResult{
		Name:   CheckStaticBinary,
		Passed: true,
	}

	file, err := macho.Open(v.BinaryPath)
	if err != nil {
		result.Passed = false
		result.Message = "Failed to open Mach-O binary"
		result.Details = err.Error()
		return result
	}
	defer file.Close()

	// Check for dynamic libraries
	if len(file.Loads) > 0 {
		libs, err := file.ImportedLibraries()
		if err != nil {
			result.Passed = false
			result.Message = "Failed to read imported libraries"
			result.Details = err.Error()
			return result
		}

		allowedLibs := map[string]bool{
			"libSystem.B.dylib": true,
			"libc++.1.dylib":    true,
			"libresolv.9.dylib": true,
		}

		unallowedLibs := make([]string, 0)
		for _, lib := range libs {
			base := filepath.Base(lib)
			if !allowedLibs[base] {
				unallowedLibs = append(unallowedLibs, lib)
			}
		}

		if len(unallowedLibs) > 0 {
			result.Passed = false
			result.Message = "Binary has non-standard dynamic library dependencies"
			result.Details = fmt.Sprintf("Unexpected libraries: %v", unallowedLibs)
			return result
		}
	}

	result.Message = "Binary uses only standard system libraries"
	return result
}

// checkStaticWindows checks if Windows binary is statically linked
func (v *IndependenceVerifier) checkStaticWindows() VerificationResult {
	result := VerificationResult{
		Name:   CheckStaticBinary,
		Passed: true,
	}

	file, err := pe.Open(v.BinaryPath)
	if err != nil {
		result.Passed = false
		result.Message = "Failed to open PE binary"
		result.Details = err.Error()
		return result
	}
	defer file.Close()

	// Check imports
	if file.OptionalHeader != nil {
		// PE files typically have some imports, check for non-system ones
		imports, err := file.ImportedLibraries()
		if err != nil {
			result.Passed = false
			result.Message = "Failed to read imported libraries"
			result.Details = err.Error()
			return result
		}

		allowedImports := map[string]bool{
			"KERNEL32.dll": true,
			"USER32.dll":   true,
			"GDI32.dll":    true,
			"ADVAPI32.dll": true,
			"SHELL32.dll":  true,
			"WS2_32.dll":   true,
			"MSVCRT.dll":   true,
		}

		unallowedImports := make([]string, 0)
		for _, imp := range imports {
			upper := strings.ToUpper(imp)
			if !allowedImports[upper] {
				unallowedImports = append(unallowedImports, imp)
			}
		}

		if len(unallowedImports) > 0 {
			result.Passed = false
			result.Message = "Binary has non-standard DLL dependencies"
			result.Details = fmt.Sprintf("Unexpected imports: %v", unallowedImports)
			return result
		}
	}

	result.Message = "Binary uses only standard system libraries"
	return result
}

// verifyNoCLIImports checks for imports from cli/src/ directory
func (v *IndependenceVerifier) verifyNoCLIImports() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoCLIImports,
		Passed: true,
	}

	cliSrcPath := filepath.Join(v.ProjectRoot, "..", "cli", "src")
	if _, err := os.Stat(cliSrcPath); os.IsNotExist(err) {
		result.Message = "cli/src directory does not exist, skipping import check"
		return result
	}

	// Check all Go files in the project
	goFiles := make([]string, 0)
	err := filepath.Walk(v.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})

	if err != nil {
		result.Passed = false
		result.Message = "Failed to walk project directory"
		result.Details = err.Error()
		return result
	}

	// Parse each Go file for imports
	cliImports := make(map[string][]string) // file -> imports
	for _, file := range goFiles {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			// Check for imports from cli/src or similar paths
			if strings.Contains(importPath, "/cli/src") ||
				strings.HasPrefix(importPath, "cli/src") ||
				strings.Contains(importPath, "github.com/cline/cline/cli/src") {
				relPath, _ := filepath.Rel(v.ProjectRoot, file)
				cliImports[relPath] = append(cliImports[relPath], importPath)
			}
		}
	}

	if len(cliImports) > 0 {
		result.Passed = false
		result.Message = "Found imports from cli/src directory"
		details := make([]string, 0)
		for file, imports := range cliImports {
			details = append(details, fmt.Sprintf("%s: %v", file, imports))
		}
		result.Details = strings.Join(details, "\n")
	} else {
		result.Message = "No imports from cli/src directory found"
	}

	return result
}

// verifyNoNPMDependencies checks for npm/yarn dependencies
func (v *IndependenceVerifier) verifyNoNPMDependencies() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoNPMDeps,
		Passed: true,
	}

	// Check for package-lock.json
	packageLockPath := filepath.Join(v.ProjectRoot, "package-lock.json")
	if _, err := os.Stat(packageLockPath); err == nil {
		result.Passed = false
		result.Message = "package-lock.json found in Go CLI project"
		result.Details = fmt.Sprintf("package-lock.json exists at: %s", packageLockPath)
		return result
	}

	// Check for yarn.lock
	yarnLockPath := filepath.Join(v.ProjectRoot, "yarn.lock")
	if _, err := os.Stat(yarnLockPath); err == nil {
		result.Passed = false
		result.Message = "yarn.lock found in Go CLI project"
		result.Details = fmt.Sprintf("yarn.lock exists at: %s", yarnLockPath)
		return result
	}

	// Check for pnpm-lock.yaml
	pnpmLockPath := filepath.Join(v.ProjectRoot, "pnpm-lock.yaml")
	if _, err := os.Stat(pnpmLockPath); err == nil {
		result.Passed = false
		result.Message = "pnpm-lock.yaml found in Go CLI project"
		result.Details = fmt.Sprintf("pnpm-lock.yaml exists at: %s", pnpmLockPath)
		return result
	}

	// Check for npm-shrinkwrap.json
	shrinkwrapPath := filepath.Join(v.ProjectRoot, "npm-shrinkwrap.json")
	if _, err := os.Stat(shrinkwrapPath); err == nil {
		result.Passed = false
		result.Message = "npm-shrinkwrap.json found in Go CLI project"
		result.Details = fmt.Sprintf("npm-shrinkwrap.json exists at: %s", shrinkwrapPath)
		return result
	}

	// Check .npmrc
	npmrcPath := filepath.Join(v.ProjectRoot, ".npmrc")
	if _, err := os.Stat(npmrcPath); err == nil {
		result.Passed = false
		result.Message = ".npmrc found in Go CLI project"
		result.Details = fmt.Sprintf(".npmrc exists at: %s", npmrcPath)
		return result
	}

	result.Message = "No npm dependency files found"
	return result
}

// verifyNoTypeScriptFiles checks for TypeScript files in the project
func (v *IndependenceVerifier) verifyNoTypeScriptFiles() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoTypeScript,
		Passed: true,
	}

	tsFiles := make([]string, 0)
	err := filepath.Walk(v.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common directories that might have test fixtures
			if info.Name() == "testdata" || info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".ts" || ext == ".tsx" || ext == ".mts" || ext == ".cts" {
			relPath, _ := filepath.Rel(v.ProjectRoot, path)
			tsFiles = append(tsFiles, relPath)
		}
		return nil
	})

	if err != nil {
		result.Passed = false
		result.Message = "Failed to scan for TypeScript files"
		result.Details = err.Error()
		return result
	}

	if len(tsFiles) > 0 {
		result.Passed = false
		result.Message = "TypeScript files found in Go CLI project"
		result.Details = fmt.Sprintf("Found %d TypeScript files:\n%s", len(tsFiles), strings.Join(tsFiles, "\n"))
	} else {
		result.Message = "No TypeScript files found"
	}

	return result
}

// verifyNoPackageJSON checks for package.json files
func (v *IndependenceVerifier) verifyNoPackageJSON() VerificationResult {
	result := VerificationResult{
		Name:   CheckNoPackageJSON,
		Passed: true,
	}

	packageFiles := make([]string, 0)
	err := filepath.Walk(v.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common directories
			if info.Name() == "testdata" || info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(info.Name()) == "package.json" {
			relPath, _ := filepath.Rel(v.ProjectRoot, path)
			packageFiles = append(packageFiles, relPath)
		}
		return nil
	})

	if err != nil {
		result.Passed = false
		result.Message = "Failed to scan for package.json files"
		result.Details = err.Error()
		return result
	}

	if len(packageFiles) > 0 {
		result.Passed = false
		result.Message = "package.json files found in Go CLI project"
		result.Details = fmt.Sprintf("Found %d package.json files:\n%s", len(packageFiles), strings.Join(packageFiles, "\n"))
	} else {
		result.Message = "No package.json files found"
	}

	return result
}

// verifyGoModIntegrity checks go.mod for any suspicious dependencies
func (v *IndependenceVerifier) verifyGoModIntegrity() VerificationResult {
	result := VerificationResult{
		Name:   CheckGoModIntegrity,
		Passed: true,
	}

	goModPath := filepath.Join(v.ProjectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		result.Passed = false
		result.Message = "Failed to read go.mod"
		result.Details = err.Error()
		return result
	}

	contentStr := string(content)

	// Check for JavaScript/TypeScript related Go packages that might indicate JS embedding
	suspiciousPatterns := []string{
		"github.com/robertkrimen/otto",        // JavaScript interpreter
		"github.com/dop251/goja",              // JavaScript engine
		"github.com/traefik/yaegi",            // Go interpreter (could be used for JS)
		"esbuild",                             // JS bundler
		"webpack",                             // JS bundler
		"typescript",                          // TypeScript compiler
		"babel",                               // JS transpiler
	}

	found := make([]string, 0)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(contentStr, pattern) {
			found = append(found, pattern)
		}
	}

	if len(found) > 0 {
		result.Passed = false
		result.Message = "Suspicious JavaScript-related dependencies found in go.mod"
		result.Details = fmt.Sprintf("Found: %v", found)
	} else {
		result.Message = "go.mod integrity verified"
	}

	return result
}

// RunCommand executes the verification as a CLI command
func RunCommand(binaryPath string, verbose bool, jsonOutput bool) int {
	// Determine project root (parent of current directory, or current directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		return 1
	}

	// If we're in tests directory, go up one level
	if strings.HasSuffix(projectRoot, "/tests") || strings.HasSuffix(projectRoot, "\\tests") {
		projectRoot = filepath.Dir(projectRoot)
	}

	verifier := NewIndependenceVerifier(projectRoot, binaryPath, verbose)
	report, err := verifier.Verify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification error: %v\n", err)
		return 1
	}

	if jsonOutput {
		output, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(output))
	} else {
		printTextReport(report)
	}

	return report.ExitCode
}

// printTextReport prints a human-readable report
func printTextReport(report *VerificationReport) {
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("  Independence Verification Report")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()

	for _, result := range report.Results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		}
		fmt.Printf("[%s] %s\n", status, result.Name)
		fmt.Printf("       %s\n", result.Message)
		if result.Details != "" {
			fmt.Printf("       Details: %s\n", result.Details)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 62))
	fmt.Printf("Summary: %s\n", report.Summary)
	fmt.Printf("Exit Code: %d\n", report.ExitCode)
}

// GetBinaryPath attempts to find the built binary
func GetBinaryPath(projectRoot string) string {
	binaryName := "cline"
	if runtime.GOOS == "windows" {
		binaryName = "cline.exe"
	}

	// Common locations to check
	locations := []string{
		filepath.Join(projectRoot, binaryName),
		filepath.Join(projectRoot, "cmd", "cline", binaryName),
		filepath.Join(projectRoot, "bin", binaryName),
		filepath.Join(projectRoot, "dist", binaryName),
		filepath.Join(projectRoot, "build", binaryName),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Try to find with 'which' or 'where'
	if runtime.GOOS == "windows" {
		out, err := exec.Command("where", binaryName).Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	} else {
		out, err := exec.Command("which", binaryName).Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	return ""
}