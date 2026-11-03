# Runtime extraction

The source is research commit `003dc9c45ac9b0c1f48565632f5be80f0f05beed`, including the change that retains conflicting receipt evidence across recovery. The original module at that commit was already named `github.com/JakeFAU/alan`. An application using those package paths can keep its imports. The research checkout developed after the split uses `github.com/JakeFAU/alan-research`. The library builds from this module and the pinned SQLite driver. [Frozen research source](https://github.com/JakeFAU/alan-research/tree/003dc9c45ac9b0c1f48565632f5be80f0f05beed)

## Package inventory

| Location | Scope |
| --- | --- |
| `author`, `schema`, `compose` | Public compiler, typed retained records and bounded collections, ordinary application wiring and finite invocation. |
| `runtime`, `effects` | Public SQLite executor facade, durable inspection/replay, resource limits, dispatch authority, adapter and receipt protocols. |
| `adapters/lookup`, `adapters/openai`, `adapters/receiptquery` | Existing corpus lookup, Responses adapter and controlled queries for original receipts. Adapter tests use controlled endpoints. |
| `internal/executor`, `internal/boundary`, `internal/fixture` | Reducer, journal, SQLite schemas, supervision, dispatch, settlement and reconciliation. The last two packages supply process boundaries and the executor's external wire protocol. |
| `cmd/alan` | Existing provision, run, input, inspect and replay CLI. |
| `internal/durabletest`, `internal/testcmd/alan-resource`, `internal/storetest` | Selected real-process and adversarial tests, their controlled endpoint executable, and disposable copies of frozen SQLite fixtures. |
| `examples/quickstart` | New public consumer covering provision, wait, reopen, effect execution, inspection and read-only replay. |

All 59 implementation files in the first five rows were copied byte for byte. The public signatures, serialized identities, reduction rules, SQLite DDL and version checks come from the same source. Tests that used a research application, exported campaign evidence or opened files from the historical archive now use local test inputs. All test inputs travel with the module. The controlled endpoint command moved under `internal/testcmd` for subprocess tests.

The research repository keeps the monograph, independent bounded models, historical consumers, research CLIs, model mutations, campaign harnesses and raw archives. Those qualifications are tied to their recorded sources and assumptions. The following checks exercise the extracted implementation.

## Verification

Run from a checkout with Go 1.27.3, CGO, a C compiler, local loopback, subprocesses and a filesystem with SQLite locking.

```sh
export GOTOOLCHAIN=local GOWORK=off
go mod verify
go mod tidy -diff
gofmt -l adapters author cmd compose effects examples internal runtime schema
go build ./...
go vet ./...
go vet -tags integration ./...
go test -count=1 -timeout=60m ./...
go test -count=1 -timeout=60m -tags integration ./...
go test -count=1 -timeout=60m -race -tags integration ./...
```

The durable tests acknowledge the targeted execution boundary before killing a child. They reopen its SQLite store in a new process, check the original identities and compare inspection before and after replay. The integration suite exercises a missing non-idempotent receipt, permission to read an original result with adoption denied, competing resolvers, forged bindings and preservation of the first settled outcome. One counterexample holds a grandchild's admitted call after its parent fails. The parent alone can drain while the owning scope must still wait for the grandchild's accounting. Executor tests also cover revoked dispatch, sticky accounting errors, admission and effect ceilings, physical capacity and failure propagation. The [quickstart](../README.md#run-a-computation) exercises the public entry points.

## SQLite compatibility fixtures

The following files under `internal/storetest/testdata` decompress to the original database bytes. Paths in the table are relative to `docs/foundational` in the frozen research commit. Only tests expand the files, each into its own temporary directory.

| Local name (`.sqlite.gz`) | Source database |
| --- | --- |
| `legacy-effects` | `phase1f/evidence/live-phase1e/store.sqlite` |
| `recorded-adapter` | `phase2a/evidence/process-normal/TestPhase2KnownEvidenceCrashCuts_adapter_recorded-cut-local.sqlite.gz` |
| `recorded-idempotent` | `phase2a/evidence/process-normal/TestPhase2KnownEvidenceCrashCuts_idempotent_recorded-cut-local.sqlite.gz` |
| `adopted-handoff` | `phase2c/evidence/process-normal/atomic-import/final-local.sqlite.gz` |
| `unrecorded-read` | `phase2c/evidence/process-normal/held-lost/lost-absence-local.sqlite.gz` |

Ordinary, composition and reconciliation stores use formats 3, 4 and 5. The first conflicting valid receipt retained as evidence can promote them to 6, 7 and 8 respectively. The first result remains selectable. The tests exercise the existing version acceptance, promotion and rejection paths without reinitializing the database. See the [maintenance guidance](../README.md#development) for porting a fix between the two repositories.
