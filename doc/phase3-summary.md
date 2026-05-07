# 第三阶段实现总结

日期：2026-05-07

## 已实现功能

### 1. 压测报告导出

新增 `--report` 全局参数，支持 JSON / CSV / HTML 三种格式。

**使用方式**：

```bash
# JSON 格式（适合程序解析）
go-mqtt-bench pub -t test -c 100 --report result.json

# CSV 格式（适合 Excel 分析）
go-mqtt-bench sub -t test -c 100 --payload-hdrs ts --report result.csv

# HTML 格式（可视化报告，浏览器查看）
go-mqtt-bench conn -h broker -c 10000 --report result.html
```

格式根据文件扩展名自动识别。

**报告内容**：
- 命令类型、运行时长
- 连接统计（成功/失败/活跃/重连/断开）
- 发布统计（总计/成功/失败/字节/速率）
- 订阅统计（接收/字节/速率/乱序）
- 延迟统计（样本数/min/max/avg/p50/p90/p95/p99）

**改动文件**：

| 文件 | 说明 |
|------|------|
| `internal/stats/exporter.go` | 新增，JSON/CSV/HTML 三种格式导出逻辑 |
| `internal/stats/reporter.go` | 新增 `Elapsed()` 方法，供报告获取运行时长 |
| `cmd/root.go` | 新增 `--report` 持久标志 |
| `cmd/conn.go` | 新增 `saveReport()` 辅助函数，三个子命令 RunE 末尾调用 |
| `cmd/pub.go` | RunE 末尾调用 `saveReport` |
| `cmd/sub.go` | RunE 末尾调用 `saveReport` |
| `README.md` | 新增报告导出文档和 `--report` 参数说明 |

## 剩余未完成

- QUIC — `--quic` 参数保留，输入时提示未实现
- GitHub Actions CI — 需要仓库配置
- REST API 控制接口
- 分布式压测协调器
- 更精准的 QoS 1/2 inflight 和重发控制
- QoE 事件日志
