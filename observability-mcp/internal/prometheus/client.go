package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) Query(
	ctx context.Context,
	query string,
) (*QueryResponse, error) {

	params := url.Values{}
	params.Set("query", query)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v1/query?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"prometheus returned HTTP %d",
			resp.StatusCode,
		)
	}

	var result QueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(
			"decode prometheus response: %w",
			err,
		)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf(
			"prometheus query failed: status=%s",
			result.Status,
		)
	}

	return &result, nil
}
