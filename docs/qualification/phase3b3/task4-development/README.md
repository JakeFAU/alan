# Reciprocal fixture development evidence

This archive precedes independent Task 4 review and a committed fixture-source
freeze. It preserves the complete development record, including failed test
controller attempts, and the final candidate's 48-case ordinary and race
corpora. It is not the final frozen-source phase qualification.

`candidate.tar.gz` contains the exact eight new fixture/test sources over
`347094a7b18d351c6ad18df526ef6133777bca44`, the full 304 Go/module-input manifest,
review diff and implementation report. It changes no existing production or
baseline source. `evidence.tar.gz` contains 7,249 files under
`task4-development/`; the wrapper manifest records every member and both
archive hashes. All member hashes were verified before packaging, inside the
archive and again against the source directory. No SQLite file was opened.

Use `qualified-ordinary/case-index.json` and `qualified-race/case-index.json`
inside the archive for the candidate's 48 cases. Each corpus has 1,761 files
bound by per-case manifests. Earlier directories retain apparatus failures and
superseded runs; they must not be added to the final passing-case count.
The initial fixture compile failure is not an authority counterexample.

The `controls/control-wrong-prior-selection-1914204028/final/inspect.json`
member is the original input to the separate research selection-narrowing
audit (SHA256
`60897acf3d7cc14ab351e23314ddb656e1b68801bbeaa772521623bb79dc0327`). It retains
an actual accepted exact grant and earlier BA selection, followed by ready
selection and a refused service. This is the declared immediate-selection
restriction, not missing original consent. The later qualified corpus retains
the same control on the complete candidate source.

Keep command records, absolute historical case paths and manifests unchanged.
Map their original output-root prefix to the extracted root in a working
reproduction configuration; path remapping does not change logical runtime IDs.
Follow the parent README for source restoration and overlay reproduction.
Never open archived SQLite originals, including in read-only mode: copy every
database/WAL/SHM member to a disposable directory first. Post-Open snapshots
were taken through SQLite online backup before Run; they are not live-file
copies.
