# NewAPI Tools V3.0 Go Build

BINARY_NAME  ?= newapi
BUILD_DIR    ?= ./dist
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v3.0.0-dev)
GIT_COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE   ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS      := -X github.com/Bonus520/newapi-tools/internal/core.Version=$(VERSION) \
                -X github.com/Bonus520/newapi-tools/internal/core.GitCommit=$(GIT_COMMIT) \
                -X github.com/Bonus520/newapi-tools/internal/core.BuildDate=$(BUILD_DATE)

.PHONY: build clean test run lint fmt vet coverage install release snapshot check i18n-extract docs

## build: Build the binary with version info
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/newapi/

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## test: Run all tests
test:
	go test ./...

## run: Run the binary locally
run:
	go run ./cmd/newapi/

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	gofmt -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## coverage: Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## install: Install the binary to GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/newapi/

## release: Create a release using goreleaser
release:
	goreleaser release --clean

## snapshot: Create a snapshot build (no tag required)
snapshot:
	goreleaser release --snapshot --clean

## check: Run fmt + vet + test
check: vet test

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

## i18n-extract: Extract all i18n.T() keys from source code
i18n-extract:
	@echo "Extracting i18n keys from source..."
	@grep -rn 'i18n\.T(' internal/ | sed 's/.*i18n\.T("\([^"]*\)".*/\1/' | sort -u

## docs: Generate error code documentation
docs:
	@echo "Generating error code documentation..."
	@go run cmd/gendocs/main.go
