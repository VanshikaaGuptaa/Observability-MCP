package loki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryRangeMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": [
					{
						"stream": {"service_name": "orderflow"},
						"values": [
							["1726998000000000000", "{\"time\":\"2026-09-22T08:00:00Z\",\"level\":\"ERROR\",\"msg\":\"database error\"}"]
						]
					}
				]
			}
		}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	end := time.Now()
	start := end.Add(-10 * time.Minute)

	result, err := client.QueryRange(context.Background(), `{service_name="orderflow"}`, start, end, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data.Result) != 1 {
		t.Fatalf("expected 1 stream, got %d", len(result.Data.Result))
	}
}

func TestQueryRangeEmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": []
			}
		}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	end := time.Now()
	start := end.Add(-10 * time.Minute)

	result, err := client.QueryRange(context.Background(), `{service_name="orderflow"}`, start, end, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil response")
	}
	if len(result.Data.Result) != 0 {
		t.Fatalf("expected 0 streams, got %d", len(result.Data.Result))
	}
}

func TestQueryRangeErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal server error`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	end := time.Now()
	start := end.Add(-10 * time.Minute)

	_, err := client.QueryRange(context.Background(), `{service_name="orderflow"}`, start, end, 10)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}
