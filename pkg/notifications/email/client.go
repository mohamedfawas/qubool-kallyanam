package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

var (
	ErrInvalidConfig = errors.New("invalid email configuration")
	ErrSendFailed    = errors.New("failed to send email")
)

// Config holds SMTP configuration
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

// NewClient validates and returns an email Client
func NewClient(cfg Config) (*Client, error) {
	if cfg.SMTPHost == "" || cfg.SMTPPort == 0 || cfg.FromEmail == "" {
		return nil, ErrInvalidConfig
	}
	// For real servers, require credentials
	if cfg.SMTPHost != "mailhog" && (cfg.SMTPUsername == "" || cfg.SMTPPassword == "") {
		return nil, ErrInvalidConfig
	}
	return &Client{config: cfg}, nil
}

// EmailData contains email details
type EmailData struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

// SendEmail builds headers, optionally negotiates TLS and sends the email
func (c *Client) SendEmail(d EmailData) error {
	// Sanitize header inputs
	sanitize := func(s string) string {
		return strings.NewReplacer("\r", "", "\n", "").Replace(s)
	}

	from := sanitize(c.config.FromEmail)
	if c.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", sanitize(c.config.FromName), from)
	}
	t0 := sanitize(d.To)
	subj := sanitize(d.Subject)

	// Prepare headers in deterministic order
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = t0
	headers["Subject"] = subj
	headers["MIME-Version"] = "1.0"

	contentType := "text/plain"
	if d.IsHTML {
		contentType = "text/html"
	}
	headers["Content-Type"] = fmt.Sprintf("%s; charset=\"UTF-8\"", contentType)
	headers["Content-Transfer-Encoding"] = "7bit"

	// Build message
	var b strings.Builder
	order := []string{"From", "To", "Subject", "MIME-Version", "Content-Type", "Content-Transfer-Encoding"}
	for _, k := range order {
		if v, ok := headers[k]; ok {
			b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	b.WriteString("\r\n")
	b.WriteString(d.Body)
	msg := []byte(b.String())

	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)

	// If MailHog, send plain
	if c.config.SMTPHost == "mailhog" {
		return smtp.SendMail(addr, nil, c.config.FromEmail, []string{d.To}, msg)
	}

	// Determine TLS vs StartTLS
	if c.config.SMTPPort == 465 {
		// Implicit TLS
		dialer := &tls.Config{ServerName: c.config.SMTPHost}
		conn, err := tls.Dial("tcp", addr, dialer)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		client, err := smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		defer client.Quit()

		// Auth
		auth := smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}

		// Set sender and recipient
		if err := client.Mail(c.config.FromEmail); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		if err := client.Rcpt(d.To); err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}

		// Data
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		_, err = w.Write(msg)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrSendFailed, err)
		}
		return w.Close()
	}

	// Port 587: STARTTLS
	auth := smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
	// The standard library will issue STARTTLS automatically on SendMail
	return smtp.SendMail(addr, auth, c.config.FromEmail, []string{d.To}, msg)
}

// SendOTPEmail helper
func (c *Client) SendOTPEmail(to, otp string) error {
	body := fmt.Sprintf(`<h1>Verification Code</h1><p>Your code: <strong>%s</strong></p>`, otp)
	return c.SendEmail(EmailData{To: to, Subject: "Your Verification Code", Body: body, IsHTML: true})
}
