package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// updateUserStatus handles PUT /workspace/:id/status
func (server *Server) updateUserStatus(ctx *gin.Context) {
	var req service.UpdateUserStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Update status
	status, err := server.statusService.SetUserStatus(ctx, currentUser.ID, workspaceID, req.Status, req.CustomStatus)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// getUserStatus handles GET /workspace/:id/status/:user_id
func (server *Server) getUserStatus(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get user ID from URL
	userIDStr := ctx.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	// Get status
	status, err := server.statusService.GetUserStatus(ctx, userID, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// getWorkspaceUserStatuses handles GET /workspace/:id/status
func (server *Server) getWorkspaceUserStatuses(ctx *gin.Context) {
	var req service.GetMessagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get statuses
	statuses, err := server.statusService.GetWorkspaceUserStatuses(ctx, workspaceID, req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"statuses": statuses})
}

// updateUserActivity handles POST /workspace/:id/activity
func (server *Server) updateUserActivity(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Update activity
	err = server.statusService.UpdateUserActivity(ctx, currentUser.ID, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Activity updated successfully"})
}
