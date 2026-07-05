# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/snippet-service ./cmd/snippet-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/snippet-service /bin/snippet-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/snippet-service"]
