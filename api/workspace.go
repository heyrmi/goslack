package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// createWorkspace creates a new workspace
func (server *Server) createWorkspace(ctx *gin.Context) {
	var req service.CreateWorkspaceRequest
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

	workspace, err := server.workspaceService.CreateWorkspace(ctx, user.ID, req)
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

	ctx.JSON(http.StatusCreated, workspace)
}

// getWorkspace retrieves a workspace by ID
func (server *Server) getWorkspace(ctx *gin.Context) {
	var req getWorkspaceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	workspace, err := server.workspaceService.GetWorkspace(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, workspace)
}

// listWorkspaces lists workspaces in the user's organization
func (server *Server) listWorkspaces(ctx *gin.Context) {
	var req listWorkspacesRequest
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

	workspaces, err := server.workspaceService.ListWorkspacesByOrganization(
		ctx,
		user.OrganizationID,
		req.PageSize,
		(req.PageID-1)*req.PageSize,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, workspaces)
}

// updateWorkspace updates a workspace's information
func (server *Server) updateWorkspace(ctx *gin.Context) {
	var uriReq getWorkspaceRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateWorkspaceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	workspace, err := server.workspaceService.UpdateWorkspace(ctx, uriReq.ID, req.Name)
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

	ctx.JSON(http.StatusOK, workspace)
}

// deleteWorkspace deletes a workspace
func (server *Server) deleteWorkspace(ctx *gin.Context) {
	var req getWorkspaceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.workspaceService.DeleteWorkspace(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "workspace deleted successfully"})
}

type getWorkspaceRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type listWorkspacesRequest struct {
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=5,max=50"`
}

type updateWorkspaceRequest struct {
	Name string `json:"name" binding:"required"`
}
