# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install only necessary build dependencies
RUN apk add --no-cache ca-certificates curl gcompat

# Copy go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install build tools
RUN go install github.com/a-h/templ/cmd/templ@latest

# Install make
RUN apk add --no-cache make

# Copy source code
COPY . .

# Generate templates first
RUN templ generate

# CSS is already built in the repository, skip building in Docker

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pippaothy ./cmd/main.go

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

CMD ["./pippaothy"]
