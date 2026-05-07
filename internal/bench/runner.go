// Package bench 提供压测的核心执行逻辑：连接 Runner、发布 Runner、订阅 Runner 以及速率控制。
package bench

import (
	"context"
	"crypto/tls"

	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/mqtt"
	"emqx-bench-go/internal/stats"
)

// Runner 是所有压测 Runner 的统一接口。
type Runner interface {
	Run(ctx context.Context) error
	Stats() *stats.Collector
	Histogram() *stats.Histogram
}

// BaseRunner 包含所有 Runner 共用的依赖：统计收集器、直方图、限速器和 TLS 配置。
type BaseRunner struct {
	collector *stats.Collector
	histogram *stats.Histogram
	limiter   *RateLimiter
	tlsConfig *tls.Config
}

// NewBaseRunner 根据通用配置创建 BaseRunner。
// 如果启用了 TLS，会在此处完成 TLS 配置的加载。
func NewBaseRunner(common config.CommonConfig) (*BaseRunner, error) {
	var tlsCfg *tls.Config
	if common.TLS {
		var err error
		tlsCfg, err = mqtt.NewTLSConfig(common.CAFile, common.CertFile, common.KeyFile)
		if err != nil {
			return nil, err
		}
	}

	return &BaseRunner{
		collector: &stats.Collector{
			ConnTotal: int64(common.Count),
		},
		histogram: stats.NewHistogram(),
		limiter:   NewRateLimiter(common.ConnRate, common.Interval),
		tlsConfig: tlsCfg,
	}, nil
}

func (b *BaseRunner) Stats() *stats.Collector    { return b.collector }
func (b *BaseRunner) Histogram() *stats.Histogram { return b.histogram }
func (b *BaseRunner) Limiter() *RateLimiter        { return b.limiter }
func (b *BaseRunner) TLSConfig() *tls.Config       { return b.tlsConfig }
