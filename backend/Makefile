export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

SHELL := /bin/bash

API_DIR = cmd/api
API_NAME = mapctf-api
API_CODE = ${API_DIR:=/*.go}

DEST ?= /opt/mapctf

OUTPUT = bin

STATIC_ARGS = -ldflags "-linkmode external -extldflags -static"
BUILD_ARGS = -ldflags "-s -w -X main.buildCommit=$(shell git rev-parse HEAD) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: build static clean api run

# Build code according to caller OS and architecture
build:
	make api

# Build everything statically
static:
	make api-static

# Build API
api:
	go build $(BUILD_ARGS) -o $(OUTPUT)/$(API_NAME) $(API_CODE)

# Build API statically
api-static:
	go build $(BUILD_ARGS) $(STATIC_ARGS) -o $(OUTPUT)/$(API_NAME) -a $(API_CODE)

# Delete all compiled binaries
clean:
	rm -rf $(OUTPUT)/$(API_NAME)

# Run API server locally
run:
	go run $(API_CODE)

# Remove all unused dependencies
tidy:
	make clean
	go mod tidy

# Run linter
lint:
	golangci-lint run
