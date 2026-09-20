package handler

import (
	"net/http"
	"strconv"
	"strings"

	"orderflow/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

func NewProductHandler(service *service.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

func (h *ProductHandler) GetProducts(
	w http.ResponseWriter,
	r *http.Request,
) {

	products, err := h.service.GetProducts(r.Context())

	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"failed to fetch products",
		)
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) GetProduct(
	w http.ResponseWriter,
	r *http.Request,
) {

	idString := strings.TrimPrefix(
		r.URL.Path,
		"/api/products/",
	)

	id, err := strconv.ParseInt(idString, 10, 64)

	if err != nil || id <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	product, err := h.service.GetProduct(
		r.Context(),
		id,
	)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(
				w,
				http.StatusNotFound,
				"product not found",
			)
			return
		}

		writeError(
			w,
			http.StatusInternalServerError,
			"failed to fetch product",
		)
		return
	}

	writeJSON(w, http.StatusOK, product)
}
