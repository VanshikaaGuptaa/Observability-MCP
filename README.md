# OrderFlow Service

OrderFlow is a core backend service in the Observability Platform responsible for handling product catalog, inventory, order processing, and payment orchestration.

---

## 📁 Project Structure

```text
orderflow/
│
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   │
│   ├── config/
│   │   └── config.go
│   │
│   ├── model/
│   │   ├── order.go
│   │   └── product.go
│   │
│   ├── repository/
│   │   ├── order_repository.go
│   │   └── product_repository.go
│   │
│   ├── cache/
│   │   └── product_cache.go
│   │
│   ├── service/
│   │   ├── order_service.go
│   │   ├── product_service.go
│   │   └── payment_service.go
│   │
│   └── handler/
│       ├── order_handler.go
│       ├── product_handler.go
│       ├── payment_handler.go
│       └── health_handler.go
│
├── migrations/
│   └── 001_init.sql
│
├── .env
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

---

## 🚀 Quick Start

### 1. Prerequisites
- Go 1.22+ installed
- PostgreSQL & Redis (optional for development; defaults to thread-safe in-memory storage)

### 2. Configure Environment
Copy and customize the `.env` file:
```bash
cp .env.example .env
```

### 3. Run the Service
```bash
go run ./cmd/server
```

The service will start by default on port `8080`.

---

## 📡 API Endpoints

### Health
- `GET /health` - Service health status

### Products
- `GET /products` - List all products
- `GET /products/{id}` - Get product by ID
- `POST /products` - Create a new product

### Orders
- `POST /orders` - Place a new order
- `GET /orders/{id}` - Get order by ID
- `GET /orders?customer_id={id}` - List orders for a specific customer

### Payments
- `POST /payments/process` - Process a payment transaction

---

## 🗄️ Database Migrations
Migrations are located in `migrations/`. To apply them to your PostgreSQL database:
```bash
psql -h ${DB_HOST} -U ${DB_USER} -d ${DB_NAME} -f migrations/001_init.sql
```
