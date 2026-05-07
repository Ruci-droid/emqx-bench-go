# 第二阶段实现总结

日期：2026-05-07

## 已实现功能

1. **hdrhistogram-go 延迟直方图** — 替换简易有序切片为 `github.com/HdrHistogram/hdrhistogram-go`，覆盖 1ns~60s，精度 3 位有效数字。文件: `internal/stats/histogram.go`

2. **cnt64 乱序检测** — 每个 topic 独立追踪最后收到的 cnt64 值（sync.Map），非严格递增时递增 OutOfOrder 计数器。cnt64 改为每个客户端独立（PayloadBuilder 从全局移到 per-goroutine）。文件: `internal/bench/sub_runner.go`, `internal/stats/collector.go`

3. **多 ifaddr 轮询** — `--ifaddr` 支持逗号分隔多个本地 IP，客户端按 index 轮询分配。文件: `internal/config/config.go` (IfAddrs方法), `internal/bench/*_runner.go`

4. **WebSocket 传输** — 利用 Paho 库内置 ws:// / wss:// scheme，新增 `WebSocket` 字段到 ClientOptions。文件: `internal/mqtt/client.go`

5. **JSON 配置文件** — `--config config.json` 加载配置，CLI 参数优先级高于 JSON。文件: `internal/config/json.go`, `cmd/conn.go` (applyJSONConfig)

6. **Prometheus /metrics** — `--prometheus --restapi :9090` 暴露 12 个标准 Prometheus 指标（使用标准库自研，无外部依赖）。文件: `internal/stats/prometheus.go`

7. **Dockerfile** — 多阶段构建 (golang:1.22-alpine → alpine:3.20)。文件: `Dockerfile`

## 关键设计决策

- cnt64 改为每个客户端独立计数（PayloadBuilder 从全局移到 per-goroutine），确保 topic 级乱序检测正确
- WebSocket 直接利用 Paho 库内置能力（ws:// scheme），无需额外实现
- Prometheus 使用标准库自研（无外部依赖），输出符合 Prometheus 文本格式

## 剩余未完成（第三阶段）

- QUIC — --quic 参数保留，输入时提示未实现
- GitHub Actions CI — 需要仓库配置
