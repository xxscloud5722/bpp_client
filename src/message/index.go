package message

type Message interface {
	// Push 推送消息
	Push(title, content string) error
}

// NewCPWeChat 创建一个企业微信推送渠道.
func NewCPWeChat(url string) Message {
	return &CPWeChatMessage{URL: url}
}
