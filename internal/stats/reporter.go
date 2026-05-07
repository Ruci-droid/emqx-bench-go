// Package stats 提供压测指标收集、延迟直方图和终端统计输出。
package stats

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
)

// Reporter 每秒输出统计信息到控制台。
type Reporter struct {
	collector *Collector
	histogram *Histogram
	output    io.Writer
	ticker    *time.Ticker
	done      chan struct{}
	last      Snapshot
	startTime time.Time
	cmd       string // 命令类型: "conn", "pub", "sub"
}

// NewReporter 创建 Reporter。
func NewReporter(collector *Collector, histogram *Histogram, cmd string) *Reporter {
	return &Reporter{
		collector: collector,
		histogram: histogram,
		output:    os.Stdout,
		done:      make(chan struct{}),
		cmd:       cmd,
	}
}

// SetOutput 设置输出目标。
func (r *Reporter) SetOutput(w io.Writer) {
	r.output = w
}

// Start 启动定时统计输出。
func (r *Reporter) Start(interval time.Duration) {
	r.ticker = time.NewTicker(interval)
	r.startTime = time.Now()
	go r.loop()
}

// Stop 停止统计输出。
func (r *Reporter) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
	}
	close(r.done)
}

func (r *Reporter) loop() {
	for {
		select {
		case <-r.ticker.C:
			r.report()
		case <-r.done:
			return
		}
	}
}

func (r *Reporter) report() {
	cur := r.collector.TakeSnapshot()
	elapsed := time.Since(r.startTime).Seconds()

	switch r.cmd {
	case "conn":
		r.reportConn(cur, elapsed)
	case "pub":
		r.reportPub(cur, elapsed)
	case "sub":
		r.reportSub(cur, elapsed)
	}

	r.last = cur
}

func (r *Reporter) reportConn(cur Snapshot, elapsed float64) {
	fmt.Fprintf(r.output,
		"[连接] 成功=%d 失败=%d 活跃=%d 重连=%d 断开=%d 耗时=%.0fs\n",
		cur.ConnSuccess, cur.ConnFailed, cur.ConnActive, cur.Reconnects, cur.Disconnects, elapsed,
	)
}

func (r *Reporter) reportPub(cur Snapshot, elapsed float64) {
	rate := float64(cur.PubSuccess-r.last.PubSuccess) / elapsed
	bytesRate := float64(cur.PubBytes-r.last.PubBytes) / elapsed
	if rate < 0 {
		rate = 0
	}
	if bytesRate < 0 {
		bytesRate = 0
	}

	fmt.Fprintf(r.output,
		"发送(汇总): 总计=%d 速率=%.0f(条/秒) 流量=%s 成功=%d 失败=%d 飞行窗口=%d\n",
		cur.PubSuccess, rate, formatBytes(bytesRate), cur.PubSuccess, cur.PubFailed,
		cur.PubTotal-cur.PubSuccess-cur.PubFailed,
	)

	min, max, avg, p50, p90, p95, p99 := r.histogram.Stats()
	if r.histogram.Count() > 0 {
		fmt.Fprintf(r.output,
			"  延迟: 最小=%s 最大=%s 平均=%s p50=%s p90=%s p95=%s p99=%s\n",
			durStr(min), durStr(max), durStr(avg), durStr(p50), durStr(p90), durStr(p95), durStr(p99),
		)
		r.histogram.Reset()
	}
}

func (r *Reporter) reportSub(cur Snapshot, elapsed float64) {
	rate := float64(cur.SubRecvTotal-r.last.SubRecvTotal) / elapsed
	bytesRate := float64(cur.SubRecvBytes-r.last.SubRecvBytes) / elapsed
	if rate < 0 {
		rate = 0
	}
	if bytesRate < 0 {
		bytesRate = 0
	}

	fmt.Fprintf(r.output,
		"接收(汇总): 总计=%d 速率=%.0f(条/秒) 流量=%s\n",
		cur.SubRecvTotal, rate, formatBytes(bytesRate),
	)

	min, max, avg, p50, p90, p95, p99 := r.histogram.Stats()
	if r.histogram.Count() > 0 {
		fmt.Fprintf(r.output,
			"  延迟: 最小=%s 最大=%s 平均=%s p50=%s p90=%s p95=%s p99=%s\n",
			durStr(min), durStr(max), durStr(avg), durStr(p50), durStr(p90), durStr(p95), durStr(p99),
		)
		r.histogram.Reset()
	}
}

// PrintFinal 输出最终统计摘要。
func (r *Reporter) PrintFinal() {
	cur := r.collector.TakeSnapshot()
	total := time.Since(r.startTime).Seconds()
	fmt.Fprintf(r.output, "\n=== 最终统计 ===\n")
	fmt.Fprintf(r.output, "运行时间: %.2fs\n", total)
	fmt.Fprintf(r.output, "连接: 成功=%d 失败=%d 活跃=%d 重连=%d 断开=%d\n",
		cur.ConnSuccess, cur.ConnFailed, cur.ConnActive, cur.Reconnects, cur.Disconnects)
	if cur.PubSuccess > 0 || cur.PubFailed > 0 {
		fmt.Fprintf(r.output, "发布: 总计=%d 成功=%d 失败=%d 字节=%d 速率=%.1f 条/秒\n",
			cur.PubSuccess+cur.PubFailed, cur.PubSuccess, cur.PubFailed, cur.PubBytes,
			float64(cur.PubSuccess)/total)
	}
	if cur.SubRecvTotal > 0 {
		fmt.Fprintf(r.output, "订阅: 接收=%d 字节=%d 速率=%.1f 条/秒\n",
			cur.SubRecvTotal, cur.SubRecvBytes, float64(cur.SubRecvTotal)/total)
	}
}

// formatBytes 将字节速率格式化为人类可读的字符串。
func formatBytes(bps float64) string {
	if bps < 1024 {
		return fmt.Sprintf("%.0f B/秒", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%.2f KB/秒", bps/1024)
	}
	return fmt.Sprintf("%.2f MB/秒", bps/(1024*1024))
}

// durStr 将纳秒值格式化为人类可读的延迟字符串。
func durStr(ns int64) string {
	if ns < 1000 {
		return fmt.Sprintf("%dns", ns)
	} else if ns < 1000*1000 {
		return fmt.Sprintf("%.2fus", float64(ns)/1000)
	} else if ns < 1000*1000*1000 {
		return fmt.Sprintf("%.2fms", float64(ns)/(1000*1000))
	}
	return fmt.Sprintf("%.2fs", float64(ns)/(1000*1000*1000))
}

// LogSummary 使用 zap 输出最终摘要（结构化日志形式）。
func LogSummary(cmd string, cur Snapshot, elapsed time.Duration, hist *Histogram) {
	total := elapsed.Seconds()
	zap.L().Info("=== 最终统计 ===",
		zap.Float64("运行时间(秒)", total),
		zap.Int64("连接成功", cur.ConnSuccess),
		zap.Int64("连接失败", cur.ConnFailed),
		zap.Int64("活跃连接", cur.ConnActive),
		zap.Int64("重连次数", cur.Reconnects),
		zap.Int64("断开次数", cur.Disconnects),
	)
	if cur.PubSuccess > 0 || cur.PubFailed > 0 {
		zap.L().Info("发布统计",
			zap.Int64("发布总计", cur.PubSuccess+cur.PubFailed),
			zap.Int64("发布成功", cur.PubSuccess),
			zap.Int64("发布失败", cur.PubFailed),
			zap.Float64("发布速率", float64(cur.PubSuccess)/total),
		)
	}
	if cur.SubRecvTotal > 0 {
		zap.L().Info("订阅统计",
			zap.Int64("接收总计", cur.SubRecvTotal),
			zap.Float64("接收速率", float64(cur.SubRecvTotal)/total),
		)
	}
}
