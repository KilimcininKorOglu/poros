# Poros Makefile
# Modern Network Path Tracer

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOGET = $(GOCMD) get
GOFMT = gofmt
GOVET = $(GOCMD) vet

# Binary names
BINARY_NAME = poros
BINARY_DIR = bin

# Build flags
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Platforms for cross-compilation
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test lint fmt vet deps build-all install help

# Default target
all: clean deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/poros
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BINARY_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		output=$(BINARY_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $$output ./cmd/poros; \
	done
	@echo "All builds complete!"

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/poros

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Lint the code
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@echo "Clean complete"

# Run the application
run: build
	@$(BINARY_DIR)/$(BINARY_NAME) $(ARGS)

# Development build (faster, no optimizations stripped)
dev:
	@echo "Building development version..."
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/poros

# Show help
help:
	@echo "Poros Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the project (clean, deps, test, build)"
	@echo "  make build        Build the binary"
	@echo "  make build-all    Build for all platforms"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make bench        Run benchmarks"
	@echo "  make lint         Run golangci-lint"
	@echo "  make fmt          Format code"
	@echo "  make vet          Run go vet"
	@echo "  make deps         Download dependencies"
	@echo "  make clean        Clean build artifacts"
	@echo "  make run ARGS=... Run the application"
	@echo "  make dev          Quick development build"
	@echo "  make help         Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  DATE=$(DATE)"
