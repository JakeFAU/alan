#!/bin/sh
# Run serially on a quiet host. Full 256 execution can take several minutes.
set -eu
mode=${1:?usage: reproduce.sh baseline-or-final SOURCE OUTPUT [BASE_SOURCE]}
source_dir=$(CDPATH= cd -- "${2:?source export required}" && pwd)
output=${3:?new output directory required}
package_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
mkdir -p "$output"
output=$(CDPATH= cd -- "$output" && pwd)
if [ -e "$output/module" ]; then
 printf '%s\n' 'Output/module already exists; choose a fresh output directory.' >&2
 exit 1
fi
case "$mode" in baseline|final) ;; *) printf '%s\n' 'Mode must be baseline or final.' >&2; exit 1;; esac
if [ "$mode" = final ]; then
 base_source=$(CDPATH= cd -- "${4:?final mode also needs verified base source export}" && pwd)
fi
cp -R "$package_dir/module" "$output/module"
cd "$output/module"
export CGO_ENABLED=1 GOWORK=off GOPROXY=off GOTOOLCHAIN=local
export ALAN_SOURCE="$source_dir"
export SERIES_FIXTURE_DIR="$output/fixtures"
go mod edit "-replace=github.com/JakeFAU/alan=$source_dir"
go version > "$output/toolchain.txt"
go env > "$output/go-env.txt"
uname -a > "$output/os.txt"
if [ "$mode" = baseline ]; then
 go test -mod=mod -run='^$' -bench='BenchmarkManual(Loop)?Construction$' -benchmem -benchtime=100ms -count=5 -cpu=1 > "$output/construction.log" 2>&1
 go test -mod=mod -run='^$' -bench='Benchmark(Create|Execute|InspectReplay)/(1|8|32)$' -benchmem -benchtime=1x -count=1 -cpu=1 -timeout=30m > "$output/runtime-small.log" 2>&1
 go test -mod=mod -run='^$' -bench='BenchmarkCreate/256$' -benchmem -benchtime=1x -count=1 -cpu=1 -timeout=5m > "$output/create256.log" 2>&1
 go test -mod=mod -run='^$' -bench='BenchmarkCompleteBound256$' -benchmem -benchtime=1x -count=1 -cpu=1 -timeout=30m > "$output/runtime256.log" 2>&1
 go test -mod=mod -run='Test(FreshStorageFormats|HistoricalArchivesRemainReadOnly)$' -v -count=1 > "$output/compatibility.log" 2>&1
 exit 0
fi

go test -mod=mod -tags=series -run='^$' -bench='Benchmark(ManualConstruction|ManualLoopConstruction|SeriesConstruction|SeriesProvisionCopy)$' -benchmem -benchtime=100ms -count=5 -cpu=1 > "$output/construction.log" 2>&1
go test -mod=mod -tags=series -run='^$' -bench='BenchmarkSeriesRuntime$' -benchmem -benchtime=1x -count=1 -cpu=1 -timeout=30m > "$output/runtime.log" 2>&1
go test -mod=mod -tags=series -run='Test(FreshStorageFormats|HistoricalArchivesRemainReadOnly|NewSeriesFixture|CanonicalLoweringEquality)$' -v -count=1 > "$output/compatibility.log" 2>&1
go test -mod=mod -tags=series -run='^$' -bench='BenchmarkSeriesView(Provisioned|Completed8)$' -benchmem -benchtime=100ms -count=5 -cpu=1 -timeout=5m > "$output/view.log" 2>&1

# A distinct module/binary binds only the old generic runtime to read new stores.
cp -R "$package_dir/module" "$output/base-reader-module"
(
 cd "$output/base-reader-module"
 go mod edit "-replace=github.com/JakeFAU/alan=$base_source"
 go build -mod=mod -o "$output/base-reader" ./cmd/base-reader
)
cp "$SERIES_FIXTURE_DIR/series11.sqlite" "$SERIES_FIXTURE_DIR/base-reader11.sqlite"
"$output/base-reader" "$SERIES_FIXTURE_DIR/base-reader11.sqlite" > "$output/base-reader11.log" 2>&1
cp "$SERIES_FIXTURE_DIR/series11.sqlite" "$SERIES_FIXTURE_DIR/series12.sqlite"
python3 - "$base_source" "$package_dir/overlay/promote_series_test.go.txt" "$output/overlay.json" <<'PY'
import json,sys
from pathlib import Path
base,driver,destination=map(Path,sys.argv[1:])
Path(destination).write_text(json.dumps({'Replace':{str(base/'internal/executor/independent_series_promotion_test.go'):str(driver)}},indent=2)+'\n')
PY
(
 cd "$base_source"
 SERIES_PROMOTE_PATH="$SERIES_FIXTURE_DIR/series12.sqlite" go test "-overlay=$output/overlay.json" -run='^TestIndependentPromoteSeries$' -v -count=1 ./internal/executor > "$output/promotion12.log" 2>&1
)
cp "$SERIES_FIXTURE_DIR/series12.sqlite" "$SERIES_FIXTURE_DIR/base-reader12.sqlite"
"$output/base-reader" "$SERIES_FIXTURE_DIR/base-reader12.sqlite" > "$output/base-reader12.log" 2>&1
go test -mod=mod -tags=series -run='^TestRetainedSeriesFormats$' -v -count=1 > "$output/retained-formats.log" 2>&1
