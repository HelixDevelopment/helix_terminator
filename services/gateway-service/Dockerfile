# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/gateway-service ./cmd/gateway-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/gateway-service /bin/gateway-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/gateway-service"]
