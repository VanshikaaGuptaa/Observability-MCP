package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"log/slog"
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
		if strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "greater than zero") ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "insufficient stock") {
			writeError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		slog.Error(
    "failed to create order",
    "error", err,
    "method", r.Method,
    "path", r.URL.Path,
)

writeError(
    w,
    http.StatusInternalServerError,
    "failed to create order",
)
return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}
