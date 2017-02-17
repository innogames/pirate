package pirate

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
)

type Config struct {
	UdpAddress     string        `yaml:"udp_address"`
	GraphiteTarget string        `yaml:"graphite_target"`
	Gzip           bool          `yaml:"gzip"`
	LogLevelStr    string        `yaml:"log_level"`
	LogLevel       logging.Level `yaml:"-"`
	Projects       map[string]*ProjectConfig
}

type ProjectConfig struct {
	GraphitePattern  string        `yaml:"graphite_path"`
	GraphiteTemplate *pathTemplate `yaml:"-"`
	Metrics          map[string]*MetricConfig
	Attributes       map[string]string
	AttributesRegex  map[string]*regexp.Regexp `yaml:"-"`
}

type MetricConfig struct {
	GraphitePattern  string        `yaml:"graphite_path"`
	GraphiteTemplate *pathTemplate `yaml:"-"`
	Min              float64       `yaml:"min"`
	Max              float64       `yaml:"max"`
}

func LoadConfig(filename string) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config file from %s: %s", filename, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("Failed to parse configuration file: %s", err)
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
	logger.Debugf("[Config] UDP Address: %s", cfg.UdpAddress)
	logger.Debugf("[Config] Graphite Target: %s", cfg.GraphiteTarget)
	logger.Debugf("[Config] Projects:")

	for pid, project := range cfg.Projects {
		logger.Debugf("[Config]   - %s", pid)

		for mid, metric := range project.Metrics {
			logger.Debugf("[Config]     - %s [min=%.0f max=%.0f path=%s]", mid, metric.Min, metric.Max, metric.GraphitePattern)
		}
	}
}
