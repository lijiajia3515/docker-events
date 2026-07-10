package config

import (
	"strings"
	"time"
)

type Config struct {
	DockerFilters    []string
	DockerEventTypes []string
	NotifySubject    string
	MessageTemplate  string
	LogLines         int
	EventGroupWindow time.Duration
	Slack            SlackConfig
	Telegram         TelegramConfig
	Discord          DiscordConfig
	Teams            TeamsConfig
	WeChatWork       WeChatWorkConfig
	DingTalk         DingTalkConfig
}

type SlackConfig struct {
	Enabled  bool
	Token    string
	Channels []string
}

type TelegramConfig struct {
	Enabled bool
	Token   string
	ChatIDs []int64
}

type DiscordConfig struct {
	Enabled     bool
	Token       string
	ChannelIDs  []string
	WebhookURLs []string
}

type TeamsConfig struct {
	Enabled     bool
	WebhookURLs []string
}

type WeChatWorkConfig struct {
	Enabled     bool
	WebhookURLs []string
}

type DingTalkConfig struct {
	Enabled     bool
	WebhookURLs []string
}

func (c *Config) Validate() error {
	var missing []string

	if !c.Slack.Enabled && !c.Telegram.Enabled && !c.Discord.Enabled && !c.Teams.Enabled && !c.WeChatWork.Enabled && !c.DingTalk.Enabled {
		missing = append(missing, "通知渠道凭据 (Slack、Telegram、Discord、Teams、企业微信或钉钉)")
	}

	if len(missing) > 0 {
		return &configError{msg: "缺少必需的配置: " + strings.Join(missing, ", ")}
	}

	return nil
}

type configError struct{ msg string }

func (e *configError) Error() string { return e.msg }
