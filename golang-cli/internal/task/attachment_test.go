package task

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAttachmentManager(t *testing.T) {
	am := NewAttachmentManager()
	
	if am == nil {
		t.Fatal("Expected AttachmentManager to be created")
	}
	
	if am.MaxSize != 10*1024*1024 {
		t.Errorf("Expected MaxSize to be 10MB, got %d", am.MaxSize)
	}
	
	if am.AllowedTypes == nil {
		t.Fatal("Expected AllowedTypes to be initialized")
	}
	
	// Check some allowed types
	if am.AllowedTypes[".png"] != "image/png" {
		t.Errorf("Expected .png to map to image/png, got %s", am.AllowedTypes[".png"])
	}
	
	if am.AllowedTypes[".go"] != "text/x-go" {
		t.Errorf("Expected .go to map to text/x-go, got %s", am.AllowedTypes[".go"])
	}
}

func TestAttachmentManagerLoadImage(t *testing.T) {
	am := NewAttachmentManager()
	
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")
	
	// Create a simple PNG-like file (just magic bytes)
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(testFile, pngMagic, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("valid image file", func(t *testing.T) {
		attachment, err := am.LoadImage(testFile)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if attachment == nil {
			t.Fatal("Expected attachment to be returned")
		}
		
		if attachment.Type != AttachmentTypeImage {
			t.Errorf("Expected type to be image, got %s", attachment.Type)
		}
		
		if attachment.MimeType != "image/png" {
			t.Errorf("Expected MIME type to be image/png, got %s", attachment.MimeType)
		}
		
		if attachment.Name != "test.png" {
			t.Errorf("Expected name to be test.png, got %s", attachment.Name)
		}
	})
	
	t.Run("non-existent file", func(t *testing.T) {
		_, err := am.LoadImage("/non/existent/file.png")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
	
	t.Run("directory instead of file", func(t *testing.T) {
		_, err := am.LoadImage(tmpDir)
		if err == nil {
			t.Error("Expected error for directory")
		}
		
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("Expected 'directory' error, got: %v", err)
		}
	})
	
	t.Run("unsupported image format", func(t *testing.T) {
		unsupportedFile := filepath.Join(tmpDir, "test.xyz")
		if err := os.WriteFile(unsupportedFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		
		_, err := am.LoadImage(unsupportedFile)
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
		
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Expected 'unsupported' error, got: %v", err)
		}
	})
	
	t.Run("file too large", func(t *testing.T) {
		// Set a small max size
		am.MaxSize = 1
		
		_, err := am.LoadImage(testFile)
		if err == nil {
			t.Error("Expected error for file too large")
		}
		
		if !strings.Contains(err.Error(), "too large") {
			t.Errorf("Expected 'too large' error, got: %v", err)
		}
	})
}

func TestLoadAttachment(t *testing.T) {
	am := NewAttachmentManager()
	
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	
	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("valid file", func(t *testing.T) {
		attachment, err := am.LoadAttachment(testFile)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if attachment == nil {
			t.Fatal("Expected attachment to be returned")
		}
		
		if attachment.Type != AttachmentTypeFile {
			t.Errorf("Expected type to be file, got %s", attachment.Type)
		}
		
		if attachment.MimeType != "text/plain" {
			t.Errorf("Expected MIME type to be text/plain, got %s", attachment.MimeType)
		}
		
		if attachment.Name != "test.txt" {
			t.Errorf("Expected name to be test.txt, got %s", attachment.Name)
		}
	})
	
	t.Run("non-existent file", func(t *testing.T) {
		_, err := am.LoadAttachment("/non/existent/file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
	
	t.Run("directory instead of file", func(t *testing.T) {
		_, err := am.LoadAttachment(tmpDir)
		if err == nil {
			t.Error("Expected error for directory")
		}
		
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("Expected 'directory' error, got: %v", err)
		}
	})
	
	t.Run("file too large", func(t *testing.T) {
		// Set a small max size
		am.MaxSize = 1
		
		_, err := am.LoadAttachment(testFile)
		if err == nil {
			t.Error("Expected error for file too large")
		}
		
		if !strings.Contains(err.Error(), "too large") {
			t.Errorf("Expected 'too large' error, got: %v", err)
		}
	})
}

func TestAttachmentToDataURL(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte("test content"))
	attachment := &Attachment{
		Type:     AttachmentTypeImage,
		MimeType: "image/png",
		Content:  content,
		Name:     "test.png",
	}
	
	expected := fmt.Sprintf("data:image/png;base64,%s", content)
	result := attachment.ToDataURL()
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestAttachmentToMarkdown(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte("test content"))
	attachment := &Attachment{
		Type:     AttachmentTypeImage,
		MimeType: "image/png",
		Content:  content,
		Name:     "test.png",
	}
	
	t.Run("image attachment", func(t *testing.T) {
		markdown := attachment.ToMarkdown()
		if !strings.HasPrefix(markdown, "![test.png](data:image/png;base64,") {
			t.Errorf("Unexpected markdown for image: %s", markdown)
		}
	})
	
	t.Run("file attachment", func(t *testing.T) {
		fileAttachment := &Attachment{
			Type:     AttachmentTypeFile,
			MimeType: "text/plain",
			Content:  content,
			Name:     "test.txt",
		}
		
		markdown := fileAttachment.ToMarkdown()
		if !strings.HasPrefix(markdown, "[test.txt](data:text/plain;base64,") {
			t.Errorf("Unexpected markdown for file: %s", markdown)
		}
	})
	
	t.Run("unknown type", func(t *testing.T) {
		unknownAttachment := &Attachment{
			Type:     AttachmentType("unknown"),
			MimeType: "application/octet-stream",
			Content:  content,
			Name:     "test.bin",
		}
		
		markdown := unknownAttachment.ToMarkdown()
		if !strings.HasPrefix(markdown, "data:application/octet-stream;base64,") {
			t.Errorf("Unexpected markdown for unknown type: %s", markdown)
		}
	})
}

func TestGetSizeHuman(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{100, "100 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1536 * 1024, "1.50 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1536 * 1024 * 1024, "1.50 GB"},
	}
	
	for _, test := range tests {
		attachment := &Attachment{Size: test.size}
		result := attachment.GetSizeHuman()
		
		if result != test.expected {
			t.Errorf("Size %d: expected %s, got %s", test.size, test.expected, result)
		}
	}
}

func TestLoadImagesFromPaths(t *testing.T) {
	am := NewAttachmentManager()
	tmpDir := t.TempDir()
	
	// Create test images
	testFiles := []string{
		filepath.Join(tmpDir, "image1.png"),
		filepath.Join(tmpDir, "image2.png"),
	}
	
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for _, file := range testFiles {
		if err := os.WriteFile(file, pngMagic, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	t.Run("valid images", func(t *testing.T) {
		attachments, err := am.LoadImagesFromPaths(testFiles)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if len(attachments) != 2 {
			t.Errorf("Expected 2 attachments, got %d", len(attachments))
		}
	})
	
	t.Run("empty path skipped", func(t *testing.T) {
		paths := []string{testFiles[0], "", testFiles[1]}
		attachments, err := am.LoadImagesFromPaths(paths)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if len(attachments) != 2 {
			t.Errorf("Expected 2 attachments (empty skipped), got %d", len(attachments))
		}
	})
	
	t.Run("invalid image", func(t *testing.T) {
		invalidPath := []string{"/non/existent.png"}
		_, err := am.LoadImagesFromPaths(invalidPath)
		if err == nil {
			t.Error("Expected error for invalid image path")
		}
	})
}

func TestAttachmentsToDataURLs(t *testing.T) {
	content1 := base64.StdEncoding.EncodeToString([]byte("content1"))
	content2 := base64.StdEncoding.EncodeToString([]byte("content2"))
	
	attachments := []*Attachment{
		{
			Type:     AttachmentTypeImage,
			MimeType: "image/png",
			Content:  content1,
			Name:     "test1.png",
		},
		{
			Type:     AttachmentTypeImage,
			MimeType: "image/jpeg",
			Content:  content2,
			Name:     "test2.jpg",
		},
	}
	
	urls := AttachmentsToDataURLs(attachments)
	
	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}
	
	expected1 := fmt.Sprintf("data:image/png;base64,%s", content1)
	if urls[0] != expected1 {
		t.Errorf("Expected first URL to be %s, got %s", expected1, urls[0])
	}
}

func TestValidateImagePaths(t *testing.T) {
	am := NewAttachmentManager()
	tmpDir := t.TempDir()
	
	// Create test image
	testFile := filepath.Join(tmpDir, "image.png")
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(testFile, pngMagic, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("valid paths", func(t *testing.T) {
		err := am.ValidateImagePaths([]string{testFile})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
	
	t.Run("empty path skipped", func(t *testing.T) {
		err := am.ValidateImagePaths([]string{testFile, ""})
		if err != nil {
			t.Errorf("Expected no error (empty skipped), got: %v", err)
		}
	})
	
	t.Run("invalid path", func(t *testing.T) {
		err := am.ValidateImagePaths([]string{"/non/existent.png"})
		if err == nil {
			t.Error("Expected error for invalid path")
		}
	})
}

func TestCopyAttachment(t *testing.T) {
	original := &Attachment{
		Type:     AttachmentTypeImage,
		Path:     "/path/to/test.png",
		Content:  "base64content",
		MimeType: "image/png",
		Size:     1024,
		Name:     "test.png",
	}
	
	copy := original.CopyAttachment()
	
	if copy == nil {
		t.Fatal("Expected copy to be returned")
	}
	
	// Verify all fields are copied
	if copy.Type != original.Type {
		t.Errorf("Type mismatch: expected %v, got %v", original.Type, copy.Type)
	}
	if copy.Path != original.Path {
		t.Errorf("Path mismatch: expected %v, got %v", original.Path, copy.Path)
	}
	if copy.Content != original.Content {
		t.Errorf("Content mismatch: expected %v, got %v", original.Content, copy.Content)
	}
	if copy.MimeType != original.MimeType {
		t.Errorf("MimeType mismatch: expected %v, got %v", original.MimeType, copy.MimeType)
	}
	if copy.Size != original.Size {
		t.Errorf("Size mismatch: expected %v, got %v", original.Size, copy.Size)
	}
	if copy.Name != original.Name {
		t.Errorf("Name mismatch: expected %v, got %v", original.Name, copy.Name)
	}
	
	// Verify it's a different pointer (deep copy)
	copy.Content = "modified"
	if original.Content == "modified" {
		t.Error("Copy should be independent of original")
	}
}

func TestWriteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	content := "Hello, World!"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
	
	attachment := &Attachment{
		Content: encodedContent,
	}
	
	outputPath := filepath.Join(tmpDir, "output.txt")
	err := attachment.WriteToFile(outputPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify file contents
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	
	if string(data) != content {
		t.Errorf("Expected %q, got %q", content, string(data))
	}
	
	t.Run("invalid base64", func(t *testing.T) {
		invalidAttachment := &Attachment{
			Content: "invalid-base64!!!",
		}
		
		err := invalidAttachment.WriteToFile(filepath.Join(tmpDir, "invalid.txt"))
		if err == nil {
			t.Error("Expected error for invalid base64")
		}
	})
}

func TestReader(t *testing.T) {
	content := "Hello, World!"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
	
	attachment := &Attachment{
		Content: encodedContent,
	}
	
	reader, err := attachment.Reader()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Read all content
	buf := make([]byte, len(content))
	n, err := reader.Read(buf)
	if err != nil {
		t.Errorf("Expected no error reading, got: %v", err)
	}
	
	if n != len(content) {
		t.Errorf("Expected to read %d bytes, got %d", len(content), n)
	}
	
	if string(buf) != content {
		t.Errorf("Expected %q, got %q", content, string(buf))
	}
	
	t.Run("invalid base64", func(t *testing.T) {
		invalidAttachment := &Attachment{
			Content: "invalid-base64!!!",
		}
		
		_, err := invalidAttachment.Reader()
		if err == nil {
			t.Error("Expected error for invalid base64")
		}
	})
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		attachment *Attachment
		expected   bool
	}{
		{
			&Attachment{Type: AttachmentTypeImage, MimeType: "image/png"},
			true,
		},
		{
			&Attachment{Type: AttachmentTypeFile, MimeType: "image/jpeg"},
			true,
		},
		{
			&Attachment{Type: AttachmentTypeFile, MimeType: "text/plain"},
			false,
		},
		{
			&Attachment{Type: AttachmentTypeFile, MimeType: "application/pdf"},
			false,
		},
	}
	
	for i, test := range tests {
		result := test.attachment.IsImage()
		if result != test.expected {
			t.Errorf("Test %d: expected %v, got %v", i, test.expected, result)
		}
	}
}

func TestIsText(t *testing.T) {
	tests := []struct {
		attachment *Attachment
		expected   bool
	}{
		{
			&Attachment{MimeType: "text/plain"},
			true,
		},
		{
			&Attachment{MimeType: "text/x-go"},
			true,
		},
		{
			&Attachment{MimeType: "application/json"},
			true,
		},
		{
			&Attachment{MimeType: "application/xml"},
			true,
		},
		{
			&Attachment{MimeType: "application/yaml"},
			true,
		},
		{
			&Attachment{MimeType: "image/png"},
			false,
		},
		{
			&Attachment{MimeType: "application/pdf"},
			false,
		},
	}
	
	for i, test := range tests {
		result := test.attachment.IsText()
		if result != test.expected {
			t.Errorf("Test %d: expected %v, got %v", i, test.expected, result)
		}
	}
}

func TestExpandPath(t *testing.T) {
	am := NewAttachmentManager()
	
	t.Run("absolute path", func(t *testing.T) {
		absPath := "/tmp/test.txt"
		result, err := am.expandPath(absPath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if !filepath.IsAbs(result) {
			t.Errorf("Expected absolute path, got %s", result)
		}
	})
	
	t.Run("relative path", func(t *testing.T) {
		relPath := "test.txt"
		result, err := am.expandPath(relPath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if !filepath.IsAbs(result) {
			t.Errorf("Expected absolute path, got %s", result)
		}
	})
	
	t.Run("home directory expansion", func(t *testing.T) {
		homePath := "~/test.txt"
		result, err := am.expandPath(homePath)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		homeDir, _ := os.UserHomeDir()
		if !strings.HasPrefix(result, homeDir) {
			t.Errorf("Expected path to start with home directory, got %s", result)
		}
	})
}

func TestLoadFile(t *testing.T) {
	am := NewAttachmentManager()
	tmpDir := t.TempDir()
	
	// Create test file with unknown extension
	testFile := filepath.Join(tmpDir, "test.unknown")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	t.Run("unknown extension", func(t *testing.T) {
		attachment, err := am.loadFile(testFile, AttachmentTypeFile)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		if attachment.MimeType != "application/octet-stream" {
			t.Errorf("Expected default MIME type, got %s", attachment.MimeType)
		}
	})
	
	t.Run("non-existent file", func(t *testing.T) {
		_, err := am.loadFile("/non/existent/file.txt", AttachmentTypeFile)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}