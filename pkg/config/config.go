package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// GitCheck is a structure type to store config for a Git check
type GitCheckConfig struct {
	Name     string `yaml:"name"`
	Url      string `yaml:"url"`
	Revision string `yaml:"revision"`
	Path     string `yaml:"path"`
	Token    string `yaml:"token"`
}

// QuayCheck is a structure type to store config for a Quay check
type QuayCheckConfig struct {
	Name     string   `yaml:"name"`
	PullSpec string   `yaml:"pullspec"`
	Tags     []string `yaml:"tags"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
}

// GitCheck is a structure type to store config for a Git check
type HttpCheckConfig struct {
	Name     string `yaml:"name"`
	Url      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Cert     string `yaml:"cert"`
	Key      string `yaml:"key"`
	Insecure bool   `yaml:"insecure"`
	Follow   bool   `yaml:"follow_redirect"`
}

// ServiceConfig is a structure type to store the configs for the service
type ServiceConfig struct {
	// service:map[listen_port:8080 pool_interval:60]
	ListenPort    int    `yaml:"listen_port"`
	PollInterval  int    `yaml:"pool_interval"`
	MetricsPrefix string `yaml:"metrics_prefix"`
}

// CheckConfig is a structure type to store check configuration
type CheckConfig struct {
	Git  []GitCheckConfig  `yaml:"git"`
	Quay []QuayCheckConfig `yaml:"quay"`
	Http []HttpCheckConfig `yaml:"http"`
}

type Config struct {
	Service ServiceConfig `yaml:"service"`
	Checks  CheckConfig   `yaml:"checks"`
}

func LoadConfig(configFile string) (Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
