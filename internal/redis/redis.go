package redis

import (
	"fmt"

	"github.com/CentraGlobal/backend-payment-go/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

// NewClient creates a new go-redis client from the given RedisConfig.
func NewClient(cfg config.RedisConfig) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}
