package config

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PIServer struct {
		BaseURL     string `yaml:"base_url"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		InsecureTLS bool   `yaml:"insecure_tls"`
	} `yaml:"pi_server"`
	Alerts struct {
		Enabled    bool   `yaml:"enabled"`
		WebhookURL string `yaml:"webhook_url"`
	} `yaml:"alerts"`
	Gateways []Gateway `yaml:"gateways"`
}

type Gateway struct {
	Name         string `yaml:"name"`
	Address      string `yaml:"address"`
	LocalAddress string `yaml:"localAddress"`
	SlaveID      byte   `yaml:"slave_id"`
	PollInterval string `yaml:"poll_interval"`
	Tags         []Tag  `yaml:"tags"`
}

type Tag struct {
	Name           string `yaml:"name"`
	Register       uint16 `yaml:"register"`
	OMFContainerID string `yaml:"omf_container_id"`
	PIWebID        string `yaml:"pi_web_id"`
	DeviceType     string `yaml:"device_type"`
}

//go:embed config.yaml
var DefaultConfig []byte

func Load() (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(DefaultConfig, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
