# Xinkgo — the harness.
#
# Every command a developer needs is here; scripts/ holds the ones with
# logic in them. `make help` lists them all.
#
# Windows users without make run the same commands by hand: each target
# prints what it runs, and docs/HARNESS.md spells them out.

GO ?= go
export GOTOOLCHAIN ?= auto

APP      := xinkgo
CMD      := ./cmd/$(APP)
BIN      := bin
# The platforms `make package` builds, and the ones `make cross` compiles.
PLATFORMS ?= windows/amd64
CROSS_PLATFORMS ?= windows/amd64 windows/arm64 darwin/arm64 darwin/amd64 linux/amd64

.PHONY: help run dev build build-windows package cross test test-race \
	vet fmt fmt-check lint doctor check debug clean

help: ## List the commands
	@grep -hE '^[a-z][a-z0-9-]*:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk -F':.*?## ' '{printf "  make %-14s %s\n", $$1, $$2}'

## Run

run: ## Run the app from the sources
	$(GO) run $(CMD)

dev: ## Run the app with live reload: it rebuilds when a .go file changes
	$(GO) tool mygo dev

doctor: ## Check that this machine has what MyGo needs
	$(GO) tool mygo doctor

## Build

build: ## Build the app for this machine into $(BIN)/
	$(GO) build -o $(BIN)/ $(CMD)

build-windows: ## Cross-compile the Windows executable (no packaging)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(BIN)/windows-amd64/ $(CMD)

package: ## Package installable apps: .exe on Windows (PLATFORMS="windows/amd64 windows/arm64")
	scripts/package.sh $(PLATFORMS)

cross: ## Compile for every platform we ship, without packaging
	@for p in $(CROSS_PLATFORMS); do \
		GOOS=$${p%/*} GOARCH=$${p#*/} CGO_ENABLED=0 $(GO) build ./... && \
			echo "  ok  $$p" || exit 1; \
	done

## Test

test: ## Run the tests
	$(GO) test ./...

test-race: ## Run the tests with the race detector
	$(GO) test -race ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format the code
	gofmt -w .

fmt-check: ## Fail on code gofmt would change
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then \
		echo "not formatted:"; echo "$$out"; echo "run: make fmt"; exit 1; \
	fi

lint: ## Run golangci-lint, when it is installed
	@command -v golangci-lint >/dev/null 2>&1 && \
		golangci-lint run ./... || echo "golangci-lint is not installed; see docs/CONTRIBUTING.md"

check: ## Everything a change passes before it is committed
	scripts/check.sh

## Debug

debug: ## Run the app under the Delve debugger
	dlv debug $(CMD)

## Housekeeping

clean: ## Remove what builds produce
	rm -rf $(BIN) build .mygo
