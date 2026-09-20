package model

import "time"

type OrderItemRequest struct {
	ProductID int64 `json:"productId"`
	Quantity  int   `json:"quantity"`
}

type CreateOrderRequest struct {
	CustomerName string             `json:"customerName"`
	Items        []OrderItemRequest `json:"items"`
}

type OrderItem struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"productId"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Order struct {
	ID           int64       `json:"id"`
	CustomerName string      `json:"customerName"`
	TotalAmount  float64     `json:"totalAmount"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"createdAt"`
	Items        []OrderItem `json:"items"`
}
