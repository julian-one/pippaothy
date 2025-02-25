APP_NAME := pippaothy

.PHONY: tailwind-build templ-generate build tailwind-watch templ-watch dev

# Build 
tailwind-build:
	~/bin/tailwindcss --input input.css --output ./static/css/output.css

templ-generate:
	templ generate

build: tailwind-build templ-generate
	go build -o ./bin/$(APP_NAME) ./cmd/main.go

# Watchers
tailwind-watch:
	~/bin/tailwindcss --input input.css --output ./static/css/output.css --watch

templ-watch:
	templ generate --watch

# Development target: build first, then start watchers and air concurrently.
dev: build
	$(MAKE) tailwind-watch &
	$(MAKE) templ-watch &
	air

