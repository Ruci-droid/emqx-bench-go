package stats

import (
	"sync"
	"time"

	hdr "github.com/HdrHistogram/hdrhistogram-go"
)

// Histogram 基于 hdrhistogram-go 存储延迟样本，支持精确百分位计算。
// 覆盖 1ns 到 60s 的范围，精度为 3 位有效数字。
type Histogram struct {
	mu  sync.Mutex
	hdr *hdr.Histogram
	min int64
	max int64
}

// NewHistogram 创建延迟直方图。
func NewHistogram() *Histogram {
	return &Histogram{
		hdr: hdr.New(1, 60000000000, 3), // 1ns ~ 60s, 3 significant digits
		min: 1<<63 - 1,
	}
}

// Record 记录一次延迟样本。O(1) 时间。
func (h *Histogram) Record(d time.Duration) {
	ns := d.Nanoseconds()
	if ns <= 0 {
		ns = 1
	}
	h.mu.Lock()
	_ = h.hdr.RecordValue(ns)
	if ns < h.min {
		h.min = ns
	}
	if ns > h.max {
		h.max = ns
	}
	h.mu.Unlock()
}

// Stats 返回 min, max, avg, p50, p90, p95, p99（纳秒）。
func (h *Histogram) Stats() (min, max, avg, p50, p90, p95, p99 int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hdr.TotalCount() == 0 {
		return 0, 0, 0, 0, 0, 0, 0
	}

	min = h.min
	max = h.max
	avg = int64(h.hdr.Mean())
	p50 = h.hdr.ValueAtQuantile(50)
	p90 = h.hdr.ValueAtQuantile(90)
	p95 = h.hdr.ValueAtQuantile(95)
	p99 = h.hdr.ValueAtQuantile(99)
	return
}

// Count 返回已记录样本数。
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return int64(h.hdr.TotalCount())
}

// Reset 清空所有样本。
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hdr.Reset()
	h.min = 1<<63 - 1
	h.max = 0
}
