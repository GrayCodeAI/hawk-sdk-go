NAME := hawk-sdk-go

.PHONY: all test lint fmt vet clean help

all: lint test

test: ## Run tests with race detector
	go test ./... -race -count=1 -timeout=60s

test-coverage: ## Run tests with coverage
	go test ./... -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | grep "^total:"

lint: ## Run linter
	golangci-lint run ./... --timeout=5m

fmt: ## Format code
	gofumpt -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

clean: ## Clean artifacts
	rm -f coverage.out

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
