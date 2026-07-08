// Package mailer sends transactional email over SMTP (Mailpit in dev).
package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host     string
	port     string
	user     string
	pass     string
	from     string
	fromName string
}

func New(host, port, user, pass, from, fromName string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from, fromName: fromName}
}

// Send delivers a simple HTML email.
func (m *Mailer) Send(to, subject, htmlBody string) error {
	fromHeader := m.from
	if m.fromName != "" {
		fromHeader = fmt.Sprintf("%q <%s>", m.fromName, m.from)
	}
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
		"",
		htmlBody,
	}, "\r\n")

	addr := m.host + ":" + m.port
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	if err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", to, err)
	}
	return nil
}
