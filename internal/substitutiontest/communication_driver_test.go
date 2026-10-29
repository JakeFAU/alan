//go:build integration

package substitutiontest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	communication "github.com/JakeFAU/alan/research/communication"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

// This host apparatus selects only exported operations on unchanged artifacts.
// In particular it never interprets branches or computes a successor projection.
type communicationDriver struct {
	t            *testing.T
	path         string
	store        *runtime.Store
	journal      *os.File
	repeats      []communication.Authorization
	readmissions []communication.Admission
}

func newCommunicationDriver(t *testing.T, path string) *communicationDriver {
	t.Helper()
	s, e := runtime.Open(t.Context(), path, runtime.DefaultLimits())
	must(t, e)
	dir := os.Getenv("ALAN_I_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	dir = filepath.Join(dir, hash(t.Name())[:12])
	must(t, os.MkdirAll(dir, 0755))
	f, e := os.CreateTemp(dir, fmt.Sprintf("journal-%d-*.jsonl", os.Getpid()))
	must(t, e)
	d := &communicationDriver{t: t, path: path, store: s, journal: f}
	for _, p := range inspect(t, path).Tables["programs"] {
		raw := []byte(p["bytes"].(string))
		id := p["id"].(string)
		if fmt.Sprintf("%x", shaBytes(raw)) != id {
			t.Fatal("program content identity", id)
		}
		file := filepath.Join(dir, "program-"+id+".json")
		if old, e := os.ReadFile(file); e == nil {
			if string(old) != string(raw) {
				t.Fatal("artifact collision")
			}
		} else {
			must(t, os.WriteFile(file, raw, 0644))
		}
	}
	d.record(map[string]any{"kind": "open", "pid": os.Getpid(), "inspection_hash": hash(inspect(t, path))})
	t.Cleanup(func() { must(t, f.Close()); must(t, s.Close()) })
	t.Logf("step journal %s", f.Name())
	return d
}
func (d *communicationDriver) record(v any) {
	d.t.Helper()
	must(d.t, json.NewEncoder(d.journal).Encode(v))
	must(d.t, d.journal.Sync())
}
func communicationIdentity(v ...any) string { raw, _ := json.Marshal(v); return string(raw) }
func communicationReport(s runtime.Inspection) intent {
	for _, r := range s.Tables["intents"] {
		i := decode[intent](r, "descriptor")
		if i.Kind == "message" && i.Run.Subject == "peer" && i.Target == "root" && i.Obligation == "phase1j_s_join" {
			return i
		}
	}
	return intent{}
}
func communicationRow(s runtime.Inspection, table, key string, value any) map[string]any {
	for _, r := range s.Tables[table] {
		if reflect.DeepEqual(r[key], value) {
			return r
		}
	}
	return nil
}
func (d *communicationDriver) step(id runtime.RunID) bool {
	t := d.t
	pre, e := d.store.Snapshot(t.Context(), id)
	must(t, e)
	if pre.Projection.Status != "open" {
		return false
	}
	before := inspect(t, d.path)
	program := communicationRow(before, "programs", "id", pre.ProgramID)
	var p struct{ Code []json.RawMessage }
	must(t, json.Unmarshal([]byte(program["bytes"].(string)), &p))
	if pre.Projection.PC < 0 || pre.Projection.PC >= len(p.Code) {
		t.Fatal("invalid pre-PC")
	}
	raw := p.Code[pre.Projection.PC]
	var instruction struct{ Op string }
	must(t, json.Unmarshal(raw, &instruction))
	ids := []string{""}
	rows := map[string]map[string]any{}
	if instruction.Op == "receive" {
		ids = nil
		for _, r := range before.Tables["observations"] {
			o := decode[observation](r, "descriptor")
			if o.Run == id && r["consumed_sequence"] == nil {
				ids = append(ids, o.ID)
				rows[o.ID] = r
			}
		}
		sort.Strings(ids)
	}
	for _, selected := range ids {
		next, err := d.store.Commit(t.Context(), pre.Authority(), selected)
		after := inspect(t, d.path)
		rec := map[string]any{"kind": "commit", "program_id": pre.ProgramID, "program_sha256": pre.ProgramID, "instruction": raw, "pre": pre, "authority": pre.Authority(), "selected": rows[selected], "selected_id": selected, "post": next, "before_hash": hash(before), "after_hash": hash(after)}
		if err != nil {
			rec["kind"] = "rejected_commit_probe"
			rec["error"] = err.Error()
			d.record(rec)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected probe changed retained state", err)
			}
			now, e := d.store.Snapshot(t.Context(), id)
			must(t, e)
			if !reflect.DeepEqual(pre, now) {
				t.Fatal("rejected probe changed authority")
			}
			if instruction.Op != "receive" {
				t.Fatal("pure instruction failed", err)
			}
			continue
		}
		if next.Sequence != pre.Sequence+1 || next.ProgramID != pre.ProgramID {
			t.Fatal("commit did not execute original next step")
		}
		var history map[string]any
		for _, h := range after.Tables["history"] {
			rid, e := evidenceID(h)
			must(t, e)
			seq, e := evidenceNumber(h["sequence"])
			must(t, e)
			if rid == id && seq == next.Sequence {
				history = h
			}
		}
		if history == nil || !reflect.DeepEqual(decode[runtime.Projection](history, "successor"), next.Projection) {
			t.Fatal("committed history absent")
		}
		if selected != "" {
			row := communicationRow(after, "observations", "id", selected)
			n, e := evidenceNumber(row["consumed_sequence"])
			must(t, e)
			if history["selected_observation"] != selected || n != next.Sequence {
				t.Fatal("selection/consumption mismatch")
			}
		}
		rec["history"] = history
		d.record(rec)
		return true
	}
	return false
}
func (d *communicationDriver) service() bool {
	t := d.t
	s := inspect(t, d.path)
	progress := false
	for _, row := range s.Tables["intents"] {
		in := decode[intent](row, "descriptor")
		if in.Kind == "invocation" {
			if communicationRow(s, "composition_admission_decisions", "operation_id", in.ID) == nil {
				a, e := d.store.AdmitInvocation(t.Context(), in.ID)
				must(t, e)
				d.record(map[string]any{"kind": "admit_invocation", "operation": in.ID, "result": a})
				progress = true
			}
			p, e := d.store.PrepareInvocation(t.Context(), in.ID)
			must(t, e)
			if p.Ready && communicationRow(s, "composition_outcome_deliveries", "operation_id", in.ID) == nil {
				r, e := d.store.SettleInvocation(t.Context(), in.ID)
				must(t, e)
				d.record(map[string]any{"kind": "settle_invocation", "operation": in.ID, "preparation": p, "result": r})
				progress = true
			}
		}
		if in.Kind == "message" && in.Run.Subject != "peer" && communicationRow(s, "message_acknowledgments", "intent_id", in.ID) == nil {
			d.deliver(in.ID)
			progress = true
		}
	}
	return progress
}
func (d *communicationDriver) toReport() {
	for n := 0; n < 200; n++ {
		if communicationReport(inspect(d.t, d.path)).ID != "" {
			return
		}
		progress := d.service()
		s := inspect(d.t, d.path)
		for _, r := range s.Tables["runs"] {
			id, e := evidenceID(r)
			must(d.t, e)
			if id.Subject != "root" && id.Subject != "peer" {
				progress = d.step(id) || progress
			}
		}
		progress = d.step(runtime.RunID{Subject: "root", Generation: 1}) || progress
		progress = d.step(runtime.RunID{Subject: "peer", Generation: 1}) || progress
		if !progress {
			d.t.Fatal("public driver blocked before original report")
		}
	}
	d.t.Fatal("public prefix operational bound")
}
func (d *communicationDriver) authorize(id string) communication.Authorization {
	s := inspect(d.t, d.path)
	in := decode[intent](communicationRow(s, "intents", "id", id), "descriptor")
	pre, e := d.store.Snapshot(d.t.Context(), in.Run)
	must(d.t, e)
	old := communicationRow(s, "attempts", "intent_id", id)
	a, e := d.store.AuthorizeIntent(d.t.Context(), id)
	must(d.t, e)
	d.record(map[string]any{"kind": "authorize", "intent": id, "pre": pre, "result": a, "repeated": old != nil})
	p := communicationSourceProjection(inspect(d.t, d.path), a.Source.Run, a.Source.Sequence)
	result := communication.Authorization{ID: a.ID, IntentID: a.IntentID, Source: a.Source.Run.Subject, Generation: a.Source.Run.Generation, Epoch: a.Source.Epoch, Revision: a.Source.Revision, Sequence: a.Source.Sequence, SourceStatus: p.Status, Allowed: a.Allowed, Reason: a.Reason}
	if old != nil {
		d.repeats = append(d.repeats, result)
	} else if a.Source != pre.Authority() {
		d.t.Fatal("first authorization differs from actual current authority")
	}
	return result
}
func (d *communicationDriver) deliver(id string) {
	transport := communicationIdentity("message-delivery", id)
	before := inspect(d.t, d.path)
	old := communicationRow(before, "admissions", "transport_id", transport)
	r, e := d.store.DeliverMessage(d.t.Context(), id, transport)
	must(d.t, e)
	must(d.t, d.store.AcknowledgeMessage(d.t.Context(), id, transport))
	after := inspect(d.t, d.path)
	d.record(map[string]any{"kind": "deliver_and_dispatch_ack", "intent": id, "transport": transport, "result": r, "admission": communicationRow(after, "admissions", "transport_id", transport), "ack": communicationRow(after, "message_acknowledgments", "intent_id", id), "repeated": old != nil})
	if old != nil {
		a := communicationRow(after, "attempts", "intent_id", id)
		d.readmissions = append(d.readmissions, communication.Admission{ID: communicationIdentity("admission", "message", transport), IntentID: id, AuthorizationID: a["id"].(string), Accepted: r.Accepted, Reason: r.Reason, ObservationID: r.ObservationID})
	}
}
func (d *communicationDriver) finishPeer() {
	for n := 0; n < 10; n++ {
		if projection(inspect(d.t, d.path), "peer").Status != "open" {
			return
		}
		if !d.step(runtime.RunID{Subject: "peer", Generation: 1}) {
			d.t.Fatal("peer terminal suffix not enabled")
		}
	}
	d.t.Fatal("peer suffix operational bound")
}
func (d *communicationDriver) drive(order string) runtime.Inspection {
	d.toReport()
	s := inspect(d.t, d.path)
	in := communicationReport(s)
	if order == "report-no-authorization" {
		return s
	}
	if order == "early" || order == "report-authorized-peer-terminal" {
		d.authorize(in.ID)
	}
	d.finishPeer()
	if order == "report-authorized-peer-terminal" {
		return inspect(d.t, d.path)
	}
	d.authorize(in.ID)
	d.deliver(in.ID)
	if order != "report-refused-peer-terminal" {
		for n := 0; n < 80; n++ {
			if !d.step(runtime.RunID{Subject: "root", Generation: 1}) {
				break
			}
		}
		s = inspect(d.t, d.path)
		p, e := d.store.PrepareRoot(d.t.Context(), s.Roots[0].RootID)
		must(d.t, e)
		if p.Ready {
			_, e = d.store.SettleRoot(d.t.Context(), s.Roots[0].RootID)
			must(d.t, e)
		}
	}
	s = inspect(d.t, d.path)
	tr, e := decodeCommunication(s, model.Request{ID: 1, Value: 0, Mode: "answer"}, communication.J1)
	must(d.t, e)
	if len(tr.Messages) > 0 {
		tr.Messages[0].Reauthorizations = d.repeats
		tr.Messages[0].Readmissions = d.readmissions
	}
	if v := communication.CheckTrace(tr); len(v) > 0 {
		d.t.Fatal(v)
	}
	d.record(map[string]any{"kind": "final_trace", "trace": tr})
	return s
}
func driveCommunication(t *testing.T, path, protocol, order string) runtime.Inspection {
	t.Helper()
	if protocol != communication.J1 {
		t.Fatal("J2 construction obstructed; see frozen context evidence")
	}
	return newCommunicationDriver(t, path).drive(order)
}
