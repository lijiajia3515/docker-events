package notifier

import (
	"fmt"

	"github.com/nikoksr/notify/service/msteams"

	"github.com/filippofinke/docker-events/internal/config"
)

func (n *notifierImpl) addTeams(cfg config.TeamsConfig) error {
	if len(cfg.WebhookURLs) == 0 {
		return fmt.Errorf("已启用 Teams 但未配置 Webhook 地址")
	}

	service := msteams.New()
	service.DisableWebhookValidation()
	service.AddReceivers(cfg.WebhookURLs...)
	n.client.UseServices(service)
	return nil
}
