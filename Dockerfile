# Browser WASM build stage
FROM golang:1.26-alpine AS wasm-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o actual-helper.wasm ./cmd/wasm

# Frontend build stage
FROM node:26-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
COPY --from=wasm-builder /app/actual-helper.wasm public/actual-helper.wasm
RUN npm run build

# Backend build stage
FROM wasm-builder AS builder
ARG VERSION
COPY --from=frontend-builder /app/dist frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -tags embed -trimpath \
    -ldflags="-s -w -X actual_helper/internal/config.Version=${VERSION:-unknown}" \
    -o actual_helper ./cmd/app

# Runtime stage
FROM gcr.io/distroless/static-debian13
WORKDIR /app
COPY --from=builder /app/actual_helper actual_helper
ENV APP_ENV=production
ENV PORT=8080
EXPOSE $PORT
CMD ["/app/actual_helper"]
