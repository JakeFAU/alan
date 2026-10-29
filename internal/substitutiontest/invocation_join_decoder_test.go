//go:build integration

package substitutiontest

import (
	"encoding/json"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"testing"
)

// Catches raw identity/selection/authorization corruption before normalization;
// mutates only detached inspection copies, never authoritative retained state.
func TestInvocationJoinDecoder(t *testing.T) {
	_, path, c := invocationJoinFixture(t, "b", "answer", "answer", 3)
	run(t, path)
	original := inspect(t, path)
	tr := invocationJoinOriginal(t, original, c)
	raw, e := json.Marshal(original)
	must(t, e)
	for _, name := range []string{"coherent-revision", "orphan-intent", "unadmitted-revision", "original-handle", "selected-identity", "selected-payload", "missing-delivery", "missing-request", "request-source", "authorization-sequence", "reply-authorization", "duplicate-selection", "extra-charge", "fact-order", "program-hash", "duplicate-program", "parent-pure-path", "request-bound-payload", "unconsumed-request-obligation"} {
		t.Run(name, func(t *testing.T) {
			var s runtime.Inspection
			must(t, json.Unmarshal(raw, &s))
			changed := false
			switch name {
			case "coherent-revision":
				invocationJoinForgeRevision(t, &s, tr.Handles["e"], 2)
				changed = true
			case "orphan-intent":
				original := communicationRow(s, "intents", "id", tr.Handles["e"])
				copied := map[string]any{}
				for key, value := range original {
					copied[key] = value
				}
				orphan := decode[executor.Intent](original, "descriptor")
				orphan.ID = "uncommitted-extra-invocation"
				orphan.Sequence = 999
				copied["id"] = orphan.ID
				copied["sequence"] = int64(999)
				invocationJoinWriteJSON(t, copied, "descriptor", orphan)
				s.Tables["intents"] = append(s.Tables["intents"], copied)
				changed = true
			case "unadmitted-revision":
				_, path, cfg := invocationJoinFixture(t, "b", "answer", "answer", 3)
				driver := newCommunicationDriver(t, path)
				if !driver.step(runtime.RunID{Subject: "root", Generation: 1}) {
					t.Fatal("commit e")
				}
				s = inspect(t, path)
				prefix := invocationJoinOriginal(t, s, cfg)
				if len(s.Tables["composition_admission_decisions"]) != 0 || len(s.Tables["attempts"]) != 0 {
					t.Fatal("not committed-unadmitted prefix")
				}
				invocationJoinForgeRevision(t, &s, prefix.Handles["e"], 2)
				changed = true

			case "original-handle", "selected-payload":
				for _, r := range s.Tables["history"] {
					if r["subject"] != "root" {
						continue
					}
					p := decode[runtime.Projection](r, "successor")
					if name == "original-handle" && p.Data["candidate_operation"].Text != "" {
						v := p.Data["candidate_operation"]
						v.Text = tr.Handles["s"]
						p.Data["candidate_operation"] = v
					} else if name == "selected-payload" && r["selected_observation"] != nil {
						v := p.Data["candidate_id"]
						v.Number = 2
						p.Data["candidate_id"] = v
					} else {
						continue
					}
					b, e := json.Marshal(p)
					must(t, e)
					r["successor"] = string(b)
					changed = true
					break
				}
			case "selected-identity":
				for _, r := range s.Tables["history"] {
					if r["subject"] == "root" && r["selected_observation"] != nil {
						r["selected_observation"] = "unrelated-observation"
						changed = true
						break
					}
				}
			case "missing-delivery":
				s.Tables["composition_outcome_deliveries"] = s.Tables["composition_outcome_deliveries"][1:]
				changed = true
			case "missing-request":
				s.Tables["composition_requests"] = s.Tables["composition_requests"][1:]
				changed = true
			case "request-source":
				for _, r := range s.Tables["observations"] {
					o := decode[map[string]any](r, "descriptor")
					if o["Provenance"] == "runtime-invocation-request" {
						o["Source"] = "unrelated-caller"
						b, e := json.Marshal(o)
						must(t, e)
						r["descriptor"] = string(b)
						changed = true
						break
					}
				}
			case "authorization-sequence":
				for _, r := range s.Tables["attempts"] {
					a := decode[map[string]any](r, "result")
					src := a["Source"].(map[string]any)
					src["Sequence"] = float64(0)
					b, e := json.Marshal(a)
					must(t, e)
					r["result"] = string(b)
					changed = true
					break
				}
			case "reply-authorization":
				r := s.Tables["composition_replies"][0]
				reply := decode[map[string]any](r, "descriptor")
				reply["Authorization"].(map[string]any)["Allowed"] = false
				b, e := json.Marshal(reply)
				must(t, e)
				r["descriptor"] = string(b)
				changed = true
			case "duplicate-selection":
				for _, r := range s.Tables["observations"] {
					if r["consumed_sequence"] != nil {
						s.Tables["observations"] = append(s.Tables["observations"], r)
						changed = true
						break
					}
				}
			case "program-hash":
				r := s.Tables["programs"][0]
				r["bytes"] = r["bytes"].(string) + " "
				changed = true
			case "duplicate-program":
				s.Tables["programs"] = append(s.Tables["programs"], s.Tables["programs"][0])
				changed = true
			case "parent-pure-path":
				for _, r := range s.Tables["history"] {
					if r["subject"] != "root" {
						continue
					}
					p := decode[runtime.Projection](r, "successor")
					if p.PC == 5 {
						p.PC = 7
						b, e := json.Marshal(p)
						must(t, e)
						r["successor"] = string(b)
						invocationJoinRehashHistory(t, &s, r)
						changed = true
						break
					}
				}
			case "request-bound-payload":
				for _, r := range s.Tables["history"] {
					selected, _ := r["selected_observation"].(string)
					if selected == "" {
						continue
					}
					o := decode[observation](communicationRow(s, "observations", "id", selected), "descriptor")
					if o.Provenance != "runtime-invocation-request" {
						continue
					}
					p := decode[runtime.Projection](r, "successor")
					for key, v := range p.Data {
						if v.Kind == "int" {
							v.Number = 9
							p.Data[key] = v
							break
						}
					}
					b, e := json.Marshal(p)
					must(t, e)
					r["successor"] = string(b)
					invocationJoinRehashHistory(t, &s, r)
					changed = true
					break
				}
			case "unconsumed-request-obligation":
				// The initial request remains unconsumed in a separate public cut.
				d, path, cfg := invocationJoinFixture(t, "b", "answer", "answer", 3)
				_ = d
				driver := newCommunicationDriver(t, path)
				if !driver.step(runtime.RunID{Subject: "root", Generation: 1}) {
					t.Fatal("commit")
				}
				prefix := invocationJoinOriginal(t, inspect(t, path), cfg)
				invocationJoinAdmit(t, driver, prefix.Handles["e"])
				s = inspect(t, path)
				for _, row := range s.Tables["composition_requests"] {
					o := decode[observation](row, "descriptor")
					o.Obligation = "spoofed"
					b, e := json.Marshal(o)
					must(t, e)
					row["descriptor"] = string(b)
					communicationRow(s, "observations", "id", o.ID)["descriptor"] = string(b)
					changed = true
					break
				}

			case "extra-charge":
				copied := map[string]any{}
				for key, value := range s.Tables["composition_charges"][0] {
					copied[key] = value
				}
				copied["operation_id"] = "uncommitted-extra-charge"
				charge := decode[map[string]any](copied, "descriptor")
				charge["OperationID"] = "uncommitted-extra-charge"
				b, e := json.Marshal(charge)
				must(t, e)
				copied["descriptor"] = string(b)
				s.Tables["composition_charges"] = append(s.Tables["composition_charges"], copied)
				s.Roots[0].Charges.Invocations++
				s.Roots[0].Charges.CreatedRuns++
				changed = true

			case "fact-order":
				r := s.Tables["composition_facts"][0]
				r["decision_order"] = float64(0)
				changed = true
			}
			if !changed {
				t.Fatal("mutation target absent")
			}
			decoded, e := decodeInvocationJoin(s, c)
			if e == nil || decoded.RawValid {
				retain(t, "accepted-corruption", map[string]any{"mutation": name, "inspection": s, "decoded": decoded})
				t.Fatal("raw corruption accepted", name)
			}
			retain(t, "rejection", map[string]any{"mutation": name, "error": e.Error(), "source_inspection_hash": hash(original)})
		})
	}
	// The untouched root/selected rows are still the authentic public execution.
	if len(tr.Prefix) != 0 || len(tr.Component) != 0 || len(tr.Guarantee) != 0 {
		t.Fatal(tr)
	}
	retain(t, "original", map[string]any{"inspection": original, "trace": tr})
}

// Rehash a detached copy so path/binding attacks cannot be rejected only because
// their source digest is stale. No live retained row is edited.
func invocationJoinRehashHistory(t *testing.T, s *runtime.Inspection, row map[string]any) {
	t.Helper()
	id, e := evidenceID(row)
	must(t, e)
	seq, e := evidenceNumber(row["sequence"])
	must(t, e)
	epoch, e := evidenceNumber(row["epoch"])
	must(t, e)
	revision, e := evidenceNumber(row["source_revision"])
	must(t, e)
	selected, _ := row["selected_observation"].(string)
	source := struct {
		Run                       runtime.RunID
		Sequence, Epoch, Revision int64
		Selected                  string
		Successor, Intents        json.RawMessage
	}{id, seq, epoch, revision, selected, json.RawMessage(row["successor"].(string)), json.RawMessage(row["intents"].(string))}
	for _, frow := range s.Tables["composition_facts"] {
		f := decode[invocationJoinFact](frow, "descriptor")
		if f.Kind == "execution" && f.Run == id && f.Sequence == seq {
			f.RecordDigest = invocationJoinDigest("alan/history-row/v1", source)
			b, e := json.Marshal(f)
			must(t, e)
			frow["descriptor"] = string(b)
			return
		}
	}
	t.Fatal("original execution fact missing")
}

func TestInvocationJoinIncompletePrefixes(t *testing.T) {
	_, path, c := invocationJoinFixture(t, "b", "answer", "answer", 3)
	d := newCommunicationDriver(t, path)
	statuses := []map[string]any{}
	observe := func(stage string) {
		s := inspect(t, path)
		tr := invocationJoinOriginal(t, s, c)
		if len(tr.Component) != 0 || len(tr.Guarantee) == 0 {
			t.Fatal("prefix promoted to completion", stage, tr)
		}
		statuses = append(statuses, map[string]any{"stage": stage, "inspection_hash": hash(s), "trace": tr})
	}
	observe("initial")
	if !d.step(runtime.RunID{Subject: "root", Generation: 1}) {
		t.Fatal("commit e")
	}
	observe("e-committed-unadmitted")
	tr := invocationJoinOriginal(t, inspect(t, path), c)
	invocationJoinAdmit(t, d, tr.Handles["e"])
	observe("e-admitted-request-unconsumed")
	if !d.step(runtime.RunID{Subject: "root", Generation: 1}) {
		t.Fatal("commit s")
	}
	observe("both-committed-s-unadmitted")
	// First authorization may observe a renewed epoch at an unchanged retained
	// sequence. Revision authentication must preserve this legitimate case.
	before := inspect(t, path)
	tr = invocationJoinOriginal(t, before, c)
	must(t, d.store.Close())
	reopened := newCommunicationDriver(t, path)
	invocationJoinAdmit(t, reopened, tr.Handles["s"])
	after := inspect(t, path)
	a := decode[executor.Authorization](communicationRow(after, "attempts", "intent_id", tr.Handles["s"]), "result")
	committed := communicationRow(before, "history", "sequence", int64(2))
	oldEpoch, e := evidenceNumber(committed["epoch"])
	must(t, e)
	if a.Source.Sequence != 2 || a.Source.Epoch <= oldEpoch || a.Source.Revision != 1 {
		t.Fatal("first authorization lacks legitimate renewed epoch", a)
	}
	observe("s-admitted-after-epoch-renewal")
	retain(t, "prefixes", statuses)
}

// Coherently alter the mutually agreeing source rows, preserving the ordered
// authorization digest. The actual subject/commitment/admission remain original.
func invocationJoinForgeRevision(t *testing.T, s *runtime.Inspection, op string, revision int64) {
	t.Helper()
	row := communicationRow(*s, "intents", "id", op)
	if row == nil {
		t.Fatal("original intent missing")
	}
	row["source_revision"] = revision
	for _, row := range s.Tables["attempts"] {
		if row["intent_id"] != op {
			continue
		}
		request := decode[map[string]any](row, "descriptor")
		request["Revision"] = revision
		invocationJoinWriteJSON(t, row, "descriptor", request)
		a := decode[executor.Authorization](row, "result")
		a.Source.Revision = revision
		invocationJoinWriteJSON(t, row, "result", a)
		for _, frow := range s.Tables["composition_facts"] {
			f := decode[invocationJoinFact](frow, "descriptor")
			if f.Kind == "authorization" && f.OperationID == op {
				f.RecordDigest = invocationJoinDigest("alan/authorization-row/v1", json.RawMessage(row["result"].(string)))
				invocationJoinWriteJSON(t, frow, "descriptor", f)
			}
		}
	}
}
func invocationJoinWriteJSON(t *testing.T, row map[string]any, key string, value any) {
	t.Helper()
	raw, e := json.Marshal(value)
	must(t, e)
	row[key] = string(raw)
}
