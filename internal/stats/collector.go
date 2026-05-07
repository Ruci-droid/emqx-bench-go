package stats

import "sync/atomic"

// Collector 使用无锁 atomic 收集所有压测指标，避免统计成为性能瓶颈。
type Collector struct {
	ConnTotal    int64 // 目标连接数
	ConnSuccess  atomic.Int64
	ConnFailed   atomic.Int64
	ConnActive   atomic.Int64
	Reconnects   atomic.Int64
	Disconnects  atomic.Int64

	PubTotal  atomic.Int64
	PubSuccess atomic.Int64
	PubFailed  atomic.Int64
	PubBytes   atomic.Int64

	SubRecvTotal atomic.Int64
	SubRecvBytes atomic.Int64
}

// Snapshot 是所有计数器的快照，用于 Reporter 计算速率。
type Snapshot struct {
	Time          string
	ConnSuccess   int64
	ConnFailed    int64
	ConnActive    int64
	Reconnects    int64
	Disconnects   int64
	PubTotal      int64
	PubSuccess    int64
	PubFailed     int64
	PubBytes      int64
	SubRecvTotal  int64
	SubRecvBytes  int64
}

// TakeSnapshot 无锁地读取所有计数器当前值。
func (c *Collector) TakeSnapshot() Snapshot {
	return Snapshot{
		ConnSuccess:  c.ConnSuccess.Load(),
		ConnFailed:   c.ConnFailed.Load(),
		ConnActive:   c.ConnActive.Load(),
		Reconnects:   c.Reconnects.Load(),
		Disconnects:  c.Disconnects.Load(),
		PubTotal:     c.PubTotal.Load(),
		PubSuccess:    c.PubSuccess.Load(),
		PubFailed:     c.PubFailed.Load(),
		PubBytes:      c.PubBytes.Load(),
		SubRecvTotal:  c.SubRecvTotal.Load(),
		SubRecvBytes:  c.SubRecvBytes.Load(),
	}
}
