package notifier

import (
	"fmt"
	stdhttp "net/http"
	"strings"

	"github.com/nikoksr/notify/service/discord"
	notifyhttp "github.com/nikoksr/notify/service/http"

	"github.com/filippofinke/docker-events/internal/config"
)

func (n *notifierImpl) addDiscord(cfg config.DiscordConfig) error {
	// 如果提供了 Bot Token，则配置 Discord Bot
	if strings.TrimSpace(cfg.Token) != "" {
		if len(cfg.ChannelIDs) == 0 {
			return fmt.Errorf("已配置 Discord Bot 但未指定频道")
		}
		service := discord.New()
		if err := service.AuthenticateWithBotToken(cfg.Token); err != nil {
			return fmt.Errorf("Discord Bot 认证失败: %w", err)
		}
		service.AddReceivers(cfg.ChannelIDs...)
		n.client.UseServices(service)
	}

	// 如果提供了 Webhook URL，则通过 notify 的 HTTP 服务配置 Discord Webhook
	if len(cfg.WebhookURLs) > 0 {
		httpService := notifyhttp.New()

		for _, url := range cfg.WebhookURLs {
			httpService.AddReceivers(&notifyhttp.Webhook{
				URL:         url,
				Header:      stdhttp.Header{},
				ContentType: "application/json",
				Method:      stdhttp.MethodPost,
				BuildPayload: func(subject, message string) (payload any) {
					return map[string]any{
						"content": fmt.Sprintf("**%s**\n%s", subject, message),
						"embeds":  map[string]any{},
					}
				},
			})
		}

		n.client.UseServices(httpService)
	}

	if strings.TrimSpace(cfg.Token) == "" && len(cfg.WebhookURLs) == 0 {
		return fmt.Errorf("已启用 Discord 但未配置 Bot Token 或 Webhook 地址")
	}

	return nil
}
