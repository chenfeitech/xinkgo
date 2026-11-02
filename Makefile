# Xinkgo — the harness.
#
# macOS is the primary target: it is the machine we develop on, so it is
# built first, it is what the commands default to, and the gate checks it
# before anything else. Windows and Linux stay supported, and are built
# and packaged from here too, in that order of priority:
#
#   P0  darwin/arm64, darwin/amd64      built, packaged, gated first
#   P1  windows/amd64                   cross-compiled and packaged
#   P2  windows/arm64, linux/amd64      cross-compiled (CI smoke)
#
# scripts/ holds the commands with logic in them; `make help` lists them.
# Windows users without make run the same commands by hand: each target
# prints what it runs, and docs/HARNESS.md spells them out.

GO ?= go
export GOTOOLCHAIN ?= auto

APP := xinkgo
CMD := ./cmd/$(APP)
BIN := bin

# The machine we are on, as Go names it: darwin/arm64 on Apple silicon.
HOST_PLATFORM := $(shell uname -s | tr '[:upper:]' '[:lower:]')/$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# What `make package` builds: the machine's own platform, which on a Mac
# is macOS. Override to package another: PLATFORMS="windows/amd64".
PLATFORMS ?= $(HOST_PLATFORM)
# P0: macOS, both architectures. "darwin/universal" builds one app of both.
MACOS_PLATFORMS ?= darwin/arm64 darwin/amd64
# P1/P2: Windows, then Linux.
WINDOWS_PLATFORMS ?= windows/amd64 windows/arm64
ALL_PLATFORMS ?= $(MACOS_PLATFORMS) $(WINDOWS_PLATFORMS) linux/amd64

.PHONY: help deps run dev build build-macos build-windows package \
	package-macos icon cross test test-race vet fmt fmt-check lint doctor \
	check debug clean

help: ## List the commands
	@grep -hE '^[a-z][a-z0-9-]*:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk -F':.*?## ' '{printf "  make %-14s %s\n", $$1, $$2}'

## Dependencies

deps: ## Download and verify the dependencies, and the pinned MyGo CLI
	$(GO) mod download
	$(GO) mod verify
	$(GO) tool mygo version

## Run

run: deps ## Run the app from the sources
	$(GO) run $(CMD)

dev: deps ## Run the app with live reload: it rebuilds when a .go file changes
	$(GO) tool mygo dev

doctor: deps ## Check that this machine has what MyGo needs
	$(GO) tool mygo doctor

## Build

build: deps ## Build for this machine into $(BIN)/<platform>/
	$(GO) build -o $(BIN)/$(subst /,-,$(HOST_PLATFORM))/ $(CMD)

build-macos: deps ## Build macOS, the primary target (MACOS_PLATFORMS="darwin/universal" for one app of both)
	@for p in $(MACOS_PLATFORMS); do \
		GOOS=$${p%/*} GOARCH=$${p#*/} CGO_ENABLED=0 $(GO) build \
			-o $(BIN)/$${p%/*}-$${p#*/}/ $(CMD) && echo "  ok  $$p" || exit 1; \
	done

build-windows: deps ## Cross-compile the Windows executables
	@for p in $(WINDOWS_PLATFORMS); do \
		GOOS=$${p%/*} GOARCH=$${p#*/} CGO_ENABLED=0 $(GO) build \
			-o $(BIN)/$${p%/*}-$${p#*/}/$(APP).exe $(CMD) && \
			echo "  ok  $$p" || exit 1; \
	done

## Package

package: deps ## Package the app for this machine: .app and .dmg on macOS (PLATFORMS="windows/amd64" for another)
	scripts/package.sh $(PLATFORMS)

package-macos: deps ## Package macOS, the primary target: .app and .dmg for both architectures
	scripts/package.sh $(MACOS_PLATFORMS)

icon: ## Write resources/icon.png from resources/xink.icns, on a Mac
	scripts/icon.sh

cross: deps ## Compile every platform we ship, macOS first, without packaging
	@for p in $(ALL_PLATFORMS); do \
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

check: ## Everything a change passes before it is committed: macOS first
	scripts/check.sh

## Debug

debug: ## Run the app under the Delve debugger
	dlv debug $(CMD)

## Housekeeping

clean: ## Remove what builds produce
	rm -rf $(BIN) build .mygo
