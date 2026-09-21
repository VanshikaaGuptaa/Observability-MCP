package prometheus

import (
	"context"
	"testing"
)

func TestQuery(t *testing.T) {

	client := NewClient("http://localhost:9090")

	result, err := client.Query(
		context.Background(),
		`up{job="orderflow"}`,
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(result.Data.Result) == 0 {
		t.Fatal("expected orderflow result")
	}

	t.Logf("Prometheus result: %+v", result.Data.Result)
}
