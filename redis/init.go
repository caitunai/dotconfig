package redis

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var rdb *redis.Client

func Init() {
	if rdb != nil {
		return
	}
	addr := viper.GetString("redis.host") + ":" + viper.GetString("redis.port")
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
}

func GetClient() *redis.Client {
	if rdb == nil {
		Init()
	}
	return rdb
}
