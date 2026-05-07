package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"emqx-bench-go/internal/bench"
	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/stats"
)

func init() {
	rootCmd.AddCommand(pubCmd)

	// 预注册 help 标志，防止 cobra 自动添加 -h 简写（与 --host 的 -h 冲突）
	pubCmd.Flags().Bool("help", false, "显示 pub 帮助信息")

	// 通用参数
	pubCmd.Flags().StringP("host", "h", "localhost", "MQTT Broker 地址，支持逗号分隔多个")
	pubCmd.Flags().IntP("port", "p", 1883, "MQTT Broker 端口")
	pubCmd.Flags().IntP("version", "V", 5, "MQTT 协议版本: 3=MQTT 3.1, 4=MQTT 3.1.1, 5=MQTT 5.0")
	pubCmd.Flags().IntP("count", "c", 200, "客户端数量")
	pubCmd.Flags().IntP("connrate", "R", 0, "每秒创建连接数")
	pubCmd.Flags().IntP("interval", "i", 10, "创建客户端的时间间隔（毫秒）")
	pubCmd.Flags().String("ifaddr", "", "本地绑定的 IP 地址")
	pubCmd.Flags().String("prefix", "", "Client ID 前缀")
	pubCmd.Flags().Bool("shortids", false, "使用短 Client ID")
	pubCmd.Flags().IntP("startnumber", "n", 0, "客户端序号起始值")
	pubCmd.Flags().Int("num-retry-connect", 0, "连接失败后的重试次数")
	pubCmd.Flags().Int("reconnect", 0, "断线后的最大重连次数（0 表示禁用）")
	pubCmd.Flags().StringP("username", "u", "", "用户名")
	pubCmd.Flags().StringP("password", "P", "", "密码")
	pubCmd.Flags().IntP("keepalive", "k", 300, "Keep Alive 秒数")
	pubCmd.Flags().BoolP("clean", "C", true, "Clean Session / Clean Start")
	pubCmd.Flags().IntP("session-expiry", "x", 0, "MQTT 5 Session Expiry 秒数")
	pubCmd.Flags().BoolP("ssl", "S", false, "启用 TLS")
	pubCmd.Flags().String("cacertfile", "", "CA 证书文件路径")
	pubCmd.Flags().String("certfile", "", "客户端证书文件路径")
	pubCmd.Flags().String("keyfile", "", "客户端私钥文件路径")
	pubCmd.Flags().Bool("ws", false, "启用 WebSocket（暂未实现）")
	pubCmd.Flags().Bool("quic", false, "启用 QUIC（暂未实现）")
	pubCmd.Flags().Bool("prometheus", false, "启用 Prometheus 指标")
	pubCmd.Flags().String("restapi", "", "REST API 监听地址")

	// pub 专用参数
	pubCmd.Flags().StringP("topic", "t", "", "发布主题（必填），支持 %i %c %u %s 占位符")
	pubCmd.Flags().IntP("qos", "q", 0, "发布 QoS (0, 1, 2)")
	pubCmd.Flags().BoolP("retain", "r", false, "Retain 标志")
	pubCmd.Flags().IntP("size", "s", 256, "Payload 大小（字节）")
	pubCmd.Flags().StringP("message", "m", "", "固定消息内容")
	pubCmd.Flags().IntP("interval-of-msg", "I", 1000, "单客户端发布消息间隔（毫秒）")
	pubCmd.Flags().IntP("limit", "L", 0, "最大发布消息数（0 表示无限制）")
	pubCmd.Flags().IntP("inflight", "F", 1, "QoS 1/2 最大飞行窗口")
	pubCmd.Flags().BoolP("wait-before-publishing", "w", false, "等待所有客户端连接后再开始发布")
	pubCmd.Flags().Int("max-random-wait", 0, "发布前最大随机等待时间（毫秒）")
	pubCmd.Flags().Int("min-random-wait", 0, "发布前最小随机等待时间（毫秒）")
	pubCmd.Flags().Int("retry-interval", 0, "QoS 1/2 重发间隔（预留参数）")
	pubCmd.Flags().String("payload-hdrs", "", "Payload Headers: cnt64,ts（逗号分隔）")
	pubCmd.Flags().String("topics-payload", "", "多 Topic JSON 配置文件（预留参数）")

	pubCmd.MarkFlagRequired("topic")
}

// pubCmd 是 pub 子命令，用于发布压测。
var pubCmd = &cobra.Command{
	Use:   "pub",
	Short: "发布压测",
	Long:  "创建 MQTT 发布客户端，按指定速率持续发布消息。",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ws, _ := cmd.Flags().GetBool("ws"); ws {
			return fmt.Errorf("WebSocket 传输在此版本中未实现")
		}
		if quic, _ := cmd.Flags().GetBool("quic"); quic {
			return fmt.Errorf("QUIC 传输在此版本中未实现")
		}

		common := parseCommonConfig(cmd)

		cfg := config.PubConfig{
			Common:            common,
			Topic:             mustString(cmd, "topic"),
			QoS:               mustInt(cmd, "qos"),
			Retain:            mustBool(cmd, "retain"),
			Size:              mustInt(cmd, "size"),
			Message:           mustString(cmd, "message"),
			IntervalOfMsg:     mustInt(cmd, "interval-of-msg"),
			Limit:             mustInt(cmd, "limit"),
			Inflight:          mustInt(cmd, "inflight"),
			WaitBeforePublish: mustBool(cmd, "wait-before-publishing"),
			MaxRandomWait:     mustInt(cmd, "max-random-wait"),
			MinRandomWait:     mustInt(cmd, "min-random-wait"),
			RetryInterval:     mustInt(cmd, "retry-interval"),
			PayloadHdrs:       mustString(cmd, "payload-hdrs"),
			TopicsPayloadFile: mustString(cmd, "topics-payload"),
		}

		runner, err := bench.NewPubRunner(cfg)
		if err != nil {
			return fmt.Errorf("创建 pub runner 失败: %w", err)
		}

		reporter := stats.NewReporter(runner.Stats(), runner.Histogram(), "pub")
		reporter.Start(1 * time.Second)

		ctx := context.Background()
		if err := runner.Run(ctx); err != nil {
			return err
		}

		reporter.Stop()
		reporter.PrintFinal()

		return nil
	},
}

// mustString 从命令标志中获取字符串值，忽略错误。
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// mustInt 从命令标志中获取整数值，忽略错误。
func mustInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

// mustBool 从命令标志中获取布尔值，忽略错误。
func mustBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}
