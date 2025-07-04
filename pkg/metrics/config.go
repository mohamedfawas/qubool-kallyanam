package metrics

// Config holds metrics configuration
type Config struct {
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled"`
	ServiceName string `mapstructure:"service_name" yaml:"service_name"`
	Port        int    `mapstructure:"port" yaml:"port"`
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		ServiceName: "unknown",
		Port:        9090,
	}
}
