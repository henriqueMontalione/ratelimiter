package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/henriquemontalione/ratelimiter/internal/adapters/redis"
	httpadapter "github.com/henriquemontalione/ratelimiter/internal/adapters/http"
	"github.com/henriquemontalione/ratelimiter/internal/config"
	"github.com/henriquemontalione/ratelimiter/internal/limiter"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}

	store := redis.NewStore(redisClient)
	rl := limiter.NewRateLimiter(store, cfg)

	router := gin.Default()
	router.Use(httpadapter.RateLimit(rl))

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Rate Limiter OK")
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
