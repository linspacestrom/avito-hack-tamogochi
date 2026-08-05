package config

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const (
	defaultConfigPath = "config/config.yaml"
	configPathEnv     = "CONFIG_PATH"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	HTTP     HTTPConfig     `yaml:"http"`
	Log      LogConfig      `yaml:"log"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Cookie   CookieConfig   `yaml:"cookie"`
}

type AppConfig struct {
	Name string `env:"APP_NAME" env-default:"app"`
	Env  string `env:"APP_ENV" env-default:"development"`
}

type HTTPConfig struct {
	Host string `env:"HTTP_HOST" env-default:"0.0.0.0" yaml:"host"`
	Port string `env:"HTTP_PORT" env-default:"8080" yaml:"port"`

	ReadTimeout     time.Duration `yaml:"read_timeout" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env-default:"5s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

func (c HTTPConfig) Addr() string {
	return net.JoinHostPort(c.Host, c.Port)
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"json"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" env-default:"localhost"`
	Port     string `env:"DB_PORT" env-default:"5432"`
	Username string `env:"DB_USERNAME" env-default:"timur"`
	Password string `env:"DB_PASSWORD" env-default:"Mars237s!"`
	Database string `env:"DB_NAME" env-default:"testings"`
	SSLMode  string `env:"DB_SSLMODE" env-default:"disable"`

	MaxOpenConns    int32         `yaml:"max_open_conns" env-default:"25"`
	MaxIdleConns    int32         `yaml:"max_idle_conns" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env-default:"1h"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" env-default:"30m"`

	Timeout time.Duration `yaml:"timeout" env-default:"5s"`
}

type AuthConfig struct {
	Secret          string        `env:"AUTH_SECRET" env_default:"secret"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" env-default:"720h"`
}

type CookieConfig struct {
	Name   string `env:"COOKIE_NAME" env-default:"session"`
	Domain string `env:"COOKIE_DOMAIN" env-default:"/"`
	Path   string `env:"COOKIE_PATH" env-default:"/"`

	Secure   bool `env:"COOKIE_SECURE" env-default:"false"`
	HTTPOnly bool `env:"COOKIE_HTTP_ONLY" env-default:"true"`

	SameSite string `env:"COOKIE_SAME_SITE" env-default:"Lax"`
}

func MustLoad() *Config {
	cfg, err := load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	return cfg
}

func load() (*Config, error) {
	if _, err := os.Stat(".env"); err == nil {
		if loadErr := godotenv.Load(); loadErr != nil {
			return nil, fmt.Errorf("failed to load .env: %w", loadErr)
		}
	}

	configPath := getConfigPath()
	if configPath != "" {
		fileInfo, err := os.Stat(configPath)

		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s does not exist", configPath)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to access config file: %w", err)
		}

		if fileInfo.IsDir() {
			return nil, fmt.Errorf("config path is a directory, not a file: %s", configPath)
		}
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read env vars: %w", err)
	}

	return &cfg, nil
}

func getConfigPath() string {
	configPath := os.Getenv(configPathEnv)
	if configPath == "" {
		configPath = defaultConfigPath
	}

	return configPath
}
