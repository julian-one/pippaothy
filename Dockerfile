FROM golang:1.24

WORKDIR /app

RUN apt-get update && apt-get install -y gcc musl-dev sqlite3 libsqlite3-dev

COPY go.mod go.sum ./
RUN go mod tidy

RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-arm64 \
    && chmod +x tailwindcss-linux-arm64 \
    && mv tailwindcss-linux-arm64 /usr/local/bin/tailwindcss

COPY . .

RUN go install github.com/a-h/templ/cmd/templ@latest

RUN templ generate

RUN tailwindcss -i ./static/css/input.css -o ./static/css/output.css

RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o pippaothy ./cmd/main.go

RUN chmod +x /app/pippaothy

EXPOSE 8080

CMD ["/app/pippaothy"]
