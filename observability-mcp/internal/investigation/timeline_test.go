package investigation

import (
	"testing"
	"time"

	prom "observability-mcp/internal/prometheus"
)

func TestTimelineScenarios(t *testing.T) {
	now := time.Now()

	// Scenario 1: Normal
	t.Run("1_normal", func(t *testing.T) {
		inv := &Investigation{
			Service: "orderflow",
		}
		inv.Metrics.RequestRate = prom.Float64Ptr(10.0)
		inv.Metrics.ErrorRate = prom.Float64Ptr(0.0)
		inv.Metrics.P95Latency = prom.Float64Ptr(0.1)
		inv.Metrics.P99Latency = prom.Float64Ptr(0.2)
		inv.Dependencies = []DependencyStatus{
			{Name: "redis", CurrentStatus: "healthy", WindowStatus: "healthy"},
			{Name: "postgresql", CurrentStatus: "healthy", WindowStatus: "healthy"},
			{Name: "payment-provider", CurrentStatus: "healthy", WindowStatus: "healthy"},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) != 0 {
			t.Fatalf("expected 0 timeline events in normal scenario, got %d", len(timeline))
		}
	})

	// Scenario 2: Redis failure
	t.Run("2_redis_failure", func(t *testing.T) {
		inv := &Investigation{
			Service: "orderflow",
			RecentErrors: []Error{
				{
					Timestamp: now.Add(-3 * time.Minute).Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "failed to get product from cache: redis connection refused",
				},
			},
			Dependencies: []DependencyStatus{
				{
					Name:                 "redis",
					CurrentStatus:        "degraded",
					WindowStatus:         "degraded",
					HistoricalErrorCount: 5,
					RecentErrorCount:     5,
				},
			},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) == 0 {
			t.Fatal("expected timeline events for redis failure")
		}

		found := false
		for _, ev := range timeline {
			if ev.Dependency == "redis" && ev.Signal == "dependency_error" {
				found = true
				if ev.Severity != "warning" && ev.Severity != "critical" {
					t.Errorf("unexpected severity: %s", ev.Severity)
				}
			}
		}
		if !found {
			t.Fatalf("expected redis dependency_error event, got %+v", timeline)
		}
	})

	// Scenario 3: PostgreSQL failure
	t.Run("3_postgresql_failure", func(t *testing.T) {
		inv := &Investigation{
			Service: "orderflow",
			RecentErrors: []Error{
				{
					Timestamp: now.Add(-4 * time.Minute).Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "database query failed: pq: connection refused",
				},
			},
			Dependencies: []DependencyStatus{
				{
					Name:                 "postgresql",
					CurrentStatus:        "degraded",
					WindowStatus:         "degraded",
					HistoricalErrorCount: 12,
					RecentErrorCount:     12,
				},
			},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) == 0 {
			t.Fatal("expected timeline events for postgresql failure")
		}

		found := false
		for _, ev := range timeline {
			if ev.Dependency == "postgresql" && ev.Signal == "dependency_error" {
				found = true
				if ev.Severity != "critical" {
					t.Errorf("expected critical severity for postgresql failure, got %s", ev.Severity)
				}
			}
		}
		if !found {
			t.Fatalf("expected postgresql dependency_error event, got %+v", timeline)
		}
	})

	// Scenario 4: Payment failure
	t.Run("4_payment_failure", func(t *testing.T) {
		inv := &Investigation{
			Service: "orderflow",
			RecentErrors: []Error{
				{
					Timestamp: now.Add(-2 * time.Minute).Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "payment transaction failed: provider timeout",
				},
			},
			Dependencies: []DependencyStatus{
				{
					Name:                 "payment-provider",
					CurrentStatus:        "degraded",
					WindowStatus:         "degraded",
					HistoricalErrorCount: 8,
					RecentErrorCount:     8,
				},
			},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) == 0 {
			t.Fatal("expected timeline events for payment failure")
		}

		found := false
		for _, ev := range timeline {
			if ev.Dependency == "payment-provider" && ev.Signal == "dependency_error" {
				found = true
				if ev.Severity != "critical" {
					t.Errorf("expected critical severity for payment failure, got %s", ev.Severity)
				}
			}
		}
		if !found {
			t.Fatalf("expected payment-provider dependency_error event, got %+v", timeline)
		}
	})

	// Scenario 5: Slow
	t.Run("5_slow", func(t *testing.T) {
		inv := &Investigation{
			Service: "orderflow",
		}
		inv.Metrics.P99Latency = prom.Float64Ptr(3.5)

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) == 0 {
			t.Fatal("expected timeline events for high latency")
		}

		found := false
		for _, ev := range timeline {
			if ev.Signal == "latency_spike" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected latency_spike event, got %+v", timeline)
		}
	})

	// Scenario 6: Multiple failures within one window
	t.Run("6_multiple_failures_within_one_window", func(t *testing.T) {
		t1 := now.Add(-8 * time.Minute)
		t2 := now.Add(-5 * time.Minute)
		t3 := now.Add(-3 * time.Minute)

		inv := &Investigation{
			Service: "orderflow",
			RecentDeployments: []Deployment{
				{
					ID:          "deploy-42",
					Version:     "v1.4.0",
					Environment: "production",
					Timestamp:   t1,
					Status:      "success",
				},
			},
			RecentErrors: []Error{
				{
					Timestamp: t2.Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "redis connection refused",
				},
				{
					Timestamp: t3.Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "database query failed",
				},
			},
			Dependencies: []DependencyStatus{
				{Name: "redis", CurrentStatus: "degraded", WindowStatus: "degraded", HistoricalErrorCount: 5},
				{Name: "postgresql", CurrentStatus: "degraded", WindowStatus: "degraded", HistoricalErrorCount: 10},
			},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		if len(timeline) < 3 {
			t.Fatalf("expected at least 3 events, got %d", len(timeline))
		}

		// Verify chronological ordering
		for i := 0; i < len(timeline)-1; i++ {
			if timeline[i].Timestamp.After(timeline[i+1].Timestamp) {
				t.Fatalf("timeline not in chronological order: index %d (%v) after index %d (%v)",
					i, timeline[i].Timestamp, i+1, timeline[i+1].Timestamp)
			}
		}
	})

	// Scenario 7: Recovery
	t.Run("7_recovery", func(t *testing.T) {
		tErr := now.Add(-7 * time.Minute)

		inv := &Investigation{
			Service: "orderflow",
			RecentErrors: []Error{
				{
					Timestamp: tErr.Format(time.RFC3339),
					Level:     "ERROR",
					Message:   "redis connection timeout",
				},
			},
			Dependencies: []DependencyStatus{
				{
					Name:                 "redis",
					CurrentStatus:        "healthy",
					WindowStatus:         "degraded",
					HistoricalErrorCount: 15,
					RecentErrorCount:     0,
				},
			},
		}

		timeline := BuildTimeline(inv, 10*time.Minute)
		foundRecovery := false
		foundError := false

		for _, ev := range timeline {
			if ev.Signal == "dependency_error" && ev.Dependency == "redis" {
				foundError = true
			}
			if ev.Signal == "recovery" && ev.Dependency == "redis" {
				foundRecovery = true
			}
		}

		if !foundError {
			t.Fatal("expected initial error event")
		}
		if !foundRecovery {
			t.Fatal("expected recovery event")
		}

		// Verify error preceded recovery
		if len(timeline) >= 2 && timeline[0].Timestamp.After(timeline[1].Timestamp) {
			t.Fatalf("expected error before recovery, got %v and %v", timeline[0].Timestamp, timeline[1].Timestamp)
		}
	})
}
