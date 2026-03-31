// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AttachmentType represents the type of attachment
type AttachmentType string

const (
	// AttachmentTypeImage represents an image attachment
	AttachmentTypeImage AttachmentType = "image"
	// AttachmentTypeFile represents a generic file attachment
	AttachmentTypeFile AttachmentType = "file"
)

// Attachment represents a file attachment to a task
type Attachment struct {
	// Type is the attachment type
	Type AttachmentType `json:"type"`

	// Path is the original file path
	Path string `json:"path"`

	// Content is the base64-encoded content
	Content string `json:"content"`

	// MimeType is the detected MIME type
	MimeType string `json:"mimeType"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// Name is the file name
	Name string `json:"name"`
}

// AttachmentManager handles loading and validation of attachments
type AttachmentManager struct {
	// MaxSize is the maximum allowed file size in bytes (0 = no limit)
	MaxSize int64

	// AllowedTypes maps file extensions to MIME types
	AllowedTypes map[string]string
}

// NewAttachmentManager creates a new attachment manager with default settings
func NewAttachmentManager() *AttachmentManager {
	return &AttachmentManager{
		MaxSize: 10 * 1024 * 1024, // 10MB default limit
		AllowedTypes: map[string]string{
			// Image formats
			".png":  "image/png",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".gif":  "image/gif",
			".webp": "image/webp",
			".bmp":  "image/bmp",
			".svg":  "image/svg+xml",
			".tiff": "image/tiff",
			".tif":  "image/tiff",
			// Document formats
			".pdf":  "application/pdf",
			".txt":  "text/plain",
			".md":   "text/markdown",
			".json": "application/json",
			".xml":  "application/xml",
			".yaml": "application/yaml",
			".yml":  "application/yaml",
			// Code files
			".go":   "text/x-go",
			".js":   "text/javascript",
			".ts":   "text/typescript",
			".py":   "text/x-python",
			".java": "text/x-java",
			".c":    "text/x-c",
			".cpp":  "text/x-c++",
			".h":    "text/x-c-header",
			".rs":   "text/x-rust",
			".rb":   "text/x-ruby",
			".php":  "text/x-php",
			".sh":   "text/x-shellscript",
		},
	}
}

// LoadImage loads and validates an image file for attachment
func (am *AttachmentManager) LoadImage(path string) (*Attachment, error) {
	// Validate image file
	if err := am.validateImageFile(path); err != nil {
		return nil, err
	}

	// Load the file
	return am.loadFile(path, AttachmentTypeImage)
}

// LoadAttachment loads a generic file attachment
func (am *AttachmentManager) LoadAttachment(path string) (*Attachment, error) {
	// Check if file exists and is accessible
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access file %s: %w", path, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Check size limit
	if am.MaxSize > 0 && info.Size() > am.MaxSize {
		return nil, fmt.Errorf("file too large: %s (%d bytes, max %d bytes)", path, info.Size(), am.MaxSize)
	}

	// Load the file
	return am.loadFile(path, AttachmentTypeFile)
}

// validateImageFile validates that a file is a valid image
func (am *AttachmentManager) validateImageFile(path string) error {
	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", path)
		}
		return fmt.Errorf("cannot access image file %s: %w", path, err)
	}

	// Check it's a file not directory
	if info.IsDir() {
		return fmt.Errorf("image path is a directory: %s", path)
	}

	// Check size limit
	if am.MaxSize > 0 && info.Size() > am.MaxSize {
		return fmt.Errorf("image file too large: %s (%d bytes, max %d bytes)", path, info.Size(), am.MaxSize)
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(path))
	validImageExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
		".svg":  true,
		".tiff": true,
		".tif":  true,
	}

	if !validImageExts[ext] {
		return fmt.Errorf("unsupported image format: %s (supported: png, jpg, jpeg, gif, webp, bmp, svg, tiff)", ext)
	}

	return nil
}

// loadFile loads a file and creates an Attachment
func (am *AttachmentManager) loadFile(path string, attachType AttachmentType) (*Attachment, error) {
	// Expand and clean path
	absPath, err := am.expandPath(path)
	if err != nil {
		return nil, err
	}

	// Get file info
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(absPath))
	mimeType := am.AllowedTypes[ext]
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(content)

	return &Attachment{
		Type:     attachType,
		Path:     absPath,
		Content:  encoded,
		MimeType: mimeType,
		Size:     info.Size(),
		Name:     filepath.Base(absPath),
	}, nil
}

// expandPath expands a path, handling ~ and converting to absolute
func (am *AttachmentManager) expandPath(path string) (string, error) {
	// Expand home directory
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	return absPath, nil
}

// ToDataURL converts the attachment to a data URL format
func (a *Attachment) ToDataURL() string {
	return fmt.Sprintf("data:%s;base64,%s", a.MimeType, a.Content)
}

// ToMarkdown creates a markdown representation of the attachment
func (a *Attachment) ToMarkdown() string {
	switch a.Type {
	case AttachmentTypeImage:
		return fmt.Sprintf("![%s](%s)", a.Name, a.ToDataURL())
	case AttachmentTypeFile:
		return fmt.Sprintf("[%s](%s)", a.Name, a.ToDataURL())
	default:
		return a.ToDataURL()
	}
}

// GetSizeHuman returns a human-readable size string
func (a *Attachment) GetSizeHuman() string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case a.Size >= GB:
		return fmt.Sprintf("%.2f GB", float64(a.Size)/GB)
	case a.Size >= MB:
		return fmt.Sprintf("%.2f MB", float64(a.Size)/MB)
	case a.Size >= KB:
		return fmt.Sprintf("%.2f KB", float64(a.Size)/KB)
	default:
		return fmt.Sprintf("%d bytes", a.Size)
	}
}

// LoadImagesFromPaths loads multiple images from paths
func (am *AttachmentManager) LoadImagesFromPaths(paths []string) ([]*Attachment, error) {
	attachments := make([]*Attachment, 0, len(paths))

	for _, path := range paths {
		if path == "" {
			continue
		}

		attachment, err := am.LoadImage(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", path, err)
		}

		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// AttachmentsToDataURLs converts attachments to data URL strings
func AttachmentsToDataURLs(attachments []*Attachment) []string {
	urls := make([]string, 0, len(attachments))
	for _, a := range attachments {
		urls = append(urls, a.ToDataURL())
	}
	return urls
}

// ValidateImagePaths validates a list of image paths without loading them
func (am *AttachmentManager) ValidateImagePaths(paths []string) error {
	for _, path := range paths {
		if path == "" {
			continue
		}

		if err := am.validateImageFile(path); err != nil {
			return err
		}
	}

	return nil
}

// CopyAttachment creates a copy of an attachment with new content
func (a *Attachment) CopyAttachment() *Attachment {
	return &Attachment{
		Type:     a.Type,
		Path:     a.Path,
		Content:  a.Content,
		MimeType: a.MimeType,
		Size:     a.Size,
		Name:     a.Name,
	}
}

// WriteToFile writes the attachment content to a file
func (a *Attachment) WriteToFile(path string) error {
	// Decode base64 content
	data, err := base64.StdEncoding.DecodeString(a.Content)
	if err != nil {
		return fmt.Errorf("failed to decode attachment content: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write attachment to file: %w", err)
	}

	return nil
}

// Reader returns an io.Reader for the attachment content
func (a *Attachment) Reader() (io.Reader, error) {
	data, err := base64.StdEncoding.DecodeString(a.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode attachment content: %w", err)
	}

	return strings.NewReader(string(data)), nil
}

// IsImage returns true if the attachment is an image
func (a *Attachment) IsImage() bool {
	return a.Type == AttachmentTypeImage || strings.HasPrefix(a.MimeType, "image/")
}

// IsText returns true if the attachment is a text file
func (a *Attachment) IsText() bool {
	return strings.HasPrefix(a.MimeType, "text/") ||
		a.MimeType == "application/json" ||
		a.MimeType == "application/xml" ||
		a.MimeType == "application/yaml"
}