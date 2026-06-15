BINARY := yore
PKG := github.com/Dev-Bilaspure/yore
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X '$(PKG)/internal/cli.version=$(VERSION)' \
	-X '$(PKG)/internal/cli.commit=$(COMMIT)' \
	-X '$(PKG)/internal/cli.date=$(DATE)'

.PHONY: all build install test race cover vet fmt lint check clean

all: check build

build: ## Build the yore binary into ./bin
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/yore

install: ## Install yore into $GOBIN / $GOPATH/bin
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/yore

test: ## Run tests
	go test ./...

race: ## Run tests with the race detector
	go test -race ./...

cover: ## Run tests and print coverage
	go test -cover ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -w .

lint: ## Run golangci-lint (if installed)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

check: fmt vet race lint ## Format, vet, race-test and lint

clean: ## Remove build artifacts
	rm -rf bin dist
