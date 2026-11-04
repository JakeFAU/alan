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
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type bindingCaptureRef struct {
	Ordinal                       int
	Predecessor, Label, Directory string
	OwnerPID                      int
	AcknowledgedSIGKILL           bool
}
type bindingCase struct {
	Version, Name, Schedule, Profile, Classification string
	Manifest                                         delegation.RequestBindingManifest
	PrincipalRequests                                [2]string
	AuxiliaryRequests                                []string
	Captures                                         []bindingCaptureRef
}

func bindingDirectory(t *testing.T, name string) string {
	t.Helper()
	root := os.Getenv("ALAN_PHASE3B4_EVIDENCE")
	if root == "" {
		return t.TempDir()
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, name+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("raw evidence:", dir)
	return dir
}
func bindingIndex(t *testing.T, dir string, c bindingCase, p runtime.Provision) {
	t.Helper()
	c.Version = "alan/phase3b4-case/v1"
	c.Profile = p.Composition.RequestAuthority
	c.PrincipalRequests = c.Manifest.PrincipalRequests
	for i := range c.Captures {
		r := &c.Captures[i]
		if r.OwnerPID != 0 {
			continue
		}
		r.OwnerPID = os.Getpid()
		if strings.HasSuffix(r.Directory, "/post-open") {
			var opened struct{ PID int }
			raw, err := os.ReadFile(filepath.Join(dir, strings.TrimSuffix(r.Directory, "/post-open"), "open.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(raw, &opened); err != nil || opened.PID <= 0 {
				t.Fatal("capture owner", err)
			}
			r.OwnerPID = opened.PID
		}
	}

	c.Manifest.ProgramIDs = map[string]string{}
	for _, r := range p.Runs {
		c.Manifest.ProgramIDs[r.ID.Subject] = r.ProgramID
	}
	baseline3b3Write(t, dir, "case.json", c)
	baseline3b3Write(t, dir, "manifest.json", c.Manifest)
	reciprocalSources(t, dir)
	files := map[string]string{}
	for _, pat := range []string{"phase3b4_*test.go", "../../examples/delegation/request_binding*.go"} {
		paths, err := filepath.Glob(pat)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			files[path] = hex.EncodeToString(sum[:])
		}
	}
	baseline3b3Write(t, dir, "binding-source.json", files)
	baseline3b3Manifest(t, dir)
}
func bindingVersion(t *testing.T, p *runtime.Provision, version string) {
	t.Helper()
	p.Composition.RequestAuthority = version
	for i := range p.Programs {
		old := p.Programs[i].ID
		raw := []byte(strings.ReplaceAll(strings.ReplaceAll(string(p.Programs[i].Bytes), runtime.RequestAuthorityBindingVersion, version), runtime.RequestAuthorityVersion, version))
		sum := sha256.Sum256(raw)
		id := hex.EncodeToString(sum[:])
		p.Programs[i].Bytes = raw
		p.Programs[i].ID = id
		for j := range p.Runs {
			if p.Runs[j].ProgramID == old {
				p.Runs[j].ProgramID = id
			}
		}
	}
}

// Update all genuine input claim copies before hashing/provisioning. An
// ordinary request payload remains data, but the fixture declarations agree.
func bindingClaimAll(t *testing.T, p *runtime.Provision, edit func(*runtime.DelegationClaim)) {
	t.Helper()
	for _, subject := range []string{"a", "b"} {
		reciprocalEdit(t, p, subject, func(program map[string]any) {
			var visit func(any)
			visit = func(x any) {
				switch v := x.(type) {
				case map[string]any:
					for k, y := range v {
						if k == "Text" {
							if str, ok := y.(string); ok {
								var claim runtime.DelegationClaim
								if json.Unmarshal([]byte(str), &claim) == nil && claim.OriginalRequestID != "" {
									edit(&claim)
									raw, err := json.Marshal(claim)
									if err != nil {
										t.Fatal(err)
									}
									v[k] = string(raw)
								}
							}
						}
						visit(y)
					}
				case []any:
					for _, y := range v {
						visit(y)
					}
				}
			}
			visit(program)
		})
	}
}

func bindingFixture(t *testing.T, ua, ub string, n int, branch *bool, ready bool) (runtime.Provision, delegation.RequestBindingManifest) {
	t.Helper()
	p, m, err := delegation.RequestBinding(delegation.RequestBindingOptions{EndpointA: ua, EndpointB: ub, LocalSteps: n, ForwardBranch: branch, ReadyFirst: ready, TerminalHandshake: true})
	if err != nil {
		t.Fatal(err)
	}
	return p, m
}
func bindingEndpoints(t *testing.T, dir, pa, pb string, ca, cb *child) map[string]resource.Snapshot {
	t.Helper()
	e := map[string]resource.Snapshot{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)}
	phaseStopEndpoint(t, ca)
	phaseStopEndpoint(t, cb)
	baseline3b3Write(t, dir, "endpoints.json", e)
	reciprocalCopyDB(t, pa, filepath.Join(dir, "endpoint-a.sqlite"))
	reciprocalCopyDB(t, pb, filepath.Join(dir, "endpoint-b.sqlite"))
	return e
}
func bindingRun(t *testing.T, s *runtime.Store, dir string, maxSteps int) (runtime.RunReport, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	var events []boundary.Event
	ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
		reciprocalIndexMu.Lock()
		defer reciprocalIndexMu.Unlock()
		events = append(events, e)
		return nil
	})
	opts := runtime.RunOptions{MaxSteps: maxSteps, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second}
	report, err := s.Run(ctx, opts)
	baseline3b3Write(t, dir, "run.json", map[string]any{"Options": opts, "OuterTimeout": 15 * time.Second, "Report": report, "Error": fmt.Sprint(err)})
	baseline3b3Write(t, dir, "boundaries.json", events)
	return report, err
}
func bindingSelected(t *testing.T, s *runtime.ExecutionSnapshot, run runtime.RunID, operation string) *runtime.ObservationView {
	t.Helper()
	for _, o := range s.Observations {
		if o.Run == run && o.SourceOperationID == operation && o.Kind == "message" {
			return o
		}
	}
	t.Fatal("missing original observation", operation)
	return nil
}
func bindingQ2(t *testing.T, p *runtime.Provision, m *delegation.RequestBindingManifest, selectOriginal bool) {
	t.Helper()
	reciprocalEdit(t, p, "b", func(program map[string]any) {
		code := program["Code"].([]any)
		reciprocalShift(code, 5, 1)
		raw, _ := json.Marshal(code[0])
		var q2 any
		json.Unmarshal(raw, &q2)
		q2.(map[string]any)["Dst"] = "q2"
		program["Code"] = append(append(append([]any{}, code[:5]...), q2), code[5:]...)
	})
	reciprocalEdit(t, p, "a", func(program map[string]any) {
		code := program["Code"].([]any)
		if selectOriginal {
			reciprocalShift(code, 3, 1)
			raw, _ := json.Marshal(code[1])
			var receive any
			json.Unmarshal(raw, &receive)
			program["Code"] = append(append(append([]any{}, code[:3]...), receive), code[3:]...)
		} else {
			code[1], code[2] = code[2], code[1]
		}
	})
	m.Reciprocal.ABService = `["b",1,10,0]`
	m.Reciprocal.ABReply = `["b",1,13,0]`
	if selectOriginal {
		m.ServiceSequence = 5
		m.ReplySequence = 8
		m.Reciprocal.BAService = `["a",1,5,0]`
		m.Reciprocal.BAReply = `["a",1,8,0]`
		m.Reciprocal.Claim.ServiceOperationID = m.Reciprocal.BAService
		bindingClaimAll(t, p, func(c *runtime.DelegationClaim) { c.ServiceOperationID = m.Reciprocal.BAService })
	}
}
func TestPhase3B4CampaignPositive(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name                     string
		n                        int
		branch                   *bool
		ready, v1, q2, overwrite bool
	}{
		{name: "P01-v1", ready: true, v1: true}, {name: "P01-v2", ready: true}, {name: "P02-v1", v1: true}, {name: "P02-v2"}, {name: "P04", n: 1}, {name: "P05", n: 3}, {name: "P06", branch: &yes}, {name: "P07", branch: &no}, {name: "P10", q2: true}, {name: "P11", n: 1, overwrite: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := bindingDirectory(t, tc.name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			p, m := bindingFixture(t, ua, ub, tc.n, tc.branch, tc.ready)
			if tc.v1 {
				bindingVersion(t, &p, runtime.RequestAuthorityVersion)
				m.Reciprocal.Claim.Version = runtime.RequestAuthorityVersion
			}
			if tc.q2 {
				bindingQ2(t, &p, &m, true)
			}
			if tc.overwrite {
				reciprocalEdit(t, &p, "a", func(program map[string]any) {
					in := reciprocalInstruction(program, 3)
					in["Dst"] = "peer_request"
					in["A"] = reciprocalLiteral(schema.Text("overwritten"))
				})
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			report, runErr := bindingRun(t, s, dir, 256)
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if runErr != nil {
				t.Fatal(runErr)
			}
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			snap := bindingCapture(t, db, filepath.Join(dir, "final"))
			e, ok := snap.Effect(m.Reciprocal.BAService)
			if !ok || e.ExternalAttempt == nil || e.ExternalAttempt.Authority == nil {
				t.Fatal("missing decision", e)
			}
			q := bindingSelected(t, snap, m.Reciprocal.A, m.Reciprocal.BARequest)
			want := int64(2)
			if tc.ready {
				want = 3
			}
			if q.SelectedAt == nil || q.SelectedAt.Sequence != want || len(q.Admissions) == 0 {
				t.Fatal("original selection", q)
			}
			reciprocalSelectedHistory(t, snap, q)
			if tc.overwrite {
				a, _ := snap.Run(m.Reciprocal.A)
				if a.Projection.Data["peer_request"].Text != "overwritten" {
					t.Fatal("dead register not overwritten")
				}
			}
			classification := "safe-complete"
			if tc.v1 && !tc.ready {
				classification = "authority_request_not_admitted"
				if e.ExternalAttempt.Allowed || e.ExternalAttempt.Authority.Reason != classification || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal(e, endpoints)
				}
			} else {
				bindingComplete(t, snap, m.Reciprocal, endpoints)
				if e.ExternalAttempt.Authority.RequestConsumedSequence != want || e.ExternalAttempt.Authority.RequestObservationID != q.ID {
					t.Fatal("not exact q witness", e)
				}
				if !report.Quiescent || len(report.OpenRuns) != 0 || len(report.PendingDeliveries) != 0 || len(report.UnresolvedIntents) != 0 || len(snap.Pending) != 0 {
					t.Fatal("not globally complete", report, snap.Pending)
				}
			}
			if tc.branch != nil {
				a, _ := snap.Run(m.Reciprocal.A)
				wantScratch := int64(2)
				if *tc.branch {
					wantScratch = 1
				}
				if a.Projection.Data["scratch"].Number != wantScratch {
					t.Fatal(a)
				}
				page, err := snap.History(0, 1000)
				if err != nil {
					t.Fatal(err)
				}
				wantPC := []int{3, 6, 7}
				if *tc.branch {
					wantPC = []int{3, 4, 5}
				}
				seen := 0
				for _, step := range page.Steps {
					if step.Run == m.Reciprocal.A && step.Sequence >= 4 && step.Sequence <= 6 {
						if step.BeforePC != wantPC[step.Sequence-4] {
							t.Fatal("wrong executed branch PC", step)
						}
						seen++
					}
				}
				if seen != 3 {
					t.Fatal("missing branch reductions")
				}

			}
			c := bindingCase{Name: tc.name, Schedule: "automatic-run", Classification: classification, Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}}}
			if tc.q2 {
				c.AuxiliaryRequests = []string{`["b",1,6,0]`}
				q2 := bindingSelected(t, snap, m.Reciprocal.A, c.AuxiliaryRequests[0])
				if q2.SelectedAt == nil || q2.SelectedAt.Sequence != 4 || q2.ID == q.ID {
					t.Fatal("q2", q2)
				}
				original, _ := snap.Operation(m.Reciprocal.BARequest)
				aux, _ := snap.Operation(c.AuxiliaryRequests[0])
				a, _ := json.Marshal(original.Payload)
				b, _ := json.Marshal(aux.Payload)
				if string(a) != string(b) {
					t.Fatal("payloads differ")
				}
				for _, o := range snap.Observations {
					if o.Kind == "message" && o.Payload["request_id"].Text == c.AuxiliaryRequests[0] && o.Payload["accepted"].Flag {
						t.Fatal("auxiliary silently counted paid", o)
					}
				}

			}
			baseline3b3Write(t, dir, "causal-assertions.json", map[string]any{"Original": q, "Authority": e.ExternalAttempt.Authority, "Pending": snap.Pending, "PrincipalComplete": classification == "safe-complete"})
			bindingIndex(t, dir, c, p)
		})
	}
}
