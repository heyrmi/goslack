package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// @Summary Update User Status
// @Description Update user's online status in a workspace (requires workspace membership)
// @Tags status
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param status body service.UpdateUserStatusRequest true "Status update details"
// @Success 200 {object} service.UserStatusResponse "Status updated successfully"
// @Failure 400 {object} map[string]string "Invalid request or workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/status [put]
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

// @Summary Get User Status
// @Description Get a specific user's status in a workspace (requires workspace membership)
// @Tags status
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} service.UserStatusResponse "User status information"
// @Failure 400 {object} map[string]string "Invalid workspace ID or user ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/status/{user_id} [get]
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

// @Summary Get Workspace User Statuses
// @Description Get all user statuses in a workspace (requires workspace membership)
// @Tags status
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param limit query int false "Number of statuses to retrieve (default: 50, max: 100)" minimum(1) maximum(100)
// @Param offset query int false "Number of statuses to skip (default: 0)" minimum(0)
// @Success 200 {object} map[string]interface{} "Workspace user statuses"
// @Failure 400 {object} map[string]string "Invalid request or workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/status [get]
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

// @Summary Update User Activity
// @Description Update user's last activity timestamp (requires workspace membership)
// @Tags status
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {object} map[string]string "Activity updated successfully"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/activity [post]
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
