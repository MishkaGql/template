package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type DBConfig struct {
	URL               string        `env:"DATABASE_URL" validate:"required"`
	MaxConns          int32         `env:"DATABASE_MAX_CONNS" validate:"required,gt=1"`
	MinConns          int32         `env:"DATABASE_MIN_CONNS" validate:"required,gt=0"`
	MaxConnLifetime   time.Duration `env:"DATABASE_MAX_CONN_LIFETIME" validate:"required,gt=0"`
	ConnectTimeout    time.Duration `env:"DATABASE_CONNECT_TIMEOUT" validate:"required,gt=0"`
	QueryTimeout      time.Duration `env:"DATABASE_QUERY_TIMEOUT" validate:"required,gt=0"`
	Jitter            time.Duration `env:"DATABASE_CONNECTIONS_JITTER" validate:"required,gte=0"`
	PingTimeout       time.Duration `env:"DATABASE_PING_TIMEOUT" validate:"required,gte=0"`
	HealthCheckPeriod time.Duration `env:"DATABASE_HEALTHCHECK_PERIOD" validate:"required,gt=0"`
}

type HTTPConfig struct {
	Addr              string        `env:"HTTP_ADDR" validate:"required"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" validate:"required"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" validate:"required"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" validate:"required"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" validate:"required"`
}

type Config struct {
	DB              DBConfig      `validate:"required"`
	HTTP            HTTPConfig    `validate:"required"`
	LogLevel        string        `env:"LOG_LEVEL" validate:"required,oneof=debug warn info error"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" validate:"required,gt=0"`
}

func MustLoad(validate *validator.Validate) *Config {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		panic(fmt.Sprintf("parse config failed: %v ", err))
	}

	if err := validate.Struct(&cfg); err != nil {
		panic(fmt.Sprintf("validate config failed: %v", err))
	}

	return &cfg
}
