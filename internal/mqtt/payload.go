package mqtt

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"
)

// Payload Header 类型常量
const (
	HdrCnt64 = "cnt64" // 自增计数器
	HdrTS    = "ts"    // 纳秒时间戳
)

// HeaderSize 计算指定 header 列表所需的总字节数。
func HeaderSize(headers []string) int {
	size := 0
	for _, h := range headers {
		switch h {
		case HdrCnt64:
			size += 8
		case HdrTS:
			size += 8
		}
	}
	return size
}

// PayloadBuilder 高效构造 MQTT Payload，支持缓冲区复用。
// 每条消息调用 Build 时，尽量复用已有切片，减少 GC 压力。
type PayloadBuilder struct {
	size    int      // Payload 总大小
	headers []string // header 类型列表
	fixed   string   // 固定消息内容

	counter atomic.Uint64 // cnt64 自增计数器

	// 预生成的缓存 payload，用于无 header 且无固定消息的快速路径
	cachedPayload []byte
}

// NewPayloadBuilder 创建 Payload 构造器。
// size 必须大于等于 header 所需字节数。
func NewPayloadBuilder(size int, message string, headers []string) (*PayloadBuilder, error) {
	hdrSize := HeaderSize(headers)
	if size < hdrSize {
		return nil, fmt.Errorf("payload 大小 %d 小于 header 所需大小 %d", size, hdrSize)
	}

	pb := &PayloadBuilder{
		size:    size,
		headers: headers,
		fixed:   message,
	}

	// 无 header 且无固定消息时，预填充 'x' 作为默认内容
	if len(headers) == 0 && message == "" {
		pb.cachedPayload = make([]byte, size)
		for i := range pb.cachedPayload {
			pb.cachedPayload[i] = 'x'
		}
	}

	return pb, nil
}

// Build 按照配置构造 Payload。
// 传入的 buf 如果容量足够会被复用，避免每次分配新内存。
func (pb *PayloadBuilder) Build(buf []byte) []byte {
	// 快速路径：无 header 且无固定消息，直接返回缓存
	if len(pb.headers) == 0 && pb.fixed == "" {
		if len(pb.cachedPayload) > 0 {
			return pb.cachedPayload
		}
	}

	payload := ensureCap(buf, pb.size)

	offset := 0
	for _, h := range pb.headers {
		switch h {
		case HdrCnt64:
			cnt := pb.counter.Add(1)
			binary.BigEndian.PutUint64(payload[offset:offset+8], cnt)
			offset += 8
		case HdrTS:
			ts := time.Now().UnixNano()
			binary.BigEndian.PutUint64(payload[offset:offset+8], uint64(ts))
			offset += 8
		}
	}

	body := payload[offset:]

	if pb.fixed != "" {
		copy(body, pb.fixed)
		// 剩余部分填充 'x'
		for i := len(pb.fixed); i < len(body); i++ {
			body[i] = 'x'
		}
	} else {
		for i := range body {
			body[i] = 'x'
		}
	}

	return payload
}

// ParseHeader 从接收到的消息 Payload 中解析 header 内容。
// 返回 cnt64 计数、时间戳和可能的错误。
func ParseHeader(headers []string, payload []byte) (cnt64 uint64, ts int64, err error) {
	offset := 0
	for _, h := range headers {
		switch h {
		case HdrCnt64:
			if offset+8 > len(payload) {
				return 0, 0, fmt.Errorf("payload 太短，无法解析 cnt64 header")
			}
			cnt64 = binary.BigEndian.Uint64(payload[offset : offset+8])
			offset += 8
		case HdrTS:
			if offset+8 > len(payload) {
				return 0, 0, fmt.Errorf("payload 太短，无法解析 ts header")
			}
			ts = int64(binary.BigEndian.Uint64(payload[offset : offset+8]))
			offset += 8
		}
	}
	return
}

// ensureCap 确保切片容量足够，不够则分配新的。
func ensureCap(buf []byte, size int) []byte {
	if cap(buf) >= size {
		return buf[:size]
	}
	return make([]byte, size)
}
