package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// LokiProvider implements telemetry.LoggingProvider using Grafana Loki
type LokiProvider struct {
	lokiURL       string
	httpClient    *http.Client
	batchSize     int
	flushInterval time.Duration
	labels        map[string]string
	entries       []logEntry
	mu            sync.Mutex
	stopCh        chan struct{}
	logger        logging.Logger
}

// Config holds configuration for LokiProvider
type Config struct {
	URL           string
	BatchSize     int
	FlushInterval time.Duration
	Labels        map[string]string
	Timeout       time.Duration
}

type logEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    map[string]interface{}
}

// NewLokiProvider creates a new LokiProvider
func NewLokiProvider(cfg Config, logger logging.Logger) *LokiProvider {
	if logger == nil {
		logger = logging.Get().Named("loki")
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}

	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	// Ensure we have at least some basic labels
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]string)
	}
	if _, ok := cfg.Labels["service"]; !ok {
		cfg.Labels["service"] = "matrimony"
	}

	provider := &LokiProvider{
		lokiURL:       cfg.URL,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		labels:        cfg.Labels,
		entries:       make([]logEntry, 0, cfg.BatchSize),
		stopCh:        make(chan struct{}),
		logger:        logger,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}

	// Start background worker for flushing logs
	go provider.flushWorker()

	return provider
}

// Name returns the name of the provider
func (p *LokiProvider) Name() string {
	return "loki"
}

// Shutdown gracefully shuts down the provider
func (p *LokiProvider) Shutdown(ctx context.Context) error {
	// Signal the flush worker to stop
	close(p.stopCh)

	// Flush remaining logs
	return p.flush(ctx)
}

// Log sends a log entry to Loki
func (p *LokiProvider) Log(ctx context.Context, level string, message string, fields map[string]interface{}) error {
	// Extract trace context if available
	if ctx != nil {
		// Add trace ID if present in the context
		// This is a simplified example - actual implementation depends on your tracing setup
		if traceID, ok := ctx.Value("trace_id").(string); ok && traceID != "" {
			if fields == nil {
				fields = make(map[string]interface{})
			}
			fields["trace_id"] = traceID
		}
	}

	// Create log entry
	entry := logEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	// Add to batch
	p.mu.Lock()
	p.entries = append(p.entries, entry)
	size := len(p.entries)
	p.mu.Unlock()

	// Flush if batch is full
	if size >= p.batchSize {
		go p.flush(context.Background())
	}

	return nil
}

// flush sends batched logs to Loki
func (p *LokiProvider) flush(ctx context.Context) error {
	p.mu.Lock()
	if len(p.entries) == 0 {
		p.mu.Unlock()
		return nil
	}

	// Swap entries with empty slice
	entries := p.entries
	p.entries = make([]logEntry, 0, p.batchSize)
	p.mu.Unlock()

	// Convert entries to Loki push format
	pushRequest := struct {
		Streams []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"streams"`
	}{
		Streams: []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		}{
			{
				Stream: p.labels,
				Values: make([][]string, 0, len(entries)),
			},
		},
	}

	// Add all log entries
	for _, entry := range entries {
		// Prepare log line as JSON
		logLine := struct {
			Timestamp time.Time              `json:"timestamp"`
			Level     string                 `json:"level"`
			Message   string                 `json:"message"`
			Fields    map[string]interface{} `json:"fields,omitempty"`
		}{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Message:   entry.Message,
			Fields:    entry.Fields,
		}

		lineJSON, err := json.Marshal(logLine)
		if err != nil {
			p.logger.Error("Failed to marshal log entry",
				logging.String("error", err.Error()),
				logging.String("message", entry.Message),
			)
			continue
		}

		// Format timestamp as nanoseconds since epoch
		timestampNano := fmt.Sprintf("%d", entry.Timestamp.UnixNano())

		// Add to values
		pushRequest.Streams[0].Values = append(
			pushRequest.Streams[0].Values,
			[]string{timestampNano, string(lineJSON)},
		)
	}

	// Marshal the request
	body, err := json.Marshal(pushRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal Loki push request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.lokiURL+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Loki push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to Loki: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Loki push failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// flushWorker periodically flushes logs
func (p *LokiProvider) flushWorker() {
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.flush(context.Background()); err != nil {
				p.logger.Error("Failed to flush logs to Loki", logging.Error(err))
			}
		case <-p.stopCh:
			return
		}
	}
}

// Common logging utilities for matrimony application

// LokiLogUserActivity logs user activity
func LokiLogUserActivity(ctx context.Context, provider *LokiProvider, userID, action, profileID string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["user_id"] = userID
	details["profile_id"] = profileID

	provider.Log(ctx, "info", fmt.Sprintf("User activity: %s", action), details)
}

// LokiLogMatchingEvent logs profile matching events
func LokiLogMatchingEvent(ctx context.Context, provider *LokiProvider, matchType string, userID string, matchCount int, duration time.Duration) {
	fields := map[string]interface{}{
		"match_type":  matchType,
		"user_id":     userID,
		"match_count": matchCount,
		"duration_ms": duration.Milliseconds(),
	}

	provider.Log(ctx, "info", "Profile matching completed", fields)
}
