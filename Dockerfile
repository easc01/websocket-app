# ======= Build Stage =======
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build main.go
RUN mkdir -p /app/bin
RUN go build -o /app/bin/main ./cmd/main.go

# ======= Final Stage =======
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/main /app/main

ENTRYPOINT ["/app/main"]
