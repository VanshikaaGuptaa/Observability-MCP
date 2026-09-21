package main

import (
	"fmt"
	"os"
)

type Config struct {
	PrometheusURL string
	LokiURL       string
	OrderFlowURL  string
}

func LoadConfig() (*Config, error) {
	config := &Config{
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
		LokiURL:       os.Getenv("LOKI_URL"),
		OrderFlowURL:  os.Getenv("ORDERFLOW_URL"),
	}

	if config.PrometheusURL == "" {
		return nil, fmt.Errorf("PROMETHEUS_URL is required")
	}

	if config.LokiURL == "" {
		return nil, fmt.Errorf("LOKI_URL is required")
	}

	if config.OrderFlowURL == "" {
		return nil, fmt.Errorf("ORDERFLOW_URL is required")
	}

	return config, nil
}
