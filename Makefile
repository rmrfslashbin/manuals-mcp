# Makefile for manuals-mcp

BINARY_NAME=manuals-mcp
BIN_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build clean test coverage vet staticcheck deadcode lint check install-tools help

all: check build

## Build targets

build: ## Build the binary
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/manuals-mcp

build-linux: ## Build for Linux amd64
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/manuals-mcp

build-darwin: ## Build for macOS amd64
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/manuals-mcp

build-darwin-arm64: ## Build for macOS arm64
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/manuals-mcp

build-all: build-linux build-darwin build-darwin-arm64 ## Build for all platforms

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html

## Test targets

test: ## Run tests
	go test -v ./...

coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

coverage-html: coverage ## Generate HTML coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Analysis targets

vet: ## Run go vet
	go vet ./...

staticcheck: ## Run staticcheck
	@which staticcheck > /dev/null 2>&1 || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	@staticcheck ./... 2>/dev/null || $(shell go env GOPATH)/bin/staticcheck ./...

deadcode: ## Run deadcode analysis
	@which deadcode > /dev/null 2>&1 || (echo "Installing deadcode..." && go install golang.org/x/tools/cmd/deadcode@latest)
	@deadcode ./... 2>/dev/null || $(shell go env GOPATH)/bin/deadcode ./...

lint: vet staticcheck deadcode ## Run all linters

check: lint test ## Run all checks (lint + test)

## Dependency targets

tidy: ## Tidy go modules
	go mod tidy

deps: ## Download dependencies
	go mod download

install-tools: ## Install analysis tools
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/deadcode@latest

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
