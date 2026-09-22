package investigation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	deployments "observability-mcp/internal/deployments"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
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

func TestLiveInvestigation(t *testing.T) {
	promURL := loadEnvValue("PROMETHEUS_URL", "http://127.0.0.1:9090")
	lokiURL := loadEnvValue("LOKI_URL", "http://127.0.0.1:3100")
	deployURL := loadEnvValue("ORDERFLOW_URL", "http://127.0.0.1:8080")

	promClient := prom.NewClient(promURL)
	lokiClient := loki.NewClient(lokiURL)
	deployClient := deployments.NewClient(deployURL)

	svc := NewService(promClient, lokiClient, deployClient)
	ctx := context.Background()

	inv, err := svc.Investigate(ctx, "orderflow", 10*time.Minute)
	if err != nil {
		t.Fatalf("Investigate failed: %v", err)
	}

	invJson, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		t.Fatalf("Marshal inv failed: %v", err)
	}
	fmt.Printf("--- INVESTIGATION ---\n%s\n", string(invJson))

	inc := BuildIncident(inv)
	incJson, err := json.MarshalIndent(inc, "", "  ")
	if err != nil {
		t.Fatalf("Marshal inc failed: %v", err)
	}
	fmt.Printf("--- INCIDENT ---\n%s\n", string(incJson))
}
