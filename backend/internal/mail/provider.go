// Package mail delivers transactional email. Configuration is resolved
// dynamically: an explicit database setting wins, otherwise the process
// environment supplies the default. When nothing is configured the Sender is a
// no-op, so callers never have to special-case "email is off".
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"github.com/fintrack/backend/internal/infrastructure/crypto"
)

// Config is the resolved mail transport configuration.
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// FromName is the display name used in the From header.
	FromName string
	// UseTLS selects implicit TLS (port 465). Otherwise STARTTLS is attempted.
	UseTLS bool
}

// SettingReader reads a raw setting value by key ("" when absent). Implemented
// by the settings repository; kept as an interface so mail has no DB import.
type SettingReader interface {
	GetString(ctx context.Context, key string) (string, error)
}

// Setting keys consulted, in priority order: DB setting first, then env.
const (
	KeyMailEnabled  = "mail_enabled"
	KeyMailHost     = "mail_host"
	KeyMailPort     = "mail_port"
	KeyMailUsername = "mail_username"
	KeyMailPassword = "mail_password" // stored encrypted (AES-GCM hex)
	KeyMailFrom     = "mail_from"
	KeyMailFromName = "mail_from_name"
	KeyMailUseTLS   = "mail_use_tls"
)

// Sender sends an email. The no-op implementation reports ErrDisabled so
// callers can distinguish "skipped" from "failed".
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
	// Configured reports whether a real transport is wired up.
	Configured() bool
}

// ErrDisabled is returned when email is not configured.
var ErrDisabled = fmt.Errorf("mail: not configured")

// noopSender is used when email is disabled or unconfigured.
type noopSender struct{}

func (noopSender) Send(context.Context, string, string, string) error { return ErrDisabled }
func (noopSender) Configured() bool                                   { return false }

// Provider resolves the effective config and hands out a Sender. Config is read
// through the DB setting when present, otherwise from env — so an operator can
// change SMTP without a redeploy, while env still works out of the box.
type Provider struct {
	settings SettingReader
	// encryptionKey decrypts a stored mail_password (32 bytes, AES-256-GCM).
	encryptionKey []byte
}

func NewProvider(settings SettingReader, encryptionKey []byte) *Provider {
	return &Provider{settings: settings, encryptionKey: encryptionKey}
}

// resolve reads the DB setting, falling back to the environment default.
func (p *Provider) resolve(ctx context.Context, settingKey, envKey, def string) string {
	if p.settings != nil {
		if v, err := p.settings.GetString(ctx, settingKey); err == nil && strings.TrimSpace(v) != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	return def
}

func (p *Provider) resolveBool(ctx context.Context, settingKey, envKey string, def bool) bool {
	raw := p.resolve(ctx, settingKey, envKey, "")
	if raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return b
}

// Config returns the effective configuration. A stored password is decrypted;
// a plaintext env password is used as-is.
func (p *Provider) Config(ctx context.Context) Config {
	cfg := Config{
		Enabled:  p.resolveBool(ctx, KeyMailEnabled, "MAIL_ENABLED", false),
		Host:     p.resolve(ctx, KeyMailHost, "SMTP_HOST", ""),
		Username: p.resolve(ctx, KeyMailUsername, "SMTP_USERNAME", ""),
		From:     p.resolve(ctx, KeyMailFrom, "MAIL_FROM", ""),
		FromName: p.resolve(ctx, KeyMailFromName, "MAIL_FROM_NAME", "FinTrack"),
		UseTLS:   p.resolveBool(ctx, KeyMailUseTLS, "SMTP_USE_TLS", false),
	}
	portRaw := p.resolve(ctx, KeyMailPort, "SMTP_PORT", "587")
	if n, err := strconv.Atoi(portRaw); err == nil {
		cfg.Port = n
	} else {
		cfg.Port = 587
	}

	// Password: prefer the encrypted DB value, else the env default.
	cfg.Password = strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	if p.settings != nil {
		if enc, err := p.settings.GetString(ctx, KeyMailPassword); err == nil && enc != "" {
			if len(p.encryptionKey) == 32 {
				if plain, derr := crypto.Decrypt(enc, p.encryptionKey); derr == nil {
					cfg.Password = string(plain)
				}
			}
		}
	}
	return cfg
}

// Sender returns a Sender for the current config: a real SMTP sender when
// enabled and fully configured, otherwise the no-op sender.
func (p *Provider) Sender(ctx context.Context) Sender {
	cfg := p.Config(ctx)
	if !cfg.Enabled || cfg.Host == "" || cfg.From == "" {
		return noopSender{}
	}
	return &smtpSender{cfg: cfg}
}

// smtpSender sends via net/smtp.
type smtpSender struct{ cfg Config }

func (s *smtpSender) Configured() bool { return true }

func (s *smtpSender) Send(_ context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	from := s.cfg.From
	fromHeader := from
	if s.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.cfg.FromName, from)
	}
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"utf-8\"",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	// Implicit TLS (typically 465): dial TLS directly.
	if s.cfg.UseTLS {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return fmt.Errorf("mail: dial tls: %w", err)
		}
		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("mail: new client: %w", err)
		}
		defer func() { _ = client.Close() }()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("mail: auth: %w", err)
			}
		}
		if err := client.Mail(from); err != nil {
			return fmt.Errorf("mail: from: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("mail: rcpt: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("mail: data: %w", err)
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return fmt.Errorf("mail: write: %w", err)
		}
		return w.Close()
	}

	// STARTTLS via smtp.SendMail (upgrades automatically when the server offers it).
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

// EncryptPassword seals a plaintext SMTP password for storage in settings.
// Returns "" when no encryption key is configured (caller should then keep it
// in env instead of the database).
func EncryptPassword(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", crypto.ErrInvalidKeyLength
	}
	return crypto.Encrypt([]byte(plaintext), key)
}
