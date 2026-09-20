package handler

import (
	"encoding/json"
	"net/http"

	"orderflow/internal/model"
	"orderflow/internal/service"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(
	service *service.OrderService,
) *OrderHandler {

	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) CreateOrder(
	w http.ResponseWriter,
	r *http.Request,
) {

	var request model.CreateOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	order, err := h.service.CreateOrder(
		r.Context(),
		request,
	)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}
