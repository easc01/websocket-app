# ======= Build Stage =======
FROM golang:1.24-alpine AS builder

# Install git for Go modules
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum for caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build all binaries
RUN mkdir -p /app/bin
RUN go build -o /app/bin/api ./cmd/api
RUN go build -o /app/bin/websocket ./cmd/websocket
RUN go build -o /app/bin/worker ./cmd/worker

# ======= Final Stage =======
FROM alpine:3.18

# Install ca-certificates if your app uses HTTPS
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin /app/bin

# Copy entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Default service if SERVICE env is not set
ENV SERVICE=api