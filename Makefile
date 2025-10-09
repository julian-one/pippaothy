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

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev install-tools clean

# Tool installation
install-tools:
	@echo "Installing development tools..."
	@if ! test -f $(TAILWINDCSS); then \
		echo "Installing Tailwind CSS..."; \
		ARCH=$$(uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/'); \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		curl -sLO "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-$${OS}-$${ARCH}"; \
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
	@test -f $(AIR) || { echo "Air not found. Run 'make install-tools' first."; exit 1; }
	@trap 'kill 0' EXIT; \
	$(MAKE) tailwind-watch & \
	$(MAKE) templ-watch & \
	$(AIR)

clean:
	rm -rf ./bin

