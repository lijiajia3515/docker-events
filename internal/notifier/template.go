package notifier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/filippofinke/docker-events/internal/docker"
)

type templateData struct {
	Event       docker.Event
	Events      []docker.Event
	ShortID     string
	Name        string
	Logs        string
	Time        string
	containerID string
	logLines    int64
	dockerCli   *client.Client
	isGrouped   bool
}

// EventCount 返回事件数量（用于分组事件）
func (t *templateData) EventCount() int {
	if t.isGrouped {
		return len(t.Events)
	}
	return 1
}

// Type 返回事件类型
func (t *templateData) Type() string {
	return t.Event.Type
}

// Action 返回事件动作
func (t *templateData) Action() string {
	return t.Event.Action
}

// ID 返回完整 ID
func (t *templateData) ID() string {
	return t.Event.ID
}

// Status 返回事件状态
func (t *templateData) Status() string {
	return t.Event.Status
}

// From 返回事件来源字段
func (t *templateData) From() string {
	return t.Event.From
}

// Scope 返回事件范围
func (t *templateData) Scope() string {
	return t.Event.Scope
}

// Actor 返回执行者
func (t *templateData) Actor() docker.Actor {
	return t.Event.Actor
}

// Attribute 返回特定属性值
func (t *templateData) Attribute(key string) string {
	if t.Event.Actor.Attributes != nil {
		return t.Event.Actor.Attributes[key]
	}
	return ""
}

// GetLogs 获取容器日志（如果可用）
func (t *templateData) GetLogs() string {
	if t.Logs != "" {
		return t.Logs
	}

	if t.dockerCli == nil || t.containerID == "" || t.logLines <= 0 {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", t.logLines),
	}

	logs, err := t.dockerCli.ContainerLogs(ctx, t.containerID, options)
	if err != nil {
		return fmt.Sprintf("[获取日志失败: %v]", err)
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	_, _ = io.Copy(buf, logs)

	t.Logs = strings.TrimSpace(buf.String())
	return t.Logs
}

func formatEventWithTemplate(tmplStr string, event docker.Event, dockerCli *client.Client, logLines int) (string, string, error) {
	// 从属性中提取容器名称
	containerName := ""
	containerID := ""

	if event.Type == "container" {
		if name, ok := event.Actor.Attributes["name"]; ok {
			containerName = name
		}
		containerID = event.Actor.ID
	}

	// 创建短 ID（前12个字符）
	shortID := event.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	data := &templateData{
		Event:       event,
		ShortID:     shortID,
		Name:        containerName,
		Time:        event.Timestamp.Format(time.RFC3339),
		containerID: containerID,
		logLines:    int64(logLines),
		dockerCli:   dockerCli,
	}

	// 解析模板
	tmpl, err := template.New("message").Parse(tmplStr)
	if err != nil {
		return "", "", fmt.Errorf("解析模板失败: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("执行模板失败: %w", err)
	}

	return strings.TrimSpace(buf.String()), "", nil
}

func formatGroupedEventsWithTemplate(tmplStr string, events []docker.Event, dockerCli *client.Client, logLines int) (string, string, error) {
	if len(events) == 0 {
		return "", "", fmt.Errorf("没有可格式化的事件")
	}

	if len(events) == 1 {
		return formatEventWithTemplate(tmplStr, events[0], dockerCli, logLines)
	}

	// 使用第一个事件作为模板数据的主事件
	firstEvent := events[0]

	// 从属性中提取容器名称
	containerName := ""
	containerID := ""

	if firstEvent.Type == "container" {
		if name, ok := firstEvent.Actor.Attributes["name"]; ok {
			containerName = name
		}
		containerID = firstEvent.Actor.ID
	}

	// 创建短 ID（前12个字符）
	shortID := firstEvent.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	data := &templateData{
		Event:       firstEvent,
		Events:      events,
		ShortID:     shortID,
		Name:        containerName,
		Time:        firstEvent.Timestamp.Format(time.RFC3339),
		containerID: containerID,
		logLines:    int64(logLines),
		dockerCli:   dockerCli,
		isGrouped:   true,
	}

	// 解析模板
	tmpl, err := template.New("message").Parse(tmplStr)
	if err != nil {
		return "", "", fmt.Errorf("解析模板失败: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", fmt.Errorf("执行模板失败: %w", err)
	}

	return strings.TrimSpace(buf.String()), "", nil
}
