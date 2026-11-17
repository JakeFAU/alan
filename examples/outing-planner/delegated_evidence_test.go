package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestDelegatedCausalEvidence(t *testing.T) {
	server, _ := fixtureServer(t, "1", "3")
	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	options := defaultDeploymentOptions(delegatedWeather)
	path := filepath.Join(t.TempDir(), "completed.db")
	if err := createPlannerDeployment(t.Context(), path, c, options); err != nil {
		t.Fatal(err)
	}
	first, _, err := inspectPlannerDeployment(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := operateBoundedDeployment(t.Context(), path, first.Namespace, c, options, standardBounds); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, want string
		mutate     func(*runtime.ExecutionSnapshot)
	}{
		{"original forecast owner", "unexpected delegated forecast", func(s *runtime.ExecutionSnapshot) {
			e := s.Effects[0]
			e.Source = weatherRun
			for i := range s.Operations {
				if s.Operations[i].ID == e.OperationID {
					s.Operations[i].Run = weatherRun
				}
			}
		}},
		{"forecast view sequence", "unexpected retained delegated effect", func(s *runtime.ExecutionSnapshot) {
			s.Effects[0].Sequence++
		}},
		{"wrong original forecast result", "different original result", func(s *runtime.ExecutionSnapshot) {
			s.Effects[0].AdapterResult, s.Effects[1].AdapterResult = s.Effects[1].AdapterResult, s.Effects[0].AdapterResult
		}},
		{"wrong published helper response", "changes the original recorded helper reply", func(s *runtime.ExecutionSnapshot) {
			s.Invocations[0].Outcome.Response = s.Invocations[0].Inputs
		}},
		{"foreign forecast selection", "selection run", func(s *runtime.ExecutionSnapshot) {
			o, _ := s.Observation(s.Effects[0].Settlement.ObservationID)
			o.SelectedAt.Run, o.SelectedAt.Sequence = weatherRun, 100000
		}},
		{"foreign helper result", "outcome observation", func(s *runtime.ExecutionSnapshot) {
			call := s.Invocations[0]
			call.OutcomeObservationID = call.RequestObservationID
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cut, _, err := inspectPlannerDeployment(t.Context(), path)
			if err != nil {
				t.Fatalf("genuine completed cut rejected: %v", err)
			}
			tc.mutate(cut)
			if err := verifyPlannerEvidence(cut, c, delegatedWeather); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDelegatedCausalEvidenceUnknownAndPending(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		conn, _, err := http.NewResponseController(w).Hijack()
		if err == nil {
			err = conn.Close()
		}
		if err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	c := fixtureConfiguration()
	c.Endpoint = server.URL + fixturePath
	path := filepath.Join(t.TempDir(), "unknown.db")
	config := writeDelegatedConfig(t, c)
	plannerOutcome(t, plannerOutput(t, "start", "--weather", delegatedWeather, "--json", "--db", path, "--config", config), pendingOutcome)
	plannerOutcome(t, plannerOutput(t, "run", "--json", "--db", path), unknownForecast)
	cut, _, err := inspectPlannerDeployment(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cut.Effects) != 1 || len(cut.Invocations) != 1 {
		t.Fatalf("unknown result cut has %d forecast operations and %d invocations", len(cut.Effects), len(cut.Invocations))
	}
	call, effect := cut.Invocations[0], cut.Effects[0]
	if call.Outcome != nil || call.Admission == nil || !call.Admission.Accepted || effect.DispatchAdmission == nil ||
		effect.AdapterResult != nil || effect.Settlement != nil {
		t.Fatal("unknown cut did not retain the original admitted invocation and unsettled forecast")
	}
	for _, command := range []string{"status", "inspect", "replay", "resume", "resume"} {
		output := plannerOutput(t, command, "--json", "--db", path)
		plannerOutcome(t, output, unknownForecast)
		requireReportedEffect(t, output, effect.OperationID)
		var report plannerReport
		if err := json.Unmarshal([]byte(output), &report); err != nil {
			t.Fatal(err)
		}
		if report.Delegated == nil || report.Delegated.OperationID != call.OperationID ||
			report.Delegated.InvocationID != call.Admission.Invocation.ID ||
			report.Assessment != nil || report.Decision != nil || report.Recommendation != nil ||
			len(report.Effects) != 1 {
			t.Fatalf("%s changed the unknown cut", command)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("unknown result contacted fixture %d times", calls.Load())
	}

	pending := slices.Clone(cut.Pending)
	cut.Pending = slices.DeleteFunc(pending, func(p runtime.PendingObligation) bool {
		return p.Kind == "effect_reserved_unknown" && p.OperationID == effect.OperationID
	})
	if len(cut.Pending) != len(pending)-1 {
		t.Fatal("control did not omit exactly one outstanding forecast obligation")
	}
	err = verifyPlannerEvidence(cut, c, delegatedWeather)
	if err == nil || !strings.Contains(err.Error(), "missing delegated pending obligation") {
		t.Fatalf("omitted pending operation passed: %v", err)
	}
}
