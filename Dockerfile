FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN GOOS=linux GOARCH=arm64 go build -o pippaothy ./cmd/main.go

FROM gcr.io/distroless/base:nonroot

WORKDIR /app

COPY --from=builder /app/pippaothy /app/pippaothy

COPY static /app/static

EXPOSE 8080

CMD ["/app/pippaothy"]

