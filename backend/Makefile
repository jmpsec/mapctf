export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

SHELL := /bin/bash


API_DIR = cmd/api
API_NAME = mapctf-api
MAP_DIR = cmd/map
MAP_NAME = mapctf-map
API_CODE = ${API_DIR:=/*.go}
MAP_CODE = ${MAP_DIR:=/*.go}

DEST ?= /opt/mapctf

OUTPUT = bin

STATIC_ARGS = -ldflags "-linkmode external -extldflags -static"
BUILD_ARGS = -ldflags "-s -w -X main.buildCommit=$(shell git rev-parse HEAD) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: build static clean api map run_api run_map

# Build code according to caller OS and architecture
build:
	make api
	make map

# Build everything statically
static:
	make api-static
	make map-static

# Build API
api:
	go build $(BUILD_ARGS) -o $(OUTPUT)/$(API_NAME) $(API_CODE)

# Build API statically
api-static:
	go build $(BUILD_ARGS) $(STATIC_ARGS) -o $(OUTPUT)/$(API_NAME) -a $(API_CODE)

# Build map
map:
	go build $(BUILD_ARGS) -o $(OUTPUT)/$(MAP_NAME) $(MAP_CODE)

# Build map statically
map-static:
	go build $(BUILD_ARGS) $(STATIC_ARGS) -o $(OUTPUT)/$(MAP_NAME) -a $(MAP_CODE)

# Delete all compiled binaries
clean:
	rm -rf $(OUTPUT)/$(API_NAME)
	rm -rf $(OUTPUT)/$(MAP_NAME)

# Run API server locally
run_api:
	go run $(API_CODE)

# Run map server locally
run_map:
	go run $(MAP_CODE)

# Remove all unused dependencies
tidy:
	make clean
	go mod tidy

# Keep dependencies up to date
deps-update:
ifeq (,$(wildcard go.mod))
	$(error Missing go.mod file)
endif
	go get -u ./...

# Run linter
lint:
	golangci-lint run

# Test with coverage
test:
	go test ./pkg/version -v -cover
	go test ./pkg/users -v -cover
	go test ./pkg/teams -v -cover
	go test ./pkg/cache -v -cover
	go test ./pkg/backend -v -cover
	go test ./pkg/challenges -v -cover
