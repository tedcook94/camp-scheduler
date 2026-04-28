package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Port     int    `envconfig:"PORT" default:"9100"`
	Mode     string `envconfig:"MODE" default:"prod"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

// AuthConfig configures how the Go server talks to the BetterAuth-based
// auth-server. ServerURL is the base URL (used both to fetch JWKS and to
// call the internal admin API for organization sync). SharedSecret is the
// bearer secret accepted by the auth-server's `/internal/*` routes.
//
// These fields are validated at server construction (`server.New`) rather
// than at config load, so test/CLI callers that don't need an authenticated
// server (integration tests, seed scripts) can still load config.
type AuthConfig struct {
	ServerURL           string        `envconfig:"AUTH_SERVER_URL"`
	SharedSecret        string        `envconfig:"AUTH_SHARED_SECRET"`
	JWKSTimeout         time.Duration `envconfig:"AUTH_JWKS_TIMEOUT" default:"5s"`
	RevocationCacheTTL  time.Duration `envconfig:"AUTH_REVOCATION_CACHE_TTL" default:"30s"`
	RevocationTimeout   time.Duration `envconfig:"AUTH_REVOCATION_TIMEOUT" default:"5s"`
}

// JWKSURL is the JWKS endpoint exposed by the auth-server.
func (c AuthConfig) JWKSURL() string {
	return strings.TrimRight(c.ServerURL, "/") + "/api/auth/jwks"
}

// InternalURL is the base of the auth-server's internal admin API.
func (c AuthConfig) InternalURL() string {
	return strings.TrimRight(c.ServerURL, "/") + "/internal"
}

type DatabaseConfig struct {
	// URL takes precedence over the discrete Host/User/etc. fields when set.
	// In normal operation it's APP_DATABASE_URL (the runtime URL using the
	// `app_user` role whose search_path is fixed to the `app` schema). The
	// discrete fields remain available as a fallback for callers that
	// configure the database piecemeal.
	URL      string        `envconfig:"APP_DATABASE_URL"`
	Host     string        `envconfig:"DATABASE_HOST"`
	Port     uint          `envconfig:"DATABASE_PORT"`
	User     string        `envconfig:"DATABASE_USER"`
	Password string        `envconfig:"DATABASE_PASSWORD"`
	SSLMode  string        `envconfig:"DATABASE_SSL_MODE"`
	Name     string        `envconfig:"DATABASE_NAME"`
	Timeout  time.Duration `envconfig:"DATABASE_TIMEOUT" default:"10s"`
}

func Load() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return cfg, err
}

func (c DatabaseConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
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
