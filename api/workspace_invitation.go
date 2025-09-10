package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// @Summary Invite User to Workspace
// @Description Invite a user to join a workspace (requires workspace admin role)
// @Tags workspace-invitations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param invitation body service.InviteUserRequest true "Invitation details"
// @Success 201 {object} service.WorkspaceInvitationResponse "Invitation created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 409 {object} map[string]string "User already in workspace"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/invitations [post]
func (server *Server) inviteUserToWorkspace(ctx *gin.Context) {
	var req service.InviteUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	invitation, err := server.workspaceInvitationService.InviteUser(ctx, workspaceID, currentUser.ID, req)
	if err != nil {
		if err.Error() == "user is already a member of this workspace" {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, invitation)
}

// @Summary Join Workspace
// @Description Join a workspace using invitation code
// @Tags workspace-invitations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param invitation body service.JoinWorkspaceRequest true "Join workspace details"
// @Success 200 {object} service.UserResponse "Successfully joined workspace"
// @Failure 400 {object} map[string]string "Invalid request or invitation code"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 409 {object} map[string]string "User already in a workspace"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/join [post]
func (server *Server) joinWorkspace(ctx *gin.Context) {
	var req service.JoinWorkspaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	user, err := server.workspaceInvitationService.JoinWorkspace(ctx, currentUser.ID, req)
	if err != nil {
		switch err.Error() {
		case "invalid or expired invitation code":
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		case "invitation is not for this user", "user is already a member of another workspace":
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	ctx.JSON(http.StatusOK, user)
}

// @Summary List Workspace Invitations
// @Description List invitations for a workspace (requires workspace admin role)
// @Tags workspace-invitations
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param page_id query int false "Page ID (default: 1)"
// @Param page_size query int false "Page size (default: 10, max: 50)"
// @Success 200 {array} service.WorkspaceInvitationResponse "List of invitations"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/invitations [get]
func (server *Server) listWorkspaceInvitations(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get pagination parameters
	pageID := int32(1)
	pageSize := int32(10)

	if pageIDStr := ctx.Query("page_id"); pageIDStr != "" {
		if pid, err := strconv.ParseInt(pageIDStr, 10, 32); err == nil && pid > 0 {
			pageID = int32(pid)
		}
	}

	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.ParseInt(pageSizeStr, 10, 32); err == nil && ps > 0 && ps <= 50 {
			pageSize = int32(ps)
		}
	}

	offset := (pageID - 1) * pageSize

	invitations, err := server.workspaceInvitationService.ListWorkspaceInvitations(ctx, workspaceID, pageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, invitations)
}

// @Summary List Workspace Members
// @Description List members of a workspace (requires workspace membership)
// @Tags workspace-members
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param page_id query int false "Page ID (default: 1)"
// @Param page_size query int false "Page size (default: 10, max: 50)"
// @Success 200 {array} service.UserResponse "List of workspace members"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/members [get]
func (server *Server) listWorkspaceMembers(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get pagination parameters
	pageID := int32(1)
	pageSize := int32(10)

	if pageIDStr := ctx.Query("page_id"); pageIDStr != "" {
		if pid, err := strconv.ParseInt(pageIDStr, 10, 32); err == nil && pid > 0 {
			pageID = int32(pid)
		}
	}

	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.ParseInt(pageSizeStr, 10, 32); err == nil && ps > 0 && ps <= 50 {
			pageSize = int32(ps)
		}
	}

	offset := (pageID - 1) * pageSize

	members, err := server.workspaceInvitationService.ListWorkspaceMembers(ctx, workspaceID, pageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, members)
}

// @Summary Remove User from Workspace
// @Description Remove a user from workspace (requires workspace admin role)
// @Tags workspace-members
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} service.UserResponse "User removed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 404 {object} map[string]string "User not found in workspace"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/members/{user_id} [delete]
func (server *Server) removeUserFromWorkspace(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get user ID from URL
	userIDStr := ctx.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.workspaceInvitationService.RemoveUserFromWorkspace(ctx, userID, workspaceID)
	if err != nil {
		if err.Error() == "user not found in workspace" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// @Summary Update Workspace Member Role
// @Description Update a user's role in workspace (requires workspace admin role)
// @Tags workspace-members
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param user_id path int true "User ID"
// @Param role body service.UpdateUserRoleRequest true "Role update details"
// @Success 200 {object} service.UserResponse "Role updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 404 {object} map[string]string "User not found in workspace"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/members/{user_id}/role [put]
func (server *Server) updateWorkspaceMemberRole(ctx *gin.Context) {
	var req service.UpdateUserRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get user ID from URL
	userIDStr := ctx.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.workspaceInvitationService.UpdateWorkspaceMemberRole(ctx, userID, workspaceID, req.Role)
	if err != nil {
		if err.Error() == "user not found in workspace" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}
