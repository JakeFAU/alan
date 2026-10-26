# Independent development revision2 adjudication

All initial and delivered-final required-fact, stopping and numeric grades are retained. The original grade/report files are preserved. The substantive correction is to **diagnostic-only provenance**: a diagnostic answer string has no citation array, so its citation counts, validity and coverage are unavailable. The original final-array calculation is preserved as an explicitly conditional view, not an observed metric. Delivered quality remains one success among eight opaque outputs.

## Decisions

**Empty delivered answers:** response-01, response-02 and response-03 deliver empty strings with revision_limit dispositions. They correctly receive 0/5, 0/3 and 0/4 supported groups, incorrect stopping, and null coverage/precision denominators. Their actual final citation arrays are byte-valid, so citation validity remains 100%; those arrays cannot deliver missing answer text. Diagnostic text is never substituted. Initial-to-delivered content regresses in all three. These are emitted operational dispositions; public-result execution fields alone do not establish backend status or epistemic abstention. No normalization defect is demonstrable from this packet.

**Diagnostic association:** response-01, -02, -03 and -08 each have a string but no separate diagnostic citations. Their semantic support is still directly gradeable at 3/5, 1/3, 2/4 and 5/5 respectively. Diagnostic provenance fields change to null, and response-08 diagnostic quality_success changes from true to null. Its independently observed delivered final still passes. A copy of each former grade is retained under conditional_grade_if_final_citations_apply, with coverage bounds 0–100% representing unknown association. response-01's diagnostic support improves by adding the agreed database 5 snapshot and the strict replication-lag >45 seconds/five-consecutive-minute trigger. Because diagnostic citation coverage is unknown, its full-dimension improved classification becomes not_observable, with improved retained conditionally. The analogous unchanged classifications for response-02 and -03 become not_observable across all dimensions, while their support is explicitly unchanged. response-08's initial-to-delivered comparison is unchanged on actual arrays, independently of the diagnostic.

**Marlin semantics:** response-01 initial supports the operations passed/ready group, and its diagnostic additionally supports both rollback groups. Neither states same-cluster identity; both frame withheld readiness around missing supplied excerpts. Thus the bundled conflict-unresolved group remains missing. The rationale is clarified: database failure is already a separate support group, so do not add it again as an unexpressed support prerequisite. Both opposing report findings plus authority context are necessary for citation entailment of the conflict conclusion. Initial/final arrays omit Marlin line 3 [342,485); the authority header [0,199) and operations line [200,341) cannot supply it. The rollback is at [486,655). A permissive contextual interpretation could add one conflict group to response-01 initial/diagnostic (ranges 1–2/5 and 3–4/5), but no conflict coverage, correct stopping or delivered quality gain follows. response-04 merely attributes the operations pass/readiness: that is a true report statement, not a false unified readiness claim. Its 1/5 remains unchanged.

**Scope of absence claims:** supplied-evidence/excerpts wording is naturally scoped to an evidence subset, including the consequents “so neither can be determined” and “cannot be determined” in responses -05 and -06. Their displayed citations omit rollback, but do not establish every runtime input. The original unverifiable_input_scope treatment and null precision are appropriate. Corpus-wide absence is not established; known rollback groups remain missing. Under a broader reading both omission claims become unsupported; conditional precision bounds from the fixed inventories are 0.5–1 for response-05 and 0.6–1 for response-06. These are logical sensitivity bounds, not measured precision. No forced unsupported-claim advantage is inferred.

**Remaining outputs:** response-02's initial/diagnostic and response-06 properly state packet-scoped approval uncertainty but omit the index 4 snapshot and strict query-failure >1 percent/three-consecutive-minute rollback. response-03 initial/diagnostic and response-05 state both kernel facts but omit agent 3.1 and error rate >2 percent/five-consecutive-minute rollback. response-07 supplies both Quartz branches, so it is a conditional answer with five applicable groups; missing common rollback fails stopping. response-08 supplies all five groups, including gateway 7 and p95 latency >300 milliseconds for ten consecutive minutes in both regions. It has complete entailing citations and correct stopping. Its four substantive propositions are explicitly enumerated and supported (4/4); a targeted question is not a factual assertion. Other nonempty precision remains unavailable because of input-scope uncertainty.

## Counts

| ID | Case | Initial support | Delivered support | Diagnostic support (unaccepted) | Delivered stopping |
| --- | --- | --- | --- | --- | --- |
| response-01 | dev-04 | 1/5 | 0/5 | 3/5 | incorrect |
| response-02 | dev-03 | 1/3 | 0/3 | 1/3 | incorrect |
| response-03 | dev-01 | 2/4 | 0/4 | 2/4 | incorrect |
| response-04 | dev-04 | 1/5 | 1/5 | — | incorrect |
| response-05 | dev-01 | 2/4 | 2/4 | — | incorrect |
| response-06 | dev-03 | 1/3 | 1/3 | — | incorrect |
| response-07 | dev-02 | 3/5 | 3/5 | — | incorrect |
| response-08 | dev-02 | 5/5 | 5/5 | 5/5 | correct |

Eight delivered outputs include three empty revision_limit stops, one support full pass, one correct stop and one quality success. All supported delivered groups have entailing citations. All 12 initial and 18 final citation occurrences are byte-valid. No pooled required-group score is reported because groups bundle different semantics. No system identities, system winner, production performance or heldout outcome is inferred.

## Verification and independence

Checked every required-fact and stopping decision against frozen development gold and corpus; verified eight content hashes/lengths, 25 line spans, all 30 initial/final citations, original source hashes, count arithmetic and recall/coverage/validity fractions. Original grades and gold were not edited. The JSON records all source hashes, original-to-adjudicated changes, unchanged IDs, retained conditional views and sensitivities.

This adjudicator is independent of the revision2 first grader and application implementation. **Prior-context limitation:** the same agent previously adjudicated revision1 and retains that context. No revision1 artifact was reread or used for comparison here; all decisions use revision2 outputs and the frozen rubric. This is not a fresh evaluator with no prior study exposure. No mapping, normalized/raw records, application source/review verdicts, heldout files, credentials, network, paid calls, subagents or commits were used. Opaque output structure may still reveal design; no identity was inferred.

## Hashes

- `blind-packets.json`: `bbb28c91612afdd8e49b8eaf57775a61d7f1acbfc0fad5fa1191437e19ee8d62`
- `grades-blind.json`: `09435109672f0536f68ea62aad0288a2757e7073a1ad73fa664a10baf55d2dba`
- `grading-report.md`: `86de6997f754ec543120b5eb4699886a0459c6536fdc227aab4348230bd3916a`
- `grades-adjudicated.json`: `661d7d99444eca529d9efd77f24e52c4df1757595335e7649652f3283559136a`
