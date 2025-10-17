APP_NAME := pippaothy
GOPATH := $(shell go env GOPATH)

# Tool paths
TAILWINDCSS := $(GOPATH)/bin/tailwindcss
TEMPL := $(GOPATH)/bin/templ
AIR := $(GOPATH)/bin/air

# Load environment variables from .env
ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev dev-tunnel dev-local stop-tunnel k3s-tunnel stop-k3s-tunnel tunnels stop-tunnels install-tools clean

# Tool installation
install-tools:
	@echo "Installing development tools..."
	@if ! test -f $(TAILWINDCSS); then \
		echo "Installing Tailwind CSS..."; \
		ARCH=$$(uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/'); \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]' | sed 's/darwin/macos/'); \
		URL="https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-$${OS}-$${ARCH}"; \
		echo "Downloading from: $$URL"; \
		curl -sLO "$$URL"; \
		if [ ! -f "tailwindcss-$${OS}-$${ARCH}" ] || [ $$(wc -c < "tailwindcss-$${OS}-$${ARCH}") -lt 1000 ]; then \
			echo "Error: Failed to download Tailwind CSS binary"; \
			rm -f "tailwindcss-$${OS}-$${ARCH}"; \
			exit 1; \
		fi; \
		chmod +x "tailwindcss-$${OS}-$${ARCH}"; \
		mkdir -p $(GOPATH)/bin; \
		mv "tailwindcss-$${OS}-$${ARCH}" $(GOPATH)/bin/tailwindcss; \
		echo "Installed tailwindcss to $(GOPATH)/bin/tailwindcss"; \
	fi
	@if ! test -f $(TEMPL); then \
		echo "Installing templ..."; \
		go install github.com/a-h/templ/cmd/templ@latest; \
	fi
	@if ! test -f $(AIR); then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "Done! Tools installed to $(GOPATH)/bin"

# Build targets
tailwind-build:
	@test -f $(TAILWINDCSS) || { echo "Tailwind CSS not found. Run 'make install-tools' first."; exit 1; }
	$(TAILWINDCSS) --input ./static/css/input.css --output ./static/css/output.css

templ-generate:
	@test -f $(TEMPL) || { echo "Templ not found. Run 'make install-tools' first."; exit 1; }
	$(TEMPL) generate

build: tailwind-build templ-generate
	go build -o ./bin/$(APP_NAME) ./main.go

# Watchers
tailwind-watch:
	@test -f $(TAILWINDCSS) || { echo "Tailwind CSS not found. Run 'make install-tools' first."; exit 1; }
	$(TAILWINDCSS) --input ./static/css/input.css --output ./static/css/output.css --watch

templ-watch:
	@test -f $(TEMPL) || { echo "Templ not found. Run 'make install-tools' first."; exit 1; }
	$(TEMPL) generate --watch

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
	@test -f $(AIR) || { echo "Air not found. Run 'make install-tools' first."; exit 1; }
	@trap 'kill 0' EXIT; \
	$(MAKE) tailwind-watch & \
	$(MAKE) templ-watch & \
	DB_HOST=localhost DB_PORT=5433 DB_USER=k3s_user DB_PASSWORD=pippa DB_NAME=k3s_db $(AIR)

# Alias for backward compatibility
dev-tunnel: dev

# Development without SSH tunnel (uses environment variables or .env file)
dev-local: build
	@test -f $(AIR) || { echo "Air not found. Run 'make install-tools' first."; exit 1; }
	@trap 'kill 0' EXIT; \
	$(MAKE) tailwind-watch & \
	$(MAKE) templ-watch & \
	DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) $(AIR)

clean:
	rm -rf ./bin

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

