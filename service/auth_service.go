package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
	"github.com/sqlc-dev/pqtype"
)

const (
	MaxFailedAttempts    = 5
	LockoutDuration      = 30 * time.Minute
	EmailVerificationExp = 24 * time.Hour
	PasswordResetExp     = 1 * time.Hour
)

type AuthService struct {
	store        db.Store
	tokenMaker   token.Maker
	emailService EmailServiceInterface
	config       util.Config
}

type EmailVerificationRequest struct {
	Email     string `json:"email" binding:"required,email"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type PasswordResetRequest struct {
	Email     string `json:"email" binding:"required,email"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	IPAddress       string `json:"-"`
	UserAgent       string `json:"-"`
}

type VerifyEmailRequest struct {
	Token     string `json:"token" binding:"required"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type AuthResponse struct {
	User         *db.User  `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SecurityEventRequest struct {
	UserID      int64                  `json:"user_id"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	IPAddress   net.IP                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func NewAuthService(store db.Store, tokenMaker token.Maker, emailService EmailServiceInterface, config util.Config) *AuthService {
	return &AuthService{
		store:        store,
		tokenMaker:   tokenMaker,
		emailService: emailService,
		config:       config,
	}
}

// GenerateSecureToken generates a cryptographically secure random token
func (s *AuthService) GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendEmailVerification sends an email verification token
func (s *AuthService) SendEmailVerification(ctx context.Context, req EmailVerificationRequest) error {
	// Get user by email
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if email exists or not for security
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if already verified
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Generate verification token
	token, err := s.GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Store verification token
	var ipAddr pqtype.Inet
	if req.IPAddress != "" {
		ip := net.ParseIP(req.IPAddress)
		if ip != nil {
			ipAddr = pqtype.Inet{IPNet: net.IPNet{IP: ip}, Valid: true}
		}
	}

	_, err = s.store.CreateEmailVerificationToken(ctx, db.CreateEmailVerificationTokenParams{
		UserID:    user.ID,
		Token:     token,
		Email:     req.Email,
		TokenType: "email_verification",
		ExpiresAt: time.Now().Add(EmailVerificationExp),
		IpAddress: ipAddr,
		UserAgent: sql.NullString{String: req.UserAgent, Valid: req.UserAgent != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	// Send verification email
	err = s.emailService.SendEmailVerification(req.Email, token, user.FirstName)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, SecurityEventRequest{
		UserID:      user.ID,
		EventType:   "email_verification_sent",
		Description: "Email verification token sent",
		IPAddress:   net.ParseIP(req.IPAddress),
		UserAgent:   req.UserAgent,
	})

	return nil
}

// VerifyEmail verifies an email using the provided token
func (s *AuthService) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	// Get and validate token
	tokenData, err := s.store.GetEmailVerificationToken(ctx, req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid or expired verification token")
		}
		return fmt.Errorf("failed to get verification token: %w", err)
	}

	// Mark token as used
	err = s.store.UseEmailVerificationToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("failed to use verification token: %w", err)
	}

	// Verify user's email
	err = s.store.VerifyUserEmail(ctx, tokenData.UserID)
	if err != nil {
		return fmt.Errorf("failed to verify user email: %w", err)
	}

	// Get user for logging and welcome email
	user, err := s.store.GetUser(ctx, tokenData.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Send welcome email
	err = s.emailService.SendWelcomeEmail(user.Email, user.FirstName)
	if err != nil {
		// Log error but don't fail the verification
		fmt.Printf("Failed to send welcome email: %v\n", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, SecurityEventRequest{
		UserID:      tokenData.UserID,
		EventType:   "email_verified",
		Description: "Email address successfully verified",
		IPAddress:   net.ParseIP(req.IPAddress),
		UserAgent:   req.UserAgent,
	})

	return nil
}

// SendPasswordReset sends a password reset token
func (s *AuthService) SendPasswordReset(ctx context.Context, req PasswordResetRequest) error {
	// Get user by email
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if email exists or not for security
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if account is locked
	locked, err := s.IsAccountLocked(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check account lock status: %w", err)
	}
	if locked {
		return fmt.Errorf("account is locked due to security reasons")
	}

	// Generate reset token
	token, err := s.GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Delete any existing reset tokens for this user
	err = s.store.DeleteUserPasswordResetTokens(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing tokens: %w", err)
	}

	// Store reset token
	var ipAddr pqtype.Inet
	if req.IPAddress != "" {
		ip := net.ParseIP(req.IPAddress)
		if ip != nil {
			ipAddr = pqtype.Inet{IPNet: net.IPNet{IP: ip}, Valid: true}
		}
	}

	_, err = s.store.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(PasswordResetExp),
		IpAddress: ipAddr,
		UserAgent: sql.NullString{String: req.UserAgent, Valid: req.UserAgent != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	// Send reset email
	err = s.emailService.SendPasswordReset(req.Email, token, user.FirstName)
	if err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, SecurityEventRequest{
		UserID:      user.ID,
		EventType:   "password_reset_requested",
		Description: "Password reset token requested",
		IPAddress:   net.ParseIP(req.IPAddress),
		UserAgent:   req.UserAgent,
	})

	return nil
}

// ResetPassword resets a user's password using the provided token
func (s *AuthService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Get and validate token
	tokenData, err := s.store.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid or expired reset token")
		}
		return fmt.Errorf("failed to get reset token: %w", err)
	}

	// Hash new password
	hashedPassword, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user's password
	_, err = s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:             tokenData.UserID,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	err = s.store.UsePasswordResetToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("failed to use reset token: %w", err)
	}

	// Reset failed login attempts
	err = s.store.ResetFailedAttempts(ctx, tokenData.UserID)
	if err != nil {
		// Log error but don't fail the reset
		fmt.Printf("Failed to reset failed attempts: %v\n", err)
	}

	// Get user for notification
	user, err := s.store.GetUser(ctx, tokenData.UserID)
	if err == nil {
		// Send security alert
		err = s.emailService.SendSecurityAlert(
			user.Email,
			user.FirstName,
			"password_changed",
			"Your password has been successfully changed.",
		)
		if err != nil {
			// Log error but don't fail the reset
			fmt.Printf("Failed to send security alert: %v\n", err)
		}
	}

	// Log security event
	s.logSecurityEvent(ctx, SecurityEventRequest{
		UserID:      tokenData.UserID,
		EventType:   "password_reset_completed",
		Description: "Password successfully reset using token",
		IPAddress:   net.ParseIP(req.IPAddress),
		UserAgent:   req.UserAgent,
	})

	return nil
}

// CheckAccountLockout checks if account should be locked after failed login
func (s *AuthService) CheckAccountLockout(ctx context.Context, userID int64, ipAddress, userAgent string) error {
	// Get current lockout status
	lockout, err := s.store.GetAccountLockout(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new lockout record
			_, err = s.store.CreateAccountLockout(ctx, userID)
			return err
		}
		return err
	}

	// Increment failed attempts
	lockout, err = s.store.IncrementFailedAttempts(ctx, userID)
	if err != nil {
		return err
	}

	// Lock account if max attempts reached
	if lockout.FailedAttempts >= MaxFailedAttempts {
		lockUntil := time.Now().Add(LockoutDuration)
		err = s.store.LockAccount(ctx, db.LockAccountParams{
			UserID:      userID,
			LockedUntil: sql.NullTime{Time: lockUntil, Valid: true},
		})
		if err != nil {
			return err
		}

		// Send security alert
		user, err := s.store.GetUser(ctx, userID)
		if err == nil {
			s.emailService.SendSecurityAlert(
				user.Email,
				user.FirstName,
				"account_locked",
				fmt.Sprintf("Account locked due to %d failed login attempts. It will be unlocked automatically after %d minutes.", MaxFailedAttempts, int(LockoutDuration.Minutes())),
			)
		}

		// Log security event
		s.logSecurityEvent(ctx, SecurityEventRequest{
			UserID:      userID,
			EventType:   "account_locked",
			Description: fmt.Sprintf("Account locked after %d failed login attempts", MaxFailedAttempts),
			IPAddress:   net.ParseIP(ipAddress),
			UserAgent:   userAgent,
		})
	}

	return nil
}

// IsAccountLocked checks if an account is currently locked
func (s *AuthService) IsAccountLocked(ctx context.Context, userID int64) (bool, error) {
	result, err := s.store.IsAccountLocked(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result.Bool, nil
}

// UnlockAccount manually unlocks an account (admin function)
func (s *AuthService) UnlockAccount(ctx context.Context, userID int64, adminID int64, ipAddress, userAgent string) error {
	err := s.store.UnlockAccount(ctx, userID)
	if err != nil {
		return err
	}

	// Log security event
	s.logSecurityEvent(ctx, SecurityEventRequest{
		UserID:      userID,
		EventType:   "account_unlocked",
		Description: fmt.Sprintf("Account manually unlocked by admin %d", adminID),
		IPAddress:   net.ParseIP(ipAddress),
		UserAgent:   userAgent,
		Metadata:    map[string]interface{}{"admin_id": adminID},
	})

	return nil
}

// ResetFailedAttempts resets failed login attempts after successful login
func (s *AuthService) ResetFailedAttempts(ctx context.Context, userID int64) error {
	return s.store.ResetFailedAttempts(ctx, userID)
}

// logSecurityEvent logs a security event
func (s *AuthService) logSecurityEvent(ctx context.Context, req SecurityEventRequest) {
	var metadata pqtype.NullRawMessage
	if req.Metadata != nil {
		// Convert metadata to JSON bytes (simplified)
		metadata = pqtype.NullRawMessage{RawMessage: []byte("{}"), Valid: true}
	}

	var ipAddr pqtype.Inet
	if req.IPAddress != nil {
		ipAddr = pqtype.Inet{IPNet: net.IPNet{IP: req.IPAddress}, Valid: true}
	}

	_, err := s.store.CreateSecurityEvent(ctx, db.CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: req.UserID, Valid: req.UserID > 0},
		EventType:   req.EventType,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		IpAddress:   ipAddr,
		UserAgent:   sql.NullString{String: req.UserAgent, Valid: req.UserAgent != ""},
		Metadata:    metadata,
	})
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log security event: %v\n", err)
	}
}

// GetUserSecurityEvents gets security events for a user
func (s *AuthService) GetUserSecurityEvents(ctx context.Context, userID int64, limit, offset int32) ([]db.SecurityEvent, error) {
	return s.store.GetUserSecurityEvents(ctx, db.GetUserSecurityEventsParams{
		UserID: sql.NullInt64{Int64: userID, Valid: true},
		Limit:  limit,
		Offset: offset,
	})
}

// CleanupExpiredTokens cleans up expired tokens (should be run periodically)
func (s *AuthService) CleanupExpiredTokens(ctx context.Context) error {
	// Clean up expired email verification tokens
	err := s.store.DeleteExpiredEmailVerificationTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup email verification tokens: %w", err)
	}

	// Clean up expired password reset tokens
	err = s.store.DeleteExpiredPasswordResetTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup password reset tokens: %w", err)
	}

	// Unlock expired accounts
	err = s.store.UnlockExpiredAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlock expired accounts: %w", err)
	}

	return nil
}
