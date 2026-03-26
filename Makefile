# Makefile for AlgoGPU - Simple & Efficient

# Go commands
GO := go
GOLANGCI_LINT := golangci-lint

# Build output directory
BIN_DIR := bin

.PHONY: all build build-standalone build-plugin test test-race static-check clean deps tidy help

# Default target
all: static-check test build

# ============================================
# Build Targets
# ============================================

build: build-standalone build-plugin
	@echo "Build complete: standalone and plugin modes"

build-standalone:
	@echo "Building standalone mode..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BIN_DIR)/scheduler ./cmd/scheduler

build-plugin:
	@echo "Building plugin mode..."
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BIN_DIR)/plugin ./cmd/plugin

# ============================================
# Test Targets
# ============================================

test:
	@echo "Running tests..."
	@$(GO) test -v -timeout 60s \
		./internal/plugin/ \
		./internal/scheduler/ \
		./internal/gpu/ \
		./internal/queue/ \
		./internal/executor/ \
		./pkg/types/

test-race:
	@echo "Running tests with race detector..."
	@$(GO) test -race -v -timeout 60s \
		./internal/plugin/ \
		./internal/scheduler/ \
		./internal/gpu/ \
		./internal/queue/ \
		./internal/executor/ \
		./pkg/types/

# ============================================
# Code Quality Targets
# ============================================

static-check:
	@echo "Running static checks..."
	@echo "  - Formatting code..."
	@$(GO) fmt ./...
	@echo "  - Running go vet..."
	@$(GO) vet ./...
	@echo "  - Running golangci-lint..."
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "  - golangci-lint not found, skipping..."; \
		echo "    Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	@echo "Static checks passed!"

# ============================================
# Run Targets
# ============================================

run-standalone: build-standalone
	@echo "Running standalone mode..."
	@$(BIN_DIR)/scheduler

run-plugin: build-plugin
	@echo "Running plugin mode..."
	@$(BIN_DIR)/plugin

# ============================================
# Clean Targets
# ============================================

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)/

# ============================================
# Dependency Targets
# ============================================

deps:
	@echo "Downloading dependencies..."
	@$(GO) mod download

tidy:
	@echo "Tidying dependencies..."
	@$(GO) mod tidy

# ============================================
# Help
# ============================================

help:
	@echo "AlgoGPU Makefile - Simple & Efficient"
	@echo "======================================"
	@echo ""
	@echo "Build:"
	@echo "  make build             - Build both modes (standalone + plugin)"
	@echo "  make build-standalone  - Build standalone service mode"
	@echo "  make build-plugin      - Build plugin mode"
	@echo ""
	@echo "Test:"
	@echo "  make test              - Run all tests"
	@echo "  make test-race         - Run tests with race detector"
	@echo ""
	@echo "Code Quality:"
	@echo "  make static-check      - Run all static checks (fmt + vet + lint)"
	@echo ""
	@echo "Run:"
	@echo "  make run-standalone    - Run standalone service mode"
	@echo "  make run-plugin        - Run plugin mode"
	@echo ""
	@echo "Clean:"
	@echo "  make clean             - Clean build artifacts"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps              - Download dependencies"
	@echo "  make tidy              - Tidy dependencies"
	@echo ""
	@echo "Other:"
	@echo "  make all               - Static check + test + build (default)"
	@echo "  make help              - Show this help message"