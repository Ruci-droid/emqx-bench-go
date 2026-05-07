package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// WithSignal 将 Ctrl+C (SIGINT) 和 SIGTERM 绑定到 context 的取消操作。
// 返回的 context 在收到信号时自动取消，用于实现优雅退出。
func WithSignal(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(ch)
	}()
	return ctx
}
