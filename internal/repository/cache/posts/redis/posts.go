package redis

import (
	"comments-system/internal/config"

	wbfredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

type PostsCache struct {
	client  *wbfredis.Client
	retries retry.Strategy
}

func NewPostsCache(cfg *config.Config, retries retry.Strategy) *PostsCache {
	client := wbfredis.New(cfg.RedisAddr(), cfg.Redis.Pass, cfg.Redis.DB)
	return &PostsCache{
		client:  client,
		retries: retries,
	}
}
