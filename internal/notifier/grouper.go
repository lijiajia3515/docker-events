package notifier

import (
	"context"
	"sync"
	"time"

	"github.com/filippofinke/docker-events/internal/config"
	"github.com/filippofinke/docker-events/internal/docker"
)

type EventGrouper struct {
	notifier      Notifier
	cfg           *config.Config
	mu            sync.Mutex
	groups        map[string]*eventGroup
	windowDur     time.Duration
	flushCallback func(containerID string, events []docker.Event)
}

type eventGroup struct {
	containerID string
	events      []docker.Event
	timer       *time.Timer
}

func NewEventGrouper(notifier Notifier, cfg *config.Config) *EventGrouper {
	eg := &EventGrouper{
		notifier:  notifier,
		cfg:       cfg,
		groups:    make(map[string]*eventGroup),
		windowDur: cfg.EventGroupWindow,
	}

	eg.flushCallback = func(containerID string, events []docker.Event) {
		eg.flushGroup(events)
	}

	return eg
}

func (eg *EventGrouper) HandleEvent(ctx context.Context, event docker.Event) error {
	if eg.windowDur <= 0 {
		// 分组功能已禁用，立即发送
		return eg.notifier.NotifyEvent(ctx, eg.cfg, event)
	}

	containerID := event.ID
	if containerID == "" {
		containerID = event.Actor.ID
	}

	eg.mu.Lock()
	defer eg.mu.Unlock()

	group, exists := eg.groups[containerID]
	if !exists {
		// 创建新分组
		group = &eventGroup{
			containerID: containerID,
			events:      []docker.Event{event},
		}
		eg.groups[containerID] = group

		// 设置定时器，在窗口时间后刷新此分组
		group.timer = time.AfterFunc(eg.windowDur, func() {
			eg.mu.Lock()
			grp := eg.groups[containerID]
			if grp != nil {
				events := grp.events
				delete(eg.groups, containerID)
				eg.mu.Unlock()
				eg.flushCallback(containerID, events)
			} else {
				eg.mu.Unlock()
			}
		})
	} else {
		// 添加到现有分组
		group.events = append(group.events, event)
		// 重置定时器以延长窗口
		if group.timer != nil {
			group.timer.Stop()
			group.timer = time.AfterFunc(eg.windowDur, func() {
				eg.mu.Lock()
				grp := eg.groups[containerID]
				if grp != nil {
					events := grp.events
					delete(eg.groups, containerID)
					eg.mu.Unlock()
					eg.flushCallback(containerID, events)
				} else {
					eg.mu.Unlock()
				}
			})
		}
	}

	return nil
}

func (eg *EventGrouper) flushGroup(events []docker.Event) {
	if len(events) == 0 {
		return
	}

	ctx := context.Background()

	if len(events) == 1 {
		// 单个事件，正常发送
		_ = eg.notifier.NotifyEvent(ctx, eg.cfg, events[0])
	} else {
		// 多个事件，分组发送
		_ = eg.notifier.NotifyGroupedEvents(ctx, eg.cfg, events)
	}
}

func (eg *EventGrouper) Shutdown() {
	eg.mu.Lock()
	defer eg.mu.Unlock()

	for containerID, group := range eg.groups {
		if group.timer != nil {
			group.timer.Stop()
		}
		eg.flushCallback(containerID, group.events)
	}

	eg.groups = make(map[string]*eventGroup)
}
