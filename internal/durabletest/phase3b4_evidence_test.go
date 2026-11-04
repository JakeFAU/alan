//go:build integration

package durabletest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/examples/delegation"
	resource "github.com/JakeFAU/alan/internal/fixture"
	"github.com/JakeFAU/alan/runtime"
)

func bindingCapture(t *testing.T, path, dir string) *runtime.ExecutionSnapshot {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// The source owner has closed or been SIGKILLed. Inspect a complete disposable
	// copy, then archive exact stopped bytes without ever opening that archive.
	probe := filepath.Join(t.TempDir(), "probe.sqlite")
	reciprocalCopyDB(t, path, probe)
	raw, rawErr := runtime.Inspect(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "inspect.json", raw)
	s, typedErr := runtime.InspectExecution(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "typed-inspection.json", s)
	replay, replayErr := runtime.Replay(t.Context(), probe, runtime.DefaultLimits())
	baseline3b3Write(t, dir, "replay.json", map[string]any{"Report": replay, "Error": fmt.Sprint(replayErr)})
	baseline3b3Write(t, dir, "outcomes.json", map[string]string{"Inspect": fmt.Sprint(rawErr), "InspectExecution": fmt.Sprint(typedErr), "Replay": fmt.Sprint(replayErr)})
	reciprocalCopyDB(t, path, filepath.Join(dir, "runtime.sqlite"))
	if rawErr != nil || typedErr != nil || replayErr != nil {
		t.Fatal("snapshot validation", rawErr, typedErr, replayErr)
	}
	bindingAccounting(t, raw, s)
	bindingSidecar(t, dir, raw, s)
	cursor := 0
	for n := 0; ; n++ {
		page, err := s.History(cursor, 64)
		if err != nil {
			t.Fatal(err)
		}
		baseline3b3Write(t, dir, fmt.Sprintf("history-%03d.json", n), page)
		if !page.More {
			break
		}
		cursor = page.Next
	}
	return s
}

func bindingComplete(t *testing.T, s *runtime.ExecutionSnapshot, m delegation.Manifest, endpoints map[string]resource.Snapshot) {
	t.Helper()
	if len(s.Pending) != 0 {
		t.Fatal("completion has pending obligations", s.Pending)
	}
	for _, id := range []runtime.RunID{m.A, m.B} {
		r, ok := s.Run(id)
		if !ok || r.Projection.Status != "succeeded" || !r.Projection.Data["paid"].Flag {
			t.Fatalf("original not complete %+v", r)
		}
	}
	for _, x := range []struct {
		request, service, reply string
		server, requester       runtime.RunID
		endpoint                string
	}{{m.BARequest, m.BAService, m.BAReply, m.A, m.B, "b"}, {m.ABRequest, m.ABService, m.ABReply, m.B, m.A, "a"}} {
		e, ok := s.Effect(x.service)
		if !ok || e.Source != x.server || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || e.ExternalReceipt == nil || e.Settlement == nil || len(e.AuthorityAdmissions) == 0 {
			t.Fatal("missing exact service", e)
		}
		w := e.ExternalAttempt.Authority
		server, _ := s.Run(x.server)
		if w == nil || !w.Allowed || w.Use == nil || w.ServingRootID != server.RootID {
			t.Fatal("wrong serving-root authority", w)
		}
		if x.server == m.A && m.Options.Scenario == delegation.S2Handshake {
			if w.Use.Path != "delegated" || w.Use.GrantID != m.Grant || w.Use.Root != m.BAServiceRoot || w.OriginalRequestID != m.BARequest || w.ServiceOperationID != m.BAService || w.RequestConsumedSequence <= 0 || w.RequestConsumedSequence >= e.Sequence || w.GrantDecisionSequence < 1 {
				t.Fatal("delegated custody witness", w)
			}
		}
		for i, g := range e.AuthorityAdmissions {
			if !g.Allowed || g.Ordinal != int64(i+1) || g.Source.Run != x.server || g.Namespace != s.Namespace || g.AttemptDigest == "" {
				t.Fatal("gate identity", g)
			}
		}
		var selected *runtime.ObservationView
		for _, o := range s.Observations {
			if o.Run == x.server && o.SourceOperationID == x.request && o.Kind == "message" {
				if selected != nil {
					t.Fatal("duplicate request")
				}
				selected = o
			}
		}
		if selected == nil || selected.ProducerRun == nil || *selected.ProducerRun != x.requester || selected.SelectedAt == nil || selected.SelectedAt.Run != x.server || selected.SelectedAt.Sequence >= e.Sequence {
			t.Fatal("original request selection", selected, e)
		}
		result, ok := s.Observation(e.Settlement.ObservationID)
		if !ok || result.Run != x.server || result.Operation != x.service || result.SourceOperationID != x.service || result.SelectedAt == nil || result.SelectedAt.Run != x.server || result.SelectedAt.Sequence != e.Sequence+1 || result.Payload["mutation"].Number != e.ExternalReceipt.Mutation || result.Payload["total"].Number != e.ExternalReceipt.Total {
			t.Fatal("original result selection", result)
		}
		found := false
		for _, o := range s.Observations {
			if o.Run == x.requester && o.SourceOperationID == x.reply && o.Kind == "message" {
				if o.ProducerRun == nil || *o.ProducerRun != x.server || o.SelectedAt == nil || o.SelectedAt.Run != x.requester || o.Payload["request_id"].Text != x.request || o.Payload["service_operation"].Text != x.service || o.Payload["mutation"] != result.Payload["mutation"] || o.Payload["total"] != result.Payload["total"] || !o.Payload["accepted"].Flag {
					t.Fatal("exact result-bearing reply", o)
				}
				reply, ok := s.Operation(x.reply)
				if !ok || reply.Run != x.server || reply.Sequence <= result.SelectedAt.Sequence {
					t.Fatal("reply before custody")
				}
				reciprocalSelectedHistory(t, s, o)
				found = true
			}
		}
		reciprocalSelectedHistory(t, s, result)
		reciprocalSelectedHistory(t, s, selected)
		if !found {
			t.Fatal("original reply not selected")
		}
		ledger := endpoints[x.endpoint]
		if len(ledger.Applications) != 1 || ledger.Applications[0] != *e.ExternalReceipt || ledger.Total != 1 || ledger.Contacts < 1 {
			t.Fatal("endpoint exact result", ledger, e)
		}
		for _, req := range ledger.Requests {
			if req != e.ExternalAttempt.Request {
				t.Fatal("original wire changed", req, e.ExternalAttempt.Request)
			}
		}
	}
}

func bindingAccounting(t *testing.T, raw runtime.Inspection, s *runtime.ExecutionSnapshot) {
	t.Helper()
	if s.StorageVersion != 13 && s.StorageVersion != 14 && s.StorageVersion != 15 && s.StorageVersion != 16 {
		t.Fatal("request authority profile", s.StorageVersion)
	}
	for _, e := range s.Effects {
		allowed := false
		if e.ExternalAttempt != nil {
			allowed = e.ExternalAttempt.Allowed
		}
		if e.AdapterAttempt != nil {
			allowed = e.AdapterAttempt.Allowed
		}
		count := 0
		for _, row := range raw.Tables["composition_charges"] {
			if row["operation_id"] != e.OperationID {
				continue
			}
			var c struct {
				RootID, OperationID, Kind         string
				Effects, Invocations, CreatedRuns int64
			}
			if err := json.Unmarshal([]byte(row["descriptor"].(string)), &c); err != nil {
				t.Fatal(err)
			}
			run, _ := s.Run(e.Source)
			if c.RootID != run.RootID || row["root_id"] != run.RootID || c.OperationID != e.OperationID || c.Kind != "effect" || c.Effects != 1 || c.Invocations != 0 || c.CreatedRuns != 0 {
				t.Fatal("misattributed debit", c, e)
			}
			count++
		}
		want := 0
		if allowed {
			want = 1
		}
		if count != want {
			t.Fatal("original debit count", e.OperationID, count, want)
		}
	}
}
