# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage:'
	@sed -n 's/^[a-zA-Z0-9_-]*:.*##/&/p' ${MAKEFILE_LIST} | \
		awk 'BEGIN {FS = ":.*##"}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-update-golden
test-update-golden: ## Update golden files
	@echo "Updating golden files..."
	go test . -update
	@echo "Golden files updated successfully!"

.PHONY: test-clean
test-clean: ## Run tests without cache
	@echo "Running tests without cache..."
	@go test -count=1 -v ./...

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@gofmt -s -w .
	@echo "Code formatted successfully!"

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: build-example
build-example: ## Build custom golangci-lint with this linter
	@echo "Building custom golangci-lint binary..."
	@cd example && golangci-lint custom
	@echo "Custom binary built: example/custom-gcl"

.PHONY: install-example
install-example: build-example ## Install custom golangci-lint binary
	@echo "Installing custom binary to $$GOPATH/bin..."
	@cp example/custom-gcl $$GOPATH/bin/custom-gcl
	@echo "Installed to: $$GOPATH/bin/custom-gcl"

.PHONY: run-example
run-example: build-example ## Run custom linter on example code
	@echo "Running custom linter on example..."
	@cd example && ./custom-gcl run

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy
	@echo "Modules tidied successfully!"

.PHONY: clean
clean: ## Clean build artifacts and test cache
	@echo "Cleaning..."
	@go clean -cache -testcache
	@rm -f coverage.out coverage.html
	@rm -f example/custom-gcl
	@echo "Cleaned successfully!"

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "All checks passed!"

.PHONY: ci
ci: vet test-race ## Run CI checks
	@echo "CI checks passed!"
