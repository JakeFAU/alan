# Independent blind development revision3 grading

Eight opaque outputs were graded against the frozen development gold and exact corpus bytes. Initial and delivered final answers are separate. All eight corpus hashes/sizes matched the manifest, and all emitted citation spans/version hashes were byte-valid. No system labels, application reviewer verdicts, older grades, raw provider records or heldout material were inspected. Answer structure limits practical blinding.

| Opaque ID | Case | Initial supported groups | Delivered supported groups | Delivered supported-fact citation coverage | Strict stopping | Initial → delivered |
|---|---|---:|---:|---:|---|---|
| response-01 | dev-04 | 3/5 | 4/5 | 4/4 | fail | improved |
| response-02 | dev-03 | 1/3 | 0/3 | unavailable (no supported facts) | fail | regressed |
| response-03 | dev-01 | 2/4 | 0/4 | unavailable (no supported facts) | fail | regressed |
| response-04 | dev-04 | 1/5 | 1/5 | 1/1 | fail | unchanged |
| response-05 | dev-01 | 2/4 | 2/4 | 2/2 | fail | unchanged |
| response-06 | dev-03 | 1/3 | 1/3 | 1/1 | fail | unchanged |
| response-07 | dev-02 | 3/5 | 3/5 | 3/3 | fail | unchanged |
| response-08 | dev-02 | 5/5 | 5/5 | 5/5 | pass | unchanged |

These denominators are heterogeneous frozen fact groups. They are not uniformly weighted atomic facts and are not pooled into an overall recall score. Exact answer text, partial-group reasons, assertion lists and source hashes are in `grades-blind.json`. Valid citation bytes do not establish complete claim coverage.

- **response-01:** Improves from operations-only partial knowledge to both conflicting reports plus exact agreed rollback. Full conflict group remains missing because equal authority/common-context/non-supersession semantics are omitted; the authority header is also uncited. Strict stopping fails because authoritative reconciliation is not requested. This is the main stopping adjudication question: practical withholding and rollback are correct. The final has 11 separately enumerated supported assertions, giving claim precision 11/11; that does not repair missing grouped semantics.
- **response-02:** Delivered final is exactly empty despite four valid citations. Initial preserves unknown approval but omits known rollback. Diagnostic-only text includes index 4 snapshot and query failures strictly above 1% for three consecutive minutes, so all three required groups are semantically present there. Its extra assertion about no exact version/build identifier is ambiguous: it may mean a further snapshot identifier, or may misleadingly deny the exact index 4 version already given. Diagnostic precision is null and provenance is unavailable; no delivered improvement is awarded.
- **response-03:** Delivered final is exactly empty despite four valid citations. Initial answers compatibility only. Diagnostic-only text fully supplies both kernels and restoration to agent 3.1 after agent errors strictly above 2% for five consecutive minutes. Its additional distinction between trigger and unspecified completion deadline is supported. Diagnostic claim precision is 8/8; diagnostic citation coverage remains unavailable.
- **response-04:** Operations readiness is accurately attributed, with no false unified verdict. Database failure, authority conflict and agreed rollback are omitted. The supplied-report limitation may be true about retrieval and is not treated as a corpus-absence hallucination.
- **response-05:** Both kernel facts are supported and cited; exact rollback target/trigger are omitted. The supplied-evidence limitation does not satisfy the supported-case stopping rule.
- **response-06:** Unknown approval and absence of signed approval/rejection are correct and cited. Known index 4 rollback and its exact trigger are omitted, so stopping fails.
- **response-07:** Both conditional regional limits and need for region context are correct/cited. This is a conditional answer, not clarification alone; both common rollback groups therefore apply and are missing.
- **response-08:** All five required groups are supported and cited, stopping is correct, and six separately enumerated substantive assertions are supported (precision 6/6). Identical initial/final text needs no revision.

For the scoped retrieval-absence statements in initial response-01/02/03 and response-04/05/06/07, the exact complete supplied context is not available in the blind packet. Emitted citations omit the facts, but do not prove that no additional excerpt was supplied. Their claim precision is therefore null with explicit claim lists/reasons; a gold-group count is never used as a substitute denominator. No unsupported extra claim is established in the delivered answers under the explicit scoped interpretation, while completeness failures remain.

Diagnostic content is labeled unaccepted and cannot replace an empty delivered answer. No explicit diagnostic citation association exists, including when a diagnostic string equals delivered text. Diagnostic citation validity/coverage remain null, with no associations inferred from final citation arrays.

A second independent reviewer should adjudicate the strict response-01 authority-context/stopping interpretation and response-02 diagnostic build caveat. No gold edits, application changes, network calls, provider requests or commits were made.
