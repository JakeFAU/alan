# Public effect boundary probes

`New(Config{Parent:true, Peer:true, Delta:1})` constructs an ordinary invocation
parent, controlled effectful child, and independent effectful peer. The parent
receives only its original invocation outcome. `Registry(endpoint, mode)` supplies
real HTTP mutation inside Execute. Modes are success, selected-error (genuine
receipt retained but normalized unsuccessful), and unknown (no result retained).
The independent delta-two peer always uses success mode. Prepare performs no I/O.

Use the separate public-import consumer module:

```sh
go run . init -db /tmp/alan-parent.db
go run . run -db /tmp/alan-parent.db -endpoint http://127.0.0.1:PORT/apply -mode unknown
go run . inspect -db /tmp/alan-parent.db
go run . replay -db /tmp/alan-parent.db
```

Run these commands from `examples/effectboundary/consumer`. The endpoint is the
controlled persistent `alan-resource` fixture, not a real external service.
The consumer's output is host inspection, never a parent observation. See
`docs/foundational/phase2a/05-parent-boundary.md` for the exact knowledge boundary,
real connection-loss distinction, SIGKILL witnesses, and qualification evidence.
