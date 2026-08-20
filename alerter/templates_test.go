package alerter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateConstants_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		template string
	}{
		{"FeishuText", FeishuText},
		{"DingTalkText", DingTalkText},
		{"WeComText", WeComText},
		{"TelegramText", TelegramText},
		{"SlackText", SlackText},
		{"DiscordText", DiscordText},
		{"TeamsText", TeamsText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewWebhookAlerter("test", server.Client(), server.URL, tt.template)
			err := a.Alert("test message")
			assert.NoError(t, err, "template %s should be valid", tt.name)
		})
	}
}

func TestShortcutConstructors(t *testing.T) {
	client := &http.Client{}
	url := "http://example.com"

	tests := []struct {
		name string
		fn   func() *WebhookAlerter
	}{
		{"Feishu", func() *WebhookAlerter { return NewFeishuAlerter("Feishu", client, url) }},
		{"DingTalk", func() *WebhookAlerter { return NewDingTalkAlerter("DingTalk", client, url) }},
		{"WeCom", func() *WebhookAlerter { return NewWeComAlerter("WeCom", client, url) }},
		{"Telegram", func() *WebhookAlerter { return NewTelegramAlerter("Telegram", client, url) }},
		{"Slack", func() *WebhookAlerter { return NewSlackAlerter("Slack", client, url) }},
		{"Discord", func() *WebhookAlerter { return NewDiscordAlerter("Discord", client, url) }},
		{"Teams", func() *WebhookAlerter { return NewTeamsAlerter("Teams", client, url) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.fn()
			assert.Equal(t, tt.name, a.Name)
			assert.Equal(t, url, a.URL)
			assert.NotEmpty(t, a.BodyTemplate)
		})
	}
}

func TestShortcutConstructors_WithOptions(t *testing.T) {
	client := &http.Client{}
	a := NewFeishuAlerter("Feishu", client, "http://example.com", WithWebhookMethod(http.MethodPut))
	assert.Equal(t, http.MethodPut, a.Method)
	assert.Equal(t, FeishuText, a.BodyTemplate)
}

func TestTemplates_RenderMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		template string
	}{
		{"FeishuText", FeishuText},
		{"DingTalkText", DingTalkText},
		{"WeComText", WeComText},
		{"TelegramText", TelegramText},
		{"SlackText", SlackText},
		{"DiscordText", DiscordText},
		{"TeamsText", TeamsText},
	}

	msg := "服务异常: MySQL"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewWebhookAlerter("test", server.Client(), server.URL, tt.template)
			err := a.Alert(msg)
			assert.NoError(t, err)
		})
	}
}
