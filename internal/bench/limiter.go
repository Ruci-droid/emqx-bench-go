package bench

import (
	"context"
	"time"
)

// RateLimiter 控制客户端的创建速率。
// 支持两种模式：
//   - connrate: 每秒创建 N 个连接（优先级更高）
//   - interval: 每隔 N 毫秒创建一个连接
type RateLimiter struct {
	connRate int // 每秒连接数，0 表示使用 interval
	interval int // 连接间隔（毫秒）
}

// NewRateLimiter 创建速率限制器。
func NewRateLimiter(connRate, interval int) *RateLimiter {
	return &RateLimiter{
		connRate: connRate,
		interval: interval,
	}
}

// Wait 阻塞直到可以创建下一个客户端。
// 返回 false 表示 context 已取消。
func (r *RateLimiter) Wait(ctx context.Context) bool {
	if r.connRate > 0 {
		delay := time.Second / time.Duration(r.connRate)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(delay):
			return true
		}
	}

	if r.interval > 0 {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Duration(r.interval) * time.Millisecond):
			return true
		}
	}

	// 无速率限制，直接放行
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// WaitForAll 阻塞直到所有预期的客户端都已完成创建。
// 用于 pub 的 --wait-before-publishing 功能。
func WaitForAll(ctx context.Context, active *int32, expected int) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			if int(*active) >= expected {
				return
			}
		}
	}
}
