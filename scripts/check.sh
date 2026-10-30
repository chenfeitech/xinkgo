#!/usr/bin/env bash
#
# The gate a change passes before it is committed: the code is formatted,
# vet is quiet, the tests pass, and every platform we ship still compiles.
#
#	scripts/check.sh           # the whole gate
#	scripts/check.sh -fast     # tests for this machine only
#
set -euo pipefail

cd "$(dirname "$0")/.."
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

fast=0
if [ "${1:-}" = "-fast" ]; then
	fast=1
fi

step() { printf '\n== %s ==\n' "$1"; }

step "gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
	echo "$unformatted"
	echo "Run 'make fmt' (or: gofmt -w .)"
	exit 1
fi
echo "ok"

step "go vet ./..."
go vet ./...
echo "ok"

step "go test ./..."
go test ./...
echo "ok"

if [ "$fast" = "1" ]; then
	step "skipped: cross-compilation (-fast)"
	exit 0
fi

step "compiling for every platform"
# macOS is the primary target: the machine's own platform goes first, then
# the rest of macOS, then Windows, then Linux. A change that breaks the
# platform we develop on fails here, before anything else is built.
primary="darwin/arm64 darwin/amd64"
secondary="windows/amd64 windows/arm64 linux/amd64"
platforms="$(go env GOOS)/$(go env GOARCH)"
for p in $primary $secondary; do
	case " $platforms " in
	*" $p "*) ;;
	*) platforms="$platforms $p" ;;
	esac
done
for platform in $platforms; do
	GOOS="${platform%/*}" GOARCH="${platform#*/}" CGO_ENABLED=0 go build ./...
	echo "  ok  $platform"
done

step "versions agree"
json_version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' mygo.json | head -1)
go_version=$(sed -n 's/^const Version = "\(.*\)"$/\1/p' internal/app/app.go | head -1)
if [ "$json_version" != "$go_version" ]; then
	echo "mygo.json says $json_version, internal/app/app.go says $go_version"
	exit 1
fi
echo "ok  $json_version"

echo
echo "Everything passed."
