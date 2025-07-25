# Email Server

一个高性能的邮件发送服务，支持异步队列处理和多种队列实现方式。

## 功能特性

- 🚀 **异步邮件发送** - 基于队列的异步处理，避免阻塞API响应
- 📧 **SMTP支持** - 支持各种SMTP邮件服务商（QQ邮箱、163邮箱、Gmail等）
- 🔄 **多队列实现** - 支持内存队列、Redis队列、NATS队列（可扩展）
- ⚡ **高并发处理** - 可配置的Worker池，支持并发邮件发送
- 🔁 **智能重试机制** - 失败自动重试，支持指数退避和抖动算法
- 📊 **任务状态跟踪** - 详细的日志记录和错误处理
- 🛡️ **TLS安全连接** - 支持安全的SMTP连接
- ⚙️ **灵活配置** - 支持环境变量和Viper配置文件两种方式，自动选择加载

## 快速开始

### 1. 环境要求

- Go 1.21+
- SMTP邮箱账号（推荐使用应用专用密码）

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置环境变量

```bash
# SMTP配置
export SMTP_HOST="smtp.163.com"
export SMTP_PORT="25"
export SMTP_USER="your-email@163.com"
export SMTP_PASS="your-app-password"

# 服务配置
export SERVER_PORT="8080"
export MAX_WORKERS="10"
export MAX_QUEUE_SIZE="1000"
```

### 4. 启动服务

#### 方式一：使用环境变量（默认）

```bash
go run cmd/emailer/main.go
```

#### 方式二：使用配置文件

```bash
# 指定配置文件路径
export CONFIG_FILE="local.yaml"
go run cmd/emailer/main.go
```

服务将在 `http://localhost:8080` 启动。

## API 文档

### 发送邮件

**接口地址：** `POST /v1/send-event-email`

**请求格式：**
```json
// {
//     "subject": "邮件主题",
//     "body": "<h1>HTML邮件内容</h1><p>支持HTML格式</p>",
//     "recipients": [
//         "recipient1@example.com",
//         "recipient2@example.com"
//     ]
// }

{
  "subject": "系统维护通知",
  "recipients": ["your_email@example.com"],
  "template_id": "zh/notification_email.html",
  "template_data": {
    "title": "系统维护通知",
    "message": "尊敬的用户，系统将于今晚0点进行维护，预计持续2小时。",
    "extra": "如有疑问请联系 support@example.com"
  }
}
```

**响应格式：**
```json
{
    "message": "Jobs accepted for processing.",
    "count": 2
}
```

**示例请求：**
```bash
curl -X POST http://localhost:8080/v1/send-event-email \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "[通知] 系统更新完成",
    "body": "<h1>尊敬的用户：</h1><p>系统已成功更新至最新版本。</p>",
    "recipients": ["user@example.com"]
  }'
```

## 配置说明

系统支持两种配置加载方式，通过 `CONFIG_FILE` 环境变量自动选择：

- **未设置 `CONFIG_FILE`**: 使用环境变量配置（默认）
- **设置了 `CONFIG_FILE`**: 使用Viper从配置文件加载

### 方式一：环境变量配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `SMTP_HOST` | SMTP服务器地址 | `smtp.qq.com` |
| `SMTP_PORT` | SMTP服务器端口 | `587` |
| `SMTP_USER` | SMTP用户名 | - |
| `SMTP_PASS` | SMTP密码/应用密码 | - |
| `SERVER_PORT` | HTTP服务端口 | `8080` |
| `MAX_WORKERS` | 工作线程数量 | `10` |
| `MAX_QUEUE_SIZE` | 队列缓冲区大小 | `1000` |

### 常用SMTP配置

#### QQ邮箱
```bash
export SMTP_HOST="smtp.qq.com"
export SMTP_PORT="587"
export SMTP_USER="your-qq-number@qq.com"
export SMTP_PASS="your-app-password"  # 需要在QQ邮箱设置中生成
```

#### 163邮箱
```bash
export SMTP_HOST="smtp.163.com"
export SMTP_PORT="25"
export SMTP_USER="your-email@163.com"
export SMTP_PASS="your-app-password"  # 需要在163邮箱设置中生成
```

#### Gmail
```bash
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASS="your-app-password"  # 需要启用两步验证并生成应用密码
```

### 方式二：Viper配置文件

支持YAML、JSON、TOML等多种格式，支持环境变量覆盖。

#### YAML配置示例

```yaml
# config.yaml
smtp:
  host: "smtp.163.com"
  port: 25
  user: "your-email@163.com"
  pass: "your-password"

server:
  port: "8080"

max_workers: 10
max_queue_size: 1000

# 队列配置
queue:
  type: "memory"  # 可选: memory, redis, nats
  memory:
    buffer_size: 1000
```

#### 使用方法

```bash
# 设置配置文件路径
export CONFIG_FILE="config.yaml"
go run cmd/emailer/main.go
```

#### 环境变量覆盖

即使使用配置文件，仍可用环境变量覆盖特定配置：

```bash
export CONFIG_FILE="config.yaml"
export EMAIL_SMTP_HOST="smtp.gmail.com"  # 覆盖配置文件中的smtp.host
export EMAIL_SMTP_PORT="587"             # 覆盖配置文件中的smtp.port
```

## 队列系统

### 架构设计

```
API请求 -> 任务队列 -> Worker池 -> SMTP发送
```

- **API层**: 接收HTTP请求，验证参数，将任务推入队列
- **队列层**: 支持多种队列实现（内存、Redis、NATS）
- **Worker池**: 并发处理队列中的邮件发送任务
- **SMTP层**: 实际的邮件发送逻辑

### 队列类型

#### 1. 内存队列（默认）
- **适用场景**: 单机部署，开发测试
- **特点**: 简单快速，服务重启后任务丢失
- **配置**: 默认启用，无需额外配置

#### 2. Redis队列（规划中）
- **适用场景**: 生产环境，需要持久化
- **特点**: 数据持久化，支持水平扩展
- **配置**: 需要Redis实例

#### 3. NATS队列（规划中）
- **适用场景**: 微服务架构，高并发场景
- **特点**: 高性能消息传递，集群支持
- **配置**: 需要NATS服务器

## 项目结构

```
email-service/
├── cmd/emailer/           # 主程序入口
│   └── main.go
├── internal/
│   ├── api/               # HTTP API处理
│   │   ├── handler.go
│   │   ├── server.go
│   │   └── service.go
│   ├── config/            # 配置管理
│   │   └── config.go
│   ├── mailer/            # 邮件发送核心
│   │   ├── dispatcher.go  # 任务调度器
│   │   ├── worker.go      # 工作线程
│   │   └── job.go         # 任务定义
│   └── queue/             # 队列系统
│       ├── interface.go   # 队列配置
│       ├── factory.go     # 队列工厂
│       └── memory/        # 内存队列实现
├── pkg/jobqueue/          # 队列接口定义
│   └── interface.go
├── config-*.yaml          # 配置文件示例
├── go.mod
├── go.sum
└── README.md
```

## 部署指南

### Docker部署（推荐）

1. 创建Dockerfile：
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o email-service cmd/emailer/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/email-service .
EXPOSE 8080
CMD ["./email-service"]
```

2. 构建和运行：
```bash
docker build -t email-service .
docker run -d -p 8080:8080 \
  -e SMTP_HOST="smtp.163.com" \
  -e SMTP_PORT="25" \
  -e SMTP_USER="your-email@163.com" \
  -e SMTP_PASS="your-password" \
  email-service
```

### 系统服务部署

1. 编译二进制文件：
```bash
go build -o email-service cmd/emailer/main.go
```

2. 创建systemd服务：
```ini
[Unit]
Description=Email Service
After=network.target

[Service]
Type=simple
User=email-service
WorkingDirectory=/opt/email-service
ExecStart=/opt/email-service/email-service
Environment=SMTP_HOST=smtp.163.com
Environment=SMTP_PORT=25
Environment=SMTP_USER=your-email@163.com
Environment=SMTP_PASS=your-password
Restart=always

[Install]
WantedBy=multi-user.target
```

## 监控和日志

### 日志级别
- **INFO**: 正常操作日志（配置加载、Worker启动等）
- **ERROR**: 错误日志（SMTP连接失败、邮件发送失败等）
- **WARNING**: 警告日志（队列满、任务重试等）

### 关键指标监控
- 队列长度
- 邮件发送成功率
- Worker处理速度
- SMTP连接状态

## 开发指南

### 添加新的队列实现

1. 在 `internal/queue/` 下创建新目录
2. 实现 `pkg/jobqueue.JobQueue` 接口
3. 在 `factory.go` 中注册新的队列类型
4. 更新配置结构体

## 重试机制

### 智能重试特性

- **指数退避算法**: 重试延迟时间逐次翻倍（1分钟 → 2分钟 → 4分钟 → ...）
- **最大重试次数**: 默认3次，可配置
- **抖动机制**: 添加±10%随机抖动，避免雷群效应
- **最大延迟限制**: 防止延迟时间过长
- **详细日志**: 记录每次重试的时间和原因

### 重试场景

- SMTP服务器临时不可用
- 网络连接暂时中断
- 认证临时失败
- 服务器负载过高

### 重试配置

通过环境变量可以调整重试行为（当前使用默认配置）：

```bash
# 邮件任务默认最大重试3次
# 基础延迟时间：1分钟
# 最大延迟时间：30分钟
# 退避倍数：2.0
# 启用抖动：是
```

### 扩展功能建议
- [ ] 邮件模板系统
- [ ] 发送状态回调
- [ ] 邮件发送统计
- [ ] 附件支持
- [ ] 定时发送
- [ ] 邮件加密
- [x] 失败重试与指数退避

## 故障排除

### 常见问题

1. **SMTP连接失败**
   - 检查SMTP服务器地址和端口
   - 确认用户名和密码正确
   - 检查网络连接和防火墙设置

2. **邮件发送失败**
   - 检查应用密码是否正确设置
   - 确认SMTP服务商的安全设置
   - 查看详细错误日志

3. **队列堆积**
   - 增加Worker数量
   - 检查SMTP服务器响应速度
   - 监控系统资源使用情况

4. **重试不生效**
   - 检查重试日志是否正常输出
   - 确认错误类型是否适合重试
   - 验证定时器是否正常工作

5. **重试过于频繁**
   - 检查指数退避配置
   - 考虑增加基础延迟时间
   - 减少最大重试次数

### 调试模式

启用详细日志：
```bash
export LOG_LEVEL=DEBUG
go run cmd/emailer/main.go
```

## 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建Pull Request

## 许可证

MIT License

## 联系方式

如有问题或建议，请创建Issue或联系维护者。
