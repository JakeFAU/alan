# Compact scaling apparatus

This overlay reuses the hash-pinned P2 multi-file public driver and SQLite
accounting. It does not overwrite earlier tools or archives. `inputs.json`
binds every original driver/instrument input; a changed input is an error.
The clean source inventory is separately required for generation and every row.

`apparatus.py generate` copies the driver into a new external directory and
adds `-approval-mode=snapshot|targeted`. Snapshot retains the original public
inspection/basis call; targeted uses a Go interface assertion so baseline builds
remain valid and reject targeted mode only when selected. Both routes retain
the original approval transaction, every Run/report/prefix proof, final proof,
completed reopen/no-contact resume and Replay. `prefix_proof.go` is unchanged.
There is no tool-only continuation reader or SQL shortcut.

For `--kind=counters`, the pinned P2 generator's SQLite accounting is reused
unchanged, while its work injection block is replaced with
`baseline-anchors.json`. Each insertion must match exactly once. Future changed
helper bodies require an explicitly reviewed new anchor profile supplied with
`--anchors`; a removed anchor fails closed rather than silently reporting zero.
The profile instruments actual loops, never predicted counts or an analytic
shortcut. Generated original/modified manifests and exact patches are retained.

Per-stage/prefix snapshots contain fixed SQL totals, fixed scope totals,
overflow metrics and bounded Go counter names. They never construct the full
SQL family/cell snapshot. One `execute_counter_registry` delta is emitted at Execute
end, using a start and end full registry captured only at those two boundaries
for exact family/scope attribution and reconciliation. All intermediate resource
snapshots use the compact vocabulary. Execute SQL scalar deltas also remain in
the enclosing stage. Diagnostic timings are never
headline timings. SQLite row returns are not rows examined; unavailable
connection-setup internal VM work and unknown transaction scopes remain visible.
No scope/family totals may be added together as SQL executions.

Example, outside the original working directory:

```sh
python3 apparatus.py generate --kind ordinary --source "$EXPORT" \
  --source-manifest "$SOURCE_MANIFEST" --out "$NEW_BUILD"
python3 apparatus.py generate --kind counters --source "$EXPORT" \
  --source-manifest "$SOURCE_MANIFEST" --sqlite "$PINNED_SQLITE" --out "$NEW_COUNTER_BUILD"
python3 apparatus.py record --binary "$NEW_BUILD/driver.bin" --overlay \
  --source "$EXPORT" --source-manifest "$SOURCE_MANIFEST" \
  --out "$NEW_ROW" --bound 8 --label "$SOURCE_COMMIT"
python3 -m unittest discover -s . -p 'test_*.py' -v
```

Generation/build commands and source/tool/generated/binary hashes are in
`overlay.json`; row argv/cwd/whitelisted environment, exit, stdout/stderr and
before/after binary/source identities are in `row.json`. Existing rows/outputs
cannot be replaced. Failed rows remain and stop the recorder. These are pilot
and diagnostic tools, **not** the confirmatory controller: the contract's exact
bounds, excluded warmups, six alternating adjacent measured pairs and two
comparisons must be frozen independently before confirmatory execution. Pilots
cannot be relabeled as confirmatory samples.
