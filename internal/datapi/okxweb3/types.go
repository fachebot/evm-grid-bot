package okxweb3

import "encoding/json"

type Message struct {
	Event string          `json:"event"`
	Arg   map[string]any  `json:"arg"`
	Data  json.RawMessage `json:"data"`
}

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

type okxResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}
