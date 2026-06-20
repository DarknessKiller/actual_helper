# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o actual_helper ./cmd/app

# Runtime stage
FROM scratch
WORKDIR /app
COPY --from=builder /app/actual_helper actual_helper
COPY --from=builder /app/provider_config.json provider_config.json
ENV PROVIDER_CONFIG_PATH=/app/provider_config.json
ENV APP_ENV=production
ENV PORT=8080
EXPOSE 8080
CMD ["/app/actual_helper"]