package validation

import (
	"strings"
	"testing"
	"time"
)

func TestValidateService(t *testing.T) {
	tests := []struct {
		service string
		wantErr bool
	}{
		{"orderflow", false},
		{"", true},
		{"   ", true},
		{"unknown-svc", true},
		{"orderflow; DROP TABLE orders;", true},
		{"../etc/passwd", true},
	}

	for _, tt := range tests {
		err := ValidateService(tt.service)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateService(%q) error = %v, wantErr = %v", tt.service, err, tt.wantErr)
		}
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		duration string
		wantDur  time.Duration
		wantErr  bool
	}{
		{"1m", 1 * time.Minute, false},
		{"5m", 5 * time.Minute, false},
		{"10m", 10 * time.Minute, false},
		{"15m", 15 * time.Minute, false},
		{"30m", 30 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"6h", 6 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"", 0, true},
		{"5x", 0, true},
		{"100m", 0, true},
		{"-5m", 0, true},
		{"5m; rm -rf", 0, true},
	}

	for _, tt := range tests {
		d, err := ValidateDuration(tt.duration)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateDuration(%q) error = %v, wantErr = %v", tt.duration, err, tt.wantErr)
		}
		if !tt.wantErr && d != tt.wantDur {
			t.Errorf("ValidateDuration(%q) = %v, want %v", tt.duration, d, tt.wantDur)
		}
	}
}

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		wantErr  bool
	}{
		{"/api/orders", false},
		{"/api/products", false},
		{"/api/products/1", false},
		{"/api/payment", false},
		{"/api/health", false},
		{"/api/deployments", false},
		{"", true},
		{"/admin", true},
		{"/api/unknown", true},
		{"/api/orders/../../etc/passwd", true},
		{`" OR 1=1 --`, true},
	}

	for _, tt := range tests {
		err := ValidateEndpoint(tt.endpoint)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateEndpoint(%q) error = %v, wantErr = %v", tt.endpoint, err, tt.wantErr)
		}
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		query   string
		wantErr bool
	}{
		{"error", false},
		{"redis connection failed", false},
		{"payment provider timeout", false},
		{"", true},
		{"   ", true},
		{strings.Repeat("a", 501), true},
		{"invalid\x00byte", true},
		{"line1\nline2", true},
	}

	for _, tt := range tests {
		err := ValidateQuery(tt.query)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateQuery(%q) error = %v, wantErr = %v", tt.query, err, tt.wantErr)
		}
	}
}
