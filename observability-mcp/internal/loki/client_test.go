package loki

import (
	"context"
	"testing"
	"time"
)

func TestQueryRange(t *testing.T) {
	client := NewClient("http://localhost:3100")

	end := time.Now()
	start := end.Add(-10 * time.Minute)

	result, err := client.QueryRange(
		context.Background(),
		`{service_name="orderflow"}`,
		start,
		end,
		10,
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(result.Data.Result) == 0 {
		t.Fatal("expected Loki results")
	}

	t.Logf(
		"Loki streams: %+v",
		result.Data.Result,
	)
}
