package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	BaseURL      string
}

// EmailServiceInterface defines the interface for email operations
type EmailServiceInterface interface {
	SendEmailVerification(email, token, firstName string) error
	SendPasswordReset(email, token, firstName string) error
	SendWelcomeEmail(email, firstName string) error
	SendSecurityAlert(email, firstName, eventType, description string) error
	GenerateSecureToken() (string, error)
}

type EmailService struct {
	config EmailConfig
}

type EmailTemplate struct {
	Subject string
	Body    string
}

func NewEmailService(config EmailConfig) *EmailService {
	return &EmailService{
		config: config,
	}
}

// GenerateSecureToken generates a cryptographically secure random token
func (s *EmailService) GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendEmail sends an email using SMTP
func (s *EmailService) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.config.SMTPHost, s.config.SMTPPort, s.config.SMTPUsername, s.config.SMTPPassword)

	return d.DialAndSend(m)
}

// SendEmailVerification sends an email verification link
func (s *EmailService) SendEmailVerification(email, token, firstName string) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.config.BaseURL, token)

	template := s.getEmailVerificationTemplate(firstName, verificationURL)
	return s.SendEmail(email, template.Subject, template.Body)
}

// SendPasswordReset sends a password reset link
func (s *EmailService) SendPasswordReset(email, token, firstName string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, token)

	template := s.getPasswordResetTemplate(firstName, resetURL)
	return s.SendEmail(email, template.Subject, template.Body)
}

// SendSecurityAlert sends a security alert notification
func (s *EmailService) SendSecurityAlert(email, firstName, eventType, description string) error {
	template := s.getSecurityAlertTemplate(firstName, eventType, description)
	return s.SendEmail(email, template.Subject, template.Body)
}

// SendWelcomeEmail sends a welcome email after successful registration
func (s *EmailService) SendWelcomeEmail(email, firstName string) error {
	template := s.getWelcomeTemplate(firstName)
	return s.SendEmail(email, template.Subject, template.Body)
}

// Email Templates

func (s *EmailService) getEmailVerificationTemplate(firstName, verificationURL string) EmailTemplate {
	subject := "Verify your GoSlack account"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verify your GoSlack account</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4A154B; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; background: #007A5A; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Welcome to GoSlack!</h1>
    </div>
    <div class="content">
        <h2>Hi %s,</h2>
        <p>Thank you for signing up for GoSlack! To complete your registration and start collaborating with your team, please verify your email address.</p>
        
        <p><a href="%s" class="button">Verify Email Address</a></p>
        
        <p>If the button doesn't work, you can also copy and paste this link into your browser:</p>
        <p><a href="%s">%s</a></p>
        
        <p>This verification link will expire in 24 hours for security reasons.</p>
        
        <p>If you didn't create a GoSlack account, you can safely ignore this email.</p>
        
        <div class="footer">
            <p>Best regards,<br>The GoSlack Team</p>
            <p><em>This is an automated message, please do not reply to this email.</em></p>
        </div>
    </div>
</body>
</html>`, firstName, verificationURL, verificationURL, verificationURL)

	return EmailTemplate{Subject: subject, Body: body}
}

func (s *EmailService) getPasswordResetTemplate(firstName, resetURL string) EmailTemplate {
	subject := "Reset your GoSlack password"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset your GoSlack password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4A154B; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; background: #007A5A; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 14px; color: #666; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Password Reset Request</h1>
    </div>
    <div class="content">
        <h2>Hi %s,</h2>
        <p>We received a request to reset your GoSlack password. If you made this request, click the button below to set a new password:</p>
        
        <p><a href="%s" class="button">Reset Password</a></p>
        
        <p>If the button doesn't work, you can also copy and paste this link into your browser:</p>
        <p><a href="%s">%s</a></p>
        
        <div class="warning">
            <strong>⚠️ Security Notice:</strong> This password reset link will expire in 1 hour for security reasons. If you didn't request a password reset, please ignore this email and your password will remain unchanged.
        </div>
        
        <p>If you're having trouble with your account, please contact our support team.</p>
        
        <div class="footer">
            <p>Best regards,<br>The GoSlack Team</p>
            <p><em>This is an automated message, please do not reply to this email.</em></p>
        </div>
    </div>
</body>
</html>`, firstName, resetURL, resetURL, resetURL)

	return EmailTemplate{Subject: subject, Body: body}
}

func (s *EmailService) getSecurityAlertTemplate(firstName, eventType, description string) EmailTemplate {
	subject := fmt.Sprintf("GoSlack Security Alert - %s", strings.Title(strings.ReplaceAll(eventType, "_", " ")))
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoSlack Security Alert</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #d73027; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .alert { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔒 Security Alert</h1>
    </div>
    <div class="content">
        <h2>Hi %s,</h2>
        <p>We're writing to inform you of important security activity on your GoSlack account.</p>
        
        <div class="alert">
            <strong>Event:</strong> %s<br>
            <strong>Details:</strong> %s<br>
            <strong>Time:</strong> %s
        </div>
        
        <p><strong>What should you do?</strong></p>
        <ul>
            <li>If this was you, no action is needed.</li>
            <li>If this wasn't you, please secure your account immediately by changing your password and enabling two-factor authentication.</li>
            <li>Review your recent account activity and revoke any suspicious sessions.</li>
        </ul>
        
        <p>If you have any concerns about your account security, please contact our support team immediately.</p>
        
        <div class="footer">
            <p>Best regards,<br>The GoSlack Security Team</p>
            <p><em>This is an automated security notification. Please do not reply to this email.</em></p>
        </div>
    </div>
</body>
</html>`, firstName, strings.Title(strings.ReplaceAll(eventType, "_", " ")), description, time.Now().Format("January 2, 2006 at 3:04 PM MST"))

	return EmailTemplate{Subject: subject, Body: body}
}

func (s *EmailService) getWelcomeTemplate(firstName string) EmailTemplate {
	subject := "Welcome to GoSlack - Let's get started!"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to GoSlack</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4A154B; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .feature { background: white; padding: 15px; margin: 15px 0; border-left: 4px solid #007A5A; border-radius: 4px; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🎉 Welcome to GoSlack!</h1>
    </div>
    <div class="content">
        <h2>Hi %s,</h2>
        <p>Your email has been successfully verified! You're now ready to start collaborating with your team on GoSlack.</p>
        
        <h3>Here's what you can do next:</h3>
        
        <div class="feature">
            <strong>💬 Start messaging</strong><br>
            Send direct messages to team members or join channels to participate in group conversations.
        </div>
        
        <div class="feature">
            <strong>📁 Share files</strong><br>
            Upload and share documents, images, and other files with your team members.
        </div>
        
        <div class="feature">
            <strong>🏢 Create workspaces</strong><br>
            Organize your team into different workspaces for better collaboration.
        </div>
        
        <div class="feature">
            <strong>🔒 Secure your account</strong><br>
            Enable two-factor authentication for enhanced security.
        </div>
        
        <p>If you have any questions or need help getting started, don't hesitate to reach out to our support team.</p>
        
        <div class="footer">
            <p>Happy collaborating!<br>The GoSlack Team</p>
            <p><em>This is an automated welcome message.</em></p>
        </div>
    </div>
</body>
</html>`, firstName)

	return EmailTemplate{Subject: subject, Body: body}
}
