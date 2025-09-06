package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
	"github.com/heyrmi/goslack/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
	currentUserKey          = "current_user"
)

// AuthMiddleware creates a gin middleware for authorization
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	})
}

// authWithUserMiddleware extends authMiddleware to also load the current user
func authWithUserMiddleware(tokenMaker token.Maker, userService *service.UserService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Perform basic authentication
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)

		// Get the payload and load the user
		user, err := userService.GetUserByEmail(ctx, payload.Username)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(currentUserKey, user)
		ctx.Next()
	})
}

// requireWorkspaceMember middleware ensures the user is a member of the specified workspace
func requireWorkspaceMember(userService *service.UserService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Get workspace ID from URL parameter
		workspaceIDStr := ctx.Param("id")
		workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
		if err != nil {
			err := errors.New("invalid workspace ID")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		// Get current user
		currentUser, exists := ctx.Get(currentUserKey)
		if !exists {
			err := errors.New("user not found in context")
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		user := currentUser.(service.UserResponse)

		// Check if user is a member of the workspace
		isMember, err := userService.IsWorkspaceMember(ctx, user.ID, workspaceID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		if !isMember {
			err := errors.New("access denied: user is not a member of this workspace")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err))
			return
		}

		ctx.Next()
	})
}

// requireWorkspaceAdmin middleware ensures the user is an admin of the specified workspace
func requireWorkspaceAdmin(userService *service.UserService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Get workspace ID from URL parameter
		workspaceIDStr := ctx.Param("id")
		workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
		if err != nil {
			err := errors.New("invalid workspace ID")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		// Get current user
		currentUser, exists := ctx.Get(currentUserKey)
		if !exists {
			err := errors.New("user not found in context")
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		user := currentUser.(service.UserResponse)

		// Check if user is an admin of the workspace
		isAdmin, err := userService.IsWorkspaceAdmin(ctx, user.ID, workspaceID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		if !isAdmin {
			err := errors.New("access denied: user is not an admin of this workspace")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err))
			return
		}

		ctx.Next()
	})
}

// requireSameWorkspaceForUserRole middleware ensures admin can only modify users in the same workspace
func requireSameWorkspaceForUserRole(userService *service.UserService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Get target user ID from URL parameter
		targetUserIDStr := ctx.Param("user_id")
		targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
		if err != nil {
			err := errors.New("invalid user ID")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		// Get current user
		currentUser, exists := ctx.Get(currentUserKey)
		if !exists {
			err := errors.New("user not found in context")
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		user := currentUser.(service.UserResponse)

		// Get target user to check their workspace
		targetUser, err := userService.GetUser(ctx, targetUserID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusNotFound, errorResponse(err))
			return
		}

		// Check if both users are in the same workspace
		if user.WorkspaceID == nil || targetUser.WorkspaceID == nil {
			err := errors.New("users must be in a workspace")
			ctx.AbortWithStatusJSON(http.StatusBadRequest, errorResponse(err))
			return
		}

		if *user.WorkspaceID != *targetUser.WorkspaceID {
			err := errors.New("access denied: users are not in the same workspace")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err))
			return
		}

		// Check if current user is admin of the workspace
		isAdmin, err := userService.IsWorkspaceAdmin(ctx, user.ID, *user.WorkspaceID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		if !isAdmin {
			err := errors.New("access denied: user is not an admin")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err))
			return
		}

		ctx.Next()
	})
}
