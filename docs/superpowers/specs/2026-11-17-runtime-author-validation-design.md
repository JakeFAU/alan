# Runtime author validation port

The user approved the delegated Weather extension and relevant research ports.
This separately reviewed port carries the compiler correction from research
commit `28cf2e28236026e3ea00fc43e0ee3367de7e7ae2` into the runtime. Research main
is `4a4a7026aead71470111f51e64650ab988ef174e`; runtime starting main is
`5fdfa72c34235ea2b09f969b06dfcdd7c281aa7d`. The compiler branch will be stacked
on the reviewed scheduler port, which changes no compiler behavior.

## Contract

- Newly authored ordinary strings must fit every reachable tracked alternative
  in the destination's byte bound and enum; partial overlap is insufficient.
- Branch joins preserve alternatives and tracked correlations. Invalid adapter
  replies contribute typed defaults until an unchanged validity guard rules
  them out. Collection elements and loop-carried values preserve the originating
  correction's conservative containment rules.
- `AwaitInvocation` requires original provenance from an Invoke of the same
  declaration, including copies and retained collection/snapshot flow. A raw
  string, overwritten handle, wrong declaration or loop-laundered value cannot
  acquire this authority merely by having compatible text.
- Runtime checks remain authoritative for actual values, identities and
  resource bounds. Stronger author checking does not erase tests of runtime
  defenses or reclassify historical admitted artifacts as new author input.
- Existing retained artifacts execute/replay under their recorded bytes and
  semantic versions. No new storage, compiler tag or program semantic version
  is introduced merely for stricter admission of newly authored definitions.
- Do not copy unrelated research models or composition-v2/storage work.

## Port and compatibility evidence

Adapt the research compiler, its bound/provenance helpers and meaningful
regressions to the runtime fork. Include narrow corrections to newly authored
example definitions only where the stronger checker exposes an actual unsound
flow; document any such adaptation. Preserve historical invalid-program
runtime guards with source-bound prechange artifact fixtures rather than
removing the tests because the compiler now refuses their builders.

Before replacing production compiler code, reproduce failed containment and
handle-provenance cases on the old compiler. Keep the originating fixture bytes
and identify their source SHA. Test widening/join, Unicode byte bounds, invalid
reply defaults, loop/collection lifetime and wrong invocation declaration.
Verify valid copies, guarded replies and correct invocation handles continue
to compile and execute. Test that the existing outing programs retain their
artifacts and can still inspect/replay Stores created by the parent revision.
If source-equivalent valid artifacts change or old Store inspection breaks,
report the exact cause and correct compatibility explicitly before proceeding.

The receiving self-review exposed a known-empty loop regression in the
originating flow pass: skipping an infeasible lexical body can omit a nested
loop's required collection annotation and reject a previously valid artifact.
Preserve structural validation and annotations for such bodies while excluding
their infeasible edges from reachable string/provenance proofs. Keep the
minimal prechange/ported witness and require exact prior artifact bytes where
the authored program is unchanged. This is an acceptance compatibility issue;
it does not itself establish an unsound accepted program.

## Delivery

Document the research source, RED/GREEN, receiving-repository checks and limits
in `docs/research-ports/author-validation.md`. Deliver a narrowly scoped stacked
PR, with no automatic merge. The delegated Weather app is a later branch.
The manuscript already states these corrected authoring constraints; this is
receiving-runtime qualification, not a new theorem or evidence that every
compiler acceptance implies eventual completion.
