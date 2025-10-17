APP_NAME := pippaothy

# Tool paths - can be overridden with environment variables
TAILWIND_BIN ?= $(shell which tailwindcss || echo "$(HOME)/bin/tailwindcss")
TEMPL_BIN ?= $(shell which templ || echo "$(shell go env GOPATH)/bin/templ")
AIR_BIN ?= $(shell which air || echo "$(shell go env GOPATH)/bin/air")

ifneq (,$(wildcard .env))
    include .env
    export $(shell sed 's/=.*//' .env)
endif

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev dev-tunnel dev-local stop-tunnel k3s-tunnel stop-k3s-tunnel tunnels stop-tunnels install-tools

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
	@echo "Setting up SSH tunnel to database..."
	@# Check if tunnel already exists on port 5433
	@if ! lsof -ti:5433 >/dev/null 2>&1; then \
		echo "Creating SSH tunnel (local port 5433 -> remote PostgreSQL 5432)..."; \
		ssh -f -N -L 5433:localhost:5432 online-1 || \
		{ echo "Failed to create SSH tunnel. Please check your SSH connection to online-1."; \
		  echo "Make sure your SSH key is properly configured for online-1"; \
		  exit 1; }; \
		echo "SSH tunnel created successfully on port 5433"; \
		sleep 1; \
	else \
		echo "SSH tunnel already exists on port 5433 or port is in use"; \
	fi
	$(MAKE) tailwind-watch &
	$(MAKE) templ-watch &
	@if [ ! -f "$(AIR_BIN)" ]; then echo "Air not found. Run 'make install-tools' first."; exit 1; fi
	DB_HOST=localhost DB_PORT=5433 DB_USER=k3s_user DB_PASSWORD=pippa DB_NAME=k3s_db $(AIR_BIN)

# Alias for backward compatibility
dev-tunnel: dev

# Development without SSH tunnel (uses environment variables or .env file)
dev-local: build
	$(MAKE) tailwind-watch &
	$(MAKE) templ-watch &
	@if [ ! -f "$(AIR_BIN)" ]; then echo "Air not found. Run 'make install-tools' first."; exit 1; fi
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) $(AIR_BIN)

# Stop the SSH tunnel
stop-tunnel:
	@echo "Stopping SSH tunnel..."
	@PID=$$(lsof -ti:5433 | grep -v $$$$ | head -1); \
	if [ ! -z "$$PID" ]; then \
		kill $$PID 2>/dev/null && echo "SSH tunnel stopped (PID: $$PID)" || echo "Failed to stop tunnel"; \
	else \
		echo "No SSH tunnel found on port 5433"; \
	fi

# K3s/kubectl tunnel management
k3s-tunnel:
	@echo "Setting up SSH tunnel for kubectl/K3s..."
	@# Check if tunnel already exists on port 6443
	@if ! lsof -ti:6443 >/dev/null 2>&1; then \
		echo "Creating K3s tunnel (local port 6443 -> online-0:6443)..."; \
		ssh -f -N -L 6443:localhost:6443 online-0 || \
		{ echo "Failed to create K3s tunnel. Please check your SSH connection to online-0."; \
		  exit 1; }; \
		echo "K3s tunnel created successfully on port 6443"; \
	else \
		echo "K3s tunnel already exists on port 6443 or port is in use"; \
	fi
	@echo "Updating kubectl config to use localhost..."
	@kubectl config set-cluster default --server=https://localhost:6443 >/dev/null
	@echo "Testing kubectl connection..."
	@kubectl get nodes >/dev/null 2>&1 && echo "✓ kubectl is working through SSH tunnel" || echo "✗ kubectl connection failed"

stop-k3s-tunnel:
	@echo "Stopping K3s tunnel..."
	@PID=$$(lsof -ti:6443 | grep -v $$$$ | head -1); \
	if [ ! -z "$$PID" ]; then \
		kill $$PID 2>/dev/null && echo "K3s tunnel stopped (PID: $$PID)" || echo "Failed to stop tunnel"; \
	else \
		echo "No K3s tunnel found on port 6443"; \
	fi
	@echo "Reverting kubectl config to direct connection..."
	@kubectl config set-cluster default --server=https://192.168.68.62:6443 >/dev/null

# Start all tunnels (database and K3s)
tunnels: k3s-tunnel
	@echo "All tunnels are ready!"

# Stop all tunnels
stop-tunnels: stop-tunnel stop-k3s-tunnel
	@echo "All tunnels stopped"

