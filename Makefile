# Fuego API Testing Framework Makefile

.PHONY: build test clean fmt lint vet install run-example help

# Variables
BINARY_NAME=fuego
BUILD_DIR=build
MAIN_PATH=./cmd/fuego
EXAMPLE_SCENARIO=examples/simple-api-test.yaml

# Default target
help: ## Show this help message
	@echo "Fuego API Testing Framework"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "Installed $(BINARY_NAME) to GOPATH/bin"

test: ## Run tests
	@echo "Running tests..."
	@go test ./... -v

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run golint
	@echo "Running golint..."
	@golint ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -f examples/*.json examples/*.html examples/*.md
	@echo "Cleaned build artifacts"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

run-example: build ## Build and run example scenario
	@echo "Running example scenario..."
	@$(BUILD_DIR)/$(BINARY_NAME) run $(EXAMPLE_SCENARIO)

run-example-verbose: build ## Build and run example scenario with verbose output
	@echo "Running example scenario (verbose)..."
	@$(BUILD_DIR)/$(BINARY_NAME) run --verbose $(EXAMPLE_SCENARIO)

run-example-json: build ## Build and run example scenario with JSON output
	@echo "Running example scenario (JSON output)..."
	@$(BUILD_DIR)/$(BINARY_NAME) run --format json $(EXAMPLE_SCENARIO)

dev: ## Development mode - format, vet, test, build
	@$(MAKE) fmt
	@$(MAKE) vet
	@$(MAKE) test
	@$(MAKE) build

all: clean deps fmt vet test build ## Run all checks and build

# Docker targets (for future use)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t fuego:latest .

docker-run: ## Run in Docker container
	@echo "Running in Docker..."
	@docker run --rm -v $(PWD)/examples:/examples fuego:latest run /examples/simple-api-test.yaml

# Release targets
release-build: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "Built binaries for multiple platforms in $(BUILD_DIR)/"

check: fmt vet test ## Run all checks (format, vet, test)