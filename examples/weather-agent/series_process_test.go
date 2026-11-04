package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

func seriesDeclaration(t *testing.T, p *weatherProvider) author.Series {
	t.Helper()
	s, err := (configuration{Latitude: "26.93", Longitude: "-80.08", Units: "fahrenheit", Endpoint: p.endpoint(), Allowance: 8}).series()
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func seriesInspect(t *testing.T, s *runtime.ExecutionSnapshot, p *weatherProvider) author.SeriesView {
	t.Helper()
	v, err := seriesDeclaration(t, p).Inspect(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func completeSeriesThrough(t *testing.T, binary, db string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		if i > 1 {
			weatherCommand(t, binary, "refresh", "--db", db, "--read", fmt.Sprint(i))
		}
		weatherCommand(t, binary, "resume", "--db", db)
	}
}
func preserveSeries(t *testing.T, db, name string) {
	t.Helper()
	dir := os.Getenv("ALAN_SERIES_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(db + suffix)
		if os.IsNotExist(err) && suffix != "" {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".sqlite"+suffix), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func recordSeries(t *testing.T, s *runtime.ExecutionSnapshot, p *weatherProvider, stop any) {
	t.Helper()
	v := seriesInspect(t, s, p)
	approvals := []any{}
	positions := []any{}
	for _, pos := range v.Positions {
		selected := ""
		if pos.Observation != nil {
			selected = pos.Observation.ID
		}
		positions = append(positions, map[string]any{"index": pos.Index, "run": pos.Run.ID, "state": pos.State, "root": pos.Root.RootID, "charges": pos.Root.Charges, "selected_observation": selected})
		if pos.Approval != nil {
			approvals = append(approvals, pos.Approval)
		}
	}
	contacts := []any{}
	for _, c := range p.contacts() {
		contacts = append(contacts, map[string]any{"index": c.index, "operation": c.operation, "client_id": c.clientID, "uri": c.uri, "written": c.written, "status": c.status, "body": c.body})
	}
	b, err := json.Marshal(map[string]any{"case": t.Name(), "namespace": s.Namespace, "series": "weather", "bound": 8, "selected": v.Selected, "completed": v.Completed, "pending": v.Pending, "unresolved": v.Unresolved, "charged": v.Spent, "approvals": approvals, "positions": positions, "effects": weatherFacts(t, s), "contacts": contacts, "interruption": stop})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SERIES_CONTROLLED %s", b)
}
func TestWeatherSeriesProcess(t *testing.T) {
	binary := publicWeatherBinary(t)
	referenceProvider := newWeatherProvider(t, seriesReplies()...)
	referenceDB := startWeather(t, binary, referenceProvider.endpoint(), 8)
	completeSeriesThrough(t, binary, referenceDB, 8)
	reference, _ := inspectAndReplayWeather(t, binary, referenceDB, referenceProvider, 8, 8)
	preserveSeries(t, referenceDB, "ordinary-eight")
	recordSeries(t, reference, referenceProvider, nil)
	cases := []struct {
		name, boundary, phase string
		target, predecessors  int
		approval, lost        bool
	}{
		{"before_fourth_approval", "continuation.approval", "pre-commit", 4, 3, true, false},
		{"after_fourth_approval", "continuation.approval", "post-commit", 4, 3, true, false},
		{"after_fourth_selection", "execution.committed", "post-commit", 4, 3, false, false},
		{"after_fourth_authorization", "intent.authorized", "authorized", 4, 3, false, false},
		{"after_fourth_reservation", "dispatch.consider", "pre-admission", 4, 3, false, true},
		{"after_fourth_gate", "dispatch.admitted", "pre-call", 4, 3, false, true},
		{"third_accepted_response_lost", "", "", 3, 2, false, true},
		{"third_result_retained", "adapter.result-recorded", "pre-settlement", 3, 2, false, false},
		{"third_settled", "adapter.settled", "pre-consumption", 3, 2, false, false},
		{"third_terminal", "execution.committed", "terminal", 3, 2, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			replies := seriesReplies()
			if tc.boundary == "" {
				replies[2].hold = true
			}
			p := newWeatherProvider(t, replies...)
			db := startWeather(t, binary, p.endpoint(), 8)
			completeSeriesThrough(t, binary, db, tc.predecessors)
			for range tc.predecessors {
				p.waitAccepted(t)
			}
			before := weatherSnapshot(t, db)
			if err := run(t.Context(), []string{"refresh", "--db", db, "--read", "5"}, io.Discard); err == nil {
				t.Fatal("fifth approval without paid fourth")
			}
			e := weatherEffect(t, reference, readID(tc.target).Subject)
			a, _ := reference.Continuation(e.Source)
			r, _ := reference.Run(e.Source)
			action := "resume"
			if tc.approval {
				action = fmt.Sprintf("approve-%d", tc.target)
			} else {
				weatherCommand(t, binary, "refresh", "--db", db, "--read", fmt.Sprint(tc.target))
			}
			var owner *weatherOwner
			var witness string
			if tc.boundary == "" {
				owner = launchWeather(t, binary, db, action, nil)
				contact := p.waitAccepted(t)
				if contact.index != 3 || contact.operation != e.OperationID {
					t.Fatal("wrong accepted third operation")
				}
				witness = "independent provider accepted third original operation; response held"
			} else {
				filter := weatherCut{Subject: readID(tc.target).Subject, Generation: 1, Boundary: tc.boundary, Phase: tc.phase}
				switch tc.boundary {
				case "continuation.approval":
					filter.Intent = weatherEffect(t, before, readID(tc.target-1).Subject).OperationID
					filter.Attempt = declaredWeatherSlot(t, before, tc.target).ApprovalID
				case "execution.committed":
					if tc.phase == "terminal" {
						filter.Sequence = r.Sequence
					} else {
						filter.Sequence = a.SelectedAt.Sequence
					}
				default:
					filter.Intent = e.OperationID
				}
				owner = launchWeather(t, binary, db, action, &filter)
				witness = owner.awaitCut(t).witness()
			}
			owner.kill(t, witness)
			stopped := weatherSnapshot(t, db)
			sv := seriesInspect(t, stopped, p)
			inspectAndReplayWeather(t, binary, db, p, 8, sv.Spent)
			checkSeriesCut(t, tc.name, sv, tc.target, p)
			for i := 0; i < tc.predecessors; i++ {
				if !reflect.DeepEqual(weatherEffect(t, before, readID(i+1).Subject), sv.Positions[i].Effect) {
					t.Fatal("cut changed an earlier original effect")
				}
			}
			if sv.Selected > tc.target || sv.Spent > tc.target {
				t.Fatal("cut exceeded specifically approved position")
			}
			preserveSeries(t, db, tc.name+"-stopped")
			stop := weatherInterruption(t, witness, owner.cmd.Process.Pid, len(p.contacts()), stopped)
			stop["approvals"] = sv.Positions[tc.target-1].Approval
			for range 2 {
				weatherCommand(t, binary, "resume", "--db", db)
			}
			recovered := weatherSnapshot(t, db)
			if got := len(p.contacts()); got != tc.predecessors && tc.lost && tc.target == 4 {
				t.Fatal("reopen contacted provider for spent fourth")
			}
			if tc.target == 3 && len(p.contacts()) != 3 {
				t.Fatal("third recovery performed an additional contact")
			}
			v := seriesInspect(t, recovered, p)
			if tc.name == "before_fourth_approval" {
				if v.Spent != 3 || len(p.contacts()) != 3 || v.Positions[3].Approval != nil {
					t.Fatal("owner supplied absent fourth approval")
				}
			} else if tc.lost {
				wantUnresolved := 1
				if tc.name == "after_fourth_reservation" {
					wantUnresolved = 0
				}
				if v.Spent != tc.target || v.Completed != tc.predecessors || v.Unresolved != wantUnresolved {
					t.Fatalf("lost original was refunded or fabricated: %+v", v)
				}
				if v.Positions[tc.target-1].Effect.OperationID != e.OperationID || v.Positions[tc.target-1].Observation != nil {
					t.Fatal("unknown result became selected")
				}
			} else if v.Completed != tc.target || v.Spent != tc.target {
				t.Fatalf("retained authorized work did not finish: %+v", v)
			}
			for i := 0; i < tc.predecessors; i++ {
				old := weatherEffect(t, before, readID(i+1).Subject)
				now := v.Positions[i].Effect
				if !reflect.DeepEqual(old, now) {
					t.Fatal("earlier evidence changed")
				}
			}
			// A third unknown may be followed only by a separately approved fourth;
			// all fifth through eighth positions still require their own decisions.
			if tc.target == 3 || tc.name == "before_fourth_approval" {
				for range 2 {
					weatherCommand(t, binary, "refresh", "--db", db, "--read", "4")
				}
				for range 2 {
					weatherCommand(t, binary, "resume", "--db", db)
				}
			} else {
				for range 2 {
					weatherCommand(t, binary, "refresh", "--db", db, "--read", "4")
				}
			}
			final := weatherSnapshot(t, db)
			fv := seriesInspect(t, final, p)
			inspectAndReplayWeather(t, binary, db, p, 8, 4)
			wantContacts := 4
			if tc.lost && tc.target == 4 {
				wantContacts = 3
			}
			p.requireContacts(t, wantContacts)
			if fv.Spent != 4 || fv.Positions[4].State != author.SeriesAwaitingApproval {
				t.Fatalf("fourth approval spilled authority: %+v", fv)
			}
			for _, pos := range fv.Positions[4:] {
				if pos.Approval != nil || pos.Effect != nil || pos.Observation != nil {
					t.Fatal("unapproved future work")
				}
			}
			for _, pos := range fv.Positions[5:] {
				if pos.State != author.SeriesNotEligible {
					t.Fatal("future slot eligible without paid predecessor")
				}
			}
			for i, pos := range sv.Positions {
				if pos.Effect != nil && pos.Effect.AdapterResult != nil && !reflect.DeepEqual(pos.Effect.AdapterResult, fv.Positions[i].Effect.AdapterResult) {
					t.Fatal("retained result bytes changed")
				}
				if pos.Approval != nil && (pos.Approval.Approval != fv.Positions[i].Approval.Approval || pos.Approval.DecisionOrder != fv.Positions[i].Approval.DecisionOrder) {
					t.Fatal("approval scope changed")
				}
			}
			if tc.name == "third_accepted_response_lost" && (fv.Completed != 3 || fv.Unresolved != 1 || fv.Positions[2].Observation != nil || fv.Positions[3].Observation == nil) {
				t.Fatal("replacement fabricated or refunded lost third result")
			}
			preserveSeries(t, db, tc.name+"-final")
			recordSeries(t, final, p, stop)
		})
	}
}

func checkSeriesCut(t *testing.T, name string, v author.SeriesView, target int, p *weatherProvider) {
	t.Helper()
	position := v.Positions[target-1]
	expected := map[string]author.SeriesState{
		"before_fourth_approval":       author.SeriesAwaitingApproval,
		"after_fourth_approval":        author.SeriesApproved,
		"after_fourth_selection":       author.SeriesApproved,
		"after_fourth_authorization":   author.SeriesAuthorized,
		"after_fourth_reservation":     author.SeriesSpentBeforeDispatch,
		"after_fourth_gate":            author.SeriesOutcomeUnknown,
		"third_accepted_response_lost": author.SeriesOutcomeUnknown,
		"third_result_retained":        author.SeriesResultUnselected,
		"third_settled":                author.SeriesResultUnselected,
		"third_terminal":               author.SeriesSelected,
	}[name]
	if position.State != expected {
		t.Fatalf("stopped state %s, want %s", position.State, expected)
	}
	paid := name == "after_fourth_reservation" || name == "after_fourth_gate" || target == 3
	wantSpent := target - 1
	if paid {
		wantSpent++
	}
	wantContacts := target - 1
	if target == 3 {
		wantContacts++
	}
	p.requireContacts(t, wantContacts)
	if v.Spent != wantSpent || position.Root.Charges.Effects != int64(wantSpent-target+1) {
		t.Fatal("cut lost exact scoped charge")
	}
	if (position.Approval != nil) != (name != "before_fourth_approval") {
		t.Fatal("cut approval presence differs")
	}
	if position.Approval != nil {
		selected := name != "after_fourth_approval"
		if (position.Approval.SelectedAt != nil) != selected {
			t.Fatal("approval selection does not match cut")
		}
		predecessor := v.Positions[target-2].Effect
		basis := position.Approval.Approval
		if basis.PredecessorID != predecessor.OperationID || basis.PreparedDigest != predecessor.AdapterAttempt.Prepared.Digest || basis.ResultDigest != predecessor.AdapterResult.Digest || basis.Namespace != v.Namespace || basis.Target != position.Run.ID || basis.Reason != afterReport || !basis.Admitted {
			t.Fatal("approval lost original evidence scope")
		}
	}
	effectExpected := paid || name == "after_fourth_authorization"
	if (position.Effect != nil) != effectExpected {
		t.Fatal("cut has incorrect operation presence")
	}
	if e := position.Effect; e != nil {
		gate := name == "after_fourth_gate" || target == 3
		result := name == "third_result_retained" || name == "third_settled" || name == "third_terminal"
		settled := name == "third_settled" || name == "third_terminal"
		if e.Authorization == nil || !e.Authorization.Allowed || (e.AdapterAttempt != nil) != paid || (e.DispatchAdmission != nil) != gate || (e.AdapterResult != nil) != result || (e.Settlement != nil) != settled || (position.Observation != nil) != (name == "third_terminal") {
			t.Fatal("cut differs from original authorization/reservation/gate/result/selection boundary")
		}
	}
	for _, future := range v.Positions[target:] {
		if future.Effect != nil || future.Approval != nil || future.Observation != nil {
			t.Fatal("cut contains unapproved future work")
		}
	}
}
