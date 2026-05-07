package mqtt

import (
	"fmt"
	"strings"
)

// ClientInfo 保存客户端标识信息，用于 topic 模板渲染。
type ClientInfo struct {
	Index    int64  // 客户端序号
	ClientID string // 客户端 ID
	Username string // 用户名
}

// RenderTopic 将 topic 模板中的占位符替换为实际值。
// 支持: %i (序号), %c (Client ID), %u (用户名), %s (同 %i)。
// 使用 strings.Builder 避免每次替换产生大量临时字符串。
func RenderTopic(template string, info ClientInfo) string {
	if !strings.Contains(template, "%") {
		return template
	}

	var b strings.Builder
	b.Grow(len(template) + 32)

	for i := 0; i < len(template); i++ {
		if template[i] == '%' && i+1 < len(template) {
			switch template[i+1] {
			case 'i':
				b.WriteString(fmt.Sprintf("%d", info.Index))
				i++
				continue
			case 'c':
				b.WriteString(info.ClientID)
				i++
				continue
			case 'u':
				b.WriteString(info.Username)
				i++
				continue
			case 's':
				b.WriteString(fmt.Sprintf("%d", info.Index))
				i++
				continue
			}
		}
		b.WriteByte(template[i])
	}
	return b.String()
}
