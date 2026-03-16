# Purpose: Canonical local build, test, release, and CI entrypoints for Spartan Scraper.
# Responsibilities: Define reproducible developer workflows, enforce the repo toolchain contract, and centralize shared build/test commands.
# Scope: Repository-local automation for Go, web, and pi-bridge tasks that must run the same way locally and in CI.
# Usage: Invoke targets with `make <target>` (for example `make ci`, `make build`, `make verify-toolchain`).
# Invariants/Assumptions: `.tool-versions` is the authoritative toolchain contract; PATH must resolve the exact pinned Go, Node, and pnpm versions.
SHELL := /bin/bash

APP_NAME := spartan
BIN_DIR := bin
WEB_DIR := web
PI_BRIDGE_DIR := tools/pi-bridge
DATA_DIR := .data
TOOLCHAIN_FILE := .tool-versions

GO_VERSION := $(strip $(shell awk '$$1=="golang" { print $$2 }' $(TOOLCHAIN_FILE)))
NODE_VERSION := $(strip $(shell awk '$$1=="nodejs" { print $$2 }' $(TOOLCHAIN_FILE)))
PNPM_VERSION := $(strip $(shell awk '$$1=="pnpm" { print $$2 }' $(TOOLCHAIN_FILE)))

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
GITLEAKS_VERSION ?= v8.30.1
PLAYWRIGHT_GO_VERSION ?= v0.5700.1
PLAYWRIGHT_INSTALL_CMD := go run github.com/playwright-community/playwright-go/cmd/playwright@$(PLAYWRIGHT_GO_VERSION) install --with-deps

.PHONY: audit-public secret-scan install update lint type-check format clean test test-ci generate build install-bin install-playwright ci ci-pr ci-slow ci-network ci-manual verify-clean-tree verify-toolchain web-dev

verify-toolchain:
	@set -euo pipefail; \
	toolchain_file="$(CURDIR)/$(TOOLCHAIN_FILE)"; \
	expected_go="$(GO_VERSION)"; \
	expected_node="$(NODE_VERSION)"; \
	expected_pnpm="$(PNPM_VERSION)"; \
	if [[ -z "$$expected_go" || -z "$$expected_node" || -z "$$expected_pnpm" ]]; then \
		echo "verify-toolchain: failed to parse $$toolchain_file" >&2; \
		exit 1; \
	fi; \
	for tool in go node pnpm; do \
		if ! command -v $$tool >/dev/null 2>&1; then \
			echo "verify-toolchain: required tool '$$tool' is not on PATH" >&2; \
			exit 1; \
		fi; \
	done; \
	actual_go="$$(go env GOVERSION | sed 's/^go//')"; \
	actual_node="$$(node --version | sed 's/^v//')"; \
	actual_pnpm="$$(pnpm --version)"; \
	echo "Toolchain contract ($$toolchain_file)"; \
	echo "  Go:   expected $$expected_go | actual $$actual_go"; \
	echo "  Node: expected $$expected_node | actual $$actual_node"; \
	echo "  pnpm: expected $$expected_pnpm | actual $$actual_pnpm"; \
	status=0; \
	if [[ "$$actual_go" != "$$expected_go" ]]; then \
		echo "verify-toolchain: Go version mismatch" >&2; \
		status=1; \
	fi; \
	if [[ "$$actual_node" != "$$expected_node" ]]; then \
		echo "verify-toolchain: Node version mismatch" >&2; \
		status=1; \
	fi; \
	if [[ "$$actual_pnpm" != "$$expected_pnpm" ]]; then \
		echo "verify-toolchain: pnpm version mismatch" >&2; \
		status=1; \
	fi; \
	exit $$status

audit-public: verify-toolchain
	node $(CURDIR)/scripts/public_audit.mjs

secret-scan: verify-toolchain
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) detect --source $(CURDIR) --log-opts="--all" --gitleaks-ignore-path $(CURDIR)/.gitleaksignore --redact --no-banner

install: verify-toolchain
	go mod download
	cd $(PI_BRIDGE_DIR) && pnpm install --frozen-lockfile
	cd $(WEB_DIR) && pnpm install --frozen-lockfile

update: verify-toolchain
	@echo "Updating Go dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Updating pnpm dependencies in $(PI_BRIDGE_DIR)..."
	cd $(PI_BRIDGE_DIR) && pnpm update --latest
	@echo "Updating pnpm dependencies in $(WEB_DIR)..."
	cd $(WEB_DIR) && pnpm update --latest
	@echo "Dependency update complete. Review changes before committing."

lint: verify-toolchain
	go vet ./...
	cd $(WEB_DIR) && pnpm exec biome lint .

type-check: verify-toolchain
	cd $(WEB_DIR) && pnpm exec tsc --noEmit

format: verify-toolchain
	gofmt -w ./cmd ./internal
	cd $(WEB_DIR) && pnpm exec biome format . --write

# clean: Remove build artifacts, dependencies, and temporary files
# Removes: bin/, .data/, node_modules/ (root + web), dist/, installed binary
# Also removes: Go test binaries (*.test), coverage files (*.out), out/stress (log artifacts)
# Preserves: Source-controlled lock files (go.sum, web/pnpm-lock.yaml, tools/pi-bridge/pnpm-lock.yaml)
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

test: verify-toolchain
	CI=1 go test ./... -p=1 -timeout 5m

test-ci: verify-toolchain
	CI=1 go test $$(go list ./... | grep -v /e2e) -p=1 -timeout 5m
	node $(CURDIR)/scripts/strip_openapi_todos.test.mjs
	node $(CURDIR)/scripts/public_audit.test.mjs
	cd $(PI_BRIDGE_DIR) && pnpm test
	cd $(WEB_DIR) && CI=1 NODE_OPTIONS=--localstorage-file=.vitest-localstorage pnpm run test -- --run --maxWorkers=$(CI_VITEST_MAX_WORKERS)

generate: verify-toolchain
	cd $(WEB_DIR) && pnpm exec openapi-ts -i ../api/openapi.yaml -o src/api
	node $(CURDIR)/scripts/strip_openapi_todos.mjs --path $(WEB_DIR)/src/api

build: verify-toolchain
	mkdir -p $(BIN_DIR)
	cd $(PI_BRIDGE_DIR) && pnpm run build
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	cd $(WEB_DIR) && pnpm run build

install-bin: build
	@echo "Installing $(APP_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(APP_NAME)"

install-playwright: verify-toolchain
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

web-dev: verify-toolchain
	cd $(WEB_DIR) && pnpm run dev
