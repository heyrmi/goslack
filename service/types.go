package service

import "time"

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	OrganizationID int64  `json:"organization_id" binding:"required,min=1"`
	Email          string `json:"email" binding:"required,email"`
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	Password       string `json:"password" binding:"required,min=6"`
}

// LoginUserRequest represents the request to login a user
type LoginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginUserResponse represents the response after successful login
type LoginUserResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}

// UserResponse represents a user in API responses (without sensitive data)
type UserResponse struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	WorkspaceID    *int64    `json:"workspace_id,omitempty"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateOrganizationRequest represents the request to create a new organization
type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

// UpdateUserProfileRequest represents the request to update user profile
type UpdateUserProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// ChangePasswordRequest represents the request to change user password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// CreateWorkspaceRequest represents the request to create a new workspace
type CreateWorkspaceRequest struct {
	Name string `json:"name" binding:"required"`
}

// WorkspaceResponse represents a workspace in API responses
type WorkspaceResponse struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateChannelRequest represents the request to create a new channel
type CreateChannelRequest struct {
	Name      string `json:"name" binding:"required"`
	IsPrivate bool   `json:"is_private"`
}

// ChannelResponse represents a channel in API responses
type ChannelResponse struct {
	ID          int64     `json:"id"`
	WorkspaceID int64     `json:"workspace_id"`
	Name        string    `json:"name"`
	IsPrivate   bool      `json:"is_private"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateUserRoleRequest represents the request to update a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}

// ListChannelsRequest represents the request to list channels
type ListChannelsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=50"`
}
