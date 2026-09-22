package investigation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type IncidentEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`               // "prometheus", "loki", "deployments", "investigation"
	Signal      string    `json:"signal"`               // "deployment", "dependency_error", "http_5xx", "latency_spike", "recovery"
	Dependency  string    `json:"dependency,omitempty"` // "redis", "postgresql", "payment-provider"
	Endpoint    string    `json:"endpoint,omitempty"`   // e.g. "/api/orders"
	Severity    string    `json:"severity"`             // "info", "warning", "critical"
	Description string    `json:"description"`
}

// ParseLogTimestamp attempts to parse a Loki timestamp string (Unix nanoseconds or RFC3339).
func ParseLogTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Now()
	}

	// Try unix nanoseconds
	if nano, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(0, nano)
	}

	// Try RFC3339 / RFC3339Nano
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}

	return time.Now()
}

// BuildTimeline transforms evidence from Prometheus, Loki, and Deployments into a chronological timeline.
func BuildTimeline(inv *Investigation, windowDuration time.Duration) []IncidentEvent {
	if inv == nil {
		return nil
	}

	events := make([]IncidentEvent, 0)
	lastErrorPerDep := make(map[string]time.Time)
	firstErrorPerDep := make(map[string]time.Time)

	// 1. Deployments
	for _, dep := range inv.RecentDeployments {
		events = append(events, IncidentEvent{
			Timestamp: dep.Timestamp,
			Source:    "deployments",
			Signal:    "deployment",
			Severity:  "info",
			Description: fmt.Sprintf(
				"Deployment %s deployed to %s (status: %s)",
				dep.Version,
				dep.Environment,
				dep.Status,
			),
		})
	}

	// 2. Recent error logs from Loki
	// Track seen message classes to avoid duplicates within small time windows
	type logKey struct {
		bucket int64 // unix minute
		signal string
		dep    string
	}
	seenLogs := make(map[logKey]bool)

	for _, errLog := range inv.RecentErrors {
		ts := ParseLogTimestamp(errLog.Timestamp)
		msg := strings.ToLower(errLog.Message)

		var dep string
		var signal string
		var severity string
		var endpoint string
		var desc string

		switch {
		case strings.Contains(msg, "payment"):
			dep = "payment-provider"
			signal = "dependency_error"
			severity = "critical"
			endpoint = "/api/payment"
			desc = "Payment provider failures detected"

		case strings.Contains(msg, "redis"):
			dep = "redis"
			signal = "dependency_error"
			severity = "warning"
			endpoint = "/api/products"
			desc = "Redis dependency errors detected"

		case strings.Contains(msg, "database") || strings.Contains(msg, "postgres"):
			dep = "postgresql"
			signal = "dependency_error"
			severity = "critical"
			endpoint = "/api/orders"
			desc = "Database errors detected"

		case strings.Contains(msg, "500") || strings.Contains(msg, "502") || strings.Contains(msg, "503") || strings.Contains(msg, "504"):
			signal = "http_5xx"
			severity = "critical"
			desc = "HTTP 5xx server error logged"

		default:
			signal = "application_error"
			severity = "warning"
			desc = fmt.Sprintf("Error logged: %s", errLog.Message)
		}

		if dep != "" {
			if _, ok := firstErrorPerDep[dep]; !ok || ts.Before(firstErrorPerDep[dep]) {
				firstErrorPerDep[dep] = ts
			}
			if ts.After(lastErrorPerDep[dep]) {
				lastErrorPerDep[dep] = ts
			}
		}

		key := logKey{
			bucket: ts.Unix() / 30, // 30-second deduplication bucket
			signal: signal,
			dep:    dep,
		}
		if !seenLogs[key] {
			seenLogs[key] = true
			events = append(events, IncidentEvent{
				Timestamp:   ts,
				Source:      "loki",
				Signal:      signal,
				Dependency:  dep,
				Endpoint:    endpoint,
				Severity:    severity,
				Description: desc,
			})
		}
	}

	// 3. Recovery detection
	// If a dependency had errors earlier in the window, but current_status is healthy:
	for _, dep := range inv.Dependencies {
		if dep.WindowStatus == "degraded" && dep.CurrentStatus == "healthy" {
			lastErrTime, hasLastErr := lastErrorPerDep[dep.Name]
			recoveryTime := time.Now()
			if hasLastErr {
				// Mark recovery after the last error
				recoveryTime = lastErrTime.Add(30 * time.Second)
			}

			events = append(events, IncidentEvent{
				Timestamp:   recoveryTime,
				Source:      "investigation",
				Signal:      "recovery",
				Dependency:  dep.Name,
				Severity:    "info",
				Description: fmt.Sprintf("Recovery detected: %s dependency has recovered to healthy status", dep.Name),
			})
		}
	}

	// 4. Latency spike event (if P99 latency >= 2s)
	if inv.Metrics.P99Latency != nil && *inv.Metrics.P99Latency >= 2.0 {
		// Use window start + some offset or now
		events = append(events, IncidentEvent{
			Timestamp: time.Now().Add(-windowDuration / 2),
			Source:    "prometheus",
			Signal:    "latency_spike",
			Severity:  "warning",
			Description: fmt.Sprintf(
				"High P99 latency detected: %.2fs",
				*inv.Metrics.P99Latency,
			),
		})
	}

	// 5. Sort events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events
}
