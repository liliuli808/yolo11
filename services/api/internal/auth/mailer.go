package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/yiguan/api/internal/platform/config"
)

// Mailer sends transactional emails to users.
type Mailer interface {
	SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error
}

// NewMailerFromConfig returns the configured Mailer implementation.
func NewMailerFromConfig(cfg *config.Config, logger *slog.Logger) Mailer {
	switch strings.ToLower(cfg.EmailAdapter) {
	case "smtp":
		return &SMTPMailer{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.EmailFrom,
			UseTLS:   cfg.SMTPTLS,
			logger:   logger,
		}
	default:
		return &ConsoleMailer{logger: logger}
	}
}

// ConsoleMailer prints email content to stdout for local development capture.
type ConsoleMailer struct {
	logger *slog.Logger
}

func (m *ConsoleMailer) SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error {
	if m.logger != nil {
		m.logger.Info("sending email code",
			slog.String("to", to),
			slog.Duration("expires_in", expiresIn),
		)
	}
	fmt.Printf("\n--- Email to: %s ---\n", to)
	fmt.Printf("Subject: Your Lantern verification code\n\n")
	fmt.Printf("Your Lantern verification code is %s.\n", code)
	fmt.Printf("It expires in %d minutes.\n", int(expiresIn.Minutes()))
	fmt.Printf("If you did not request this code, you can safely ignore this email.\n")
	fmt.Println("---")
	return nil
}

// SMTPMailer sends email through an SMTP relay.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	logger   *slog.Logger
}

func (m *SMTPMailer) SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error {
	if m.Host == "" {
		return fmt.Errorf("smtp host is not configured")
	}

	addr := net.JoinHostPort(m.Host, fmt.Sprintf("%d", m.Port))
	msg := buildEmailCodeMessage(m.From, to, code, expiresIn)

	var err error
	if m.UseTLS {
		err = m.sendTLS(ctx, addr, to, msg)
	} else {
		err = m.sendPlain(ctx, addr, to, msg)
	}
	if err != nil {
		return fmt.Errorf("send smtp email: %w", err)
	}

	if m.logger != nil {
		m.logger.Info("sent email code via smtp", slog.String("to", to), slog.String("host", m.Host))
	}
	return nil
}

func (m *SMTPMailer) sendPlain(ctx context.Context, addr, to string, msg []byte) error {
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}

	// net/smtp does not accept a context; use a short timeout for the operation.
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.From, []string{to}, msg)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *SMTPMailer) sendTLS(ctx context.Context, addr, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("dial smtps: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if err := client.Mail(m.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func buildEmailCodeMessage(from, to, code string, expiresIn time.Duration) []byte {
	subject := "Your Lantern verification code"
	body := fmt.Sprintf(
		"Your Lantern verification code is %s.\n\n"+
			"It expires in %d minutes.\n\n"+
			"If you did not request this code, you can safely ignore this email.\n",
		code, int(expiresIn.Minutes()),
	)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
	fmt.Fprintf(&b, "\r\n")
	fmt.Fprint(&b, body)
	return []byte(b.String())
}
