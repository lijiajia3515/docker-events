<div align="center">
  <a href="https://github.com/filippofinke/docker-events">
    <img width="200px" src="https://github.com/user-attachments/assets/e9712d6a-32e9-4e9b-a545-11aa88dadf65" alt="Docker Events" />
  </a>
  <h3 align="center">Docker Events</h3>
</div>

> 实时监控 Docker 事件，通过轻量级 Go 服务发送丰富的通知消息。

本项目监听 Docker 系统事件，并通过 [`nikoksr/notify`](https://github.com/nikoksr/notify) 驱动的可配置通知渠道转发有价值的事件摘要。项目设计小巧、可靠，易于扩展新的传输方式或事件处理规则。

## 功能特性

- [x] 通过托管的监听器实时流式接收 `docker system events`
- [x] 包含上下文信息（时间戳、执行者属性、状态）的友好通知
- [x] 多渠道投递（Slack、Telegram、Discord、Teams、企业微信、钉钉），由 `github.com/nikoksr/notify` 驱动
- [x] 支持按 Docker 事件类型和 CLI 过滤器进行可选过滤
- [x] 环境变量驱动的配置，启动时自动验证
- [x] 可组合的代码结构（config、watcher、notifier）和格式化辅助函数的单元测试

## 快速开始

前置条件

- Go 1.24+
- Docker CLI 并能访问目标 Docker 守护进程
- 至少配置一个通知渠道（Slack bot token + 频道 ID、Telegram bot token + 聊天 ID、Discord bot token + 频道 ID、Discord webhook URL、Microsoft Teams webhook URL、企业微信 webhook URL 或钉钉 webhook URL）

克隆并安装依赖

```bash
git clone https://github.com/filippofinke/docker-events.git
cd docker-events
go mod tidy
```

配置环境

1. 复制示例环境文件：

```bash
cp .env.example .env
```

2. 在 `.env` 中填写 Slack token、目标频道以及其他可选的 Docker 过滤器。

本地运行

```bash
# 启动监听器（模块感知路径）
go run ./cmd
```

服务会将日志输出到标准输出，并为匹配的 Docker 事件发送通知。按 `Ctrl+C` 停止。

## 配置说明

所有设置通过环境变量配置（参见 `.env.example`）。主要选项：

- `SLACK_BOT_TOKEN`: 用于通知器认证的 Slack bot token。
- `SLACK_CHANNEL_IDS`: 逗号分隔的 Slack 频道 ID 列表（如 `C0123456,C0ABCDEF`）。
- `TELEGRAM_BOT_TOKEN`: 通过 [BotFather](https://core.telegram.org/bots#6-botfather) 创建的 Telegram bot token。
- `TELEGRAM_CHAT_IDS`: 逗号分隔的聊天 ID 列表（支持负数表示群聊）。
- `DISCORD_BOT_TOKEN`: 从开发者门户生成的 Discord bot token。
- `DISCORD_CHANNEL_IDS`: 逗号分隔的 Discord 频道 ID 列表。
- `DISCORD_WEBHOOK_URLS`: 逗号分隔的 Discord webhook URL 列表（推荐用于简单通知，替代 bot token）。
- `TEAMS_WEBHOOK_URLS`: 逗号分隔的 Microsoft Teams webhook URL 列表（使用 Adaptive Cards）。
- `WECOM_WEBHOOK_URLS`: 逗号分隔的企业微信 webhook URL 列表。
- `DINGTALK_WEBHOOK_URLS`: 逗号分隔的钉钉 webhook URL 列表。
- `NOTIFY_SUBJECT_PREFIX`: 通知主题前缀（默认为 `Docker 事件`）。
- `MESSAGE_TEMPLATE`: 使用 Go 模板语法的自定义消息模板（参见下方[消息自定义](#消息自定义)）。
- `MESSAGE_LOG_LINES`: 为事件获取的容器日志行数（默认为 0，禁用）。
- `EVENT_GROUP_WINDOW`: 同一容器事件分组的时间窗口（如 `5s`、`10s`、`1m`）。同一容器在此窗口内的事件将合并为一条通知。设为 `0` 禁用分组（默认 `5s`）。
- `DOCKER_EVENT_FILTERS`: 传递给 `docker system events` 的逗号分隔过滤器（与 CLI `--filter` 参数语法相同，如 `status=start,type=container`）。
- `DOCKER_EVENT_TYPES`: 逗号分隔的 Docker 事件类型列表（如 `container,image,volume`）。

> **安全提示：** 请勿提交包含真实 token 的 `.env` 文件。请在本地使用 `.env` 或通过编排工具提供变量。

## Docker 事件过滤器

Docker CLI 支持丰富的过滤器，可在 `DOCKER_EVENT_FILTERS` 中组合使用。支持的过滤键包括：

- `config=<名称或 ID>`
- `container=<名称或 ID>`
- `daemon=<名称或 ID>`
- `event=<事件动作>`
- `image=<仓库或标签>`
- `label=<键>` 或 `label=<键>=<值>`
- `network=<名称或 ID>`
- `node=<ID>`
- `plugin=<名称或 ID>`
- `scope=<local 或 swarm>`
- `secret=<名称或 ID>`
- `service=<名称或 ID>`
- `type=<container|image|volume|network|daemon|plugin|service|node|secret|config>`
- `volume=<名称>`

通过逗号分隔多个过滤条目（如 `DOCKER_EVENT_FILTERS=event=start,scope=swarm`）；服务会将每个条目转换为 `docker system events` 的独立 `--filter` 参数。

更多详情请参阅 [Docker 文档](https://docs.docker.com/reference/cli/docker/system/events/#filter)。

## Docker 事件类型

`DOCKER_EVENT_TYPES` 用于限定处理的 Docker 对象类型。有效值：

- `container`
- `image`
- `plugin`
- `volume`
- `network`
- `daemon`
- `service`
- `node`
- `secret`
- `config`

将该变量留空即可接受事件流中的所有类型。

更多详情请参阅 [Docker 文档](https://docs.docker.com/engine/reference/commandline/system_events/#object-types)。

## 事件分组

为防止容器在短时间内经历多个事件时产生通知轰炸（如重启期间：kill → stop → die → start → restart），docker-events 可以在可配置的时间窗口内将同一容器的事件分组。

### 工作原理

启用事件分组后（默认：5 秒），服务将：

1. 收集同一容器在时间窗口内的所有事件
2. 等待窗口过期或收到不同容器的事件
3. 发送一条包含所有事件的分组通知，而非单独的消息

### 配置

设置 `EVENT_GROUP_WINDOW` 环境变量来控制分组窗口：

```bash
EVENT_GROUP_WINDOW=5s   # 默认：5 秒内的事件分组
EVENT_GROUP_WINDOW=10s  # 10 秒内的事件分组
EVENT_GROUP_WINDOW=1m   # 1 分钟内的事件分组
EVENT_GROUP_WINDOW=0    # 禁用分组，立即发送所有事件
```

有效时间单位：`ns`、`us`（或 `µs`）、`ms`、`s`、`m`、`h`

### 分组通知格式

多个事件被分组时，通知将包含：

- 容器 ID 和事件总数
- 时间范围（第一个到最后一个事件）
- 所有事件共享的公共属性
- 所有事件列表及其时间戳和操作

分组通知示例：

```
Docker 事件: 容器 d23c731f32ba 共 5 个事件 (die, kill, restart, start, stop)

容器: d23c731f32ba41defa48b2804299e9378b84442857701b1d51b8e6aca77c35da
事件数量: 5
时间范围: 2025-10-09T08:10:58Z 至 2025-10-09T08:10:59Z

公共属性:
  - com.docker.compose.project=myapp
  - com.docker.compose.service=web
  - image=nginx:latest

事件列表:
  1. [08:10:58] container kill
  2. [08:10:59] container stop
  3. [08:10:59] container die
  4. [08:10:59] container start
  5. [08:10:59] container restart
```

### 优势

- **减少通知轰炸**：重启操作通常会产生 5+ 个事件，现在合并为一条消息
- **更好的上下文**：在同一消息中查看所有相关事件及其时间
- **更整洁的通知渠道**：减少需要翻阅的消息
- **保留完整信息**：所有事件信息均被保留，只是组织得更好

### 行为说明

- 单个事件会立即发送（无分组开销）
- 不同容器的事件永远不会被分组
- 每当同一容器有新事件到达时，计时器会重置
- 关闭时，所有待处理的分组事件会立即刷新
- **完全支持自定义模板**：如果配置了 `MESSAGE_TEMPLATE`，它将同时用于单个和分组事件。在模板中使用 `{{.EventCount}}` 和 `{{.Events}}` 来区分处理分组事件。

## 消息自定义

默认情况下，docker-events 会发送包含所有可用事件信息的详细通知。你可以通过 `MESSAGE_TEMPLATE` 环境变量使用 Go 模板自定义消息格式。

### 可用模板占位符

- `{{.Type}}` - 事件类型（container、image、volume、network 等）
- `{{.Action}}` - 事件动作（start、stop、create、destroy 等）
- `{{.ID}}` - 完整对象 ID
- `{{.ShortID}}` - 短 ID（前 12 个字符）
- `{{.Name}}` - 容器/对象名称（可用时从属性中提取）
- `{{.Status}}` - 事件状态
- `{{.From}}` - 来源字段（容器事件通常为镜像名称）
- `{{.Time}}` - RFC3339 格式的事件时间戳
- `{{.Scope}}` - 事件范围（local 或 swarm）
- `{{.Actor.ID}}` - 执行者 ID
- `{{.Attribute "key"}}` - 按键获取特定属性值
- `{{.GetLogs}}` - 获取容器日志（需要 `MESSAGE_LOG_LINES` > 0）
- `{{.EventCount}}` - 事件数量（单个事件返回 1，分组事件返回 >1）
- `{{.Events}}` - 所有事件的数组（仅分组事件可用；配合 range 使用）

**注意：** 当事件被分组时（同一容器的多个事件），`{{.Type}}`、`{{.Action}}` 等引用的是分组中的第一个事件。使用 `{{.Events}}` 可以访问分组中的所有事件。

### 模板示例

**简单通知：**

```bash
MESSAGE_TEMPLATE="容器 {{.Name}} ({{.ShortID}}) {{.Action}} 时间: {{.Time}}"
```

**带容器日志：**

```bash
MESSAGE_TEMPLATE="容器 {{.Name}} {{.Action}}\n镜像: {{.From}}\n日志:\n{{.GetLogs}}"
MESSAGE_LOG_LINES=20
```

**自定义属性：**

```bash
MESSAGE_TEMPLATE="{{.Type}} {{.Action}}: {{.Name}}\n项目: {{.Attribute \"com.docker.compose.project\"}}\n服务: {{.Attribute \"com.docker.compose.service\"}}"
```

**条件格式化：**

```bash
MESSAGE_TEMPLATE="{{.Type}} {{.Action}}: {{if .Name}}{{.Name}}{{else}}{{.ShortID}}{{end}}\n时间: {{.Time}}"
```

**分组事件 + 自定义模板：**

```bash
MESSAGE_TEMPLATE="{{if gt .EventCount 1}}🔄 {{.Name}} ({{.ShortID}}) 共 {{.EventCount}} 个事件\n{{range .Events}}- [{{.Timestamp.Format \"15:04:05\"}}] {{.Action}}\n{{end}}{{else}}{{.Type}} {{.Action}}: {{.Name}}\n时间: {{.Time}}{{end}}"
EVENT_GROUP_WINDOW=5s
```

**分组事件 + 日志：**

```bash
MESSAGE_TEMPLATE="容器: {{.Name}} ({{.ShortID}})\n事件数: {{.EventCount}}\n{{if gt .EventCount 1}}动作: {{range .Events}}{{.Action}} {{end}}\n{{end}}{{if .GetLogs}}\n日志:\n{{.GetLogs}}{{end}}"
MESSAGE_LOG_LINES=20
EVENT_GROUP_WINDOW=5s
```

### 日志配置

设置 `MESSAGE_LOG_LINES` 在使用 `{{.GetLogs}}` 时获取容器日志的最后 N 行：

```bash
MESSAGE_LOG_LINES=10  # 获取最后 10 行
MESSAGE_LOG_LINES=50  # 获取最后 50 行
MESSAGE_LOG_LINES=0   # 禁用日志获取（默认）
```

**注意：** 日志获取仅对容器事件有效，可能会增加通知延迟。请使用合理的行数以避免性能问题。

### 默认模板

如果未配置自定义模板，默认格式如下：

```
时间: <时间戳>
状态: <状态>
来源: <来源>
范围: <范围>
ID: <ID>
执行者: <执行者ID>
```

## Discord Webhook 与 Bot 的区别

本项目支持两种方式发送 Discord 通知：

**Discord Webhook（推荐）**

Discord webhook 是发送通知的最简单方式。它使用 HTTP POST 请求，无需 bot 会话或网关连接。

优势：

- 更简单的配置 - 只需在 Discord 频道设置中创建 webhook
- 无需 bot 权限或 OAuth 作用域
- 更高效 - 使用普通 HTTP 而非维护 WebSocket 连接
- 可通过提供多个 webhook URL 发送到多个频道

创建 Discord webhook 的步骤：

1. 打开 Discord 服务器，进入需要通知的频道
2. 点击频道名称旁的齿轮图标（编辑频道）
3. 进入 "Integrations" → "Webhooks" → "New Webhook"
4. 复制 webhook URL 并添加到 `DISCORD_WEBHOOK_URLS`

示例：

```bash
DISCORD_WEBHOOK_URLS=https://discord.com/api/webhooks/123456789/your-webhook-token
```

**Discord Bot（替代方案）**

如果需要更高级的功能或已有 bot 基础设施，可以使用 bot token。通过 `DISCORD_BOT_TOKEN` 和 `DISCORD_CHANNEL_IDS` 配置。

也可以同时使用 webhook 和 bot token。

## Microsoft Teams Webhook

Microsoft Teams 支持接收通知的 incoming webhook。本项目使用 **Adaptive Cards**（1.4 版）向 Teams 发送通知。

创建 Teams webhook 的步骤（使用 Workflows / Power Automate）：
1. 在 Microsoft Teams 中，进入要添加 webhook 的频道，选择 **Workflows**（或 **Integrations**）。
2. 搜索 **Post to a channel when a webhook request is received** 并选择。
3. 选择团队和频道，然后添加工作流。
4. 复制生成的 webhook URL 并添加到 `.env` 文件中的 `TEAMS_WEBHOOK_URLS`：
   ```bash
   TEAMS_WEBHOOK_URLS=https://your-tenant.webhook.office.com/webhookb2/your-webhook-token
   ```

## 企业微信 Webhook

企业微信支持通过群机器人 webhook 接收通知。本项目使用 markdown 消息格式发送通知。

配置方式：在 `.env` 文件中添加企业微信 webhook URL：

```bash
WECOM_WEBHOOK_URLS=https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-key
```

支持配置多个 webhook URL（逗号分隔）。

## 钉钉 Webhook

钉钉支持通过群机器人 webhook 接收通知。本项目使用 markdown 消息格式发送通知。

配置方式：在 `.env` 文件中添加钉钉 webhook URL：

```bash
DINGTALK_WEBHOOK_URLS=https://oapi.dingtalk.com/robot/send?access_token=your-token
```

支持配置多个 webhook URL（逗号分隔）。

## 扩展通知渠道

`internal/notifier` 封装了 `github.com/nikoksr/notify`，因此添加新的通知目标非常简单：

1. 导入所需的服务包（如 `github.com/nikoksr/notify/service/telegram`）。
2. 在 `Setup` 中基于新配置创建服务实例。
3. 注册到共享通知器（`n.client.UseServices(service)`）。

## 运行测试

```bash
go test ./...
```

## Docker 使用

可以通过以下命令构建最小化的容器镜像：

```bash
# 在本地编译 Go 二进制文件并打包为容器镜像
docker build -t docker-events:latest .
```

包含的 `Dockerfile` 使用多阶段构建：在 Go 构建器镜像中编译静态 Go 二进制文件，然后复制到官方 Docker CLI 镜像中，以便二进制文件在需要时可以调用 `docker`。

重要的运行时注意事项：

- 服务需要与 Docker 守护进程通信。在大多数部署中，你需要将宿主机的 Docker socket 挂载到容器中，以便服务可以监听事件：

  - `/var/run/docker.sock:/var/run/docker.sock:ro`（只读挂载，参见示例 compose 文件）

- 使用环境变量进行配置。仓库包含 `.env.example` 文件——将其复制为 `.env` 并设置你的通知渠道 token 和频道。请勿提交包含真实密钥的 `.env`。

Compose 示例（自动加载 `.env`）

```yaml
services:
  docker-events:
    build: .
    env_file:
      - .env
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    restart: unless-stopped
```

使用 docker-compose 启动：

```bash
# 确保 .env 存在于项目根目录（从 .env.example 复制）
cp .env.example .env
# 构建并在后台启动
docker compose up -d --build
# 查看日志
docker compose logs -f docker-events
```

如果希望直接运行镜像：

```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --env-file .env filippofinke/docker-events:latest
```

提示：默认情况下 Docker Compose 会加载顶层 `.env` 文件。上面的 `env_file` 条目是显式指定的，也可被其他支持 `env_file` 的工具使用。

## 作者

👤 **Filippo Finke**

- 网站: [https://filippofinke.ch](https://filippofinke.ch)
- Twitter: [@filippofinke](https://twitter.com/filippofinke)
- GitHub: [@filippofinke](https://github.com/filippofinke)
- LinkedIn: [@filippofinke](https://linkedin.com/in/filippofinke)
