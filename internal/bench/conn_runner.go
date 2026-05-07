package bench

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/mqtt"
	"emqx-bench-go/internal/util"
)

// ConnRunner 负责连接压测：创建大量 MQTT 连接并保持，统计连接成功/失败/断开等指标。
type ConnRunner struct {
	*BaseRunner
	cfg     config.ConnConfig
	hosts   []string      // 解析后的 Broker 地址列表
	ifAddrs []string      // 解析后的本地绑定 IP 列表
	active  atomic.Int32  // 当前活跃连接数

	clients []*mqtt.Client
	mu      sync.Mutex
}

// NewConnRunner 创建连接压测 Runner。
func NewConnRunner(cfg config.ConnConfig) (*ConnRunner, error) {
	base, err := NewBaseRunner(cfg.Common)
	if err != nil {
		return nil, err
	}
	return &ConnRunner{
		BaseRunner: base,
		cfg:        cfg,
		hosts:      cfg.Common.Hosts(),
		ifAddrs:    cfg.Common.IfAddrs(),
	}, nil
}

// Run 执行连接压测。按限速器创建指定数量的客户端，连接成功后保持直到收到退出信号。
func (r *ConnRunner) Run(ctx context.Context) error {
	ctx = util.WithSignal(ctx)
	count := r.cfg.Common.Count
	r.clients = make([]*mqtt.Client, 0, count)

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			goto shutdown
		default:
		}

		if !r.limiter.Wait(ctx) {
			goto shutdown
		}

		// 按 index 轮询分配 host
		host := r.hosts[i%len(r.hosts)]
		clientID := util.GenerateClientID(
			r.cfg.Common.Prefix,
			r.cfg.Common.ShortIDs,
			r.cfg.Common.StartNumber,
			i,
		)

		localAddr := ""
		if len(r.ifAddrs) > 0 {
			localAddr = r.ifAddrs[i%len(r.ifAddrs)]
		}

		client, err := mqtt.NewClient(mqtt.ClientOptions{
			Index:         int64(i),
			ClientID:      clientID,
			Host:          host,
			Port:          r.cfg.Common.Port,
			Version:       r.cfg.Common.MQTTVersion(),
			Username:      r.cfg.Common.Username,
			Password:      r.cfg.Common.Password,
			KeepAlive:     r.cfg.Common.KeepAlive,
			CleanSession:  r.cfg.Common.Clean,
			SessionExpiry: r.cfg.Common.SessionExpiry,
			TLSConfig:     r.tlsConfig,
			LocalAddr:     localAddr,
			WebSocket:     r.cfg.Common.WS,
			OnDisconnect: func(c *mqtt.Client, err error) {
				r.collector.Disconnects.Add(1)
				r.active.Add(-1)
				zap.L().Debug("客户端断开连接",
					zap.Int64("index", c.Index),
					zap.String("client_id", c.ClientID),
				)
			},
		})
		if err != nil {
			r.collector.ConnFailed.Add(1)
			zap.L().Error("创建客户端失败",
				zap.Int("index", i),
				zap.String("host", host),
				zap.Error(err),
			)
			continue
		}

		// 连接，支持重试
		connected := false
		for retry := 0; retry <= r.cfg.Common.NumRetryConnect; retry++ {
			if retry > 0 {
				zap.L().Debug("重试连接",
					zap.Int("index", i),
					zap.Int("retry", retry),
				)
			}

			err := client.Connect()
			if err == nil {
				connected = true
				break
			}

			zap.L().Debug("连接失败",
				zap.Int("index", i),
				zap.String("host", host),
				zap.String("client_id", clientID),
				zap.Error(err),
			)
		}

		if !connected {
			r.collector.ConnFailed.Add(1)
			zap.L().Error("多次重试后连接仍然失败",
				zap.Int("index", i),
				zap.String("host", host),
			)
			continue
		}

		r.collector.ConnSuccess.Add(1)
		r.active.Add(1)
		r.mu.Lock()
		r.clients = append(r.clients, client)
		r.mu.Unlock()

		// 保持连接，等待退出信号后断开
		wg.Add(1)
		go func(c *mqtt.Client) {
			defer wg.Done()
			<-ctx.Done()
			c.Disconnect()
		}(client)
	}

shutdown:
	zap.L().Info("正在关闭，断开所有客户端连接...")
	wg.Wait()

	// 断开所有剩余连接
	r.mu.Lock()
	for _, c := range r.clients {
		if c.IsConnected() {
			c.Disconnect()
		}
	}
	r.mu.Unlock()

	return nil
}
