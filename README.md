# go-knocker

[![Go Version](https://img.shields.io/badge/Go-1.27.0-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**go-knocker** 是一个用 Go 编写的高性能服务健康检查与告警工具。它支持对 HTTP 服务和 TCP 端口进行周期性健康检查，并在服务状态变化时通过多种渠道（飞书、钉钉、企业微信、Telegram、Slack、Discord、Teams、邮件、自定义 Webhook）实时推送告警通知。

## 目录

- [核心亮点](#核心亮点)
- [项目架构](#项目架构)
- [状态机设计](#状态机设计)
- [快速开始](#快速开始)
- [配置文件结构](#配置文件结构)
- [配置字段详解](#配置字段详解)
- [使用方法](#使用方法)
- [Docker 部署](#docker-部署)
- [项目结构](#项目结构)
- [License](#license)

---

## 核心亮点

### 1. 多类型健康检查

| 检查类型 | 说明 | 典型场景 |
|---------|------|---------|
| **HTTP** | 发起 HTTP 请求，校验状态码和响应体 | REST API、Web 服务、健康检查端点 |
| **TCP** | 通过 TCP 三次握手检测端口连通性 | MySQL、Redis、PostgreSQL 等数据库 |

### 2. 多通道告警通知

内置 **8 种办公平台 + 邮件 + 自定义 Webhook** 共 10 种告警通道：

| 告警类型 | 配置值 | 说明 |
|---------|-------|------|
| 飞书/Lark | `feishu` | 飞书机器人文本消息 |
| 钉钉 | `dingtalk` | 钉钉机器人文本消息 |
| 企业微信 | `wecom` | 企业微信机器人文本消息 |
| Telegram | `telegram` | Telegram Bot 文本消息 |
| Slack | `slack` | Slack Incoming Webhook |
| Discord | `discord` | Discord Webhook |
| Teams | `teams` | Microsoft Teams Incoming Webhook |
| 邮件 | `email` | SMTP 邮件告警 |
| 自定义 Webhook | `webhook` | 自定义 HTTP 请求，支持 Go 模板语法 |

内置平台开箱即用，只需填入 Webhook URL 即可；自定义 Webhook 支持通过 `{{.Message}}` 模板变量自由构造请求体。

### 3. 智能重试机制

- 可配置重试次数（`retry_count`）和重试间隔（`retry_seconds`）
- 避免网络抖动造成的误报
- 重试期间支持优雅退出中断
- 当重试总耗时超过检查间隔时，自动输出 **WARN 日志**提醒用户

### 4. 状态变化告警（非重复告警）

基于状态机的设计，**只在状态发生变化时发送告警**，避免告警轰炸：

- 服务从健康变为不健康 → 发送"服务异常"告警
- 服务从不健康恢复为健康 → 发送"服务已恢复"告警
- 服务持续健康 → 不发送告警
- 服务持续不健康 → 不重复发送告警

### 5. 双配置格式支持

支持 **JSON** 和 **YAML** 两种配置文件格式，通过 `CONFIG_PATH` 环境变量指定，自动根据文件后缀识别格式。

### 6. 优雅退出

捕获 `SIGINT` 信号，等待当前正在执行的检查完成后再退出，不会造成检查中断或数据丢失。

### 7. 接口化设计，易于扩展

`Target` 和 `Alerter` 均基于接口设计，只需实现对应接口即可扩展新的检查类型和告警渠道。

### 8. Docker 多阶段构建

- 构建阶段使用 `golang:1.27.0`，编译出静态链接的二进制文件
- 运行阶段使用 `alpine:3.24`，最终镜像体积极小
- 内置时区配置（默认 `Asia/Shanghai`）

---

## 项目架构

```
                    ┌──────────────┐
                    │   main.go    │  入口：加载配置, 启动 Knocker
                    └──────┬───────┘
                           │
              ┌────────────▼────────────┐
              │    config/config.go     │  配置解析 (JSON / YAML)
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │  knocker/knocker.go     │  调度器：定时执行 RunOnce
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │   knocker/item.go       │  检查项：状态机 + 重试 + 告警
              └──────┬──────────┬───────┘
                     │          │
          ┌──────────▼──┐  ┌────▼──────────┐
          │ target/     │  │ alerter/      │
          │ ─────────── │  │ ───────────── │
          │ target.go   │  │ alerter.go    │  接口定义
          │ httptarget  │  │ webhookalerter│  平台 Webhook 告警
          │ tcptarget   │  │ emailalerter  │  邮件告警
          └─────────────┘  │ templates.go  │  内置平台模板
                           └───────────────┘
```

---

## 状态机设计

每个检查项 (`Item`) 维护一个内部状态，状态转换如下：

```
                    ┌──────────┐
                    │  Unknown  │  初始状态
                    └─────┬─────┘
                          │
              ┌───────────┴───────────┐
              │ 首次检查               │
              ▼                       ▼
        ┌──────────┐           ┌────────────┐
        │ Healthy  │           │ Unhealthy  │
        └────┬─────┘           └──────┬─────┘
             │                        │
    ┌────────┴────────┐      ┌────────┴────────┐
    │ 检查失败         │      │ 检查成功         │
    ▼                 │      ▼                 │
  ┌────────────┐      │   ┌──────────┐        │
  │ Unhealthy  │      │   │ Healthy  │        │
  │ (发送告警) │      │   │ (发送告警)│        │
  └────────────┘      │   └──────────┘        │
       │              │        │              │
       │ 检查失败      │        │ 检查成功      │
       ▼              │        ▼              │
     (不发送告警)     │     (不发送告警)       │
                      │                       │
                      └───────────────────────┘
                      持续健康 / 持续不健康
```

---

## 快速开始

### 前提条件

- Go 1.27.0+
- 或 Docker（用于容器化部署）

### 从源码运行

```bash
# 克隆项目
git clone https://github.com/LouYuanbo1/go-knocker.git
cd go-knocker

# 编译
go build -o go-knocker .

# 使用 JSON 配置运行（默认）
./go-knocker

# 使用 YAML 配置运行
CONFIG_PATH=config/knocker.yaml ./go-knocker
```

---

## 配置文件结构

配置文件支持 JSON 和 YAML 两种格式，由 `CONFIG_PATH` 环境变量指定路径（默认 `config/knocker.json`）。

### 顶层结构

```yaml
interval_seconds: 30       # 全局检查间隔（秒）
alerters:                  # 告警器列表
  - ...
targets:                   # 检查目标列表
  - ...
items:                     # 检查项列表（关联 target 和 alerter）
  - ...
```

### 完整配置示例 (YAML)

```yaml
interval_seconds: 30

# ==================== 告警器 ====================
alerters:
  # 飞书
  - name: 飞书
    type: feishu
    webhook_url: https://open.feishu.cn/open-apis/bot/v2/hook/your-key

  # 钉钉
  - name: 钉钉
    type: dingtalk
    webhook_url: https://oapi.dingtalk.com/robot/send?access_token=your-token

  # 企业微信
  - name: 企业微信
    type: wecom
    webhook_url: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-key

  # Telegram
  - name: Telegram
    type: telegram
    webhook_url: https://api.telegram.org/bot<TOKEN>/sendMessage

  # Slack
  - name: Slack
    type: slack
    webhook_url: https://hooks.slack.com/services/your/webhook/url

  # Discord
  - name: Discord
    type: discord
    webhook_url: https://discord.com/api/webhooks/your/webhook/url

  # Microsoft Teams
  - name: Teams
    type: teams
    webhook_url: https://your-tenant.webhook.office.com/webhookb2/...

  # 自定义 Webhook
  - name: 自定义Webhook
    type: webhook
    webhook_url: https://your-api.com/alert
    body_template: '{"message":"{{.Message}}"}'

  # 邮件告警
  - name: QQ邮箱
    type: email
    smtp_host: smtp.qq.com
    smtp_port: "587"
    from: your-account@qq.com
    password: your-auth-code
    to:
      - admin@qq.com
    subject: Go Knocker 告警

# ==================== 检查目标 ====================
targets:
  # HTTP 目标
  - name: API服务
    type: http
    url: http://10.0.0.1:8080/health
    method: GET
    expected_status: 200

  - name: 百度
    type: http
    url: https://www.baidu.com
    method: GET
    expected_status: 200

  # TCP 目标
  - name: MySQL
    type: tcp
    address: 10.0.0.2:3306
    timeout_seconds: 5

  - name: Redis
    type: tcp
    address: 10.0.0.3:6379

# ==================== 检查项 ====================
items:
  - target: API服务
    alerters:
      - 飞书
      - QQ邮箱
    retry_count: 3
    retry_seconds: 5

  - target: 百度
    alerters: []

  - target: MySQL
    alerters:
      - 企业微信
      - QQ邮箱
    retry_count: 3
    retry_seconds: 5

  - target: Redis
    alerters:
      - 钉钉
```

---

## 配置字段详解

### 顶层字段

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `interval_seconds` | int | 否 | 30 | 全局健康检查间隔（秒） |
| `alerters` | array | 是 | - | 告警器列表 |
| `targets` | array | 是 | - | 检查目标列表 |
| `items` | array | 是 | - | 检查项列表 |

### AlerterConf（告警器配置）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 告警器名称，在 `items` 中引用 |
| `type` | string | 是 | 告警类型：`feishu` / `dingtalk` / `wecom` / `telegram` / `slack` / `discord` / `teams` / `email` / `webhook` |
| `webhook_url` | string | 条件必填 | Webhook 地址（除 `email` 外都需要） |
| `body_template` | string | `webhook` 必填 | 自定义请求体模板，支持 `{{.Message}}` |
| `smtp_host` | string | `email` 必填 | SMTP 服务器地址 |
| `smtp_port` | string | `email` 必填 | SMTP 端口 |
| `from` | string | `email` 必填 | 发件人邮箱 |
| `password` | string | `email` 必填 | SMTP 授权码 |
| `to` | []string | `email` 必填 | 收件人列表 |
| `subject` | string | 否 | 邮件主题，默认 `Go Knocker Alert` |

### TargetConf（检查目标配置）

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `name` | string | 是 | - | 目标名称，在 `items` 中引用 |
| `type` | string | 是 | - | 目标类型：`http` / `tcp` |
| `url` | string | `http` 必填 | - | HTTP 请求 URL |
| `method` | string | `http` 必填 | - | HTTP 请求方法（GET/POST/HEAD 等） |
| `expected_status` | int | `http` 必填 | - | 期望的 HTTP 状态码 |
| `expected_response` | string | 否 | - | 期望的响应体内容（精确匹配） |
| `address` | string | `tcp` 必填 | - | TCP 地址（host:port） |
| `timeout_seconds` | int | 否 | 5 | TCP 连接超时（秒） |

### ItemConf（检查项配置）

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `target` | string | 是 | - | 关联的目标名称 |
| `alerters` | []string | 是 | - | 关联的告警器名称列表，可为空数组 `[]` |
| `retry_count` | int | 否 | 3 | 失败后重试次数 |
| `retry_seconds` | int | 否 | 5 | 重试间隔（秒） |

---

## 使用方法

### 本地运行

```bash
# 使用 JSON 配置（默认）
go run main.go

# 指定配置文件
CONFIG_PATH=/path/to/config.yaml go run main.go

# 编译后运行
go build -o go-knocker .
./go-knocker
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `CONFIG_PATH` | `config/knocker.json` | 配置文件路径，根据后缀自动识别 JSON/YAML |

### 运行日志示例

```
[API服务] 初始检查通过
[MySQL] 初始检查通过
[API服务] 检查通过
[MySQL] 服务异常: tcp dial 10.0.0.2:3306: connection refused
[MySQL] 重试 1/3 失败: tcp dial 10.0.0.2:3306: connection refused
[MySQL] 重试 2/3 失败: tcp dial 10.0.0.2:3306: connection refused
[MySQL] 重试 3/3 失败: tcp dial 10.0.0.2:3306: connection refused
[MySQL] 服务已恢复
```

---

## Docker 部署

### 构建镜像

```bash
# 构建 Docker 镜像
docker build -t go-knocker:latest .

# 查看镜像大小
docker images go-knocker
```

### 准备配置文件

在运行容器前，需要准备配置文件并挂载到容器中：

```bash
# 创建配置目录
mkdir -p /opt/go-knocker/config

# 将配置文件放入该目录
cp config/knocker.yaml /opt/go-knocker/config/
```

### 运行容器

```bash
# 使用 JSON 配置
docker run -d \
  --name go-knocker \
  --restart unless-stopped \
  -v /opt/go-knocker/config:/app/config \
  -e CONFIG_PATH=/app/config/knocker.json \
  go-knocker:latest

# 使用 YAML 配置
docker run -d \
  --name go-knocker \
  --restart unless-stopped \
  -v /opt/go-knocker/config:/app/config \
  -e CONFIG_PATH=/app/config/knocker.yaml \
  go-knocker:latest
```

### Docker Compose 部署

创建 `docker-compose.yml`：

```yaml
services:
  go-knocker:
    build: .
    container_name: go-knocker
    restart: unless-stopped
    environment:
      - CONFIG_PATH=/app/config/knocker.yaml
      - TZ=Asia/Shanghai
    volumes:
      - ./config/knocker.yaml:/app/config/knocker.yaml
    # 如果使用 TCP 检查同一网络中的服务，可配置网络
    # networks:
    #   - monitor
```

启动：

```bash
docker-compose up -d
```

### 查看日志

```bash
# 查看容器日志
docker logs -f go-knocker

# Docker Compose 日志
docker-compose logs -f go-knocker
```

### Dockerfile 说明

项目采用**多阶段构建**，分为两个阶段：

| 阶段 | 基础镜像 | 作用 |
|------|---------|------|
| **构建阶段** | `golang:1.27.0` | 下载依赖，编译静态二进制文件 |
| **运行阶段** | `alpine:3.24` | 仅包含二进制 + CA 证书 + 时区数据 |

最终镜像仅包含运行时必需的组件，体积小、安全性高。

---

## 项目结构

```
go-knocker/
├── main.go                        # 程序入口
├── go.mod                         # Go 模块依赖
├── go.sum                         # 依赖校验
├── Dockerfile                     # Docker 多阶段构建
├── .dockerignore                  # Docker 构建忽略文件
├── .gitignore                     # Git 忽略文件
├── LICENSE                        # MIT 许可证
├── README.md                      # 项目文档
│
├── config/
│   ├── config.go                  # 配置加载与解析（JSON + YAML）
│   ├── config_test.go             # 配置测试
│   ├── knocker.example.json       # JSON 配置示例
│   └── knocker.example.yaml       # YAML 配置示例
│
├── knocker/
│   ├── knocker.go                 # 核心调度器：定时执行健康检查
│   ├── knocker_test.go            # 调度器测试
│   ├── item.go                    # 检查项：状态机 + 重试 + 告警触发
│   └── item_test.go               # 检查项测试
│
├── target/
│   ├── target.go                  # Target 接口定义
│   ├── httptarget.go              # HTTP 健康检查实现
│   ├── httptarget_test.go         # HTTP 检查测试
│   ├── tcptarget.go               # TCP 健康检查实现
│   └── tcptarget_test.go          # TCP 检查测试
│
└── alerter/
    ├── alerter.go                 # Alerter 接口定义
    ├── webhookalerter.go          # Webhook 告警实现（含自定义模板）
    ├── webhookalerter_test.go     # Webhook 告警测试
    ├── emailalerter.go            # SMTP 邮件告警实现
    ├── emailalerter_test.go       # 邮件告警测试
    ├── templates.go               # 内置平台模板 + 快捷构造函数
    └── templates_test.go          # 模板测试
```

---

## License

本项目基于 [MIT License](LICENSE) 开源发布。

Copyright (c) 2026 louyuanbo1