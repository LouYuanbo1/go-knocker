package alerter

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// ==================== WebhookAlerter ====================

// WebhookAlerter 通过 HTTP Webhook 发送告警，兼容飞书、Telegram、Slack、钉钉等。
// BodyTemplate 使用 Go text/template 语法，{{.Message}} 会被替换为告警内容。
//
// 示例模板：
//
//	Telegram:  {"chat_id":"-123","text":"{{.Message}}"}
//	飞书:      {"msg_type":"text","content":{"text":"{{.Message}}"}}
//	Slack:     {"text":"{{.Message}}"}
//	钉钉:      {"msgtype":"text","text":{"content":"{{.Message}}"}}
type WebhookAlerter struct {
	Name         string
	Client       *http.Client
	URL          string
	Method       string
	BodyTemplate string
}

// WebhookOption 用于配置 WebhookAlerter。
type WebhookOption func(*WebhookAlerter)

// NewWebhookAlerter 创建 Webhook 告警器，必要参数为 name、url、bodyTemplate。
func NewWebhookAlerter(name string, client *http.Client, url, bodyTemplate string, opts ...WebhookOption) *WebhookAlerter {
	if client == nil {
		client = http.DefaultClient
	}

	a := &WebhookAlerter{
		Name:         name,
		Client:       client,
		URL:          url,
		Method:       http.MethodPost,
		BodyTemplate: bodyTemplate,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func WithWebhookMethod(method string) WebhookOption {
	return func(a *WebhookAlerter) { a.Method = method }
}

// GetName 返回告警器名称。
func (a *WebhookAlerter) GetName() string {
	return a.Name
}

// Alert 发送 Webhook 告警。
func (a *WebhookAlerter) Alert(msg string) error {
	tmpl, err := template.New("body").Parse(a.BodyTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"Message": msg}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	req, err := http.NewRequest(a.Method, a.URL, strings.NewReader(buf.String()))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
