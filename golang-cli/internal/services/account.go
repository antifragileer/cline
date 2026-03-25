// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// AccountService implements the AccountService gRPC interface
type AccountService struct {
	cline.UnimplementedAccountServiceServer
	secrets *storage.ClineFileStorage
	state   *storage.ClineFileStorage
}

// NewAccountService creates a new AccountService instance
func NewAccountService(secrets, state *storage.ClineFileStorage) *AccountService {
	return &AccountService{
		secrets: secrets,
		state:   state,
	}
}

// AccountLoginClicked handles the user clicking the login link in the UI
func (s *AccountService) AccountLoginClicked(ctx context.Context, req *cline.EmptyRequest) (*cline.String, error) {
	// Generate a secure nonce for state validation
	nonce, err := generateSecureNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Store nonce in secrets
	if err := s.secrets.Set("auth_nonce", nonce); err != nil {
		return nil, fmt.Errorf("failed to store nonce: %w", err)
	}

	// Build authentication URL
	authURL := buildAuthURL(nonce)

	return &cline.String{Value: authURL}, nil
}

// AccountLogoutClicked handles the user clicking the logout button
func (s *AccountService) AccountLogoutClicked(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Clear API keys
	_ = s.secrets.Delete("api_key")
	_ = s.secrets.Delete("auth_token")
	_ = s.secrets.Delete("auth_nonce")

	// Clear user state
	_ = s.state.Set("user_info", nil)

	return &cline.Empty{}, nil
}

// SubscribeToAuthStatusUpdate subscribes to auth status update events
func (s *AccountService) SubscribeToAuthStatusUpdate(req *cline.EmptyRequest, stream cline.AccountService_SubscribeToAuthStatusUpdateServer) error {
	// Get current auth state
	userInfo, err := s.getCurrentUserInfo()
	if err != nil {
		return err
	}

	// Send initial state
	if err := stream.Send(&cline.AuthState{
		User: userInfo,
	}); err != nil {
		return fmt.Errorf("failed to send auth state: %w", err)
	}

	// Keep stream open for updates (simplified - in production would use proper pub/sub)
	<-stream.Context().Done()
	return nil
}

// AuthStateChanged handles authentication state changes
func (s *AccountService) AuthStateChanged(ctx context.Context, req *cline.AuthStateChangedRequest) (*cline.AuthState, error) {
	if req.User == nil {
		// Clear user info
		if err := s.state.Set("user_info", nil); err != nil {
			return nil, fmt.Errorf("failed to clear user info: %w", err)
		}
		return &cline.AuthState{}, nil
	}

	// Store user info
	userData := map[string]interface{}{
		"uid":         req.User.Uid,
		"displayName": req.User.DisplayName,
		"email":       req.User.Email,
		"photoUrl":    req.User.PhotoUrl,
		"appBaseUrl":  req.User.AppBaseUrl,
	}
	if err := s.state.Set("user_info", userData); err != nil {
		return nil, fmt.Errorf("failed to store user info: %w", err)
	}

	return &cline.AuthState{
		User: req.User,
	}, nil
}

// GetUserCredits fetches all user credits data
func (s *AccountService) GetUserCredits(ctx context.Context, req *cline.EmptyRequest) (*cline.UserCreditsData, error) {
	// Get user ID from stored info
	userInfo, err := s.getCurrentUserInfo()
	if err != nil || userInfo == nil {
		return nil, fmt.Errorf("user not authenticated")
	}

	// In production, this would call the Cline API
	// For now, return placeholder data
	return &cline.UserCreditsData{
		Balance: &cline.UserCreditsBalance{
			CurrentBalance: 0.0,
		},
		UsageTransactions:    []*cline.UsageTransaction{},
		PaymentTransactions:  []*cline.PaymentTransaction{},
	}, nil
}

// GetOrganizationCredits fetches organization credits
func (s *AccountService) GetOrganizationCredits(ctx context.Context, req *cline.GetOrganizationCreditsRequest) (*cline.OrganizationCreditsData, error) {
	// Verify user is authenticated
	userInfo, err := s.getCurrentUserInfo()
	if err != nil || userInfo == nil {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Return placeholder data
	return &cline.OrganizationCreditsData{
		OrganizationId:      req.OrganizationId,
		Balance:             &cline.UserCreditsBalance{CurrentBalance: 0.0},
		UsageTransactions:   []*cline.OrganizationUsageTransaction{},
	}, nil
}

// GetUserOrganizations fetches all user organizations
func (s *AccountService) GetUserOrganizations(ctx context.Context, req *cline.EmptyRequest) (*cline.UserOrganizationsResponse, error) {
	// Verify user is authenticated
	userInfo, err := s.getCurrentUserInfo()
	if err != nil || userInfo == nil {
		return nil, fmt.Errorf("user not authenticated")
	}

	// Return placeholder - in production would fetch from API
	return &cline.UserOrganizationsResponse{
		Organizations: []*cline.UserOrganization{},
	}, nil
}

// SetUserOrganization sets the active user organization
func (s *AccountService) SetUserOrganization(ctx context.Context, req *cline.UserOrganizationUpdateRequest) (*cline.Empty, error) {
	if req.OrganizationId == nil {
		_ = s.state.Delete("active_organization_id")
	} else {
		if err := s.state.Set("active_organization_id", *req.OrganizationId); err != nil {
			return nil, fmt.Errorf("failed to set organization: %w", err)
		}
	}
	return &cline.Empty{}, nil
}

// OpenrouterAuthClicked handles OpenRouter auth
func (s *AccountService) OpenrouterAuthClicked(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	// Launch browser for OpenRouter OAuth
	authURL := "https://openrouter.ai/auth"
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}
	return &cline.Empty{}, nil
}

// RequestyAuthClicked handles Requesty auth
func (s *AccountService) RequestyAuthClicked(ctx context.Context, req *cline.StringRequest) (*cline.Empty, error) {
	// Store API key
	if err := s.secrets.Set("requesty_api_key", req.Value); err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}
	return &cline.Empty{}, nil
}

// HicapAuthClicked handles HiCap auth
func (s *AccountService) HicapAuthClicked(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	authURL := "https://hicap.dev/auth"
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}
	return &cline.Empty{}, nil
}

// GetRedirectUrl returns a redirect URL for the IDE
func (s *AccountService) GetRedirectUrl(ctx context.Context, req *cline.EmptyRequest) (*cline.String, error) {
	// Return the CLI's local callback URL
	return &cline.String{Value: "cline://auth/callback"}, nil
}

// OpenAiCodexSignIn handles OpenAI Codex OAuth
func (s *AccountService) OpenAiCodexSignIn(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	authURL := "https://chat.openai.com/auth"
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}
	return &cline.Empty{}, nil
}

// OpenAiCodexSignOut signs out of OpenAI Codex
func (s *AccountService) OpenAiCodexSignOut(ctx context.Context, req *cline.EmptyRequest) (*cline.Empty, error) {
	_ = s.secrets.Delete("openai_codex_token")
	return &cline.Empty{}, nil
}

// Helper methods

func (s *AccountService) getCurrentUserInfo() (*cline.UserInfo, error) {
	data, ok := s.state.Get("user_info")
	if !ok {
		return nil, nil
	}
	if data == nil {
		return nil, nil
	}

	userMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid user info format")
	}

	userInfo := &cline.UserInfo{}
	if uid, ok := userMap["uid"].(string); ok {
		userInfo.Uid = uid
	}
	if displayName, ok := userMap["displayName"].(string); ok {
		userInfo.DisplayName = &displayName
	}
	if email, ok := userMap["email"].(string); ok {
		userInfo.Email = &email
	}
	if photoUrl, ok := userMap["photoUrl"].(string); ok {
		userInfo.PhotoUrl = &photoUrl
	}
	if appBaseUrl, ok := userMap["appBaseUrl"].(string); ok {
		userInfo.AppBaseUrl = &appBaseUrl
	}

	return userInfo, nil
}

func generateSecureNonce() (string, error) {
	// In production, use crypto/rand to generate secure nonce
	// For now, return a placeholder
	return "secure_nonce_placeholder", nil
}

func buildAuthURL(nonce string) string {
	return fmt.Sprintf("https://cline.bot/auth?nonce=%s", nonce)
}

func openBrowser(url string) error {
	// Platform-specific browser opening
	// This is a placeholder - implement with actual platform detection
	return fmt.Errorf("browser opening not implemented")
}