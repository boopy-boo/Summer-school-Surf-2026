package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPAddr       string        `env:"HTTP_ADDR" env-default:":8080"`
	BackendURL     string        `env:"BACKEND_URL" env-default:"http://localhost:8081"`
	BackendTimeout time.Duration `env:"BACKEND_TIMEOUT" env-default:"5s"`
	RedisAddr      string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
	RedisPass      string        `env:"REDIS_PASS"`
	JWTSecret      string        `env:"JWT_SECRET" env-default:"dev-secret-change-me"`
	OTPTTL         time.Duration `env:"OTP_TTL" env-default:"5m"`
	Environment    string        `env:"ENV" env-default:"development"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}