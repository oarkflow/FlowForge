# ==============================================================================
# Golang HTTP Server — Multi-stage Docker Build
# Stage 1: Build frontend, Stage 2: Build Go binary, Stage 3: Minimal runtime
# ==============================================================================

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --prefer-offline --no-audit
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS go-builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy frontend dist into the project for embedding
COPY --from=frontend-builder /app/frontend/dist ./static/

# Build with embedded static files
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

# Copy binary
COPY --from=go-builder /app/server .

# Copy migration files
COPY migrations/ ./migrations/

# Copy email templates
COPY templates/ ./templates/

# Copy static files (also served by Nginx, but available to Go if needed)
COPY --from=frontend-builder /app/frontend/dist ./static/

# Create non-root user
RUN adduser -D -s /bin/sh appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./server"]
