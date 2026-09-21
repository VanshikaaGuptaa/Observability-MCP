package investigation

import (
	"fmt"
	"strings"
	"time"
)

type Diagnosis struct {
	Summary            string   `json:"summary"`
	Severity           string   `json:"severity"`
	Signals            []string `json:"signals"`
	PossibleCauses     []string `json:"possible_causes"`
	RelatedDeployments []string `json:"related_deployments"`
}

func Diagnose(
	investigation *Investigation,
	duration time.Duration,
) *Diagnosis {

	result := &Diagnosis{
		Severity: "normal",
	}

	if investigation == nil {
		result.Summary = "No investigation data available."
		return result
	}

	var signals []string
	var causes []string
	var deployments []string

	// --------------------------------------------------
	// 1. Error rate
	// --------------------------------------------------

	if investigation.Metrics.ErrorRate > 0 {
		signals = append(
			signals,
			fmt.Sprintf(
				"HTTP error rate is %.2f%%",
				investigation.Metrics.ErrorRate*100,
			),
		)
	}

	if investigation.Metrics.ErrorRate >= 0.05 {
		result.Severity = "high"

		causes = append(
			causes,
			"Elevated HTTP 5xx error rate",
		)
	}

	// --------------------------------------------------
	// 2. Latency
	// --------------------------------------------------

	if investigation.Metrics.P95Latency >= 1 {
		signals = append(
			signals,
			fmt.Sprintf(
				"P95 latency is %.2f seconds",
				investigation.Metrics.P95Latency,
			),
		)
	}

	if investigation.Metrics.P99Latency >= 2 {
		signals = append(
			signals,
			fmt.Sprintf(
				"P99 latency is %.2f seconds",
				investigation.Metrics.P99Latency,
			),
		)

		if result.Severity == "normal" {
			result.Severity = "medium"
		}

		causes = append(
			causes,
			"High request latency",
		)
	}

	// --------------------------------------------------
	// 3. Recent errors
	// --------------------------------------------------

	if len(investigation.RecentErrors) > 0 {

		signals = append(
			signals,
			fmt.Sprintf(
				"%d error logs found in the investigation window",
				len(investigation.RecentErrors),
			),
		)

		if result.Severity == "normal" {
			result.Severity = "medium"
		}

		for _, err := range investigation.RecentErrors {

			message := strings.ToLower(err.Message)

			switch {
			case strings.Contains(message, "payment"):
				causes = append(
					causes,
					"Payment dependency failures detected",
				)

			case strings.Contains(message, "redis"):
				causes = append(
					causes,
					"Redis dependency errors detected",
				)

			case strings.Contains(message, "database"):
				causes = append(
					causes,
					"Database errors detected",
				)
			}
		}
	}

	// --------------------------------------------------
	// 4. Recent deployments
	// --------------------------------------------------

	cutoff := time.Now().Add(-duration)

	for _, deployment := range investigation.RecentDeployments {

		if deployment.Timestamp.Before(cutoff) {
			continue
		}

		deploymentInfo := fmt.Sprintf(
			"%s (%s)",
			deployment.Version,
			deployment.Timestamp.Format(time.RFC3339),
		)

		deployments = append(
			deployments,
			deploymentInfo,
		)

		signals = append(
			signals,
			fmt.Sprintf(
				"Deployment %s occurred during the investigation window",
				deployment.Version,
			),
		)
	}

	// --------------------------------------------------
	// 5. Correlation logic
	// --------------------------------------------------

	if len(deployments) > 0 &&
		investigation.Metrics.ErrorRate >= 0.05 {

		causes = append(
			causes,
			"Recent deployment temporally correlates with elevated error rate",
		)
	}

	if len(deployments) > 0 &&
		investigation.Metrics.P99Latency >= 2 {

		causes = append(
			causes,
			"Recent deployment temporally correlates with elevated latency",
		)
	}

	// --------------------------------------------------
	// 6. Deduplicate causes
	// --------------------------------------------------

	causes = uniqueStrings(causes)

	// --------------------------------------------------
	// 7. Summary
	// --------------------------------------------------

	switch result.Severity {

	case "high":
		result.Summary =
			"Service is experiencing elevated errors and requires investigation."

	case "medium":
		result.Summary =
			"Service shows signs of degradation."

	default:
		result.Summary =
			"No significant service degradation detected."
	}

	result.Signals = signals
	result.PossibleCauses = causes
	result.RelatedDeployments = deployments

	return result
}

func uniqueStrings(values []string) []string {

	seen := make(map[string]bool)
	result := make([]string, 0, len(values))

	for _, value := range values {

		if seen[value] {
			continue
		}

		seen[value] = true
		result = append(result, value)
	}

	return result
}
