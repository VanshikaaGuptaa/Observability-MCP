package handler

import (
	"encoding/json"
	"net/http"

	"orderflow/internal/failure"
)

type FailureHandler struct {
	simulator *failure.Simulator
}

func NewFailureHandler(simulator *failure.Simulator) *FailureHandler {
	return &FailureHandler{
		simulator: simulator,
	}
}

type failureRequest struct {
	Scenario string `json:"scenario"`
}

type failureResponse struct {
	Scenario string `json:"scenario"`
}

func (h *FailureHandler) SetScenario(w http.ResponseWriter, r *http.Request) {
	var request failureRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	scenario := failure.Scenario(request.Scenario)

	switch scenario {
	case failure.Normal,
		failure.Slow,
		failure.PaymentFailure,
		failure.DBFailure,
		failure.RedisFailure,
		failure.ErrorSpike:

		h.simulator.SetScenario(scenario)

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(failureResponse{
			Scenario: string(scenario),
		})

	default:
		http.Error(w, "unknown failure scenario", http.StatusBadRequest)
	}
}

func (h *FailureHandler) GetScenario(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(failureResponse{
		Scenario: string(h.simulator.GetScenario()),
	})
}