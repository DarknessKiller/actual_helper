# Frontend build stage
FROM node:26-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Backend build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
ARG VERSION
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/dist frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -tags embed -trimpath \
    -ldflags="-s -w -X actual_helper/internal/config.Version=${VERSION:-$(git describe --tags --always --dirty)}" \
    -o actual_helper ./cmd/app

# Runtime stage
FROM alpine:3.23
RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories \
 && apk add --no-cache tesseract-ocr tesseract-ocr-data-eng tesseract-ocr-data-msa poppler-utils imagemagick
WORKDIR /app
COPY --from=builder /app/actual_helper actual_helper
COPY --from=builder /app/provider_config.json provider_config.json
ENV PROVIDER_CONFIG_PATH=/app/provider_config.json
EXPOSE $PORT
CMD ["/app/actual_helper"]
