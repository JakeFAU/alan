# Phase 1K independent available-artifact correspondence audit

Result: **partial source/artifact correspondence established by inspection;
complete J2 correspondence unestablished because the frozen peer does not
compile.** The K1 model's bounded ACK-source-lifetime counterexample remains a
model result. Neither a complete compiled J2 composition nor its public-driver
or automatic-runner trace is supplied by the available artifacts.

This audit began only after the corrected reference source freeze and inspected
new J2 root artifacts for the first time. The corrected freeze is
`revision2/reference-freeze.json`, SHA256
`7ef775362562faa6b7ac1c4a364638545160be57b75daed81577b5417e0e7868`.
All six frozen model/oracle source hashes and all 26 reused join/substitution
Go-source hashes still match. No model source, API, transition, census, test,
production file, manuscript or historical evidence was changed in this audit.
Only this note was written. No Go command, compile, runtime execution, new A/B
census, nested agent or TeX command was run.

## Frozen inputs and actual availability

All 16 entries of `../context/frozen/sha256.json` were independently hashed and
verified. That manifest's SHA256 is
`aabc70a53bc69a0e4f43b81dbf4745b975e68a2ac03f24d626f1cb0b756efb8b`.
`availability.json` explicitly records compiled_j2_peer=false and
compiled_j2_provision=false; its SHA256 is
`258425592a54314ef399745cbd1b7f09df59c38e122345beef562bdd83bc8856`.
The directory contains actual roots, original candidate/S templates and original
J1 peer programs; it contains no compiled J2 peer. An original J1 peer must not
be substituted for the failed J2 peer.

| Frozen artifact | SHA256 |
|---|---|
| a-j2-root-program.json | 8be92ba120c345154c0f741e71a27d9f305b66a447dc8c91844c1614d8a5c8f2 |
| b-j2-root-program.json | 59973862fd07b0fa61d2051eea697addf6f3d0cd9e60cc537ebd1a8bf8b69884 |
| a-candidate-template.json | db5453b6e08abe5ecbf9614d472626c228675ba197abdaa36cad4aa3c5b04409 |
| b-candidate-template.json | 8693f675a4ec4be6d14a859157fbbf9d21b3608e4485d32b900421c0d7d7b5aa |
| a-s-template.json and b-s-template.json | 786e6c6a2122962469a1e263c57d30b81481faeafdbe9f395d0c669763874241 |
| a-original-j1-peer-program.json and b-original-j1-peer-program.json | 6a77bf8322f8bf227e7c78597d6e753d8d9390f3c981fec7f2bdc59814ea71ef |
| a-peer-error.txt and b-peer-error.txt | 11d4ae33a11c8913a287ceec4e11f141fdbb83a75165a058d443a78189b87193 |

The candidate/S template and original peer bytes equal their Phase1J
`evidence/context/artifacts/join-{a,b}-*` counterparts exactly. The available
manifests describe original J1 resource roots/catalogues and fixed_runs=2; these
are inherited input/topology evidence, not successful J2 provisioning evidence.

## Available root mapping

Both actual J2 roots have 29 retained instructions and alan/1g/v1 semantics.
Their first two instructions are structurally identical to the corresponding
frozen J1 root. All new-root branches and jumps point forward. The mapping below
uses zero-based program counters, not runtime history sequence numbers.

| Actual root PC | Instruction/fact | Independent model correspondence and limit |
|---|---|---|
| 0 | Original candidate Invoke into pending | unchanged candidate invoke_commit |
| 1 | Original AwaitInvocation matching pending; bind status/reason/presence/response | unchanged candidate outcome_consume; no root terminal at this point |
| 2 | Ordinary report Receive from peer/generation1/phase1k_s_report | report_consume after candidate consumption; exact seven report fields bound |
| 3–4 | Compare report request id, then forward jump | finite pure correlation stutter on primary valid input |
| 5–6 | Additional report Receive, then Failure | wrong-request branch; no second report in the declared profile, outside the positive census |
| 7 | Ordinary ACK Send to peer/generation1/phase1k_s_ack | ack_commit; payload uses the actually bound join_request_id and join_s_operation |
| 8–11 | Compute complement from request_value | finite pure stutter; no communication/admission/effect |
| 12–19 | Candidate succeeded/present/id/complement; S succeeded/present/id/zero checks | unchanged J1 final predicate, finite pure stutter |
| 20 | Success | root_terminal succeeded |
| 21–28 | Failure arms | known-negative/mismatch decisions; actual shared-scope failure behavior is not the positive-answer model product |

The complete 21-instruction suffix at new PCs8..28 equals old J1 PCs7..27 after
subtracting one from its branch/jump destinations. This was checked structurally
against the frozen JSON, not inferred from constructor names. Root report Receive
retains the same J1 payload binding/schema with the new obligation. ACK has exactly
the two frozen fields; it is sent before the success predicate and thus asserts
report consumption even on known-negative branches. It binds no message receipt.

For id1/value0 and valid successful consumed outcomes, the actual root suffix
contains a finite path from ACK commitment to genuine local Success with no
intervening receive or required dispatch. Thus the model's immediate abstract
root-terminal eligibility corresponds to these finite pure steps at the root
fragment. Protocol service may be interleaved before/after those steps under the
independent model profile. This source argument establishes neither a particular
automatic-runner schedule nor completion of a composition with the absent peer.

## Reused candidate and S correspondence

The 13 A, 16 B-entry and 7 B-helper instructions embedded in the current frozen
templates were decoded independently. Their program-byte digests and all 36
instruction operands match the previously retained Phase1I mapping exactly:
`docs/foundational/phase1i/evidence/model/artifact-correspondence.json`, SHA256
`c2ac40dbb2477786e5b8eb79cf7636993bf35a4553899ccf700fdbedaa44a3`.
The reused transcription is `research/substitution/candidates.go`, SHA256
`670d0265608ff7c519b6838bd65485427f231591516aaa6f75d8ad42566ade69`.

| Actual fragment | Mapping inspected |
|---|---|
| A PCs0,3/5,6/8/10 | Request consumption, actual id/complement reply operands, terminal outcomes |
| A PCs1/2/4/7/9 | Explicit private branch/jump steps; answer/value0 follows 0→1→2→3→4→6 |
| A PCs11/12 | Original pending receive; trailing terminal unreachable with no second request |
| B entry PCs0/1/2/6/7 | Request consumption, one helper Invoke, actual matching helper outcome, exact helper reply fields, Success |
| B entry PCs3/4/5/8/10 | Explicit private status/presence/mode branches; positive path 0→1→2→3→4→5→6→7 |
| B entry PCs9/11/14/15 | Retained negative/absent terminal arms; not exercised by this audit |
| B entry PCs12/13 | Original pending receive and unreachable trailing failure without another request |
| B helper PCs0..6 | Request; value branch; complement Set/jump; actual id/complement reply; Success. Primary value0 path 0→1→4→5→6 |
| Original S PCs0/1/2 | Request id consumption; reply same id and literal value0; Success |

The extra helper choice/ticket exclusion in the K1 reference is the already
reviewed abstraction; A selects local and B selects helper through the independently
transcribed fragments. No B-derived surrogate replaced K1. These inherited byte
and operand correspondences permit reuse of those primitives, but do not create
a new compiled J2 peer or justify claiming actual A/B J2 executions. No additional
A/B graph census was run merely to populate a missing qualification cell.

The original J1 peer's available 32 Sets, S Invoke and matching Await are intact.
The attempted J2 source preserves their constructor sequence. Because no J2 peer
artifact is returned, this is source/prefix-intent correspondence only: it cannot
be reported as a verified 34-instruction prefix of a compiled J2 peer.

## Exact obstruction and compiler cause

The preserved error is:

```
steps/37:if/then/0:if: branch type mismatch: string and request:invocation:peer
```

`examples/communication/deployment.go:185–188` attempts the required comparison
of `ack_s_operation` from ordinary ACK Receive with the original `peer_pending`
Invoke handle. Source inspection supports the reported diagnosis:

- `author/compile.go:734` types an emitted invocation handle as
  request:invocation:peer. `Set` uses ordinary read/write and preserves that type.
- ACK schema s_operation is string; Receive binds it with that schema's scalar
  type. It supplies no static invocation-declaration proof.
- `readAs` at compile.go:531 allows a request handle to serialize as string at a
  typed payload/list boundary. This permits the truthful report Send.
- The branch checker at compile.go:675–686 does not have such a string target
  schema for these two scalar operands. `compiler.kind` at 862–869 records only
  list descriptors in c.fields. The comparison therefore sees different types
  and rejects. Swapping the operands or copying the handle with Set does not
  provide a schema conversion in this inspected rule.

Compiler SHA256: `45e5ede143a60159d3beeb970c2e17adf1989f5c788fb4107e506034fbf58307`.
Builder SHA256 recorded by the context freeze:
`7f0cdabf070fb1898613fe178a28f449d1def11a098ab09576085dc2d7705e89`.
The context worker wrapped source lines while this audit was reading; the current
builder SHA256 is `b0f68dd7ad34c193251cfe9a7d2450d907d91e3b9fd3d3661025378f677d671e`.
The line reference above uses this latter source. Inspection of the complete diff
against the preserved first attempt shows the disclosed empty-Provision error
handling/comment correction plus line wrapping, with the compared expressions
and protocol constructor sequence unchanged. The frozen artifact hashes did not
change. This audit does not silently substitute the later source identity for
the source recorded by the original context-freeze command.
Preserved first-attempt builder SHA256:
`51ffabbcaa1eacd24537ef05d6dfcdb51cf1931b835820dde2f4f695b03fbd2a`.
The current builder clears the old J1 Provision before constructing J2 and returns
the error before Application.Provision. A failed attempt consequently cannot be
mistaken for a runnable old J1 deployment.

Concrete compilation is attributed to the context workstream, not rerun here:
`checks/first-green.json` records exit1 against the attempted source (its filename
is not evidence of success); `checks/context-obstruction-freeze.json` records
exit0 for the expected-obstruction regression/export command. That command's
manifest SHA256 is `82af3aea6f633969a008e87667937b871500dd2242150b2c58a3fb675f26e864`.
Its passing Policy subtests require the same compile rejection; they are not
passing execution tests of answer, failure, pending or wrong-correlation branches.
This distinction is explicit in the inspected test source.

## Scope of alternatives and unresolved correspondence

The obstruction is honestly scoped to the frozen exact-handle authoring attempt.
It is not proof that all public author programs or raw retained artifacts are
unable to compare serialized handles. Existing public list operations already
supply a different path: `Append` uses readAs against a string element and
`ForEach` binds that element back as a plain string. The existing
`author/retained_handles_test.go:8` exercises a serialized handle through a list;
`author/retained_handles_runtime_test.go` separately checks authentic invocation
outcome matching after such serialization. These tests were inspected, not rerun.
They do not directly qualify a J2 ACK equality implementation.

A list serialization variant would require new collection/control state and an
inspected new suffix/mapping. Conventional Collections initialization in
compile.go:88–115 inserts an instruction before the declared Steps, changing the
frozen 32-Set/Invoke/Await prefix; supplying another input/schema instead changes
the provisioned contract. Such a protocol can be studied under a new freeze but
is not this frozen artifact. Rebinding over peer_pending would lose the original
comparison basis; dropping the S-handle check would weaken required correlation.
Additional message round trips or a lower-level raw peer artifact are likewise
different authored artifacts/protocol scope. No such alternative was implemented
or declared impossible here, and no production compiler change was attempted.

The model makes the intended exact handle equality available abstractly; the
attempted author program cannot currently realize it in this frozen form. The
model's four maxima and ACK-source_closed trace remain conditional bounded
protocol results. Its peer ACK-consume/Success transition has no compiled J2
counterpart to inspect, so full instruction/stutter correspondence, J2 resource
spending in a real execution, and both decisive J2 public-driver traces remain
unestablished. The available root alone cannot discharge those obligations.

This typing obstruction is separate from the intended communication-lifetime
question and from program-visible authorization receipts. Fixing or bypassing
the comparison would not, by itself, remove the modeled ACK authorization race.
The next narrow research choice is whether to study a newly frozen public
serialization variant or the exact scalar-handle authoring boundary, while
retaining the existing model result and its qualification.

## Read-only audit checks actually performed

Used rg/cat/sed to inspect the listed sources and prior mappings. Python3 with
standard-library json/base64/hashlib decoded frozen files and asserted all16
context hashes, exact candidate/S/original-peer equality to J1, exact root
Invoke/Await prefixes, forward root control, the 21-instruction relocated suffix,
all36 inherited candidate/helper instruction mappings and both frozen model
source sets. These checks exited0. No compiled source execution or model
exploration was performed. A `git diff --no-index` against the preserved attempted
builder exited1 for the disclosed error-handling/comments/formatting differences;
that is an inspected diff, not a failing semantic test. The inspected-byte identities above make this a
source/artifact audit, not an additional runtime or compiler qualification run.
