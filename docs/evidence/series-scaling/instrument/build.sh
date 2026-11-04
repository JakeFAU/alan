#!/bin/sh
# Generate and build only the diagnostic driver; source inputs remain untouched.
set -eu
if [ "$#" -ne 4 ]; then
  echo 'usage: build.sh ALAN_SOURCE SQLITE_MODULE_SOURCE DRIVER_SOURCE NEW_OUTPUT_DIR' >&2
  exit 2
fi
instrument=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
python3 "$instrument/generate.py" --alan "$1" --sqlite "$2" --driver "$3" --out "$4"
output=$(CDPATH= cd -- "$4" && pwd)
export GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
cd "$output/driver"
cp go.mod measurement.mod
cp go.sum measurement.sum
go mod edit -modfile=measurement.mod "-replace=github.com/JakeFAU/alan=$output/alan" "-replace=github.com/mattn/go-sqlite3=$output/sqlite"
go build -mod=readonly -modfile=measurement.mod -o "$output/driver.bin" .
shasum -a 256 "$output/driver.bin" "$output/driver/measurement.mod" > "$output/build.sha256"
