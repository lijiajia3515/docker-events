package notifier

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/filippofinke/docker-events/internal/docker"
)

// formatEvent 格式化单个 Docker 事件，返回通知主题和正文
func formatEvent(subjectPrefix string, event docker.Event) (string, string) {
	prefix := strings.TrimSpace(subjectPrefix)
	if prefix == "" {
		prefix = "Docker 事件"
	}

	subject := fmt.Sprintf("%s: %s %s", prefix, event.Type, event.Action)
	if event.Actor.ID != "" {
		subject = fmt.Sprintf("%s (%s)", subject, event.Actor.ID)
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf("🕐 时间: %s\n", event.Timestamp.Format(time.RFC3339)))
	if event.Status != "" {
		body.WriteString(fmt.Sprintf("📋 状态: %s\n", event.Status))
	}
	if event.From != "" {
		body.WriteString(fmt.Sprintf("📦 来源: %s\n", event.From))
	}
	if event.Scope != "" {
		body.WriteString(fmt.Sprintf("🌐 范围: %s\n", event.Scope))
	}
	if event.ID != "" {
		body.WriteString(fmt.Sprintf("🔖 ID: %s\n", event.ID))
	}
	if event.Actor.ID != "" {
		body.WriteString(fmt.Sprintf("👤 执行者: %s\n", event.Actor.ID))
	}

	if len(event.Actor.Attributes) > 0 {
		body.WriteString("🏷️ 属性:\n")
		keys := make([]string, 0, len(event.Actor.Attributes))
		for key := range event.Actor.Attributes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			body.WriteString(fmt.Sprintf("  - %s=%s\n", key, event.Actor.Attributes[key]))
		}
	}

	return subject, strings.TrimSpace(body.String())
}

// formatGroupedEvents 格式化多个分组的 Docker 事件，返回通知主题和正文
func formatGroupedEvents(subjectPrefix string, events []docker.Event) (string, string) {
	if len(events) == 0 {
		return "", ""
	}

	if len(events) == 1 {
		return formatEvent(subjectPrefix, events[0])
	}

	prefix := strings.TrimSpace(subjectPrefix)
	if prefix == "" {
		prefix = "Docker 事件"
	}

	// 从第一个事件获取容器 ID/执行者
	containerID := events[0].ID
	if containerID == "" {
		containerID = events[0].Actor.ID
	}

	// 收集去重的操作类型
	actions := make(map[string]bool)
	for _, event := range events {
		actions[event.Action] = true
	}
	actionList := make([]string, 0, len(actions))
	for action := range actions {
		actionList = append(actionList, action)
	}
	sort.Strings(actionList)

	subject := fmt.Sprintf("🐳 %s: 容器 %s 共 %d 个事件 (%s)", prefix, containerID[:12], len(events), strings.Join(actionList, ", "))

	var body strings.Builder
	body.WriteString(fmt.Sprintf("📦 容器: %s\n", containerID))
	body.WriteString(fmt.Sprintf("🔢 事件数量: %d\n", len(events)))
	body.WriteString(fmt.Sprintf("📅 时间范围: %s 至 %s\n\n",
		events[0].Timestamp.Format(time.RFC3339),
		events[len(events)-1].Timestamp.Format(time.RFC3339)))

	// 获取公共属性
	commonAttrs := make(map[string]string)
	if len(events[0].Actor.Attributes) > 0 {
		// 以第一个事件的属性为初始值
		maps.Copy(commonAttrs, events[0].Actor.Attributes)

		// 仅保留所有事件中相同的属性
		for _, event := range events[1:] {
			for k, v := range commonAttrs {
				if eventV, ok := event.Actor.Attributes[k]; !ok || eventV != v {
					delete(commonAttrs, k)
				}
			}
		}
	}

	if len(commonAttrs) > 0 {
		body.WriteString("🔗 公共属性:\n")
		keys := make([]string, 0, len(commonAttrs))
		for key := range commonAttrs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			body.WriteString(fmt.Sprintf("  - %s=%s\n", key, commonAttrs[key]))
		}
		body.WriteString("\n")
	}

	body.WriteString("📝 事件列表:\n")
	for i, event := range events {
		body.WriteString(fmt.Sprintf("  %d. [%s] %s %s",
			i+1,
			event.Timestamp.Format("15:04:05"),
			event.Type,
			event.Action))

		if event.Status != "" && event.Status != event.Action {
			body.WriteString(fmt.Sprintf(" (📋 状态: %s)", event.Status))
		}
		body.WriteString("\n")
	}

	return subject, strings.TrimSpace(body.String())
}
