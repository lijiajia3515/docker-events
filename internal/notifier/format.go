package notifier

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/filippofinke/docker-events/internal/docker"
)

// ansiPattern 匹配 ANSI 转义序列（终端颜色、光标控制等）
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI 移除字符串中的 ANSI 转义序列
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// stripDockerLogHeaders 剥离 Docker ContainerLogs 流中每帧前的 8 字节二进制头
// 格式：[1B stream type][3B zero][4B big-endian size][payload]
func stripDockerLogHeaders(raw []byte) string {
	var result strings.Builder
	buf := raw
	for len(buf) >= 8 {
		// 读取帧大小（字节 4-7，big-endian uint32）
		frameSize := int(binary.BigEndian.Uint32(buf[4:8]))
		buf = buf[8:]
		if frameSize > len(buf) {
			frameSize = len(buf)
		}
		result.Write(buf[:frameSize])
		buf = buf[frameSize:]
	}
	// 剩余不足 8 字节的内容直接追加（可能是无头的纯文本）
	if len(buf) > 0 {
		result.Write(buf)
	}
	return result.String()
}

// fetchContainerLogs 获取容器日志，仅在 container 类型事件且 logLines > 0 时拉取
func fetchContainerLogs(dockerCli *dockerclient.Client, containerID string, event docker.Event, logLines int) string {
	if logLines <= 0 || dockerCli == nil || containerID == "" {
		return ""
	}
	if event.Type != "container" {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", logLines),
	}

	logs, err := dockerCli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		return fmt.Sprintf("[获取日志失败: %v]", err)
	}
	defer logs.Close()

	raw, err := io.ReadAll(logs)
	if err != nil {
		return fmt.Sprintf("[读取日志失败: %v]", err)
	}

	return strings.TrimSpace(stripANSI(stripDockerLogHeaders(raw)))
}

// formatEvent 格式化单个 Docker 事件，返回通知主题和正文
func formatEvent(subjectPrefix string, event docker.Event, dockerCli *dockerclient.Client, logLines int) (string, string) {
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

	// 拉取容器日志
	containerID := event.Actor.ID
	if containerID == "" {
		containerID = event.ID
	}
	if logs := fetchContainerLogs(dockerCli, containerID, event, logLines); logs != "" {
		body.WriteString(fmt.Sprintf("\n📝 日志:\n%s\n", logs))
	}

	return subject, strings.TrimSpace(body.String())
}

// formatGroupedEvents 格式化多个分组的 Docker 事件，返回通知主题和正文
func formatGroupedEvents(subjectPrefix string, events []docker.Event, dockerCli *dockerclient.Client, logLines int) (string, string) {
	if len(events) == 0 {
		return "", ""
	}

	if len(events) == 1 {
		return formatEvent(subjectPrefix, events[0], dockerCli, logLines)
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

		// 显示每个事件相对于公共属性的差异属性（最多 5 个，避免消息过长）
		if len(event.Actor.Attributes) > 0 {
			diffKeys := make([]string, 0)
			for key, val := range event.Actor.Attributes {
				if commonVal, ok := commonAttrs[key]; !ok || commonVal != val {
					diffKeys = append(diffKeys, key)
				}
			}
			sort.Strings(diffKeys)
			maxDiff := 5
			for j, key := range diffKeys {
				if j >= maxDiff {
					body.WriteString(fmt.Sprintf("     ...还有 %d 个属性\n", len(diffKeys)-maxDiff))
					break
				}
				body.WriteString(fmt.Sprintf("     %s=%s\n", key, event.Actor.Attributes[key]))
			}
		}
	}

	// 拉取第一个事件的容器日志
	if logs := fetchContainerLogs(dockerCli, containerID, events[0], logLines); logs != "" {
		body.WriteString(fmt.Sprintf("\n📝 日志:\n%s\n", logs))
	}

	return subject, strings.TrimSpace(body.String())
}
