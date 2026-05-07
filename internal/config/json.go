package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONConfig 是配置文件的顶层结构，各子命令对应一个字段。
type JSONConfig struct {
	Common *jsonCommonConfig `json:"common,omitempty"`
	Conn   *jsonConnConfig   `json:"conn,omitempty"`
	Pub    *jsonPubConfig    `json:"pub,omitempty"`
	Sub    *jsonSubConfig    `json:"sub,omitempty"`
}

// jsonCommonConfig 对应 CommonConfig 的 JSON 可序列化字段。
type jsonCommonConfig struct {
	Host            *string `json:"host"`
	Port            *int    `json:"port"`
	Version         *int    `json:"version"`
	Count           *int    `json:"count"`
	ConnRate        *int    `json:"connrate"`
	Interval        *int    `json:"interval"`
	IfAddr          *string `json:"ifaddr"`
	Prefix          *string `json:"prefix"`
	ShortIDs        *bool   `json:"shortids"`
	StartNumber     *int    `json:"startnumber"`
	NumRetryConnect *int    `json:"num_retry_connect"`
	Reconnect       *int    `json:"reconnect"`
	Username        *string `json:"username"`
	Password        *string `json:"password"`
	KeepAlive       *int    `json:"keepalive"`
	Clean           *bool   `json:"clean"`
	SessionExpiry   *int    `json:"session_expiry"`
	TLS             *bool   `json:"ssl"`
	CAFile          *string `json:"cacertfile"`
	CertFile        *string `json:"certfile"`
	KeyFile         *string `json:"keyfile"`
	WS              *bool   `json:"ws"`
	QUIC            *bool   `json:"quic"`
	Prometheus      *bool   `json:"prometheus"`
	RestAPI         *string `json:"restapi"`
	LogTo           *string `json:"log_to"`
}

type jsonConnConfig struct {
	Common *jsonCommonConfig `json:"common"`
}

type jsonPubConfig struct {
	Common        *jsonCommonConfig `json:"common"`
	Topic         *string           `json:"topic"`
	QoS           *int              `json:"qos"`
	Retain        *bool             `json:"retain"`
	Size          *int              `json:"size"`
	Message       *string           `json:"message"`
	IntervalOfMsg *int              `json:"interval_of_msg"`
	Limit         *int              `json:"limit"`
	Inflight      *int              `json:"inflight"`
}

type jsonSubConfig struct {
	Common      *jsonCommonConfig `json:"common"`
	Topic       *string           `json:"topic"`
	QoS         *int              `json:"qos"`
	PayloadHdrs *string           `json:"payload_hdrs"`
}

// LoadJSON 读取 JSON 配置文件，CLI 标志中的非零值会覆盖 JSON 配置。
func LoadJSON(path string) (*JSONConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	var cfg JSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return &cfg, nil
}

// ApplyCommon 将 JSON 配置应用到 CommonConfig，仅填充零值字段（CLI 参数优先）。
func ApplyCommon(cfg *CommonConfig, j *jsonCommonConfig) {
	if j == nil {
		return
	}
	if j.Host != nil && cfg.Host == "localhost" {
		cfg.Host = *j.Host
	}
	if j.Port != nil && cfg.Port == 1883 {
		cfg.Port = *j.Port
	}
	if j.Version != nil && cfg.Version == 5 {
		cfg.Version = *j.Version
	}
	if j.Count != nil && cfg.Count == 200 {
		cfg.Count = *j.Count
	}
	if j.ConnRate != nil && cfg.ConnRate == 0 {
		cfg.ConnRate = *j.ConnRate
	}
	if j.Interval != nil && cfg.Interval == 10 {
		cfg.Interval = *j.Interval
	}
	if j.IfAddr != nil && cfg.IfAddr == "" {
		cfg.IfAddr = *j.IfAddr
	}
	if j.Prefix != nil && cfg.Prefix == "" {
		cfg.Prefix = *j.Prefix
	}
	if j.ShortIDs != nil {
		cfg.ShortIDs = *j.ShortIDs
	}
	if j.StartNumber != nil && cfg.StartNumber == 0 {
		cfg.StartNumber = *j.StartNumber
	}
	if j.NumRetryConnect != nil && cfg.NumRetryConnect == 0 {
		cfg.NumRetryConnect = *j.NumRetryConnect
	}
	if j.Reconnect != nil && cfg.Reconnect == 0 {
		cfg.Reconnect = *j.Reconnect
	}
	if j.Username != nil && cfg.Username == "" {
		cfg.Username = *j.Username
	}
	if j.Password != nil && cfg.Password == "" {
		cfg.Password = *j.Password
	}
	if j.KeepAlive != nil && cfg.KeepAlive == 300 {
		cfg.KeepAlive = *j.KeepAlive
	}
	if j.Clean != nil {
		cfg.Clean = *j.Clean
	}
	if j.SessionExpiry != nil && cfg.SessionExpiry == 0 {
		cfg.SessionExpiry = *j.SessionExpiry
	}
	if j.TLS != nil {
		cfg.TLS = *j.TLS
	}
	if j.CAFile != nil && cfg.CAFile == "" {
		cfg.CAFile = *j.CAFile
	}
	if j.CertFile != nil && cfg.CertFile == "" {
		cfg.CertFile = *j.CertFile
	}
	if j.KeyFile != nil && cfg.KeyFile == "" {
		cfg.KeyFile = *j.KeyFile
	}
	if j.WS != nil {
		cfg.WS = *j.WS
	}
	if j.QUIC != nil {
		cfg.QUIC = *j.QUIC
	}
	if j.Prometheus != nil {
		cfg.Prometheus = *j.Prometheus
	}
	if j.RestAPI != nil && cfg.RestAPI == "" {
		cfg.RestAPI = *j.RestAPI
	}
	if j.LogTo != nil && cfg.LogTo == "console" {
		cfg.LogTo = *j.LogTo
	}
}
