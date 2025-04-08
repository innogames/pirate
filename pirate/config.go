package pirate

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
	"time"
)

type Config struct {
	UdpAddress         string           `yaml:"udp_address"`
	GraphiteTarget     string           `yaml:"graphite_target"`
	PerIpRateLimit     *RateLimitConfig `yaml:"per_ip_ratelimit"`
	Gzip               bool             `yaml:"gzip"`
	LogLevelStr        string           `yaml:"log_level"`
	LogLevel           logging.Level    `yaml:"-"`
	MonitoringEnabled  bool             `yaml:"monitoring_enabled"`
	MonitoringPattern  string           `yaml:"monitoring_path"`
	MonitoringTemplate *pathTemplate    `yaml:"-"`
	Projects           map[string]*ProjectConfig
}

type ProjectConfig struct {
	GraphitePattern  string                    `yaml:"graphite_path"`
	GraphiteTemplate *pathTemplate             `yaml:"-"`
	Metrics          map[string]*MetricConfig  `yaml:"metrics"`
	Attributes       map[string]string         `yaml:"attributes"`
	AttributesRegex  map[string]*regexp.Regexp `yaml:"-"`
}

type MetricConfig struct {
	GraphitePattern  string        `yaml:"graphite_path"`
	GraphiteTemplate *pathTemplate `yaml:"-"`
	Min              float64       `yaml:"min"`
	Max              float64       `yaml:"max"`
}

type RateLimitConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Amount   int           `yaml:"amount"`
	Interval time.Duration `yaml:"interval"`
}

var DefaultConfig = Config{
	UdpAddress:        "0.0.0.0:33333",
	GraphiteTarget:    "tcp://127.0.0.1:3002",
	MonitoringEnabled: true,
	MonitoringPattern: "pirate.{metric.name}",
	Gzip:              true,
	LogLevelStr:       "info",
	PerIpRateLimit: &RateLimitConfig{
		Enabled:  true,
		Amount:   100,
		Interval: 1 * time.Minute,
	},
	Projects: make(map[string]*ProjectConfig),
}

func LoadConfig(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config file from %s: %s", filename, err)
	}

	cfg := &DefaultConfig
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse configuration file: %s", err)
	}

	// initialize monitoring template
	if cfg.MonitoringEnabled {
		if cfg.MonitoringTemplate, err = ParsePathTemplate([]byte(cfg.MonitoringPattern)); err != nil {
			return nil, fmt.Errorf(`Invalid path for "monitoring_path": %s`, err)
		}
	}

	// initialize regexps and templates
	for pid, project := range cfg.Projects {
		// initialize graphite path templates
		if project.GraphiteTemplate, err = ParsePathTemplate([]byte(project.GraphitePattern)); err != nil {
			return nil, fmt.Errorf(`Invalid path for "projects.%s.graphite_path": %s`, pid, err)
		}

		// initialize attribute regexps
		project.AttributesRegex = make(map[string]*regexp.Regexp)
		for aid, attr := range project.Attributes {
			if cfg.Projects[pid].AttributesRegex[aid], err = regexp.Compile(attr); err != nil {
				return nil, fmt.Errorf(`Invalid regexp for "projects.%s.attributes.%s": %s`, pid, aid, err)
			}
		}

		// initialize graphite path templates for metrics
		for mid, metric := range project.Metrics {
			// use same template from project, if not overridden
			if metric.GraphitePattern == "" {
				metric.GraphitePattern = project.GraphitePattern
				metric.GraphiteTemplate = project.GraphiteTemplate
			} else {
				// compile custom template for metric
				if metric.GraphiteTemplate, err = ParsePathTemplate([]byte(metric.GraphitePattern)); err != nil {
					return nil, fmt.Errorf(`Invalid path for "projects.%s.%s.graphite_path": %s`, pid, mid, err)
				}
			}
		}
	}

	// initialize log level
	cfg.LogLevel = logging.WARNING
	if cfg.LogLevelStr != "" {
		cfg.LogLevel, err = logging.LogLevel(cfg.LogLevelStr)
		if err != nil {
			return nil, errors.New("Invalid log level. Allowed levels are: DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL")
		}
	}

	return cfg, nil
}

func (cfg *Config) Log(logger *logging.Logger) {
	logger.Infof("[Config] UDP Address: %s", cfg.UdpAddress)
	logger.Infof("[Config] Graphite Target: %s", cfg.GraphiteTarget)
	logger.Infof("[Config] UDP Rate Limit: %d metrics per %s per IP", cfg.PerIpRateLimit.Amount, cfg.PerIpRateLimit.Interval)
	logger.Infof("[Config] Projects:")

	for pid, project := range cfg.Projects {
		logger.Infof("[Config]   - %s", pid)

		for mid, metric := range project.Metrics {
			logger.Infof("[Config]     - %s [min=%.0f max=%.0f path=%s]", mid, metric.Min, metric.Max, metric.GraphitePattern)
		}
	}
}
