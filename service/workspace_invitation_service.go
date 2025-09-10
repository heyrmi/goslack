package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// WorkspaceInvitationService handles workspace invitation logic
type WorkspaceInvitationService struct {
	store db.Store
}

// NewWorkspaceInvitationService creates a new workspace invitation service
func NewWorkspaceInvitationService(store db.Store) *WorkspaceInvitationService {
	return &WorkspaceInvitationService{
		store: store,
	}
}

// InviteUserRequest represents the request to invite a user to workspace
type InviteUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin member"`
}

// WorkspaceInvitationResponse represents workspace invitation in API responses
type WorkspaceInvitationResponse struct {
	ID             int64              `json:"id"`
	WorkspaceID    int64              `json:"workspace_id"`
	InviterID      int64              `json:"inviter_id"`
	InviteeEmail   string             `json:"invitee_email"`
	InviteeID      *int64             `json:"invitee_id,omitempty"`
	InvitationCode string             `json:"invitation_code"`
	Role           string             `json:"role"`
	Status         string             `json:"status"`
	ExpiresAt      time.Time          `json:"expires_at"`
	AcceptedAt     *time.Time         `json:"accepted_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	Inviter        *UserResponse      `json:"inviter,omitempty"`
	Workspace      *WorkspaceResponse `json:"workspace,omitempty"`
}

// JoinWorkspaceRequest represents the request to join a workspace
type JoinWorkspaceRequest struct {
	InvitationCode string `json:"invitation_code" binding:"required"`
}

// generateInvitationCode generates a secure random invitation code
func (s *WorkspaceInvitationService) generateInvitationCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 8

	bytes := make([]byte, codeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// InviteUser invites a user to join a workspace
func (s *WorkspaceInvitationService) InviteUser(ctx context.Context, workspaceID, inviterID int64, req InviteUserRequest) (WorkspaceInvitationResponse, error) {
	// Check if user is already in the workspace
	existingUser, err := s.store.GetUserByEmail(ctx, req.Email)
	if err == nil {
		// User exists, check if they're already in this workspace
		if existingUser.WorkspaceID.Valid && existingUser.WorkspaceID.Int64 == workspaceID {
			return WorkspaceInvitationResponse{}, errors.New("user is already a member of this workspace")
		}
	}

	// Generate invitation code
	invitationCode, err := s.generateInvitationCode()
	if err != nil {
		return WorkspaceInvitationResponse{}, fmt.Errorf("failed to generate invitation code: %w", err)
	}

	// Set expiration (7 days from now)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create invitation
	var inviteeID sql.NullInt64
	if err == nil {
		inviteeID = sql.NullInt64{Int64: existingUser.ID, Valid: true}
	}

	arg := db.CreateWorkspaceInvitationParams{
		WorkspaceID:    workspaceID,
		InviterID:      inviterID,
		InviteeEmail:   req.Email,
		InviteeID:      inviteeID,
		InvitationCode: invitationCode,
		Role:           req.Role,
		ExpiresAt:      expiresAt,
	}

	invitation, err := s.store.CreateWorkspaceInvitation(ctx, arg)
	if err != nil {
		return WorkspaceInvitationResponse{}, fmt.Errorf("failed to create invitation: %w", err)
	}

	return s.toInvitationResponse(invitation), nil
}

// JoinWorkspace allows a user to join a workspace using invitation code
func (s *WorkspaceInvitationService) JoinWorkspace(ctx context.Context, userID int64, req JoinWorkspaceRequest) (UserResponse, error) {
	// Get the invitation
	invitation, err := s.store.GetWorkspaceInvitationByCode(ctx, req.InvitationCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("invalid or expired invitation code")
		}
		return UserResponse{}, fmt.Errorf("failed to get invitation: %w", err)
	}

	// Get the user
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify the invitation is for this user (by email)
	if !strings.EqualFold(user.Email, invitation.InviteeEmail) {
		return UserResponse{}, errors.New("invitation is not for this user")
	}

	// Check if user is already in a workspace
	if user.WorkspaceID.Valid {
		return UserResponse{}, errors.New("user is already a member of another workspace")
	}

	// Add user to workspace
	addUserArg := db.AddUserToWorkspaceParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: invitation.WorkspaceID, Valid: true},
		Role:        invitation.Role,
	}

	updatedUser, err := s.store.AddUserToWorkspace(ctx, addUserArg)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to add user to workspace: %w", err)
	}

	// Accept the invitation
	_, err = s.store.AcceptWorkspaceInvitation(ctx, db.AcceptWorkspaceInvitationParams{
		InvitationCode: req.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to update invitation status: %v\n", err)
	}

	return s.toUserResponse(updatedUser), nil
}

// ListWorkspaceInvitations lists invitations for a workspace
func (s *WorkspaceInvitationService) ListWorkspaceInvitations(ctx context.Context, workspaceID int64, limit, offset int32) ([]WorkspaceInvitationResponse, error) {
	arg := db.ListWorkspaceInvitationsParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}

	invitations, err := s.store.ListWorkspaceInvitations(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	responses := make([]WorkspaceInvitationResponse, len(invitations))
	for i, invitation := range invitations {
		responses[i] = s.toInvitationResponse(invitation)
	}

	return responses, nil
}

// RemoveUserFromWorkspace removes a user from workspace
func (s *WorkspaceInvitationService) RemoveUserFromWorkspace(ctx context.Context, userID, workspaceID int64) (UserResponse, error) {
	arg := db.RemoveUserFromWorkspaceParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: workspaceID, Valid: true},
	}

	user, err := s.store.RemoveUserFromWorkspace(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found in workspace")
		}
		return UserResponse{}, fmt.Errorf("failed to remove user from workspace: %w", err)
	}

	return s.toUserResponse(user), nil
}

// UpdateWorkspaceMemberRole updates a user's role in workspace
func (s *WorkspaceInvitationService) UpdateWorkspaceMemberRole(ctx context.Context, userID, workspaceID int64, role string) (UserResponse, error) {
	arg := db.UpdateWorkspaceMemberRoleParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: workspaceID, Valid: true},
		Role:        role,
	}

	user, err := s.store.UpdateWorkspaceMemberRole(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return UserResponse{}, errors.New("user not found in workspace")
		}
		return UserResponse{}, fmt.Errorf("failed to update user role: %w", err)
	}

	return s.toUserResponse(user), nil
}

// ListWorkspaceMembers lists members of a workspace
func (s *WorkspaceInvitationService) ListWorkspaceMembers(ctx context.Context, workspaceID int64, limit, offset int32) ([]UserResponse, error) {
	arg := db.ListWorkspaceMembersParams{
		WorkspaceID: sql.NullInt64{Int64: workspaceID, Valid: true},
		Limit:       limit,
		Offset:      offset,
	}

	users, err := s.store.ListWorkspaceMembers(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace members: %w", err)
	}

	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = s.toUserResponseFromMemberRow(user)
	}

	return responses, nil
}

func (s *WorkspaceInvitationService) toInvitationResponse(invitation db.WorkspaceInvitation) WorkspaceInvitationResponse {
	resp := WorkspaceInvitationResponse{
		ID:             invitation.ID,
		WorkspaceID:    invitation.WorkspaceID,
		InviterID:      invitation.InviterID,
		InviteeEmail:   invitation.InviteeEmail,
		InvitationCode: invitation.InvitationCode,
		Role:           invitation.Role,
		Status:         invitation.Status,
		ExpiresAt:      invitation.ExpiresAt,
		CreatedAt:      invitation.CreatedAt,
	}

	if invitation.InviteeID.Valid {
		resp.InviteeID = &invitation.InviteeID.Int64
	}

	if invitation.AcceptedAt.Valid {
		resp.AcceptedAt = &invitation.AcceptedAt.Time
	}

	return resp
}

func (s *WorkspaceInvitationService) toUserResponse(user db.User) UserResponse {
	resp := UserResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Role:           user.Role,
		CreatedAt:      user.CreatedAt,
	}

	if user.WorkspaceID.Valid {
		resp.WorkspaceID = &user.WorkspaceID.Int64
	}

	return resp
}

func (s *WorkspaceInvitationService) toUserResponseFromMemberRow(user db.ListWorkspaceMembersRow) UserResponse {
	resp := UserResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Role:           user.Role,
		CreatedAt:      user.CreatedAt,
	}

	if user.WorkspaceID.Valid {
		resp.WorkspaceID = &user.WorkspaceID.Int64
	}

	return resp
}
