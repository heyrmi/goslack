package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
	"github.com/lib/pq"
)

// createUser creates a new user
func (server *Server) createUser(ctx *gin.Context) {
	var req service.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.userService.CreateUser(ctx, req)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			case "foreign_key_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// loginUser handles user login
func (server *Server) loginUser(ctx *gin.Context) {
	var req service.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.userService.LoginUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// getUser retrieves a user by ID
func (server *Server) getUser(ctx *gin.Context) {
	var req getUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	user, err := server.userService.GetUserByEmail(ctx, authPayload.Username)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	// Check if the user is requesting their own data or if they're in the same organization
	if user.ID != req.ID {
		// Additional authorization logic can be added here
		// For now, we allow users to view other users in the same organization
		requestedUser, err := server.userService.GetUser(ctx, req.ID)
		if err != nil {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		if requestedUser.OrganizationID != user.OrganizationID {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusOK, requestedUser)
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// updateUserProfile updates a user's profile
func (server *Server) updateUserProfile(ctx *gin.Context) {
	var uriReq getUserRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.UpdateUserProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	user, err := server.userService.GetUserByEmail(ctx, authPayload.Username)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	// Users can only update their own profile
	if user.ID != uriReq.ID {
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	updatedUser, err := server.userService.UpdateUserProfile(ctx, user.ID, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, updatedUser)
}

// changePassword changes a user's password
func (server *Server) changePassword(ctx *gin.Context) {
	var uriReq getUserRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req service.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	user, err := server.userService.GetUserByEmail(ctx, authPayload.Username)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	// Users can only change their own password
	if user.ID != uriReq.ID {
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	err = server.userService.ChangePassword(ctx, user.ID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// listUsers lists users in an organization
func (server *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	user, err := server.userService.GetUserByEmail(ctx, authPayload.Username)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(err))
		return
	}

	users, err := server.userService.ListUsers(ctx, user.OrganizationID, req.PageSize, (req.PageID-1)*req.PageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, users)
}

type getUserRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type listUsersRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}
