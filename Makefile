APP_NAME           := macos-menu
APP_VERSION 	   := v0.1.0
YAML_FILES         := $(shell find . -type f \( -iname "*.yml" -o -iname "*.yaml" \))

# Go 
GO111MODULE     := on
CGO_ENABLED	    := 0

# Environment for Go commands
GO_ENV := \
	GO111MODULE=$(GO111MODULE) \
	CGO_ENABLED=$(CGO_ENABLED)

.PHONY: all build lint clean test help tidy upgrade tag pre bench vet fmt run

all: help

pre: tidy lint test vet ## Run all quality checks

build: ## Build the Go binary locally
	$(GO_ENV) go build -v -o bin/$(APP_NAME) main.go

clean: ## Clean the build artifacts
	$(GO_ENV) go clean -x; \
	rm -f bin/$(APP_NAME)

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

run: ## Run the application
	$(GO_ENV) go run main.go

test: ## Run Go tests and generate coverage report
	$(GO_ENV) go test -count=1 -covermode=atomic -coverprofile=coverage.out ./... || exit 1; \
	echo "Generating coverage report..."; \
	$(GO_ENV) go tool cover -func=coverage.out

vet: ## Vet the Go code
	$(GO_ENV) go vet ./...

tag: ## Creates a release tag
	git tag -s -m "version bump to $(APP_VERSION)" $(APP_VERSION); \
	git push origin $(APP_VERSION)

help: ## Displays available commands
	@echo "Available make targets:"; \
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

