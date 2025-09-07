package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/lib/pq"
)

// @Summary Create Organization
// @Description Create a new organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param organization body service.CreateOrganizationRequest true "Organization creation details"
// @Success 200 {object} db.Organization "Organization created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Organization name already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /organizations [post]
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

// @Summary Get Organization
// @Description Retrieve an organization by ID
// @Tags organizations
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} db.Organization "Organization details"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 404 {object} map[string]string "Organization not found"
// @Router /organizations/{id} [get]
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

// @Summary Update Organization
// @Description Update an organization's information (requires authentication)
// @Tags organizations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Organization ID"
// @Param organization body service.CreateOrganizationRequest true "Organization update details"
// @Success 200 {object} db.Organization "Organization updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /organizations/{id} [put]
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

// @Summary List Organizations
// @Description List all organizations with pagination
// @Tags organizations
// @Produce json
// @Param page_id query int false "Page ID (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 10, max: 10)" minimum(5) maximum(10)
// @Success 200 {array} db.Organization "List of organizations"
// @Failure 400 {object} map[string]string "Invalid pagination parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /organizations [get]
func (server *Server) listOrganizations(ctx *gin.Context) {
	var req listOrganizationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default values if not provided
	if req.PageID == 0 {
		req.PageID = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	organizations, err := server.organizationService.ListOrganizations(ctx, req.PageSize, (req.PageID-1)*req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, organizations)
}

// @Summary Delete Organization
// @Description Delete an organization (requires authentication)
// @Tags organizations
// @Security BearerAuth
// @Produce json
// @Param id path int true "Organization ID"
// @Success 200 {object} map[string]string "Organization deleted successfully"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /organizations/{id} [delete]
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
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=5,max=10"`
}
