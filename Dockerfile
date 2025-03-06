FROM golang:1.24

WORKDIR /app

# Install system dependencies required for CGO (SQLite)
RUN apt-get update && apt-get install -y gcc musl-dev sqlite3 libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*  # Clean up to reduce image size

# Install dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Install TailwindCSS binary (ARM64 version for Raspberry Pi)
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-arm64 \
    && chmod +x tailwindcss-linux-arm64 \
    && mv tailwindcss-linux-arm64 /usr/local/bin/tailwindcss

# Copy the project files
COPY . .

# Install Templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Generate Templ templates
RUN templ generate

# Build TailwindCSS
RUN tailwindcss -i ./input.css -o ./static/css/output.css

# Build the Go application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o pippaothy ./cmd/main.go

# Set executable permissions
RUN chmod +x /app/pippaothy

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["/app/pippaothy"]

