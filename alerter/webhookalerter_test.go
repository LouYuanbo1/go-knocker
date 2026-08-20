package alerter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWebhookAlerter_NilClient(t *testing.T) {
	a := NewWebhookAlerter("test", nil, "http://example.com", `{"text":"{{.Message}}"}`)
	assert.Equal(t, http.DefaultClient, a.Client)
	assert.Equal(t, "test", a.Name)
	assert.Equal(t, "http://example.com", a.URL)
	assert.Equal(t, http.MethodPost, a.Method)
	assert.Equal(t, `{"text":"{{.Message}}"}`, a.BodyTemplate)
}

func TestNewWebhookAlerter_CustomClient(t *testing.T) {
	client := &http.Client{}
	a := NewWebhookAlerter("test", client, "http://example.com", `{"text":"{{.Message}}"}`)
	assert.Same(t, client, a.Client)
}

func TestNewWebhookAlerter_CustomMethod(t *testing.T) {
	a := NewWebhookAlerter("test", nil, "http://example.com", `{"text":"{{.Message}}"}`,
		WithWebhookMethod(http.MethodPut),
	)
	assert.Equal(t, http.MethodPut, a.Method)
}

func TestWebhookAlerter_GetName(t *testing.T) {
	a := NewWebhookAlerter("feishu", nil, "http://example.com", FeishuText)
	assert.Equal(t, "feishu", a.GetName())
}

func TestWebhookAlerter_Alert_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "MySQL 连接失败")
		w.WriteHeader(200)
	}))
	defer server.Close()

	a := NewWebhookAlerter("test", server.Client(), server.URL, `{"text":"{{.Message}}"}`)
	err := a.Alert("MySQL 连接失败")
	assert.NoError(t, err)
}

func TestWebhookAlerter_Alert_InvalidTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	// 未闭合的模板语法，parse 阶段就会失败
	a := NewWebhookAlerter("test", server.Client(), server.URL, `{"text":"{{.Message}"}`)
	err := a.Alert("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse template")
}

func TestWebhookAlerter_Alert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	a := NewWebhookAlerter("test", server.Client(), server.URL, `{"text":"{{.Message}}"}`)
	err := a.Alert("test")
	assert.Error(t, err)
}
