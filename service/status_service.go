package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// StatusService handles user status-related business logic
type StatusService struct {
	store db.Store
	hub   WebSocketHub // Interface for WebSocket hub
}

// NewStatusService creates a new status service
func NewStatusService(store db.Store, hub WebSocketHub) *StatusService {
	return &StatusService{
		store: store,
		hub:   hub,
	}
}

// SetUserOnline sets a user as online in a workspace
func (s *StatusService) SetUserOnline(ctx context.Context, userID, workspaceID int64) error {
	arg := db.UpsertUserStatusParams{
		UserID:       userID,
		WorkspaceID:  workspaceID,
		Status:       "online",
		CustomStatus: sql.NullString{Valid: false},
	}

	userStatus, err := s.store.UpsertUserStatus(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to set user online: %w", err)
	}

	// Broadcast status change to WebSocket clients
	if s.hub != nil {
		statusResponse, err := s.toUserStatusResponse(ctx, userStatus)
		if err == nil {
			wsMessage := &WSMessage{
				Type:        "status_changed",
				Data:        statusResponse,
				WorkspaceID: workspaceID,
				UserID:      userID,
				Timestamp:   time.Now(),
			}
			s.hub.BroadcastToWorkspace(workspaceID, wsMessage)
		}
	}

	return nil
}

// SetUserOffline sets a user as offline in a workspace
func (s *StatusService) SetUserOffline(ctx context.Context, userID, workspaceID int64) error {
	arg := db.UpsertUserStatusParams{
		UserID:       userID,
		WorkspaceID:  workspaceID,
		Status:       "offline",
		CustomStatus: sql.NullString{Valid: false},
	}

	userStatus, err := s.store.UpsertUserStatus(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to set user offline: %w", err)
	}

	// Broadcast status change to WebSocket clients
	if s.hub != nil {
		statusResponse, err := s.toUserStatusResponse(ctx, userStatus)
		if err == nil {
			wsMessage := &WSMessage{
				Type:        "status_changed",
				Data:        statusResponse,
				WorkspaceID: workspaceID,
				UserID:      userID,
				Timestamp:   time.Now(),
			}
			s.hub.BroadcastToWorkspace(workspaceID, wsMessage)
		}
	}

	return nil
}

// SetUserStatus sets a user's status and custom status
func (s *StatusService) SetUserStatus(ctx context.Context, userID, workspaceID int64, status, customStatus string) (*UserStatusResponse, error) {
	arg := db.UpsertUserStatusParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Status:      status,
		CustomStatus: sql.NullString{
			String: customStatus,
			Valid:  customStatus != "",
		},
	}

	userStatus, err := s.store.UpsertUserStatus(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to set user status: %w", err)
	}

	statusResponse, err := s.toUserStatusResponse(ctx, userStatus)
	if err != nil {
		return nil, err
	}

	// Broadcast status change to WebSocket clients
	if s.hub != nil {
		wsMessage := &WSMessage{
			Type:        "status_changed",
			Data:        statusResponse,
			WorkspaceID: workspaceID,
			UserID:      userID,
			Timestamp:   time.Now(),
		}
		s.hub.BroadcastToWorkspace(workspaceID, wsMessage)
	}

	return statusResponse, nil
}

// GetUserStatus retrieves a user's status
func (s *StatusService) GetUserStatus(ctx context.Context, userID, workspaceID int64) (*UserStatusResponse, error) {
	arg := db.GetUserStatusParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}

	userStatus, err := s.store.GetUserStatus(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user status not found")
		}
		return nil, fmt.Errorf("failed to get user status: %w", err)
	}

	return s.toUserStatusResponse(ctx, userStatus)
}

// GetWorkspaceUserStatuses retrieves all user statuses in a workspace with pagination
func (s *StatusService) GetWorkspaceUserStatuses(ctx context.Context, workspaceID int64, limit, offset int32) ([]*UserStatusResponse, error) {
	arg := db.GetWorkspaceUserStatusesParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}

	statuses, err := s.store.GetWorkspaceUserStatuses(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace user statuses: %w", err)
	}

	return s.toUserStatusResponses(ctx, statuses), nil
}

// UpdateUserActivity updates a user's last activity timestamp
func (s *StatusService) UpdateUserActivity(ctx context.Context, userID, workspaceID int64) error {
	arg := db.UpdateLastActivityParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}

	err := s.store.UpdateLastActivity(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to update user activity: %w", err)
	}

	return nil
}

// SetInactiveUsersOffline sets users offline who have been inactive for a specified duration
func (s *StatusService) SetInactiveUsersOffline(ctx context.Context, inactivityDuration time.Duration) error {
	cutoffTime := time.Now().Add(-inactivityDuration)

	err := s.store.SetUsersOfflineAfterInactivity(ctx, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to set inactive users offline: %w", err)
	}

	return nil
}

// GetOnlineUsersInWorkspace retrieves all online users in a workspace
func (s *StatusService) GetOnlineUsersInWorkspace(ctx context.Context, workspaceID int64) ([]*UserStatusResponse, error) {
	statuses, err := s.store.GetOnlineUsersInWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get online users: %w", err)
	}

	return s.toOnlineUserStatusResponses(statuses), nil
}

// StartInactivityMonitor starts a background goroutine to monitor user inactivity
func (s *StatusService) StartInactivityMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Set users offline who have been inactive for 30 minutes
			err := s.SetInactiveUsersOffline(ctx, 30*time.Minute)
			if err != nil {
				// Log error but don't stop the monitor
				fmt.Printf("Error setting inactive users offline: %v\n", err)
			}
		}
	}
}

// Helper function to convert db user status to response
func (s *StatusService) toUserStatusResponse(ctx context.Context, userStatus db.UserStatus) (*UserStatusResponse, error) {
	// Get user info
	user, err := s.store.GetUser(ctx, userStatus.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Convert user to UserResponse
	var workspaceID *int64
	if user.WorkspaceID.Valid {
		workspaceID = &user.WorkspaceID.Int64
	}

	userResponse := UserResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Email:          user.Email,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		WorkspaceID:    workspaceID,
		Role:           user.Role,
		CreatedAt:      user.CreatedAt,
	}

	response := &UserStatusResponse{
		UserID:      userStatus.UserID,
		WorkspaceID: userStatus.WorkspaceID,
		Status:      userStatus.Status,
		LastSeenAt:  userStatus.LastSeenAt,
		User:        userResponse,
	}

	if userStatus.CustomStatus.Valid {
		response.CustomStatus = userStatus.CustomStatus.String
	}

	return response, nil
}

// Helper function to convert multiple user statuses to responses
func (s *StatusService) toUserStatusResponses(ctx context.Context, statuses []db.GetWorkspaceUserStatusesRow) []*UserStatusResponse {
	responses := make([]*UserStatusResponse, len(statuses))
	for i, status := range statuses {
		userResponse := UserResponse{
			ID:        status.UserID,
			Email:     status.Email,
			FirstName: status.FirstName,
			LastName:  status.LastName,
		}

		response := &UserStatusResponse{
			UserID:      status.UserID,
			WorkspaceID: status.WorkspaceID,
			Status:      status.Status,
			LastSeenAt:  status.LastSeenAt,
			User:        userResponse,
		}

		if status.CustomStatus.Valid {
			response.CustomStatus = status.CustomStatus.String
		}

		responses[i] = response
	}
	return responses
}

// Helper function to convert online user statuses to responses
func (s *StatusService) toOnlineUserStatusResponses(statuses []db.GetOnlineUsersInWorkspaceRow) []*UserStatusResponse {
	responses := make([]*UserStatusResponse, len(statuses))
	for i, status := range statuses {
		userResponse := UserResponse{
			ID:        status.UserID,
			Email:     status.Email,
			FirstName: status.FirstName,
			LastName:  status.LastName,
		}

		response := &UserStatusResponse{
			UserID:      status.UserID,
			WorkspaceID: status.WorkspaceID,
			Status:      status.Status,
			LastSeenAt:  status.LastSeenAt,
			User:        userResponse,
		}

		if status.CustomStatus.Valid {
			response.CustomStatus = status.CustomStatus.String
		}

		responses[i] = response
	}
	return responses
}
