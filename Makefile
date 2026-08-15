.PHONY: build dev dev-down test migrate-up migrate-down migrate-test-up migrate-test-down lint fmt vet validate-contract clean clean-all reset help

export PATH := $(HOME)/go/bin:$(HOME)/.local/bin:$(HOME)/.local/go/bin:$(PATH)

API_DIR := services/api
INFRA_DIR := infra
MIGRATIONS_DIR := $(API_DIR)/migrations
MIGRATION_SERVICES := auth identity content moderation
DATABASE_URL ?= postgres://yiguan:yiguan@localhost:15432/yiguan?sslmode=disable
TEST_DATABASE_URL ?= postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build the API server binary
	@mkdir -p .build
	cd $(API_DIR) && go build -o ../../.build/api ./cmd/api

dev: ## Start local dependencies and the API server
	@echo "Starting local infrastructure..."
	cd $(INFRA_DIR) && docker compose up -d --wait
	@if [ ! -f $(API_DIR)/.env ]; then \
		echo "Creating $(API_DIR)/.env from example..."; \
		cp $(INFRA_DIR)/.env.example $(API_DIR)/.env; \
	fi
	cd $(API_DIR) && go run ./cmd/api

dev-down: ## Stop local dependencies
	cd $(INFRA_DIR) && docker compose down

test: ## Run all Go unit tests
	cd $(API_DIR) && TEST_DATABASE_URL=$(TEST_DATABASE_URL) go test ./... -race -count=1 -p 1

.prepare-migrations:
	@rm -rf .build/migrations
	@mkdir -p .build/migrations
	@for svc in $(MIGRATION_SERVICES); do \
		for f in $(MIGRATIONS_DIR)/$$svc/*.sql; do \
			[ -e "$$f" ] || continue; \
			name=$$(basename "$$f"); \
			ver=$${name%%_*}; \
			rest=$${name#*_}; \
			cp "$$f" ".build/migrations/$${ver}_$${svc}_$${rest}"; \
		done; \
	done

migrate-up: .prepare-migrations ## Apply all database migrations
	migrate -path .build/migrations -database "$(DATABASE_URL)" up

migrate-down: .prepare-migrations ## Rollback all database migrations
	migrate -path .build/migrations -database "$(DATABASE_URL)" down -all

migrate-test-up: .prepare-migrations ## Apply all migrations to the test database
	migrate -path .build/migrations -database "$(TEST_DATABASE_URL)" up

migrate-test-down: .prepare-migrations ## Rollback all migrations from the test database
	migrate -path .build/migrations -database "$(TEST_DATABASE_URL)" down -all

lint: fmt vet ## Run all linters
	cd $(API_DIR) && if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed, skipping"; fi

fmt: ## Format Go source files
	cd $(API_DIR) && go fmt ./...

vet: ## Run go vet
	cd $(API_DIR) && go vet ./...

validate-contract: ## Validate the OpenAPI contract
	@scripts/validate-openapi.sh

clean: ## Clean build artifacts and test cache
	cd $(API_DIR) && go clean -testcache
	rm -rf .build

clean-all: ## Stop dependencies, remove data volumes, and clean build artifacts
	cd $(INFRA_DIR) && docker compose down -v
	$(MAKE) clean

reset: clean-all ## Alias for clean-all
