# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/user-service ./cmd/user-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/user-service /bin/user-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/user-service"]
