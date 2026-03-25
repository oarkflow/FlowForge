package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port              string        `mapstructure:"port"`
	DatabasePath      string        `mapstructure:"database_path"`
	JWTSecret         string        `mapstructure:"jwt_secret"`
	JWTExpiration     time.Duration `mapstructure:"jwt_expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
	EncryptionKey     string        `mapstructure:"encryption_key"`
	AllowedOrigins    string        `mapstructure:"allowed_origins"`
	LogLevel          string        `mapstructure:"log_level"`
	MaxUploadSize     int           `mapstructure:"max_upload_size"`

	// EmbeddedWorker enables the in-process worker that executes jobs
	// directly in the server process. When false, jobs are only dispatched
	// to external agents via gRPC. Default: true.
	EmbeddedWorker bool `mapstructure:"embedded_worker"`
}

func Load() *Config {
	v := viper.New()

	v.SetDefault("port", "8081")
	v.SetDefault("database_path", "data/flowforge.db")
	v.SetDefault("jwt_secret", "change-me-in-production")
	v.SetDefault("jwt_expiration", 15*time.Minute)
	v.SetDefault("refresh_expiration", 7*24*time.Hour)
	v.SetDefault("encryption_key", "0000000000000000000000000000000000000000000000000000000000000000")
	v.SetDefault("allowed_origins", "*")
	v.SetDefault("log_level", "info")
	v.SetDefault("max_upload_size", 50*1024*1024)
	v.SetDefault("embedded_worker", true)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/flowforge")

	v.SetEnvPrefix("FLOWFORGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.ReadInConfig()

	cfg := &Config{}
	_ = v.Unmarshal(cfg)
	return cfg
}
