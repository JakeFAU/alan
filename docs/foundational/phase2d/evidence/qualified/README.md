# Frozen process and SQLite evidence

The qualification checkout was `8ad4b3cffb5185ec236f152c87fc7b9ce3676c36`. The archives retain each captured file at its path relative to the qualification output root. Each producer exited zero on that commit with an empty working-tree status and no input differences. Every tar member was extracted and compared byte for byte against the completed producer output. Original gzip-wrapped SQLite backups and compressed traces retain their original bytes.

| Capture group | Archives per regime | Normal files | Normal archive bytes | Race files | Race archive bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| construction | 1 | 7 | 227,544 | 7 | 224,854 |
| process | 19 | 198 | 10,869,776 | 198 | 10,881,307 |
| grid | 5 | 372 | 14,206,856 | 372 | 14,194,547 |
| adverse | 9 | 146 | 5,601,724 | 146 | 5,597,230 |
| prior | 2 | 814 | 4,548,780 | 814 | 4,538,744 |
| Total | 36 | 1,537 | 35,454,680 | 1,537 | 35,436,682 |

Together the 72 archives contain 3,074 files in 70,891,362 compressed bytes. The member index records every tar path, original file size and SHA-256, the archive size and SHA-256, and the SHA-256 of the originating command record(s). The outcome index identifies the specific local SQLite backup for each projection, the actual parent and resolver statuses, and the actual parent classifier fields. Grid entries also include the driver's case parameters and declared result. A null classifier means the retained parent has no such field. [Member index](member-index.json.gz) · [Outcome index](outcomes.json.gz) · [Commands and independent decoders](../../07-runbook.md)

## Find a capture

Each `process-` archive holds one principal family. The paired family contains the common prefix, both continuations and the paired comparison. In the grid, AA through BB contain the 16 ordered mode pairs per placement. The controls archive contains the global plan and index, three nested relay cases, four one-less cases and the slow read case. The prior archives retain the inherited 2B flat capture and the 2C per-scenario capture.

| Family or subset | Normal | Race |
| --- | --- | --- |
| construction | [Archive](normal/construction.tar.gz) | [Archive](race/construction.tar.gz) |
| process-atomic | [Archive](normal/process-atomic.tar.gz) | [Archive](race/process-atomic.tar.gz) |
| process-auto-acks | [Archive](normal/process-auto-acks.tar.gz) | [Archive](race/process-auto-acks.tar.gz) |
| process-capacity-one | [Archive](normal/process-capacity-one.tar.gz) | [Archive](race/process-capacity-one.tar.gz) |
| process-equal-reverse | [Archive](normal/process-equal-reverse.tar.gz) | [Archive](race/process-equal-reverse.tar.gz) |
| process-finite-reads | [Archive](normal/process-finite-reads.tar.gz) | [Archive](race/process-finite-reads.tar.gz) |
| process-forward | [Archive](normal/process-forward.tar.gz) | [Archive](race/process-forward.tar.gz) |
| process-fresh-ack | [Archive](normal/process-fresh-ack.tar.gz) | [Archive](race/process-fresh-ack.tar.gz) |
| process-lost-absence | [Archive](normal/process-lost-absence.tar.gz) | [Archive](race/process-lost-absence.tar.gz) |
| process-manual-ack | [Archive](normal/process-manual-ack.tar.gz) | [Archive](race/process-manual-ack.tar.gz) |
| process-mixed | [Archive](normal/process-mixed.tar.gz) | [Archive](race/process-mixed.tar.gz) |
| process-original-selected | [Archive](normal/process-original-selected.tar.gz) | [Archive](race/process-original-selected.tar.gz) |
| process-paired | [Archive](normal/process-paired.tar.gz) | [Archive](race/process-paired.tar.gz) |
| process-pre-retention | [Archive](normal/process-pre-retention.tar.gz) | [Archive](race/process-pre-retention.tar.gz) |
| process-read-only | [Archive](normal/process-read-only.tar.gz) | [Archive](race/process-read-only.tar.gz) |
| process-retained-absence | [Archive](normal/process-retained-absence.tar.gz) | [Archive](race/process-retained-absence.tar.gz) |
| process-reverse | [Archive](normal/process-reverse.tar.gz) | [Archive](race/process-reverse.tar.gz) |
| process-spent | [Archive](normal/process-spent.tar.gz) | [Archive](race/process-spent.tar.gz) |
| process-stable-duplicate | [Archive](normal/process-stable-duplicate.tar.gz) | [Archive](race/process-stable-duplicate.tar.gz) |
| process-stable-error | [Archive](normal/process-stable-error.tar.gz) | [Archive](race/process-stable-error.tar.gz) |
| grid-AA | [Archive](normal/grid-AA.tar.gz) | [Archive](race/grid-AA.tar.gz) |
| grid-AB | [Archive](normal/grid-AB.tar.gz) | [Archive](race/grid-AB.tar.gz) |
| grid-BA | [Archive](normal/grid-BA.tar.gz) | [Archive](race/grid-BA.tar.gz) |
| grid-BB | [Archive](normal/grid-BB.tar.gz) | [Archive](race/grid-BB.tar.gz) |
| grid-controls | [Archive](normal/grid-controls.tar.gz) | [Archive](race/grid-controls.tar.gz) |
| adverse-competing-adopter-query | [Archive](normal/adverse-competing-adopter-query.tar.gz) | [Archive](race/adverse-competing-adopter-query.tar.gz) |
| adverse-competing-original | [Archive](normal/adverse-competing-original.tar.gz) | [Archive](race/adverse-competing-original.tar.gz) |
| adverse-external-multiplicity | [Archive](normal/adverse-external-multiplicity.tar.gz) | [Archive](race/adverse-external-multiplicity.tar.gz) |
| adverse-ledger-replacement | [Archive](normal/adverse-ledger-replacement.tar.gz) | [Archive](race/adverse-ledger-replacement.tar.gz) |
| adverse-lying-second-snapshot | [Archive](normal/adverse-lying-second-snapshot.tar.gz) | [Archive](race/adverse-lying-second-snapshot.tar.gz) |
| adverse-read-only | [Archive](normal/adverse-read-only.tar.gz) | [Archive](race/adverse-read-only.tar.gz) |
| adverse-unsupported | [Archive](normal/adverse-unsupported.tar.gz) | [Archive](race/adverse-unsupported.tar.gz) |
| adverse-wrong-grant | [Archive](normal/adverse-wrong-grant.tar.gz) | [Archive](race/adverse-wrong-grant.tar.gz) |
| adverse-wrong-role | [Archive](normal/adverse-wrong-role.tar.gz) | [Archive](race/adverse-wrong-role.tar.gz) |
| prior-2b | [Archive](normal/prior-2b.tar.gz) | [Archive](race/prior-2b.tar.gz) |
| prior-2c | [Archive](normal/prior-2c.tar.gz) | [Archive](race/prior-2c.tar.gz) |

## Parent and role outcomes

The counts below read the retained local `runs.projection` for `parent`, `resolver_l` and `resolver_r`. For grids each declared decision also matched the actual parent. A declared pending case had an open parent and no decision register. Process calls compare the decision, role classes and tree class against each available `business.json`. The process totals include both paired continuations and count the common prefix separately.

| Regime | Section | Snapshots | Applied | Applied with tree diagnosis | Invalid evidence | Open with no decision |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| normal | construction frontier | 1 | 0 | 0 | 0 | 1 |
| normal | process final | 20 | 16 | 0 | 0 | 4 |
| normal | process shared prefix | 1 | 0 | 0 | 0 | 1 |
| normal | grid AA | 16 | 1 | 8 | 0 | 7 |
| normal | grid AB | 16 | 1 | 8 | 0 | 7 |
| normal | grid BA | 16 | 1 | 8 | 0 | 7 |
| normal | grid BB | 16 | 1 | 8 | 0 | 7 |
| normal | grid controls | 8 | 2 | 4 | 1 | 1 |
| normal | adverse final | 9 | 3 | 0 | 2 | 4 |
| race | construction frontier | 1 | 0 | 0 | 0 | 1 |
| race | process final | 20 | 16 | 0 | 0 | 4 |
| race | process shared prefix | 1 | 0 | 0 | 0 | 1 |
| race | grid AA | 16 | 1 | 8 | 0 | 7 |
| race | grid AB | 16 | 1 | 8 | 0 | 7 |
| race | grid BA | 16 | 1 | 8 | 0 | 7 |
| race | grid BB | 16 | 1 | 8 | 0 | 7 |
| race | grid controls | 8 | 2 | 4 | 1 | 1 |
| race | adverse final | 9 | 3 | 0 | 2 | 4 |

| Regime and family | Role | Applied direct | Applied issued | Applied after original failure | Invalid | Unset |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| normal process | left | 0 | 15 | 1 | 0 | 4 |
| normal process | right | 0 | 16 | 0 | 0 | 4 |
| normal grid | left | 41 | 2 | 0 | 0 | 29 |
| normal grid | right | 40 | 2 | 0 | 1 | 29 |
| normal adverse | left | 0 | 3 | 0 | 2 | 4 |
| normal adverse | right | 0 | 5 | 0 | 0 | 4 |
| race process | left | 0 | 15 | 1 | 0 | 4 |
| race process | right | 0 | 16 | 0 | 0 | 4 |
| race grid | left | 41 | 2 | 0 | 0 | 29 |
| race grid | right | 40 | 2 | 0 | 1 | 29 |
| race adverse | left | 0 | 3 | 0 | 2 | 4 |
| race adverse | right | 0 | 5 | 0 | 0 | 4 |

## Inspect a backup

From this directory, this example lists the paired files and extracts a continuation's local Store into a new temporary directory.

```sh
dir=$(mktemp -d)
tar -tzf normal/process-paired.tar.gz
tar -xzOf normal/process-paired.tar.gz normal-process/paired/lost-response/final-local.sqlite.gz | gzip -dc > "$dir/local.sqlite"
sqlite3 -readonly "$dir/local.sqlite" "select subject, json_extract(projection,'$.Status'), json_extract(projection,'$.Data.decision.Text') from runs where subject in ('parent','resolver_l','resolver_r') order by subject"
```

To reconstruct the input directories for the independent decoders, extract each regime's archives under one directory. Each file path occurs in exactly one archive.

```sh
dir=$(mktemp -d)
for archive in normal/*.tar.gz race/*.tar.gz
do
  tar -xzf "$archive" -C "$dir"
done
```
