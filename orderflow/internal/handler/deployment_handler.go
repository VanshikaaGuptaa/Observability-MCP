package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"orderflow/internal/deployment"
)

type DeploymentHandler struct{}

func NewDeploymentHandler() *DeploymentHandler {
	return &DeploymentHandler{}
}

func (h *DeploymentHandler) GetDeployments(
	w http.ResponseWriter,
	r *http.Request,
) {
	deployments := []deployment.Deployment{
		{
			ID:          "deploy-003",
			Service:     "orderflow",
			Version:     "v1.2.0",
			Environment: "production",
			DeployedBy:  "ci-pipeline",
			Timestamp:   time.Now().Add(-30 * time.Minute),
			Status:      "success",
		},
		{
			ID:          "deploy-002",
			Service:     "orderflow",
			Version:     "v1.1.0",
			Environment: "production",
			DeployedBy:  "ci-pipeline",
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Status:      "success",
		},
		{
			ID:          "deploy-001",
			Service:     "orderflow",
			Version:     "v1.0.0",
			Environment: "production",
			DeployedBy:  "ci-pipeline",
			Timestamp:   time.Now().Add(-6 * time.Hour),
			Status:      "success",
		},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(deployments); err != nil {
		http.Error(
			w,
			"failed to encode deployments",
			http.StatusInternalServerError,
		)
	}
}
