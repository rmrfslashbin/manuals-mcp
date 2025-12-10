# Manuals MCP Server Makefile

# Binary name
BINARY := manuals-mcp

# Build information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOMOD := $(GOCMD) mod

# Coverage threshold
COVERAGE_THRESHOLD := 80

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

.DEFAULT_GOAL := help

.PHONY: help
help: ## Display this help message
	@echo "$(CYAN)Manuals MCP Server - Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'

## Build targets

.PHONY: build
build: ## Build binary for current platform
	@echo "$(CYAN)Building $(BINARY)...$(NC)"
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY) ./cmd/$(BINARY)
	@echo "$(GREEN)✓ Build complete: bin/$(BINARY)$(NC)"

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "$(CYAN)Building for all platforms...$(NC)"
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/$(BINARY)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/$(BINARY)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/$(BINARY)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/$(BINARY)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)-windows-amd64.exe ./cmd/$(BINARY)
	@echo "$(GREEN)✓ Cross-compilation complete$(NC)"
	@ls -lh bin/

## Test targets

.PHONY: test
test: ## Run tests with race detector
	@echo "$(CYAN)Running tests...$(NC)"
	$(GOTEST) -v -race ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

.PHONY: test-short
test-short: ## Run tests without race detector
	$(GOTEST) -v -short ./...

.PHONY: coverage
coverage: ## Run tests with coverage (requires >= 80%)
	@echo "$(CYAN)Running tests with coverage...$(NC)"
	@$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(CYAN)Generating coverage report...$(NC)"
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(CYAN)Checking coverage threshold (>=$(COVERAGE_THRESHOLD)%)...$(NC)"
	@total=$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$total < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "$(RED)✗ Coverage $$total% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)✓ Coverage $$total% meets threshold$(NC)"; \
	fi
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

.PHONY: coverage-report
coverage-report: ## Generate and open coverage report
	@$(MAKE) coverage
	@command -v open >/dev/null 2>&1 && open coverage.html || echo "Open coverage.html in your browser"

## Code quality targets

.PHONY: fmt
fmt: ## Format code
	@echo "$(CYAN)Formatting code...$(NC)"
	@gofmt -l -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@echo "$(CYAN)Checking code formatting...$(NC)"
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "$(RED)✗ The following files are not formatted:$(NC)"; \
		echo "$$files"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ All files are formatted$(NC)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(CYAN)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ go vet passed$(NC)"

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "$(CYAN)Running staticcheck...$(NC)"
	@command -v staticcheck >/dev/null 2>&1 || { \
		echo "$(YELLOW)staticcheck not installed. Run: make install-tools$(NC)"; \
		exit 1; \
	}
	@staticcheck ./...
	@echo "$(GREEN)✓ staticcheck passed$(NC)"

.PHONY: deadcode
deadcode: ## Detect dead code
	@echo "$(CYAN)Checking for dead code...$(NC)"
	@command -v deadcode >/dev/null 2>&1 || { \
		echo "$(YELLOW)deadcode not installed. Run: make install-tools$(NC)"; \
		exit 1; \
	}
	@deadcode -test ./...
	@echo "$(GREEN)✓ No dead code found$(NC)"

.PHONY: vulncheck
vulncheck: ## Check for known vulnerabilities
	@echo "$(CYAN)Running govulncheck...$(NC)"
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "$(YELLOW)govulncheck not installed. Run: make install-tools$(NC)"; \
		exit 1; \
	}
	@govulncheck ./...
	@echo "$(GREEN)✓ No vulnerabilities found$(NC)"

.PHONY: lint
lint: ## Run golangci-lint
	@echo "$(CYAN)Running golangci-lint...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "$(YELLOW)golangci-lint not installed. See: https://golangci-lint.run/usage/install/$(NC)"; \
		exit 1; \
	}
	@golangci-lint run
	@echo "$(GREEN)✓ golangci-lint passed$(NC)"

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "$(CYAN)Running golangci-lint with auto-fix...$(NC)"
	@golangci-lint run --fix
	@echo "$(GREEN)✓ Fixes applied$(NC)"

## Quality gate targets

.PHONY: analyze
analyze: vet staticcheck deadcode ## Run all static analysis tools
	@echo "$(GREEN)✓ All static analysis checks passed$(NC)"

.PHONY: quality
quality: fmt-check analyze lint vulncheck ## Run all code quality checks
	@echo "$(GREEN)✓ All quality checks passed$(NC)"

.PHONY: check
check: quality coverage ## Run all checks including tests with coverage
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)✓ All checks passed successfully!$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════════════$(NC)"

.PHONY: ci
ci: tidy quality coverage build ## Full CI pipeline
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)✓ CI pipeline completed successfully!$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════════════$(NC)"

## Maintenance targets

.PHONY: tidy
tidy: ## Tidy go.mod
	@echo "$(CYAN)Tidying go.mod...$(NC)"
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ go.mod tidied$(NC)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(NC)"
	@rm -rf bin/ coverage.out coverage.html
	@echo "$(GREEN)✓ Clean complete$(NC)"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(CYAN)Installing development tools...$(NC)"
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install golang.org/x/tools/cmd/deadcode@latest
	@echo "$(GREEN)✓ Tools installed successfully$(NC)"

## Run targets

.PHONY: run
run: build ## Build and run the server
	./bin/$(BINARY) serve

.PHONY: run-debug
run-debug: build ## Build and run with debug logging
	./bin/$(BINARY) serve --log-level debug

.PHONY: index
index: build ## Build and run indexer
	./bin/$(BINARY) index --docs-path $(DOCS_PATH) --db-path $(DB_PATH)

## Info targets

.PHONY: version
version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

.PHONY: info
info: ## Display project information
	@echo "$(CYAN)Project Information$(NC)"
	@echo "Binary:     $(BINARY)"
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo ""
	@echo "$(CYAN)Coverage Threshold$(NC)"
	@echo "Required:   $(COVERAGE_THRESHOLD)%"
	@echo ""
	@echo "$(CYAN)Available Quality Checks$(NC)"
	@echo "  - fmt-check    : Code formatting"
	@echo "  - vet          : Go vet"
	@echo "  - staticcheck  : Static analysis"
	@echo "  - deadcode     : Dead code detection"
	@echo "  - lint         : golangci-lint"
	@echo "  - vulncheck    : Vulnerability scanning"
	@echo "  - coverage     : Test coverage >= $(COVERAGE_THRESHOLD)%"
