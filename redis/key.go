package redis

import "github.com/spf13/viper"

const (
	AliveOpenID = "prefix:aliveOpenid"
)

func WithPrefix(k string) string {
	prefix := viper.GetString("redis.prefix")
	if prefix != "" {
		return prefix + ":" + k
	}
	return k
}
