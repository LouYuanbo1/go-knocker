package alerter

import (
	"fmt"
	"net/smtp"
)

// ==================== EmailAlerter ====================

// EmailAlerter 通过 SMTP 邮件发送告警。
type EmailAlerter struct {
	Name     string
	SMTPHost string
	SMTPPort string
	From     string
	Password string
	To       []string
	Subject  string
}

// EmailOption 用于配置 EmailAlerter。
type EmailOption func(*EmailAlerter)

// NewEmailAlerter 创建邮件告警器，必要参数为 name、smtpHost、smtpPort、from、password、to。
func NewEmailAlerter(name, smtpHost, smtpPort, from, password string, to []string, opts ...EmailOption) *EmailAlerter {
	a := &EmailAlerter{
		Name:     name,
		SMTPHost: smtpHost,
		SMTPPort: smtpPort,
		From:     from,
		Password: password,
		To:       to,
		Subject:  "Go Knocker Alert",
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func WithEmailSubject(subject string) EmailOption {
	return func(a *EmailAlerter) { a.Subject = subject }
}

// GetName 返回告警器名称。
func (a *EmailAlerter) GetName() string {
	return a.Name
}

// Alert 发送告警邮件。
func (a *EmailAlerter) Alert(msg string) error {
	auth := smtp.PlainAuth("", a.From, a.Password, a.SMTPHost)
	body := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", a.From, a.Subject, msg)
	return smtp.SendMail(
		a.SMTPHost+":"+a.SMTPPort,
		auth,
		a.From,
		a.To,
		[]byte(body),
	)
}
