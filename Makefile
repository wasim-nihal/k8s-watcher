# Build variables
BINARY_NAME=k8s-watcher
BUILD_DIR=build
MAIN_PATH=cmd/k8s-watcher/main.go

# Go variables
GO=go
GOFLAGS=-v
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Testing variables
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
TEST_FLAGS=-race -covermode=atomic -coverprofile=$(COVERAGE_FILE)

.PHONY: all build clean test coverage lint vet fmt

all: clean build test

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)

test:
	@echo "Running tests..."
	$(GO) test ./... -v

coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test ./... $(TEST_FLAGS)
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_DIR)/coverage.html

lint:
	@echo "Running linter..."
	golangci-lint run

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Cross compilation targets
.PHONY: build-linux build-darwin build-windows

build-linux:
	GOOS=linux GOARCH=amd64 $(MAKE) build

build-darwin:
	GOOS=darwin GOARCH=amd64 $(MAKE) build

build-windows:
	GOOS=windows GOARCH=amd64 $(MAKE) build

# Docker targets
.PHONY: docker-build docker-push

docker-build:
	docker build -t $(BINARY_NAME):latest .

docker-push:
	docker push $(BINARY_NAME):latest
