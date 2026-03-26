package scripts

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultScoopManifest(t *testing.T) {
	version := "1.0.0"
	manifest := DefaultScoopManifest(version)

	if manifest.Version != version {
		t.Errorf("Expected version '%s', got '%s'", version, manifest.Version)
	}

	if manifest.Description != "AI-powered coding assistant CLI" {
		t.Errorf("Expected description 'AI-powered coding assistant CLI', got '%s'", manifest.Description)
	}

	if manifest.Homepage != "https://github.com/cline/cline" {
		t.Errorf("Expected homepage 'https://github.com/cline/cline', got '%s'", manifest.Homepage)
	}

	if manifest.License != "Apache-2.0" {
		t.Errorf("Expected license 'Apache-2.0', got '%s'", manifest.License)
	}

	if manifest.Bin != "cline.exe" {
		t.Errorf("Expected bin 'cline.exe', got '%s'", manifest.Bin)
	}

	if manifest.Architecture.AMD64 != nil {
		t.Error("Expected AMD64 to be nil initially")
	}

	if manifest.Architecture.ARM64 != nil {
		t.Error("Expected ARM64 to be nil initially")
	}

	if manifest.Checkver == nil {
		t.Error("Expected Checkver to be set")
	} else if manifest.Checkver.Github != "https://github.com/cline/cline" {
		t.Errorf("Expected Checkver.Github 'https://github.com/cline/cline', got '%s'", manifest.Checkver.Github)
	}

	if manifest.Autoupdate == nil {
		t.Error("Expected Autoupdate to be set")
	}
}

func TestScoopManifestAddPlatform(t *testing.T) {
	// Create test content and calculate expected SHA256
	testContent := []byte("test binary content for sha256 calculation")
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	manifest := DefaultScoopManifest("1.0.0")
	url := server.URL + "/test.zip"

	err := manifest.AddPlatform("amd64", url)
	if err != nil {
		t.Fatalf("AddPlatform failed: %v", err)
	}

	if manifest.Architecture.AMD64 == nil {
		t.Fatal("Expected AMD64 platform to be set")
	}

	if manifest.Architecture.AMD64.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, manifest.Architecture.AMD64.URL)
	}

	expectedHash := "sha256:" + expectedSHA256
	if manifest.Architecture.AMD64.Hash != expectedHash {
		t.Errorf("Expected Hash '%s', got '%s'", expectedHash, manifest.Architecture.AMD64.Hash)
	}

	// Test unsupported architecture
	err = manifest.AddPlatform("unsupported", url)
	if err == nil {
		t.Error("Expected error for unsupported architecture")
	}
}

func TestScoopManifestAddPlatformWithLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cline_1.0.0_windows_amd64.zip")
	testContent := []byte("test binary content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	manifest := DefaultScoopManifest("1.0.0")
	url := "https://example.com/cline_1.0.0_windows_amd64.zip"

	err := manifest.AddPlatformWithLocalFile("amd64", url, testFile)
	if err != nil {
		t.Fatalf("AddPlatformWithLocalFile failed: %v", err)
	}

	if manifest.Architecture.AMD64 == nil {
		t.Fatal("Expected AMD64 platform to be set")
	}

	expectedHash := "sha256:" + expectedSHA256
	if manifest.Architecture.AMD64.Hash != expectedHash {
		t.Errorf("Expected Hash '%s', got '%s'", expectedHash, manifest.Architecture.AMD64.Hash)
	}
}

func TestScoopManifestValidate(t *testing.T) {
	tests := []struct {
		name      string
		manifest  *ScoopManifest
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty manifest",
			manifest:  &ScoopManifest{},
			wantError: true,
			errorMsg:  "manifest version is required",
		},
		{
			name: "missing description",
			manifest: &ScoopManifest{
				Version: "1.0.0",
			},
			wantError: true,
			errorMsg:  "manifest description is required",
		},
		{
			name: "missing homepage",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
			},
			wantError: true,
			errorMsg:  "manifest homepage is required",
		},
		{
			name: "missing bin",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
			},
			wantError: true,
			errorMsg:  "manifest bin is required",
		},
		{
			name: "no platforms",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
				Bin:         "cline.exe",
			},
			wantError: true,
			errorMsg:  "at least one platform is required",
		},
		{
			name: "missing platform URL",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
				Bin:         "cline.exe",
				Architecture: ScoopArchitectureManifest{
					AMD64: &ScoopPlatform{
						Hash: "sha256:" + strings.Repeat("a", 64),
					},
				},
			},
			wantError: true,
			errorMsg:  "amd64: URL is required",
		},
		{
			name: "missing platform hash",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
				Bin:         "cline.exe",
				Architecture: ScoopArchitectureManifest{
					AMD64: &ScoopPlatform{
						URL: "https://example.com",
					},
				},
			},
			wantError: true,
			errorMsg:  "amd64: hash is required",
		},
		{
			name: "invalid hash format",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
				Bin:         "cline.exe",
				Architecture: ScoopArchitectureManifest{
					AMD64: &ScoopPlatform{
						URL:  "https://example.com",
						Hash: "invalid-hash",
					},
				},
			},
			wantError: true,
			errorMsg:  "amd64: invalid hash format",
		},
		{
			name: "valid manifest",
			manifest: &ScoopManifest{
				Version:     "1.0.0",
				Description: "Test description",
				Homepage:    "https://example.com",
				Bin:         "cline.exe",
				Architecture: ScoopArchitectureManifest{
					AMD64: &ScoopPlatform{
						URL:  "https://example.com",
						Hash: "sha256:" + strings.Repeat("a", 64),
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestScoopManifestGenerate(t *testing.T) {
	manifest := &ScoopManifest{
		Version:     "1.0.0",
		Description: "AI-powered coding assistant CLI",
		Homepage:    "https://github.com/cline/cline",
		License:     "Apache-2.0",
		Bin:         "cline.exe",
		Architecture: ScoopArchitectureManifest{
			AMD64: &ScoopPlatform{
				URL:  "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_windows_amd64.zip",
				Hash: "sha256:" + strings.Repeat("a", 64),
			},
		},
		Checkver: &ScoopCheckver{
			Github: "https://github.com/cline/cline",
		},
	}

	content, err := manifest.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check that the content contains expected elements
	expectedElements := []string{
		`"version": "1.0.0"`,
		`"description": "AI-powered coding assistant CLI"`,
		`"homepage": "https://github.com/cline/cline"`,
		`"license": "Apache-2.0"`,
		`"bin": "cline.exe"`,
		`"64bit"`,
		`"url"`,
		`"hash"`,
		`"checkver"`,
		`"github"`,
	}

	for _, elem := range expectedElements {
		if !strings.Contains(content, elem) {
			t.Errorf("Generated manifest missing expected element: %s", elem)
		}
	}
}

func TestScoopManifestGenerateToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "bucket", "cline.json")

	manifest := &ScoopManifest{
		Version:     "1.0.0",
		Description: "Test description",
		Homepage:    "https://example.com",
		Bin:         "cline.exe",
		Architecture: ScoopArchitectureManifest{
			AMD64: &ScoopPlatform{
				URL:  "https://example.com",
				Hash: "sha256:" + strings.Repeat("a", 64),
			},
		},
	}

	err := manifest.GenerateToFile(outputPath)
	if err != nil {
		t.Fatalf("GenerateToFile() failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created at %s", outputPath)
	}

	// Check content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), `"version": "1.0.0"`) {
		t.Error("Generated file does not contain expected version")
	}
}

func TestCalculateScoopSHA256FromURL(t *testing.T) {
	// Create test content
	testContent := []byte("test content for sha256")
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create a test HTTP server with 404 handling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nonexistent" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	result, err := CalculateScoopSHA256FromURL(server.URL + "/test.zip")
	if err != nil {
		t.Fatalf("CalculateScoopSHA256FromURL() failed: %v", err)
	}

	if result != expectedSHA256 {
		t.Errorf("Expected SHA256 '%s', got '%s'", expectedSHA256, result)
	}

	// Test 404 response
	_, err = CalculateScoopSHA256FromURL(server.URL + "/nonexistent")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestDefaultScoopBucket(t *testing.T) {
	bucket := DefaultScoopBucket()

	if bucket.Name != "cline-scoop" {
		t.Errorf("Expected bucket name 'cline-scoop', got '%s'", bucket.Name)
	}

	if bucket.ManifestDir != "bucket" {
		t.Errorf("Expected manifest dir 'bucket', got '%s'", bucket.ManifestDir)
	}
}

func TestScoopBucketCreateBucketStructure(t *testing.T) {
	tmpDir := t.TempDir()
	bucket := DefaultScoopBucket()

	err := bucket.CreateBucketStructure(tmpDir)
	if err != nil {
		t.Fatalf("CreateBucketStructure() failed: %v", err)
	}

	// Check bucket directory was created
	bucketDir := filepath.Join(tmpDir, "bucket")
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		t.Errorf("Bucket directory was not created")
	}

	// Check README was created
	readmePath := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Errorf("README.md was not created")
	}

	// Check README content
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if !strings.Contains(string(readmeContent), "Scoop Bucket for Cline") {
		t.Error("README does not contain expected title")
	}
}

func TestScoopBucketGetManifestPath(t *testing.T) {
	bucket := DefaultScoopBucket()
	rootPath := "/tmp/bucket"
	manifestName := "cline"

	expected := filepath.Join(rootPath, "bucket", "cline.json")
	result := bucket.GetManifestPath(rootPath, manifestName)

	if result != expected {
		t.Errorf("GetManifestPath() = %s, want %s", result, expected)
	}
}

func TestGenerateScoopReleaseURL(t *testing.T) {
	baseURL := "https://github.com/cline/cline/releases/download"
	version := "1.0.0"
	binaryName := "cline"
	arch := "amd64"

	expected := "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_windows_amd64.zip"
	result := GenerateScoopReleaseURL(baseURL, version, binaryName, arch)

	if result != expected {
		t.Errorf("GenerateScoopReleaseURL() = %s, want %s", result, expected)
	}
}

func TestScoopSupportedArchitectures(t *testing.T) {
	archs := ScoopSupportedArchitectures()

	expected := []string{"amd64", "arm64"}

	if len(archs) != len(expected) {
		t.Errorf("Expected %d architectures, got %d", len(expected), len(archs))
	}

	for i, arch := range archs {
		if arch != expected[i] {
			t.Errorf("Architecture %d: expected '%s', got '%s'", i, expected[i], arch)
		}
	}
}

func TestIsScoopSupportedArchitecture(t *testing.T) {
	tests := []struct {
		arch     string
		expected bool
	}{
		{"amd64", true},
		{"arm64", true},
		{"386", false},
		{"arm", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			result := IsScoopSupportedArchitecture(tt.arch)
			if result != tt.expected {
				t.Errorf("IsScoopSupportedArchitecture(%s) = %v, want %v", tt.arch, result, tt.expected)
			}
		})
	}
}

func TestIsValidSHA256Hash(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{
			name:     "valid hash",
			hash:     "sha256:" + strings.Repeat("a", 64),
			expected: true,
		},
		{
			name:     "missing prefix",
			hash:     strings.Repeat("a", 64),
			expected: false,
		},
		{
			name:     "too short",
			hash:     "sha256:" + strings.Repeat("a", 63),
			expected: false,
		},
		{
			name:     "too long",
			hash:     "sha256:" + strings.Repeat("a", 65),
			expected: false,
		},
		{
			name:     "empty",
			hash:     "",
			expected: false,
		},
		{
			name:     "invalid characters",
			hash:     "sha256:" + strings.Repeat("g", 64),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSHA256Hash(tt.hash)
			if result != tt.expected {
				t.Errorf("isValidSHA256Hash(%q) = %v, want %v", tt.hash, result, tt.expected)
			}
		})
	}
}

func TestScoopManifestGenerateInvalid(t *testing.T) {
	manifest := &ScoopManifest{
		// Missing required fields
		Version: "",
	}

	_, err := manifest.Generate()
	if err == nil {
		t.Error("Expected error for invalid manifest, got nil")
	}
}

func TestScoopManifestMultiplePlatforms(t *testing.T) {
	tmpDir := t.TempDir()
	testFileAMD64 := filepath.Join(tmpDir, "cline_1.0.0_windows_amd64.zip")
	testFileARM64 := filepath.Join(tmpDir, "cline_1.0.0_windows_arm64.zip")

	if err := os.WriteFile(testFileAMD64, []byte("amd64 content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFileARM64, []byte("arm64 content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manifest := DefaultScoopManifest("1.0.0")

	if err := manifest.AddPlatformWithLocalFile("amd64", "https://example.com/amd64.zip", testFileAMD64); err != nil {
		t.Fatalf("Failed to add amd64 platform: %v", err)
	}

	if err := manifest.AddPlatformWithLocalFile("arm64", "https://example.com/arm64.zip", testFileARM64); err != nil {
		t.Fatalf("Failed to add arm64 platform: %v", err)
	}

	if manifest.Architecture.AMD64 == nil {
		t.Error("Expected AMD64 platform to be set")
	}

	if manifest.Architecture.ARM64 == nil {
		t.Error("Expected ARM64 platform to be set")
	}

	content, err := manifest.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if !strings.Contains(content, `"64bit"`) {
		t.Error("Generated manifest missing 64bit section")
	}

	if !strings.Contains(content, `"arm64"`) {
		t.Error("Generated manifest missing arm64 section")
	}
}