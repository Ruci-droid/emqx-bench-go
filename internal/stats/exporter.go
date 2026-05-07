package stats

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportData 包含生成报告所需的所有数据。
type ReportData struct {
	Cmd       string        // conn / pub / sub
	Elapsed   time.Duration // 运行时长
	Snap      Snapshot      // 最终统计快照
	HistMin   int64         // 最小延迟 (ns)
	HistMax   int64         // 最大延迟 (ns)
	HistAvg   int64         // 平均延迟 (ns)
	HistP50   int64
	HistP90   int64
	HistP95   int64
	HistP99   int64
	HistCount int64 // 延迟样本数
}

// ExportReport 根据文件扩展名自动选择格式导出报告。
// 支持 .json / .csv / .html，其他扩展名默认按 JSON 处理。
func ExportReport(path string, data ReportData) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return exportCSV(path, data)
	case ".html", ".htm":
		return exportHTML(path, data)
	default:
		return exportJSON(path, data)
	}
}

func exportJSON(path string, data ReportData) error {
	type output struct {
		Cmd        string  `json:"cmd"`
		ElapsedSec float64 `json:"elapsed_sec"`

		ConnSuccess int64 `json:"conn_success"`
		ConnFailed  int64 `json:"conn_failed"`
		ConnActive  int64 `json:"conn_active"`
		Reconnects  int64 `json:"reconnects"`
		Disconnects int64 `json:"disconnects"`

		PubTotal   int64 `json:"pub_total,omitempty"`
		PubSuccess int64 `json:"pub_success,omitempty"`
		PubFailed  int64 `json:"pub_failed,omitempty"`
		PubBytes   int64 `json:"pub_bytes,omitempty"`
		PubRate    float64 `json:"pub_rate_msg_per_sec,omitempty"`

		SubTotal  int64 `json:"sub_total,omitempty"`
		SubBytes  int64 `json:"sub_bytes,omitempty"`
		SubRate   float64 `json:"sub_rate_msg_per_sec,omitempty"`
		OutOfOrder int64 `json:"out_of_order,omitempty"`

		Latency *latencyStats `json:"latency,omitempty"`
	}

	o := output{
		Cmd:         data.Cmd,
		ElapsedSec:  data.Elapsed.Seconds(),
		ConnSuccess: data.Snap.ConnSuccess,
		ConnFailed:  data.Snap.ConnFailed,
		ConnActive:  data.Snap.ConnActive,
		Reconnects:  data.Snap.Reconnects,
		Disconnects: data.Snap.Disconnects,
	}

	total := data.Elapsed.Seconds()
	if total <= 0 {
		total = 1
	}

	if data.Snap.PubTotal > 0 {
		o.PubTotal = data.Snap.PubTotal
		o.PubSuccess = data.Snap.PubSuccess
		o.PubFailed = data.Snap.PubFailed
		o.PubBytes = data.Snap.PubBytes
		o.PubRate = float64(data.Snap.PubSuccess) / total
	}

	if data.Snap.SubRecvTotal > 0 {
		o.SubTotal = data.Snap.SubRecvTotal
		o.SubBytes = data.Snap.SubRecvBytes
		o.SubRate = float64(data.Snap.SubRecvTotal) / total
		o.OutOfOrder = data.Snap.OutOfOrder
	}

	if data.HistCount > 0 {
		o.Latency = &latencyStats{
			Count: data.HistCount,
			Min:   durStr(data.HistMin),
			Max:   durStr(data.HistMax),
			Avg:   durStr(data.HistAvg),
			P50:   durStr(data.HistP50),
			P90:   durStr(data.HistP90),
			P95:   durStr(data.HistP95),
			P99:   durStr(data.HistP99),
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建报告文件失败: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}

type latencyStats struct {
	Count int64  `json:"count"`
	Min   string `json:"min"`
	Max   string `json:"max"`
	Avg   string `json:"avg"`
	P50   string `json:"p50"`
	P90   string `json:"p90"`
	P95   string `json:"p95"`
	P99   string `json:"p99"`
}

func exportCSV(path string, data ReportData) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建报告文件失败: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	total := data.Elapsed.Seconds()
	if total <= 0 {
		total = 1
	}

	write := func(k, v string) { w.Write([]string{k, v}) }

	write("指标", "值")
	write("命令", data.Cmd)
	write("运行时长(秒)", fmt.Sprintf("%.2f", total))
	write("连接成功", fmt.Sprintf("%d", data.Snap.ConnSuccess))
	write("连接失败", fmt.Sprintf("%d", data.Snap.ConnFailed))
	write("活跃连接", fmt.Sprintf("%d", data.Snap.ConnActive))
	write("重连次数", fmt.Sprintf("%d", data.Snap.Reconnects))
	write("断开次数", fmt.Sprintf("%d", data.Snap.Disconnects))

	if data.Snap.PubTotal > 0 {
		write("发布总计", fmt.Sprintf("%d", data.Snap.PubTotal))
		write("发布成功", fmt.Sprintf("%d", data.Snap.PubSuccess))
		write("发布失败", fmt.Sprintf("%d", data.Snap.PubFailed))
		write("发布字节", fmt.Sprintf("%d", data.Snap.PubBytes))
		write("发布速率(条/秒)", fmt.Sprintf("%.1f", float64(data.Snap.PubSuccess)/total))
	}

	if data.Snap.SubRecvTotal > 0 {
		write("接收总计", fmt.Sprintf("%d", data.Snap.SubRecvTotal))
		write("接收字节", fmt.Sprintf("%d", data.Snap.SubRecvBytes))
		write("接收速率(条/秒)", fmt.Sprintf("%.1f", float64(data.Snap.SubRecvTotal)/total))
		write("乱序消息", fmt.Sprintf("%d", data.Snap.OutOfOrder))
	}

	if data.HistCount > 0 {
		write("延迟样本数", fmt.Sprintf("%d", data.HistCount))
		write("最小延迟", durStr(data.HistMin))
		write("最大延迟", durStr(data.HistMax))
		write("平均延迟", durStr(data.HistAvg))
		write("p50延迟", durStr(data.HistP50))
		write("p90延迟", durStr(data.HistP90))
		write("p95延迟", durStr(data.HistP95))
		write("p99延迟", durStr(data.HistP99))
	}

	return nil
}

func exportHTML(path string, data ReportData) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建报告文件失败: %w", err)
	}
	defer f.Close()

	total := data.Elapsed.Seconds()
	if total <= 0 {
		total = 1
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="zh"><head><meta charset="UTF-8"><title>压测报告</title>
<style>
body{font-family:-apple-system,system-ui,sans-serif;max-width:800px;margin:40px auto;padding:0 20px;color:#333}
h1{color:#1a73e8;border-bottom:2px solid #1a73e8;padding-bottom:8px}
h2{color:#555;margin-top:30px}
table{width:100%;border-collapse:collapse;margin:12px 0}
th,td{border:1px solid #ddd;padding:8px 12px;text-align:left}
th{background:#f5f5f5;font-weight:600}
.latency td{font-family:monospace;font-size:14px}
</style></head><body>`)

	fmt.Fprintf(&b, "<h1>go-mqtt-bench 压测报告</h1>\n")
	fmt.Fprintf(&b, "<p>命令: <strong>%s</strong> | 运行时长: <strong>%.2f 秒</strong></p>\n",
		html.EscapeString(data.Cmd), total)

	// 连接统计
	b.WriteString("<h2>连接统计</h2>\n<table>\n")
	b.WriteString("<tr><th>指标</th><th>数值</th></tr>\n")
	row(&b, "连接成功", fmt.Sprintf("%d", data.Snap.ConnSuccess))
	row(&b, "连接失败", fmt.Sprintf("%d", data.Snap.ConnFailed))
	row(&b, "活跃连接", fmt.Sprintf("%d", data.Snap.ConnActive))
	row(&b, "重连次数", fmt.Sprintf("%d", data.Snap.Reconnects))
	row(&b, "断开次数", fmt.Sprintf("%d", data.Snap.Disconnects))
	b.WriteString("</table>\n")

	// 发布统计
	if data.Snap.PubTotal > 0 {
		b.WriteString("<h2>发布统计</h2>\n<table>\n")
		b.WriteString("<tr><th>指标</th><th>数值</th></tr>\n")
		row(&b, "发布总计", fmt.Sprintf("%d", data.Snap.PubTotal))
		row(&b, "发布成功", fmt.Sprintf("%d", data.Snap.PubSuccess))
		row(&b, "发布失败", fmt.Sprintf("%d", data.Snap.PubFailed))
		row(&b, "发布字节", fmt.Sprintf("%d", data.Snap.PubBytes))
		row(&b, "平均速率", fmt.Sprintf("%.1f 条/秒", float64(data.Snap.PubSuccess)/total))
		b.WriteString("</table>\n")
	}

	// 订阅统计
	if data.Snap.SubRecvTotal > 0 {
		b.WriteString("<h2>订阅统计</h2>\n<table>\n")
		b.WriteString("<tr><th>指标</th><th>数值</th></tr>\n")
		row(&b, "接收总计", fmt.Sprintf("%d", data.Snap.SubRecvTotal))
		row(&b, "接收字节", fmt.Sprintf("%d", data.Snap.SubRecvBytes))
		row(&b, "平均速率", fmt.Sprintf("%.1f 条/秒", float64(data.Snap.SubRecvTotal)/total))
		row(&b, "乱序消息", fmt.Sprintf("%d", data.Snap.OutOfOrder))
		b.WriteString("</table>\n")
	}

	// 延迟统计
	if data.HistCount > 0 {
		b.WriteString("<h2>延迟统计</h2>\n<table class=\"latency\">\n")
		b.WriteString("<tr><th>指标</th><th>数值</th></tr>\n")
		row(&b, "样本数", fmt.Sprintf("%d", data.HistCount))
		row(&b, "最小", durStr(data.HistMin))
		row(&b, "最大", durStr(data.HistMax))
		row(&b, "平均", durStr(data.HistAvg))
		row(&b, "P50", durStr(data.HistP50))
		row(&b, "P90", durStr(data.HistP90))
		row(&b, "P95", durStr(data.HistP95))
		row(&b, "P99", durStr(data.HistP99))
		b.WriteString("</table>\n")
	}

	b.WriteString("</body></html>")
	f.WriteString(b.String())
	return nil
}

func row(b *strings.Builder, k, v string) {
	fmt.Fprintf(b, "<tr><td>%s</td><td>%s</td></tr>\n", html.EscapeString(k), html.EscapeString(v))
}
