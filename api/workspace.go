package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// @Summary Create Workspace
// @Description Create a new workspace in the user's organization
// @Tags workspaces
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspace body service.CreateWorkspaceRequest true "Workspace creation details"
// @Success 201 {object} service.WorkspaceResponse "Workspace created successfully"
// @Failure 400 {object} map[string]string "Invalid request or foreign key violation"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 409 {object} map[string]string "Workspace name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces [post]
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
	user := currentUser.(*db.User)

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

// @Summary Get Workspace
// @Description Retrieve workspace information by ID
// @Tags workspaces
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {object} service.WorkspaceResponse "Workspace information"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 404 {object} map[string]string "Workspace not found"
// @Router /workspaces/{id} [get]
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

// @Summary List Workspaces
// @Description List workspaces in the authenticated user's organization
// @Tags workspaces
// @Security BearerAuth
// @Produce json
// @Param page_id query int false "Page ID (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 50, max: 50)" minimum(5) maximum(50)
// @Success 200 {array} service.WorkspaceResponse "List of workspaces"
// @Failure 400 {object} map[string]string "Invalid pagination parameters"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces [get]
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
	user := currentUser.(*db.User)

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

// @Summary Update Workspace
// @Description Update workspace information (requires workspace admin role)
// @Tags workspaces
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param workspace body updateWorkspaceRequest true "Workspace update details"
// @Success 200 {object} service.WorkspaceResponse "Workspace updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 409 {object} map[string]string "Workspace name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id} [put]
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

// @Summary Delete Workspace
// @Description Delete a workspace (requires workspace admin role)
// @Tags workspaces
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {object} map[string]string "Workspace deleted successfully"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace admin access required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id} [delete]
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
