package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
)

// UserService handles user-related business logic
type UserService struct {
	store       db.Store
	tokenMaker  token.Maker
	config      util.Config
	authService *AuthService // Add reference to auth service for lockout checks
}

// NewUserService creates a new user service
func NewUserService(store db.Store, tokenMaker token.Maker, config util.Config) *UserService {
	return &UserService{
		store:      store,
		tokenMaker: tokenMaker,
		config:     config,
	}
}

// SetAuthService sets the auth service reference (to avoid circular dependency)
func (s *UserService) SetAuthService(authService *AuthService) {
	s.authService = authService
}

// CreateUser creates a new user in the specified organization
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	// Hash the password
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create the user in the database
	arg := db.CreateUserParams{
		OrganizationID: req.OrganizationID,
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		HashedPassword: hashedPassword,
	}

	user, err := s.store.CreateUser(ctx, arg)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to create user: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// LoginUser authenticates a user and returns an access token
func (s *UserService) LoginUser(ctx context.Context, req LoginUserRequest) (LoginUserResponse, error) {
	// Get user from database
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return LoginUserResponse{}, errors.New("user not found")
		}
		return LoginUserResponse{}, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if account is locked (if auth service is available)
	if s.authService != nil {
		locked, err := s.authService.IsAccountLocked(ctx, user.ID)
		if err != nil {
			return LoginUserResponse{}, fmt.Errorf("failed to check account lock status: %w", err)
		}
		if locked {
			// Log failed login attempt due to locked account
			s.logSecurityEvent(ctx, user.ID, "login_failed", "Login attempt on locked account", req.IPAddress, req.UserAgent)
			return LoginUserResponse{}, errors.New("account is locked due to security reasons")
		}
	}

	// Check if password is correct
	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		// Handle failed login attempt
		if s.authService != nil {
			err = s.authService.CheckAccountLockout(ctx, user.ID, req.IPAddress, req.UserAgent)
			if err != nil {
				// Log error but don't fail the login response
				fmt.Printf("Failed to update account lockout: %v\n", err)
			}
		}

		// Log failed login attempt
		s.logSecurityEvent(ctx, user.ID, "login_failed", "Invalid password", req.IPAddress, req.UserAgent)
		return LoginUserResponse{}, errors.New("incorrect password")
	}

	// Check email verification
	if !user.EmailVerified {
		return LoginUserResponse{}, errors.New("email not verified. Please check your email for verification link")
	}

	// Reset failed attempts on successful login
	if s.authService != nil {
		err = s.authService.ResetFailedAttempts(ctx, user.ID)
		if err != nil {
			// Log error but don't fail the login
			fmt.Printf("Failed to reset failed attempts: %v\n", err)
		}
	}

	// Create access token
	accessToken, _, err := s.tokenMaker.CreateToken(
		user.Email,
		s.config.AccessTokenDuration,
	)
	if err != nil {
		return LoginUserResponse{}, fmt.Errorf("failed to create access token: %w", err)
	}

	// Log successful login
	s.logSecurityEvent(ctx, user.ID, "login_success", "Successful login", req.IPAddress, req.UserAgent)

	rsp := LoginUserResponse{
		AccessToken: accessToken,
		User:        s.ToUserResponse(user),
	}

	return rsp, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID int64) (UserResponse, error) {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found")
		}
		return UserResponse{}, fmt.Errorf("failed to get user: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (UserResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found")
		}
		return UserResponse{}, fmt.Errorf("failed to get user: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// GetUserByEmailRaw retrieves a raw db.User by email (for internal use)
func (s *UserService) GetUserByEmailRaw(ctx context.Context, email string) (*db.User, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserRaw retrieves a raw db.User by ID (for internal use)
func (s *UserService) GetUserRaw(ctx context.Context, userID int64) (*db.User, error) {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateUserProfile updates a user's profile information
func (s *UserService) UpdateUserProfile(ctx context.Context, userID int64, req UpdateUserProfileRequest) (UserResponse, error) {
	arg := db.UpdateUserProfileParams{
		ID:        userID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	user, err := s.store.UpdateUserProfile(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found")
		}
		return UserResponse{}, fmt.Errorf("failed to update user profile: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, userID int64, req ChangePasswordRequest) error {
	// Get current user
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if old password is correct
	err = util.CheckPassword(req.OldPassword, user.HashedPassword)
	if err != nil {
		return errors.New("incorrect old password")
	}

	// Hash the new password
	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	arg := db.UpdateUserPasswordParams{
		ID:             userID,
		HashedPassword: hashedPassword,
	}

	_, err = s.store.UpdateUserPassword(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ListUsers lists users in an organization with pagination
func (s *UserService) ListUsers(ctx context.Context, organizationID int64, limit, offset int32) ([]UserResponse, error) {
	arg := db.ListUsersParams{
		OrganizationID: organizationID,
		Limit:          limit,
		Offset:         offset,
	}

	users, err := s.store.ListUsers(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = s.ToUserResponse(user)
	}

	return userResponses, nil
}

// UpdateUserRole updates a user's role in their workspace
func (s *UserService) UpdateUserRole(ctx context.Context, userID int64, role string) (UserResponse, error) {
	arg := db.UpdateUserRoleParams{
		ID:   userID,
		Role: role,
	}

	user, err := s.store.UpdateUserRole(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found")
		}
		return UserResponse{}, fmt.Errorf("failed to update user role: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// AssignUserToWorkspace assigns a user to a workspace with a specific role
func (s *UserService) AssignUserToWorkspace(ctx context.Context, userID, workspaceID int64, role string) (UserResponse, error) {
	arg := db.UpdateUserWorkspaceParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: workspaceID, Valid: true},
		Role:        role,
	}

	user, err := s.store.UpdateUserWorkspace(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found")
		}
		return UserResponse{}, fmt.Errorf("failed to assign user to workspace: %w", err)
	}

	return s.ToUserResponse(user), nil
}

// CheckUserWorkspaceRole checks if a user has a specific role in a workspace
func (s *UserService) CheckUserWorkspaceRole(ctx context.Context, userID, workspaceID int64) (string, error) {
	arg := db.CheckUserWorkspaceRoleParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: workspaceID, Valid: true},
	}

	role, err := s.store.CheckUserWorkspaceRole(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("user not found in workspace")
		}
		return "", fmt.Errorf("failed to check user role: %w", err)
	}

	return role, nil
}

// IsWorkspaceAdmin checks if a user is an admin in a workspace
func (s *UserService) IsWorkspaceAdmin(ctx context.Context, userID, workspaceID int64) (bool, error) {
	role, err := s.CheckUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		// If user not found in workspace, they're not an admin
		if err.Error() == "user not found in workspace" {
			return false, nil
		}
		return false, err
	}
	return role == "admin", nil
}

// IsWorkspaceMember checks if a user is a member (admin or member) in a workspace
func (s *UserService) IsWorkspaceMember(ctx context.Context, userID, workspaceID int64) (bool, error) {
	role, err := s.CheckUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		// If user not found in workspace, they're not a member
		if err.Error() == "user not found in workspace" {
			return false, nil
		}
		return false, err
	}
	return role == "admin" || role == "member", nil
}

// UserBelongsToWorkspace checks if a user belongs to a workspace (for backward compatibility)
func (s *UserService) UserBelongsToWorkspace(userID, workspaceID int64) bool {
	ctx := context.Background()
	isMember, err := s.IsWorkspaceMember(ctx, userID, workspaceID)
	if err != nil {
		return false
	}
	return isMember
}

// toUserResponse converts a db.User to UserResponse (removes sensitive data)
func (s *UserService) ToUserResponse(user db.User) UserResponse {
	var workspaceID *int64
	if user.WorkspaceID.Valid {
		workspaceID = &user.WorkspaceID.Int64
	}

	return UserResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		WorkspaceID:    workspaceID,
		Role:           user.Role,
		CreatedAt:      user.CreatedAt,
	}
}

// logSecurityEvent logs a security event (helper method)
func (s *UserService) logSecurityEvent(ctx context.Context, userID int64, eventType, description, ipAddress, userAgent string) {
	if s.authService != nil {
		req := SecurityEventRequest{
			UserID:      userID,
			EventType:   eventType,
			Description: description,
			IPAddress:   net.ParseIP(ipAddress),
			UserAgent:   userAgent,
		}
		s.authService.logSecurityEvent(ctx, req)
	}
}
