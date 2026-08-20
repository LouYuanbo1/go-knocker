package config

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		IntervalSeconds: 30,
		Alerters: []AlerterConf{
			{Name: "飞书", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "API", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "API", Alerters: []string{"飞书"}},
		},
	}
	data, err := json.Marshal(cfg)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(path, data, 0644))

	k, err := Load(path, http.DefaultClient)
	assert.NoError(t, err)
	assert.NotNil(t, k)
	assert.Len(t, k.Items, 1)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.json", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	assert.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	_, err := Load(path, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestBuild_AllAlerterTypes(t *testing.T) {
	tests := []struct {
		alerterType string
	}{
		{"feishu"},
		{"dingtalk"},
		{"wecom"},
		{"telegram"},
		{"slack"},
		{"discord"},
		{"teams"},
		{"webhook"},
		{"email"},
	}

	for _, tt := range tests {
		t.Run(tt.alerterType, func(t *testing.T) {
			cfg := &Config{
				IntervalSeconds: 30,
				Alerters: []AlerterConf{
					{
						Name:         "test",
						Type:         tt.alerterType,
						WebhookURL:   "http://example.com",
						BodyTemplate: `{"msg":"{{.Message}}"}`,
						SMTPHost:     "smtp.example.com",
						SMTPPort:     "587",
						From:         "a@b.com",
						Password:     "p",
						To:           []string{"c@d.com"},
					},
				},
				Targets: []TargetConf{
					{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
				},
				Items: []ItemConf{
					{Target: "srv", Alerters: []string{"test"}},
				},
			}
			k, err := cfg.Build(http.DefaultClient)
			assert.NoError(t, err)
			assert.NotNil(t, k)
		})
	}
}

func TestBuild_WebhookWithoutBodyTemplate(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "webhook", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"test"}},
		},
	}
	_, err := cfg.Build(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "body_template is required")
}

func TestBuild_UnsupportedAlerterType(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "unknown"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"test"}},
		},
	}
	_, err := cfg.Build(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported alerter type")
}

func TestBuild_UnsupportedTargetType(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "unknown"},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"test"}},
		},
	}
	_, err := cfg.Build(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported target type")
}

func TestBuild_MissingTargetReference(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "nonexistent", Alerters: []string{"test"}},
		},
	}
	_, err := cfg.Build(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "references target")
}

func TestBuild_MissingAlerterReference(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"nonexistent"}},
		},
	}
	_, err := cfg.Build(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alerter")
}

func TestBuild_TCPTarget(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "mysql", Type: "tcp", Address: "127.0.0.1:3306", TimeoutSeconds: 3},
		},
		Items: []ItemConf{
			{Target: "mysql", Alerters: []string{"test"}},
		},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotNil(t, k)
	assert.Len(t, k.Items, 1)
}

func TestBuild_HTTPTargetWithExpectedResponse(t *testing.T) {
	resp := "ok"
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{
				Name:             "api",
				Type:             "http",
				URL:              "http://example.com/health",
				Method:           "GET",
				ExpectedStatus:   200,
				ExpectedResponse: &resp,
			},
		},
		Items: []ItemConf{
			{Target: "api", Alerters: []string{"test"}},
		},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotNil(t, k)
}

func TestBuild_EmailWithSubject(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{
				Name:     "邮件",
				Type:     "email",
				SMTPHost: "smtp.example.com",
				SMTPPort: "587",
				From:     "a@b.com",
				Password: "p",
				To:       []string{"c@d.com"},
				Subject:  "自定义主题",
			},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"邮件"}},
		},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotNil(t, k)
}

func TestBuild_ZeroIntervalDefaults(t *testing.T) {
	cfg := &Config{
		IntervalSeconds: 0,
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"test"}},
		},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotZero(t, k.Interval)
	// NewKnocker handles the 0→30s default internally
}

func TestBuild_RetryWarning(t *testing.T) {
	cfg := &Config{
		IntervalSeconds: 10,
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{
			{Target: "srv", Alerters: []string{"test"}, RetryCount: 3, RetrySeconds: 5},
		},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotNil(t, k)
	// 3*5=15 >= 10，应产生 WARN 日志
}

func TestLoadYAML_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yamlContent := `
interval_seconds: 30
alerters:
  - name: 飞书
    type: feishu
    webhook_url: http://example.com
targets:
  - name: API
    type: http
    url: http://example.com
    method: GET
    expected_status: 200
items:
  - target: API
    alerters:
      - 飞书
`
	assert.NoError(t, os.WriteFile(path, []byte(yamlContent), 0644))

	k, err := LoadYAML(path, http.DefaultClient)
	assert.NoError(t, err)
	assert.NotNil(t, k)
	assert.Len(t, k.Items, 1)
}

func TestLoadYAML_FileNotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/config.yaml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestBuild_EmptyItems(t *testing.T) {
	cfg := &Config{
		Alerters: []AlerterConf{
			{Name: "test", Type: "feishu", WebhookURL: "http://example.com"},
		},
		Targets: []TargetConf{
			{Name: "srv", Type: "http", URL: "http://example.com", Method: "GET", ExpectedStatus: 200},
		},
		Items: []ItemConf{},
	}
	k, err := cfg.Build(nil)
	assert.NoError(t, err)
	assert.NotNil(t, k)
	assert.Empty(t, k.Items)
}
