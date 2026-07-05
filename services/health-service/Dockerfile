# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/health-service ./cmd/health-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/health-service /bin/health-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/health-service"]
