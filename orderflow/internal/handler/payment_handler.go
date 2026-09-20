package handler

import (
	"encoding/json"
	"net/http"

	"orderflow/internal/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(
	service *service.PaymentService,
) *PaymentHandler {

	return &PaymentHandler{
		service: service,
	}
}

func (h *PaymentHandler) ProcessPayment(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request service.PaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	response, err := h.service.ProcessPayment(
		r.Context(),
		request,
	)

	if err != nil {
		writeError(
			w,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
