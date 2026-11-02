#!/usr/bin/env bash
#
# Writes the PNG the app's icon is rendered from out of the .icns of macOS,
# so that the two never drift apart: macOS is packaged with the .icns as it
# is, and Windows and Linux with the icon rendered from this PNG.
#
#	scripts/icon.sh       # assets/icon.png from assets/xink.icns
#
# It runs on a Mac, whose tools are the ones that read an .icns. The PNG is
# kept in the repository, so packaging needs no Mac of its own.
set -euo pipefail

cd "$(dirname "$0")/.."

icns=assets/xink.icns
png=assets/icon.png

if [ ! -f "$icns" ]; then
	echo "There is no $icns, whose icon the app is packaged with."
	exit 1
fi
if [ "$(uname -s)" != "Darwin" ]; then
	echo "Only a Mac reads an .icns: run this there, and commit the $png it writes."
	exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
iconutil -c iconset "$icns" -o "$work/icon.iconset"
# The largest of them, icon_512x512@2x: every icon is drawn from 1024 down.
cp "$work/icon.iconset/icon_512x512@2x.png" "$png"
sips -g pixelWidth -g pixelHeight "$png" | tail -2
echo "Wrote $png from $icns"
