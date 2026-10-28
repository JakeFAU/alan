# Go test selector investigation — disproved hypothesis

This is selector-only diagnostic evidence, not an ALAN behavioral test or counterexample. The temporary standard-library fixture uses the real top-level and subtest names and the exact original reproduction selector. Go1.27.3 exited0 and selected both intended process cuts, the causal capacity case, unknown sibling, live companion, isolated replay and faulty application. `command.json` and raw stdout preserve the actual selection.

The diagnostic Python wrapper then exited1 because its own final assertion incorrectly expected five missing top-level controls. That hypothesis was falsified by the actual Go output. No selector bug or reproduction-runtime failure was established; the original selector remains unchanged. Local Go1.27.3 `testing/match.go` splitRegexp explicitly constructs alternationMatch for top-level `|` while retaining slash-separated subtest paths within each alternative.

A later reproduction-tool guard will require the exact selected pass names from actual extracted Go JSON, guarding future selection drift without rewriting the working selector. Its change and source identity remain separate from active frozen-source suite results. No ALAN state, bound, primitive or oracle was changed by this probe.
