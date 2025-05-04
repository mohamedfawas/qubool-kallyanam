package email

import (
	"errors"
	"fmt"
	"net/smtp"
)

var (
	ErrInvalidConfig = errors.New("invalid email configuration")
	ErrSendFailed    = errors.New("failed to send email")
)

// Config holds the configuration for the email client
type Config struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// Client handles sending emails
type Client struct {
	config Config
}

// NewClient creates a new email client
func NewClient(config Config) (*Client, error) {
	// Basic validation of configuration
	if config.SMTPHost == "" || config.SMTPPort == 0 || config.FromEmail == "" {
		return nil, ErrInvalidConfig
	}

	return &Client{
		config: config,
	}, nil
}

// EmailData contains the data for an email
type EmailData struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// SendEmail sends an email with the provided data
func (c *Client) SendEmail(data EmailData) error {
	// Construct email headers
	from := c.config.FromEmail
	if c.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.config.FromName, c.config.FromEmail)
	}

	// Set up email headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = data.To
	headers["Subject"] = data.Subject

	// Set content type
	contentType := "text/plain"
	if data.IsHTML {
		contentType = "text/html"
	}
	headers["Content-Type"] = contentType + "; charset=UTF-8"

	// Construct message
	message := ""
	for key, value := range headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	message += "\r\n" + data.Body

	// Connect to SMTP server
	auth := smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)

	// Send email
	err := smtp.SendMail(addr, auth, c.config.FromEmail, []string{data.To}, []byte(message))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return nil
}

// SendOTPEmail sends an email with an OTP
func (c *Client) SendOTPEmail(to, otp string) error {
	subject := "Your Verification Code"
	body := fmt.Sprintf(`
		<h1>Verification Code</h1>
		<p>Your verification code is: <strong>%s</strong></p>
		<p>This code will expire in 5 minutes.</p>
		<p>If you did not request this code, please ignore this email.</p>
	`, otp)

	return c.SendEmail(EmailData{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	})
}
