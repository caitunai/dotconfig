package redis

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	ErrAddSortItem        = errors.New("error to add item to sort set")
	ErrRemoveSortItem     = errors.New("error to remove item from sort set")
	ErrGetMinimalSortItem = errors.New("error to get the minimal sort item")
)

func AddSortItem(ctx context.Context, key, item string, value int64) error {
	res := rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(value),
		Member: item,
	})
	err := res.Err()
	if err != nil {
		return errors.Join(err, ErrAddSortItem)
	}
	return nil
}

// RemoveSortItem remove items include min, max value
func RemoveSortItem(ctx context.Context, key string, minV, maxV int64) error {
	res := rdb.ZRemRangeByScore(ctx, key, strconv.FormatInt(minV, 10), strconv.FormatInt(maxV, 10))
	err := res.Err()
	if err != nil {
		return errors.Join(err, ErrRemoveSortItem)
	}
	return nil
}

func GetMinSortItem(ctx context.Context, key string) (string, error) {
	res := rdb.ZRange(ctx, key, 0, 0)
	result, err := res.Result()
	if err != nil {
		return "", errors.Join(err, ErrGetMinimalSortItem)
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return "", nil
}
