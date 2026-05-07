package bench

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/mqtt"
	"emqx-bench-go/internal/util"
)

// PubRunner 负责发布压测：创建指定数量的发布客户端，按指定速率持续发布消息。
type PubRunner struct {
	*BaseRunner
	cfg    config.PubConfig
	hosts  []string      // 解析后的 Broker 地址列表
	active atomic.Int32  // 当前活跃连接数
	ready  atomic.Int32  // 已完成连接尝试的客户端数量

	clients []*mqtt.Client
	mu      sync.Mutex
}

// NewPubRunner 创建发布压测 Runner。
func NewPubRunner(cfg config.PubConfig) (*PubRunner, error) {
	base, err := NewBaseRunner(cfg.Common)
	if err != nil {
		return nil, err
	}
	return &PubRunner{
		BaseRunner: base,
		cfg:        cfg,
		hosts:      cfg.Common.Hosts(),
	}, nil
}

// Run 执行发布压测。先创建客户端，再启动每个客户端的发布循环。
func (r *PubRunner) Run(ctx context.Context) error {
	ctx = util.WithSignal(ctx)
	count := r.cfg.Common.Count
	r.clients = make([]*mqtt.Client, 0, count)

	// 解析 payload headers
	hdrs := parseHdrs(r.cfg.PayloadHdrs)
	payloadBuilder, err := mqtt.NewPayloadBuilder(r.cfg.Size, r.cfg.Message, hdrs)
	if err != nil {
		return err
	}

	// 预渲染每个客户端的 topic，避免运行时重复计算
	topics := make([]string, count)
	for i := 0; i < count; i++ {
		info := mqtt.ClientInfo{
			Index:    int64(i),
			ClientID: util.GenerateClientID(r.cfg.Common.Prefix, r.cfg.Common.ShortIDs, r.cfg.Common.StartNumber, i),
			Username: r.cfg.Common.Username,
		}
		topics[i] = mqtt.RenderTopic(r.cfg.Topic, info)
	}

	// 每个客户端的发布计数，用于限制消息数量
	pubCounts := make([]atomic.Int64, count)

	var wg sync.WaitGroup   // 等待所有客户端断开
	var pubWg sync.WaitGroup // 等待所有发布 goroutine 结束

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			goto shutdown
		default:
		}

		if !r.limiter.Wait(ctx) {
			goto shutdown
		}

		host := r.hosts[i%len(r.hosts)]
		clientID := util.GenerateClientID(
			r.cfg.Common.Prefix,
			r.cfg.Common.ShortIDs,
			r.cfg.Common.StartNumber,
			i,
		)

		topic := topics[i]

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
			LocalAddr:     r.cfg.Common.IfAddr,
			OnDisconnect: func(c *mqtt.Client, err error) {
				r.collector.Disconnects.Add(1)
				r.active.Add(-1)
			},
		})
		if err != nil {
			r.collector.ConnFailed.Add(1)
			zap.L().Error("创建客户端失败",
				zap.Int("index", i),
				zap.String("host", host),
				zap.Error(err),
			)
			r.ready.Add(1) // 计入尝试数
			continue
		}

		connected := false
		for retry := 0; retry <= r.cfg.Common.NumRetryConnect; retry++ {
			err := client.Connect()
			if err == nil {
				connected = true
				break
			}
		}

		if !connected {
			r.collector.ConnFailed.Add(1)
			r.ready.Add(1)
			zap.L().Error("多次重试后连接仍然失败",
				zap.Int("index", i),
				zap.String("host", host),
			)
			continue
		}

		r.collector.ConnSuccess.Add(1)
		r.active.Add(1)
		r.ready.Add(1)
		r.mu.Lock()
		r.clients = append(r.clients, client)
		r.mu.Unlock()

		// 启动发布 goroutine
		pubWg.Add(1)
		go func(idx int, c *mqtt.Client, t string) {
			defer pubWg.Done()

			// 如果启用了 wait-before-publishing，等待所有客户端连接就绪
			if r.cfg.WaitBeforePublish {
				for r.ready.Load() < int32(count) {
					select {
					case <-ctx.Done():
						return
					case <-time.After(50 * time.Millisecond):
					}
				}
			}

			// 发布前随机等待，用于错峰
			if r.cfg.MinRandomWait > 0 || r.cfg.MaxRandomWait > 0 {
				delay := r.cfg.MinRandomWait
				if r.cfg.MaxRandomWait > r.cfg.MinRandomWait {
					delay += rand.Intn(r.cfg.MaxRandomWait - r.cfg.MinRandomWait)
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(delay) * time.Millisecond):
				}
			}

			qos := byte(r.cfg.QoS)
			retained := r.cfg.Retain

			// 可复用的 payload 缓冲区，减少 GC 压力
			var buf []byte

			ticker := time.NewTicker(time.Duration(r.cfg.IntervalOfMsg) * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}

				// 检查消息数限制
				if r.cfg.Limit > 0 && pubCounts[idx].Load() >= int64(r.cfg.Limit) {
					return
				}

				// 构建 payload（复用缓冲区）
				buf = payloadBuilder.Build(buf)

				r.collector.PubTotal.Add(1)

				err := c.Publish(t, qos, retained, buf)
				if err != nil {
					r.collector.PubFailed.Add(1)
					zap.L().Debug("发布失败",
						zap.Int64("index", c.Index),
						zap.String("client_id", c.ClientID),
						zap.Error(err),
					)
				} else {
					r.collector.PubSuccess.Add(1)
					r.collector.PubBytes.Add(int64(len(buf)))
					pubCounts[idx].Add(1)
				}
			}
		}(i, client, topic)

		// 保活 goroutine，等待退出信号后断开连接
		wg.Add(1)
		go func(c *mqtt.Client) {
			defer wg.Done()
			<-ctx.Done()
			c.Disconnect()
		}(client)
	}

shutdown:
	zap.L().Info("正在关闭，等待发布者停止...")
	pubWg.Wait()

	zap.L().Info("正在断开所有客户端连接...")
	wg.Wait()

	r.mu.Lock()
	for _, c := range r.clients {
		if c.IsConnected() {
			c.Disconnect()
		}
	}
	r.mu.Unlock()

	return nil
}
