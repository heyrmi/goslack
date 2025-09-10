package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// @Summary Create Channel
// @Description Create a new channel in a workspace (requires workspace membership)
// @Tags channels
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param channel body service.CreateChannelRequest true "Channel creation details"
// @Success 201 {object} service.ChannelResponse "Channel created successfully"
// @Failure 400 {object} map[string]string "Invalid request or foreign key violation"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 409 {object} map[string]string "Channel name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/channels [post]
func (server *Server) createChannel(ctx *gin.Context) {
	// Get workspace ID from URL parameter
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.CreateChannelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user from context
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		err := fmt.Errorf("user not found in context")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	user := currentUser.(*db.User)

	channel, err := server.channelService.CreateChannel(ctx, user.ID, workspaceID, req)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusConflict, errorResponse(err))
				return
			case "foreign_key_violation":
				ctx.JSON(http.StatusBadRequest, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, channel)
}

// @Summary Get Channel
// @Description Retrieve channel information by ID (requires channel access)
// @Tags channels
// @Security BearerAuth
// @Produce json
// @Param id path int true "Channel ID"
// @Success 200 {object} service.ChannelResponse "Channel information"
// @Failure 400 {object} map[string]string "Invalid channel ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Channel access required"
// @Failure 404 {object} map[string]string "Channel not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /channels/{id} [get]
func (server *Server) getChannel(ctx *gin.Context) {
	var req getChannelRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user from context
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		err := fmt.Errorf("user not found in context")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	user := currentUser.(*db.User)

	// Check if user has access to this channel
	err := server.channelService.CheckChannelAccess(ctx, user.ID, req.ID)
	if err != nil {
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	channel, err := server.channelService.GetChannel(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, channel)
}

// @Summary List Channels
// @Description List channels in a workspace (requires workspace membership)
// @Tags channels
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param page_id query int false "Page ID (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 50, max: 50)" minimum(5) maximum(50)
// @Success 200 {array} service.ChannelResponse "List of channels"
// @Failure 400 {object} map[string]string "Invalid request or pagination parameters"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/channels [get]
func (server *Server) listChannels(ctx *gin.Context) {
	// Get workspace ID from URL parameter
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.ListChannelsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default values if not provided
	if req.PageID == 0 {
		req.PageID = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 50
	}

	// Get current user from context
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		err := fmt.Errorf("user not found in context")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	user := currentUser.(*db.User)

	channels, err := server.channelService.ListChannelsByWorkspace(
		ctx,
		user.ID,
		workspaceID,
		req.PageSize,
		(req.PageID-1)*req.PageSize,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, channels)
}

// @Summary Update Channel
// @Description Update channel information (requires channel access)
// @Tags channels
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Channel ID"
// @Param channel body updateChannelRequest true "Channel update details"
// @Success 200 {object} service.ChannelResponse "Channel updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Channel access required"
// @Failure 409 {object} map[string]string "Channel name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /channels/{id} [put]
func (server *Server) updateChannel(ctx *gin.Context) {
	var uriReq getChannelRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateChannelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user from context
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		err := fmt.Errorf("user not found in context")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	user := currentUser.(*db.User)

	channel, err := server.channelService.UpdateChannel(ctx, user.ID, uriReq.ID, req.Name, req.IsPrivate)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusConflict, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, channel)
}

// @Summary Delete Channel
// @Description Delete a channel (requires channel access)
// @Tags channels
// @Security BearerAuth
// @Produce json
// @Param id path int true "Channel ID"
// @Success 200 {object} map[string]string "Channel deleted successfully"
// @Failure 400 {object} map[string]string "Invalid channel ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Channel access required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /channels/{id} [delete]
func (server *Server) deleteChannel(ctx *gin.Context) {
	var req getChannelRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user from context
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		err := fmt.Errorf("user not found in context")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	user := currentUser.(*db.User)

	err := server.channelService.DeleteChannel(ctx, user.ID, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "channel deleted successfully"})
}

// @Summary Update User Role
// @Description Update a user's role in their workspace (admin only, same workspace)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param user_id path int true "Target User ID"
// @Param role body service.UpdateUserRoleRequest true "Role update details"
// @Success 200 {object} service.UserResponse "User role updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Admin access required in same workspace"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{user_id}/role [patch]
func (server *Server) updateUserRole(ctx *gin.Context) {
	// Get target user ID from URL parameter
	targetUserIDStr := ctx.Param("user_id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.UpdateUserRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.userService.UpdateUserRole(ctx, targetUserID, req.Role)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

type getChannelRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateChannelRequest struct {
	Name      string `json:"name" binding:"required"`
	IsPrivate bool   `json:"is_private"`
}
