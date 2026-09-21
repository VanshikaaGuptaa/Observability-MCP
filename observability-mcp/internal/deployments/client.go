package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Deployment struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	DeployedBy  string    `json:"deployed_by"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) GetDeployments(
	ctx context.Context,
) ([]Deployment, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/deployments",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create deployment request: %w",
			err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"deployment request failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"deployment service returned HTTP %d",
			resp.StatusCode,
		)
	}

	var deployments []Deployment

	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		return nil, fmt.Errorf(
			"decode deployments: %w",
			err,
		)
	}

	return deployments, nil
}
