# SendFlix Makefile
# Production-ready build and deployment commands

.PHONY: help build run test clean docker-build docker-up docker-down migrate lint fmt deps

# Variables
APP_NAME := sendflix
VERSION := 1.0.0
BUILD_DIR := bin
GO := go
DOCKER_COMPOSE := docker-compose

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

##@ General

help: ## Display this help
	@echo "$(COLOR_BOLD)SendFlix - Email Delivery Service$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(COLOR_BLUE)<target>$(COLOR_RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-15s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Download Go dependencies
	@echo "$(COLOR_GREEN)Downloading dependencies...$(COLOR_RESET)"
	$(GO) mod download
	$(GO) mod tidy

build: ## Build the application
	@echo "$(COLOR_GREEN)Building $(APP_NAME)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/api

run: ## Run the application
	@echo "$(COLOR_GREEN)Running $(APP_NAME)...$(COLOR_RESET)"
	$(GO) run ./cmd/api

run-dev: ## Run with hot reload (requires air)
	@echo "$(COLOR_GREEN)Running with hot reload...$(COLOR_RESET)"
	air

test: ## Run tests
	@echo "$(COLOR_GREEN)Running tests...$(COLOR_RESET)"
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "$(COLOR_GREEN)Generating coverage report...$(COLOR_RESET)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_BLUE)Coverage report: coverage.html$(COLOR_RESET)"

test-integration: ## Run integration tests
	@echo "$(COLOR_GREEN)Running integration tests...$(COLOR_RESET)"
	$(GO) test -v -tags=integration ./tests/integration/...

benchmark: ## Run benchmarks
	@echo "$(COLOR_GREEN)Running benchmarks...$(COLOR_RESET)"
	$(GO) test -bench=. -benchmem ./...

lint: ## Run linter
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	golangci-lint run --timeout=5m

fmt: ## Format code
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	$(GO) fmt ./...
	gofumpt -l -w .

vet: ## Run go vet
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	$(GO) vet ./...

clean: ## Clean build artifacts
	@echo "$(COLOR_YELLOW)Cleaning...$(COLOR_RESET)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

##@ Database

migrate-up: ## Run database migrations up
	@echo "$(COLOR_GREEN)Running migrations up...$(COLOR_RESET)"
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run database migrations down
	@echo "$(COLOR_YELLOW)Running migrations down...$(COLOR_RESET)"
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create: ## Create new migration (use: make migrate-create name=create_users)
	@echo "$(COLOR_GREEN)Creating migration: $(name)...$(COLOR_RESET)"
	migrate create -ext sql -dir migrations -seq $(name)

migrate-force: ## Force migration version (use: make migrate-force version=1)
	@echo "$(COLOR_YELLOW)Forcing migration version: $(version)...$(COLOR_RESET)"
	migrate -path migrations -database "$(DATABASE_URL)" force $(version)

##@ Docker

docker-build: ## Build Docker image
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

docker-up: ## Start all services with Docker Compose
	@echo "$(COLOR_GREEN)Starting services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop all services
	@echo "$(COLOR_YELLOW)Stopping services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down

docker-logs: ## Show logs
	@echo "$(COLOR_BLUE)Showing logs...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) logs -f api

docker-restart: docker-down docker-up ## Restart all services

docker-clean: ## Remove all containers and volumes
	@echo "$(COLOR_YELLOW)Cleaning Docker resources...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

##@ Tools

install-tools: ## Install development tools
	@echo "$(COLOR_GREEN)Installing development tools...$(COLOR_RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/cosmtrek/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

install-migrate: ## Install golang-migrate
	@echo "$(COLOR_GREEN)Installing golang-migrate...$(COLOR_RESET)"
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

##@ Production

build-linux: ## Build for Linux (production)
	@echo "$(COLOR_GREEN)Building for Linux...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/api

build-all: ## Build for all platforms
	@echo "$(COLOR_GREEN)Building for all platforms...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/api
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/api
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/api
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/api

release: clean build-all ## Create release builds
	@echo "$(COLOR_GREEN)Creating release archives...$(COLOR_RESET)"
	cd $(BUILD_DIR) && tar czf $(APP_NAME)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar czf $(APP_NAME)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64
	cd $(BUILD_DIR) && tar czf $(APP_NAME)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64
	cd $(BUILD_DIR) && zip $(APP_NAME)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe

##@ Information

version: ## Show version
	@echo "$(COLOR_BLUE)$(APP_NAME) version $(VERSION)$(COLOR_RESET)"

status: ## Show service status
	@echo "$(COLOR_BLUE)Service status:$(COLOR_RESET)"
	$(DOCKER_COMPOSE) ps

info: ## Show project information
	@echo "$(COLOR_BOLD)Project Information$(COLOR_RESET)"
	@echo "Name:        $(APP_NAME)"
	@echo "Version:     $(VERSION)"
	@echo "Go Version:  $(shell $(GO) version)"
	@echo "Build Dir:   $(BUILD_DIR)"