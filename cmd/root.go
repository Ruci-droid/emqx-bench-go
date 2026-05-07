// Package cmd 提供 CLI 命令行定义，基于 cobra 框架。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"emqx-bench-go/internal/log"
)

var (
	logTo      string // 全局日志输出目标
	configFile string // JSON 配置文件路径
	reportFile string // 压测报告导出路径
)

// rootCmd 是 CLI 的根命令，负责注册全局标志和日志初始化。
var rootCmd = &cobra.Command{
	Use:   "go-mqtt-bench",
	Short: "MQTT 压测工具",
	Long: `go-mqtt-bench 是一个跨平台 MQTT v3/v5 压测工具，
支持连接压测 (conn)、发布压测 (pub) 和订阅压测 (sub)。`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Init(logTo)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logTo, "log-to", "console", "日志输出: console 或 null")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "JSON 配置文件路径")
	rootCmd.PersistentFlags().StringVar(&reportFile, "report", "", "导出压测报告 (.json/.csv/.html)")
}

// Execute 执行根命令，入口由 main 调用。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
