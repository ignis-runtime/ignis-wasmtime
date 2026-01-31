package cache

import (
	"context"
	"errors"
	"time"

	"github.com/ignis-runtime/ignis-wasmtime/types"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCache{client: rdb}
}

func (c *RedisCache) Get(ctx context.Context, key string) (*types.Module, bool) {
	module := &types.Module{}
	val, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false
	} else if err != nil {
		return nil, false
	}
	if err := proto.Unmarshal(val, module); err != nil {
		return nil, false
	}

	return module, true
}

func (c *RedisCache) Set(ctx context.Context, key string, module *types.Module, expiration time.Duration) error {
	b, err := proto.Marshal(module)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, expiration).Err()
}
