package redis

import (
	wbfredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

type RedisCache struct {
	client  *wbfredis.Client
	retries retry.Strategy
}
