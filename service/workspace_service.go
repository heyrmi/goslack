package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// WorkspaceService handles workspace-related business logic
type WorkspaceService struct {
	store       db.Store
	userService *UserService
}

// NewWorkspaceService creates a new workspace service
func NewWorkspaceService(store db.Store, userService *UserService) *WorkspaceService {
	return &WorkspaceService{
		store:       store,
		userService: userService,
	}
}

// CreateWorkspace creates a new workspace and assigns the creating user as admin
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, userID int64, req CreateWorkspaceRequest) (WorkspaceResponse, error) {
	// Get the user to find their organization
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return WorkspaceResponse{}, errors.New("user not found")
		}
		return WorkspaceResponse{}, fmt.Errorf("failed to get user: %w", err)
	}

	// Create the workspace
	arg := db.CreateWorkspaceParams{
		OrganizationID: user.OrganizationID,
		Name:           req.Name,
	}

	workspace, err := s.store.CreateWorkspace(ctx, arg)
	if err != nil {
		return WorkspaceResponse{}, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Assign the creating user as admin of the workspace
	_, err = s.userService.AssignUserToWorkspace(ctx, userID, workspace.ID, "admin")
	if err != nil {
		// If assigning user fails, we should clean up the workspace
		// In a real implementation, this should be done in a transaction
		s.store.DeleteWorkspace(ctx, workspace.ID)
		return WorkspaceResponse{}, fmt.Errorf("failed to assign user as admin: %w", err)
	}

	return s.toWorkspaceResponse(workspace), nil
}

// GetWorkspace retrieves a workspace by ID
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID int64) (WorkspaceResponse, error) {
	workspace, err := s.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return WorkspaceResponse{}, errors.New("workspace not found")
		}
		return WorkspaceResponse{}, fmt.Errorf("failed to get workspace: %w", err)
	}

	return s.toWorkspaceResponse(workspace), nil
}

// ListWorkspacesByOrganization lists workspaces in an organization with pagination
func (s *WorkspaceService) ListWorkspacesByOrganization(ctx context.Context, organizationID int64, limit, offset int32) ([]WorkspaceResponse, error) {
	arg := db.ListWorkspacesByOrganizationParams{
		OrganizationID: organizationID,
		Limit:          limit,
		Offset:         offset,
	}

	workspaces, err := s.store.ListWorkspacesByOrganization(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	workspaceResponses := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		workspaceResponses[i] = s.toWorkspaceResponse(workspace)
	}

	return workspaceResponses, nil
}

// UpdateWorkspace updates a workspace's information
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, workspaceID int64, name string) (WorkspaceResponse, error) {
	arg := db.UpdateWorkspaceParams{
		ID:   workspaceID,
		Name: name,
	}

	workspace, err := s.store.UpdateWorkspace(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return WorkspaceResponse{}, errors.New("workspace not found")
		}
		return WorkspaceResponse{}, fmt.Errorf("failed to update workspace: %w", err)
	}

	return s.toWorkspaceResponse(workspace), nil
}

// DeleteWorkspace deletes a workspace
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID int64) error {
	err := s.store.DeleteWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}
	return nil
}

// CheckUserWorkspaceAccess checks if a user has access to a workspace
func (s *WorkspaceService) CheckUserWorkspaceAccess(ctx context.Context, userID, workspaceID int64) error {
	isMember, err := s.userService.IsWorkspaceMember(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user does not have access to this workspace")
	}
	return nil
}

// CheckUserWorkspaceAdmin checks if a user is an admin of a workspace
func (s *WorkspaceService) CheckUserWorkspaceAdmin(ctx context.Context, userID, workspaceID int64) error {
	isAdmin, err := s.userService.IsWorkspaceAdmin(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return errors.New("user is not an admin of this workspace")
	}
	return nil
}

// toWorkspaceResponse converts a db.Workspace to WorkspaceResponse
func (s *WorkspaceService) toWorkspaceResponse(workspace db.Workspace) WorkspaceResponse {
	return WorkspaceResponse{
		ID:             workspace.ID,
		OrganizationID: workspace.OrganizationID,
		Name:           workspace.Name,
		CreatedAt:      workspace.CreatedAt,
	}
}
