package prometheus

import (
	"context"
	"os"
	"strings"
	"testing"
)

func loadEnvValue(key, fallback string) string {
	paths := []string{".env", "../../.env", "../../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
				val := strings.TrimSpace(parts[1])
				return strings.Trim(val, `"'`)
			}
		}
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestQuery(t *testing.T) {
	promURL := loadEnvValue("PROMETHEUS_URL", "http://127.0.0.1:9090")
	client := NewClient(promURL)

	result, err := client.Query(
		context.Background(),
		`up{job="orderflow"}`,
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(result.Data.Result) == 0 {
		t.Fatal("expected orderflow result")
	}

	t.Logf("Prometheus result: %+v", result.Data.Result)
}
