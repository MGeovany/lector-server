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
	@if command -v npx >/dev/null 2>&1 && [ -f package.json ]; then \
		npx prettier --write "**/*.go" 2>/dev/null || true; \
	fi

install-deps: ## Install dependencies
	$(GO) mod download
	$(GO) install github.com/air-verse/air@latest

kill-port: ## Kill process using port 8080
	@echo "Killing process on port 8080..."
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || echo "No process found on port 8080"

test: ## Run all tests
	$(GO) test ./... -v -race -coverprofile=coverage.out -coverpkg=./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-short: ## Run tests without coverage
	$(GO) test ./... -short -v

lint: ## Run linter
	@if command -v ~/go/bin/golangci-lint >/dev/null 2>&1; then \
		~/go/bin/golangci-lint run; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing..."; \
		$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		~/go/bin/golangci-lint run; \
	fi

fmt: ## Format code
	$(GO) fmt ./...
	@if command -v npx >/dev/null 2>&1 && [ -f package.json ]; then \
		npx prettier --write "**/*.go" 2>/dev/null || true; \
	fi

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

vet: ## Run go vet
	$(GO) vet ./...

check-env: ## Check if environment is properly configured
	@echo "Checking environment configuration..."
	@if [ ! -f .env ]; then echo "❌ .env file not found"; exit 1; fi
	@if ! grep -q "SUPABASE_URL=https://" .env; then echo "⚠️  Please configure SUPABASE_URL in .env"; fi
	@if ! grep -q "SUPABASE_ANON_KEY=" .env; then echo "⚠️  Please configure SUPABASE_ANON_KEY in .env"; fi
	@echo "✅ Environment check complete"

ci: ## Run CI pipeline (fmt, vet, lint, test)
	@echo "Running CI pipeline..."
	make fmt
	make vet
	make lint
	make test
	@echo "CI pipeline completed successfully!"
