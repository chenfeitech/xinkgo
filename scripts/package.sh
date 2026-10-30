#!/usr/bin/env bash
#
# Packages Xinkgo for the platforms given as arguments, with `mygo build`.
# Without arguments it builds Windows on amd64, the app's main target.
#
#	scripts/package.sh                                  # Windows, x64
#	scripts/package.sh windows/amd64 windows/arm64      # both
#	scripts/package.sh darwin/universal                 # macOS
#
# The version has to be the same in mygo.json and in the source, so the
# app never shows a version it was not packaged as.
#
set -euo pipefail

cd "$(dirname "$0")/.."
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

platforms=("$@")
if [ ${#platforms[@]} -eq 0 ]; then
	platforms=(windows/amd64)
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
