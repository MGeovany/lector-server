.PHONY: run dev build clean help

# Default target
.DEFAULT_GOAL := help

# Variables
AIR := $(shell which ~/go/bin/air 2>/dev/null || echo "~/go/bin/air")
GO := go

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Run with hot-reload (requires air)
	@if [ ! -f ~/go/bin/air ]; then \
		echo "Air not found. Installing..."; \
		$(GO) install github.com/air-verse/air@latest; \
	fi
	@~/go/bin/air

run: ## Run the server directly
	$(GO) run cmd/server/main.go

build: ## Build the server
	@mkdir -p bin
	$(GO) build -o bin/server ./cmd/server

clean: ## Clean build artifacts
	rm -rf bin/ tmp/ build-errors.log

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format code
	$(GO) fmt ./...

install-deps: ## Install dependencies
	$(GO) mod download
	$(GO) install github.com/air-verse/air@latest

kill-port: ## Kill process using port 8080
	@echo "Killing process on port 8080..."
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || echo "No process found on port 8080"

check-env: ## Check if environment is properly configured
	@echo "Checking environment configuration..."
	@if [ ! -f .env ]; then echo "❌ .env file not found"; exit 1; fi
	@if ! grep -q "SUPABASE_URL=https://" .env; then echo "⚠️  Please configure SUPABASE_URL in .env"; fi
	@if ! grep -q "SUPABASE_ANON_KEY=" .env; then echo "⚠️  Please configure SUPABASE_ANON_KEY in .env"; fi
	@echo "✅ Environment check complete"

