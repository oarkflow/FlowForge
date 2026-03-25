package notifications

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"
)

// EmailConfig holds SMTP configuration for email notifications.
type EmailConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

// EmailNotifier sends notifications via SMTP email.
type EmailNotifier struct {
	config EmailConfig
}

// NewEmailNotifier creates a new email notifier.
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{config: config}
}

func (e *EmailNotifier) Type() string { return "email" }

func (e *EmailNotifier) Send(event *Event) error {
	subject := fmt.Sprintf("[FlowForge] Pipeline %s #%d — %s",
		event.PipelineName, event.RunNumber, strings.ToUpper(event.Status))

	body, err := e.renderHTML(event)
	if err != nil {
		return fmt.Errorf("email: render template: %w", err)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n"+
		"MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		e.config.From,
		strings.Join(e.config.To, ", "),
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	var auth smtp.Auth
	if e.config.Username != "" {
		auth = smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.Host)
	}

	err = smtp.SendMail(addr, auth, e.config.From, e.config.To, []byte(msg))
	if err != nil {
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}

var emailTpl = template.Must(template.New("email").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0d1117; color: #c9d1d9; padding: 20px;">
  <div style="max-width: 600px; margin: 0 auto; background: #161b22; border-radius: 8px; border: 1px solid #30363d; padding: 24px;">
    <h2 style="margin: 0 0 16px; color: {{if eq .Status "success"}}#3fb950{{else if eq .Status "failure"}}#f85149{{else}}#d29922{{end}};">
      Pipeline {{.PipelineName}} #{{.RunNumber}} — {{.Status}}
    </h2>
    <table style="width: 100%; border-collapse: collapse;">
      <tr><td style="padding: 8px 0; color: #8b949e;">Branch</td><td style="padding: 8px 0;">{{.Branch}}</td></tr>
      <tr><td style="padding: 8px 0; color: #8b949e;">Author</td><td style="padding: 8px 0;">{{.Author}}</td></tr>
      <tr><td style="padding: 8px 0; color: #8b949e;">Commit</td><td style="padding: 8px 0;"><code>{{.ShortSHA}}</code></td></tr>
      {{if .Duration}}<tr><td style="padding: 8px 0; color: #8b949e;">Duration</td><td style="padding: 8px 0;">{{.Duration}}</td></tr>{{end}}
    </table>
    <hr style="border: none; border-top: 1px solid #30363d; margin: 16px 0;">
    <p style="color: #8b949e; font-size: 12px; margin: 0;">FlowForge CI/CD • {{.Timestamp}}</p>
  </div>
</body>
</html>`))

func (e *EmailNotifier) renderHTML(event *Event) (string, error) {
	sha := event.CommitSHA
	if len(sha) > 8 {
		sha = sha[:8]
	}

	data := struct {
		PipelineName string
		RunNumber    int
		Status       string
		Branch       string
		Author       string
		ShortSHA     string
		Duration     string
		Timestamp    string
	}{
		PipelineName: event.PipelineName,
		RunNumber:    event.RunNumber,
		Status:       event.Status,
		Branch:       event.Branch,
		Author:       event.Author,
		ShortSHA:     sha,
		Timestamp:    event.Timestamp.Format(time.RFC1123),
	}
	if event.Duration > 0 {
		data.Duration = event.Duration.Round(time.Second).String()
	}

	var buf bytes.Buffer
	if err := emailTpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
