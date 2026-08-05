package cache

import (
	"context"
	"errors"
	"time"

	"github.com/caitunai/dotconfig/redis"
	"github.com/go-redis/cache/v9"
)

var (
	cli          *cache.Cache
	ErrPutString = errors.New("error to put string to redis")
	ErrGetString = errors.New("error to get string from redis")
)

func InitCache() {
	cli = cache.New(&cache.Options{
		Redis: redis.GetClient(),
	})
}

func GetClient() *cache.Cache {
	return cli
}

func PutString(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := cli.Set(buildItem(ctx, key, value, ttl)); err != nil {
		return errors.Join(err, ErrPutString)
	}
	return nil
}

func GetString(ctx context.Context, key string) (string, error) {
	var wanted string
	err := cli.Get(ctx, redis.WithPrefix(key), &wanted)
	if err == nil {
		return wanted, nil
	}
	return wanted, errors.Join(err, ErrGetString)
}

func buildItem(ctx context.Context, key string, value any, ttl time.Duration) *cache.Item {
	return &cache.Item{
		Ctx:   ctx,
		Key:   redis.WithPrefix(key),
		Value: value,
		TTL:   ttl,
	}
}
