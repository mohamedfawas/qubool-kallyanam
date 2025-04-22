package email

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/smtp"
	"strings"
	"text/template"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Client represents an email client for sending emails
type Client struct {
	config Config
	auth   smtp.Auth
	logger logging.Logger
}

// Config contains SMTP configuration
type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	FromName string `yaml:"from_name"`
	// Optional TLS settings
	UseTLS             bool   `yaml:"use_tls"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	ServerName         string `yaml:"server_name"`
}

// Message represents an email message
type Message struct {
	To          []string
	Subject     string
	Body        string
	ContentType string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	Data     []byte
	MimeType string
}

// OTPData contains data for OTP email template
type OTPData struct {
	Username    string
	OTP         string
	ExpiryTime  string
	CompanyName string
	SupportLink string
	ValidMins   int
}

// New creates a new email client
func New(config Config, logger logging.Logger) (*Client, error) {
	if logger == nil {
		logger = logging.Get().Named("email")
	}

	// Initialize auth
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	return &Client{
		config: config,
		auth:   auth,
		logger: logger,
	}, nil
}

// Send sends an email
func (c *Client) Send(message Message) error {
	// Format from address
	from := c.config.From
	if c.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", c.config.FromName, c.config.From)
	}

	// Format headers
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = strings.Join(message.To, ", ")
	headers["Subject"] = message.Subject

	contentType := message.ContentType
	if contentType == "" {
		contentType = "text/html; charset=UTF-8"
	}
	headers["Content-Type"] = contentType

	// Build email message
	var buf bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(&buf, "\r\n%s", message.Body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	var err error
	if c.config.UseTLS {
		// Configure TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.config.InsecureSkipVerify,
			ServerName:         c.config.ServerName,
		}

		// Connect to TLS server
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, c.config.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		// Authenticate
		if err = client.Auth(c.auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		// Set the sender and recipients
		if err = client.Mail(c.config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, to := range message.To {
			if err = client.Rcpt(to); err != nil {
				return fmt.Errorf("failed to set recipient: %w", err)
			}
		}

		// Send the email body
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to open data pipe: %w", err)
		}

		_, err = w.Write(buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to write email data: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data pipe: %w", err)
		}

		return client.Quit()
	} else {
		// Send using standard SMTP
		err = smtp.SendMail(
			addr,
			c.auth,
			c.config.From,
			message.To,
			buf.Bytes(),
		)

		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	c.logger.Info("Email sent successfully",
		logging.String("to", strings.Join(message.To, ", ")),
		logging.String("subject", message.Subject),
	)

	return nil
}

// GenerateOTP generates a random OTP code
func GenerateOTP(length int) string {
	if length <= 0 {
		length = 6 // Default OTP length
	}

	rand.Seed(time.Now().UnixNano())
	digits := "0123456789"

	var otp strings.Builder
	for i := 0; i < length; i++ {
		otp.WriteByte(digits[rand.Intn(len(digits))])
	}

	return otp.String()
}

// SendOTPEmail sends an OTP to the provided email address
func (c *Client) SendOTPEmail(to string, data OTPData) error {
	if to == "" {
		return errors.New("recipient email is required")
	}

	// Generate OTP if not provided
	if data.OTP == "" {
		data.OTP = GenerateOTP(6)
	}

	// Set default validity period if not provided
	if data.ValidMins <= 0 {
		data.ValidMins = 10
	}

	// Set expiry time if not provided
	if data.ExpiryTime == "" {
		data.ExpiryTime = time.Now().Add(time.Duration(data.ValidMins) * time.Minute).Format("15:04:05")
	}

	// Set company name if not provided
	if data.CompanyName == "" {
		data.CompanyName = "Qubool Kallyanam"
	}

	// Render email template
	body, err := renderOTPTemplate(data)
	if err != nil {
		return fmt.Errorf("failed to render OTP template: %w", err)
	}

	// Create and send message
	message := Message{
		To:      []string{to},
		Subject: "Your Verification Code",
		Body:    body,
	}

	return c.Send(message)
}

// renderOTPTemplate renders the OTP email template
func renderOTPTemplate(data OTPData) (string, error) {
	// Simple HTML template for OTP email
	const otpTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Email Verification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; }
        .otp-box { background-color: #f7f7f7; padding: 15px; text-align: center; font-size: 24px; letter-spacing: 5px; font-weight: bold; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 12px; color: #777; text-align: center; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Email Verification</h2>
        </div>
        
        <p>Hello{{if .Username}} {{.Username}}{{end}},</p>
        
        <p>Please use the following verification code to complete your registration:</p>
        
        <div class="otp-box">{{.OTP}}</div>
        
        <p>This code will expire in {{.ValidMins}} minutes (at {{.ExpiryTime}}).</p>
        
        <p>If you didn't request this code, please ignore this email.</p>
        
        <div class="footer">
            <p>&copy; {{.CompanyName}}</p>
            {{if .SupportLink}}<p>Need help? <a href="{{.SupportLink}}">Contact Support</a></p>{{end}}
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("otp_email").Parse(otpTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
