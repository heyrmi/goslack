package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"

	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/util"
)

type TwoFactorService struct {
	store db.Store
}

type Setup2FARequest struct {
	UserID int64 `json:"user_id"`
}

type Setup2FAResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	QRCodeData  string   `json:"qr_code_data"` // Base64 encoded PNG
	BackupCodes []string `json:"backup_codes"`
}

type Verify2FARequest struct {
	UserID int64  `json:"user_id"`
	Code   string `json:"code" binding:"required"`
}

type Disable2FARequest struct {
	UserID   int64  `json:"user_id"`
	Password string `json:"password" binding:"required"`
}

type Validate2FARequest struct {
	UserID int64  `json:"user_id"`
	Code   string `json:"code" binding:"required"`
}

func NewTwoFactorService(store db.Store) *TwoFactorService {
	return &TwoFactorService{
		store: store,
	}
}

// Setup2FA initializes 2FA for a user
func (s *TwoFactorService) Setup2FA(ctx context.Context, req Setup2FARequest) (*Setup2FAResponse, error) {
	// Get user information
	user, err := s.store.GetUser(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if 2FA is already enabled
	existing2FA, err := s.store.GetUser2FA(ctx, req.UserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing 2FA: %w", err)
	}
	if existing2FA.Enabled {
		return nil, fmt.Errorf("2FA is already enabled for this user")
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoSlack",
		AccountName: user.Email,
		SecretSize:  32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.generateBackupCodes(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Store 2FA configuration (not enabled yet)
	_, err = s.store.CreateUser2FA(ctx, db.CreateUser2FAParams{
		UserID:      req.UserID,
		Secret:      key.Secret(),
		BackupCodes: backupCodes,
	})
	if err != nil {
		// If record exists, update it
		if strings.Contains(err.Error(), "duplicate") {
			err = s.store.UpdateUser2FABackupCodes(ctx, db.UpdateUser2FABackupCodesParams{
				UserID:      req.UserID,
				BackupCodes: backupCodes,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update 2FA setup: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to store 2FA setup: %w", err)
		}
	}

	// Generate QR code
	qrCodeData, err := s.generateQRCode(key.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	return &Setup2FAResponse{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		QRCodeData:  qrCodeData,
		BackupCodes: backupCodes,
	}, nil
}

// Verify2FA verifies the TOTP code and enables 2FA
func (s *TwoFactorService) Verify2FA(ctx context.Context, req Verify2FARequest) error {
	// Get 2FA configuration
	user2FA, err := s.store.GetUser2FA(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("2FA not set up for this user")
	}

	if user2FA.Enabled {
		return fmt.Errorf("2FA is already enabled")
	}

	// Validate TOTP code
	valid := totp.Validate(req.Code, user2FA.Secret)

	if !valid {
		return fmt.Errorf("invalid 2FA code")
	}

	// Enable 2FA
	err = s.store.EnableUser2FA(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	return nil
}

// Validate2FA validates a 2FA code for an already enabled user
func (s *TwoFactorService) Validate2FA(ctx context.Context, req Validate2FARequest) (bool, error) {
	// Get 2FA configuration
	user2FA, err := s.store.GetUser2FA(ctx, req.UserID)
	if err != nil {
		return false, fmt.Errorf("2FA not enabled for this user")
	}

	if !user2FA.Enabled {
		return false, fmt.Errorf("2FA is not enabled for this user")
	}

	// Check if it's a backup code
	if s.isBackupCode(req.Code, user2FA.BackupCodes) {
		// Remove used backup code
		newBackupCodes := s.removeBackupCode(req.Code, user2FA.BackupCodes)
		err = s.store.UpdateUser2FABackupCodes(ctx, db.UpdateUser2FABackupCodesParams{
			UserID:      req.UserID,
			BackupCodes: newBackupCodes,
		})
		if err != nil {
			return false, fmt.Errorf("failed to update backup codes: %w", err)
		}
		return true, nil
	}

	// Validate TOTP code
	valid := totp.Validate(req.Code, user2FA.Secret)

	return valid, nil
}

// Disable2FA disables 2FA for a user
func (s *TwoFactorService) Disable2FA(ctx context.Context, req Disable2FARequest) error {
	// Get user to verify password
	user, err := s.store.GetUser(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		return fmt.Errorf("invalid password")
	}

	// Check if 2FA is enabled
	user2FA, err := s.store.GetUser2FA(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("2FA not enabled for this user")
	}

	if !user2FA.Enabled {
		return fmt.Errorf("2FA is not enabled for this user")
	}

	// Disable 2FA
	err = s.store.DisableUser2FA(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	return nil
}

// Is2FAEnabled checks if 2FA is enabled for a user
func (s *TwoFactorService) Is2FAEnabled(ctx context.Context, userID int64) (bool, error) {
	user2FA, err := s.store.GetUser2FA(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return user2FA.Enabled, nil
}

// RegenerateBackupCodes generates new backup codes for a user
func (s *TwoFactorService) RegenerateBackupCodes(ctx context.Context, userID int64) ([]string, error) {
	// Check if 2FA is enabled
	user2FA, err := s.store.GetUser2FA(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("2FA not enabled for this user")
	}

	if !user2FA.Enabled {
		return nil, fmt.Errorf("2FA is not enabled for this user")
	}

	// Generate new backup codes
	backupCodes, err := s.generateBackupCodes(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Update backup codes
	err = s.store.UpdateUser2FABackupCodes(ctx, db.UpdateUser2FABackupCodesParams{
		UserID:      userID,
		BackupCodes: backupCodes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update backup codes: %w", err)
	}

	return backupCodes, nil
}

// generateBackupCodes generates random backup codes
func (s *TwoFactorService) generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate 8-character alphanumeric code
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}

		// Convert to base32 and take first 8 characters
		code := base64.StdEncoding.EncodeToString(bytes)[:8]
		codes[i] = strings.ToUpper(code)
	}
	return codes, nil
}

// generateQRCode generates a QR code as base64 encoded PNG
func (s *TwoFactorService) generateQRCode(url string) (string, error) {
	qrCode, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(qrCode), nil
}

// isBackupCode checks if the provided code is a valid backup code
func (s *TwoFactorService) isBackupCode(code string, backupCodes []string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, backupCode := range backupCodes {
		if code == backupCode {
			return true
		}
	}
	return false
}

// removeBackupCode removes a used backup code from the list
func (s *TwoFactorService) removeBackupCode(usedCode string, backupCodes []string) []string {
	usedCode = strings.ToUpper(strings.TrimSpace(usedCode))
	var newCodes []string
	for _, code := range backupCodes {
		if code != usedCode {
			newCodes = append(newCodes, code)
		}
	}
	return newCodes
}
