# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and git.
RUN apk add --no-cache ca-certificates git

# Copy dependency files first.
# This allows Docker to cache dependency installation.
COPY go.mod go.sum ./

RUN go mod download

# Copy application source.
COPY . .

# Build the Go binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /orderflow ./cmd/server


# ---------- Runtime stage ----------
FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    wget

COPY --from=builder /orderflow /app/orderflow

EXPOSE 8080

ENTRYPOINT ["/app/orderflow"]