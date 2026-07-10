package notifier

import (
	"fmt"
	stdhttp "net/http"

	notifyhttp "github.com/nikoksr/notify/service/http"

	"github.com/filippofinke/docker-events/internal/config"
)

func (n *notifierImpl) addDingTalk(cfg config.DingTalkConfig) error {
	if len(cfg.WebhookURLs) == 0 {
		return fmt.Errorf("已启用钉钉但未配置 Webhook 地址")
	}

	httpService := notifyhttp.New()

	for _, url := range cfg.WebhookURLs {
		httpService.AddReceivers(&notifyhttp.Webhook{
			URL:         url,
			Header:      stdhttp.Header{},
			ContentType: "application/json",
			Method:      stdhttp.MethodPost,
			BuildPayload: func(subject, message string) (payload any) {
				return map[string]any{
					"msgtype": "markdown",
					"markdown": map[string]string{
						"title": subject,
						"text":  fmt.Sprintf("## %s\n%s", subject, message),
					},
				}
			},
		})
	}

	n.client.UseServices(httpService)
	return nil
}
