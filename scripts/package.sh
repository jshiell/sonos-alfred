#!/usr/bin/env bash
# Builds the arm64 binary and packs it into a .alfredworkflow. Usage: package.sh <output path>
set -euo pipefail

out=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
cd "$(dirname "$0")/.."

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

GOOS=darwin GOARCH=arm64 go build -o "$stage/sonos-alfred" .
rm -f "$out"
cp workflow/info.plist "$stage/"
(cd "$stage" && zip -q "$out" sonos-alfred info.plist)
