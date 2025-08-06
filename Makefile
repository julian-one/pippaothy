APP_NAME := pippaothy

# Tool paths - can be overridden with environment variables
TAILWIND_BIN ?= $(shell which tailwindcss || echo "$(HOME)/bin/tailwindcss")
TEMPL_BIN ?= $(shell which templ || echo "$(shell go env GOPATH)/bin/templ")
AIR_BIN ?= $(shell which air || echo "$(shell go env GOPATH)/bin/air")

ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev install-tools

# Tool installation
install-tools:
	@echo "Installing development tools..."
	@if ! command -v tailwindcss >/dev/null 2>&1; then \
		echo "Installing Tailwind CSS..."; \
		curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-$$(uname -s | tr '[:upper:]' '[:lower:]')-$$(uname -m | sed 's/x86_64/x64/'); \
		chmod +x tailwindcss-*; \
		mkdir -p $(HOME)/bin; \
		mv tailwindcss-* $(HOME)/bin/tailwindcss; \
	fi
	@if ! command -v templ >/dev/null 2>&1; then \
		echo "Installing templ..."; \
		go install github.com/a-h/templ/cmd/templ@latest; \
	fi
	@if ! command -v air >/dev/null 2>&1; then \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi

# Build targets
tailwind-build:
	@if [ ! -f "$(TAILWIND_BIN)" ]; then echo "Tailwind CSS not found. Run 'make install-tools' first."; exit 1; fi
	$(TAILWIND_BIN) --input ./static/css/input.css --output ./static/css/output.css

templ-generate:
	@if [ ! -f "$(TEMPL_BIN)" ]; then echo "Templ not found. Run 'make install-tools' first."; exit 1; fi
	$(TEMPL_BIN) generate

build: tailwind-build templ-generate
	go build -o ./bin/$(APP_NAME) ./cmd/main.go

build-scraper:
	go build -o ./bin/hbh-scraper ./cmd/hbh_scraper.go

# Watchers
tailwind-watch:
	@if [ ! -f "$(TAILWIND_BIN)" ]; then echo "Tailwind CSS not found. Run 'make install-tools' first."; exit 1; fi
	$(TAILWIND_BIN) --input ./static/css/input.css --output ./static/css/output.css --watch

templ-watch:
	@if [ ! -f "$(TEMPL_BIN)" ]; then echo "Templ not found. Run 'make install-tools' first."; exit 1; fi
	$(TEMPL_BIN) generate --watch

dev: build
	$(MAKE) tailwind-watch &
	$(MAKE) templ-watch &
	@if [ ! -f "$(AIR_BIN)" ]; then echo "Air not found. Run 'make install-tools' first."; exit 1; fi
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) $(AIR_BIN)

