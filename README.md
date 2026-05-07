# go-mqtt-bench

跨平台 MQTT 压测工具，仿照 EMQX 官方 [emqtt-bench](https://github.com/emqx/emqtt-bench)，使用 Go 语言开发。

支持 Windows、Linux、macOS，解决官方工具没有 Windows 原生构建的问题。

## 安装

从源码编译：

```bash
git clone https://github.com/Ruci-droid/emqx-bench-go.git
cd emqx-bench-go
```

方式一：Makefile

```bash
make release-linux-amd64    # 仅 Linux amd64
make release-linux-arm64    # 仅 Linux arm64
make release-linux-arm32    # 仅 Linux arm32
make release-linux          # Linux 全架构（上述三个）
make release-windows-amd64  # 仅 Windows amd64
make package                # 全平台编译 + 打包 tar.gz / zip
```

方式二：Shell 脚本（无需 make）

```bash
./scripts/build.sh linux      # Linux amd64 + arm64 + arm32
./scripts/build.sh windows    # Windows amd64
./scripts/build.sh all        # 全平台（含 macOS）
./scripts/build.sh package    # 全平台编译 + 自动打包
```

产物格式

```
┌─────────┬─────────────────────────────────────┐
│  平台   │              归档格式               │
├─────────┼─────────────────────────────────────┤
│ Linux   │ go-mqtt-bench-linux-amd64.tar.gz    │
├─────────┼─────────────────────────────────────┤
│ Linux   │ go-mqtt-bench-linux-arm64.tar.gz    │
├─────────┼─────────────────────────────────────┤
│ Linux   │ go-mqtt-bench-linux-arm32.tar.gz    │
├─────────┼─────────────────────────────────────┤
│ Windows │ go-mqtt-bench-windows-amd64.exe.zip │
└─────────┴─────────────────────────────────────┘
```

所有产物在 build/ 目录，编译时自动注入版本号（从 git tag 读取，否则为 dev）。

## 快速开始

### 连接压测

```bash
# 以每秒 100 个连接的速率创建 1000 个连接
go-mqtt-bench conn -h 127.0.0.1 -p 1883 -c 1000 -R 100

# 创建 10000 个连接并保持
go-mqtt-bench conn -h emqx-server -c 10000

# 绑定本地 IP
go-mqtt-bench conn -h 192.168.0.99 -c 50000 --ifaddr 192.168.0.100
```

### 订阅压测

```bash
# 500 个客户端分别订阅 bench/0 到 bench/499
go-mqtt-bench sub -h 127.0.0.1 -t "bench/%i" -c 500 -q 0

# 订阅带 QoS 2
go-mqtt-bench sub -c 50000 -i 10 -t "bench/%i" -q 2
```

### 发布压测

```bash
# 100 个客户端，每个每 10ms 发布一条 256 字节消息
go-mqtt-bench pub -h 127.0.0.1 -t "bench/%i" -c 100 -I 10 -s 256 -q 0

# 发布到单个 topic，20 个客户端，每 100ms 一条
go-mqtt-bench pub -t t -h 192.168.0.99 -c 20 -I 100
```

### 吞吐测试（双终端）

终端 A（订阅端）：
```bash
go-mqtt-bench sub -h 127.0.0.1 -t t -c 500 -q 0
```

终端 B（发布端）：
```bash
go-mqtt-bench pub -h 127.0.0.1 -t t -c 20 -I 100 -s 256 -q 0
```

## 通用参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--host` | `-h` | localhost | MQTT Broker 地址，支持逗号分隔多个 |
| `--port` | `-p` | 1883 | Broker 端口 |
| `--version` | `-V` | 5 | MQTT 版本 (3=3.1, 4=3.1.1, 5=5.0) |
| `--count` | `-c` | 200 | 客户端数量 |
| `--connrate` | `-R` | 0 | 每秒连接数（优先于 interval） |
| `--interval` | `-i` | 10 | 创建客户端间隔 (ms) |
| `--ifaddr` | | | 本地绑定 IP 地址 |
| `--prefix` | | | Client ID 前缀 |
| `--shortids` | | false | 使用短 Client ID |
| `--startnumber` | `-n` | 0 | 客户端起始序号 |
| `--num-retry-connect` | | 0 | 连接失败重试次数 |
| `--reconnect` | | 0 | 断线重连次数 (0=禁用) |
| `--username` | `-u` | | 用户名 |
| `--password` | `-P` | | 密码 |
| `--keepalive` | `-k` | 300 | Keep Alive 秒数 |
| `--clean` | `-C` | true | Clean Session / Clean Start |
| `--session-expiry` | `-x` | 0 | MQTT 5 Session Expiry 秒数 |
| `--ssl` | `-S` | false | 启用 TLS |
| `--cacertfile` | | | CA 证书路径 |
| `--certfile` | | | 客户端证书路径 |
| `--keyfile` | | | 客户端私钥路径 |
| `--ws` | | false | 启用 WebSocket 传输 |
| `--quic` | | false | QUIC（暂未实现） |
| `--prometheus` | | false | 启用 Prometheus 指标 |
| `--restapi` | | | REST API 监听地址 |
| `--log-to` | | console | 日志输出 (console/null) |
| `--config` | | | JSON 配置文件路径 |
| `--report` | | | 导出压测报告 (.json/.csv/.html) |

## pub 额外参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--topic` | `-t` | (必填) | 发布主题，支持 %i %c %u %s |
| `--qos` | `-q` | 0 | 发布 QoS |
| `--retain` | `-r` | false | Retain 标志 |
| `--size` | `-s` | 256 | Payload 大小 (字节) |
| `--message` | `-m` | | 固定消息内容 |
| `--interval-of-msg` | `-I` | 1000 | 发布间隔 (ms) |
| `--limit` | `-L` | 0 | 最大消息数 (0=无限制) |
| `--inflight` | `-F` | 1 | QoS 1/2 最大飞行窗口 |
| `--wait-before-publishing` | `-w` | false | 等待所有客户端连接后再发布 |
| `--max-random-wait` | | 0 | 发布前最大随机等待 (ms) |
| `--min-random-wait` | | 0 | 发布前最小随机等待 (ms) |
| `--payload-hdrs` | | | Payload headers: cnt64,ts |

## sub 额外参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--topic` | `-t` | (必填) | 订阅主题，支持 %i %c %u |
| `--qos` | `-q` | 0 | 订阅 QoS |
| `--payload-hdrs` | | | Payload headers: cnt64,ts |

## Topic 占位符

| 占位符 | 说明 |
|--------|------|
| `%i` | 客户端序号 |
| `%c` | Client ID |
| `%u` | Username |
| `%s` | 同 %i |

## Client ID 生成规则

1. 提供 `--prefix`：生成 `<prefix><sequence>`
2. 未提供 `--prefix`：生成 `<hostname>_bench_<random>_<sequence>`
3. 启用 `--shortids`：直接使用 sequence（有 prefix 则 `<prefix><sequence>`）

## 高连接数注意事项

- **单源 IP 限制**：TCP 临时端口约 64K，单 IP 不建议超过 ~60000 连接
- **Linux 调优**：
  ```bash
  ulimit -n 1048576
  sysctl -w net.ipv4.ip_local_port_range="1024 65535"
  sysctl -w net.ipv4.tcp_tw_reuse=1
  ```
- **多 IP 分发**：使用 `--ifaddr` 指定多个本地 IP（逗号分隔），客户端轮询分配
- **GOMAXPROCS**：工具默认使用 `runtime.NumCPU()`

## 统计输出

```
send(total): total=100000, rate=10000(msg/sec), bytes_rate=2.44 MB/sec, success=100000, failed=0, inflight=0
recv(total): total=2102563, rate=99725(msg/sec), bytes_rate=24.3 MB/sec
[conn] connected=1000 failed=0 active=1000 reconnects=0 disconnected=0 elapsed=10s
```

按 `Ctrl+C` 优雅退出，退出时显示最终统计摘要。

## 进阶功能

### WebSocket 连接

```bash
go-mqtt-bench conn -h broker.example.com -p 8083 --ws
go-mqtt-bench pub -t test -h broker.example.com -p 8084 --ws --ssl --cacertfile ca.pem
```

使用 `--ws` 启用 MQTT over WebSocket，配合 `--ssl` 时自动使用 `wss://`。

### JSON 配置文件

```bash
go-mqtt-bench pub --config config.json
```

CLI 参数优先级高于 JSON 配置。示例见 `config.example.json`。

### Prometheus 指标

```bash
go-mqtt-bench pub -t test -c 100 --prometheus --restapi :9090
```

访问 `http://localhost:9090/metrics` 查看 Prometheus 格式指标。

### Docker

```bash
docker build -t go-mqtt-bench .
docker run --rm go-mqtt-bench conn -h host.docker.internal -c 100
```

### 多本地 IP 绑定

```bash
go-mqtt-bench conn -h broker --ifaddr 192.168.1.10,192.168.1.11 -c 100000
```

客户端按轮询分配到不同的本地 IP，突破单 IP 64K 端口限制。

### 延迟统计

```bash
# pub 端添加 ts header
go-mqtt-bench pub -t test -c 10 --payload-hdrs ts

# sub 端解析 ts header，自动统计端到端延迟
go-mqtt-bench sub -t test -c 10 --payload-hdrs ts
```

终端输出每秒 p50/p90/p95/p99 延迟。

### 乱序检测

```bash
go-mqtt-bench pub -t bench/%i -c 10 --payload-hdrs cnt64
go-mqtt-bench sub -t bench/%i -c 10 --payload-hdrs cnt64
```

订阅端按 topic 追踪 cnt64 序列号，检测消息乱序。

### 报告导出

```bash
# 导出 JSON 报告（适合程序解析）
go-mqtt-bench pub -t test -c 100 --report result.json

# 导出 CSV 报告（适合 Excel 分析）
go-mqtt-bench sub -t test -c 100 --payload-hdrs ts --report result.csv

# 导出 HTML 报告（浏览器直接查看）
go-mqtt-bench conn -h broker -c 10000 --report result.html
```

报告包含连接、发布/订阅、延迟统计的完整数据，格式根据扩展名自动选择。

## 编译

```bash
make build    # 编译当前平台
make test     # 运行测试
make lint     # 代码检查
make package  # 全平台交叉编译 + 打包 tar.gz/zip
```

## 项目结构

```
cmd/            # CLI 命令 (cobra)
internal/
  config/       # 配置结构定义
  mqtt/         # MQTT 客户端封装、TLS、Topic、Payload
  bench/        # 压测 Runner (conn/pub/sub)、Rate Limiter
  stats/        # 统计收集器、Reporter、延迟直方图
  log/          # 日志初始化
  util/         # 工具函数 (Client ID 生成、信号处理)
main.go
```

## 路线图

- [x] MVP: conn/sub/pub 三个子命令
- [x] TCP MQTT 3.1.1 / 5.0
- [x] 用户名密码、KeepAlive、Clean Session
- [x] TLS 单向认证（mTLS 双向认证已支持）
- [x] Topic 占位符
- [x] 每秒统计输出
- [x] Ctrl+C 优雅退出
- [x] WebSocket 传输 (ws:// / wss://)
- [x] 多 ifaddr 轮询绑定
- [x] hdrhistogram-go 延迟直方图
- [x] cnt64 乱序检测
- [x] JSON 配置文件
- [x] Prometheus /metrics 端点
- [x] Dockerfile
- [ ] QUIC 支持
- [ ] GitHub Actions CI
