package prometheus

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Float64Ptr is a helper that returns a pointer to the given float64 value.
func Float64Ptr(v float64) *float64 {
	return &v
}

// ExtractScalar safely parses a scalar value from a Prometheus query response.
// It distinguishes between:
//   - Missing / empty result sets -> (nil, nil)
//   - Unavailable / NaN / Inf values -> (nil, nil)
//   - Valid real zero (0.0) -> (&0.0, nil)
//   - Valid non-zero values -> (&val, nil)
//   - Malformed data / unexpected types -> (nil, error)
func ExtractScalar(resp *QueryResponse) (*float64, error) {
	if resp == nil || len(resp.Data.Result) == 0 {
		return nil, nil
	}

	result := resp.Data.Result[0]
	if len(result.Value) < 2 {
		return nil, fmt.Errorf("unexpected Prometheus value format: missing timestamp or value")
	}

	raw := result.Value[1]
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if strings.EqualFold(s, "nan") || strings.EqualFold(s, "+inf") || strings.EqualFold(s, "-inf") || strings.EqualFold(s, "inf") {
			return nil, nil
		}

		parsed, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Prometheus scalar %q: %w", s, err)
		}

		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return nil, nil
		}

		return &parsed, nil

	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil
		}
		return &v, nil

	case int:
		f := float64(v)
		return &f, nil

	case int64:
		f := float64(v)
		return &f, nil

	case nil:
		return nil, nil

	default:
		return nil, fmt.Errorf("unexpected Prometheus value type: %T", raw)
	}
}
