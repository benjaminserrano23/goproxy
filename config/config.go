package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Routes []Route      `yaml:"routes"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type Route struct {
	Path        string   `yaml:"path"`
	Upstream    string   `yaml:"upstream"`
	Middlewares []string `yaml:"middlewares"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = "8080"
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if len(c.Routes) == 0 {
		return fmt.Errorf("config: at least one route is required")
	}

	for i, route := range c.Routes {
		if route.Path == "" {
			return fmt.Errorf("config: route[%d] missing path", i)
		}
		if route.Upstream == "" {
			return fmt.Errorf("config: route[%d] missing upstream", i)
		}
		if _, err := url.Parse(route.Upstream); err != nil {
			return fmt.Errorf("config: route[%d] invalid upstream URL %q: %w", i, route.Upstream, err)
		}
	}

	return nil
}
