APP_NAME := pippaothy

ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev

# Build 
tailwind-build:
	~/bin/tailwindcss --input ./static/css/input.css --output ./static/css/output.css

templ-generate:
	templ generate

build: tailwind-build templ-generate
	go build -o ./bin/$(APP_NAME) ./cmd/main.go

# Watchers
tailwind-watch:
	~/bin/tailwindcss --input ./static/css/input.css --output ./static/css/output.css --watch

templ-watch:
	templ generate --watch

dev: build
	$(MAKE) tailwind-watch &
	$(MAKE) templ-watch &
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) air

