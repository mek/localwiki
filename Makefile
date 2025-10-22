.PHONY: help test test-verbose test-coverage test-race build run clean docker-build docker-run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage report
	go test -cover ./...

test-coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	go test -race ./...

test-benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

build: ## Build the application
	go build -o wiki .

run: ## Run the application
	go run .

clean: ## Clean build artifacts and test files
	rm -f wiki coverage.out coverage.html
	rm -rf data/

docker-build: ## Build Docker image
	docker build -t wiki:latest .

docker-run: ## Run application in Docker
	docker compose build --no-cache
	docker-compose up -d

docker-stop: ## Stop Docker containers
	docker-compose down

lint: ## Run linter (requires golangci-lint)
	@which golangci-lint > /dev/null || echo "golangci-lint not installed. Install with: brew install golangci-lint"
	@which golangci-lint > /dev/null && golangci-lint run || true

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

ci: fmt vet test ## Run CI checks (format, vet, test)
