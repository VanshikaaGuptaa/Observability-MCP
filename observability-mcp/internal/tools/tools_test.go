package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	deployments "observability-mcp/internal/deployments"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
)

func setupMockServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Mock Prometheus
	promMux := http.NewServeMux()
	promMux.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		switch q {
		case `up{job="orderflow"}`:
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"orderflow"},"value":[1726998000.0,"1"]}]}}`))
		case `sum(rate(orderflow_http_requests_total[5m]))`:
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1726998000.0,"12.5"]}]}}`))
		case `sum(rate(orderflow_http_requests_total{status=~"5.."}[5m])) / sum(rate(orderflow_http_requests_total[5m]))`:
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1726998000.0,"0.01"]}]}}`))
		default:
			// Default empty vector
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	})
	promServer := httptest.NewServer(promMux)

	// Mock Loki
	lokiMux := http.NewServeMux()
	lokiMux.HandleFunc("/loki/api/v1/query_range", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	})
	lokiServer := httptest.NewServer(lokiMux)

	// Mock Deployments (OrderFlow)
	deployMux := http.NewServeMux()
	deployMux.HandleFunc("/api/deployments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"dep-1","service":"orderflow","version":"v1.0.0","environment":"production","deployed_by":"ci","timestamp":"2026-09-22T08:00:00Z","status":"success"}]`))
	})
	deployServer := httptest.NewServer(deployMux)

	promClient := prom.NewClient(promServer.URL)
	lokiClient := loki.NewClient(lokiServer.URL)
	deployClient := deployments.NewClient(deployServer.URL)

	s := NewServer(promClient, lokiClient, deployClient)

	cleanup := func() {
		promServer.Close()
		lokiServer.Close()
		deployServer.Close()
	}

	return s, cleanup
}

func TestToolsValidation(t *testing.T) {
	s, cleanup := setupMockServer(t)
	defer cleanup()

	ctx := context.Background()

	// 1. GetServiceHealth
	t.Run("GetServiceHealth_validation", func(t *testing.T) {
		_, _, err := s.GetServiceHealth(ctx, nil, HealthInput{Service: ""})
		if err == nil {
			t.Error("expected error for empty service")
		}
		_, _, err = s.GetServiceHealth(ctx, nil, HealthInput{Service: "invalid-svc"})
		if err == nil {
			t.Error("expected error for unsupported service")
		}
		_, out, err := s.GetServiceHealth(ctx, nil, HealthInput{Service: "orderflow"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Status != "healthy" {
			t.Errorf("expected healthy status, got %s", out.Status)
		}
	})

	// 2. GetRequestRate
	t.Run("GetRequestRate_validation", func(t *testing.T) {
		_, _, err := s.GetRequestRate(ctx, nil, RequestRateInput{Service: "bad"})
		if err == nil {
			t.Error("expected error for bad service")
		}
		_, _, err = s.GetRequestRate(ctx, nil, RequestRateInput{Service: "orderflow", Duration: "bad-dur"})
		if err == nil {
			t.Error("expected error for bad duration")
		}
		_, out, err := s.GetRequestRate(ctx, nil, RequestRateInput{Service: "orderflow", Duration: "5m"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.RequestsPerSecond == nil || *out.RequestsPerSecond != 12.5 {
			t.Errorf("expected 12.5 rps, got %v", out.RequestsPerSecond)
		}
	})

	// 3. GetErrorRate
	t.Run("GetErrorRate_validation", func(t *testing.T) {
		_, _, err := s.GetErrorRate(ctx, nil, ErrorRateInput{Service: "bad"})
		if err == nil {
			t.Error("expected error for bad service")
		}
		_, out, err := s.GetErrorRate(ctx, nil, ErrorRateInput{Service: "orderflow", Duration: "5m"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.ErrorRate == nil || *out.ErrorRate != 0.01 {
			t.Errorf("expected 0.01 error rate, got %v", out.ErrorRate)
		}
	})

	// 4. GetLatency
	t.Run("GetLatency_validation", func(t *testing.T) {
		_, _, err := s.GetLatency(ctx, nil, LatencyInput{Service: "orderflow", Endpoint: "/bad/endpoint"})
		if err == nil {
			t.Error("expected error for bad endpoint")
		}
		_, _, err = s.GetLatency(ctx, nil, LatencyInput{Service: "orderflow", Endpoint: "/api/orders", Duration: "bad"})
		if err == nil {
			t.Error("expected error for bad duration")
		}
		_, out, err := s.GetLatency(ctx, nil, LatencyInput{Service: "orderflow", Endpoint: "/api/orders", Duration: "5m"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Since mock returned empty vector for latency query, values should be nil (null in JSON), NOT zero!
		if out.P50Seconds != nil {
			t.Errorf("expected nil P50Seconds for empty prometheus result, got %v", *out.P50Seconds)
		}
		if out.P95Seconds != nil {
			t.Errorf("expected nil P95Seconds for empty prometheus result, got %v", *out.P95Seconds)
		}
		if out.P99Seconds != nil {
			t.Errorf("expected nil P99Seconds for empty prometheus result, got %v", *out.P99Seconds)
		}
	})

	// 5. SearchLogs
	t.Run("SearchLogs_validation", func(t *testing.T) {
		_, _, err := s.SearchLogs(ctx, nil, SearchLogsInput{Service: "orderflow", Query: ""})
		if err == nil {
			t.Error("expected error for empty query")
		}
		_, out, err := s.SearchLogs(ctx, nil, SearchLogsInput{Service: "orderflow", Query: "redis", Duration: "10m"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Logs) != 0 {
			t.Errorf("expected 0 logs, got %d", len(out.Logs))
		}
	})

	// 6. GetRecentErrors
	t.Run("GetRecentErrors_validation", func(t *testing.T) {
		_, _, err := s.GetRecentErrors(ctx, nil, RecentErrorsInput{Service: "bad-service"})
		if err == nil {
			t.Error("expected error for bad service")
		}
		_, out, err := s.GetRecentErrors(ctx, nil, RecentErrorsInput{Service: "orderflow", Duration: "10m"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Errors) != 0 {
			t.Errorf("expected 0 errors, got %d", len(out.Errors))
		}
	})

	// 7. GetRecentDeployments
	t.Run("GetRecentDeployments_validation", func(t *testing.T) {
		_, _, err := s.GetRecentDeployments(ctx, nil, RecentDeploymentsInput{Service: "unknown"})
		if err == nil {
			t.Error("expected error for unknown service")
		}
		_, out, err := s.GetRecentDeployments(ctx, nil, RecentDeploymentsInput{Service: "orderflow", Duration: "24h"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Deployments) != 1 {
			t.Errorf("expected 1 deployment, got %d", len(out.Deployments))
		}
	})

	// 8. InvestigateService & InvestigateIncident
	t.Run("Investigate_validation", func(t *testing.T) {
		_, _, err := s.InvestigateService(ctx, nil, InvestigateServiceInput{Service: "bad"})
		if err == nil {
			t.Error("expected error for bad service in InvestigateService")
		}
		_, _, err = s.InvestigateIncident(ctx, nil, InvestigateIncidentInput{Service: "bad"})
		if err == nil {
			t.Error("expected error for bad service in InvestigateIncident")
		}
	})
}

func TestToolRegistrationAndList(t *testing.T) {
	s, cleanup := setupMockServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "observability-mcp", Version: "1.0.0"},
		nil,
	)
	RegisterTools(server, s)

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-client", Version: "1.0.0"},
		nil,
	)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		_, _ = server.Connect(ctx, serverTransport, nil)
	}()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer clientSession.Close()

	listToolsRes, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	expectedTools := map[string]bool{
		"get_service_health":     false,
		"get_request_rate":       false,
		"get_error_rate":         false,
		"get_latency":            false,
		"search_logs":            false,
		"get_recent_errors":      false,
		"get_recent_deployments": false,
		"investigate_service":    false,
		"investigate_incident":   false,
	}

	for _, tool := range listToolsRes.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("tool %s was not discovered in ListTools", name)
		}
	}
}
