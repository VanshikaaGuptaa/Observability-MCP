package validation

import (
	"fmt"
	"regexp"
	"time"
)

var allowedEndpoints = map[string]bool{
	"/api/orders":   true,
	"/api/products": true,
	"/api/payment":  true,
	"/api/health":   true,
}
var durationPattern = regexp.MustCompile(
	`^[0-9]+(s|m|h)$`,
)

var allowedDurations = map[string]bool{
	"1m":  true,
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
	if service == "" {
		return fmt.Errorf("service is required")
	}

	if !allowedServices[service] {
		return fmt.Errorf(
			"unsupported service: %s",
			service,
		)
	}

	return nil
}

func ValidateDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, fmt.Errorf("duration is required")
	}

	if !durationPattern.MatchString(value) {
		return 0, fmt.Errorf(
			"invalid duration format: %s",
			value,
		)
	}

	if !allowedDurations[value] {
		return 0, fmt.Errorf(
			"unsupported duration: %s",
			value,
		)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid duration: %w",
			err,
		)
	}

	return duration, nil
}
func ValidateEndpoint(endpoint string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	if !allowedEndpoints[endpoint] {
		return fmt.Errorf(
			"unsupported endpoint: %s",
			endpoint,
		)
	}

	return nil
}
