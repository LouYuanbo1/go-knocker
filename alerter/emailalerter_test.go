package alerter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEmailAlerter(t *testing.T) {
	to := []string{"admin@example.com"}
	a := NewEmailAlerter("邮件", "smtp.example.com", "587", "from@example.com", "secret", to)
	assert.Equal(t, "邮件", a.Name)
	assert.Equal(t, "smtp.example.com", a.SMTPHost)
	assert.Equal(t, "587", a.SMTPPort)
	assert.Equal(t, "from@example.com", a.From)
	assert.Equal(t, "secret", a.Password)
	assert.Equal(t, to, a.To)
	assert.Equal(t, "Go Knocker Alert", a.Subject)
}

func TestNewEmailAlerter_CustomSubject(t *testing.T) {
	to := []string{"admin@example.com"}
	a := NewEmailAlerter("邮件", "smtp.example.com", "587", "from@example.com", "secret", to,
		WithEmailSubject("服务告警"),
	)
	assert.Equal(t, "服务告警", a.Subject)
}

func TestEmailAlerter_GetName(t *testing.T) {
	to := []string{"admin@example.com"}
	a := NewEmailAlerter("邮件", "smtp.example.com", "587", "from@example.com", "secret", to)
	assert.Equal(t, "邮件", a.GetName())
}

func TestEmailAlerter_Alert_InvalidServer(t *testing.T) {
	to := []string{"admin@example.com"}
	a := NewEmailAlerter("邮件", "invalid.smtp.example", "587", "from@example.com", "secret", to,
		WithEmailSubject("告警"),
	)
	err := a.Alert("服务异常")
	assert.Error(t, err)
}
