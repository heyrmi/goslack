package service

import (
	"crypto/rand"
	"encoding/hex"
)

// MockEmailService is a mock implementation of EmailServiceInterface for testing
type MockEmailService struct {
	SendEmailVerificationFunc func(email, token, firstName string) error
	SendPasswordResetFunc     func(email, token, firstName string) error
	SendWelcomeEmailFunc      func(email, firstName string) error
	SendSecurityAlertFunc     func(email, firstName, eventType, description string) error
	GenerateSecureTokenFunc   func() (string, error)
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

// SendEmailVerification mocks the SendEmailVerification method
func (m *MockEmailService) SendEmailVerification(email, token, firstName string) error {
	if m.SendEmailVerificationFunc != nil {
		return m.SendEmailVerificationFunc(email, token, firstName)
	}
	// Default behavior - no error
	return nil
}

// SendPasswordReset mocks the SendPasswordReset method
func (m *MockEmailService) SendPasswordReset(email, token, firstName string) error {
	if m.SendPasswordResetFunc != nil {
		return m.SendPasswordResetFunc(email, token, firstName)
	}
	// Default behavior - no error
	return nil
}

// SendWelcomeEmail mocks the SendWelcomeEmail method
func (m *MockEmailService) SendWelcomeEmail(email, firstName string) error {
	if m.SendWelcomeEmailFunc != nil {
		return m.SendWelcomeEmailFunc(email, firstName)
	}
	// Default behavior - no error
	return nil
}

// SendSecurityAlert mocks the SendSecurityAlert method
func (m *MockEmailService) SendSecurityAlert(email, firstName, eventType, description string) error {
	if m.SendSecurityAlertFunc != nil {
		return m.SendSecurityAlertFunc(email, firstName, eventType, description)
	}
	// Default behavior - no error
	return nil
}

// GenerateSecureToken mocks the GenerateSecureToken method
func (m *MockEmailService) GenerateSecureToken() (string, error) {
	if m.GenerateSecureTokenFunc != nil {
		return m.GenerateSecureTokenFunc()
	}
	// Default behavior - generate a real token
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SetSendEmailVerificationError sets the mock to return an error for SendEmailVerification
func (m *MockEmailService) SetSendEmailVerificationError(err error) {
	m.SendEmailVerificationFunc = func(email, token, firstName string) error {
		return err
	}
}

// SetSendPasswordResetError sets the mock to return an error for SendPasswordReset
func (m *MockEmailService) SetSendPasswordResetError(err error) {
	m.SendPasswordResetFunc = func(email, token, firstName string) error {
		return err
	}
}

// SetSendWelcomeEmailError sets the mock to return an error for SendWelcomeEmail
func (m *MockEmailService) SetSendWelcomeEmailError(err error) {
	m.SendWelcomeEmailFunc = func(email, firstName string) error {
		return err
	}
}

// SetSendSecurityAlertError sets the mock to return an error for SendSecurityAlert
func (m *MockEmailService) SetSendSecurityAlertError(err error) {
	m.SendSecurityAlertFunc = func(email, firstName, eventType, description string) error {
		return err
	}
}

// SetGenerateSecureTokenError sets the mock to return an error for GenerateSecureToken
func (m *MockEmailService) SetGenerateSecureTokenError(err error) {
	m.GenerateSecureTokenFunc = func() (string, error) {
		return "", err
	}
}

// Verify that MockEmailService implements EmailServiceInterface
var _ EmailServiceInterface = (*MockEmailService)(nil)
