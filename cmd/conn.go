package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"emqx-bench-go/internal/bench"
	"emqx-bench-go/internal/config"
	"emqx-bench-go/internal/stats"
)

func init() {
	rootCmd.AddCommand(connCmd)

	// 预注册 help 标志，防止 cobra 自动添加 -h 简写（与 --host 的 -h 冲突）
	connCmd.Flags().Bool("help", false, "显示 conn 帮助信息")

	// 通用参数
	connCmd.Flags().StringP("host", "h", "localhost", "MQTT Broker 地址，支持逗号分隔多个")
	connCmd.Flags().IntP("port", "p", 1883, "MQTT Broker 端口")
	connCmd.Flags().IntP("version", "V", 5, "MQTT 协议版本: 3=MQTT 3.1, 4=MQTT 3.1.1, 5=MQTT 5.0")
	connCmd.Flags().IntP("count", "c", 200, "客户端数量")
	connCmd.Flags().IntP("connrate", "R", 0, "每秒创建连接数（0 表示使用 interval）")
	connCmd.Flags().IntP("interval", "i", 10, "创建客户端的时间间隔（毫秒）")
	connCmd.Flags().String("ifaddr", "", "本地绑定的 IP 地址")
	connCmd.Flags().String("prefix", "", "Client ID 前缀")
	connCmd.Flags().Bool("shortids", false, "使用短 Client ID")
	connCmd.Flags().IntP("startnumber", "n", 0, "客户端序号起始值")
	connCmd.Flags().Int("num-retry-connect", 0, "连接失败后的重试次数")
	connCmd.Flags().Int("reconnect", 0, "断线后的最大重连次数（0 表示禁用）")
	connCmd.Flags().StringP("username", "u", "", "用户名")
	connCmd.Flags().StringP("password", "P", "", "密码")
	connCmd.Flags().IntP("keepalive", "k", 300, "Keep Alive 秒数")
	connCmd.Flags().BoolP("clean", "C", true, "Clean Session / Clean Start")
	connCmd.Flags().IntP("session-expiry", "x", 0, "MQTT 5 Session Expiry 秒数")
	connCmd.Flags().BoolP("ssl", "S", false, "启用 TLS")
	connCmd.Flags().String("cacertfile", "", "CA 证书文件路径")
	connCmd.Flags().String("certfile", "", "客户端证书文件路径")
	connCmd.Flags().String("keyfile", "", "客户端私钥文件路径")
	connCmd.Flags().Bool("ws", false, "启用 WebSocket 传输")
	connCmd.Flags().Bool("quic", false, "启用 QUIC（暂未实现）")
	connCmd.Flags().Bool("prometheus", false, "启用 Prometheus 指标")
	connCmd.Flags().String("restapi", "", "REST API 监听地址，例如 :9090")
}

// connCmd 是 conn 子命令，用于连接压测。
var connCmd = &cobra.Command{
	Use:   "conn",
	Short: "连接压测",
	Long:  "创建大量 MQTT 连接来测试 Broker 的连接容量。",
	RunE: func(cmd *cobra.Command, args []string) error {
		if quic, _ := cmd.Flags().GetBool("quic"); quic {
			return fmt.Errorf("QUIC 传输在此版本中未实现")
		}

		common := parseCommonConfig(cmd)
		applyJSONConfig(&common, nil, nil)

		cfg := config.ConnConfig{
			Common: common,
		}

		runner, err := bench.NewConnRunner(cfg)
		if err != nil {
			return fmt.Errorf("创建 conn runner 失败: %w", err)
		}

		// 启动 Prometheus 指标导出
		startPrometheus(common, runner.Stats())

		reporter := stats.NewReporter(runner.Stats(), runner.Histogram(), "conn")
		reporter.Start(1 * time.Second)

		ctx := context.Background()
		if err := runner.Run(ctx); err != nil {
			return err
		}

		reporter.Stop()
		reporter.PrintFinal()
		saveReport("conn", runner, reporter)

		return nil
	},
}

// parseCommonConfig 解析三个子命令共享的通用参数，返回 CommonConfig。
func parseCommonConfig(cmd *cobra.Command) config.CommonConfig {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	version, _ := cmd.Flags().GetInt("version")
	count, _ := cmd.Flags().GetInt("count")
	connRate, _ := cmd.Flags().GetInt("connrate")
	interval, _ := cmd.Flags().GetInt("interval")
	ifAddr, _ := cmd.Flags().GetString("ifaddr")
	prefix, _ := cmd.Flags().GetString("prefix")
	shortIDs, _ := cmd.Flags().GetBool("shortids")
	startNumber, _ := cmd.Flags().GetInt("startnumber")
	numRetryConnect, _ := cmd.Flags().GetInt("num-retry-connect")
	reconnect, _ := cmd.Flags().GetInt("reconnect")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	keepAlive, _ := cmd.Flags().GetInt("keepalive")
	clean, _ := cmd.Flags().GetBool("clean")
	sessionExpiry, _ := cmd.Flags().GetInt("session-expiry")
	ssl, _ := cmd.Flags().GetBool("ssl")
	caFile, _ := cmd.Flags().GetString("cacertfile")
	certFile, _ := cmd.Flags().GetString("certfile")
	keyFile, _ := cmd.Flags().GetString("keyfile")
	ws, _ := cmd.Flags().GetBool("ws")
	quic, _ := cmd.Flags().GetBool("quic")
	prom, _ := cmd.Flags().GetBool("prometheus")
	restAPI, _ := cmd.Flags().GetString("restapi")

	// 如果绑定了单个 IP 且连接数超过 60000，给出临时端口耗尽警告
	if ifAddr != "" && count > 60000 {
		fmt.Fprintf(os.Stderr, "警告: 单源 IP %s 可能无法支持超过约 64K 连接（受 TCP 临时端口数量限制）。\n", ifAddr)
		fmt.Fprintf(os.Stderr, "  建议使用多个 IP 或调整以下系统参数:\n")
		fmt.Fprintf(os.Stderr, "  - ulimit -n\n")
		fmt.Fprintf(os.Stderr, "  - net.ipv4.ip_local_port_range\n")
		fmt.Fprintf(os.Stderr, "  - net.ipv4.tcp_tw_reuse\n")
	}

	_ = ws
	_ = quic
	_ = prom
	_ = restAPI

	return config.CommonConfig{
		Host:            host,
		Port:            port,
		Version:         version,
		Count:           count,
		ConnRate:        connRate,
		Interval:        interval,
		IfAddr:          ifAddr,
		Prefix:          prefix,
		ShortIDs:        shortIDs,
		StartNumber:     startNumber,
		NumRetryConnect: numRetryConnect,
		Reconnect:       reconnect,
		Username:        username,
		Password:        password,
		KeepAlive:       keepAlive,
		Clean:           clean,
		SessionExpiry:   sessionExpiry,
		TLS:             ssl,
		CAFile:          caFile,
		CertFile:        certFile,
		KeyFile:         keyFile,
		WS:              ws,
		QUIC:            quic,
		Prometheus:      prom,
		RestAPI:         restAPI,
		LogTo:           logTo,
	}
}

// applyJSONConfig 如果指定了 --config，将 JSON 配置文件中的值作为默认值。
// CLI 参数优先级高于 JSON 配置。
func applyJSONConfig(common *config.CommonConfig, pub *config.PubConfig, sub *config.SubConfig) {
	if configFile == "" {
		return
	}
	cfg, err := config.LoadJSON(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 加载配置文件失败: %v\n", err)
		return
	}
	if cfg.Common != nil {
		config.ApplyCommon(common, cfg.Common)
	}
	if pub != nil && cfg.Pub != nil {
		if cfg.Pub.Common != nil {
			config.ApplyCommon(common, cfg.Pub.Common)
		}
		if cfg.Pub.Topic != nil && pub.Topic == "" {
			pub.Topic = *cfg.Pub.Topic
		}
		if cfg.Pub.QoS != nil {
			pub.QoS = *cfg.Pub.QoS
		}
		if cfg.Pub.Size != nil && pub.Size == 256 {
			pub.Size = *cfg.Pub.Size
		}
		if cfg.Pub.IntervalOfMsg != nil && pub.IntervalOfMsg == 1000 {
			pub.IntervalOfMsg = *cfg.Pub.IntervalOfMsg
		}
		if cfg.Pub.Limit != nil && pub.Limit == 0 {
			pub.Limit = *cfg.Pub.Limit
		}
	}
	if sub != nil && cfg.Sub != nil {
		if cfg.Sub.Common != nil {
			config.ApplyCommon(common, cfg.Sub.Common)
		}
		if cfg.Sub.Topic != nil && sub.Topic == "" {
			sub.Topic = *cfg.Sub.Topic
		}
		if cfg.Sub.QoS != nil {
			sub.QoS = *cfg.Sub.QoS
		}
		if cfg.Sub.PayloadHdrs != nil && sub.PayloadHdrs == "" {
			sub.PayloadHdrs = *cfg.Sub.PayloadHdrs
		}
	}
}

// startPrometheus 在 --prometheus 启用时启动 /metrics HTTP 端点。
func startPrometheus(common config.CommonConfig, collector *stats.Collector) {
	if common.Prometheus && common.RestAPI != "" {
		exporter := stats.NewPrometheusExporter(collector)
		if err := exporter.Start(common.RestAPI); err != nil {
			fmt.Fprintf(os.Stderr, "启动 Prometheus 端点失败: %v\n", err)
		}
	}
}

// saveReport 在指定 --report 时导出压测报告。
func saveReport(cmd string, runner bench.Runner, reporter *stats.Reporter) {
	if reportFile == "" {
		return
	}
	snap := runner.Stats().TakeSnapshot()
	hist := runner.Histogram()
	min, max, avg, p50, p90, p95, p99 := hist.Stats()

	data := stats.ReportData{
		Cmd:       cmd,
		Elapsed:   reporter.Elapsed(),
		Snap:      snap,
		HistMin:   min,
		HistMax:   max,
		HistAvg:   avg,
		HistP50:   p50,
		HistP90:   p90,
		HistP95:   p95,
		HistP99:   p99,
		HistCount: hist.Count(),
	}
	if err := stats.ExportReport(reportFile, data); err != nil {
		fmt.Fprintf(os.Stderr, "导出报告失败: %v\n", err)
	} else {
		fmt.Fprintf(os.Stdout, "报告已导出到: %s\n", reportFile)
	}
}
