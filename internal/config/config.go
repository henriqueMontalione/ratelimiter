package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port              int    `env:"PORT"                   envDefault:"8080"`
	IPRateLimit       int    `env:"IP_RATE_LIMIT"          envDefault:"10"`
	TokenRateLimit    int    `env:"TOKEN_RATE_LIMIT"       envDefault:"20"`
	TokenLimitsRaw    string `env:"TOKEN_LIMITS"`
	BlockDurationSecs int    `env:"BLOCK_DURATION_SECONDS" envDefault:"300"`
	RedisAddr         string `env:"REDIS_ADDR,required"`
	RedisPassword     string `env:"REDIS_PASSWORD"`
	RedisDB           int    `env:"REDIS_DB"               envDefault:"0"`

	TokenLimits map[string]int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	cfg.TokenLimits = parseTokenLimits(cfg.TokenLimitsRaw)

	return cfg, nil
}

func parseTokenLimits(raw string) map[string]int {
	limits := make(map[string]int)
	if raw == "" {
		return limits
	}

	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
		if len(parts) != 2 {
			continue
		}
		token := strings.TrimSpace(parts[0])
		limit, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || token == "" || limit <= 0 {
			continue
		}
		limits[token] = limit
	}

	return limits
}
