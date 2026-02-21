package okxweb3

import "encoding/json"

// Message WebSocket消息结构
type Message struct {
	Event string          `json:"event"` // 事件类型
	Arg   map[string]any  `json:"arg"`   // 参数
	Data  json.RawMessage `json:"data"`  // 数据
}

// GetChannel 获取消息通道名称
func (msg Message) GetChannel() string {
	v, ok := msg.Arg["channel"]
	if !ok {
		return ""
	}

	channelName, ok := v.(string)
	if !ok {
		return ""
	}
	return channelName
}

// GetTokenAddress 获取代币地址
func (msg Message) GetTokenAddress() string {
	v, ok := msg.Arg["tokenAddress"]
	if !ok {
		return ""
	}

	channelName, ok := v.(string)
	if !ok {
		return ""
	}
	return channelName
}

// okxResponse OKX API响应结构
type okxResponse struct {
	Code string          `json:"code"` // 错误码，"0"表示成功
	Msg  string          `json:"msg"`  // 错误消息
	Data json.RawMessage `json:"data"` // 数据
}
