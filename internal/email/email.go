package email

import (
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
	from   string
}

func NewEmailService() *EmailService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return nil
	}

	from := os.Getenv("FROM_EMAIL")
	if from == "" {
		from = "noreply@your-domain.com"
	}

	client := resend.NewClient(apiKey)
	return &EmailService{
		client: client,
		from:   from,
	}
}

func (e *EmailService) SendPasswordResetEmail(to, resetToken, baseURL string) error {
	if e.client == nil {
		return fmt.Errorf("email service not configured - missing RESEND_API_KEY")
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, resetToken)
	
	subject := "Password Reset Request"
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2c3e50;">Password Reset Request</h2>
        
        <p>You have requested to reset your password. Click the link below to create a new password:</p>
        
        <div style="margin: 30px 0;">
            <a href="%s" style="background-color: #3498db; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
        </div>
        
        <p>If the button doesn't work, you can copy and paste the following link into your browser:</p>
        <p style="word-break: break-all; background-color: #f8f9fa; padding: 10px; border-radius: 3px; font-family: monospace;">%s</p>
        
        <p><strong>This link will expire in 1 hour for security reasons.</strong></p>
        
        <p>If you did not request this password reset, you can safely ignore this email.</p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="font-size: 12px; color: #666;">This is an automated message, please do not reply.</p>
    </div>
</body>
</html>`, resetURL, resetURL)

	textContent := fmt.Sprintf(`Password Reset Request

You have requested to reset your password. Visit the following link to create a new password:

%s

This link will expire in 1 hour for security reasons.

If you did not request this password reset, you can safely ignore this email.

This is an automated message, please do not reply.`, resetURL)

	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      []string{to},
		Subject: subject,
		Html:    htmlContent,
		Text:    textContent,
	}

	_, err := e.client.Emails.Send(params)
	return err
}