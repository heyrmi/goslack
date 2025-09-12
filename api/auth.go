package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
)

// @Summary Send Email Verification
// @Description Send email verification link to user's email address
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body service.EmailVerificationRequest true "Email verification request"
// @Success 200 {object} map[string]string "Verification email sent"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/send-verification [post]
func (server *Server) sendEmailVerification(ctx *gin.Context) {
	var req service.EmailVerificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Add IP and User Agent
	req.IPAddress = getClientIP(ctx)
	req.UserAgent = ctx.GetHeader("User-Agent")

	err := server.authService.SendEmailVerification(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent successfully",
	})
}

// @Summary Verify Email
// @Description Verify user's email address using verification token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body service.VerifyEmailRequest true "Email verification token"
// @Success 200 {object} map[string]string "Email verified successfully"
// @Failure 400 {object} map[string]string "Invalid request or token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/verify-email [post]
func (server *Server) verifyEmail(ctx *gin.Context) {
	var req service.VerifyEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Add IP and User Agent
	req.IPAddress = getClientIP(ctx)
	req.UserAgent = ctx.GetHeader("User-Agent")

	err := server.authService.VerifyEmail(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully",
	})
}

// @Summary Request Password Reset
// @Description Send password reset link to user's email address
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body service.PasswordResetRequest true "Password reset request"
// @Success 200 {object} map[string]string "Password reset email sent"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/forgot-password [post]
func (server *Server) requestPasswordReset(ctx *gin.Context) {
	var req service.PasswordResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Add IP and User Agent
	req.IPAddress = getClientIP(ctx)
	req.UserAgent = ctx.GetHeader("User-Agent")

	err := server.authService.SendPasswordReset(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Password reset email sent successfully",
	})
}

// @Summary Reset Password
// @Description Reset user's password using reset token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body service.ResetPasswordRequest true "Password reset with token"
// @Success 200 {object} map[string]string "Password reset successfully"
// @Failure 400 {object} map[string]string "Invalid request or token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/reset-password [post]
func (server *Server) resetPassword(ctx *gin.Context) {
	var req service.ResetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Add IP and User Agent
	req.IPAddress = getClientIP(ctx)
	req.UserAgent = ctx.GetHeader("User-Agent")

	err := server.authService.ResetPassword(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// @Summary Setup 2FA
// @Description Setup two-factor authentication for the user
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} service.Setup2FAResponse "2FA setup information"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/2fa/setup [post]
func (server *Server) setup2FA(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	req := service.Setup2FARequest{
		UserID: user.ID,
	}

	response, err := server.twoFactorService.Setup2FA(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// @Summary Verify 2FA Setup
// @Description Verify 2FA code and enable two-factor authentication
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.Verify2FARequest true "2FA verification code"
// @Success 200 {object} map[string]string "2FA enabled successfully"
// @Failure 400 {object} map[string]string "Invalid code"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/2fa/verify [post]
func (server *Server) verify2FA(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.Verify2FARequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserID = user.ID

	err := server.twoFactorService.Verify2FA(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Two-factor authentication enabled successfully",
	})
}

// @Summary Disable 2FA
// @Description Disable two-factor authentication for the user
// @Tags authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.Disable2FARequest true "Password confirmation to disable 2FA"
// @Success 200 {object} map[string]string "2FA disabled successfully"
// @Failure 400 {object} map[string]string "Invalid password"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/2fa/disable [post]
func (server *Server) disable2FA(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.Disable2FARequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserID = user.ID

	err := server.twoFactorService.Disable2FA(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Two-factor authentication disabled successfully",
	})
}

// @Summary Regenerate Backup Codes
// @Description Generate new backup codes for 2FA
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "New backup codes"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/2fa/backup-codes [post]
func (server *Server) regenerateBackupCodes(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	backupCodes, err := server.twoFactorService.RegenerateBackupCodes(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"backup_codes": backupCodes,
		"message":      "New backup codes generated successfully",
	})
}

// @Summary Get Security Events
// @Description Get security events for the current user
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of events to return" default(20)
// @Param offset query int false "Number of events to skip" default(0)
// @Success 200 {object} map[string]interface{} "Security events"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/security-events [get]
func (server *Server) getSecurityEvents(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	limit := int32(20)
	offset := int32(0)

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = int32(o)
		}
	}

	events, err := server.authService.GetUserSecurityEvents(ctx, user.ID, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
		"limit":  limit,
		"offset": offset,
	})
}

// @Summary Get 2FA Status
// @Description Check if 2FA is enabled for the current user
// @Tags authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "2FA status"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/2fa/status [get]
func (server *Server) get2FAStatus(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	enabled, err := server.twoFactorService.Is2FAEnabled(ctx, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
	})
}

// Helper function to get current user from context
func getCurrentUser(ctx *gin.Context) *db.User {
	user, exists := ctx.Get(currentUserKey)
	if !exists {
		return nil
	}
	return user.(*db.User)
}
