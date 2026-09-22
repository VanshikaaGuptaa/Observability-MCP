package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"observability-mcp/internal/errors"
)

var allowedEndpoints = map[string]bool{
	"/api/orders":      true,
	"/api/products":    true,
	"/api/products/1":  true,
	"/api/payment":     true,
	"/api/health":      true,
	"/api/deployments": true,
}

var durationPattern = regexp.MustCompile(
	`^[0-9]+(s|m|h)$`,
)

var allowedDurations = map[string]bool{
	"1m":  true,
	"2m":  true,
	"5m":  true,
	"10m": true,
	"15m": true,
	"30m": true,
	"1h":  true,
	"6h":  true,
	"24h": true,
}

var allowedServices = map[string]bool{
	"orderflow": true,
}

func ValidateService(service string) error {
	if strings.TrimSpace(service) == "" {
		return errors.NewValidationError("service is required")
	}

	if !allowedServices[service] {
		return errors.NewValidationError(
			fmt.Sprintf("unsupported service: %s", service),
		)
	}

	return nil
}

func ValidateDuration(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, errors.NewValidationError("duration is required")
	}

	if !durationPattern.MatchString(value) {
		return 0, errors.NewValidationError(
			fmt.Sprintf("invalid duration format: %s", value),
		)
	}

	if !allowedDurations[value] {
		return 0, errors.NewValidationError(
			fmt.Sprintf("unsupported duration: %s", value),
		)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.NewValidationError(
			fmt.Sprintf("invalid duration: %v", err),
		)
	}

	return duration, nil
}

func ValidateEndpoint(endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errors.NewValidationError("endpoint is required")
	}

	if !allowedEndpoints[endpoint] {
		return errors.NewValidationError(
			fmt.Sprintf("unsupported endpoint: %s", endpoint),
		)
	}

	return nil
}

func ValidateQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return errors.NewValidationError("query is required")
	}

	if len(trimmed) > 500 {
		return errors.NewValidationError("query exceeds maximum length of 500 characters")
	}

	for _, r := range query {
		if r == 0 || (unicode.IsControl(r) && r != '\t') {
			return errors.NewValidationError("query contains invalid control characters")
		}
	}

	return nil
}
