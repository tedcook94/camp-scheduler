package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Port     int    `envconfig:"PORT" default:"9100"`
	Mode     string `envconfig:"MODE" default:"prod"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

type DatabaseConfig struct {
	Host     string        `envconfig:"DATABASE_HOST" required:"true"`
	Port     uint          `envconfig:"DATABASE_PORT" required:"true"`
	User     string        `envconfig:"DATABASE_USER" required:"true"`
	Password string        `envconfig:"DATABASE_PASSWORD"`
	SSLMode  string        `envconfig:"DATABASE_SSL_MODE" required:"true"`
	Name     string        `envconfig:"DATABASE_NAME" required:"true"`
	Timeout  time.Duration `envconfig:"DATABASE_TIMEOUT" default:"10s"`
}

func Load() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return cfg, err
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
		int(c.Timeout.Seconds()),
	)
}
