# Independent heldout grading

Graded all eight opaque outputs against the four published heldout cases and all exact frozen corpus bytes. No system mapping, source code, application review verdicts, raw/normalized provider records or earlier study grades were used. Opaque IDs provide practical blinding only: answer and citation structure can suggest system differences.

| Opaque output | Case | Final supported / applicable groups | Final citation-covered supported groups | Final stopping | Initial supported groups | Change |
|---|---|---:|---:|---|---:|---|
| response-01 | held-03 | 3/3 | 3/3 | pass, adjudicate next-action wording | 1/3 | improved |
| response-02 | held-03 | 1/3 | 1/1 | fail | 1/3 | unchanged |
| response-03 | held-04 | 1/5 | 1/1 | fail | 1/5 | unchanged |
| response-04 | held-02 | 3/5 | 3/3 | fail | 3/5 | unchanged |
| response-05 | held-01 | 4/4 | 4/4 | pass | 4/4 | unchanged |
| response-06 | held-01 | 4/4 | 4/4 | pass | 4/4 | improved |
| response-07 | held-04 | 0/5 | unavailable: no supported facts | fail, empty delivery | 1/5 | regressed |
| response-08 | held-02 | 5/5 | 5/5 | pass | 5/5 | unchanged |

These denominators are whole gold groups of unequal semantic size, not uniformly weighted atomic facts. No pooled recall is claimed. No delivered required fact is graded contradicted; the deficient answers omit required content or narrow their claims to incomplete evidence.

All 18 final citations and 12 initial citations pass document ID, SHA-256 version, bounds and exact UTF-8 byte validation. Provenance validity does not establish coverage. response-06 includes a duplicate Maple line and an irrelevant Birch rollback span; neither adds coverage. response-01 repeats its rollback span. response-07 has four valid final references but no delivered text.

response-01 repairs the missing Birch rollback while correctly retaining approval uncertainty. Its final has all required facts. The primary stopping grade passes the substantive withholding-plus-known-facts rule; adjudicate whether gold's instruction to obtain the signed decision requires an explicit next-action request, which is absent.

response-02 correctly says Birch approval is unestablished but leaves out exporter 6 and the strict >2% delivery-failure trigger for eight consecutive minutes. Its claim about supplied evidence can be accurate of the emitted approval excerpt; it does not establish corpus absence or completeness.

response-03 accurately attributes the service pass/ready statement. It does not clearly assert an uncontested ready verdict. It omits the reliability fail/not-ready report, the unresolved equal-authority conflict, and the agreed cache 3 rollback at strict >5% errors for two consecutive minutes. This incomplete attribution is insufficient stopping.

response-04 gives correct fast/bulk queue branches and the missing-profile condition. It is a semantic conditional answer, so the common worker 8 rollback at strict >4% job failures for six consecutive minutes is required and absent. A targeted clarification alone would have different mode-conditional requirements.

response-05 supplies all Maple facts with strict >3% rejected records for four consecutive minutes and ingestion 2.3. The canonical gold line-2 citation supports these required groups.

response-06 initially states all four Maple facts but unnecessarily withholds P12 applicability although the corpus header establishes it. The final resolves that uncertainty and includes the header. Adjudicate the initial stopping failure if excerpt-scoped caution is treated more permissively; corpus completeness favors failure.

response-07 initially gives the service finding plus part of the authority context, without the reliability finding or rollback. Its conflict-unresolved group is missing, not partially scored: actual disagreement and the complete same-pool scope are not established in the prose. The delivered final is empty and earns 0/5 with incorrect stopping. The diagnostic candidate is retained separately as unaccepted content and never substituted. It is materially more complete, but its citation association is unavailable from this packet; diagnostic provenance remains null. Its omission of explicit same-pool scope is a grouped-fact adjudication point if separately assessed. A clock-time request is unnecessary for this condition-based rollback question.

response-08 is a semantic conditional answer despite its clarification disposition: both queue limits and common exact rollback are present. The profile question conveys the missing choice; the cited queue line establishes that one profile is necessary.

Precision is explicitly separate from completeness. Final response-01 has a complete enumerated set of 10 supported substantive claims (10/10), response-05 5/5, response-06 6/6, and response-08 5/5. These include asserted scope/context and separate rollback target/trigger propositions; they are not gold-fact denominators. response-02, response-03 and response-04 retain null precision: their claims refer to a supplied evidence subset whose complete membership is not present in the blind packets. The JSON lists their substantive claims and this limitation rather than inventing a denominator. Empty response-07 has null precision. Explicitly scoped excerpt absence is not labeled a false whole-corpus claim, but known facts still remain missing. Second review should adjudicate this scope interpretation along with every fact, citation and stopping grade.

Frozen input hashes are recorded in grades-blind.json and rechecked before delivery. No frozen input was altered. Findings concern this bounded four-case synthetic heldout only; no general efficacy, durability, operational cost or production-readiness conclusion follows.
