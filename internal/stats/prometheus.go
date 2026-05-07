package stats

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// PrometheusExporter 暴露符合 Prometheus 规范的 /metrics 端点。
type PrometheusExporter struct {
	collector *Collector
	mu        sync.Mutex
	server    *http.Server
}

// NewPrometheusExporter 创建 Prometheus 指标导出器。
func NewPrometheusExporter(collector *Collector) *PrometheusExporter {
	return &PrometheusExporter{collector: collector}
}

// metric 辅助函数：输出单个 Prometheus 指标行。
func metric(sb *strings.Builder, name, help, typ string, value int64) {
	sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, help))
	sb.WriteString(fmt.Sprintf("# TYPE %s %s\n", name, typ))
	sb.WriteString(fmt.Sprintf("%s %d\n", name, value))
}

// ServeHTTP 实现 http.Handler 接口，输出 Prometheus 文本格式的指标。
func (p *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	snap := p.collector.TakeSnapshot()

	var sb strings.Builder

	metric(&sb, "mqtt_bench_connections_success_total", "Total successful connections.", "counter", snap.ConnSuccess)
	metric(&sb, "mqtt_bench_connections_failed_total", "Total failed connections.", "counter", snap.ConnFailed)
	metric(&sb, "mqtt_bench_connections_terminated_total", "Total disconnected connections.", "counter", snap.Disconnects)
	metric(&sb, "mqtt_bench_connections_reconnect_total", "Total reconnects.", "counter", snap.Reconnects)
	metric(&sb, "mqtt_bench_connections_active", "Current active connections.", "gauge", snap.ConnActive)

	metric(&sb, "mqtt_bench_publish_total", "Total publish attempts.", "counter", snap.PubTotal)
	metric(&sb, "mqtt_bench_publish_success_total", "Successful publishes.", "counter", snap.PubSuccess)
	metric(&sb, "mqtt_bench_publish_failed_total", "Failed publishes.", "counter", snap.PubFailed)
	metric(&sb, "mqtt_bench_publish_bytes_total", "Total bytes published.", "counter", snap.PubBytes)

	metric(&sb, "mqtt_bench_subscribe_received_total", "Total messages received.", "counter", snap.SubRecvTotal)
	metric(&sb, "mqtt_bench_subscribe_received_bytes_total", "Total bytes received.", "counter", snap.SubRecvBytes)

	metric(&sb, "mqtt_bench_out_of_order_total", "Total out-of-order messages detected.", "counter", snap.OutOfOrder)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(sb.String()))
}

// Start 在指定地址启动 HTTP 服务暴露 /metrics。
func (p *PrometheusExporter) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", p)
	p.server = &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("prometheus exporter error: %v\n", err)
		}
	}()
	return nil
}

// Stop 优雅关闭 HTTP 服务。
func (p *PrometheusExporter) Stop() {
	if p.server != nil {
		p.server.Close()
	}
}
