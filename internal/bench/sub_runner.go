package bench

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/mqtt"
	"emqx-bench-go/internal/util"
)

// SubRunner 负责订阅压测：创建指定数量的订阅客户端，统计消息接收速率。
type SubRunner struct {
	*BaseRunner
	cfg      config.SubConfig
	hosts    []string      // 解析后的 Broker 地址列表
	ifAddrs  []string      // 解析后的本地绑定 IP 列表
	active   atomic.Int32  // 当前活跃连接数

	clients  []*mqtt.Client
	mu       sync.Mutex
	lastCnts sync.Map // map[string]uint64, 每个 topic 最后收到的 cnt64 值, 用于乱序检测
}

// NewSubRunner 创建订阅压测 Runner。
func NewSubRunner(cfg config.SubConfig) (*SubRunner, error) {
	base, err := NewBaseRunner(cfg.Common)
	if err != nil {
		return nil, err
	}
	return &SubRunner{
		BaseRunner: base,
		cfg:        cfg,
		hosts:      cfg.Common.Hosts(),
		ifAddrs:    cfg.Common.IfAddrs(),
	}, nil
}

// Run 执行订阅压测。创建客户端、连接、订阅 topic，在消息回调中更新统计。
func (r *SubRunner) Run(ctx context.Context) error {
	ctx = util.WithSignal(ctx)
	count := r.cfg.Common.Count
	r.clients = make([]*mqtt.Client, 0, count)

	// 解析订阅端的 payload headers，用于延迟统计
	hdrs := parseHdrs(r.cfg.PayloadHdrs)
	hdrSize := mqtt.HeaderSize(hdrs)

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

		host := r.hosts[i%len(r.hosts)]
		clientID := util.GenerateClientID(
			r.cfg.Common.Prefix,
			r.cfg.Common.ShortIDs,
			r.cfg.Common.StartNumber,
			i,
		)

		info := mqtt.ClientInfo{
			Index:    int64(i),
			ClientID: clientID,
			Username: r.cfg.Common.Username,
		}
		topic := mqtt.RenderTopic(r.cfg.Topic, info)

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
			},
			OnMessage: func(c *mqtt.Client, topic string, payload []byte) {
				r.collector.SubRecvTotal.Add(1)
				r.collector.SubRecvBytes.Add(int64(len(payload)))

				// 解析 payload headers
				if len(hdrs) > 0 && len(payload) >= hdrSize {
					cnt64, ts, _ := mqtt.ParseHeader(hdrs, payload)

					// cnt64 乱序检测: 每个 topic 独立跟踪最后收到的序列号
					if ts > 0 {
						sendTime := time.Unix(0, ts)
						latency := time.Since(sendTime)
						r.histogram.Record(latency)
					}

					// 检查 cnt64 是否严格递增
					if hasHeader(hdrs, mqtt.HdrCnt64) {
						if last, ok := r.lastCnts.Load(topic); ok {
							lastVal := last.(uint64)
							if cnt64 <= lastVal {
								r.collector.OutOfOrder.Add(1)
							}
						}
						r.lastCnts.Store(topic, cnt64)
					}
				}
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
			zap.L().Error("多次重试后连接仍然失败",
				zap.Int("index", i),
				zap.String("host", host),
			)
			continue
		}

		// 订阅 topic
		if err := client.Subscribe(topic, byte(r.cfg.QoS)); err != nil {
			r.collector.ConnFailed.Add(1)
			zap.L().Error("订阅失败",
				zap.Int("index", i),
				zap.String("topic", topic),
				zap.Error(err),
			)
			client.Disconnect()
			continue
		}

		r.collector.ConnSuccess.Add(1)
		r.active.Add(1)
		r.mu.Lock()
		r.clients = append(r.clients, client)
		r.mu.Unlock()

		zap.L().Debug("订阅成功",
			zap.Int("index", i),
			zap.String("client_id", clientID),
			zap.String("topic", topic),
		)

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

	r.mu.Lock()
	for _, c := range r.clients {
		if c.IsConnected() {
			c.Disconnect()
		}
	}
	r.mu.Unlock()

	return nil
}

// parseHdrs 将逗号分隔的 header 字符串解析为字符串切片。
func parseHdrs(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, h := range strings.Split(s, ",") {
		h = strings.TrimSpace(h)
		if h != "" {
			result = append(result, h)
		}
	}
	return result
}

// hasHeader 检查 header 列表中是否包含指定类型。
func hasHeader(headers []string, name string) bool {
	for _, h := range headers {
		if h == name {
			return true
		}
	}
	return false
}
