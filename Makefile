BIN := ios-app-manager
CMD := ./cmd/ios-app-manager
CACHE_DIR := $(CURDIR)/.cache

export GOCACHE := $(CACHE_DIR)/go-build
export GOMODCACHE := $(CACHE_DIR)/gomod
export GOPATH := $(CACHE_DIR)/gopath

.PHONY: setup prepare-cache test test-update build lint

setup: prepare-cache

prepare-cache:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)"

test: prepare-cache
	go test ./...

test-update: prepare-cache
	go test ./internal/testutil -update
	go test ./...

build: prepare-cache
	go build -o $(BIN) $(CMD)

lint: prepare-cache
	go vet ./...
