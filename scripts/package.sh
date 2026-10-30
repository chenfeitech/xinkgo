#!/usr/bin/env bash
#
# Packages Xinkgo for the platforms given as arguments, with `mygo build`.
# Without arguments it packages the machine's own platform, which on a Mac
# is macOS: the primary target, whose artifacts (.app and .dmg) come first.
#
#	scripts/package.sh                                  # macOS, this machine
#	scripts/package.sh darwin/arm64 darwin/amd64        # macOS, both
#	scripts/package.sh darwin/universal                 # macOS, one app of both
#	scripts/package.sh windows/amd64                    # Windows, x64
#
# The version has to be the same in mygo.json and in the source, so the
# app never shows a version it was not packaged as.
#
set -euo pipefail

cd "$(dirname "$0")/.."
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

platforms=("$@")
if [ ${#platforms[@]} -eq 0 ]; then
	# macOS is the primary target: package the machine we are on, which on
	# a Mac is macOS, so the artifacts we develop against come first.
	platforms=("$(go env GOOS)/$(go env GOARCH)")
fi

json_version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' mygo.json | head -1)
go_version=$(sed -n 's/^const Version = "\(.*\)"$/\1/p' internal/app/app.go | head -1)
if [ -z "$json_version" ] || [ -z "$go_version" ]; then
	echo "Could not read the version of mygo.json or internal/app/app.go"
	exit 1
fi
if [ "$json_version" != "$go_version" ]; then
	echo "mygo.json says $json_version, internal/app/app.go says $go_version"
	echo "Change both, or neither: the app shows the version it was built with."
	exit 1
fi

joined=$(IFS=,; echo "${platforms[*]}")
echo "Packaging Xinkgo $json_version for $joined"
go tool mygo build -platform "$joined"
