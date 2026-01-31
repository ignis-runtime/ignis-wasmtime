package cache

import (
	"context"
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

func (c *RedisCache) Get(ctx context.Context, key string) (*types.Module, error) {
	module := &types.Module{}
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, err
	}
	if err := proto.Unmarshal(val, module); err != nil {
		return nil, err
	}

	return module, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, module *types.Module, expiration time.Duration) error {
	b, err := proto.Marshal(module)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, expiration).Err()
}
