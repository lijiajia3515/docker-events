package notifier

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"

	notifyhttp "github.com/nikoksr/notify/service/http"

	"github.com/filippofinke/docker-events/internal/config"
)

func (n *notifierImpl) addWeChatWork(cfg config.WeChatWorkConfig) error {
	if len(cfg.WebhookURLs) == 0 {
		return fmt.Errorf("已启用企业微信但未配置 Webhook 地址")
	}

	httpService := notifyhttp.New()

	for _, url := range cfg.WebhookURLs {
		httpService.AddReceivers(&notifyhttp.Webhook{
			URL:         url,
			Header:      stdhttp.Header{},
			ContentType: "application/json",
			Method:      stdhttp.MethodPost,
			BuildPayload: func(subject, message string) (payload any) {
				content := fmt.Sprintf("%s\n%s", subject, message)
				p := map[string]any{
					"msgtype": "markdown",
					"markdown": map[string]string{
						"content": content,
					},
				}
				if b, err := json.MarshalIndent(p, "", "  "); err == nil {
					n.logger.Info("企微发送消息内容", "payload", string(b))
				}
				return p
			},
		})
	}

	n.client.UseServices(httpService)
	return nil
}
