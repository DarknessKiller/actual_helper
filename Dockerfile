# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
ARG VERSION
ARG PORT=8080
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X actual_helper/internal/config.Version=${VERSION:-$(git describe --tags --always --dirty)}" \
    -o actual_helper ./cmd/app

# Runtime stage
FROM scratch
WORKDIR /app
COPY --from=builder /app/actual_helper actual_helper
COPY --from=builder /app/provider_config.json provider_config.json
ENV PROVIDER_CONFIG_PATH=/app/provider_config.json
EXPOSE $PORT
CMD ["/app/actual_helper"]