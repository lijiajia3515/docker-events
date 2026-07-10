package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSubjectPrefix   = "Docker 事件"
	defaultMessageTemplate = `🕐 时间: {{.Time}}
📋 状态: {{.Status}}
📦 来源: {{.From}}
🌐 范围: {{.Scope}}
🔖 ID: {{.ID}}
👤 执行者: {{.Actor.ID}}`
	defaultLogLines = 0
)

func Load() (*Config, error) {
	logLines := defaultLogLines
	if logLinesStr := os.Getenv("MESSAGE_LOG_LINES"); logLinesStr != "" {
		parsed, err := strconv.Atoi(logLinesStr)
		if err != nil {
			return nil, fmt.Errorf("无效的 MESSAGE_LOG_LINES 值 %q: %w", logLinesStr, err)
		}
		logLines = parsed
	}

	cfg := &Config{
		DockerFilters:    parseFilters(os.Getenv("DOCKER_EVENT_FILTERS")),
		DockerEventTypes: parseEventTypes(os.Getenv("DOCKER_EVENT_TYPES")),
		NotifySubject:    unescapeNewlines(getEnvOrDefault("NOTIFY_SUBJECT", "Docker 事件")),
		MessageTemplate:  unescapeNewlines(os.Getenv("MESSAGE_TEMPLATE")),
		LogLines:         logLines,
		EventGroupWindow: parseGroupWindow(os.Getenv("EVENT_GROUP_WINDOW")),
	}

	slackToken, ok := os.LookupEnv("SLACK_BOT_TOKEN")
	if ok && slackToken != "" {
		slackChannels := splitAndTrim(os.Getenv("SLACK_CHANNEL_IDS"))
		if len(slackChannels) == 0 {
			return nil, errors.New("已配置 Slack 但 SLACK_CHANNEL_IDS 为空")
		}

		cfg.Slack = SlackConfig{
			Enabled:  true,
			Token:    slackToken,
			Channels: slackChannels,
		}
	}

	telegramToken, ok := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	if ok && telegramToken != "" {
		rawChatIDs := splitAndTrim(os.Getenv("TELEGRAM_CHAT_IDS"))
		if len(rawChatIDs) == 0 {
			return nil, errors.New("已配置 Telegram 但 TELEGRAM_CHAT_IDS 为空")
		}

		chatIDs := make([]int64, 0, len(rawChatIDs))
		for _, rawID := range rawChatIDs {
			chatID, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无效的 TELEGRAM_CHAT_IDS 值 %q: %w", rawID, err)
			}
			chatIDs = append(chatIDs, chatID)
		}

		cfg.Telegram = TelegramConfig{
			Enabled: true,
			Token:   telegramToken,
			ChatIDs: chatIDs,
		}
	}

	discordToken, ok := os.LookupEnv("DISCORD_BOT_TOKEN")
	if ok && discordToken != "" {
		discordChannels := splitAndTrim(os.Getenv("DISCORD_CHANNEL_IDS"))
		if len(discordChannels) == 0 {
			return nil, errors.New("已配置 Discord Bot 但 DISCORD_CHANNEL_IDS 为空")
		}

		cfg.Discord = DiscordConfig{
			Enabled:    true,
			Token:      discordToken,
			ChannelIDs: discordChannels,
		}
	}

	discordWebhooks := splitAndTrim(os.Getenv("DISCORD_WEBHOOK_URLS"))
	if len(discordWebhooks) > 0 {
		if cfg.Discord.Enabled {
			cfg.Discord.WebhookURLs = discordWebhooks
		} else {
			cfg.Discord = DiscordConfig{
				Enabled:     true,
				WebhookURLs: discordWebhooks,
			}
		}
	}

	teamsWebhooks := splitAndTrim(os.Getenv("TEAMS_WEBHOOK_URLS"))
	if len(teamsWebhooks) > 0 {
		cfg.Teams = TeamsConfig{
			Enabled:     true,
			WebhookURLs: teamsWebhooks,
		}
	}

	wecomWebhooks := splitAndTrim(os.Getenv("WECOM_WEBHOOK_URLS"))
	if len(wecomWebhooks) > 0 {
		cfg.WeChatWork = WeChatWorkConfig{
			Enabled:     true,
			WebhookURLs: wecomWebhooks,
		}
	}

	dingtalkWebhooks := splitAndTrim(os.Getenv("DINGTALK_WEBHOOK_URLS"))
	if len(dingtalkWebhooks) > 0 {
		cfg.DingTalk = DingTalkConfig{
			Enabled:     true,
			WebhookURLs: dingtalkWebhooks,
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseFilters(raw string) []string {
	return splitAndTrim(raw)
}

func parseEventTypes(raw string) []string {
	if raw == "" {
		return []string{"container"}
	}
	return splitAndTrim(raw)
}

func parseGroupWindow(raw string) time.Duration {
	if raw == "" {
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 5 * time.Second
	}
	return duration
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func unescapeNewlines(s string) string {
	return strings.ReplaceAll(s, `\n`, "\n")
}

func splitAndTrim(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
