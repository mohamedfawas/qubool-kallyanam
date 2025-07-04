// pkg/tracing/config.go
package tracing

// Config holds tracing configuration following your existing pattern
type Config struct {
	Enabled     bool    `mapstructure:"enabled" yaml:"enabled"`
	ServiceName string  `mapstructure:"service_name" yaml:"service_name"`
	Environment string  `mapstructure:"environment" yaml:"environment"`
	JaegerURL   string  `mapstructure:"jaeger_url" yaml:"jaeger_url"`
	ProjectID   string  `mapstructure:"project_id" yaml:"project_id"` // For GCP
	SampleRate  float64 `mapstructure:"sample_rate" yaml:"sample_rate"`
}

// DefaultConfig returns default tracing configuration
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		ServiceName: "unknown",
		Environment: "development",
		JaegerURL:   "http://localhost:14268/api/traces",
		ProjectID:   "",
		SampleRate:  1.0, // 100% sampling for development
	}
}
