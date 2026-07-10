# MESSAGE_TEMPLATE 配置示例

# 1. 简单格式 - 显示容器名称和动作

MESSAGE_TEMPLATE="容器 {{.Name}} {{.Action}}"

# 输出: 容器 my-app-1 start

# 2. 带时间戳和短 ID

MESSAGE_TEMPLATE="[{{.Time}}] {{.Type}} {{.Action}}: {{.Name}} ({{.ShortID}})"

# 输出: [2025-10-09T07:37:05Z] container start: my-app-1 (cd280ca744b1)

# 3. 详细格式 - 显示镜像和属性

MESSAGE_TEMPLATE="容器: {{.Name}}\n动作: {{.Action}}\n镜像: {{.From}}\n项目: {{.Attribute \"com.docker.compose.project\"}}\n时间: {{.Time}}"

# 输出:

# 容器: my-app-1

# 动作: start

# 镜像: my-app-image

# 项目: my-project

# 时间: 2025-10-09T07:37:05Z

# 4. 带容器日志（最近 20 行）

MESSAGE_TEMPLATE="容器 {{.Name}} {{.Action}}\n\n日志:\n{{.GetLogs}}"
MESSAGE_LOG_LINES=20

# 输出:

# 容器 my-app-1 start

#

# 日志:

# [2025-10-09 07:37:05] Application starting...

# [2025-10-09 07:37:06] Server listening on port 8080

# ...

# 5. 极简格式

MESSAGE_TEMPLATE="{{.Type}}/{{.Action}}: {{if .Name}}{{.Name}}{{else}}{{.ShortID}}{{end}}"

# 输出: container/start: my-app-1

# 6. Slack/Discord 友好格式（带表情符号）

MESSAGE_TEMPLATE="🐳 {{.Type}} {{.Action}}\n📦 **容器:** {{.Name}}\n🖼️ **镜像:** {{.From}}\n⏰ **时间:** {{.Time}}"

# 输出:

# 🐳 container start

# 📦 **容器:** my-app-1

# 🖼️ **镜像:** my-app-image

# ⏰ **时间:** 2025-10-09T07:37:05Z

# 7. 条件格式（有名称时显示名称，否则显示短 ID）

MESSAGE_TEMPLATE="{{if .Name}}{{.Name}}{{else}}{{.ShortID}}{{end}} - {{.Action}} ({{.Status}})"

# 输出: my-app-1 - start (start)

# 8. 完整详情 + 日志（用于故障排查）

MESSAGE_TEMPLATE="事件: {{.Type}}/{{.Action}}\n容器: {{.Name}} ({{.ShortID}})\n镜像: {{.From}}\n状态: {{.Status}}\n时间: {{.Time}}\n\n最近日志:\n{{.GetLogs}}"
MESSAGE_LOG_LINES=50

# 输出:

# 事件: container/start

# 容器: my-app-1 (cd280ca744b1)

# 镜像: my-app-image

# 状态: start

# 时间: 2025-10-09T07:37:05Z

#

# 最近日志:

# [日志内容...]

# 9. 告警风格（用于监控）

MESSAGE_TEMPLATE="⚠️ 告警: 容器 {{.Name}} 已 {{.Action}}\n时间戳: {{.Time}}\nID: {{.ShortID}}"

# 输出:

# ⚠️ 告警: 容器 my-app-1 已 stop

# 时间戳: 2025-10-09T07:37:05Z

# ID: cd280ca744b1

# 10. 类 JSON 格式

MESSAGE_TEMPLATE="{\"type\": \"{{.Type}}\", \"action\": \"{{.Action}}\", \"name\": \"{{.Name}}\", \"time\": \"{{.Time}}\"}"

# 输出: {"type": "container", "action": "start", "name": "my-app-1", "time": "2025-10-09T07:37:05Z"}
