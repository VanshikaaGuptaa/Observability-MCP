package prometheus

import (
	"testing"
)

func TestExtractScalar(t *testing.T) {
	tests := []struct {
		name      string
		resp      *QueryResponse
		wantNil   bool
		wantVal   float64
		wantError bool
	}{
		{
			name:    "nil response",
			resp:    nil,
			wantNil: true,
		},
		{
			name: "empty results slice",
			resp: &QueryResponse{
				Status: "success",
			},
			wantNil: true,
		},
		{
			name: "valid zero string",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "0"},
						},
					},
				},
			},
			wantNil: false,
			wantVal: 0.0,
		},
		{
			name: "valid zero float",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "0.0"},
						},
					},
				},
			},
			wantNil: false,
			wantVal: 0.0,
		},
		{
			name: "valid non-zero",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "0.25"},
						},
					},
				},
			},
			wantNil: false,
			wantVal: 0.25,
		},
		{
			name: "nan string returns nil (unavailable)",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "NaN"},
						},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "+Inf string returns nil (unavailable)",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "+Inf"},
						},
					},
				},
			},
			wantNil: true,
		},
		{
			name: "malformed value array",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0},
						},
					},
				},
			},
			wantError: true,
		},
		{
			name: "unparseable string",
			resp: &QueryResponse{
				Status: "success",
				Data: struct {
					ResultType string `json:"resultType"`
					Result     []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					} `json:"result"`
				}{
					ResultType: "vector",
					Result: []struct {
						Metric map[string]string `json:"metric"`
						Value  []any             `json:"value"`
					}{
						{
							Metric: map[string]string{},
							Value:  []any{1726998000.0, "not-a-number"},
						},
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractScalar(tt.resp)
			if (err != nil) != tt.wantError {
				t.Fatalf("ExtractScalar() error = %v, wantError = %v", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("ExtractScalar() = %v, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ExtractScalar() = nil, want %v", tt.wantVal)
			}
			if *got != tt.wantVal {
				t.Fatalf("ExtractScalar() = %v, want %v", *got, tt.wantVal)
			}
		})
	}
}
