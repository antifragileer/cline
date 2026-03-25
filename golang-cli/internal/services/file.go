// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// FileService implements the FileService gRPC interface
type FileService struct {
	cline.UnimplementedFileServiceServer
	state *storage.ClineFileStorage
}

// NewFileService creates a new FileService instance
func NewFileService(state *storage.ClineFileStorage) *FileService {
	return &FileService{state: state}
}

func (s *FileService) CopyToClipboard(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	// Platform-specific clipboard copy
	return &cline.Empty{}, fmt.Errorf("clipboard not implemented")
}

func (s *FileService) OpenFile(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	cmd := exec.Command("code", req.Value)
	return &cline.Empty{}, cmd.Start()
}

func (s *FileService) OpenImage(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", req.GetValue())
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", req.GetValue())
	default:
		cmd = exec.Command("xdg-open", req.GetValue())
	}
	return &cline.Empty{}, cmd.Start()
}

func (s *FileService) OpenMention(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	mention := req.Value
	switch {
	case strings.HasPrefix(mention, "http://"), strings.HasPrefix(mention, "https://"):
		return s.openURL(mention)
	case strings.HasPrefix(mention, "/"):
		return s.OpenFile(ctx, &cline.StringRequest{Value: mention})
	default:
		return s.OpenFile(ctx, &cline.StringRequest{Value: mention})
	}
}

func (s *FileService) DeleteRuleFile(ctx context.Context, req *cline.RuleFileRequest) (*cline.RuleFile, error) {
	dir := s.getRulesDir(req.IsGlobal)
	path := filepath.Join(dir, req.GetRulePath())
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return &cline.RuleFile{FilePath: path, DisplayName: filepath.Base(path)}, nil
}

func (s *FileService) CreateRuleFile(ctx context.Context, req *cline.RuleFileRequest) (*cline.RuleFile, error) {
	dir := s.getRulesDir(req.IsGlobal)
	path := filepath.Join(dir, req.GetFilename())
	exists := false
	if _, err := os.Stat(path); err == nil {
		exists = true
	}
	if err := os.WriteFile(path, []byte("# Rule\n"), 0644); err != nil {
		return nil, err
	}
	return &cline.RuleFile{FilePath: path, DisplayName: req.GetFilename(), AlreadyExists: exists}, nil
}

func (s *FileService) SearchCommits(ctx context.Context, req *cline.StringRequest) (*cline.GitCommits, error) {
	cmd := exec.Command("git", "log", "--format=%H|%h|%s|%an|%ai", "-n", "20", "--grep", req.Value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var commits []*cline.GitCommit
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			commits = append(commits, &cline.GitCommit{
				Hash:      parts[0],
				ShortHash: parts[1],
				Subject:   parts[2],
				Author:    parts[3],
				Date:      parts[4],
			})
		}
	}
	return &cline.GitCommits{Commits: commits}, nil
}

func (s *FileService) SelectFiles(ctx context.Context, req *cline.BooleanRequest) (*cline.StringArrays, error) {
	return &cline.StringArrays{}, fmt.Errorf("file picker not implemented in CLI")
}

func (s *FileService) GetRelativePaths(ctx context.Context, req *cline.RelativePathsRequest) (*cline.RelativePaths, error) {
	var paths []string
	for _, uri := range req.GetUris() {
		paths = append(paths, strings.TrimPrefix(uri, "file://"))
	}
	return &cline.RelativePaths{Paths: paths}, nil
}

func (s *FileService) SearchFiles(ctx context.Context, req *cline.FileSearchRequest) (*cline.FileSearchResults, error) {
	// Use find or fd if available
	limit := int32(20)
	if req.Limit != nil {
		limit = *req.Limit
	}
	var cmd *exec.Cmd
	if _, err := exec.LookPath("fd"); err == nil {
		cmd = exec.Command("fd", req.Query, ".", "--max-results", fmt.Sprintf("%d", limit))
	} else {
		cmd = exec.Command("find", ".", "-name", fmt.Sprintf("*%s*", req.Query), "-type", "f")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var results []*cline.FileInfo
	for i, line := range strings.Split(string(output), "\n") {
		if int32(i) >= limit {
			break
		}
		if line == "" {
			continue
		}
		label := filepath.Base(line)
		results = append(results, &cline.FileInfo{
			Path:  line,
			Type:  "file",
			Label: &label,
		})
	}
	return &cline.FileSearchResults{Results: results, MentionsRequestId: req.MentionsRequestId}, nil
}

func (s *FileService) ToggleClineRule(ctx context.Context, req *cline.ToggleClineRuleRequest) (*cline.ToggleClineRules, error) {
	key := fmt.Sprintf("rule_%s_%s", req.Scope, req.RulePath)
	_ = s.state.Set(key, req.Enabled)
	return &cline.ToggleClineRules{}, nil
}

func (s *FileService) ToggleCursorRule(ctx context.Context, req *cline.ToggleCursorRuleRequest) (*cline.ClineRulesToggles, error) {
	return &cline.ClineRulesToggles{}, nil
}

func (s *FileService) ToggleWindsurfRule(ctx context.Context, req *cline.ToggleWindsurfRuleRequest) (*cline.ClineRulesToggles, error) {
	return &cline.ClineRulesToggles{}, nil
}

func (s *FileService) ToggleAgentsRule(ctx context.Context, req *cline.ToggleAgentsRuleRequest) (*cline.ClineRulesToggles, error) {
	return &cline.ClineRulesToggles{}, nil
}

func (s *FileService) RefreshRules(ctx context.Context, req *cline.EmptyRequest) (*cline.RefreshedRules, error) {
	return &cline.RefreshedRules{}, nil
}

func (s *FileService) OpenDiskConversationHistory(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	path := req.Value
	return s.OpenFile(ctx, &cline.StringRequest{Value: path})
}

func (s *FileService) ToggleWorkflow(ctx context.Context, req *cline.ToggleWorkflowRequest) (*cline.ClineRulesToggles, error) {
	return &cline.ClineRulesToggles{}, nil
}

func (s *FileService) IfFileExistsRelativePath(ctx context.Context, req *cline.StringRequest) (*cline.BooleanResponse, error) {
	_, err := os.Stat(req.Value)
	return &cline.BooleanResponse{Value: err == nil}, nil
}

func (s *FileService) OpenFileRelativePath(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	return s.OpenFile(ctx, req)
}

func (s *FileService) OpenFocusChainFile(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	path := filepath.Join(".cline", "focus", req.Value+".md")
	return s.OpenFile(ctx, &cline.StringRequest{Value: path})
}

func (s *FileService) RefreshHooks(ctx context.Context, req *cline.EmptyRequest) (*cline.HooksToggles, error) {
	return &cline.HooksToggles{}, nil
}

func (s *FileService) ToggleHook(ctx context.Context, req *cline.ToggleHookRequest) (*cline.ToggleHookResponse, error) {
	return &cline.ToggleHookResponse{}, nil
}

func (s *FileService) CreateHook(ctx context.Context, req *cline.CreateHookRequest) (*cline.CreateHookResponse, error) {
	return &cline.CreateHookResponse{}, nil
}

func (s *FileService) DeleteHook(ctx context.Context, req *cline.DeleteHookRequest) (*cline.DeleteHookResponse, error) {
	return &cline.DeleteHookResponse{}, nil
}

func (s *FileService) RefreshSkills(ctx context.Context, req *cline.EmptyRequest) (*cline.RefreshedSkills, error) {
	return &cline.RefreshedSkills{}, nil
}

func (s *FileService) ToggleSkill(ctx context.Context, req *cline.ToggleSkillRequest) (*cline.SkillsToggles, error) {
	return &cline.SkillsToggles{}, nil
}

func (s *FileService) CreateSkillFile(ctx context.Context, req *cline.CreateSkillRequest) (*cline.SkillsToggles, error) {
	return &cline.SkillsToggles{}, nil
}

func (s *FileService) DeleteSkillFile(ctx context.Context, req *cline.DeleteSkillRequest) (*cline.SkillsToggles, error) {
	return &cline.SkillsToggles{}, nil
}

func (s *FileService) getRulesDir(isGlobal bool) string {
	if isGlobal {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".cline", "rules")
	}
	return ".cline/rules"
}

func (s *FileService) openURL(url string) (*cline.Empty, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return &cline.Empty{}, cmd.Start()
}
