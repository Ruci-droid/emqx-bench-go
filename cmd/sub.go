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
	rootCmd.AddCommand(subCmd)

	// 预注册 help 标志，防止 cobra 自动添加 -h 简写（与 --host 的 -h 冲突）
	subCmd.Flags().Bool("help", false, "显示 sub 帮助信息")

	// 通用参数
	subCmd.Flags().StringP("host", "h", "localhost", "MQTT Broker 地址，支持逗号分隔多个")
	subCmd.Flags().IntP("port", "p", 1883, "MQTT Broker 端口")
	subCmd.Flags().IntP("version", "V", 5, "MQTT 协议版本: 3=MQTT 3.1, 4=MQTT 3.1.1, 5=MQTT 5.0")
	subCmd.Flags().IntP("count", "c", 200, "客户端数量")
	subCmd.Flags().IntP("connrate", "R", 0, "每秒创建连接数")
	subCmd.Flags().IntP("interval", "i", 10, "创建客户端的时间间隔（毫秒）")
	subCmd.Flags().String("ifaddr", "", "本地绑定的 IP 地址")
	subCmd.Flags().String("prefix", "", "Client ID 前缀")
	subCmd.Flags().Bool("shortids", false, "使用短 Client ID")
	subCmd.Flags().IntP("startnumber", "n", 0, "客户端序号起始值")
	subCmd.Flags().Int("num-retry-connect", 0, "连接失败后的重试次数")
	subCmd.Flags().Int("reconnect", 0, "断线后的最大重连次数（0 表示禁用）")
	subCmd.Flags().StringP("username", "u", "", "用户名")
	subCmd.Flags().StringP("password", "P", "", "密码")
	subCmd.Flags().IntP("keepalive", "k", 300, "Keep Alive 秒数")
	subCmd.Flags().BoolP("clean", "C", true, "Clean Session / Clean Start")
	subCmd.Flags().IntP("session-expiry", "x", 0, "MQTT 5 Session Expiry 秒数")
	subCmd.Flags().BoolP("ssl", "S", false, "启用 TLS")
	subCmd.Flags().String("cacertfile", "", "CA 证书文件路径")
	subCmd.Flags().String("certfile", "", "客户端证书文件路径")
	subCmd.Flags().String("keyfile", "", "客户端私钥文件路径")
	subCmd.Flags().Bool("ws", false, "启用 WebSocket（暂未实现）")
	subCmd.Flags().Bool("quic", false, "启用 QUIC（暂未实现）")
	subCmd.Flags().Bool("prometheus", false, "启用 Prometheus 指标")
	subCmd.Flags().String("restapi", "", "REST API 监听地址")

	// sub 专用参数
	subCmd.Flags().StringP("topic", "t", "", "订阅主题（必填），支持 %i %c %u 占位符")
	subCmd.Flags().IntP("qos", "q", 0, "订阅 QoS (0, 1, 2)")
	subCmd.Flags().String("payload-hdrs", "", "Payload Headers: cnt64,ts（逗号分隔）")

	subCmd.MarkFlagRequired("topic")
}

// subCmd 是 sub 子命令，用于订阅压测。
var subCmd = &cobra.Command{
	Use:   "sub",
	Short: "订阅压测",
	Long:  "创建 MQTT 订阅客户端来测试消息接收吞吐量。",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ws, _ := cmd.Flags().GetBool("ws"); ws {
			return fmt.Errorf("WebSocket 传输在此版本中未实现")
		}
		if quic, _ := cmd.Flags().GetBool("quic"); quic {
			return fmt.Errorf("QUIC 传输在此版本中未实现")
		}

		common := parseCommonConfig(cmd)

		cfg := config.SubConfig{
			Common:      common,
			Topic:       mustString(cmd, "topic"),
			QoS:         mustInt(cmd, "qos"),
			PayloadHdrs: mustString(cmd, "payload-hdrs"),
		}

		runner, err := bench.NewSubRunner(cfg)
		if err != nil {
			return fmt.Errorf("创建 sub runner 失败: %w", err)
		}

		reporter := stats.NewReporter(runner.Stats(), runner.Histogram(), "sub")
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
