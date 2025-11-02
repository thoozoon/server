# Makefile for Go server module

# Variables
BINARY_DIR = bin
GO_MODULE = server
SERVER_TEST_CONFIG = "../configs/server-config-local.toml"

# Default target
all: build

# Create bin directory if it doesn't exist
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# Build all commands in cmd directory
build: $(BINARY_DIR)
	@echo "Building Go binaries..."
	@for dir in cmd/*/; do \
		if [ -d "$$dir" ] && ls "$$dir"*.go >/dev/null 2>&1; then \
			binary_name=$$(basename "$$dir"); \
			echo "Building $$binary_name..."; \
			go build -o $(BINARY_DIR)/$$binary_name ./$$dir; \
		fi; \
	done

# Install dependencies
deps:
	go mod tidy
	go mod download

test: build
	./$(BINARY_DIR)/server "$(SERVER_TEST_CONFIG)"

deploy: build


# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

.PHONY: all build clean deps test fmt vet lint dev help
