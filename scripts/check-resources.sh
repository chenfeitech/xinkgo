#!/usr/bin/env bash
#
# The plugin host reads its shim and compatibility list from the app's
# resources at runtime. If those files do not make it into a packaged
# build, every plugin silently stops working while the rest of the app is
# fine. This script checks the source files and every build artifact that
# is present, so a missing resource fails the gate rather than a user's
# machine.
#
#   scripts/check-resources.sh            # sources + any built artifacts
#   scripts/check-resources.sh -source   # sources only
#
set -euo pipefail

cd "$(dirname "$0")/.."

# Files the host reads through mygo.Path(mygo.PathResources). Keep this in
# step with newPluginHost in internal/app/plugin_host.go.
required=(
  "obsidian-plugins/compatible.json"
  "obsidian-shim/host.html"
  "obsidian-shim/shim.js"
  "obsidian-shim/runtime.js"
)

fail=0
missing() { echo "  missing: $1"; fail=1; }

# check_dir verifies one directory that should hold the resources: the
# top-level resources/ of the sources, a bundle's Contents/Resources, or
# the resources/ beside a Windows/Linux executable.
check_dir() {
  local label="$1" dir="$2" file
  echo "  $label"
  for file in "${required[@]}"; do
    if [ ! -f "$dir/$file" ]; then
      missing "$dir/$file"
    fi
  done
}

echo "== plugin resources =="
check_dir "sources" "resources"

if [ "${1:-}" = "-source" ]; then
  if [ "$fail" -ne 0 ]; then exit 1; fi
  echo "ok"
  exit 0
fi

# macOS bundles: <build>/<platform>/<Name>.app/Contents/Resources.
shopt -s nullglob
for bundle in build/*/*.app; do
  check_dir "$bundle" "$bundle/Contents/Resources"
done

# Windows/Linux flat builds: resources/ beside the executable. Only ones
# that carry an executable are checked, so a stale empty build dir does
# not fail the gate.
for platform in build/*; do
  [ -d "$platform" ] || continue
  has_exe=0
  for exe in "$platform"/*.exe "$platform"/xinkgo "$platform"/Xinkgo; do
    [ -f "$exe" ] && has_exe=1 && break
  done
  if [ "$has_exe" -eq 1 ]; then
    check_dir "$platform/resources" "$platform/resources"
  fi
done
shopt -u nullglob

if [ "$fail" -ne 0 ]; then
  echo "Plugin resources are missing from the build. Run 'make package' and try again."
  exit 1
fi
echo "ok"
