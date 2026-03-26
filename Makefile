# Makefile for GPU Scheduler

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
BUILD_FLAGS := $(LDFLAGS) -trimpath

# Build output directories
BIN_DIR := bin
PROTO_DIR := api
PYTHON_DIR := python

# Go commands
GO := go
GOFMT := gofmt
GOLINT := ~/go/bin/golint
GOLANGCI_LINT := golangci-lint

# Docker settings
DOCKER_IMAGE := algogpu/scheduler
DOCKER_TAG := $(VERSION)

.PHONY: all build test lint vet fmt clean install run
.PHONY: build-debug build-release build-all
.PHONY: test test-coverage test-race test-bench
.PHONY: check-deps verify-proto
.PHONY: proto proto-all proto-check
.PHONY: docker-build docker-push
.PHONY: deps deps-update
.PHONY: help

# Default target
all: lint test build

# ============================================
# Build Targets
# ============================================

# Build (release)
build: clean
	@echo "Building scheduler (release)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/scheduler ./cmd/scheduler

# Build (debug)
build-debug:
	@echo "Building scheduler (debug)..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -gcflags="all=-N -l" -o $(BIN_DIR)/scheduler-debug ./cmd/scheduler

# Build all variants
build-all: build build-debug

# ============================================
# Test Targets
# ============================================

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./...

# Run tests with race detection
test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -race -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GO) test -v -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@$(GO) tool cover -func=coverage.out

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./...

# ============================================
# Code Quality Targets
# ============================================

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -l -w .
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "Files need formatting:"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

# Format check
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "Files need formatting:"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

# Static code analysis with go vet
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

# Lint with golint
lint:
	@echo "Running golint..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) ./...; \
	else \
		echo "golint not found, skipping..."; \
	fi

# Static check with golangci-lint
golangci:
	@echo "Running golangci-lint..."
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Static check (vet + lint + golangci)
static: vet lint golangci

# Security check
sec:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, skipping..."; \
	fi

# ============================================
# Dependency Management
# ============================================

# Check dependencies
check-deps:
	@echo "Checking dependencies..."
	@$(GO) mod verify
	@$(GO) mod tidy

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy

# Check for outdated dependencies
deps-outdated:
	@echo "Checking for outdated dependencies..."
	@$(GO) list -u -m all

# ============================================
# Protobuf Targets
# ============================================

# Generate protobuf code (Go)
proto:
	@echo "Generating protobuf Go code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/gpu_scheduler.proto

# Generate Python protobuf code
proto-python:
	@echo "Generating Python protobuf code..."
	@python3 -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. \
		$(PROTO_DIR)/gpu_scheduler.proto

# Generate all protobuf code
proto-all: proto proto-python

# Check if protobuf code is up to date
proto-check:
	@echo "Checking protobuf code..."
	@$(MAKE) proto-all
	@if [ -n "$$(git diff --name-only)" ]; then \
		echo "Protobuf code is out of date. Please run 'make proto-all'"; \
		git diff --name-only; \
		exit 1; \
	fi

# ============================================
# Docker Targets
# ============================================

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest

# ============================================
# Cross-compilation Targets
# ============================================

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/scheduler-linux-amd64 ./cmd/scheduler

# Build for macOS (Intel)
build-macos:
	@echo "Building for macOS (Intel)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/scheduler-darwin-amd64 ./cmd/scheduler

# Build for macOS (Apple Silicon)
build-macos-arm:
	@echo "Building for macOS (Apple Silicon)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/scheduler-darwin-arm64 ./cmd/scheduler

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/scheduler-windows-amd64.exe ./cmd/scheduler

# Build all platforms
build-platforms: build-linux build-macos build-macos-arm build-windows

# ============================================
# Run Targets
# ============================================

# Run the scheduler
run: build
	@echo "Running scheduler..."
	@$(BIN_DIR)/scheduler -port :50051

# Run with hot reload (requires air)
run-dev:
	@echo "Running with hot reload..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; \
	fi

# ============================================
# Clean Targets
# ============================================

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)/
	@rm -f coverage.out coverage.html
	@rm -f *.log

# Clean everything (including generated files)
clean-all: clean
	@echo "Cleaning generated files..."
	@rm -f $(PROTO_DIR)/*_pb2.py $(PROTO_DIR)/*_pb2_grpc.py
	@rm -f $(PYTHON_DIR)/*_pb2.py $(PYTHON_DIR)/*_pb2_grpc.py

# ============================================
# Version Info
# ============================================

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# ============================================
# CI/CD Targets
# ============================================

# Run all checks for CI
ci: fmt-check vet test-race static
	@echo "CI checks passed!"

# ============================================
# Help
# ============================================

help:
	@echo "GPU Scheduler Makefile"
	@echo "======================"
	@echo ""
	@echo "Build Targets:"
	@echo "  build            - Build scheduler (release, default)"
	@echo "  build-debug      - Build scheduler (debug)"
	@echo "  build-all        - Build all variants"
	@echo "  build-linux      - Cross-compile for Linux"
	@echo "  build-macos      - Cross-compile for macOS (Intel)"
	@echo "  build-macos-arm  - Cross-compile for macOS (Apple Silicon)"
	@echo "  build-windows    - Cross-compile for Windows"
	@echo "  build-platforms  - Build for all platforms"
	@echo ""
	@echo "Test Targets:"
	@echo "  test             - Run tests"
	@echo "  test-race        - Run tests with race detector"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-bench       - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt              - Format code"
	@echo "  fmt-check        - Check code formatting"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run golint"
	@echo "  golangci         - Run golangci-lint"
	@echo "  static           - Run all static checks"
	@echo "  sec              - Run security checks"
	@echo "  ci               - Run all CI checks"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps             - Install dependencies"
	@echo "  deps-update      - Update dependencies"
	@echo "  deps-outdated    - Check for outdated dependencies"
	@echo "  check-deps       - Verify dependencies"
	@echo ""
	@echo "Protobuf:"
	@echo "  proto            - Generate Go protobuf code"
	@echo "  proto-python     - Generate Python protobuf code"
	@echo "  proto-all        - Generate all protobuf code"
	@echo "  proto-check      - Check if protobuf is up to date"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-push      - Push Docker image"
	@echo ""
	@echo "Run:"
	@echo "  run              - Build and run scheduler"
	@echo "  run-dev          - Run with hot reload (requires air)"
	@echo ""
	@echo "Clean:"
	@echo "  clean            - Clean build artifacts"
	@echo "  clean-all        - Clean everything including generated files"
	@echo ""
	@echo "Other:"
	@echo "  version          - Show version information"
	@echo "  help             - Show this help message"
	@echo "  all              - Run lint, test and build (default)"
