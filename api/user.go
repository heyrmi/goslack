package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
	"github.com/lib/pq"
)

// @Summary Create User
// @Description Register a new user in an organization
// @Tags users
// @Accept json
// @Produce json
// @Param user body service.CreateUserRequest true "User registration details"
// @Success 200 {object} service.UserResponse "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 403 {object} map[string]string "Email already exists or invalid organization"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users [post]
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

// @Summary User Login
// @Description Authenticate a user and receive access token
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body service.LoginUserRequest true "User login credentials"
// @Success 200 {object} service.LoginUserResponse "Login successful with access token"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Router /users/login [post]
func (server *Server) loginUser(ctx *gin.Context) {
	var req service.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Add IP address and user agent for security tracking
	req.IPAddress = getClientIP(ctx)
	req.UserAgent = ctx.GetHeader("User-Agent")

	user, err := server.userService.LoginUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// @Summary Get User
// @Description Retrieve user information by ID (requires authentication)
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} service.UserResponse "User information"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Access denied - different organization"
// @Failure 404 {object} map[string]string "User not found"
// @Router /users/{id} [get]
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

// @Summary Update User Profile
// @Description Update user profile information (users can only update their own profile)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param profile body service.UpdateUserProfileRequest true "Profile update details"
// @Success 200 {object} service.UserResponse "Profile updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Can only update own profile"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{id}/profile [put]
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

// @Summary Change Password
// @Description Change user password (users can only change their own password)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param passwords body service.ChangePasswordRequest true "Password change details"
// @Success 200 {object} map[string]string "Password changed successfully"
// @Failure 400 {object} map[string]string "Invalid request or wrong old password"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Can only change own password"
// @Failure 404 {object} map[string]string "User not found"
// @Router /users/{id}/password [put]
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

// @Summary List Users
// @Description List users in the authenticated user's organization
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param page_id query int false "Page ID (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 10, max: 10)" minimum(5) maximum(10)
// @Success 200 {array} service.UserResponse "List of users"
// @Failure 400 {object} map[string]string "Invalid pagination parameters"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users [get]
func (server *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
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
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=5,max=10"`
}
