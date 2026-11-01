package executor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func externalProvision(t *testing.T, target, profile string) Provision {
	t.Helper()
	p := Provision{}
	addProgram(t, &p, "client", []Instruction{
		{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text(profile)), "delta": Lit(Int(5))}}},
		{Op: "receive", Match: &Match{Kind: "receipt", Provenance: "resource-receipt", Source: Lit(Text(target)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Operation: Ref("pending")}, Bind: map[string]string{"total": "total"}},
		{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	p.Permissions = []Permission{{Kind: "external", Source: "client", Target: target, Operation: "apply", Revision: 1}}
	return p
}
func TestExternalRunnerAuthenticatesAndConsumesDurableReceipt(t *testing.T) {
	resourcePath := filepath.Join(t.TempDir(), "resource.sqlite")
	resource, err := resourcefixture.Open(t.Context(), resourcePath, "endpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer resource.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request resourcefixture.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		receipt, err := resource.Apply(r.Context(), request)
		if err != nil {
			t.Error(err)
			return
		}
		json.NewEncoder(w).Encode(receipt)
	}))
	defer server.Close()
	s, path := messageStore(t, externalProvision(t, server.URL, "idempotent"))
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.UnresolvedIntents) != 0 || current(t, s, "client").Projection.Status != "succeeded" || current(t, s, "client").Projection.Data["total"] != Int(5) {
		t.Fatalf("external receipt not consumed %+v", report)
	}
	s.Close()
	before, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	after, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if before.Contacts != after.Contacts || len(after.Applications) != 1 {
		t.Fatal("replay contacted endpoint")
	}
}
func TestNonIdempotentUnknownNeverAutomaticallyRepeats(t *testing.T) {
	resourcePath := filepath.Join(t.TempDir(), "resource.sqlite")
	resource, err := resourcefixture.Open(t.Context(), resourcePath, "endpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer resource.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request resourcefixture.Request
		json.NewDecoder(r.Body).Decode(&request)
		if _, err := resource.Apply(r.Context(), request); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	p := externalProvision(t, server.URL, "non-idempotent")
	addProgram(t, &p, "ready-peer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, path := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.UnresolvedIntents) != 1 || current(t, s, "ready-peer").Projection.Status != "succeeded" {
		t.Fatal(report)
	}
	s.Close()
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Applications) != 1 || snapshot.Contacts != 1 {
		t.Fatalf("blind repeat: %+v", snapshot)
	}
}

func TestExternalAuthorizationSurvivesClosureAndNewAuthorizationIsDenied(t *testing.T) {
	resourcePath := filepath.Join(t.TempDir(), "resource.sqlite")
	r, err := resourcefixture.Open(t.Context(), resourcePath, "late-resource")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var input resourcefixture.Request
		json.NewDecoder(request.Body).Decode(&input)
		receipt, err := r.Apply(request.Context(), input)
		if err != nil {
			t.Error(err)
			return
		}
		json.NewEncoder(w).Encode(receipt)
	}))
	defer server.Close()
	for _, authorized := range []bool{true, false} {
		t.Run(fmt.Sprint(authorized), func(t *testing.T) {
			p := externalProvision(t, server.URL, "idempotent")
			p.Programs = nil
			p.Runs = nil
			p.Subjects = nil
			addProgram(t, &p, "closed", []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "external", Target: Lit(Text(server.URL)), Generation: Lit(Int(1)), Obligation: Lit(Text("receipt")), Payload: map[string]Operand{"profile": Lit(Text("idempotent")), "delta": Lit(Int(5))}}}, {Op: "end", Outcome: "failed"}}, map[string]Value{})
			p.Permissions[0].Source = "closed"
			s, path := messageStore(t, p)
			state, err := s.Commit(t.Context(), current(t, s, "closed").Authority(), "")
			if err != nil {
				t.Fatal(err)
			}
			id := state.Projection.Data["pending"].Text
			if authorized {
				if _, _, err = s.prepareExternal(t.Context(), id); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.Commit(t.Context(), state.Authority(), ""); err != nil {
				t.Fatal(err)
			}
			// A prior allowed reservation keeps its historical authority, but this
			// fresh exposure follows Failure. A first denial still settles normally.
			if settled, err := s.DispatchExternal(t.Context(), id); err != nil || settled != !authorized {
				t.Fatalf("settlement %v %v", settled, err)
			}
			if current(t, s, "closed").Projection.Status != "failed" {
				t.Fatal("late receipt resumed run")
			}
			outcome, err := s.externalOutcome(t.Context(), id)
			if authorized {
				if !errors.Is(err, sql.ErrNoRows) || !outcome.Attempt.Allowed {
					t.Fatal("lost original authority or invented outcome", outcome, err)
				}
			} else if err != nil || outcome.Disposition != "source_denied" {
				t.Fatal(outcome, err)
			}
			s.Close()
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestUnknownExternalIsExplicitInRunReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer server.Close()
	s, _ := messageStore(t, externalProvision(t, server.URL, "non-idempotent"))
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(report)
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	var unknown []string
	json.Unmarshal(fields["UnknownExternal"], &unknown)
	if len(unknown) != 1 || unknown[0] != `["client",1,1,0]` {
		t.Fatalf("unknown physical outcome not explicit: %s", raw)
	}
}

func TestExternalTimeoutIsUnknownAndReadyPeerStillCommits(t *testing.T) {
	resourcePath := filepath.Join(t.TempDir(), "resource.sqlite")
	r, err := resourcefixture.Open(t.Context(), resourcePath, "timeout-resource")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var input resourcefixture.Request
		json.NewDecoder(request.Body).Decode(&input)
		if _, err := r.Apply(request.Context(), input); err != nil {
			t.Error(err)
		}
		<-request.Context().Done()
	}))
	defer server.Close()
	p := externalProvision(t, server.URL, "non-idempotent")
	addProgram(t, &p, "peer", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	s, _ := messageStore(t, p)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.UnknownExternal) != 1 || current(t, s, "client").Projection.Status != "open" || current(t, s, "peer").Projection.Status != "succeeded" || count(t, s, "external_receipts") != 0 {
		t.Fatalf("timeout became failure or blocked peer: %+v", report)
	}
	snapshot, err := resourcefixture.Inspect(t.Context(), resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Applications) != 1 {
		t.Fatal("unknown not physically applied")
	}
}
