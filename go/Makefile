.PHONY: all build build-go build-ts build-extensions clean run test

# Paths
REPO_ROOT := $(shell pwd)/..
BUILD_OUT := $(REPO_ROOT)/out
NODE_BIN := $(shell nvm_path 2>/dev/null) || echo "$$(HOME)/.nvm/versions/node/v24.18.0/bin/node"

# Go
GODE_BIN := $(REPO_ROOT)/build/gode

all: build

## build-ts: Compile TypeScript extension host and core
build-ts:
	cd $(REPO_ROOT) && node ./node_modules/gulp/bin/gulp.js compile

## build-go: Compile Go binary
build-go:
	cd $(dir $(GODE_BIN)) && go build -o $(GODE_BIN) .

## build: Build everything (TS first, then Go)
build: build-ts build-go

## run: Build and run gode
run: build
	$(GODE_BIN) --user-data-dir=$(HOME)/.gode

## test: Run Go tests
test:
	 cd $(dir $(GODE_BIN)) && go test ./...

## clean: Clean build artifacts
clean:
	rm -f $(GODE_BIN)
	cd $(REPO_ROOT) && rm -rf out-vscode

help:
	@echo "Gode Build System"
	@echo "  make build          - Build TS extension host + Go binary"
	@echo "  make build-go       - Build Go binary only"
	@echo "  make build-ts       - Build TS extension host only"
	@echo "  make run            - Build and run gode"
	@echo "  make test           - Run Go tests"
	@echo "  make clean          - Clean build artifacts"
