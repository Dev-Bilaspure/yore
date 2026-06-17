BINARY := yore
PKG := github.com/Dev-Bilaspure/yore
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X '$(PKG)/internal/cli.version=$(VERSION)' \
	-X '$(PKG)/internal/cli.commit=$(COMMIT)' \
	-X '$(PKG)/internal/cli.date=$(DATE)'

.PHONY: all build install test race cover fuzz vet fmt lint check dev dev-fresh demo help clean

all: check build

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort | awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

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

fuzz: ## Fuzz the history parsers for ~10s each (no panics)
	go test -run=x -fuzz=FuzzZshParse -fuzztime=10s ./pkg/history
	go test -run=x -fuzz=FuzzBashParse -fuzztime=10s ./pkg/history

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files
	gofmt -w .

lint: ## Run golangci-lint (if installed)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

check: fmt vet race lint ## Format, vet, race-test and lint (what CI runs)

dev: ## Sandbox shell, seeded + recording on (manual testing; touches nothing real)
	@./scripts/dev-shell.sh --seed --enable

dev-fresh: ## Sandbox shell from a clean slate (to walk the onboarding journey)
	@./scripts/dev-shell.sh

demo: ## Render the demo GIF (requires vhs + fzf)
	ZDOTDIR=$$(mktemp -d) vhs docs/demo.tape

clean: ## Remove build artifacts
	rm -rf bin dist
