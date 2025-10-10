package cache

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/patrickmn/go-cache"
	gocache "github.com/patrickmn/go-cache"
)

type RouteInfo struct {
	Path    string
	Context *tgbotapi.Message
}

type MessageCache struct {
	cache *gocache.Cache
}

func NewMessageCache() *MessageCache {
	return &MessageCache{cache: gocache.New(5*time.Minute, 10*time.Minute)}
}

func (router *MessageCache) DelRoute(chatId int64, messageId int) {
	key := fmt.Sprintf("%d:%d", chatId, messageId)
	router.cache.Delete(key)
}

func (router *MessageCache) GetRoute(chatId int64, messageId int) (RouteInfo, bool) {
	key := fmt.Sprintf("%d:%d", chatId, messageId)
	value, ok := router.cache.Get(key)
	if !ok {
		return RouteInfo{}, false
	}
	return value.(RouteInfo), true
}

func (router *MessageCache) SetRoute(chatId int64, messageId int, route RouteInfo) {
	key := fmt.Sprintf("%d:%d", chatId, messageId)
	router.cache.Set(key, route, cache.DefaultExpiration)
}
