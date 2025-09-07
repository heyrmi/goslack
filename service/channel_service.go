package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// ChannelService handles channel-related business logic
type ChannelService struct {
	store            db.Store
	userService      *UserService
	workspaceService *WorkspaceService
}

// NewChannelService creates a new channel service
func NewChannelService(store db.Store, userService *UserService, workspaceService *WorkspaceService) *ChannelService {
	return &ChannelService{
		store:            store,
		userService:      userService,
		workspaceService: workspaceService,
	}
}

// CreateChannel creates a new channel in a workspace
// Note: This method assumes workspace access has been validated by middleware
func (s *ChannelService) CreateChannel(ctx context.Context, userID, workspaceID int64, req CreateChannelRequest) (ChannelResponse, error) {
	// Create the channel
	arg := db.CreateChannelParams{
		WorkspaceID: workspaceID,
		Name:        req.Name,
		IsPrivate:   req.IsPrivate,
		CreatedBy:   userID,
	}

	channel, err := s.store.CreateChannel(ctx, arg)
	if err != nil {
		return ChannelResponse{}, fmt.Errorf("failed to create channel: %w", err)
	}

	return s.toChannelResponse(channel), nil
}

// GetChannel retrieves a channel by ID
func (s *ChannelService) GetChannel(ctx context.Context, channelID int64) (ChannelResponse, error) {
	channel, err := s.store.GetChannelByID(ctx, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ChannelResponse{}, errors.New("channel not found")
		}
		return ChannelResponse{}, fmt.Errorf("failed to get channel: %w", err)
	}

	return s.toChannelResponse(channel), nil
}

// ListChannelsByWorkspace lists all channels in a workspace for workspace members
// Note: This method assumes workspace membership has been validated by middleware
func (s *ChannelService) ListChannelsByWorkspace(ctx context.Context, userID, workspaceID int64, limit, offset int32) ([]ChannelResponse, error) {
	// User is a member (validated by middleware), show all channels
	arg := db.ListChannelsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}
	channels, err := s.store.ListChannelsByWorkspace(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	channelResponses := make([]ChannelResponse, len(channels))
	for i, channel := range channels {
		channelResponses[i] = s.toChannelResponse(channel)
	}

	return channelResponses, nil
}

// ListPublicChannelsByWorkspace lists only public channels in a workspace
// This can be used for non-members or public API endpoints
func (s *ChannelService) ListPublicChannelsByWorkspace(ctx context.Context, workspaceID int64, limit, offset int32) ([]ChannelResponse, error) {
	arg := db.ListPublicChannelsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}
	channels, err := s.store.ListPublicChannelsByWorkspace(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list public channels: %w", err)
	}

	channelResponses := make([]ChannelResponse, len(channels))
	for i, channel := range channels {
		channelResponses[i] = s.toChannelResponse(channel)
	}

	return channelResponses, nil
}

// UpdateChannel updates a channel's information
func (s *ChannelService) UpdateChannel(ctx context.Context, userID, channelID int64, name string, isPrivate bool) (ChannelResponse, error) {
	// Get the channel first to check workspace access
	channel, err := s.store.GetChannelByID(ctx, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ChannelResponse{}, errors.New("channel not found")
		}
		return ChannelResponse{}, fmt.Errorf("failed to get channel: %w", err)
	}

	// Check if user has access to the workspace (only admins can update channels for now)
	err = s.workspaceService.CheckUserWorkspaceAdmin(ctx, userID, channel.WorkspaceID)
	if err != nil {
		return ChannelResponse{}, err
	}

	// Update the channel
	arg := db.UpdateChannelParams{
		ID:        channelID,
		Name:      name,
		IsPrivate: isPrivate,
	}

	updatedChannel, err := s.store.UpdateChannel(ctx, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			return ChannelResponse{}, errors.New("channel not found")
		}
		return ChannelResponse{}, fmt.Errorf("failed to update channel: %w", err)
	}

	return s.toChannelResponse(updatedChannel), nil
}

// DeleteChannel deletes a channel
func (s *ChannelService) DeleteChannel(ctx context.Context, userID, channelID int64) error {
	// Get the channel first to check workspace access
	channel, err := s.store.GetChannelByID(ctx, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("channel not found")
		}
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Check if user has admin access to the workspace
	err = s.workspaceService.CheckUserWorkspaceAdmin(ctx, userID, channel.WorkspaceID)
	if err != nil {
		return err
	}

	err = s.store.DeleteChannel(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	return nil
}

// CheckChannelAccess checks if a user can access a specific channel
func (s *ChannelService) CheckChannelAccess(ctx context.Context, userID, channelID int64) error {
	channel, err := s.store.GetChannelByID(ctx, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("channel not found")
		}
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// If channel is public, check workspace membership
	if !channel.IsPrivate {
		return s.workspaceService.CheckUserWorkspaceAccess(ctx, userID, channel.WorkspaceID)
	}

	// For private channels, user must be a workspace member
	// In Phase 3, we'll add channel-specific membership
	isMember, err := s.userService.IsWorkspaceMember(ctx, userID, channel.WorkspaceID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("access denied to private channel")
	}

	return nil
}

// UserHasChannelAccess checks if a user has access to a channel (returns boolean)
func (s *ChannelService) UserHasChannelAccess(userID, channelID int64) bool {
	ctx := context.Background()
	err := s.CheckChannelAccess(ctx, userID, channelID)
	return err == nil
}

// toChannelResponse converts a db.Channel to ChannelResponse
func (s *ChannelService) toChannelResponse(channel db.Channel) ChannelResponse {
	return ChannelResponse{
		ID:          channel.ID,
		WorkspaceID: channel.WorkspaceID,
		Name:        channel.Name,
		IsPrivate:   channel.IsPrivate,
		CreatedBy:   channel.CreatedBy,
		CreatedAt:   channel.CreatedAt,
	}
}
