package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// createChannel creates a new channel in a workspace
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
	user := currentUser.(service.UserResponse)

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

// getChannel retrieves a channel by ID
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
	user := currentUser.(service.UserResponse)

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

// listChannels lists channels in a workspace
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
	user := currentUser.(service.UserResponse)

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

// updateChannel updates a channel's information
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
	user := currentUser.(service.UserResponse)

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

// deleteChannel deletes a channel
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
	user := currentUser.(service.UserResponse)

	err := server.channelService.DeleteChannel(ctx, user.ID, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "channel deleted successfully"})
}

// updateUserRole updates a user's role in their workspace (admin only)
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
