package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/LouYuanbo1/go-knocker/alerter"
	"github.com/LouYuanbo1/go-knocker/knocker"
	"github.com/LouYuanbo1/go-knocker/target"
	"go.yaml.in/yaml/v2"
)

// Config 配置文件结构。
type Config struct {
	IntervalSeconds int           `json:"interval_seconds" yaml:"interval_seconds"`
	Alerters        []AlerterConf `json:"alerters"         yaml:"alerters"`
	Targets         []TargetConf  `json:"targets"          yaml:"targets"`
	Items           []ItemConf    `json:"items"            yaml:"items"`
}

// AlerterConf 告警器配置。
type AlerterConf struct {
	Name         string   `json:"name"          yaml:"name"`
	Type         string   `json:"type"          yaml:"type"`
	WebhookURL   string   `json:"webhook_url,omitempty"   yaml:"webhook_url,omitempty"`
	BodyTemplate string   `json:"body_template,omitempty" yaml:"body_template,omitempty"`
	SMTPHost     string   `json:"smtp_host,omitempty"     yaml:"smtp_host,omitempty"`
	SMTPPort     string   `json:"smtp_port,omitempty"     yaml:"smtp_port,omitempty"`
	From         string   `json:"from,omitempty"          yaml:"from,omitempty"`
	Password     string   `json:"password,omitempty"      yaml:"password,omitempty"`
	To           []string `json:"to,omitempty"            yaml:"to,omitempty"`
	Subject      string   `json:"subject,omitempty"       yaml:"subject,omitempty"`
}

// TargetConf 检查目标配置。
type TargetConf struct {
	Name             string  `json:"name"              yaml:"name"`
	Type             string  `json:"type"              yaml:"type"`
	URL              string  `json:"url,omitempty"              yaml:"url,omitempty"`
	Method           string  `json:"method,omitempty"           yaml:"method,omitempty"`
	ExpectedStatus   int     `json:"expected_status,omitempty"   yaml:"expected_status,omitempty"`
	ExpectedResponse *string `json:"expected_response,omitempty" yaml:"expected_response,omitempty"`
	Address          string  `json:"address,omitempty"          yaml:"address,omitempty"`
	TimeoutSeconds   int     `json:"timeout_seconds,omitempty"   yaml:"timeout_seconds,omitempty"`
}

// ItemConf 检查项配置，关联 Target 和 Alerter。
type ItemConf struct {
	Target       string   `json:"target"       yaml:"target"`
	Alerters     []string `json:"alerters"     yaml:"alerters"`
	RetryCount   int      `json:"retry_count,omitempty"   yaml:"retry_count,omitempty"`
	RetrySeconds int      `json:"retry_seconds,omitempty" yaml:"retry_seconds,omitempty"`
}

// Load 从 JSON 文件加载配置并构建 Knocker。
func Load(path string, client *http.Client) (*knocker.Knocker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg.Build(client)
}

// LoadYAML 从 YAML 文件加载配置并构建 Knocker。
func LoadYAML(path string, client *http.Client) (*knocker.Knocker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg.Build(client)
}

// Build 根据配置构建 Knocker。
func (cfg *Config) Build(client *http.Client) (*knocker.Knocker, error) {
	// 构建告警器映射
	alerterMap := make(map[string]alerter.Alerter, len(cfg.Alerters))
	for _, ac := range cfg.Alerters {
		a, err := buildAlerter(ac, client)
		if err != nil {
			return nil, fmt.Errorf("alerter %q: %w", ac.Name, err)
		}
		alerterMap[ac.Name] = a
	}

	// 构建目标映射
	targetMap := make(map[string]target.Target, len(cfg.Targets))
	for _, tc := range cfg.Targets {
		t, err := buildTarget(tc, client)
		if err != nil {
			return nil, fmt.Errorf("target %q: %w", tc.Name, err)
		}
		targetMap[tc.Name] = t
	}

	// 构建 Items
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	intervalSec := cfg.IntervalSeconds
	if intervalSec <= 0 {
		intervalSec = 30 // 与 NewKnocker 默认值一致
	}
	items := make([]*knocker.Item, 0, len(cfg.Items))
	for _, ic := range cfg.Items {
		t, ok := targetMap[ic.Target]
		if !ok {
			return nil, fmt.Errorf("item references target %q which is not defined", ic.Target)
		}
		alerters := make([]alerter.Alerter, 0, len(ic.Alerters))
		for _, name := range ic.Alerters {
			a, ok := alerterMap[name]
			if !ok {
				return nil, fmt.Errorf("item %q: alerter %q not found", ic.Target, name)
			}
			alerters = append(alerters, a)
		}

		item := knocker.NewItem(t, alerters, ic.RetryCount, time.Duration(ic.RetrySeconds)*time.Second)

		// 重试总耗时超过检查间隔时发出警告
		if retryTotal := (item.RetryCount - 1) * int(item.RetryDelay.Seconds()); retryTotal >= intervalSec {
			log.Printf("[WARN] item %q: (retry_count-1)(%d) × retry_delay(%ds) = %ds ≥ interval(%ds)，检查间隔将被拉长",
				ic.Target, item.RetryCount-1, int(item.RetryDelay.Seconds()), retryTotal, intervalSec)
		}

		items = append(items, item)
	}

	return knocker.NewKnocker(items, interval), nil
}

// buildAlerter 根据配置构建告警器。
func buildAlerter(ac AlerterConf, client *http.Client) (alerter.Alerter, error) {
	switch ac.Type {
	case "email":
		opts := []alerter.EmailOption{}
		if ac.Subject != "" {
			opts = append(opts, alerter.WithEmailSubject(ac.Subject))
		}
		return alerter.NewEmailAlerter(
			ac.Name, ac.SMTPHost, ac.SMTPPort, ac.From, ac.Password, ac.To, opts...,
		), nil
	case "webhook":
		if ac.BodyTemplate == "" {
			return nil, fmt.Errorf("body_template is required for webhook type")
		}
		return alerter.NewWebhookAlerter(ac.Name, client, ac.WebhookURL, ac.BodyTemplate), nil
	default:
		tmpl, ok := platformTemplates[ac.Type]
		if !ok {
			return nil, fmt.Errorf("unsupported alerter type: %s", ac.Type)
		}
		return alerter.NewWebhookAlerter(ac.Name, client, ac.WebhookURL, tmpl), nil
	}
}

// buildTarget 根据配置构建检查目标。
func buildTarget(tc TargetConf, client *http.Client) (target.Target, error) {
	switch tc.Type {
	case "http":
		var opts []target.HTTPOption
		if tc.ExpectedResponse != nil {
			opts = append(opts, target.WithExpectedResponse(*tc.ExpectedResponse))
		}
		return target.NewHTTPTarget(
			client, tc.Name, tc.URL, tc.Method, tc.ExpectedStatus, opts...,
		), nil
	case "tcp":
		var opts []target.TCPOption
		if tc.TimeoutSeconds > 0 {
			opts = append(opts, target.WithTCPTimeout(time.Duration(tc.TimeoutSeconds)*time.Second))
		}
		return target.NewTCPTarget(tc.Name, tc.Address, opts...), nil
	default:
		return nil, fmt.Errorf("unsupported target type: %s", tc.Type)
	}
}

// platformTemplates 内置平台 → 模板映射。
var platformTemplates = map[string]string{
	"feishu":   alerter.FeishuText,
	"dingtalk": alerter.DingTalkText,
	"wecom":    alerter.WeComText,
	"telegram": alerter.TelegramText,
	"slack":    alerter.SlackText,
	"discord":  alerter.DiscordText,
	"teams":    alerter.TeamsText,
}
