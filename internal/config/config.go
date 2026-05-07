// Package config 定义所有压测场景的配置结构体。
package config

import "strings"

// CommonConfig 保存三个子命令共享的通用参数。
type CommonConfig struct {
	Host            string   // Broker 地址，逗号分隔多个
	Port            int      // Broker 端口
	Version         int      // MQTT 协议版本: 3=3.1, 4=3.1.1, 5=5.0
	Count           int      // 客户端总数
	ConnRate        int      // 每秒连接数，0 表示使用 Interval
	Interval        int      // 客户端创建间隔（毫秒），默认 10
	IfAddr          string   // 本地绑定 IP 地址
	Prefix          string   // Client ID 前缀
	ShortIDs        bool     // 使用短 Client ID
	StartNumber     int      // 起始序号
	NumRetryConnect int      // 连接失败重试次数
	Reconnect       int      // 断线重连次数，0=禁用
	Username        string   // 用户名
	Password        string   // 密码
	KeepAlive       int      // Keep Alive 秒数
	Clean           bool     // Clean Session / Clean Start
	SessionExpiry   int      // MQTT 5 Session Expiry 秒数
	TLS             bool     // 启用 TLS
	CAFile          string   // CA 证书路径
	CertFile        string   // 客户端证书路径
	KeyFile         string   // 客户端私钥路径
	WS              bool     // WebSocket（预留）
	QUIC            bool     // QUIC（预留）
	Prometheus      bool     // 启用 Prometheus 指标
	RestAPI         string   // REST API 监听地址
	LogTo           string   // 日志输出: "console" 或 "null"
}

// Hosts 解析逗号分隔的 host 字符串，返回切片。
func (c CommonConfig) Hosts() []string {
	var hosts []string
	for _, h := range strings.Split(c.Host, ",") {
		h = strings.TrimSpace(h)
		if h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 {
		hosts = []string{"localhost"}
	}
	return hosts
}

// MQTTVersion 将配置中的版本号转为 Paho 库使用的协议版本字节。
func (c CommonConfig) MQTTVersion() byte {
	switch c.Version {
	case 3:
		return 3 // MQTT 3.1
	case 4:
		return 4 // MQTT 3.1.1
	default:
		return 5 // MQTT 5.0
	}
}

// ConnConfig 保存 conn 子命令的配置参数。
type ConnConfig struct {
	Common CommonConfig
}

// PubConfig 保存 pub 子命令的配置参数。
type PubConfig struct {
	Common             CommonConfig
	Topic              string // 发布主题模板
	QoS                int    // 发布 QoS
	Retain             bool   // Retain 标志
	Size               int    // Payload 大小（字节）
	Message            string // 固定消息内容
	IntervalOfMsg      int    // 单客户端发布间隔（毫秒）
	Limit              int    // 每客户端最大消息数，0=无限制
	Inflight           int    // QoS 1/2 最大飞行窗口
	WaitBeforePublish  bool   // 等待所有客户端连接后再发布
	MaxRandomWait      int    // 发布前最大随机等待（毫秒）
	MinRandomWait      int    // 发布前最小随机等待（毫秒）
	RetryInterval      int    // QoS 1/2 重发间隔（预留）
	PayloadHdrs        string // cnt64,ts 逗号分隔
	TopicsPayloadFile  string // 多 Topic JSON 文件（预留）
}

// SubConfig 保存 sub 子命令的配置参数。
type SubConfig struct {
	Common      CommonConfig
	Topic       string // 订阅主题模板
	QoS         int    // 订阅 QoS
	PayloadHdrs string // cnt64,ts 逗号分隔
}
