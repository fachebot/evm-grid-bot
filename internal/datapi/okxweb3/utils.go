package okxweb3

import (
	"math/rand"

	utls "github.com/refraction-networking/utls"
)

// 支持的TLS指纹客户端列表
var (
	clientHelloIDs = []utls.ClientHelloID{
		utls.HelloChrome_Auto,
		utls.HelloFirefox_Auto,
		utls.HelloEdge_Auto,
		utls.HelloSafari_Auto,
		utls.Hello360_Auto,
		utls.HelloQQ_Auto,
	}
)

// RandomClientHelloID 随机返回一个TLS指纹客户端ID
// 用于模拟不同浏览器的TLS握手特征
func RandomClientHelloID() utls.ClientHelloID {
	return clientHelloIDs[rand.Intn(len(clientHelloIDs))]
}

// ChainIdToChainIndex 将链ID转换为链索引
// chainId: 链ID
// 返回: 链索引字符串, 是否支持
func ChainIdToChainIndex(chainId int64) (string, bool) {
	switch chainId {
	case 56:
		return "56", true
	case 8453:
		return "8453", true
	case 4663:
		return "4663", true
	}
	return "", false
}
