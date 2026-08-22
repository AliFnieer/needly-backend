// Package mailer delivers transactional emails. When SMTP is not configured
// a development mailer logs the message instead of delivering it, so flows
// that send email remain exercisable locally without external dependencies.
package mailer

import (
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"

	"github.com/AliFnieer/needly-backend/internal/config"
)

// Mailer sends transactional emails.
type Mailer interface {
	Send(to, subject, body string) error
}

// New returns the configured Mailer: SMTP delivery when enabled, otherwise a
// LogMailer for development environments.
func New(cfg *config.Config) Mailer {
	if cfg.SMTP.Enabled() {
		return &SMTPMailer{cfg: cfg.SMTP}
	}
	return LogMailer{}
}

// LogMailer "sends" emails by logging them. It never fails, which keeps
// registration and password-reset flows functional in development.
type LogMailer struct{}

// Send logs the email content at info level.
func (LogMailer) Send(to, subject, body string) error {
	slog.Info("email not delivered (SMTP not configured)",
		"to", to,
		"subject", subject,
		"body", body,
	)
	return nil
}

// SMTPMailer delivers emails over SMTP using optional PLAIN authentication.
type SMTPMailer struct {
	cfg config.MailerConfig
}

// Send delivers the email through the configured SMTP server.
func (m *SMTPMailer) Send(to, subject, body string) error {
	msg := strings.Join([]string{
		"From: " + m.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")

	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}
	return nil
}
