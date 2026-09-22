package investigation

import (
	"fmt"
	"strings"
)

type Incident struct {
	Service    string          `json:"service"`
	Severity   string          `json:"severity"`
	Status     string          `json:"status"`
	Impact     Impact          `json:"impact"`
	Symptoms   []string        `json:"symptoms"`
	Evidence   []Evidence      `json:"evidence"`
	Hypotheses []Hypothesis    `json:"hypotheses"`
	Timeline   []IncidentEvent `json:"timeline"`
}

type Impact struct {
	Detected      bool     `json:"detected"`
	AffectedPaths []string `json:"affected_paths"`
	HTTPErrorRate *float64 `json:"http_error_rate"`
	Description   string   `json:"description"`
}

type Evidence struct {
	Source string `json:"source"`
	Signal string `json:"signal"`
	Value  any    `json:"value"`
}

type Hypothesis struct {
	Cause      string   `json:"cause"`
	Confidence string   `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

func BuildIncident(
	investigation *Investigation,
) *Incident {

	if investigation == nil {
		return nil
	}
	if investigation.Diagnosis == nil {
		return nil
	}
	incident := &Incident{
		Service:    investigation.Service,
		Severity:   investigation.Diagnosis.Severity,
		Status:     "healthy",
		Symptoms:   []string{},
		Evidence:   []Evidence{},
		Hypotheses: []Hypothesis{},
		Timeline:   investigation.Timeline,
		Impact: Impact{
			AffectedPaths: []string{},
			HTTPErrorRate: investigation.Metrics.ErrorRate,
			Description:   "No endpoint-level degradation detected.",
		},
	}

	// --------------------------------------------------
	// Determine status
	// --------------------------------------------------

	switch investigation.Diagnosis.Severity {
	case "high":
		incident.Status = "degraded"
	case "medium":
		incident.Status = "degraded"
	default:
		incident.Status = "healthy"
	}

	// --------------------------------------------------
	// Error-rate evidence
	// --------------------------------------------------

	if investigation.Metrics.ErrorRate != nil && *investigation.Metrics.ErrorRate > 0 {

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "prometheus",
				Signal: "error_rate",
				Value:  *investigation.Metrics.ErrorRate,
			},
		)

		if *investigation.Metrics.ErrorRate >= 0.05 {

			incident.Symptoms = append(
				incident.Symptoms,
				fmt.Sprintf(
					"Elevated HTTP error rate: %.2f%%",
					*investigation.Metrics.ErrorRate*100,
				),
			)
		}
	}

	// --------------------------------------------------
	// Latency evidence
	// --------------------------------------------------

	if investigation.Metrics.P95Latency != nil && *investigation.Metrics.P95Latency > 0 {

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "prometheus",
				Signal: "p95_latency",
				Value:  *investigation.Metrics.P95Latency,
			},
		)
	}

	if investigation.Metrics.P99Latency != nil && *investigation.Metrics.P99Latency > 0 {

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "prometheus",
				Signal: "p99_latency",
				Value:  *investigation.Metrics.P99Latency,
			},
		)

		if *investigation.Metrics.P99Latency >= 2 {

			incident.Symptoms = append(
				incident.Symptoms,
				fmt.Sprintf(
					"High P99 latency: %.2fs",
					*investigation.Metrics.P99Latency,
				),
			)
		}
	}

	// --------------------------------------------------
	// Loki evidence
	// --------------------------------------------------

	if len(investigation.RecentErrors) > 0 {

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "loki",
				Signal: "error_logs",
				Value:  len(investigation.RecentErrors),
			},
		)

		incident.Symptoms = append(
			incident.Symptoms,
			fmt.Sprintf(
				"%d error logs detected",
				len(investigation.RecentErrors),
			),
		)
	}

	// --------------------------------------------------
	// Deployment evidence
	// --------------------------------------------------

	for _, deployment := range investigation.RecentDeployments {

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "deployments",
				Signal: "recent_deployment",
				Value: map[string]any{
					"id":          deployment.ID,
					"version":     deployment.Version,
					"environment": deployment.Environment,
					"timestamp":   deployment.Timestamp,
					"status":      deployment.Status,
				},
			},
		)
	}

	// --------------------------------------------------
	// Hypothesis: payment dependency
	// --------------------------------------------------

	for _, err := range investigation.RecentErrors {

		message := strings.ToLower(err.Message)

		if strings.Contains(message, "payment") {

			incident.Hypotheses = append(
				incident.Hypotheses,
				Hypothesis{
					Cause:      "payment dependency failure",
					Confidence: "supported",
					Evidence: []string{
						"Payment-related error logs detected in Loki",
					},
				},
			)

			break
		}
	}

	// --------------------------------------------------
	// Hypothesis: Redis
	// --------------------------------------------------

	for _, err := range investigation.RecentErrors {

		message := strings.ToLower(err.Message)

		if strings.Contains(message, "redis") {

			incident.Hypotheses = append(
				incident.Hypotheses,
				Hypothesis{
					Cause:      "Redis dependency failure",
					Confidence: "supported",
					Evidence: []string{
						"Redis-related error logs detected in Loki",
					},
				},
			)

			break
		}
	}

	// --------------------------------------------------
	// Hypothesis: database
	// --------------------------------------------------

	for _, err := range investigation.RecentErrors {

		message := strings.ToLower(err.Message)

		if strings.Contains(message, "database") || strings.Contains(message, "postgres") {

			incident.Hypotheses = append(
				incident.Hypotheses,
				Hypothesis{
					Cause:      "database failure",
					Confidence: "supported",
					Evidence: []string{
						"Database-related error logs detected in Loki",
					},
				},
			)

			break
		}
	}

	// --------------------------------------------------
	// Dependencies: evidence & symptoms
	// --------------------------------------------------

	for _, dependency := range investigation.Dependencies {

		if dependency.CurrentStatus == "degraded" {
			incident.Evidence = append(
				incident.Evidence,
				Evidence{
					Source: "dependency",
					Signal: dependency.Name,
					Value: map[string]any{
						"current_status": dependency.CurrentStatus,
						"window_status":  dependency.WindowStatus,
						"error_count":    dependency.HistoricalErrorCount,
						"recent_errors":  dependency.RecentErrorCount,
					},
				},
			)

			incident.Symptoms = append(
				incident.Symptoms,
				fmt.Sprintf(
					"Dependency %s is currently degraded",
					dependency.Name,
				),
			)

			incident.Hypotheses = append(
				incident.Hypotheses,
				Hypothesis{
					Cause:      fmt.Sprintf("%s dependency failure", dependency.Name),
					Confidence: "supported",
					Evidence: []string{
						fmt.Sprintf("Active error rate detected on %s dependency", dependency.Name),
					},
				},
			)
		} else if dependency.WindowStatus == "degraded" {
			incident.Evidence = append(
				incident.Evidence,
				Evidence{
					Source: "dependency",
					Signal: dependency.Name,
					Value: map[string]any{
						"current_status": dependency.CurrentStatus,
						"window_status":  dependency.WindowStatus,
						"error_count":    dependency.HistoricalErrorCount,
						"recent_errors":  dependency.RecentErrorCount,
					},
				},
			)

			incident.Symptoms = append(
				incident.Symptoms,
				fmt.Sprintf(
					"Dependency %s experienced degradation earlier in window (recovered)",
					dependency.Name,
				),
			)

			incident.Hypotheses = append(
				incident.Hypotheses,
				Hypothesis{
					Cause:      fmt.Sprintf("historical %s dependency failure", dependency.Name),
					Confidence: "resolved",
					Evidence: []string{
						fmt.Sprintf("Historical errors on %s recorded, but dependency is now healthy", dependency.Name),
					},
				},
			)
		}
	}

	// --------------------------------------------------
	// Impact summary
	// --------------------------------------------------

	if investigation.Metrics.ErrorRate != nil && *investigation.Metrics.ErrorRate > 0 {
		incident.Impact.Detected = true
		incident.Impact.Description = "HTTP request failures detected"
	}

	for _, endpoint := range investigation.Impact.Endpoints {

		if endpoint.Status == "unknown" {
			continue
		}

		incident.Evidence = append(
			incident.Evidence,
			Evidence{
				Source: "prometheus",
				Signal: "endpoint_health",
				Value:  endpoint,
			},
		)
	}

	if investigation.Impact.Detected {
		incident.Impact.AffectedPaths = investigation.Impact.AffectedPaths
		for _, endpoint := range investigation.Impact.AffectedPaths {

			incident.Symptoms = append(
				incident.Symptoms,
				fmt.Sprintf(
					"Endpoint %s is degraded",
					endpoint,
				),
			)
		}
	}

	// --------------------------------------------------
	// Hypothesis: recent deployment
	// --------------------------------------------------

	if len(investigation.RecentDeployments) > 0 {

		hasHighError := investigation.Metrics.ErrorRate != nil && *investigation.Metrics.ErrorRate >= 0.05
		hasHighLatency := investigation.Metrics.P99Latency != nil && *investigation.Metrics.P99Latency >= 2

		if hasHighError || hasHighLatency {

			for _, deployment := range investigation.RecentDeployments {

				incident.Hypotheses = append(
					incident.Hypotheses,
					Hypothesis{
						Cause: fmt.Sprintf(
							"recent deployment %s",
							deployment.Version,
						),
						Confidence: "correlated",
						Evidence: []string{
							"Deployment occurred during the investigation window",
							"Elevated service metrics were detected",
						},
					},
				)
			}
		}
	}

	incident.Hypotheses = deduplicateHypotheses(
		incident.Hypotheses,
	)

	return incident
}

func deduplicateHypotheses(
	hypotheses []Hypothesis,
) []Hypothesis {

	seen := make(map[string]bool)

	result := make([]Hypothesis, 0, len(hypotheses))

	for _, hypothesis := range hypotheses {

		key := hypothesis.Cause

		if seen[key] {
			continue
		}

		seen[key] = true
		result = append(result, hypothesis)
	}

	return result
}
