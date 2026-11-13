# Task4 public-program and owner-loss corpus

This machinery exercises the frozen Phase3B.4 prospective family against the reviewed v2 runtime. Its raw results are preliminary until the research repository's Task5 independently checks and requalifies the final frozen source pair. No production implementation or manuscript claim is changed here.

Run the campaign from this runtime checkout with the offline environment recorded in the research Task4 report:

```sh
ALAN_PHASE3B4_EVIDENCE=/fresh/normal go test -count=1 -timeout=10m -tags integration ./internal/durabletest -run '^TestPhase3B4Campaign'
ALAN_PHASE3B4_EVIDENCE=/fresh/race go test -count=1 -timeout=10m -race -tags integration ./internal/durabletest -run '^TestPhase3B4Campaign'
ALAN_PHASE3B4_PUBLIC_PARENTS=/fresh/normal ALAN_PHASE3B4_PROMOTED_OUTPUT=/fresh/promoted go test -count=1 -tags integration ./internal/executor -run '^TestPhase3B4PromotedPublicParents$'
```

Each run requires loopback endpoints and subprocesses. Evidence roots must be fresh. Existing v1 archives and helpers remain unchanged. `case.json` names exact principal and auxiliary identities, a public-constructor manifest and ordered case-relative captures. `provision.json` and `programs/<hash>.json` contain actual authored bytes. Each capture contains stopped/backup SQLite bytes, raw and typed inspection, paged history, Replay, and a cost sidecar. Independent endpoint databases and journals record contacts/applications. The root contains source identities, exact boundary/run outputs and component digests. Source and execution path values retain provenance even when capture references are relocated.

`P02-v2` is an ordinary automatic Run. P08/P09 and negative temporal schedules use public Commit, delivery/ACK, lifecycle writers and Run. A controlled pending-decision Run stops at the next staged ordinary commit; it does not privately authorize an effect or inject a consumed observation. The C family uses actual acknowledged process barriers and SIGKILL. C07 retains both separate pre/post-Open pairs; the initial Open is separately recorded. A planned grant/service ID is not evidence that it already exists at a cut. Whole-Store loss has no newly claimed paired-agent model mapping.

The independent principal checker belongs to Task5. Fixture assertions cover exact original/service/result/reply chains, real selections, current gates, serving-root charges, endpoint evidence, terminal runs and quiescence separately. P10 names q2 separately and does not count it as a principal. Cost-only lengths0/1/8/32 have five independent observations each. Timing windows are stated in measurements.json; they include boundary/runner overhead and are not allocation counts or an O(1) claim. The exact admission lookup query plan must use the nonunique observation-ID index.

N10b deliberately permits two effects under one MaxInFlight1 root. The first genuine request occupies the slot while a second explicitly own-authorized operation waits; after release that exact second operation must authorize and complete. It claims capacity blocking, not a principal completion/custody result. N10e preserves actual rejected public Retry (`invocation_retry_unsupported`) as a construction obstruction; it does not manufacture new-generation authority. N10f has no public ownership-revision writer in the current Store/lifecycle API. False-revision claim controls are distinct from that missing transition. There is no invented TTL/expiry policy.

Generic adapter unknown and external applied-without-local-result recovery are separately named v2 ancillary controls. Known-result C06 and reserved/pre-gate C05 do not substitute for them. Unchanged v1 regressions remain identified as v1.

K01/K02 first create and run the exact authored public adapter parents under formats13/15 with a real consumed result and one independent endpoint application. The separate executor-package compatibility test copies those complete parents and uses the existing internal `recordAdapterResult` writer to append deliberately supplied rejected conflict evidence, promoting to14/16. It checks unchanged program/history/result tables, original winner consumption, Inspect/Replay/Open and no transport. This ancillary control establishes no public-only conflict-production schedule: there is no public adapter result writer, and concurrent Run is rejected with ErrRunActive. No SQL repair or private consumption is used.
