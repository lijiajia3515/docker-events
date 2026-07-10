package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/filippofinke/docker-events/internal/config"
	"github.com/filippofinke/docker-events/internal/docker"
	"github.com/filippofinke/docker-events/internal/logging"
	"github.com/filippofinke/docker-events/internal/notifier"
)

func Run(ctx context.Context, logOut io.Writer) error {
	logger := logging.NewLogger(logOut)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		return fmt.Errorf("加载配置失败: %w", err)
	}

	n := notifier.NewNotifier(logger)
	if err := n.Setup(cfg); err != nil {
		logger.Error("配置通知器失败", "error", err)
		return fmt.Errorf("初始化通知器失败: %w", err)
	}

	watcher, err := docker.NewWatcher(cfg.DockerFilters, cfg.DockerEventTypes, logger)
	if err != nil {
		logger.Error("创建 Docker 监听器失败", "error", err)
		return fmt.Errorf("创建监听器失败: %w", err)
	}

	n.SetDockerClient(watcher.Client())

	grouper := notifier.NewEventGrouper(n, cfg)
	defer grouper.Shutdown()

	logger.Info("启动 Docker 事件监听器", "filters", cfg.DockerFilters, "types", cfg.DockerEventTypes)

	err = watcher.Watch(ctx, func(ctx context.Context, event docker.Event) error {
		attrs := make([]any, 0, 10)
		attrs = append(attrs, "type", event.Type, "action", event.Action, "status", event.Status, "id", event.ID)
		if event.Actor.ID != "" {
			attrs = append(attrs, "actor", event.Actor.ID)
		}
		attrs = append(attrs, "timestamp", event.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
		logger.Info("Docker 事件", attrs...)
		return grouper.HandleEvent(ctx, event)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info("Docker 事件监听器已停止", "原因", "上下文已取消")
			return nil
		}
		logger.Error("监听 Docker 事件失败", "error", err)
		return err
	}

	return nil
}
