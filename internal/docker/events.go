package docker

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type Watcher struct {
	client  *client.Client
	options events.ListOptions
	logger  *slog.Logger
}

func NewWatcher(filtersList, eventTypes []string, logger *slog.Logger) (*Watcher, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("创建 Docker 客户端失败: %w", err)
	}

	filterArgs, err := buildFilterArgs(filtersList, eventTypes)
	if err != nil {
		_ = cli.Close()
		return nil, err
	}

	return &Watcher{
		client: cli,
		options: events.ListOptions{
			Filters: filterArgs,
		},
		logger: logger,
	}, nil
}

func (w *Watcher) Client() *client.Client {
	return w.client
}

func (w *Watcher) Watch(ctx context.Context, handle func(context.Context, Event) error) error {
	defer w.client.Close()

	eventsCh, errsCh := w.client.Events(ctx, w.options)

	for eventsCh != nil || errsCh != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errsCh:
			if !ok {
				errsCh = nil
				continue
			}
			if err != nil {
				return fmt.Errorf("Docker 事件流错误: %w", err)
			}
		case msg, ok := <-eventsCh:
			if !ok {
				eventsCh = nil
				continue
			}
			event := convertMessage(msg)
			if err := handle(ctx, event); err != nil {
				w.logger.Error("处理 Docker 事件失败", "error", err, "type", event.Type, "action", event.Action, "id", event.ID)
			}
		}
	}

	return nil
}
