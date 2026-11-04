//go:build integration

package durabletest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Task 1 only constructs new public inputs. No retained record is repaired and
// no candidate authority implementation is used. Branch PCs are code indices;
// service locators count the instructions actually executed on each path.
func bindingConstruction(t *testing.T, p *runtime.Provision, m *delegation.Manifest, requestFirst bool, padding int, branch *bool) {
	t.Helper()
	oldClaim, err := json.Marshal(m.Claim)
	if err != nil {
		t.Fatal(err)
	}
	oldService, oldReply := m.BAService, m.BAReply
	executed := padding
	if branch != nil {
		executed = 3
	}
	locator := func(sequence int) string {
		raw, err := json.Marshal([]any{"a", 1, sequence, 0})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	m.BAService, m.BAReply = locator(4+executed), locator(7+executed)
	m.Claim.ServiceOperationID = m.BAService
	newClaim, err := json.Marshal(m.Claim)
	if err != nil {
		t.Fatal(err)
	}
	literal := func(v schema.Value) map[string]any { return map[string]any{"Literal": v} }
	set := func(v any) map[string]any { return map[string]any{"Op": "set", "Dst": "binding_scratch", "A": v} }
	jump := func(value bool, yes, no int) map[string]any {
		return map[string]any{"Op": "branch", "A": literal(schema.Bool(value)), "B": literal(schema.Bool(true)), "Target": yes, "Otherwise": no}
	}
	var inserted []any
	if branch != nil {
		// Both finite forward paths execute branch, set, branch; five code
		// positions occupy three committed reductions before the same service.
		inserted = []any{jump(*branch, 4, 6), set(literal(schema.Int(1))), jump(true, 8, 8), set(literal(schema.Int(2))), jump(true, 8, 8)}
	} else {
		for i := 0; i < padding; i++ {
			value := any(literal(schema.Int(int64(i))))
			if i%2 == 1 {
				value = map[string]any{"Ref": "binding_scratch"}
			}
			inserted = append(inserted, set(value))
		}
	}
	for _, subject := range []string{"a", "b"} {
		reciprocalEdit(t, p, subject, func(program map[string]any) {
			if subject == "a" {
				code := program["Code"].([]any)
				if requestFirst {
					code[1], code[2] = code[2], code[1]
				}
				for _, raw := range code {
					in := raw.(map[string]any)
					if in["Op"] == "branch" {
						for _, key := range []string{"Target", "Otherwise"} {
							if target := int(in[key].(float64)); target >= 3 {
								in[key] = target + len(inserted)
							}
						}
					}
				}
				code = append(append(append([]any{}, code[:3]...), inserted...), code[3:]...)
				program["Code"] = code
			}
			// Replace all dependent claim copies and literal service/reply
			// references before hashing and Create, including message payloads.
			var replace func(any) any
			replace = func(value any) any {
				switch v := value.(type) {
				case string:
					switch v {
					case string(oldClaim):
						return string(newClaim)
					case oldService:
						return m.BAService
					case oldReply:
						return m.BAReply
					}
				case map[string]any:
					for k, x := range v {
						v[k] = replace(x)
					}
				case []any:
					for i, x := range v {
						v[i] = replace(x)
					}
				}
				return value
			}
			replace(program)
		})
	}
	for _, artifact := range p.Programs {
		hash := sha256.Sum256(artifact.Bytes)
		if hex.EncodeToString(hash[:]) != artifact.ID {
			t.Fatal("actual program hash mismatch")
		}
	}
}

func TestPhase3B4PublicConstruction(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name         string
		requestFirst bool
		padding      int
		branch       *bool
		wantReason   string
	}{
		{"adjacent", false, 0, nil, "authorized"},
		{"request-ready-0", true, 0, nil, "authority_request_not_admitted"},
		{"request-ready-1", true, 1, nil, "authority_request_not_selected"},
		{"request-ready-3", true, 3, nil, "authority_request_not_selected"},
		{"forward-true", true, 0, &yes, "authority_request_not_selected"},
		{"forward-false", true, 0, &no, "authority_request_not_selected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := reciprocalDirectory(t, "construction-"+tc.name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m, err := delegation.Reciprocal(delegation.Options{EndpointA: ua, EndpointB: ub, Scenario: delegation.S2Handshake, TerminalHandshake: true})
			if err != nil {
				t.Fatal(err)
			}
			bindingConstruction(t, &p, &m, tc.requestFirst, tc.padding, tc.branch)
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			baseline3b3Write(t, dir, "construction.json", map[string]any{"RequestFirst": tc.requestFirst, "LocalLength": tc.padding, "Branch": tc.branch, "ExpectedV1Reason": tc.wantReason, "OriginalService": `["a",1,4,0]`, "ActualService": m.BAService, "OriginalReply": `["a",1,7,0]`, "ActualReply": m.BAReply})
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal("public construction", err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			run, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			baseline3b3Write(t, dir, "run.json", map[string]any{"Report": run, "Error": fmt.Sprint(runErr)})
			if runErr != nil {
				t.Fatal(runErr)
			}
			ea, eb := phaseSnapshot(t, pa), phaseSnapshot(t, pb)
			phaseStopEndpoint(t, ca)
			phaseStopEndpoint(t, cb)
			baseline3b3Write(t, dir, "endpoints.json", map[string]any{"a": ea, "b": eb})
			reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
			reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
			snap := reciprocalCapture(t, db, filepath.Join(dir, "final"))
			e, ok := snap.Effect(m.BAService)
			if !ok || e.Authorization == nil || !e.Authorization.Allowed || e.ExternalAttempt == nil || e.ExternalAttempt.Authority == nil {
				t.Fatal("actual service missing", e)
			}
			w := e.ExternalAttempt.Authority
			if w.Reason != tc.wantReason || w.OriginalRequestID != m.BARequest || w.ServiceOperationID != m.BAService || w.Use == nil || w.Use.GrantID != m.Grant || e.Sequence != reciprocalServiceSequence(t, m.BAService) || w.GrantDecisionSequence <= 0 {
				t.Fatal("failure at wrong causal layer or stale service locator", e)
			}
			grant, ok := snap.LifecycleDecision(m.Grant)
			if !ok || !grant.Accepted || len(snap.LifecycleDecisions) != 1 || grant.Request.Intent.Payload["claim"].Text == "" {
				t.Fatal("genuine accepted grant missing", grant)
			}
			var claim runtime.DelegationClaim
			if err := json.Unmarshal([]byte(grant.Request.Intent.Payload["claim"].Text), &claim); err != nil || claim != m.Claim {
				t.Fatal("actual claim differs from pre-provision manifest", claim, err)
			}
			requestSequence := int64(3)
			if tc.requestFirst {
				requestSequence = 2
			}
			var original *runtime.ObservationView
			for _, o := range snap.Observations {
				if o.SourceOperationID == m.BARequest && o.Run == m.A {
					original = o
				}
			}
			if original == nil || original.SelectedAt == nil || original.SelectedAt.Sequence != requestSequence || len(original.Admissions) == 0 {
				t.Fatal("actual original not admitted and selected", original)
			}
			reciprocalSelectedHistory(t, snap, original)
			a, _ := snap.Run(m.A)
			b, _ := snap.Run(m.B)
			if tc.branch != nil {
				wantScratch := int64(2)
				if *tc.branch {
					wantScratch = 1
				}
				if a.Projection.Data["binding_scratch"].Number != wantScratch {
					t.Fatal("wrong forward branch outcome", a.Projection.Data)
				}
			}
			if tc.requestFirst {
				readySelected := false
				for _, o := range snap.Observations {
					if o.SourceOperationID == m.Ready && o.Run == m.A && o.SelectedAt != nil && o.SelectedAt.Sequence == 3 {
						reciprocalSelectedHistory(t, snap, o)
						readySelected = true
					}
				}
				if !readySelected {
					t.Fatal("actual ready selection missing")
				}
			}
			if tc.requestFirst {
				if e.ExternalAttempt.Allowed || w.Allowed || e.ExternalReceipt != nil || len(e.AuthorityAdmissions) != 0 || ea.Contacts != 0 || eb.Contacts != 0 || a.Projection.Data["paid"].Flag || b.Projection.Data["paid"].Flag {
					t.Fatal("refused construction gained exposure or payment", e, a, b)
				}
			} else if !e.ExternalAttempt.Allowed || !w.Allowed || a.Projection.Status != "succeeded" || b.Projection.Status != "succeeded" || !a.Projection.Data["paid"].Flag || !b.Projection.Data["paid"].Flag || ea.Contacts != 1 || eb.Contacts != 1 {
				t.Fatal("adjacent control did not complete", e, a, b)
			}
			baseline3b3Write(t, dir, "causal-assertions.json", map[string]any{"Claim": claim, "Authority": w, "Original": original, "A": a, "B": b, "Pending": snap.Pending, "EndpointContacts": []int64{ea.Contacts, eb.Contacts}})
			reciprocalIndex(t, reciprocalCase{Version: "alan/phase3b3-case/v1", Name: "construction-" + tc.name, Schedule: "automatic-run-public-pre-provision", Directory: dir, Classification: tc.wantReason, Manifest: m})
			// The existing exporter covers v1 helpers but not this new fixture.
			raw, err := os.ReadFile("phase3b4_construction_test.go")
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(raw)
			baseline3b3Write(t, dir, "construction-source.json", map[string]string{"File": "internal/durabletest/phase3b4_construction_test.go", "SHA256": hex.EncodeToString(hash[:])})
			baseline3b3Manifest(t, dir)
		})
	}
}
