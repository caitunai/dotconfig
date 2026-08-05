package redis

import (
	"context"
	"errors"
	"time"
)

var (
	ErrIncrementNumber = errors.New("error to increment number")
	ErrDecrementNumber = errors.New("error to decrement number")
)

func Increment(ctx context.Context, key string, t time.Duration) (int64, error) {
	res := rdb.Incr(ctx, key)
	result, err := res.Result()
	if err != nil {
		return 0, errors.Join(err, ErrIncrementNumber)
	}
	rdb.Expire(ctx, key, t)
	return result, nil
}

func Decrement(ctx context.Context, key string, t time.Duration) (int64, error) {
	res := rdb.Decr(ctx, key)
	result, err := res.Result()
	if err != nil {
		return 0, errors.Join(err, ErrDecrementNumber)
	}
	rdb.Expire(ctx, key, t)
	return result, nil
}
