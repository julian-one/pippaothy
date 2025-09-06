# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install all build dependencies in one layer
RUN apk add --no-cache ca-certificates curl gcompat make && \
    go install github.com/a-h/templ/cmd/templ@latest

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary source files (not entire context)
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY schema/ ./schema/
COPY static/ ./static/

# Generate templates and build in one RUN to reduce layers
RUN templ generate && \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o pippaothy ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy built application and static assets
COPY --from=builder /app/pippaothy .
COPY --from=builder /app/static ./static
COPY --from=builder /app/schema ./schema

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./pippaothy", "serve"]
