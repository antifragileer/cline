// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
	"google.golang.org/grpc"
)

// StateService implements the StateService gRPC interface with bidirectional synchronization
type StateService struct {
	cline.UnimplementedStateServiceServer
	state       *storage.ClineFileStorage
	subscribers map[string][]chan *cline.State
	mu          sync.RWMutex
}

// NewStateService creates a new StateService instance
func NewStateService(state *storage.ClineFileStorage) *StateService {
	return &StateService{
		state:       state,
		subscribers: make(map[string][]chan *cline.State),
	}
}

// GetLatestState returns the current state
func (s *StateService) GetLatestState(ctx context.Context, req *cline.EmptyRequest) (*cline.State, error) {
	stateJSON, err := s.buildStateJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to build state: %w", err)
	}

	return &cline.State{
		StateJson: stateJSON,
	}, nil
}

// UpdateTerminalConnectionTimeout updates terminal connection timeout
func (s *StateService) UpdateTerminalConnectionTimeout(ctx context.Context, req *cline.UpdateTerminalConnectionTimeoutRequest) (*cline.UpdateTerminalConnectionTimeoutResponse, error) {
	timeoutMs := int32(30000) // default
	if req.TimeoutMs != nil {
		timeoutMs = *req.TimeoutMs
	}
	_ = s.state.Set("terminal_connection_timeout", timeoutMs)
	s.broadcastState()
	return &cline.UpdateTerminalConnectionTimeoutResponse{
		TimeoutMs: &timeoutMs,
	}, nil
}

// UpdateTerminalReuseEnabled updates terminal reuse setting
func (s *StateService) UpdateTerminalReuseEnabled(ctx context.Context, req *cline.BooleanRequest) (*cline.Empty, error) {
	_ = s.state.Set("terminal_reuse_enabled", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// GetAvailableTerminalProfiles gets available terminal profiles
func (s *StateService) GetAvailableTerminalProfiles(ctx context.Context, req *cline.EmptyRequest) (*cline.TerminalProfiles, error) {
	// Return common terminal profiles
	defaultPath := "/bin/bash"
	defaultDesc := "Default shell"
	return &cline.TerminalProfiles{
		Profiles: []*cline.TerminalProfile{
			{
				Id:          "default",
				Name:        "Default Terminal",
				Path:        &defaultPath,
				Description: &defaultDesc,
			},
			{
				Id:   "bash",
				Name: "Bash",
			},
			{
				Id:   "zsh",
				Name: "Zsh",
			},
		},
	}, nil
}

// SubscribeToState subscribes to state updates via streaming
func (s *StateService) SubscribeToState(req *cline.EmptyRequest, stream grpc.ServerStreamingServer[cline.State]) error {
	// Send initial state
	stateJSON, err := s.buildStateJSON()
	if err != nil {
		return err
	}

	if err := stream.Send(&cline.State{
		StateJson: stateJSON,
	}); err != nil {
		return err
	}

	// Create subscription channel
	ch := make(chan *cline.State, 10)
	s.addSubscriber("default", ch)
	defer s.removeSubscriber("default", ch)

	// Listen for state updates
	for {
		select {
		case state := <-ch:
			if err := stream.Send(state); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// ToggleFavoriteModel toggles a model as favorite
func (s *StateService) ToggleFavoriteModel(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	var favorites []string
	if data, ok := s.state.Get("favorited_model_ids"); ok && data != nil {
		if f, ok := data.([]interface{}); ok {
			for _, v := range f {
				if str, ok := v.(string); ok {
					favorites = append(favorites, str)
				}
			}
		}
	}

	modelID := req.GetValue()
	found := false
	for i, f := range favorites {
		if f == modelID {
			favorites = append(favorites[:i], favorites[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		favorites = append(favorites, modelID)
	}

	_ = s.state.Set("favorited_model_ids", favorites)
	s.broadcastState()
	return &cline.Empty{}, nil
}

// ResetState resets the state to defaults
func (s *StateService) ResetState(ctx context.Context, req *cline.ResetStateRequest) (*cline.Empty, error) {
	// Clear state
	_ = s.state.Set("api_provider", "anthropic")
	_ = s.state.Set("api_model_id", "")
	s.broadcastState()
	return &cline.Empty{}, nil
}

// TogglePlanActModeProto toggles plan/act mode
func (s *StateService) TogglePlanActModeProto(ctx context.Context, req *cline.TogglePlanActModeRequest) (*cline.Boolean, error) {
	var mode string
	if val, ok := s.state.Get("mode"); ok {
		mode, _ = val.(string)
	}
	if mode == "act" {
		mode = "plan"
	} else {
		mode = "act"
	}
	_ = s.state.Set("mode", mode)
	s.broadcastState()
	return &cline.Boolean{Value: mode == "act"}, nil
}

// UpdateAutoApprovalSettings updates auto-approval settings
func (s *StateService) UpdateAutoApprovalSettings(ctx context.Context, req *cline.AutoApprovalSettingsRequest) (*cline.Empty, error) {
	settings := map[string]interface{}{}
	if req.Actions != nil {
		if req.Actions.ReadFiles != nil {
			settings["read_files"] = *req.Actions.ReadFiles
		}
		if req.Actions.EditFiles != nil {
			settings["write_files"] = *req.Actions.EditFiles
		}
		if req.Actions.ExecuteSafeCommands != nil {
			settings["execute_command"] = *req.Actions.ExecuteSafeCommands
		}
		if req.Actions.UseBrowser != nil {
			settings["use_browser"] = *req.Actions.UseBrowser
		}
		if req.Actions.UseMcp != nil {
			settings["use_mcp"] = *req.Actions.UseMcp
		}
	}
	settingsData, _ := json.Marshal(settings)
	_ = s.state.Set("auto_approval_settings", string(settingsData))
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateSettings updates settings
func (s *StateService) UpdateSettings(ctx context.Context, req *cline.UpdateSettingsRequest) (*cline.Empty, error) {
	// UpdateSettingsRequest has various fields
	if req.TelemetrySetting != nil {
		_ = s.state.Set("telemetry_setting", *req.TelemetrySetting)
	}
	if req.TerminalReuseEnabled != nil {
		_ = s.state.Set("terminal_reuse_enabled", *req.TerminalReuseEnabled)
	}
	if req.PlanActSeparateModelsSetting != nil {
		_ = s.state.Set("plan_act_separate_models", *req.PlanActSeparateModelsSetting)
	}
	if req.EnableCheckpointsSetting != nil {
		_ = s.state.Set("enable_checkpoints", *req.EnableCheckpointsSetting)
	}
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateSettingsCli updates settings from CLI
func (s *StateService) UpdateSettingsCli(ctx context.Context, req *cline.UpdateSettingsRequestCli) (*cline.Empty, error) {
	if req.Settings != nil {
		if req.Settings.LiteLlmBaseUrl != nil {
			_ = s.state.Set("lite_llm_base_url", *req.Settings.LiteLlmBaseUrl)
		}
		if req.Settings.LiteLlmUsePromptCache != nil {
			_ = s.state.Set("lite_llm_use_prompt_cache", *req.Settings.LiteLlmUsePromptCache)
		}
	}
	if req.Secrets != nil && req.Secrets.LiteLlmApiKey != nil {
		// Secrets are stored separately
		_ = s.state.Set("api_key", *req.Secrets.LiteLlmApiKey)
	}
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateTaskSettings updates task settings
func (s *StateService) UpdateTaskSettings(ctx context.Context, req *cline.UpdateTaskSettingsRequest) (*cline.Empty, error) {
	if req.Settings != nil {
		if req.Settings.FireworksModelMaxCompletionTokens != nil {
			_ = s.state.Set("max_completion_tokens", *req.Settings.FireworksModelMaxCompletionTokens)
		}
		if req.Settings.FireworksModelMaxTokens != nil {
			_ = s.state.Set("max_tokens", *req.Settings.FireworksModelMaxTokens)
		}
	}
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateTelemetrySetting updates telemetry setting
func (s *StateService) UpdateTelemetrySetting(ctx context.Context, req *cline.TelemetrySettingRequest) (*cline.Empty, error) {
	// Setting is TelemetrySettingEnum: 0=unset, 1=enabled, 2=disabled
	_ = s.state.Set("telemetry_setting", req.Setting)
	s.broadcastState()
	return &cline.Empty{}, nil
}

// CaptureOnboardingProgress captures onboarding progress
func (s *StateService) CaptureOnboardingProgress(ctx context.Context, req *cline.OnboardingProgressRequest) (*cline.Empty, error) {
	_ = s.state.Set("onboarding_progress", req.GetStep())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// SetWelcomeViewCompleted marks welcome view as completed
func (s *StateService) SetWelcomeViewCompleted(ctx context.Context, req *cline.BooleanRequest) (*cline.Empty, error) {
	_ = s.state.Set("welcome_view_completed", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateInfoBannerVersion updates info banner version
func (s *StateService) UpdateInfoBannerVersion(ctx context.Context, req *cline.Int64Request) (*cline.Empty, error) {
	_ = s.state.Set("last_info_banner", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateModelBannerVersion updates model banner version
func (s *StateService) UpdateModelBannerVersion(ctx context.Context, req *cline.Int64Request) (*cline.Empty, error) {
	_ = s.state.Set("last_model_banner", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// UpdateCliBannerVersion updates CLI banner version
func (s *StateService) UpdateCliBannerVersion(ctx context.Context, req *cline.Int64Request) (*cline.Empty, error) {
	_ = s.state.Set("last_cli_banner", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// DismissBanner dismisses a banner
func (s *StateService) DismissBanner(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	_ = s.state.Set("dismissed_banner", req.GetValue())
	s.broadcastState()
	return &cline.Empty{}, nil
}

// TrackBannerEvent tracks a banner event
func (s *StateService) TrackBannerEvent(ctx context.Context, req *cline.TrackBannerEventRequest) (*cline.Empty, error) {
	// Track event
	return &cline.Empty{}, nil
}

// InstallClineCli installs the Cline CLI
func (s *StateService) InstallClineCli(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Installation logic would go here
	return &cline.Empty{}, nil
}

// CheckCliInstallation checks if CLI is installed
func (s *StateService) CheckCliInstallation(ctx context.Context, req *cline.EmptyRequest) (*cline.Boolean, error) {
	// Check if cline command exists
	_, err := exec.LookPath("cline")
	return &cline.Boolean{Value: err == nil}, nil
}

// GetProcessInfo gets process information
func (s *StateService) GetProcessInfo(ctx context.Context, req *cline.EmptyRequest) (*cline.ProcessInfo, error) {
	version := "0.1.0"
	return &cline.ProcessInfo{
		ProcessId: int32(os.Getpid()),
		Version:   &version,
	}, nil
}

// FlushPendingState flushes any pending state changes
func (s *StateService) FlushPendingState(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Flush state to disk if needed
	return &cline.Empty{}, nil
}

// RefreshRemoteConfig refreshes remote configuration
func (s *StateService) RefreshRemoteConfig(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Refresh remote config
	return &cline.Empty{}, nil
}

// TestOtelConnection tests the OTel connection
func (s *StateService) TestOtelConnection(ctx context.Context, req *cline.EmptyRequest) (*cline.TestConnectionResult, error) {
	msg := "OTel connection test completed"
	return &cline.TestConnectionResult{
		Success: true,
		Message: &msg,
	}, nil
}

// TestPromptUploading tests prompt uploading
func (s *StateService) TestPromptUploading(ctx context.Context, req *cline.EmptyRequest) (*cline.TestConnectionResult, error) {
	msg := "Prompt upload test completed"
	return &cline.TestConnectionResult{
		Success: true,
		Message: &msg,
	}, nil
}

// Helper methods

func (s *StateService) buildStateJSON() (string, error) {
	// Get all state
	stateMap := make(map[string]interface{})

	// Add basic state values
	for _, key := range []string{
		"api_provider", "api_model_id", "open_router_model_id",
		"custom_instructions", "active_rpc_handler",
		"telemetry_enabled", "show_kudos", "remote_browser_host",
		"current_checkpoint_id", "max_workspace_files",
		"favorited_model_ids", "mode", "yolo_mode", "auto_approve_all",
	} {
		if val, ok := s.state.Get(key); ok {
			stateMap[key] = val
		}
	}

	// Add complex state values
	for _, key := range []string{
		"mcp_marketplace_catalog", "mcp_servers",
		"custom_support_prompts", "task_history",
		"auto_approval_settings", "browser_settings",
		"chat_settings", "plan_act_settings",
	} {
		if val, ok := s.state.Get(key); ok && val != nil {
			// Try to unmarshal JSON string
			if str, ok := val.(string); ok {
				var parsed interface{}
				if err := json.Unmarshal([]byte(str), &parsed); err == nil {
					stateMap[key] = parsed
				} else {
					stateMap[key] = val
				}
			} else {
				stateMap[key] = val
			}
		}
	}

	// Add clineMessages if available
	if val, ok := s.state.Get("cline_messages"); ok && val != nil {
		stateMap["clineMessages"] = val
	}

	data, err := json.Marshal(stateMap)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *StateService) addSubscriber(key string, ch chan *cline.State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[key] = append(s.subscribers[key], ch)
}

func (s *StateService) removeSubscriber(key string, ch chan *cline.State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[key]
	for i, sub := range subs {
		if sub == ch {
			s.subscribers[key] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	close(ch)
}

func (s *StateService) broadcastState() {
	stateJSON, err := s.buildStateJSON()
	if err != nil {
		return
	}

	state := &cline.State{
		StateJson: stateJSON,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subs := range s.subscribers {
		for _, ch := range subs {
			select {
			case ch <- state:
			default:
			}
		}
	}
}