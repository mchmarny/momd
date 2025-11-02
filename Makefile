APP_NAME           := momd
APP_VERSION 	   := v0.1.0
YAML_FILES         := $(shell find . -type f \( -iname "*.yml" -o -iname "*.yaml" \))

# Go 
GO111MODULE     := on
CGO_ENABLED	    := 0

# Environment for Go commands
GO_ENV := \
	GO111MODULE=$(GO111MODULE) \
	CGO_ENABLED=$(CGO_ENABLED)

.PHONY: all build lint clean test help tidy upgrade tag pre bench vet fmt server app run

all: help

pre: tidy lint test vet ## Run all quality checks

build: ## Build the Go binary locally
	$(GO_ENV) go build -v -o bin/$(APP_NAME) cmd/momd/main.go

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GO_ENV) go fmt ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO_ENV) go test ./pkg/... -bench=. -benchmem

tidy: ## Run go mod tidy in src
	$(GO_ENV) go fmt ./...; \
	$(GO_ENV) go mod tidy

upgrade: ## Upgrades all dependencies
	$(GO_ENV) go get -u ./...; \
	$(GO_ENV) go mod tidy;

lint: ## Lint the Go code and YAML files
	$(GO_ENV) golangci-lint -c .golangci.yaml run --modules-download-mode=readonly
	@if command -v yamllint >/dev/null 2>&1; then \
		yamllint $(YAML_FILES); \
	else \
		echo "yamllint not installed, skipping YAML linting"; \
	fi

server: ## Run the Go server
	$(GO_ENV) go run cmd/momd/main.go

test: ## Run Go tests and generate coverage report
	$(GO_ENV) go test -count=1 -covermode=atomic -coverprofile=coverage.out ./... || exit 1; \
	echo "Generating coverage report..."; \
	$(GO_ENV) go tool cover -func=coverage.out

vet: ## Vet the Go code
	$(GO_ENV) go vet ./...

tag: ## Creates a release tag
	git tag -s -m "version bump to $(APP_VERSION)" $(APP_VERSION); \
	git push origin $(APP_VERSION)

app: build ## Build the macOS menu bar app
	@echo "Building macOS app..."
	@mkdir -p macos/build/momd.app/Contents/MacOS
	@mkdir -p macos/build/momd.app/Contents/Resources
	
	# Compile Swift code
	swiftc -o macos/build/momd.app/Contents/MacOS/momd \
		-framework Cocoa \
		-framework Foundation \
		macos/momd/main.swift \
		macos/momd/AppDelegate.swift
	
	# Copy Info.plist
	cp macos/momd/Info.plist macos/build/momd.app/Contents/Info.plist
	
	# Copy the Go binary to Resources
	cp bin/momd macos/build/momd.app/Contents/Resources/momd
	
	@echo "macOS app built at: macos/build/momd.app"
	@echo "To run: open macos/build/momd.app"

clean: ## Clean the macOS and Go artifacts
	$(GO_ENV) go clean -x; \
	rm -f bin/$(APP_NAME); \
	rm -rf macos/build

run: app ## Build and run the macOS app
	open macos/build/momd.app

help: ## Displays available commands
	@echo "Available make targets:"; \
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

