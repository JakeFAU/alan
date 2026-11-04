#!/bin/sh
# Build this public-package driver against a selected ALAN checkout. Only the
# temporary modfile receives a local replace; the checked-in module stays portable.
set -eu
if [ "$#" -ne 2 ]; then
  echo 'usage: build.sh ALAN_SOURCE OUTPUT_BINARY' >&2
  exit 2
fi
source_path=$(CDPATH= cd -- "$1" && pwd)
driver_path=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
output_dir=$(dirname -- "$2")
mkdir -p "$output_dir"
output_dir=$(CDPATH= cd -- "$output_dir" && pwd)
output_path="$output_dir/$(basename -- "$2")"
work=$(mktemp -d "${TMPDIR:-/tmp}/alan-series-driver.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
cp "$driver_path/go.mod" "$work/driver.mod"
cp "$driver_path/go.sum" "$work/driver.sum"
export GOWORK=off GOTOOLCHAIN=local
cd "$driver_path"
go mod edit -modfile="$work/driver.mod" "-replace=github.com/JakeFAU/alan=$source_path"
go build -mod=readonly -modfile="$work/driver.mod" -o "$output_path" .
