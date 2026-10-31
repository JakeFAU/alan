# Phase 1O Task 3 — independent review R1

Spec: **REJECT** (implementation adherence to frozen `03-finite-contract.md`).

Implementation/certificate quality: **REJECT**.

These are Task 3 acceptance decisions only. The frozen contract need not be strengthened to accommodate the implementation. This review does not decide the whole phase, manuscript/04 changes, general P_boundary ARG-2, arbitrary candidate conformance, runtime correspondence, or relation nonexistence.

Reviewed base/current HEAD: `ffd00be531fbb986b86160bd6c22a87345fee14a`. Packaged diff: `.superpowers/sdd/00-plan-phase1o/task3-r1-diff.txt`, SHA-256 `f0ed02c228786356d21c70d0b2aa48c5aa043a3292df3258e7389cd39bfc791a`. Frozen03 SHA-256: `a41c2216caa80d979365e61470dce6fc03c2616ad6f2faeab2e2431b0418a083`. All 37 explicitly packaged files matched their listed sizes/hashes: seven additive Go files, frozen03, the evidence index, and 28 compressed records. No checkout source/test mutation or suite rerun occurred.

## Required corrections

1. **[P1] Nil grammar bypasses every transfer/readiness obligation.** `research/substitution/weak_refinement.go:273–275`, reached through `checkWeakCertificate` at lines 69–70. The shared indexer treats every edge rule as present when `rules == nil`, to support the rank-only caller. The certificate checker accepts this bypass: each lookup then has empty Owner, so neither the component nor environment loop checks the edge. A pairs-only certificate for X `x0 -a-> x1`, K's sole initial `k0`, P(x0)=P(k0)=0, P(x1)=1, and R={(x0,k0)} returns `accepted=true`, no issues, one checked pair, zero component/environment obligations. This is malformed-input acceptance, not a semantic rejected witness or an existence result. It violates frozen03 §§1/1.2's declared grammar, ownership and every-outgoing-edge checks.

   Remedy: separate structural graph indexing from rule validation, or disallow the absent grammar in certificate/cycle validation while preserving the intentional structural-only rank path. Every referenced edge rule must resolve to a valid owner before semantic assessment. Add cases for nil grammar, empty nonnil grammar, and a missing rule, with pairs-only certificates and component/environment edges; each must produce Input errors before acceptance. Preserve rank tests and valid explicitly empty-alphabet graphs if intentionally supported.

2. **[P2] `silent_demand` strengthens the frozen weak-path strong subset.** `research/substitution/weak_refinement.go:190–191`; the extra expectation is embedded in `weak_refinement_normalization_test.go:266`. Frozen03:38 and :374 specify endpoint demand equality and preservation of current guard dependencies on unmatched silent reference steps. Frozen01:355–358 and reviewed02:92–110 distinguish these: lawful `choose_local` decreases future demand while preserving current q/closure/source guards. Direct semantic matches may change their own state; that does not prohibit every demand-changing unmatched semantic choice.

   Concrete machinery fixture: X `x0 -a-> x1`; K `k0 -choose_local(empty word)-> kL -a-> k1`; R={(x0,k0),(x1,k1)}; matching path [choose_local,a], with a identified as Direct. P is 0 at x0/k0/kL and 1 at x1/k1. Demand is (2,2,0) at x0/k0 and (1,1,0) at x1/kL/k1. Every current guard is the same `g`. The X coarse edge's declared account excludes h and performs the visible action; K's intermediate exclusion is lawful semantic bookkeeping, not genuine private calculation. Related endpoints have equal demand, one exact visible occurrence is matched, and current guards are unchanged. The implemented subset nevertheless reports `silent_demand` at the choice. This probe is a finite machinery instance, not a newly established A_op/A_pub candidate or full Account_D certificate.

   Remedy: remove demand preservation as a requirement on all unmatched silent semantic reference actions. Keep endpoint demand equality and unmatched current-guard preservation; add separate genuine-private preservation checks as required by finding 3. Add the positive case above and retain a negative case where the silent choice changes a selected current guard. R_w must still fail its eight endpoint-demand inequalities; its four `silent_demand` findings are not failures of the frozen named subset. Adjust the test/report and retain the original 12-failure evidence as historical, rather than retroactively relabeling it. R_g and its actual direct local choice must continue to pass.

3. **[P2] `Direct` exempts genuine private work from strong public-field preservation.** `research/substitution/weak_refinement.go:108,181–191,206–208`. Direct validation checks only matching nonempty Semantic strings and words; it does not distinguish actual semantic choice from `Private=true`. Candidate/reference private checks inspect only Projection, while Direct skips selected Guards/Demand preservation. X and K can each have one Private=true edge with Semantic=`private-calc`, empty word, unchanged Projection, and identical endpoint changes in Guards (`before`→`after`) and Demand ((2,2,0)→(1,1,0)); relate matching endpoints and mark the K edge Direct. The checker returns no Input, Transfer, Readiness or Strong issues. Endpoint equality cannot make genuine private mutation public-fact preserving. Frozen03:34 and frozen01:234–240 require genuine private work to preserve required public facts; reviewed02:187–192 excludes accounting updates from pure private work.

   Remedy: check every genuine candidate private edge and traversed genuine reference private edge against all selected public bookkeeping/guard fields, independently of Direct. A shared Semantic string cannot license private work to perform an actual bookkeeping action. Restrict Direct to actual nonprivate semantic matches, or make private preservation unconditional so Direct cannot bypass it. Add guard-only and demand-only mutations on both sides, including reference interiors outside R and the supplied Direct case. Retain constant-private stutters and actual nonprivate matched local choice as positive controls. Removing finding 2's overly broad demand check alone would worsen this gap.

## Supported bounded checks

With an explicit valid grammar, the code checks every independently reachable included original state pair and every outgoing X component edge; reference component alternatives are correctly not simulated in reverse. Initial coverage, nonfunctional relations, alternative explicit paths, ordered word equality/multiplicity, endpoint pairs and immediate bilateral environment counterparts are present. F1 exercises alternative/smaller relations; missing-pair/move and duplicate-label-edge tests retain independent obligations. Malformed path/ID/owner errors are separated from legal wrong-word/endpoint/readiness rejection. No search, arbitrary cap, NONE API, or history-splitting repair was introduced.

The rank checker is a sufficient complete-graph certificate and rejects open-port fragments. The private-cycle check requires a reachable supplied cycle with one sole enabled private edge at each state. F3's negative nondivergence result therefore does not depend on starving a competing output or service. Rank rejection is not claimed to prove divergence. These checks establish no actual K1 all-mode progress.

Normalization uses an independent `weakPrivateState`, not A/B PCs/registers or erasure. Its field-derived spending/ticket map, q/closure guards and K edge/account transcription are checked across all q/closure combinations. The raw-state map retains conflicting annotations over the same m. Branch checks retain no-refund/open accepted-h/closed scope_closed behavior and identify omitted actions as ports. These are Account-subset/boundary-fragment checks; full Account_D, admissibility, full ARG-1 and W remain unassessed.

All six J c/a/d/t schedules are exercised from original CallableJoin successors, including early/late snapshots, failed Ends with present diagnostics, and pending admission before publication. The authorization/binding/admission transcription agrees with the narrowly inspected production `messages.go:103–177`, `invocation.go:123–140,296–312`, and `invocation_outcome.go:164–184` clauses. First records are validated before history reuse; saved repeats retain the first result; revision/generation priority and nonreply source/scope controls are separately qualified. Numeric Authority/closure/binding/schema premises remain supplied inputs. Original successor reuse is transcription/anchor consistency, not independent proof of original semantics, raw runtime provenance, or coherent sidecar extension for arbitrary R. No additional finding is raised against those explicit limits.

## Evidence inspected without suite rerun

Parsed all 28 compressed records; checked compressed SHA-256, argv and exit against the index. Verified every final source hash against current bytes and all final verified-record source manifests against the index. Compared each historical staticcheck-bearing source hash with `git show HEAD:<path>`. Inspected normal/race stdout for branch counts, demand conflict, normalization obligations and the six authorization snapshot/publication results.

- Final and verified normal/race records: exit 0; 23 selected top-level passes, 69 run/pass test+subtest events, no failures/skips. Verified normal/race source manifests match.
- Recorded `go vet ./research/substitution`, `go mod verify`, and `gofmt -d` on all seven new files succeed. Format/vet output is empty; modules reports all verified.
- `staticcheck ./research/substitution` is exit 1: ST1005 at `callable_join.go:290,301,307` and `callable_join_local.go:65`; U1000 at `invocation_join_explorer.go:84` and `invocation_join_oracle.go:61`. Those four files match HEAD. No suppression or new-file diagnostic is present. This is not a green whole-package staticcheck claim.
- RED, corrected and intermediate failed attempts are retained. Final logs reproduce 22 branch states/31 edges; each normalization witness has 16 pairs, 12 component and 24 bilateral environment obligations; the existing R_w run reports 12 strong issues and R_g zero. Finding 2 explains why four require reclassification after correction.

These records support the listed fixture outcomes but do not cover the three defects above.

## Single focused reviewer probe

Executed one isolated Go invocation over the byte-for-byte packaged core and exact existing Request/Observation types, without checkout code/test changes. Complete disposable source/record: `/private/tmp/alan-task3-review-bxy3178v/`, for controller archival. The exact probe is reproduced below to preserve the witnesses independently of that temporary directory.

Command: `/opt/homebrew/bin/go test -v -count=1 -run '^TestTask3ReviewBoundaries$' .`

Working directory: `/private/tmp/alan-task3-review-bxy3178v`.

Environment: inherited environment with every ALAN_* variable and OPENAI_API_KEY removed; overrides `GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/private/tmp/alan-phase1n-model-go-build`. No secret values were logged. `go.mod` declares module `task3review`, Go `1.27`. `types.go` contains unchanged Request/Observation declarations from `research/substitution/oracle.go:7–23`, prefixed with `package substitution`.

Probe source hashes:

- `go.mod`: `20d1eb4163c5d0c909c552ac9635c984aa60be8320688cbf2135d8c2baa4b041`.
- `weak_refinement.go`: `6ae6c988afbe5c694323f12c25fe0206214394078b03962aa63f1afda2d65902`.
- `types.go`: `b6e8c2cb889f918bbfc52a41ec7b431e21f47758c954431d664009dfe33690a2`.
- `review_test.go`: `c997d7e716ad4b7495aa050e6cceb8412967ed0fcbca249042799e0036f58c62`.
- `probe.json`: `9b03a9d13e8a592e2c61851bf4bd34540e6032dc8daab8c7495163737e6bbeb9`.

Executed at 2026-10-30T18:10:50.415357+00:00; exit 1. All three intended-contract assertions fail:

```text
=== RUN   TestTask3ReviewBoundaries
=== RUN   TestTask3ReviewBoundaries/nil-grammar
    review_test.go:8: accepted=true result={Input:[] Transfer:[] Readiness:[] Strong:[] Pairs:1 Components:0 Environments:0}
    review_test.go:9: missing grammar must be input error, not acceptance
=== RUN   TestTask3ReviewBoundaries/silent-choice-endpoint-demand
    review_test.go:14: accepted=true result={Input:[] Transfer:[] Readiness:[] Strong:[{Clause:silent_demand Detail: Pair:{X:x0 K:k0} Edge:L}] Pairs:2 Components:1 Environments:0}
    review_test.go:15: equal endpoint demand and preserved current guards must pass named subset
=== RUN   TestTask3ReviewBoundaries/private-direct-strong-fields
    review_test.go:20: accepted=true result={Input:[] Transfer:[] Readiness:[] Strong:[] Pairs:2 Components:1 Environments:0}
    review_test.go:21: Direct cannot exempt genuine private steps from preserving selected public fields
--- FAIL: TestTask3ReviewBoundaries (0.00s)
    --- FAIL: TestTask3ReviewBoundaries/nil-grammar (0.00s)
    --- FAIL: TestTask3ReviewBoundaries/silent-choice-endpoint-demand (0.00s)
    --- FAIL: TestTask3ReviewBoundaries/private-direct-strong-fields (0.00s)
FAIL
FAIL	task3review	0.211s
FAIL
```

Exact probe:

```go
package substitution
import "testing"
func TestTask3ReviewBoundaries(t *testing.T) {
 node := func(p string, d int, g string) weakNode { return weakNode{Projection:p,Demand:[3]int{d,d,0},Guards:g} }
 t.Run("nil-grammar",func(t *testing.T) {
  p:=weakProblem{X:weakFiniteGraph{Initial:[]string{"x0"},Nodes:map[string]weakNode{"x0":node("0",0,"g"),"x1":node("1",0,"g")},Edges:[]weakEdge{{ID:"A",From:"x0",To:"x1",Rule:"a"}}},K:weakFiniteGraph{Initial:[]string{"k0"},Nodes:map[string]weakNode{"k0":node("0",0,"g")}}}
  r:=checkWeakCertificate(p,weakCertificate{Pairs:[]weakPair{{X:"x0",K:"k0"}}})
  t.Logf("accepted=%t result=%+v",r.accepted(),r)
  if len(r.Input)==0 {t.Error("missing grammar must be input error, not acceptance")}
 })
 t.Run("silent-choice-endpoint-demand",func(t *testing.T) {
  p:=weakProblem{Demand:true,Guards:true,Rules:map[string]weakRule{"local":{Owner:"component",Semantic:"choose_local"},"a":{Owner:"component",Semantic:"reply",Word:[]Observation{{Kind:"a"}}}},X:weakFiniteGraph{Initial:[]string{"x0"},Nodes:map[string]weakNode{"x0":node("0",2,"g"),"x1":node("1",1,"g")},Edges:[]weakEdge{{ID:"A",From:"x0",To:"x1",Rule:"a"}}},K:weakFiniteGraph{Initial:[]string{"k0"},Nodes:map[string]weakNode{"k0":node("0",2,"g"),"kL":node("0",1,"g"),"k1":node("1",1,"g")},Edges:[]weakEdge{{ID:"L",From:"k0",To:"kL",Rule:"local"},{ID:"KA",From:"kL",To:"k1",Rule:"a"}}}}
  r:=checkWeakCertificate(p,weakCertificate{Pairs:[]weakPair{{X:"x0",K:"k0"},{X:"x1",K:"k1"}},Component:[]weakMatch{{Pair:weakPair{X:"x0",K:"k0"},Edge:"A",Path:[]string{"L","KA"},Direct:"KA"}}})
  t.Logf("accepted=%t result=%+v",r.accepted(),r)
  if !r.accepted()||len(r.Strong)!=0 {t.Error("equal endpoint demand and preserved current guards must pass named subset")}
 })
 t.Run("private-direct-strong-fields",func(t *testing.T) {
  p:=weakProblem{Demand:true,Guards:true,Rules:map[string]weakRule{"tau":{Owner:"component",Semantic:"private-calc",Private:true}},X:weakFiniteGraph{Initial:[]string{"x0"},Nodes:map[string]weakNode{"x0":node("0",2,"before"),"x1":node("0",1,"after")},Edges:[]weakEdge{{ID:"T",From:"x0",To:"x1",Rule:"tau"}}},K:weakFiniteGraph{Initial:[]string{"k0"},Nodes:map[string]weakNode{"k0":node("0",2,"before"),"k1":node("0",1,"after")},Edges:[]weakEdge{{ID:"KT",From:"k0",To:"k1",Rule:"tau"}}}}
  r:=checkWeakCertificate(p,weakCertificate{Pairs:[]weakPair{{X:"x0",K:"k0"},{X:"x1",K:"k1"}},Component:[]weakMatch{{Pair:weakPair{X:"x0",K:"k0"},Edge:"T",Path:[]string{"KT"},Direct:"KT"}}})
  t.Logf("accepted=%t result=%+v",r.accepted(),r)
  if len(r.Input)==0&&len(r.Transfer)==0&&len(r.Strong)==0 {t.Error("Direct cannot exempt genuine private steps from preserving selected public fields")}
 })
}
```

Required next gate: correct all three defects; preserve this rejected review and witnesses; record focused normal/race/static/module/format verification against corrected sources; obtain scoped independent rereview. General P_boundary ARG-2 remains OPEN.
