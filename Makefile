SHELL := /bin/bash

APP_NAME := spartan
BIN_DIR := bin
WEB_DIR := web
DATA_DIR := .data

# Installation directory (respects XDG_BIN_HOME if set, otherwise ~/.local/bin)
XDG_BIN_HOME ?= $(HOME)/.local/bin
INSTALL_DIR ?= $(XDG_BIN_HOME)

# Build-time variables
VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Version=$(VERSION) \
           -X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Commit=$(COMMIT) \
           -X github.com/fitchmultz/spartan-scraper/internal/buildinfo.Date=$(DATE)"

.PHONY: install update lint type-check format clean test test-ci generate build ci web-dev

install:
	go mod download
	cd $(WEB_DIR) && pnpm install

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
	rm -rf node_modules $(WEB_DIR)/node_modules
	cd $(WEB_DIR) && rm -rf dist
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
	cd $(WEB_DIR) && CI=1 pnpm run test

generate:
	cd $(WEB_DIR) && pnpm exec openapi-ts -i ../api/openapi.yaml -o src/api
	node $(CURDIR)/scripts/strip_openapi_todos.mjs --path $(WEB_DIR)/src/api

build:
	mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	cd $(WEB_DIR) && pnpm run build
	@echo "Installing $(APP_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(APP_NAME)"

ci: install generate format type-check lint build test-ci

ci-slow: build
	./scripts/stress_test.sh
	go test -v ./internal/e2e/...

web-dev:
	cd $(WEB_DIR) && pnpm run dev
