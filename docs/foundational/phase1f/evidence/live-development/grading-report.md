# Independent blinded development grading

Graded all eight opaque outputs, retaining initial and delivered-final answers separately. Frozen required-fact groups are not equal-sized atomic claims, so the table reports per-case counts without a pooled score. Exact answer excerpts and source hashes are in `grades-blind.json`. All eight corpus documents match their manifest hashes and byte spans. Every emitted citation is byte-valid; repeated headers do not establish omitted claims. No substantive-claim precision is reported because no exhaustive atomic-claim denominator was constructed.

| Opaque output | Case | Initial supported groups | Final supported groups | Final cited supported groups | Final stopping | Observed change |
| --- | --- | --- | --- | --- | --- | --- |
| response-01 | dev-04 | 1/5 | 0/5 | 0 | incorrect | regressed, adjudication-sensitive |
| response-02 | dev-03 | 1/3 | 1/3 | 1 | incorrect | unchanged |
| response-03 | dev-01 | 2/4 | 2/4 | 2 | incorrect | unchanged |
| response-04 | dev-04 | 0/5 | 0/5 | 0 | incorrect | unchanged |
| response-05 | dev-01 | 0/4 | 0/4 | 0 | incorrect | unchanged |
| response-06 | dev-03 | 1/3 | 1/3 | 1 | incorrect | unchanged |
| response-07 | dev-02 | 3/5 | 3/5 | 3 | incorrect | unchanged |
| response-08 | dev-02 | 5/5 | 5/5 | 5 | correct | unchanged |

**Cedar (dev-01):** response-03 correctly states and cites both kernel facts, but never supplies agent 3.1 or the strict error-rate >2 percent / five-consecutive-minute trigger. Its revision adds context without improving required coverage. response-05 delivers an empty `invalid_result`: no answer, no citations, and no epistemic abstention credit. Citation-validity and coverage fractions with zero denominators are null.

**Quartz (dev-02):** response-07 gives both regional limits and recognizes the missing region, but omits gateway 7 and the common p95 latency >300 milliseconds / ten-consecutive-minute rollback. It supplies a conditional answer and therefore does not qualify for the targeted-clarification exception. response-08 gives both limits, requests the region, and gives the full common rollback with entailing citations; it passes all required dimensions. Its final disposition says clarification, but semantic content is a complete conditional answer.

**Larch (dev-03):** responses-02 and -06 correctly withhold approval and cite the explicit absence of signed approval/rejection. Both omit the index 4 snapshot and query-failure >1 percent / three-consecutive-minute trigger. response-02 additionally says revision 1 does not establish when/version: this naturally refers to the packet, which does contain the rollback instruction, and is recorded as an unsupported extra with a scope caveat.

**Marlin (dev-04):** responses-01 and -04 never state the operations passed/ready and database failed/not-ready findings, the agreed database 5 snapshot, or replication lag >45 seconds / five-consecutive-minute trigger. All their references repeat the authority header. response-01 initial reproduces the literal conflict-unresolved gold group (same cluster/window, equal authority, neither supersedes, no resolution, readiness not established), so it receives one support group but zero citation coverage: both report findings are needed for cited conflict support. Its final explicitly cannot determine whether the reports agree/conflict or establish readiness, so it receives only partial-group recognition. response-04 also omits same-cluster identity and an actual conflict conclusion; its authority paraphrase is recorded as partial, not full group credit. Neither stops correctly.

## Adjudication and limits

1. **Marlin group boundary:** Whether response-01 initial's literal gold-group coverage is sufficient without explicitly stating opposing findings is the main interpretation issue. This grade treats findings as separately scored groups and credits the initial group; requiring explicit disagreement would reduce it to 0/5 and change revision to unchanged. response-04 could rise to 1/5 under a looser contextual interpretation of common-cluster/conflict meaning. Neither interpretation creates a full pass, citation coverage for conflict, or correct stopping. All reasons and exact excerpts are retained for the second reviewer.
2. **Evidence scope:** Most statements that rollback is absent are explicitly limited to supplied evidence. Their emitted snippets do lack rollback, so they are not automatically counted as corpus-wide falsehoods; they still lose completeness against known gold. response-02's final sentence has a pronoun-scope ambiguity. Carrying forward the narrow evidence-subset scope would remove its unsupported extra without changing recall or stopping.
3. **Practical blinding:** No mapping, implementation, application review, raw provider record, normalized output, or heldout file was read. Opaque labels cannot hide all output-structure clues. No system labels were guessed and no outcome preference was used.
4. **Normalization:** No suspected defect is evident from the permitted packet. Diagnostic candidate strings, where present, match delivered final text; they were not used to populate any empty final. Operational labels and actual revision occurrence cannot be verified from these blinded materials alone.
5. **Metrics:** Citation validity measures byte correctness only. Unsupported-claim and stopping judgments are independent of application disposition. No pooled accuracy, precision denominator, latency, cost, usage, or operational-reliability inference is supplied. This is four synthetic development questions with two opaque outputs each, not a heldout or production result.
