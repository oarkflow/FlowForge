package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// LogForwarder defines the interface for forwarding logs to external systems.
type LogForwarder interface {
	Forward(entries []LogEntry) error
	Close() error
}

// ForwarderManager manages log forwarding with batching and flushing.
type ForwarderManager struct {
	forwarders []LogForwarder
	buffer     []LogEntry
	mu         sync.Mutex
	batchSize  int
	flushTick  *time.Ticker
	done       chan struct{}
}

// NewForwarderManager creates a new ForwarderManager from the given configs.
func NewForwarderManager(configs []ForwarderConfig) *ForwarderManager {
	fm := &ForwarderManager{
		batchSize: 100,
		done:      make(chan struct{}),
	}

	for _, cfg := range configs {
		f := createForwarder(cfg)
		if f != nil {
			fm.forwarders = append(fm.forwarders, f)
			if cfg.BatchSize > 0 {
				fm.batchSize = cfg.BatchSize
			}
		}
	}

	return fm
}

// Start begins the background flushing goroutine.
func (fm *ForwarderManager) Start(ctx context.Context, interval time.Duration) {
	if len(fm.forwarders) == 0 {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	fm.flushTick = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-fm.flushTick.C:
				fm.Flush()
			case <-ctx.Done():
				fm.Flush()
				return
			case <-fm.done:
				return
			}
		}
	}()
}

// Push adds a log entry to the buffer. Flushes when batch size is reached.
func (fm *ForwarderManager) Push(entry LogEntry) {
	if len(fm.forwarders) == 0 {
		return
	}
	fm.mu.Lock()
	fm.buffer = append(fm.buffer, entry)
	shouldFlush := len(fm.buffer) >= fm.batchSize
	fm.mu.Unlock()

	if shouldFlush {
		fm.Flush()
	}
}

// Flush sends all buffered log entries to all forwarders.
func (fm *ForwarderManager) Flush() {
	fm.mu.Lock()
	if len(fm.buffer) == 0 {
		fm.mu.Unlock()
		return
	}
	entries := fm.buffer
	fm.buffer = nil
	fm.mu.Unlock()

	for _, f := range fm.forwarders {
		if err := f.Forward(entries); err != nil {
			log.Error().Err(err).Msg("logging: failed to forward logs")
		}
	}
}

// Close stops the manager and all forwarders.
func (fm *ForwarderManager) Close() error {
	close(fm.done)
	if fm.flushTick != nil {
		fm.flushTick.Stop()
	}
	fm.Flush()
	for _, f := range fm.forwarders {
		_ = f.Close()
	}
	return nil
}

func createForwarder(cfg ForwarderConfig) LogForwarder {
	switch cfg.Type {
	case "loki":
		return NewLokiForwarder(cfg)
	case "elasticsearch":
		return NewElasticsearchForwarder(cfg)
	case "splunk":
		return NewSplunkForwarder(cfg)
	case "cloudwatch":
		return NewCloudWatchForwarder(cfg)
	default:
		log.Warn().Str("type", cfg.Type).Msg("logging: unknown forwarder type")
		return nil
	}
}

// --------------------------------------------------------------------------
// Loki Forwarder
// --------------------------------------------------------------------------

// LokiForwarder pushes logs to Grafana Loki via the HTTP push API.
type LokiForwarder struct {
	endpoint string
	labels   map[string]string
	client   *http.Client
	auth     string
}

// NewLokiForwarder creates a new Loki forwarder.
func NewLokiForwarder(cfg ForwarderConfig) *LokiForwarder {
	endpoint := strings.TrimRight(cfg.Endpoint, "/") + "/loki/api/v1/push"
	return &LokiForwarder{
		endpoint: endpoint,
		labels:   cfg.Labels,
		client:   &http.Client{Timeout: 10 * time.Second},
		auth:     cfg.AuthToken,
	}
}

func (l *LokiForwarder) Forward(entries []LogEntry) error {
	// Build Loki push payload
	// Group entries by run_id for separate streams
	streams := make(map[string][]interface{})
	for _, e := range entries {
		key := e.RunID
		ts := strconv.FormatInt(e.Timestamp.UnixNano(), 10)
		streams[key] = append(streams[key], []string{ts, e.Content})
	}

	var streamList []map[string]interface{}
	for runID, values := range streams {
		labels := map[string]string{
			"job":    "flowforge",
			"run_id": runID,
		}
		for k, v := range l.labels {
			labels[k] = v
		}
		streamList = append(streamList, map[string]interface{}{
			"stream": labels,
			"values": values,
		})
	}

	payload := map[string]interface{}{"streams": streamList}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("loki: marshal: %w", err)
	}

	req, err := http.NewRequest("POST", l.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("loki: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if l.auth != "" {
		req.Header.Set("Authorization", "Bearer "+l.auth)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("loki: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("loki: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (l *LokiForwarder) Close() error { return nil }

// --------------------------------------------------------------------------
// Elasticsearch Forwarder
// --------------------------------------------------------------------------

// ElasticsearchForwarder bulk-indexes logs to Elasticsearch.
type ElasticsearchForwarder struct {
	endpoint string
	index    string
	client   *http.Client
	auth     string
}

// NewElasticsearchForwarder creates a new Elasticsearch forwarder.
func NewElasticsearchForwarder(cfg ForwarderConfig) *ElasticsearchForwarder {
	index := cfg.Index
	if index == "" {
		index = "flowforge-logs"
	}
	return &ElasticsearchForwarder{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		index:    index,
		client:   &http.Client{Timeout: 10 * time.Second},
		auth:     cfg.AuthToken,
	}
}

func (e *ElasticsearchForwarder) Forward(entries []LogEntry) error {
	var buf bytes.Buffer
	for _, entry := range entries {
		// Bulk API: action line + document line
		action := fmt.Sprintf(`{"index":{"_index":"%s"}}`, e.index)
		buf.WriteString(action + "\n")
		doc, _ := json.Marshal(map[string]interface{}{
			"run_id":      entry.RunID,
			"step_run_id": entry.StepRunID,
			"stream":      entry.Stream,
			"content":     entry.Content,
			"@timestamp":  entry.Timestamp.Format(time.RFC3339Nano),
		})
		buf.Write(doc)
		buf.WriteString("\n")
	}

	req, err := http.NewRequest("POST", e.endpoint+"/_bulk", &buf)
	if err != nil {
		return fmt.Errorf("elasticsearch: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if e.auth != "" {
		req.Header.Set("Authorization", "Bearer "+e.auth)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("elasticsearch: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *ElasticsearchForwarder) Close() error { return nil }

// --------------------------------------------------------------------------
// Splunk HEC Forwarder
// --------------------------------------------------------------------------

// SplunkForwarder sends logs via the Splunk HTTP Event Collector.
type SplunkForwarder struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewSplunkForwarder creates a new Splunk forwarder.
func NewSplunkForwarder(cfg ForwarderConfig) *SplunkForwarder {
	return &SplunkForwarder{
		endpoint: strings.TrimRight(cfg.Endpoint, "/") + "/services/collector/event",
		token:    cfg.AuthToken,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SplunkForwarder) Forward(entries []LogEntry) error {
	var buf bytes.Buffer
	for _, entry := range entries {
		event := map[string]interface{}{
			"event": map[string]interface{}{
				"run_id":      entry.RunID,
				"step_run_id": entry.StepRunID,
				"stream":      entry.Stream,
				"content":     entry.Content,
			},
			"time":       entry.Timestamp.Unix(),
			"sourcetype": "flowforge:pipeline",
		}
		data, _ := json.Marshal(event)
		buf.Write(data)
	}

	req, err := http.NewRequest("POST", s.endpoint, &buf)
	if err != nil {
		return fmt.Errorf("splunk: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("splunk: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("splunk: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *SplunkForwarder) Close() error { return nil }

// --------------------------------------------------------------------------
// CloudWatch Logs Forwarder
// --------------------------------------------------------------------------

// CloudWatchForwarder pushes logs to AWS CloudWatch Logs.
// Note: For production, use the AWS SDK. This is a simplified HTTP-based implementation.
type CloudWatchForwarder struct {
	logGroup  string
	logStream string
	region    string
	auth      string
	client    *http.Client
}

// NewCloudWatchForwarder creates a new CloudWatch forwarder.
func NewCloudWatchForwarder(cfg ForwarderConfig) *CloudWatchForwarder {
	return &CloudWatchForwarder{
		logGroup:  cfg.LogGroup,
		logStream: cfg.Stream,
		region:    cfg.Region,
		auth:      cfg.AuthToken,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (cw *CloudWatchForwarder) Forward(entries []LogEntry) error {
	// Build PutLogEvents payload
	var events []map[string]interface{}
	for _, entry := range entries {
		events = append(events, map[string]interface{}{
			"timestamp": entry.Timestamp.UnixMilli(),
			"message":   fmt.Sprintf("[%s][%s] %s", entry.RunID, entry.Stream, entry.Content),
		})
	}

	payload := map[string]interface{}{
		"logGroupName":  cw.logGroup,
		"logStreamName": cw.logStream,
		"logEvents":     events,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cloudwatch: marshal: %w", err)
	}

	endpoint := fmt.Sprintf("https://logs.%s.amazonaws.com/", cw.region)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cloudwatch: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.PutLogEvents")
	if cw.auth != "" {
		req.Header.Set("Authorization", cw.auth)
	}

	resp, err := cw.client.Do(req)
	if err != nil {
		return fmt.Errorf("cloudwatch: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloudwatch: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (cw *CloudWatchForwarder) Close() error { return nil }
