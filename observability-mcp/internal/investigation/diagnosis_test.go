package investigation

import (
	"testing"
	"time"

	prom "observability-mcp/internal/prometheus"
)

func TestDiagnoseHealthyService(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Health.Status = "healthy"

	investigation.Metrics.RequestRate = prom.Float64Ptr(0.5)
	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.0)
	investigation.Metrics.P95Latency = prom.Float64Ptr(0.2)
	investigation.Metrics.P99Latency = prom.Float64Ptr(0.4)

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	if result == nil {
		t.Fatal("expected diagnosis")
	}

	if result.Severity != "normal" {
		t.Fatalf(
			"expected normal severity, got %s",
			result.Severity,
		)
	}

	if len(result.PossibleCauses) != 0 {
		t.Fatalf(
			"expected no possible causes, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnosePaymentFailure(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.20)
	investigation.Metrics.P95Latency = prom.Float64Ptr(0.5)
	investigation.Metrics.P99Latency = prom.Float64Ptr(0.8)

	investigation.RecentErrors = []Error{
		{
			Timestamp: "2026-09-21T10:00:00Z",
			Level:     "ERROR",
			Message:   "payment provider unavailable",
		},
		{
			Timestamp: "2026-09-21T10:01:00Z",
			Level:     "ERROR",
			Message:   "payment request failed",
		},
	}

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	if result.Severity != "high" {
		t.Fatalf(
			"expected high severity, got %s",
			result.Severity,
		)
	}

	foundPaymentCause := false

	for _, cause := range result.PossibleCauses {
		if cause == "Payment dependency failures detected" {
			foundPaymentCause = true
			break
		}
	}

	if !foundPaymentCause {
		t.Fatalf(
			"expected payment dependency cause, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnoseRedisFailure(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.10)

	investigation.RecentErrors = []Error{
		{
			Timestamp: "2026-09-21T10:00:00Z",
			Level:     "ERROR",
			Message:   "redis connection failed",
		},
	}

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	if result.Severity != "high" {
		t.Fatalf(
			"expected high severity, got %s",
			result.Severity,
		)
	}

	found := false

	for _, cause := range result.PossibleCauses {
		if cause == "Redis dependency errors detected" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"expected Redis cause, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnoseDatabaseFailure(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.15)

	investigation.RecentErrors = []Error{
		{
			Timestamp: "2026-09-21T10:00:00Z",
			Level:     "ERROR",
			Message:   "database connection timeout",
		},
	}

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	found := false

	for _, cause := range result.PossibleCauses {
		if cause == "Database errors detected" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"expected database cause, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnoseHighLatency(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.0)
	investigation.Metrics.P95Latency = prom.Float64Ptr(1.5)
	investigation.Metrics.P99Latency = prom.Float64Ptr(3.2)

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	if result.Severity != "medium" {
		t.Fatalf(
			"expected medium severity, got %s",
			result.Severity,
		)
	}

	found := false

	for _, cause := range result.PossibleCauses {
		if cause == "High request latency" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"expected latency cause, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnoseRecentDeploymentCorrelation(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}

	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.10)
	investigation.Metrics.P99Latency = prom.Float64Ptr(2.5)

	investigation.RecentDeployments = []Deployment{
		{
			ID:          "deploy-003",
			Version:     "v1.2.0",
			Environment: "production",
			Timestamp:   time.Now().Add(-5 * time.Minute),
			Status:      "success",
		},
	}

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	found := false

	for _, cause := range result.PossibleCauses {
		if cause == "Recent deployment temporally correlates with elevated error rate" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"expected deployment correlation, got %v",
			result.PossibleCauses,
		)
	}
}

func TestDiagnoseHistoricalRecovery(t *testing.T) {
	investigation := &Investigation{
		Service: "orderflow",
	}
	investigation.Metrics.ErrorRate = prom.Float64Ptr(0.0)
	investigation.Dependencies = []DependencyStatus{
		{
			Name:                 "redis",
			Status:               "healthy",
			CurrentStatus:        "healthy",
			WindowStatus:         "degraded",
			HistoricalErrorCount: 21,
			RecentErrorCount:     0,
		},
	}

	result := Diagnose(
		investigation,
		10*time.Minute,
	)

	if result.Severity == "high" {
		t.Fatalf("expected recovered dependency not to cause high severity, got %s", result.Severity)
	}

	foundHistoricalCause := false
	for _, cause := range result.PossibleCauses {
		if cause == "Historical redis dependency degradation detected (recovered)" {
			foundHistoricalCause = true
			break
		}
	}
	if !foundHistoricalCause {
		t.Fatalf("expected historical recovered cause, got %v", result.PossibleCauses)
	}
}
