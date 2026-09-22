package investigation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	deployments "observability-mcp/internal/deployments"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
)

func TestLiveInvestigation(t *testing.T) {
	promClient := prom.NewClient("http://localhost:9090")
	lokiClient := loki.NewClient("http://localhost:3100")
	deployClient := deployments.NewClient("http://localhost:8080")

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
