# Makefile for manuals-mcp

BINARY_NAME=manuals-mcp
BIN_DIR=bin
MCPB_DIR=mcpb
MCPB_SERVER_DIR=$(MCPB_DIR)/server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build clean test coverage vet staticcheck deadcode lint check install-tools help
.PHONY: build-linux-arm64 build-windows mcpb-validate mcpb-build mcpb-pack mcpb-pack-all mcpb-info mcpb-clean

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

build-linux-arm64: ## Build for Linux arm64
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/manuals-mcp

build-windows: ## Build for Windows amd64
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/manuals-mcp

build-all: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows ## Build for all platforms

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html

## MCPB Packaging targets

mcpb-validate: ## Validate MCPB manifest
	@echo "Validating manifest.json..."
	@python3 -c "import json; json.load(open('$(MCPB_DIR)/manifest.json'))" && echo "✓ Valid JSON"
	@echo "Checking required fields..."
	@python3 -c "import json; m=json.load(open('$(MCPB_DIR)/manifest.json')); assert all(k in m for k in ['manifest_version','name','version','server']), 'Missing required fields'" && echo "✓ Required fields present"

mcpb-build: build-linux build-linux-arm64 build-darwin build-darwin-arm64 build-windows ## Build binaries for MCPB packaging
	@echo "All platform binaries built successfully"

mcpb-pack: mcpb-build ## Create MCPB package for current platform
	@echo "Creating MCPB package..."
	@mkdir -p $(MCPB_SERVER_DIR)
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		if [ "$$(uname -m)" = "arm64" ]; then \
			cp $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(MCPB_SERVER_DIR)/$(BINARY_NAME); \
		else \
			cp $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(MCPB_SERVER_DIR)/$(BINARY_NAME); \
		fi; \
	elif [ "$$(uname -s)" = "Linux" ]; then \
		if [ "$$(uname -m)" = "aarch64" ]; then \
			cp $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(MCPB_SERVER_DIR)/$(BINARY_NAME); \
		else \
			cp $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(MCPB_SERVER_DIR)/$(BINARY_NAME); \
		fi; \
	fi
	@chmod +x $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME).mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME).mcpb"

mcpb-pack-darwin-arm64: mcpb-build ## Create MCPB package for macOS arm64
	@mkdir -p $(MCPB_SERVER_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@chmod +x $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME)-darwin-arm64.mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64.mcpb"

mcpb-pack-darwin-amd64: mcpb-build ## Create MCPB package for macOS amd64
	@mkdir -p $(MCPB_SERVER_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@chmod +x $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME)-darwin-amd64.mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64.mcpb"

mcpb-pack-linux-amd64: mcpb-build ## Create MCPB package for Linux amd64
	@mkdir -p $(MCPB_SERVER_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@chmod +x $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME)-linux-amd64.mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME)-linux-amd64.mcpb"

mcpb-pack-linux-arm64: mcpb-build ## Create MCPB package for Linux arm64
	@mkdir -p $(MCPB_SERVER_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@chmod +x $(MCPB_SERVER_DIR)/$(BINARY_NAME)
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME)-linux-arm64.mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME)-linux-arm64.mcpb"

mcpb-pack-windows: mcpb-build ## Create MCPB package for Windows amd64
	@mkdir -p $(MCPB_SERVER_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MCPB_SERVER_DIR)/$(BINARY_NAME).exe
	@cd $(MCPB_DIR) && zip -r ../$(BIN_DIR)/$(BINARY_NAME)-windows-amd64.mcpb manifest.json server/
	@echo "Created $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.mcpb"

mcpb-pack-all: mcpb-pack-darwin-arm64 mcpb-pack-darwin-amd64 mcpb-pack-linux-amd64 mcpb-pack-linux-arm64 mcpb-pack-windows ## Create MCPB packages for all platforms
	@echo "All MCPB packages created in $(BIN_DIR)/"
	@ls -la $(BIN_DIR)/*.mcpb

mcpb-info: ## Show MCPB manifest info
	@echo "=== MCPB Package Info ==="
	@python3 -c "import json; m=json.load(open('$(MCPB_DIR)/manifest.json')); print(f\"Name: {m['name']}\nVersion: {m['version']}\nDisplay: {m['display_name']}\nTools: {len(m.get('tools', []))}\")"

mcpb-clean: ## Remove MCPB build artifacts
	rm -rf $(MCPB_SERVER_DIR) $(BIN_DIR)/*.mcpb

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
