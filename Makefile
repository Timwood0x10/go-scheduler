# Makefile for GPU Scheduler

.PHONY: all build test lint vet fmt clean install run

# Default target
all: lint test build

# Build the scheduler
build:
	@echo "Building scheduler..."
	@go build -o bin/scheduler ./cmd/scheduler

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Static code analysis with go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -l -w .

# Lint with golint
lint:
	@echo "Running golint..."
	@~/go/bin/golint ./...

# Static check with golangci-lint
golangci:
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

# Static check (vet + lint)
static: vet lint golangci

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run the scheduler
run:
	@./bin/scheduler -port :50051

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/gpu_scheduler.proto

# Generate Python SDK
proto-python:
	@echo "Generating Python protobuf code..."
	@python3 -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. \
		api/gpu_scheduler.proto

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Run lint, test and build (default)"
	@echo "  build        - Build the scheduler"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  vet          - Run go vet"
	@echo "  fmt          - Format code with gofmt"
	@echo "  lint         - Run golint"
	@echo "  static       - Run static analysis (vet + lint + golangci-lint)"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install dependencies"
	@echo "  run          - Run the scheduler"
	@echo "  proto        - Generate protobuf Go code"
	@echo "  proto-python - Generate protobuf Python code"
	@echo "  help         - Show this help message"
