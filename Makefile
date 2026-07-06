# Canonical hawk-eco Makefile for Go library repos.
# Source of truth: https://github.com/GrayCodeAI/hawk/blob/main/.shared-templates/Makefile.library.tmpl
# Placeholders rendered per repo: hawk-sdk-go.

# ---------------------------------------------------------------------------
# Project metadata
# ---------------------------------------------------------------------------
NAME := hawk-sdk-go

# ---------------------------------------------------------------------------
# Versioning — sourced from VERSION file; falls back to git describe.
# See https://github.com/GrayCodeAI/hawk/blob/main/VERSIONING.md.
# ---------------------------------------------------------------------------
VERSION ?= $(shell cat VERSION 2>/dev/null | head -n1 | tr -d '[:space:]' || git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# ---------------------------------------------------------------------------
# Tooling — pinned, install if missing.
# ---------------------------------------------------------------------------
GOBIN_DIR   := $(shell go env GOPATH)/bin
GOLANGCI    := $(GOBIN_DIR)/golangci-lint
GOFUMPT     := $(GOBIN_DIR)/gofumpt
GOIMPORTS   := $(GOBIN_DIR)/goimports
GOVULNCHECK := $(GOBIN_DIR)/govulncheck
OAPICODEGEN := $(GOBIN_DIR)/oapi-codegen

# ---------------------------------------------------------------------------
# Phony declarations (alphabetical).
# ---------------------------------------------------------------------------
.PHONY: all bench boundary-guard build ci clean contracts-guard cover fmt gen help \
        install lint lint-fix security setup smoke test test-10x test-race tidy version vet

# ---------------------------------------------------------------------------
# Default target.
# ---------------------------------------------------------------------------
all: lint test build ## Default — lint, test, build.

# ---------------------------------------------------------------------------
# Build (verify the library compiles).
# ---------------------------------------------------------------------------
build: ## Build the library.
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" ./...

# ---------------------------------------------------------------------------
# Tests.
# ---------------------------------------------------------------------------
test: ## Run unit tests.
	go test ./... -count=1 -timeout=120s

test-race: ## Run unit tests with the race detector.
	go test ./... -race -count=1 -timeout=180s

test-10x: ## Run tests 10 times to surface flakes.
	go test ./... -race -count=10 -timeout=600s

cover: ## Generate a coverage report (coverage.out + coverage.html).
	go test ./... -race -coverprofile=coverage.out -covermode=atomic -timeout=180s
	@go tool cover -func=coverage.out | grep "^total:"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run benchmarks.
	go test ./... -bench=. -benchmem -count=3 -timeout=300s

# ---------------------------------------------------------------------------
# Quality gates.
# ---------------------------------------------------------------------------
fmt: ## Format source files (gofumpt + goimports).
	@command -v $(GOFUMPT)   >/dev/null 2>&1 || (echo "install: go install mvdan.cc/gofumpt@latest"   && exit 1)
	@command -v $(GOIMPORTS) >/dev/null 2>&1 || (echo "install: go install golang.org/x/tools/cmd/goimports@latest" && exit 1)
	$(GOFUMPT) -w .
	$(GOIMPORTS) -w .

vet: ## Run go vet.
	go vet ./...

boundary-guard: ## Fail if the SDK imports support engines or Hawk private packages.
	bash ./scripts/check-ecosystem-boundaries.sh

ecossystem-guard: ## Fail if external ecosystem repos import hawk/internal or removed hawk/shared/types.
	bash ./scripts/check-shared-types-imports.sh

lint: ## Run golangci-lint.
	@command -v $(GOLANGCI) >/dev/null 2>&1 || (echo "install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	$(GOLANGCI) run ./... --timeout=5m

lint-fix: ## Run golangci-lint with --fix.
	@command -v $(GOLANGCI) >/dev/null 2>&1 || (echo "install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	$(GOLANGCI) run ./... --fix --timeout=5m

security: ## Run govulncheck.
	@command -v $(GOVULNCHECK) >/dev/null 2>&1 || (echo "install: go install golang.org/x/vuln/cmd/govulncheck@latest" && exit 1)
	$(GOVULNCHECK) ./...

tidy: ## Tidy go.mod / go.sum.
	go mod tidy
	go mod verify

# ---------------------------------------------------------------------------
# Composite gate used by CI and pre-push.
# ---------------------------------------------------------------------------
ci: tidy fmt vet boundary-guard lint test-race security ## Run everything CI runs.
	@echo "All CI checks passed."

# ---------------------------------------------------------------------------
# Misc.
# ---------------------------------------------------------------------------
version: ## Print the version that will be embedded.
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

gen: ## Generate code from the OpenAPI spec (requires oapi-codegen).
	go generate ./internal/spec/
	@echo "Code generation complete."

clean: ## Remove build artefacts.
	rm -rf bin/ dist/ coverage.out coverage.html
	go clean -testcache

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: hooks
hooks:
	@command -v lefthook >/dev/null 2>&1 || (echo "install: go install github.com/evilmartians/lefthook@latest" && exit 1)
	lefthook install

setup: ## Set up local development environment (tooling, git hooks).
	@command -v $(GOFUMPT)   >/dev/null 2>&1 || go install mvdan.cc/gofumpt@latest || echo "  ⚠ Could not install gofumpt"
	@command -v $(GOIMPORTS) >/dev/null 2>&1 || go install golang.org/x/tools/cmd/goimports@latest || echo "  ⚠ Could not install goimports"
	@command -v $(GOLANGCI)  >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest || echo "  ⚠ Could not install golangci-lint"
	@command -v $(GOVULNCHECK) >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest || echo "  ⚠ Could not install govulncheck"
	@echo "✓ All tools installed"
	@echo "✓ Setup complete! Run 'make ci' to verify everything works."

smoke: ## Quick build + test verification.
	@echo "Running smoke checks..."
	$(MAKE) build
	$(MAKE) test-race
	@echo "Smoke checks passed!"
