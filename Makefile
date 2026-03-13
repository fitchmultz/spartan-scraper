SHELL := /bin/bash

APP_NAME := spartan
BIN_DIR := bin
WEB_DIR := web
PI_BRIDGE_DIR := tools/pi-bridge
DATA_DIR := .data

# Installation directory (respects XDG_BIN_HOME if set, otherwise ~/.local/bin)
XDG_BIN_HOME ?= $(HOME)/.local/bin
INSTALL_DIR ?= $(XDG_BIN_HOME)

# Build-time variables
VERSION ?= 1.0.0-rc1
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Version=$(VERSION) \
           -X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Commit=$(COMMIT) \
           -X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Date=$(DATE)"

# Resource control: cap Vitest workers for deterministic CI CPU usage
CI_VITEST_MAX_WORKERS ?= 2
GITLEAKS_VERSION ?= v8.30.0
PLAYWRIGHT_GO_VERSION ?= v0.5700.1
PLAYWRIGHT_INSTALL_CMD := go run github.com/playwright-community/playwright-go/cmd/playwright@$(PLAYWRIGHT_GO_VERSION) install --with-deps

.PHONY: audit-public secret-scan install update lint type-check format clean test test-ci generate build install-bin install-playwright ci ci-pr ci-slow ci-network ci-manual verify-clean-tree web-dev

audit-public:
	node $(CURDIR)/scripts/public_audit.mjs

secret-scan:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) detect --source $(CURDIR) --log-opts="--all" --gitleaks-ignore-path $(CURDIR)/.gitleaksignore --redact --no-banner

install:
	go mod download
	cd $(PI_BRIDGE_DIR) && npm ci
	cd $(WEB_DIR) && pnpm install --frozen-lockfile

update:
	@echo "Updating Go dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Updating pnpm dependencies..."
	cd $(WEB_DIR) && pnpm update
	@echo "Dependency update complete. Review changes before committing."

lint:
	go vet ./...
	cd $(WEB_DIR) && pnpm exec biome lint .

type-check:
	cd $(WEB_DIR) && pnpm exec tsc --noEmit

format:
	gofmt -w ./cmd ./internal
	cd $(WEB_DIR) && pnpm exec biome format . --write

# clean: Remove build artifacts, dependencies, and temporary files
# Removes: bin/, .data/, node_modules/ (root + web), dist/, installed binary
# Also removes: Go test binaries (*.test), coverage files (*.out), out/stress (log artifacts)
# Preserves: Source-controlled lock files (go.sum, web/pnpm-lock.yaml)
clean:
	rm -rf $(BIN_DIR) $(DATA_DIR)
	rm -rf node_modules $(WEB_DIR)/node_modules $(PI_BRIDGE_DIR)/node_modules
	cd $(WEB_DIR) && rm -rf dist
	cd $(PI_BRIDGE_DIR) && rm -rf dist
	rm -f $(INSTALL_DIR)/$(APP_NAME)
	find . -type f -name "*.test" -delete
	find . -type f -name "*.out" -delete
	rm -rf out/stress
	rm -f $(WEB_DIR)/openapi-ts-error-*.log

test:
	CI=1 go test ./... -p=1 -timeout 5m

test-ci:
	CI=1 go test $$(go list ./... | grep -v /e2e) -p=1 -timeout 5m
	node $(CURDIR)/scripts/strip_openapi_todos.test.mjs
	node $(CURDIR)/scripts/public_audit.test.mjs
	cd $(PI_BRIDGE_DIR) && npm test
	cd $(WEB_DIR) && CI=1 NODE_OPTIONS=--localstorage-file=.vitest-localstorage pnpm run test -- --run --maxWorkers=$(CI_VITEST_MAX_WORKERS)

generate:
	cd $(WEB_DIR) && pnpm exec openapi-ts -i ../api/openapi.yaml -o src/api
	node $(CURDIR)/scripts/strip_openapi_todos.mjs --path $(WEB_DIR)/src/api

build:
	mkdir -p $(BIN_DIR)
	cd $(PI_BRIDGE_DIR) && npm run build
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	cd $(WEB_DIR) && pnpm run build

install-bin: build
	@echo "Installing $(APP_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(APP_NAME)"

install-playwright:
	$(PLAYWRIGHT_INSTALL_CMD)

# Full local CI profile (developer-friendly; no clean-tree precondition)
ci: audit-public install generate format type-check lint build test-ci

verify-clean-tree:
	@if ! git diff --quiet --ignore-submodules --; then \
		echo "verify-clean-tree: unstaged changes detected"; \
		git status --short; \
		exit 1; \
	fi
	@if ! git diff --cached --quiet --ignore-submodules --; then \
		echo "verify-clean-tree: staged changes detected"; \
		git status --short; \
		exit 1; \
	fi
	@if [ -n "$$(git ls-files --others --exclude-standard)" ]; then \
		echo "verify-clean-tree: untracked files detected"; \
		git status --short; \
		exit 1; \
	fi

# PR-equivalent deterministic gate (must run from clean git state)
ci-pr: verify-clean-tree audit-public install generate format type-check lint build test-ci verify-clean-tree

ci-slow: install install-playwright build
	./scripts/stress_test.sh
	go test -v ./internal/e2e/...

ci-network: install build
	./scripts/stress_test.sh --network

ci-manual: ci-slow ci-network

web-dev:
	cd $(WEB_DIR) && pnpm run dev
