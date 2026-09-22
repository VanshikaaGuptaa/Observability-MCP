package main

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	PrometheusURL string
	LokiURL       string
	OrderFlowURL  string
}

func loadDotEnv() {
	paths := []string{".env", "../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				if _, exists := os.LookupEnv(key); !exists {
					_ = os.Setenv(key, val)
				}
			}
		}
		break
	}
}

func LoadConfig() (*Config, error) {
	loadDotEnv()

	config := &Config{
		PrometheusURL: os.Getenv("PROMETHEUS_URL"),
		LokiURL:       os.Getenv("LOKI_URL"),
		OrderFlowURL:  os.Getenv("ORDERFLOW_URL"),
	}

	if config.PrometheusURL == "" {
		return nil, fmt.Errorf("PROMETHEUS_URL is required (can be set in .env)")
	}

	if config.LokiURL == "" {
		return nil, fmt.Errorf("LOKI_URL is required (can be set in .env)")
	}

	if config.OrderFlowURL == "" {
		return nil, fmt.Errorf("ORDERFLOW_URL is required (can be set in .env)")
	}

	return config, nil
}
