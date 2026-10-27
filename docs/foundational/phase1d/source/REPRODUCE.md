# ALAN Phase 1D complete qualified source

This archive contains all 53 qualified Go source/dependency files, prior research contracts/evidence, the durability contract, source manifest, evidence assembly tools, and the full pinned SQLite module as a file:// Go proxy. The publication repository separately retains final qualification logs and review reports. No Go or C compiler binaries are bundled.

Prerequisites: Go 1.27.3, CGO, a C compiler, qualified Unix local filesystem, loopback/subprocess/SIGKILL permissions. The qualified host is darwin/arm64; other platforms are not claimed qualified.

From this extracted root:

```sh
export GOTOOLCHAIN=local
export GOPROXY="file://$PWD/dependencies"
export GOSUMDB=off
export GOMODCACHE="$(mktemp -d)"
export GOCACHE="$(mktemp -d)"
go test -count=1 ./...
go test -count=1 -race ./...
go mod verify
go vet ./...
go build -buildvcs=false -o alan ./cmd/alan
go build -buildvcs=false -o alan-resource ./cmd/alan-resource
./alan init --db demo.sqlite --demo nested
./alan run --db demo.sqlite
./alan inspect --db demo.sqlite
./alan replay --db demo.sqlite
```

The local proxy supplies the complete pinned dependency without network access. go.sum validates it. source-manifest.json defines the Go source digest; the separate archive-manifest.json published beside this archive hashes every member including collection tools. Source identity excludes generated evidence/documents to avoid self-reference. Historical evidence and current-source comparison have different source manifests. The collector text records the actual qualification command implementation with its original fixed local paths; ordinary go test commands above are the portable reproduction.
