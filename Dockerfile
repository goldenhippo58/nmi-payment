# Build stage
FROM golang:1.20-alpine AS builder
RUN apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app/
COPY --from=builder /app/payment-service .
COPY .env .

# Create directory for logs
RUN mkdir -p /app/logs && \
    touch /app/logs/transactions.log && \
    touch /app/logs/transactions.csv

EXPOSE 8080

CMD ["./payment-service"]