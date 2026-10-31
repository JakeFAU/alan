# Independent semantic and validator review before relocation

**APPROVE** executing the frozen two-placement experiment. I reviewed the restricted implication and tried to falsify private eligibility using the actual checker and reducer. This verdict covers the source argument under its stated premises. The public relocation records and their interpretation require separate review.

The freeze commit is `8547d919062af6b3dd0d97e1fc7aa86967c78220` on base `7f077fbaee4855b2ecd73ed6fcea344cea022169`. Reviewed SHA-256 values follow.

| File | SHA-256 |
| --- | --- |
| 01-contract.md | c3d24d2e906aa59d7fc84559eaa4d81724d8b8726e469df158e1ba05d27795da |
| 02-transport.md | bc9298dbf7dfe1e71953024733c959446e19fa0b7282006df515f8c90b60ba00 |
| certificate.go | 75edb3f58e1c94928a4f7bcd4f9e867aceb9339fcaf393c4423de6b8324994df |
| operands.go | c279f86260e1e245c9fa3bc11c0c3f8af1c8ac39f780f79937e363715b561657 |
| controls_test.go | 86cce651515c61ee00ae7082e30b5119708b9a36ee7a859de4f8912ed1095ac2 |

**Checker.** I checked both pinned semantic/compiler pairs, program re-encoding, exact descriptor initialization through RetainTemplate, the singleton caller port and direct and transitive declaration-to-schema edges. The complete typed input record leaves no ambient register for a forged initial Handle. Each destination has one sort across its entire lifetime, even across a branch that never ran, an exhausted iterator, or an absent response. Scalar/application list literals use closed data kinds. Only original invocation commitment writes a new Handle. A Ref can copy that provenance without declassification. The lowering and binder rewrite endpoint fields for ordinary messages and the inbound request operation, with no corresponding rewrite of application values. Receive reads payload bindings only. The closed grammar of checked consuming operands covers every dynamic reducer case, including updates read from the pre-state, list snapshot/row bindings, payload validation and success-output checks. I rechecked the final proof's distinction between fault and quiescence against Store.selectReady. A non-receive operand error propagates immediately. For a receive, the store skips ErrIneligible and propagates other reduction errors when trying a recipient's unconsumed observation. Reachable faults require accounting in a public-to-abstract correspondence. A missing application operand cannot establish the designated pending state. Forging an author artifact by unsafe memory modification or a collision in retained digests is outside the public construction and identity assumptions. A local artifact check does not qualify an entire tree.

**Attacks.** I ran the independent scratch test reproduced below. The actual template initializer makes the application input `raw` equal to the original first key spelling in the short-subject run. The ordinary author accepted using that input directly as an Await operation. The checker rejected its operand at PC 0 with `requires handle[helper], got string`. A second candidate overwrites its genuine current handle with another Invoke while saving the old current register in a simultaneous update. The checker accepted the transitive template. On subjects of lengths 1 and 128, Reduce took identical successor PCs `[1 2 3 4 5 7 10 11 11]`. The old alias compared equal, and the overwritten current handle compared distinct. Changing the shared MaxDataBytes to 179 allowed only the short subject's first reduction. The first successful post-projections under adequate budgets required 179 and 306 bytes respectively. This is a counterexample to removing representation-size adequacy from the private claim. These were direct reductions without delivering any synthetic observation. Independent public controls exercised real selected child envelopes at two roots.

**Transport and service.** I checked the projection against P's fixed-frame contract and Q's interface and one-relay result. The original-grounding hypothesis covers the actual reachable private instructions and maximal paths of the same bytes or an explicitly justified correspondence there. It does not demand prior conformance of the relocated candidate or infer original W/Progress for B from a finite public run. The growing injection must survive alias deletion, preserve same-declaration freshness and keep literal data fixed. The root resource guard and the immediate caller route inspect separate roles. Actual accepted slots, the original S debit, a valid parent catalogue, bound late reply disposition and fully serviced original selections each remain necessary. At an infinite private projection the original weak-service reasoning retains even private silent steps. For a finite projection, an unserviced original action either persists directly or has a finite chain of persistent entry, S-gate or first-decision/admission prerequisites. Distinguishing these prevents an infinite erased suffix of repeated lookups from supplying progress. In pending, S and the chosen helper have already been selected, but e's wait cannot gain an input. Every added ancestor is at its own unmatched wait. In the two diagnosed failures, completed original selections settle descendants before first J publication independently of the weaker failed-freeze publication guard.

The proof covers either specified reading on all sixteen original requests and each finite legal, representable unary embedding, conditional on the behavioral, original-correspondence, frame, service and operational-size premises. The original unrestricted relocation and ARG-2 obligations remain open. I found no accepted representative-sensitive step under these premises. The tests below exercise finite instances and the budget boundary.

## Independent verification

Darwin arm64, Go 1.27.3. I used the following environment from the phase worktree. The separately selected integration controls exclude the gated relocation.

```sh
export GOTOOLCHAIN=local CGO_ENABLED=1 GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly GOCACHE=/private/tmp/alan-phase1q-go-cache
go test -count=1 -timeout=2m -v ./research/nominal
go test -race -count=1 -timeout=2m ./research/nominal
go test -tags=integration -count=1 -timeout=4m -run '^(TestNominalAliasFreshExecution|TestNominalWrongAnswerCertifies|TestDiagnosticRelayHandleObservation)$' ./internal/substitutiontest
```

All three commands reported PASS or ok. For the scratch test, save the Go block below in a temporary file. From the worktree root, point a Go overlay at it and run the exact test. This uses the ordinary helper constructors already in the nominal test package. The accepted program ID was `68b714e8596f259afa94d90bdf13cf869602e271356d9ad1fa5ad0c562b725ea`. Its template ID was `2bffb95a820e4b4f105782dd8d182a55e746b7b9ccc6f3f8406a765fdb1bb6a9`, certificate digest `169d85d87f41eda772967fb80e1510ffd6b74043f370eb4b099d54844eaa8d92`. The scratch test passed.

```sh
# Save the Go block as /private/tmp/phase1r-gate-probe/independent_test.go.
python3 - <<'PY'
import json, os
path = '/private/tmp/phase1r-gate-probe'
with open(path + '/overlay.json', 'w') as output:
    json.dump({'Replace': {os.getcwd() + '/research/nominal/gate_external_test.go': path + '/independent_test.go'}}, output)
PY
go test -overlay=/private/tmp/phase1r-gate-probe/overlay.json -run '^TestIndependentGate$' -count=1 -timeout=2m -v ./research/nominal
```

```go
package nominal_test

import (
	"encoding/json"
	"errors"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/research/nominal"
	"github.com/JakeFAU/alan/schema"
	"strings"
	"testing"
)

func TestIndependentGate(t *testing.T) {
	helper := echo(t, "int")
	inputs := schema.Record{"raw": {Kind: "string", MaxBytes: 300}}
	root := func(steps ...author.Step) author.Artifact {
		return compiled(t, author.Definition{Inputs: inputs, Invocations: map[string]author.Template{"helper": helper}, Ports: publicPort(schema.Field{Kind: "int"}), Steps: append(steps, author.Success())})
	}
	forged := root(await("helper", "raw", "data"))
	if _, err := nominal.CheckArtifact(forged); err == nil {
		t.Fatal("initializer forged a handle")
	} else {
		t.Logf("forged input refused: %v", err)
	}
	empty := map[string]author.Expr{"value": author.Integer(1)}
	steps := []author.Step{author.Invoke("current", "helper", empty, nil), author.Set("first", author.Var("current")), author.Invoke("current", "helper", empty, map[string]author.Expr{"saved": author.Var("current")}),
		author.IfEqual(author.Var("saved"), author.Var("first"), []author.Step{author.Set("same", author.Boolean(true))}, []author.Step{author.Set("same", author.Boolean(false))}),
		author.IfEqual(author.Var("saved"), author.Var("current"), []author.Step{author.Set("fresh", author.Boolean(false))}, []author.Step{author.Set("fresh", author.Boolean(true))})}
	art := root(steps...)
	init := map[string]schema.Value{"raw": schema.Text(`["x",1,1,0]`)}
	tmpl := template(t, art, init)
	cert, err := nominal.CheckTemplate(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("accepted program %s template %s digest %s", art.ID(), cert.ID, cert.Digest)
	var p executor.Program
	if err := json.Unmarshal(art.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	var sizes []int
	for _, sub := range []string{"x", strings.Repeat("long", 32)} {
		st := executor.Projection{PC: 0, Status: "open", Data: init}
		var path []int
		for seq := int64(1); st.Status == "open"; seq++ {
			red, err := executor.Reduce(p, st, nil, executor.RunID{Subject: sub, Generation: 1}, seq, executor.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			st = red.Projection
			path = append(path, st.PC)
			if seq == 1 {
				b, _ := json.Marshal(st)
				sizes = append(sizes, len(b))
			}
		}
		if st.Data["same"] != schema.Bool(true) || st.Data["fresh"] != schema.Bool(true) {
			t.Fatal(st)
		}
		t.Logf("subject bytes %d path %v", len(sub), path)
	}
	l := executor.DefaultLimits()
	l.MaxDataBytes = sizes[0]
	for i, sub := range []string{"x", strings.Repeat("long", 32)} {
		_, err := executor.Reduce(p, executor.Projection{PC: 0, Status: "open", Data: init}, nil, executor.RunID{Subject: sub, Generation: 1}, 1, l)
		if (i == 0 && err != nil) || (i == 1 && !errors.Is(err, executor.ErrLimit)) {
			t.Fatal(err)
		}
	}
	t.Logf("first postcommit projection sizes %v, tight common limit %d", sizes, l.MaxDataBytes)
}
```
