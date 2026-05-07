// Package mqtt 封装 Eclipse Paho MQTT 客户端，提供连接、订阅、发布等操作。
package mqtt

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// ConnectHandler 是连接成功时的回调函数类型。
type ConnectHandler func(client *Client)

// DisconnectHandler 是断开连接时的回调函数类型。
type DisconnectHandler func(client *Client, err error)

// MessageHandler 是收到消息时的回调函数类型。
type MessageHandler func(client *Client, topic string, payload []byte)

// Client 是对 Paho MQTT 客户端的封装，保存了压测所需的元信息。
type Client struct {
	Index    int64  // 客户端序号
	ClientID string // 客户端 ID
	Host     string // Broker 地址
	Port     int    // Broker 端口
	Username string // 用户名

	paho    paho.Client // 底层 Paho 客户端
	info    ClientInfo  // 用于 topic 渲染的客户端信息
	version byte        // 协议版本
}

// ClientOptions 是创建 MQTT 客户端所需的配置参数。
type ClientOptions struct {
	Index         int64
	ClientID      string
	Host          string
	Port          int
	Version       byte        // 3=MQTT 3.1, 4=MQTT 3.1.1, 5=MQTT 5.0
	Username      string
	Password      string
	KeepAlive     int         // Keep Alive 秒数
	CleanSession  bool        // Clean Session / Clean Start
	SessionExpiry int         // MQTT 5 Session Expiry
	TLSConfig     *tls.Config // TLS 配置
	LocalAddr     string      // 本地绑定地址

	OnConnect    ConnectHandler
	OnDisconnect DisconnectHandler
	OnMessage    MessageHandler
}

// NewClient 根据给定选项创建 MQTT 客户端，设置连接参数和回调。
func NewClient(opts ClientOptions) (*Client, error) {
	c := &Client{
		Index:    opts.Index,
		ClientID: opts.ClientID,
		Host:     opts.Host,
		Port:     opts.Port,
		Username: opts.Username,
		version:  opts.Version,
		info: ClientInfo{
			Index:    opts.Index,
			ClientID: opts.ClientID,
			Username: opts.Username,
		},
	}

	connOpts := paho.NewClientOptions()

	// 构造 Broker URI
	scheme := "tcp"
	if opts.TLSConfig != nil {
		scheme = "ssl"
	}
	uri := fmt.Sprintf("%s://%s:%d", scheme, opts.Host, opts.Port)
	connOpts.AddBroker(uri)

	connOpts.SetClientID(opts.ClientID)
	connOpts.SetUsername(opts.Username)
	connOpts.SetPassword(opts.Password)
	connOpts.SetKeepAlive(time.Duration(opts.KeepAlive) * time.Second)
	connOpts.SetCleanSession(opts.CleanSession)
	connOpts.SetAutoReconnect(false) // 断线重连由上层自行控制
	connOpts.SetWriteTimeout(30 * time.Second) // 单次写操作超时
	connOpts.SetOrderMatters(true)              // QoS 1/2 按序发送

	// 设置 MQTT 协议版本
	switch opts.Version {
	case 3:
		connOpts.SetProtocolVersion(3) // MQTT 3.1
	case 4:
		connOpts.SetProtocolVersion(4) // MQTT 3.1.1
	default:
		connOpts.SetProtocolVersion(5) // MQTT 5.0
	}

	if opts.TLSConfig != nil {
		connOpts.SetTLSConfig(opts.TLSConfig)
	}

	// 指定本地绑定地址
	if opts.LocalAddr != "" {
		connOpts.SetDialer(&net.Dialer{
			LocalAddr: &net.TCPAddr{IP: net.ParseIP(opts.LocalAddr)},
			Timeout:   30 * time.Second,
		})
	}

	if opts.OnConnect != nil {
		connOpts.SetOnConnectHandler(func(_ paho.Client) {
			opts.OnConnect(c)
		})
	}

	if opts.OnDisconnect != nil {
		connOpts.SetConnectionLostHandler(func(_ paho.Client, err error) {
			opts.OnDisconnect(c, err)
		})
	}

	if opts.OnMessage != nil {
		connOpts.SetDefaultPublishHandler(func(_ paho.Client, msg paho.Message) {
			opts.OnMessage(c, msg.Topic(), msg.Payload())
		})
	}

	c.paho = paho.NewClient(connOpts)
	return c, nil
}

// Connect 建立 MQTT 连接，超时 30 秒。
func (c *Client) Connect() error {
	token := c.paho.Connect()
	if !token.WaitTimeout(30 * time.Second) {
		return fmt.Errorf("客户端 %s 连接超时", c.ClientID)
	}
	return token.Error()
}

// Disconnect 优雅关闭 MQTT 连接，超时 250ms。
func (c *Client) Disconnect() {
	c.paho.Disconnect(250)
}

// Subscribe 订阅指定 topic，超时 10 秒。
func (c *Client) Subscribe(topic string, qos byte) error {
	token := c.paho.Subscribe(topic, qos, nil)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("客户端 %s 订阅 %s 超时", c.ClientID, topic)
	}
	return token.Error()
}

// Publish 发布消息到指定 topic，超时 10 秒。
func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	token := c.paho.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("客户端 %s 发布超时", c.ClientID)
	}
	return token.Error()
}

// Info 返回用于 topic 渲染的客户端信息。
func (c *Client) Info() ClientInfo {
	return c.info
}

// IsConnected 返回客户端是否处于连接状态。
func (c *Client) IsConnected() bool {
	return c.paho.IsConnected()
}

// URL 返回 Broker 的连接地址。
func (c *Client) URL() string {
	scheme := "tcp"
	return fmt.Sprintf("%s://%s:%d", scheme, c.Host, c.Port)
}
