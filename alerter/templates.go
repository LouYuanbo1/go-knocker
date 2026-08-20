package alerter

import "net/http"

// 常用办公软件 Webhook Body 模板，配合 NewWebhookAlerter 使用。
//
// 使用方式：
//
//	a := alerter.NewWebhookAlerter(client, url, alerter.FeishuText)
//	a := alerter.NewFeishuAlerter(client, url)  // 快捷方式
const (
	// FeishuText 飞书/Lark 文本消息。
	// 文档：https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN
	FeishuText = `{"msg_type":"text","content":{"text":"{{.Message}}"}}`

	// DingTalkText 钉钉 文本消息。
	// 文档：https://open.dingtalk.com/document/orgapp/custom-bot-send-message-type
	DingTalkText = `{"msgtype":"text","text":{"content":"{{.Message}}"}}`

	// WeComText 企业微信 文本消息。
	// 文档：https://developer.work.weixin.qq.com/document/path/91770
	WeComText = `{"msgtype":"text","text":{"content":"{{.Message}}"}}`

	// TelegramText Telegram Bot 文本消息。
	// 文档：https://core.telegram.org/bots/api#sendmessage
	// 注意：chat_id 需要在 URL 中指定，或替换模板中的 <CHAT_ID>。
	TelegramText = `{"chat_id":"<CHAT_ID>","text":"{{.Message}}"}`

	// SlackText Slack Incoming Webhook 文本消息。
	// 文档：https://api.slack.com/messaging/webhooks
	SlackText = `{"text":"{{.Message}}"}`

	// DiscordText Discord Webhook 文本消息。
	// 文档：https://discord.com/developers/docs/resources/webhook
	DiscordText = `{"content":"{{.Message}}"}`

	// TeamsText Microsoft Teams Incoming Webhook 文本消息。
	// 文档：https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook
	TeamsText = `{"text":"{{.Message}}"}`
)

// ==================== 快捷构造函数 ====================

// NewFeishuAlerter 创建飞书/Lark 文本消息告警器。
func NewFeishuAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, FeishuText, opts...)
}

// NewDingTalkAlerter 创建钉钉文本消息告警器。
func NewDingTalkAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, DingTalkText, opts...)
}

// NewWeComAlerter 创建企业微信文本消息告警器。
func NewWeComAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, WeComText, opts...)
}

// NewTelegramAlerter 创建 Telegram Bot 文本消息告警器。
// 注意：模板中 <CHAT_ID> 需要替换为实际 chat_id，或在 URL 中指定。
func NewTelegramAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, TelegramText, opts...)
}

// NewSlackAlerter 创建 Slack 文本消息告警器。
func NewSlackAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, SlackText, opts...)
}

// NewDiscordAlerter 创建 Discord 文本消息告警器。
func NewDiscordAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, DiscordText, opts...)
}

// NewTeamsAlerter 创建 Microsoft Teams 文本消息告警器。
func NewTeamsAlerter(name string, client *http.Client, url string, opts ...WebhookOption) *WebhookAlerter {
	return NewWebhookAlerter(name, client, url, TeamsText, opts...)
}
