package redis

import (
	"comments-system/internal/config"

	wbfredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

type CommentsCache struct {
	client  *wbfredis.Client
	retries retry.Strategy
}

func NewPostsCache(cfg *config.Config, retries retry.Strategy) *CommentsCache {
	client := wbfredis.New(cfg.RedisAddr(), cfg.Redis.Pass, cfg.Redis.DB)
	return &CommentsCache{
		client:  client,
		retries: retries,
	}
}
