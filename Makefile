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

docker-up: ## Start PostgreSQL and Ollama
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@docker-compose exec -T postgres pg_isready -U postgres || (echo "PostgreSQL not ready yet, waiting..." && sleep 5)
	@echo "PostgreSQL is ready!"
	@echo "Ollama is ready!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Pull Ollama embedding model: make ollama-pull"
	@echo "2. Build and run: make build && ./go-doc-mcp"

docker-down: ## Stop all Docker services
	docker-compose down

docker-logs: ## Show Docker logs (all services)
	docker-compose logs -f

docker-reset: ## Reset all Docker data (delete all data)
	docker-compose down -v
	docker-compose up -d

ollama-pull: ## Download Ollama embedding model (nomic-embed-text)
	@echo "Downloading Ollama embedding model..."
	docker exec -it go-doc-mcp-ollama ollama pull nomic-embed-text
	@echo "Model downloaded successfully!"

ollama-list: ## List available Ollama models
	docker exec -it go-doc-mcp-ollama ollama list

ollama-pull-large: ## Download larger embedding model (mxbai-embed-large)
	@echo "Downloading larger embedding model..."
	docker exec -it go-doc-mcp-ollama ollama pull mxbai-embed-large
	@echo "Model downloaded! Update EMBEDDING_MODEL=mxbai-embed-large and EMBEDDING_DIMENSIONS=1024 in .env"

install: build ## Build and install to $GOPATH/bin
	cp go-doc-mcp $(GOPATH)/bin/

dev: docker-up ## Start development environment
	@echo "Development environment ready!"
	@echo "PostgreSQL: localhost:5432"
	@echo "Ollama: localhost:11434"
	@echo "Run 'make ollama-pull' to download the embedding model"
	@echo "Then run 'make run' to start the server"
