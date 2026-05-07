// Package util 提供 Client ID 生成和信号处理等工具函数。
package util

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
)

// GenerateClientID 按照配置规则生成 MQTT Client ID。
//
// 生成规则：
//   - 提供 prefix + shortids:  "<prefix><seq>"
//   - 仅 shortids:              "<seq>"
//   - 提供 prefix:              "<prefix><seq>"
//   - 默认:                     "<hostname>_bench_<random>_<seq>"
func GenerateClientID(prefix string, shortids bool, startNumber, index int) string {
	seq := startNumber + index
	if shortids {
		if prefix != "" {
			return fmt.Sprintf("%s%d", prefix, seq)
		}
		return fmt.Sprintf("%d", seq)
	}

	if prefix != "" {
		return fmt.Sprintf("%s%d", prefix, seq)
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	// 清理 hostname 中的特殊字符
	hostname = strings.ReplaceAll(hostname, ".", "_")
	r := rand.Int63n(1 << 40)
	return fmt.Sprintf("%s_bench_%x_%d", hostname, r, seq)
}
