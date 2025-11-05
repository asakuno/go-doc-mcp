.PHONY: help build run test clean docker-up docker-down docker-logs deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o go-doc-mcp main.go

run: ## Run the application
	go run main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f go-doc-mcp
	go clean

docker-up: ## Start PostgreSQL with pgvector
	docker-compose up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	@docker-compose exec -T postgres pg_isready -U postgres || (echo "PostgreSQL not ready yet, waiting..." && sleep 5)
	@echo "PostgreSQL is ready!"

docker-down: ## Stop PostgreSQL
	docker-compose down

docker-logs: ## Show PostgreSQL logs
	docker-compose logs -f postgres

docker-reset: ## Reset PostgreSQL (delete all data)
	docker-compose down -v
	docker-compose up -d

install: build ## Build and install to $GOPATH/bin
	cp go-doc-mcp $(GOPATH)/bin/

dev: docker-up ## Start development environment
	@echo "Development environment ready!"
	@echo "PostgreSQL: localhost:5432"
	@echo "Run 'make run' to start the server"
