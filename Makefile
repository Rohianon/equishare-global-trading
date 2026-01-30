# EquiShare Global Trading - Makefile
# ====================================

.PHONY: all build test lint clean dev dev-down help

# Default target
all: build

# ============================================
# Build
# ============================================

# Build all services (uses go.work)
build:
	@echo "Building all services..."
	@go build ./pkg/...
	@go build ./services/api-gateway/...
	@go build ./services/auth-service/...
	@go build ./services/user-service/...
	@go build ./services/trading-service/...
	@go build ./services/payment-service/...
	@go build ./services/ussd-service/...
	@go build ./services/notification-service/...
	@go build ./services/market-data-service/...
	@go build ./services/portfolio-service/...
	@echo "Build complete!"

# Build a specific service (usage: make build-service SVC=api-gateway)
build-service:
	@echo "Building services/$(SVC)..."
	@cd services/$(SVC) && go build -o bin/service .
	@echo "Built services/$(SVC)/bin/service"

# ============================================
# Test
# ============================================

# Run all tests (uses go.work)
test:
	@echo "Running tests..."
	@go test -v ./pkg/...
	@go test -v ./services/api-gateway/...
	@go test -v ./services/auth-service/...
	@go test -v ./services/user-service/...
	@go test -v ./services/trading-service/...
	@go test -v ./services/payment-service/...
	@go test -v ./services/ussd-service/...
	@go test -v ./services/notification-service/...
	@go test -v ./services/market-data-service/...
	@go test -v ./services/portfolio-service/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ============================================
# Lint
# ============================================

# Run linter (uses go.work)
lint:
	@echo "Running linter..."
	golangci-lint run ./pkg/... ./services/...

# Fix lint issues
lint-fix:
	@echo "Fixing lint issues..."
	golangci-lint run --fix ./pkg/... ./services/...

# ============================================
# Development
# ============================================

# Start development dependencies (Postgres, Redis, Kafka)
dev:
	@echo "Starting development dependencies..."
	docker-compose up -d postgres redis kafka kafka-ui jaeger
	@echo "Dependencies started!"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis:      localhost:6379"
	@echo "  Kafka:      localhost:9092"
	@echo "  Kafka UI:   http://localhost:8080"
	@echo "  Jaeger:     http://localhost:16686"

# Start all services
dev-all:
	@echo "Starting all services..."
	docker-compose up -d

# Stop development dependencies
dev-down:
	@echo "Stopping development dependencies..."
	docker-compose down

# Reset development environment (removes volumes)
dev-reset:
	@echo "Resetting development environment..."
	docker-compose down -v
	docker-compose up -d

# View logs
dev-logs:
	docker-compose logs -f

# ============================================
# Database
# ============================================

MIGRATE_URL ?= postgres://equishare:equishare_dev@localhost:5432/equishare?sslmode=disable

# Run migrations up
migrate-up:
	@echo "Running migrations..."
	migrate -path migrations -database "$(MIGRATE_URL)" up

# Rollback one migration
migrate-down:
	@echo "Rolling back one migration..."
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

# Create new migration (usage: make migrate-create name=create_users_table)
migrate-create:
	@echo "Creating migration: $(name)"
	migrate create -ext sql -dir migrations -seq $(name)

# Force migration version (usage: make migrate-force version=1)
migrate-force:
	@echo "Forcing migration version: $(version)"
	migrate -path migrations -database "$(MIGRATE_URL)" force $(version)

# ============================================
# Code Generation
# ============================================

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	buf generate

# Generate mocks
mocks:
	@echo "Generating mocks..."
	go generate ./...

# ============================================
# Clean
# ============================================

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@for dir in services/*/; do \
		rm -rf $${dir}bin; \
	done
	rm -f coverage.out coverage.html
	@echo "Clean complete!"

# ============================================
# Utilities
# ============================================

# Sync Go workspace
sync:
	@echo "Syncing Go workspace..."
	go work sync

# Tidy all modules
tidy:
	@echo "Tidying all modules..."
	@cd pkg && go mod tidy
	@for dir in services/*/; do \
		echo "  Tidying $${dir}..."; \
		(cd $${dir} && go mod tidy); \
	done
	@echo "Tidy complete!"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofumpt -l -w .

# ============================================
# Help
# ============================================

help:
	@echo "EquiShare Global Trading - Available Commands"
	@echo "=============================================="
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build all services"
	@echo "  make build-service SVC=<name> - Build specific service"
	@echo ""
	@echo "Test:"
	@echo "  make test            - Run all tests"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo ""
	@echo "Lint:"
	@echo "  make lint            - Run linter"
	@echo "  make lint-fix        - Fix lint issues"
	@echo ""
	@echo "Development:"
	@echo "  make dev             - Start dev dependencies (Postgres, Redis, Kafka)"
	@echo "  make dev-all         - Start all services"
	@echo "  make dev-down        - Stop dev dependencies"
	@echo "  make dev-reset       - Reset dev environment (removes data)"
	@echo "  make dev-logs        - View container logs"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up      - Run all migrations"
	@echo "  make migrate-down    - Rollback one migration"
	@echo "  make migrate-create name=<name> - Create new migration"
	@echo ""
	@echo "Utilities:"
	@echo "  make sync            - Sync Go workspace"
	@echo "  make tidy            - Tidy all modules"
	@echo "  make fmt             - Format code"
	@echo "  make clean           - Clean build artifacts"
