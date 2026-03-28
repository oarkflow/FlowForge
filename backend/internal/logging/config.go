package logging

import "time"

// ForwarderConfig holds the configuration for log forwarding.
type ForwarderConfig struct {
	Type          string        `mapstructure:"type"`           // loki, elasticsearch, splunk, cloudwatch
	Endpoint      string        `mapstructure:"endpoint"`       // URL of the log destination
	AuthToken     string        `mapstructure:"auth_token"`     // Bearer token or API key
	AuthUsername  string        `mapstructure:"auth_username"`  // Basic auth username
	AuthPassword  string        `mapstructure:"auth_password"`  // Basic auth password
	BatchSize     int           `mapstructure:"batch_size"`     // Number of log entries per batch
	FlushInterval time.Duration `mapstructure:"flush_interval"` // How often to flush logs
	Labels        map[string]string `mapstructure:"labels"`     // Extra labels (for Loki)
	Index         string        `mapstructure:"index"`          // Elasticsearch index name
	Stream        string        `mapstructure:"stream"`         // CloudWatch log stream name
	LogGroup      string        `mapstructure:"log_group"`      // CloudWatch log group name
	Region        string        `mapstructure:"region"`         // AWS region for CloudWatch
}

// LogEntry represents a single log line to be forwarded.
type LogEntry struct {
	RunID     string    `json:"run_id"`
	StepRunID string    `json:"step_run_id,omitempty"`
	Stream    string    `json:"stream"` // stdout, stderr, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
