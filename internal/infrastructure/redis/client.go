package redisinfra

import (
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func NewClient(redisURL string) *goredis.Client {
	if strings.TrimSpace(redisURL) == "" {
		return nil
	}
	options, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil
	}
	options.DialTimeout = 250 * time.Millisecond
	options.ReadTimeout = 500 * time.Millisecond
	options.WriteTimeout = 500 * time.Millisecond
	return goredis.NewClient(options)
}
