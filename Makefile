.PHONY: help build run test coverage seed clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/api cmd/api/main.go
	@go build -o bin/worker cmd/worker/main.go
	@echo "Build complete: bin/api, bin/worker"

run: ## Run the application locally
	@echo "Starting application..."
	@go run cmd/api/main.go

run-worker: ## Run the worker locally
	@echo "Starting worker..."
	@go run cmd/worker/main.go

run-hybrid: ## Run API with events enabled
	@echo "Starting API with events..."
	@ENABLE_EVENTS=true go run cmd/api/main.go

test: ## Run all tests
	@echo "Running tests..."
	@go test -v -race ./...

coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

seed: ## Seed sample data to Redis
	@echo "Seeding sample data..."
	@go run scripts/seed.go

seed-mysql: ## Seed sample data to MySQL
	@echo "Seeding MySQL data..."
	@go run scripts/seed_mysql.go

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-up: ## Start services with Docker Compose
	@echo "Starting services..."
	@docker-compose up -d
	@echo "Services started. Wait a few seconds, then run: make docker-seed"

docker-down: ## Stop services with Docker Compose
	@echo "Stopping services..."
	@docker-compose down
	@echo "Services stopped"

docker-seed: ## Seed data in Docker container
	@echo "Seeding data in Docker..."
	@docker-compose exec api go run scripts/seed.go

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: fmt vet ## Run formatters and linters

install: deps build ## Install dependencies and build

.DEFAULT_GOAL := help
