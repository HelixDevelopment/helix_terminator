# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/helixtrack-bridge-service ./cmd/helixtrack-bridge-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/helixtrack-bridge-service /bin/helixtrack-bridge-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/helixtrack-bridge-service"]
