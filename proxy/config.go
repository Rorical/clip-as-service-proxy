package proxy

import (
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Config struct {
	ClipServiceURIs    []string `yaml:"clip-service-uris"`
	LoadBalanceMode    int      `yaml:"load-balance-mode"`
	TargetBatchSize    int      `yaml:"target-batch-size"`
	ServiceType        int      `yaml:"service-type"`
	TargetBatchTimeout int      `yaml:"target-batch-timeout"`
}

func ReadConfig(path string) (*Config, error) {
	var config Config
	confile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer confile.Close()
	content, err := io.ReadAll(confile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(content, &config)
	return &config, err
}
