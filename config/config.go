package config

import (
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

	return cfg, nil
}
