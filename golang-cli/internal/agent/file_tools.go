package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cline/cline/golang-cli/internal/security"
)

type ReadFileTool struct {
	BaseTool
	workingDir string
}

func NewReadFileTool(workingDir string) *ReadFileTool {
	return &ReadFileTool{
		BaseTool: BaseTool{
			Name:        "read_file",
			Description: "Read file contents",
			Usage:       "read_file <path>",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Description: "File path", Required: true},
			},
			Dangerous: false,
		},
		workingDir: workingDir,
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}
	path := GetStringParam(params, "path", "")
	fullPath := t.resolvePath(path)

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is directory: %s", path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	return string(content), nil
}

func (t *ReadFileTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path"})
}

func (t *ReadFileTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}

type WriteFileTool struct {
	BaseTool
	workingDir string
}

func NewWriteFileTool(workingDir string) *WriteFileTool {
	return &WriteFileTool{
		BaseTool: BaseTool{
			Name:        "write_to_file",
			Description: "Write content to file",
			Usage:       "write_to_file <path> <content>",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Description: "File path", Required: true},
				{Name: "content", Type: "string", Description: "File content", Required: true},
			},
			Dangerous: true,
		},
		workingDir: workingDir,
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}
	path := GetStringParam(params, "path", "")
	content := GetStringParam(params, "content", "")
	fullPath := t.resolvePath(path)

	if err := security.ValidateFilePath(fullPath, t.workingDir); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	dir := filepath.Dir(fullPath)
	os.MkdirAll(dir, 0755)

	exists := false
	if _, err := os.Stat(fullPath); err == nil {
		exists = true
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	if exists {
		return fmt.Sprintf("File updated: %s", path), nil
	}
	return fmt.Sprintf("File created: %s", path), nil
}

func (t *WriteFileTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path", "content"})
}

func (t *WriteFileTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}

type ApplyDiffTool struct {
	BaseTool
	workingDir string
}

func NewApplyDiffTool(workingDir string) *ApplyDiffTool {
	return &ApplyDiffTool{
		BaseTool: BaseTool{
			Name:        "apply_diff",
			Description: "Apply diff to file",
			Usage:       "apply_diff <path> <diff>",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Description: "File path", Required: true},
				{Name: "diff", Type: "string", Description: "Diff content", Required: true},
			},
			Dangerous: true,
		},
		workingDir: workingDir,
	}
}

func (t *ApplyDiffTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}
	path := GetStringParam(params, "path", "")
	diff := GetStringParam(params, "diff", "")
	fullPath := t.resolvePath(path)

	if err := security.ValidateFilePath(fullPath, t.workingDir); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	original, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	newContent, err := applyDiff(string(original), diff)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}
	return fmt.Sprintf("Diff applied to: %s", path), nil
}

func (t *ApplyDiffTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path", "diff"})
}

func (t *ApplyDiffTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}

func applyDiff(content, diff string) (string, error) {
	pattern := `(?s)------- SEARCH\s*(.*?)\s*=======\s*(.*?)\s*\+\+\+\+\+\+\+ REPLACE`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(diff, -1)

	if len(matches) == 0 {
		return "", fmt.Errorf("no SEARCH/REPLACE blocks")
	}

	result := content
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		search := strings.TrimRight(match[1], "\n")
		replace := strings.TrimRight(match[2], "\n")

		if !strings.Contains(result, search) {
			return "", fmt.Errorf("search not found")
		}
		result = strings.Replace(result, search, replace, 1)
	}
	return result, nil
}

type SearchFilesTool struct {
	BaseTool
	workingDir string
}

func NewSearchFilesTool(workingDir string) *SearchFilesTool {
	return &SearchFilesTool{
		BaseTool: BaseTool{
			Name:        "search_files",
			Description: "Search files for pattern",
			Usage:       "search_files <path> <regex>",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Description: "Directory path", Required: true},
				{Name: "regex", Type: "string", Description: "Search pattern", Required: true},
				{Name: "file_pattern", Type: "string", Description: "File glob", Required: false, Default: "*"},
			},
			Dangerous: false,
		},
		workingDir: workingDir,
	}
}

func (t *SearchFilesTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}
	path := GetStringParam(params, "path", "")
	regex := GetStringParam(params, "regex", "")
	filePattern := GetStringParam(params, "file_pattern", "*")
	fullPath := t.resolvePath(path)

	re, err := regexp.Compile(regex)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	var results []string
	filepath.Walk(fullPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filePattern != "*" {
			if matched, _ := filepath.Match(filePattern, filepath.Base(filePath)); !matched {
				return nil
			}
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}
		if re.Match(content) {
			rel, _ := filepath.Rel(t.workingDir, filePath)
			results = append(results, rel)
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if re.MatchString(line) {
					results = append(results, fmt.Sprintf("  L%d: %s", i+1, strings.TrimSpace(line)))
				}
			}
		}
		return nil
	})

	if len(results) == 0 {
		return "No matches found", nil
	}
	return strings.Join(results, "\n"), nil
}

func (t *SearchFilesTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path", "regex"})
}

func (t *SearchFilesTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}

type ListFilesTool struct {
	BaseTool
	workingDir string
}

func NewListFilesTool(workingDir string) *ListFilesTool {
	return &ListFilesTool{
		BaseTool: BaseTool{
			Name:        "list_files",
			Description: "List directory contents",
			Usage:       "list_files <path> [recursive]",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Description: "Directory path", Required: true},
				{Name: "recursive", Type: "boolean", Description: "Recursive listing", Required: false, Default: false},
			},
			Dangerous: false,
		},
		workingDir: workingDir,
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}
	path := GetStringParam(params, "path", "")
	recursive := GetBoolParam(params, "recursive", false)
	fullPath := t.resolvePath(path)

	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}

	var results []string
	if recursive {
		t.listRecursive(fullPath, "", &results)
	} else {
		t.listFiles(fullPath, &results)
	}
	return strings.Join(results, "\n"), nil
}

func (t *ListFilesTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path"})
}

func (t *ListFilesTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}

func (t *ListFilesTool) listFiles(dir string, results *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		*results = append(*results, name)
	}
	return nil
}

func (t *ListFilesTool) listRecursive(dir, prefix string, results *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		path := filepath.Join(dir, name)
		if e.IsDir() {
			*results = append(*results, prefix+name+"/")
			t.listRecursive(path, prefix+"  ", results)
		} else {
			*results = append(*results, prefix+name)
		}
	}
	return nil
}