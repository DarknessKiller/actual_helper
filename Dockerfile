# Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Build Go static server
FROM golang:1.26-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/staticserver/ cmd/staticserver/
RUN CGO_ENABLED=0 go build -o /staticserver ./cmd/staticserver

# Runtime
FROM alpine:3.21
COPY --from=frontend /app/dist /app/dist
COPY --from=server /staticserver /app/staticserver
EXPOSE 8080
CMD ["/app/staticserver", "-dir", "/app/dist", "-port", "8080"]
