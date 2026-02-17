.PHONY: build test test-interactive test-integration lint fmt clean install help setup-hooks

# Variables
export GOTOOLCHAIN := auto
BINARY_NAME := parkster
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/petter-b/parkster-cli/internal/commands.Version=$(VERSION) \
                     -X github.com/petter-b/parkster-cli/internal/commands.Commit=$(COMMIT) \
                     -X github.com/petter-b/parkster-cli/internal/commands.BuildDate=$(BUILD_DATE)"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Code signing (macOS only — set via env or make arg)
CODESIGN_IDENTITY ?= $(SIGN_IDENTITY)

# Directories
BIN_DIR := ./bin
CMD_DIR := ./cmd/parkster

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@if [ -n "$(CODESIGN_IDENTITY)" ] && command -v codesign >/dev/null 2>&1; then \
		echo "Signing $(BIN_DIR)/$(BINARY_NAME)..."; \
		codesign -s "$(CODESIGN_IDENTITY)" -f $(BIN_DIR)/$(BINARY_NAME); \
	fi
	@echo "Binary: $(BIN_DIR)/$(BINARY_NAME)"

## install: Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(LDFLAGS) $(CMD_DIR)

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -v ./...

## test-interactive: Run interactive tests (requires keychain access)
test-interactive:
	@echo "Running interactive tests (may prompt for keychain access)..."
	$(GOTEST) -v -tags interactive ./internal/commands/

## test-integration: Run integration tests against live API (requires .env)
test-integration:
	@echo "Running integration tests..."
	@if [ -f .env ]; then \
		set -a && . ./.env && set +a && \
		$(GOTEST) -v -tags integration ./internal/parkster/; \
	else \
		echo "Error: .env file not found. Create it with PARKSTER_USERNAME and PARKSTER_PASSWORD."; \
		exit 1; \
	fi

## test-cover: Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"



## lint: Run linters (requires golangci-lint)
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  brew install golangci-lint"; \
		echo "  or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed (optional). Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

## setup-hooks: Configure git to use project hooks
setup-hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks configured."

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

## run: Build and run with args (use: make run ARGS="--help")
run: build
	$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

## dev: Build and run in debug mode
dev: build
	$(BIN_DIR)/$(BINARY_NAME) --debug $(ARGS)

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

# Cross-compilation targets
## build-all: Build for all platforms
build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-linux-arm64 build-windows-amd64

build-darwin-arm64:
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-darwin-amd64:
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

build-linux-amd64:
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

build-linux-arm64:
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-windows-amd64:
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
