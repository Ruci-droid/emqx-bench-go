// Package cmd 提供 CLI 命令行定义，基于 cobra 框架。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"emqx-bench-go/internal/log"
)

var (
	// logTo 全局日志输出目标，由 PersistentPreRun 传入 log.Init
	logTo string
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
}

// Execute 执行根命令，入口由 main 调用。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
