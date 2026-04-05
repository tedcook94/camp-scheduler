package config

import (
	"fmt"
	"net/url"
	"strconv"
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
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Name,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	q.Set("connect_timeout", strconv.Itoa(int(c.Timeout.Seconds())))
	u.RawQuery = q.Encode()
	return u.String()
}
