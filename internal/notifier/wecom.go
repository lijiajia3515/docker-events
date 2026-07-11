package notifier

import (
	"fmt"
	stdhttp "net/http"
	"unicode/utf8"

	notifyhttp "github.com/nikoksr/notify/service/http"

	"github.com/filippofinke/docker-events/internal/config"
)

// wecomMarkdownMaxBytes 企微 markdown 消息内容上限（4096 字节）
const wecomMarkdownMaxBytes = 4096

// truncateUTF8 在 maxBytes 范围内安全截断 UTF-8 字符串，不会切断多字节字符
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// 回退到最后一个完整的 UTF-8 字符边界
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

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

				// 超出企微 markdown 字节上限时安全截断
				if len(content) > wecomMarkdownMaxBytes {
					suffix := "\n\n...(消息已截断)"
					content = truncateUTF8(content, wecomMarkdownMaxBytes-len(suffix)) + suffix
				}

				return map[string]any{
					"msgtype": "markdown",
					"markdown": map[string]string{
						"content": content,
					},
				}
			},
		})
	}

	n.client.UseServices(httpService)
	return nil
}
