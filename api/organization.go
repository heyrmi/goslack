package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// createOrganization creates a new organization
func (server *Server) createOrganization(ctx *gin.Context) {
	var req service.CreateOrganizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	organization, err := server.organizationService.CreateOrganization(ctx, req)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, organization)
}

// getOrganization retrieves an organization by ID
func (server *Server) getOrganization(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	organization, err := server.organizationService.GetOrganization(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, organization)
}

// updateOrganization updates an organization
func (server *Server) updateOrganization(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.CreateOrganizationRequest // Reusing the same struct since it only has name
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	organization, err := server.organizationService.UpdateOrganization(ctx, id, req.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, organization)
}

// listOrganizations lists organizations with pagination
func (server *Server) listOrganizations(ctx *gin.Context) {
	var req listOrganizationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	organizations, err := server.organizationService.ListOrganizations(ctx, req.PageSize, (req.PageID-1)*req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, organizations)
}

// deleteOrganization deletes an organization
func (server *Server) deleteOrganization(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.organizationService.DeleteOrganization(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "organization deleted successfully"})
}

type listOrganizationsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}
