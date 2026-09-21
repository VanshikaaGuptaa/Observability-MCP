package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string      `json:"resultType"`
		Result     []LogStream `json:"result"`
	} `json:"data"`
}

type LogStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) QueryRange(
	ctx context.Context,
	query string,
	start time.Time,
	end time.Time,
	limit int,
) (*QueryResponse, error) {

	params := url.Values{}

	params.Set("query", query)
	params.Set(
		"start",
		strconv.FormatInt(start.UnixNano(), 10),
	)
	params.Set(
		"end",
		strconv.FormatInt(end.UnixNano(), 10),
	)
	params.Set(
		"limit",
		strconv.Itoa(limit),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/loki/api/v1/query_range?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create Loki request: %w",
			err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"Loki request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Loki returned HTTP %d",
			resp.StatusCode,
		)
	}

	var result QueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(
			"decode Loki response: %w",
			err,
		)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf(
			"Loki query failed: status=%s",
			result.Status,
		)
	}

	return &result, nil
}
