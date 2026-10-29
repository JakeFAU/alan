# Public terminal probe result

Classification: concretely tested lower-level public runtime accessibility.
The separate module `examples/communication/consumer` imports standard library
and public author/runtime/schema packages only. Its executable is terminal-only;
it makes no J2 join execution claim.

Final evidence is `terminal-probe-reviewed/sha256.json`; exact source/command
identities are in `../context/source-identity.json` and `../context/checks/terminal-final-freeze.json`.
Normal and race consumer gates (`consumer-normal-reviewed`, `consumer-race-reviewed`)
and the external build/executable gates all exited0 using local Go1.27.3,
offline readonly dependencies, and no OPENAI_API_KEY.

The authored peer executes genuine Success at sequence1. Before delivery,
`terminal_obligations` freezes its receiver/done destination and actual terminal
projection. The independent original-semantics receiver matches terminal /
runtime-terminal / peer / source-generation1 / done / empty-operation. Its exact
raw bytes are hashed as the submitted public Artifact; existing compiler output
is never rewritten. `DeliverTerminal` accepts the anonymous
`struct{ Target, Obligation string }` parameter, binds target receiver generation1
and returns the original authenticated observation ID. The after-delivery
inspection shows no consumed_sequence. Genuine receiver Commit selects that ID,
binds peer_outcome=succeeded at PC1/sequence1, and retains matching history.selected
and consumed_sequence. Its next genuine End reaches Success/sequence2. There are
two static runs, no composition root and no ordinary/effect/invocation intent.
Replay is retained historical-consistency evidence only.

This terminal fact concerns the source's retained local outcome. It does not
by itself prove ordinary-message authorization, admission or consumption. The
current ordinary author Receive still cannot wait on this authenticated terminal
kind; the successful probe uses an independently encoded raw public runtime
artifact. No constructor/facade repair or third report/ACK/terminal join protocol
was introduced. Stable generation1, frozen permission and an admission-open
receiver are explicit fixture assumptions.

Earlier `terminal-probe` and `terminal-probe-final` directories preserve the
successful direct-test and executable-refactor stages respectively; the reviewed
directory is the final-source exported trace. The original Phase1J evidence and
preliminary source audit remain unchanged.
