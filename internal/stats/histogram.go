package stats

import (
	"sort"
	"sync"
	"time"
)

// Histogram 存储延迟样本并计算百分位数。
// MVP 阶段使用简易有序切片实现；后续可替换为 hdrhistogram-go。
type Histogram struct {
	mu     sync.Mutex
	values []int64 // 延迟值（纳秒）
	sum    int64   // 总和
	count  int64   // 样本数
	min    int64   // 最小值
	max    int64   // 最大值
}

// NewHistogram 创建延迟直方图。
func NewHistogram() *Histogram {
	return &Histogram{min: 1<<63 - 1}
}

// Record 记录一次延迟样本。
func (h *Histogram) Record(d time.Duration) {
	ns := d.Nanoseconds()
	h.mu.Lock()
	h.values = append(h.values, ns)
	h.sum += ns
	h.count++
	if ns < h.min {
		h.min = ns
	}
	if ns > h.max {
		h.max = ns
	}
	h.mu.Unlock()
}

// Percentile 计算指定百分位的延迟值。
func (h *Histogram) Percentile(p float64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.values) == 0 {
		return 0
	}
	sort.Slice(h.values, func(i, j int) bool { return h.values[i] < h.values[j] })
	idx := int(float64(len(h.values)) * p)
	if idx >= len(h.values) {
		idx = len(h.values) - 1
	}
	return h.values[idx]
}

// Stats 一次性返回所有统计值：min, max, avg, p50, p90, p95, p99。
func (h *Histogram) Stats() (min, max, avg, p50, p90, p95, p99 int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.count == 0 {
		return 0, 0, 0, 0, 0, 0, 0
	}
	min = h.min
	max = h.max
	avg = h.sum / h.count

	n := len(h.values)
	if n == 0 {
		return
	}
	sort.Slice(h.values, func(i, j int) bool { return h.values[i] < h.values[j] })

	p50 = h.values[int(float64(n)*0.50)]
	p90 = h.values[int(float64(n)*0.90)]
	p95 = h.values[int(float64(n)*0.95)]
	p99 = h.values[int(float64(n)*0.99)]
	return
}

// Count 返回已记录样本数。
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// Reset 清空所有样本，准备下一轮统计。
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = h.values[:0]
	h.sum = 0
	h.count = 0
	h.min = 1<<63 - 1
	h.max = 0
}
