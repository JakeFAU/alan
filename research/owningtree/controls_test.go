package owningtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

// A deterministic topological history ending just before a specified action. This
// uses the independently declared legal guards. Ties only order the archived witness.
func drive(t *testing.T, g graph, already []string, targets ...string) []string {
	t.Helper()
	history := slices.Clone(already)
	done := map[string]bool{}
	for _, id := range history {
		done[id] = true
	}
	var add func(string)
	add = func(id string) {
		if done[id] {
			return
		}
		e, ok := g[id]
		if !ok {
			t.Fatalf("no such action %s", id)
		}
		for _, need := range e.After {
			add(need)
		}
		history = append(history, id)
		done[id] = true
	}
	for _, id := range targets {
		add(id)
	}
	return history
}

type attack struct {
	Name      string
	Scenario  scenario
	History   []string
	Next      string
	Change    string
	Rejection string
}

func TestTreeCounterexamples(t *testing.T) {
	base := scenario{"fork", "BB", []request{{1, 0, "fail"}, {1, 0, "absent"}}}
	controls := []attack{}
	const fork = "O/tree"
	const l = fork + "/left"
	const r = fork + "/right"
	type mutation struct {
		name, target, change string
		edit                 func(event) event
	}
	needs := func(prior string) func(event) event {
		return func(e event) event { e.After = append(slices.Clone(e.After), prior); return e }
	}
	rules := []mutation{
		{"omitted-sibling", r + ":allow", "requires left first permission too", needs(l + ":allow")},
		{"omitted-post-end-cut", r + "/S:reply_allow", "requires sender End before first permission", needs(r + "/S:end")},
		{"omitted-pre-end-cut", r + "/S:end", "requires sender permission before End", needs(r + "/S:reply_allow")},
		{"wrong-handle", l + ":select", "selects right publication on original left Await", func(e event) event { e.After = []string{r + ":invoke", r + ":publish"}; return e }},
		{"wrong-owner", l + ":select", "attribute selection to sibling", func(e event) event { e.Owner = r; return e }},
		{"wrong-target", r + ":allow", "set sibling permission bit", func(e event) event { e.ID = l + ":allow"; return e }},
		{"snapshot-source", r + "/e/h:reply_allow", "read helper caller instead of sender", func(e event) event { e.SnapshotOf = r + "/e"; return e }},
		{"split-publication", l + ":publish", "delete atomic delivery", func(e event) event { e.Facts = e.Facts[:len(e.Facts)-1]; return e }},
		{"reordered-admission", r + ":admit", "reverse in-transaction word", func(e event) event { e.Facts = slices.Clone(e.Facts); slices.Reverse(e.Facts); return e }},
		{"dropped-diagnosis", fork + ":send", "drop actual selected right absent diagnosis", func(e event) event { e.Facts = []string{"reply_commit", strings.Split(e.Facts[1], ";")[0]}; return e }},
		{"fabricated-diagnosis", fork + ":send", "replace missing_reply by failed", func(e event) event {
			e.Facts = []string{"reply_commit", strings.ReplaceAll(e.Facts[1], "candidate_missing_reply", "candidate_failed")}
			return e
		}},
		{"authentic-wrong-answer", r + "/e/h:publish", "retain original helper operation and successful envelope; replace complement by input", func(e event) event { e.Facts = slices.Clone(e.Facts); e.Facts[2] = "1,0"; return e }},
	}
	for _, m := range rules {
		want, _ := expected(base.root())
		got := submitted(base.root())
		before := drive(t, want, nil, want[m.target].After...)
		got[m.target] = m.edit(got[m.target])
		if m.name == "wrong-handle" {
			before = drive(t, want, nil, got[m.target].After...)
		}
		_, err := audit(want, got)
		if err == nil {
			t.Fatalf("%s passed", m.name)
		}
		controls = append(controls, attack{m.name, base, before, m.target, m.change, err.Error()})
	}

	// Left failed and selected, but right still has its designated open receive.
	pending := scenario{"fork", "AB", []request{{1, 0, "fail"}, {1, 0, "pending"}}}
	want, _ := expected(pending.root())
	got := submitted(pending.root())
	history := drive(t, want, nil, l+":select", r+"/S:select", r+"/e:unmatched")
	extra := event{ID: fork + ":publish", Owner: "store", Facts: []string{"invocation_publish", "member_failed", "", "outcome_fact", "outcome_deliver"}, After: []string{l + ":select"}}
	got[extra.ID] = extra
	_, err := audit(want, got)
	if err == nil {
		t.Fatal("early failed publication accepted")
	}
	controls = append(controls, attack{"early-failure", pending, history, extra.ID, "publish after only failed left selected", err.Error()})

	// The real absent candidate outcome was delivered to J. A detached faulty receiver
	// attempting an absent-field read has no unmatched entry predicate at this cut.
	want, _ = expected(base.root())
	history = drive(t, want, nil, r+"/e:select")
	got = submitted(base.root())
	got[r+":unmatched"] = event{ID: r + ":unmatched", Owner: r, Facts: []string{"pending_receive"}, After: []string{r + "/e:select"}}
	_, err = audit(want, got)
	if err == nil {
		t.Fatal("operand fault mislabeled as legitimate pending")
	}
	controls = append(controls, attack{"fault-as-pending", base, history, r + ":unmatched", "label an absent optional-field read fault as pending_receive", err.Error()})

	for _, cap := range []struct {
		i, r, d        int
		target, reason string
	}{
		{8, 9, 4, r + "/e/h", "invocation_limit"}, {9, 8, 4, r + "/e/h", "created_run_limit"}, {9, 9, 3, l + "/e/h", "depth_limit"},
	} {
		want, _ = expected(base.root())
		var ids []string
		for _, slot := range []string{fork, l, r, l + "/e", r + "/e", l + "/S", r + "/S"} {
			ids = append(ids, slot+":admit")
		}
		ids = append(ids, l+"/e/h:invoke", r+"/e/h:invoke")
		if cap.d == 4 {
			ids = append(ids, l+"/e/h:admit")
		}
		history = drive(t, want, nil, append(ids, cap.target+":allow")...)
		charges := 0
		for _, id := range history {
			if strings.HasSuffix(id, ":admit") {
				charges++
			}
		}
		depth := strings.Count(cap.target, "/")
		reason := "accepted"
		switch {
		case depth > cap.d:
			reason = "depth_limit"
		case charges >= cap.i:
			reason = "invocation_limit"
		case charges >= cap.r:
			reason = "created_run_limit"
		}
		if reason != cap.reason {
			t.Fatalf("got %s with %d charges, depth %d", reason, charges, depth)
		}
		controls = append(controls, attack{cap.reason, base, history, cap.target + ":admit", fmt.Sprintf("I=%d,R=%d,H=%d; pre-spend=%d,depth=%d", cap.i, cap.r, cap.d, charges, depth), reason + "; zero debit"})
	}
	raw, err := json.MarshalIndent(controls, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	const dest = "../../docs/foundational/phase1t/evidence/model-controls.json"
	if os.Getenv("ALAN_WRITE_TREE_CENSUS") == "1" {
		if err = os.WriteFile(dest, raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	retained, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(raw, retained) {
		t.Fatalf("counterexample archive differs: %v", err)
	}
}
