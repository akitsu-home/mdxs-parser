APP_NAME ?= mdxs-parser
PKG ?= github.com/owner/mdxs-parser
VERSION ?= $(shell git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT) -X $(PKG)/internal/version.Date=$(DATE)
BUILD_DIR ?= dist
BIN ?= $(BUILD_DIR)/$(APP_NAME)

.PHONY: test vet fmt lint build snapshot release-dry-run check-goreleaser clean

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

lint: fmt vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; skipped"; \
	fi

build:
	mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/$(APP_NAME)

snapshot:
	goreleaser release --snapshot --clean

release-dry-run:
	goreleaser release --skip=publish --clean

check-goreleaser:
	goreleaser check

clean:
	rm -rf $(BUILD_DIR)
