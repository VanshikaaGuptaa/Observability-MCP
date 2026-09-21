package deployment

import "time"

type Deployment struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	DeployedBy  string    `json:"deployed_by"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}
