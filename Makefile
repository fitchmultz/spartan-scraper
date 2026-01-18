SHELL := /bin/bash

APP_NAME := spartan
BIN_DIR := bin
WEB_DIR := web
DATA_DIR := .data

.PHONY: install update lint type-check format clean test generate build ci web-dev

install:
	go mod download
	cd $(WEB_DIR) && pnpm install

update:
	@echo "Dependency updates are manual. Use: cd $(WEB_DIR) && pnpm up -L && go get -u ./..."

lint:
	go vet ./...
	cd $(WEB_DIR) && pnpm exec biome lint .

type-check:
	cd $(WEB_DIR) && pnpm exec tsc --noEmit

format:
	gofmt -w ./cmd ./internal
	cd $(WEB_DIR) && pnpm exec biome format . --write

clean:
	rm -rf $(BIN_DIR) $(DATA_DIR)
	cd $(WEB_DIR) && rm -rf dist node_modules

test:
	go test ./...

generate:
	cd $(WEB_DIR) && pnpm exec openapi-ts -i ../api/openapi.yaml -o src/api

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	cd $(WEB_DIR) && pnpm run build

ci: generate format type-check lint build test

web-dev:
	cd $(WEB_DIR) && pnpm run dev
