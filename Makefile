.PHONY: build clean test run install

# Build variables
BINARY_NAME=kubectx-timeout
BUILD_DIR=bin
MAIN_PATH=./cmd/kubectx-timeout

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.txt ./...

# Run the daemon
run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Install to local bin
install:
	@echo "Installing to /usr/local/bin/..."
	go build -o /usr/local/bin/$(BINARY_NAME) $(MAIN_PATH)

# Development helpers
fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linters..."
	golangci-lint run

tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run tests"
	@echo "  run      - Build and run the daemon"
	@echo "  install  - Install to /usr/local/bin"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run linters"
	@echo "  tidy     - Tidy dependencies"
