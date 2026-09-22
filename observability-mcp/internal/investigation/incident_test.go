package investigation

import (
	"testing"
	"time"

	prom "observability-mcp/internal/prometheus"
)

func TestBuildIncident(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.20)
	investigation.Metrics.P95Latency = prom.Float64Ptr(1.5)
	investigation.Metrics.P99Latency = prom.Float64Ptr(3.0)

	investigation.RecentErrors = []Error{
		{
			Timestamp: "2026-09-21T10:00:00Z",
			Level:     "ERROR",
			Message:   "payment provider unavailable",
		},
	}

	investigation.RecentDeployments = []Deployment{
		{
			ID:          "deploy-003",
			Version:     "v1.2.0",
			Environment: "production",
			Timestamp:   time.Now().Add(-5 * time.Minute),
			Status:      "success",
		},
	}

	investigation.Diagnosis = Diagnose(
		investigation,
		10*time.Minute,
	)

	incident := BuildIncident(
		investigation,
	)

	if incident == nil {
		t.Fatal("expected incident")
	}

	if incident.Status != "degraded" {
		t.Fatalf(
			"expected degraded status, got %s",
			incident.Status,
		)
	}

	if incident.Severity != "high" {
		t.Fatalf(
			"expected high severity, got %s",
			incident.Severity,
		)
	}

	if len(incident.Evidence) == 0 {
		t.Fatal("expected evidence")
	}

	if len(incident.Hypotheses) == 0 {
		t.Fatal("expected hypotheses")
	}
}

func TestBuildIncidentHistoricalRecovery(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
		Dependencies: []DependencyStatus{
			{
				Name:                 "redis",
				Status:               "healthy",
				CurrentStatus:        "healthy",
				WindowStatus:         "degraded",
				HistoricalErrorCount: 21,
				RecentErrorCount:     0,
			},
		},
	}
	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.0)
	investigation.Diagnosis = Diagnose(investigation, 10*time.Minute)

	incident := BuildIncident(investigation)
	if incident == nil {
		t.Fatal("expected incident")
	}

	// Should not report active dependency failure hypothesis
	for _, h := range incident.Hypotheses {
		if h.Cause == "redis dependency failure" && h.Confidence == "supported" {
			t.Fatalf("expected resolved/historical hypothesis, got active: %+v", h)
		}
	}
}
