# Makefile for Ralph Go CLI

.PHONY: all build clean install test test-e2e test-coverage coverage cobertura deps help quality format lint security arch run

BINARY_NAME=ralph
GO=go
INSTALL_PATH=/usr/local/bin
BUILD_OUT ?= bin/ralph
ARGS ?=

# Default target
all: build

# Common targets help
help:
	@printf "%s\n" \
	"Common targets:" \
	"  make quality      Run full quality checks" \
	"  make format       Run gofmt on tracked Go files" \
	"  make lint         Run golangci-lint" \
	"  make test         Run tests" \
	"  make test-e2e     Run end-to-end tests" \
	"  make test-race    Run tests with race detection" \
	"  make coverage     Run coverage gate only" \
	"  make mutation     Run mutation testing (final stage)" \
	"  make security     Run govulncheck and gosec" \
	"  make arch         Run go-arch-lint" \
	"  make build        Build the ralph binary" \
	"  make clean        Remove built artifacts" \
	"  make install      Install ralph to $(INSTALL_PATH)" \
	"  make deps         Download and tidy dependencies" \
	"  make run ARGS='...' Run CLI from source"

# Full quality checks
quality: test lint test-race test-coverage test-mutation security arch

# Format with gofmt
format:
	gofmt -w $$(git ls-files '*.go')

# Lint with golangci-lint
lint:
	golangci-lint run

# Run tests
test:
	$(GO) test -v ./...

# Run end-to-end tests
test-e2e:
	$(GO) test -v ./test/e2e

# Run coverage gate
coverage: test-coverage

# Coverage output file — kept between targets for CI upload
COVERAGE_OUT ?= coverage.out

# Run tests with coverage and enforce minimum threshold
test-coverage:
	@cover_status=0; \
	$(GO) test -coverprofile="$(COVERAGE_OUT)" -covermode=atomic ./... || cover_status=$$?; \
	total="$$($(GO) tool cover -func="$(COVERAGE_OUT)" | awk '/^total:/{gsub(/%/,"",$$3); print $$3}')"; \
	if awk -v total="$$total" -v minimum="90" 'BEGIN {exit !(total >= minimum)}'; then \
		exit $$cover_status; \
	else \
		echo "Coverage $${total}% is below required 90%." >&2; \
		exit 1; \
	fi

# Convert Go coverage profile to Cobertura XML for upload-code-coverage action
cobertura:
	gocover-cobertura < $(COVERAGE_OUT) > cobertura.xml

test-race:
	$(GO) test -race ./...

test-mutation:
	gremlins unleash $(ARGS)

mutation: test-mutation

# Security checks
security:
	govulncheck ./...
	gosec ./...

# Architecture checks
arch:
	go-arch-lint check

# Build the binary
build:
	$(GO) build -o $(BUILD_OUT) ./cmd/ralph

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BUILD_OUT)
	$(GO) clean

# Install the binary to system path
install: build
	install -m 0755 $(BUILD_OUT) $(INSTALL_PATH)/$(BINARY_NAME)

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Run from source
run:
	$(GO) run ./cmd/ralph $(ARGS)
