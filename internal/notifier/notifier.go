package notifier

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nikoksr/notify"

	"github.com/filippofinke/docker-events/internal/config"
	"github.com/filippofinke/docker-events/internal/docker"

	"log/slog"

	dockerclient "github.com/docker/docker/client"
)

type Notifier interface {
	Setup(cfg *config.Config) error
	NotifyEvent(ctx context.Context, cfg *config.Config, event docker.Event) error
	NotifyGroupedEvents(ctx context.Context, cfg *config.Config, events []docker.Event) error
	SetDockerClient(cli *dockerclient.Client)
}

type notifierImpl struct {
	client    *notify.Notify
	logger    *slog.Logger
	dockerCli *dockerclient.Client
}

func NewNotifier(logger *slog.Logger) Notifier {
	return &notifierImpl{
		client: notify.New(),
		logger: logger,
	}
}

func (n *notifierImpl) SetDockerClient(cli *dockerclient.Client) {
	n.dockerCli = cli
}

func (n *notifierImpl) Setup(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("配置为空")
	}

	if cfg.Slack.Enabled {
		if err := n.addSlack(cfg.Slack); err != nil {
			return fmt.Errorf("初始化 Slack 失败: %w", err)
		}
	}

	if cfg.Telegram.Enabled {
		if err := n.addTelegram(cfg.Telegram); err != nil {
			return fmt.Errorf("初始化 Telegram 失败: %w", err)
		}
	}

	if cfg.Discord.Enabled {
		if err := n.addDiscord(cfg.Discord); err != nil {
			return fmt.Errorf("初始化 Discord 失败: %w", err)
		}
	}

	if cfg.Teams.Enabled {
		if err := n.addTeams(cfg.Teams); err != nil {
			return fmt.Errorf("初始化 Teams 失败: %w", err)
		}
	}

	if cfg.WeChatWork.Enabled {
		if err := n.addWeChatWork(cfg.WeChatWork); err != nil {
			return fmt.Errorf("初始化企业微信失败: %w", err)
		}
	}

	if cfg.DingTalk.Enabled {
		if err := n.addDingTalk(cfg.DingTalk); err != nil {
			return fmt.Errorf("初始化钉钉失败: %w", err)
		}
	}

	return nil
}

func (n *notifierImpl) NotifyEvent(ctx context.Context, cfg *config.Config, event docker.Event) error {
	if cfg == nil {
		return fmt.Errorf("配置为空")
	}

	var subject, body string
	var err error

	if cfg.MessageTemplate != "" {
		body, _, err = formatEventWithTemplate(cfg.MessageTemplate, event, n.dockerCli, cfg.LogLines)
		if err != nil {
			n.logger.Warn("使用模板格式化事件失败，回退到默认格式", "error", err)
			subject, body = formatEvent(cfg.NotifySubject, event, n.dockerCli, cfg.LogLines)
		} else {
			subject = fmt.Sprintf("%s: %s %s", cfg.NotifySubject, event.Type, event.Action)
		}
	} else {
		subject, body = formatEvent(cfg.NotifySubject, event, n.dockerCli, cfg.LogLines)
	}

	if err := n.client.Send(ctx, subject, body); err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}
	return nil
}

func (n *notifierImpl) NotifyGroupedEvents(ctx context.Context, cfg *config.Config, events []docker.Event) error {
	if cfg == nil {
		return fmt.Errorf("配置为空")
	}

	if len(events) == 0 {
		return nil
	}

	if len(events) == 1 {
		return n.NotifyEvent(ctx, cfg, events[0])
	}

	var subject, body string
	var err error

	if cfg.MessageTemplate != "" {
		body, _, err = formatGroupedEventsWithTemplate(cfg.MessageTemplate, events, n.dockerCli, cfg.LogLines)
		if err != nil {
			n.logger.Warn("使用模板格式化分组事件失败，回退到默认格式", "error", err)
			subject, body = formatGroupedEvents(cfg.NotifySubject, events, n.dockerCli, cfg.LogLines)
		} else {
			// 从事件操作中生成主题
			actions := make(map[string]bool)
			for _, event := range events {
				actions[event.Action] = true
			}
			actionList := make([]string, 0, len(actions))
			for action := range actions {
				actionList = append(actionList, action)
			}
			sort.Strings(actionList)
			subject = fmt.Sprintf("%s: %d 个事件 (%s)", cfg.NotifySubject, len(events), strings.Join(actionList, ", "))
		}
	} else {
		subject, body = formatGroupedEvents(cfg.NotifySubject, events, n.dockerCli, cfg.LogLines)
	}

	if err := n.client.Send(ctx, subject, body); err != nil {
		return fmt.Errorf("发送分组通知失败: %w", err)
	}
	return nil
}
