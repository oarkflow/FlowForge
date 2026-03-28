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

	// GRPCPort is the port the gRPC agent server listens on. Agents connect
	// to this port for registration, heartbeat, and job execution. Set to
	// empty string to disable the gRPC server. Default: "9090".
	GRPCPort string `mapstructure:"grpc_port"`

	// Vault settings for HashiCorp Vault integration.
	VaultAddr       string `mapstructure:"vault_addr"`
	VaultToken      string `mapstructure:"vault_token"`
	VaultAuthMethod string `mapstructure:"vault_auth_method"` // "token" or "approle"
	VaultRoleID     string `mapstructure:"vault_role_id"`
	VaultSecretID   string `mapstructure:"vault_secret_id"`
	VaultMountPath  string `mapstructure:"vault_mount_path"`
	VaultPrefix     string `mapstructure:"vault_prefix"`

	// AWS Secrets Manager settings.
	AWSRegion          string `mapstructure:"aws_region"`
	AWSAccessKeyID     string `mapstructure:"aws_access_key_id"`
	AWSSecretAccessKey string `mapstructure:"aws_secret_access_key"`
	AWSSecretsPrefix   string `mapstructure:"aws_secrets_prefix"`

	// GCP Secret Manager settings.
	GCPProjectID      string `mapstructure:"gcp_project_id"`
	GCPCredentialsFile string `mapstructure:"gcp_credentials_file"`
	GCPSecretsPrefix  string `mapstructure:"gcp_secrets_prefix"`

	// WebhookIPAllowlist is a comma-separated list of CIDR ranges allowed
	// to call webhook endpoints. Empty means allow all. This serves as the
	// static/fallback allowlist; DB-stored entries take precedence.
	WebhookIPAllowlist string `mapstructure:"webhook_ip_allowlist"`

	// SecretRotationCheckInterval controls how often the rotation tracker
	// checks for overdue secrets. Default: 1h.
	SecretRotationCheckInterval time.Duration `mapstructure:"secret_rotation_check_interval"`

	// Log Retention
	GlobalLogRetentionDays int `mapstructure:"global_log_retention_days"` // default 90

	// Log Forwarding
	LogForwardingEnabled bool   `mapstructure:"log_forwarding_enabled"`
	LogForwardingType    string `mapstructure:"log_forwarding_type"`     // loki, elasticsearch, splunk, cloudwatch
	LogForwardingURL     string `mapstructure:"log_forwarding_url"`
	LogForwardingToken   string `mapstructure:"log_forwarding_token"`
	LogForwardingIndex   string `mapstructure:"log_forwarding_index"`

	// Backup
	BackupDir      string `mapstructure:"backup_dir"`
	BackupInterval string `mapstructure:"backup_interval"` // e.g. "24h"

	// Cache
	CacheDir   string `mapstructure:"cache_dir"`
	CacheMaxMB int64  `mapstructure:"cache_max_mb"` // max cache size in MB

	// Multi-tenant
	MultiTenantEnabled bool `mapstructure:"multi_tenant_enabled"`

	// Rate Limiting (per-user, requests per minute)
	RateLimitAdmin     int `mapstructure:"rate_limit_admin"`
	RateLimitDeveloper int `mapstructure:"rate_limit_developer"`
	RateLimitViewer    int `mapstructure:"rate_limit_viewer"`
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
	v.SetDefault("grpc_port", "9090")

	// External secret provider defaults
	v.SetDefault("vault_addr", "")
	v.SetDefault("vault_token", "")
	v.SetDefault("vault_auth_method", "token")
	v.SetDefault("vault_role_id", "")
	v.SetDefault("vault_secret_id", "")
	v.SetDefault("vault_mount_path", "secret")
	v.SetDefault("vault_prefix", "flowforge")
	v.SetDefault("aws_region", "")
	v.SetDefault("aws_access_key_id", "")
	v.SetDefault("aws_secret_access_key", "")
	v.SetDefault("aws_secrets_prefix", "flowforge/")
	v.SetDefault("gcp_project_id", "")
	v.SetDefault("gcp_credentials_file", "")
	v.SetDefault("gcp_secrets_prefix", "flowforge-")
	v.SetDefault("webhook_ip_allowlist", "")
	v.SetDefault("secret_rotation_check_interval", 1*time.Hour)

	// New feature defaults
	v.SetDefault("global_log_retention_days", 90)
	v.SetDefault("log_forwarding_enabled", false)
	v.SetDefault("log_forwarding_type", "")
	v.SetDefault("log_forwarding_url", "")
	v.SetDefault("log_forwarding_token", "")
	v.SetDefault("log_forwarding_index", "flowforge-logs")
	v.SetDefault("backup_dir", "data/backups")
	v.SetDefault("backup_interval", "24h")
	v.SetDefault("cache_dir", "/tmp/flowforge-cache")
	v.SetDefault("cache_max_mb", 1024)
	v.SetDefault("multi_tenant_enabled", false)
	v.SetDefault("rate_limit_admin", 1000)
	v.SetDefault("rate_limit_developer", 200)
	v.SetDefault("rate_limit_viewer", 100)

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
