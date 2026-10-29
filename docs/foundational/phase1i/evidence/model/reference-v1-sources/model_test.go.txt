package substitution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func emit(t *testing.T, name string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("ALAN_MODEL_EVIDENCE"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func TestReferenceProducts(t *testing.T) {
	configs := Configurations()
	if len(configs) != 3840 {
		t.Fatal(len(configs))
	}
	var records []Census
	total := Census{Name: "K1"}
	for _, c := range configs {
		r := Explore(c, Reference)
		records = append(records, r)
		total.Configurations++
		total.Entitled += r.Entitled
		total.States += r.States
		total.Transitions += r.Transitions
		total.Revisits += r.Revisits
		total.Rejections += r.Rejections
		total.Terminal += r.Terminal
		total.Pending += r.Pending
		total.Stuck += r.Stuck
		total.RankFailures += r.RankFailures
		total.Cutoffs += r.Cutoffs
		if len(r.Counterexamples) > 0 {
			total.Counterexamples = append(total.Counterexamples, r.Counterexamples...)
			if len(total.Counterexamples) < 8 {
				t.Errorf("%s: %+v", r.Name, r.Counterexamples)
			}
		}
	}
	emit(t, "reference-census", records)
	emit(t, "reference-summary", total)
	if total.Entitled != 84 {
		t.Errorf("entitled=%d want84", total.Entitled)
	}
	t.Logf("reference census: %+v", total)
}
func TestReferenceBoundaries(t *testing.T) {
	c := Config{Shape: "bare", Request: Request{1, 0, "answer"}, I: 3, R: 3, Depth: 3}
	seen := map[State]bool{}
	queue := []State{Initial(c)}
	kinds := map[string]bool{}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if seen[s] {
			continue
		}
		seen[s] = true
		for _, e := range Successors(c, Reference, s) {
			kinds[e.Event.Kind] = true
			queue = append(queue, e.Next)
		}
	}
	for _, kind := range []string{"invoke_commit", "admit", "request_consume", "reply_commit", "reply_admit", "terminal_freeze", "publish", "outcome_deliver", "outcome_consume"} {
		if !kinds[kind] {
			t.Errorf("missing boundary %s", kind)
		}
	}
}

func TestReferenceOpenEnvironment(t *testing.T) {
	configs := LocalConfigurations()
	if len(configs) != 3072 {
		t.Fatal(len(configs))
	}
	total := Census{Name: "K1-open-environment"}
	var failures []Census
	relation := RelationResult{}
	for _, c := range configs {
		r := Explore(c, Reference)
		total.Configurations++
		total.Entitled += r.Entitled
		total.States += r.States
		total.Transitions += r.Transitions
		total.Revisits += r.Revisits
		total.Rejections += r.Rejections
		total.Terminal += r.Terminal
		total.Pending += r.Pending
		total.Stuck += r.Stuck
		total.RankFailures += r.RankFailures
		if len(r.Counterexamples) > 0 {
			failures = append(failures, r)
		}
		check := CheckLocalRelation(c, func(c Config, s State) []Edge { return Successors(c, Reference, s) })
		relation.Initials += check.Initials
		relation.Pairs += check.Pairs
		relation.ComponentChecks += check.ComponentChecks
		relation.EnvironmentChecks += check.EnvironmentChecks
		relation.Failures += check.Failures
	}
	emit(t, "reference-open-environment", total)
	emit(t, "reference-relation-self", relation)
	if len(failures) > 0 {
		emit(t, "reference-open-failures", failures)
		t.Fatalf("%d failed configs; first %+v", len(failures), failures[0])
	}
	if relation.Failures != 0 {
		t.Fatal(relation)
	}
	t.Logf("open census: %+v", total)
}

func TestReferenceLocalProgressWithoutClosure(t *testing.T) {
	total := Census{Name: "K1-local-no-closure"}
	for _, c := range LocalConfigurations() {
		c.Closure = false
		r := Explore(c, Reference)
		total.Configurations++
		total.Entitled += r.Entitled
		total.States += r.States
		total.Transitions += r.Transitions
		total.Revisits += r.Revisits
		total.Rejections += r.Rejections
		total.Terminal += r.Terminal
		total.Pending += r.Pending
		total.Stuck += r.Stuck
		total.RankFailures += r.RankFailures
		if len(r.Counterexamples) > 0 {
			t.Fatalf("%s %+v", r.Name, r.Counterexamples)
		}
	}
	emit(t, "reference-local-progress", total)
	t.Logf("local progress: %+v", total)
}
