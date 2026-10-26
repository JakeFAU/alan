package semantics

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// The baseline delivered terminal notifications outside ordinary inboxes. This
// retained input is the first behavioral RED for the new ordinary boundary.
func TestSupervisionTerminalBecomesOrdinaryObservation(t *testing.T) {
	c := fixture([]instruction{{Op: fault}}).Config
	c.Programs["observer"] = []instruction{{Op: receive, Tag: "terminal"}, {Op: success}}
	c.Routes["a"] = "observer"
	c.Edges["a>observer"] = true
	m := newMachine(c)
	turn(t, &m, "a", 1, "")
	mustStep(t, &m, event{Kind: deliver, A: "a", G: 1})
	if len(m.Runs[runID("observer", 1)].Inbox) != 1 {
		t.Fatal("frozen child failure did not become an ordinary observation")
	}
}

func TestSupervisionQualification(t *testing.T) {
	sources := compositionSources(t)
	t.Log("SUPERVISION " + canonical(struct {
		Name     string
		Manifest sourceManifest
	}{"source_manifest", sources}))
	for _, spec := range supervisionScenarios() {
		t.Run(spec.Name, func(t *testing.T) {
			stats, err := exploreSupervision(spec)
			stats.SourceID = sources.SHA256
			seed, _ := replayCompositionSeed(spec)
			environmental := []event{}
			for _, e := range append(append([]event{}, spec.Seed...), spec.Alphabet...) {
				if e.Kind == reactivate || e.Kind == abandon || e.Kind == cancel || e.Kind == retire || e.Kind == retry || e.Kind == suspect {
					environmental = append(environmental, e)
				}
			}
			t.Log("SUPERVISION " + canonical(struct {
				compositionStats
				RequiredWitnesses, RequiredRejections []string
				SeedMonitor                           monitor
				EnvironmentalStimuli                  []event
			}{stats, spec.WantWitnesses, spec.WantRejects, seed.Monitor, environmental}))
			if err != nil {
				t.Fatal(err)
			}
			if stats.Violation != spec.WantViolation {
				t.Fatalf("violation=%q want=%q", stats.Violation, spec.WantViolation)
			}
			for _, w := range spec.WantWitnesses {
				trace, ok := stats.Witnesses[w]
				if !ok {
					t.Errorf("missing witness %s", w)
					continue
				}
				verifySupervisionWitnessReplay(t, spec, trace)
			}
			for _, why := range spec.WantRejects {
				if stats.Rejections[why] == 0 {
					t.Errorf("missing rejection %s", why)
				}
			}
		})
	}
	supervisionCompleteWitnesses(t, sources)
	supervisionFaultyRuntimeWitness(t, sources)
	t.Log("SUPERVISION " + canonical(supervisionInterruptionOracleRegression(t, sources.SHA256)))
}

// Complete traces are reachability examples, independently replayed through all
// transition monitors. They are not advertised as exhaustive searches.
func supervisionCompleteWitnesses(t *testing.T, sources sourceManifest) {
	var examples []compositionSpec
	b := recoveredSeed(true)
	b.local("worker", 2, "")
	terminalDecision(&b, "inner", "worker", 1, 2, "")
	terminalDecision(&b, "outer", "inner", 1, 1, "")
	lifecycleDispatch(&b, intentID("outer", 1, 10), true)
	acceptLifecycle(&b, "outer", 1, intentID("outer", 1, 10))
	examples = append(examples, compositionSpec{Name: "complete_nested_exhaustion_outer_recovery", Config: b.Config, Seed: b.Events})
	b = traceBuilder{Config: supervisionFixture(false)}
	b.Config.Programs["worker"] = []instruction{{Op: success}}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	b.add(event{Kind: deliver, A: "worker", G: 1})
	b.local("inner", 1, terminalID("worker", 1, ""))
	b.local("inner", 1, "")
	b.local("inner", 1, "")
	examples = append(examples, compositionSpec{Name: "complete_worker_success", Config: b.Config, Seed: b.Events})
	b = traceBuilder{Config: supervisionFixture(false)}
	b.Config.Programs["inner"] = supervisorProgram("worker", 1, true)
	b.Config.Supervision.Policies["inner"] = policyContract{Child: "worker", Limit: 1, AllowAbandoned: true}
	supervisorInit(&b, "inner", 1)
	b.add(life(abandon, "worker", 1), event{Kind: deliver, A: "worker", G: 1})
	b.local("inner", 1, terminalID("worker", 1, ""))
	for range 7 {
		b.local("inner", 1, "")
	}
	lifecycleDispatch(&b, intentID("inner", 1, 13), true)
	acceptLifecycle(&b, "inner", 1, intentID("inner", 1, 13))
	examples = append(examples, compositionSpec{Name: "complete_authorized_abandonment_recovery", Config: b.Config, Seed: b.Events})
	b = traceBuilder{Config: supervisionFixture(false)}
	supervisorInit(&b, "inner", 1)
	b.add(life(cancel, "worker", 1), event{Kind: deliver, A: "worker", G: 1})
	b.local("inner", 1, terminalID("worker", 1, ""))
	for range 6 {
		b.local("inner", 1, "")
	}
	examples = append(examples, compositionSpec{Name: "complete_cancelled_worker_escalation", Config: b.Config, Seed: b.Events})
	b = recoveredSeed(false)
	b.Config.Programs["inner"] = supervisorProgram("worker", 2, false)
	b.Config.Supervision.Policies["inner"] = policyContract{Child: "worker", Limit: 2}
	b.local("worker", 2, "")
	terminalDecision(&b, "inner", "worker", 1, 2, "")
	lifecycleDispatch(&b, intentID("inner", 1, 19), true)
	acceptLifecycle(&b, "inner", 1, intentID("inner", 1, 19))
	b.local("worker", 3, "")
	terminalDecision(&b, "inner", "worker", 1, 3, "")
	examples = append(examples, compositionSpec{Name: "complete_budget_two_exhaustion", Config: b.Config, Seed: b.Events})
	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			n, err := replayCompositionSeed(example)
			if err != nil {
				t.Fatal(err)
			}
			before := canonical(n.Machine)
			for id := range n.Machine.Runs {
				if err := replay(n.Machine, id, n.Machine.Config.Version); err != nil {
					t.Fatalf("replay %s: %v", id, err)
				}
			}
			if canonical(n.Machine) != before {
				t.Fatal("replay dispatched or changed machine")
			}
			t.Log("SUPERVISION " + canonical(struct {
				Name, Category, SourceID, ConfigID, ScenarioID string
				Config                                         config
				Trace                                          []event
				Final                                          machine
			}{example.Name, "complete-reachability", sources.SHA256, digest(example.Config), digest(example), example.Config, example.Seed, n.Machine}))
		})
	}
}

func TestSupervisionBoundaryContracts(t *testing.T) {
	t.Run("ingress_cannot_claim_runtime_provenance", func(t *testing.T) {
		b := traceBuilder{Config: supervisionFixture(false)}
		b.Config.Inputs["spoof"] = message{Source: "worker", Target: "inner", Tag: "child", Value: 2}
		supervisorInit(&b, "inner", 1)
		n, err := replayCompositionSeed(compositionSpec{Config: b.Config, Seed: b.Events})
		if err != nil {
			t.Fatal(err)
		}
		mustStep(t, &n.Machine, event{Kind: admit, ID: "spoof"})
		mustReject(t, &n.Machine, event{Kind: prepare, A: "inner", G: 1, C: 1, Observation: messageObservationID("spoof")}, "provenance")
	})
	t.Run("source_authorization_and_target_grant_are_distinct", func(t *testing.T) {
		b := rawRequestSeed()
		n, err := replayCompositionSeed(compositionSpec{Config: b.Config, Seed: b.Events})
		if err != nil {
			t.Fatal(err)
		}
		id := intentID("inner", 1, 6)
		mustReject(t, &n.Machine, event{Kind: authorize, ID: id, Owner: "owner", W: 7, Attempt: 1}, "lifecycle authorization required")
		mustReject(t, &n.Machine, event{Kind: authorizeLifecycle, ID: id, Owner: "intruder", W: 7}, "source ownership")
		mustStep(t, &n.Machine, event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7})
		mustStep(t, &n.Machine, life(abandon, "inner", 1))
		mustStep(t, &n.Machine, life(abandon, "worker", 1))
		mustStep(t, &n.Machine, event{Kind: decideLifecycle, ID: id})
		if n.Machine.Subjects["worker"].Current != 2 {
			t.Fatal("authorized request was lost on requester closure")
		}
		mustStep(t, &n.Machine, event{Kind: admitLifecycle, ID: id})
		if len(n.Machine.Runs[runID("inner", 1)].Inbox) != 0 {
			t.Fatal("closed requester reopened")
		}
	})
	t.Run("world_does_not_choose_ordinary_recovery", func(t *testing.T) {
		b := recoveredSeed(false)
		n, err := replayCompositionSeed(compositionSpec{Config: b.Config, Seed: b.Events})
		if err != nil {
			t.Fatal(err)
		}
		other := copyOf(n.Machine)
		other.World.Mutations = 7
		other.World.Keys["unobserved-business-effect"] = 1
		e := event{Kind: prepare, A: "worker", G: 2, C: 1}
		a, whyA := step(n.Machine, e)
		z, whyZ := step(other, e)
		if whyA != whyZ || canonical(a.Proposals) != canonical(z.Proposals) {
			t.Fatal("external world changed ordinary evaluation")
		}
	})
	t.Run("stale_target_revision_retains_rejection", func(t *testing.T) {
		b := rawRequestSeed()
		b.Config.Programs["inner"][3].Value = 6
		b.Config.Supervision.Grants[0].W = 6
		n, err := replayCompositionSeed(compositionSpec{Config: b.Config, Seed: b.Events})
		if err != nil {
			t.Fatal(err)
		}
		id := intentID("inner", 1, 6)
		mustStep(t, &n.Machine, event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7})
		mustStep(t, &n.Machine, event{Kind: decideLifecycle, ID: id})
		if n.Machine.Lifecycle.Results[id].Reason != "target ownership revision" {
			t.Fatal("missing target revision rejection")
		}
		mustStep(t, &n.Machine, event{Kind: decideLifecycle, ID: id})
	})
}

func supervisionFaultyRuntimeWitness(t *testing.T, sources sourceManifest) {
	for _, spec := range supervisionScenarios() {
		if spec.Name != "faulty_ordinary_budget_branch" {
			continue
		}
		n, err := replayCompositionSeed(spec)
		if err != nil {
			t.Fatal(err)
		}
		trace := append([]event{}, spec.Seed...)
		// Deliberately retain the faulty program's behavior after the independent
		// monitor reports its policy violation. Runtime guards remain unchanged.
		suffix := []event{{Kind: prepare, A: "inner", G: 1, C: 1, Observation: terminalID("worker", 2, "")}, {Kind: commit}}
		for range 4 {
			suffix = append(suffix, event{Kind: prepare, A: "inner", G: 1, C: 1}, event{Kind: commit})
		}
		id := intentID("inner", 1, 19)
		suffix = append(suffix, event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, event{Kind: decideLifecycle, ID: id})
		firstBad := ""
		for _, e := range suffix {
			next, why := step(n.Machine, e)
			if why != "" {
				t.Fatalf("faulty runtime rejected %s: %s", e.Kind, why)
			}
			mon, bad := inspect(n.Machine, next, e, n.Monitor)
			if firstBad == "" {
				firstBad = bad
			}
			n = compositionNode{Machine: next, Monitor: mon}
			trace = append(trace, e)
		}
		if firstBad != "supervision-retry-budget" || n.Machine.Subjects["worker"].Current != 3 {
			t.Fatal("runtime secretly enforced ordinary retry budget")
		}
		t.Log("SUPERVISION " + canonical(struct {
			Name, Category, SourceID, Violation string
			Config                              config
			Trace                               []event
			Final                               machine
		}{"faulty_program_runtime_generation_three", "program-fault-reachability", sources.SHA256, firstBad, spec.Config, trace, n.Machine}))
	}
}

func TestSupervisionUncommittedIntentInterruption(t *testing.T) {
	b := traceBuilder{Config: supervisionFixture(false)}
	supervisorInit(&b, "inner", 1)
	b.local("worker", 1, "")
	terminalDecision(&b, "inner", "worker", 1, 1, "")
	// Keep the proposal to Emit, but discard its final Commit.
	b.Events = b.Events[:len(b.Events)-1]
	n, err := replayCompositionSeed(compositionSpec{Config: b.Config, Seed: b.Events})
	if err != nil {
		t.Fatal(err)
	}
	id := intentID("inner", 1, 10)
	if n.Machine.Intents[id] != nil {
		t.Fatal("speculative proposal emitted intent")
	}
	mustStep(t, &n.Machine, event{Kind: interrupt})
	mustReject(t, &n.Machine, event{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, "uncommitted")
	if n.Machine.Runs[runID("inner", 1)].Projection.Local.Values[spent] != 0 {
		t.Fatal("interrupted proposal spent budget")
	}
}

func TestSupervisionPublicNameRenaming(t *testing.T) {
	b := recoveredSeed(true)
	b.local("worker", 2, "")
	terminalDecision(&b, "inner", "worker", 1, 2, "")
	terminalDecision(&b, "outer", "inner", 1, 1, "")
	lifecycleDispatch(&b, intentID("outer", 1, 10), true)
	spec := compositionSpec{Config: b.Config, Seed: b.Events}
	original, err := replayCompositionSeed(spec)
	if err != nil {
		t.Fatal(err)
	}
	rename := strings.NewReplacer("worker", "delta", "inner", "epsilon", "outer", "zeta")
	var renamed compositionSpec
	if err := json.Unmarshal([]byte(rename.Replace(canonical(spec))), &renamed); err != nil {
		t.Fatal(err)
	}
	actual, err := replayCompositionSeed(renamed)
	if err != nil {
		t.Fatal(err)
	}
	var expected machine
	if err := json.Unmarshal([]byte(rename.Replace(canonical(original.Machine))), &expected); err != nil {
		t.Fatal(err)
	}
	if canonical(actual.Machine) != canonical(expected) {
		t.Fatal("public naming changed interpreter behavior")
	}
}

func verifySupervisionWitnessReplay(t *testing.T, spec compositionSpec, trace []event) {
	t.Helper()
	n, err := replayCompositionSeed(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range trace {
		next, _ := step(n.Machine, e)
		mon, bad := inspect(n.Machine, next, e, n.Monitor)
		if bad != "" {
			t.Fatal(bad)
		}
		n = compositionNode{Machine: next, Monitor: mon}
	}
	before := canonical(n.Machine)
	for id := range n.Machine.Runs {
		if err := replay(n.Machine, id, n.Machine.Config.Version); err != nil {
			t.Fatalf("witness replay %s: %v", id, err)
		}
	}
	if canonical(n.Machine) != before {
		t.Fatal("witness replay dispatched")
	}
}

// Retain the reviewer's exact R2 input from source 0f507d1d. This is a
// witness-oracle regression: every runtime transition and safety check is valid.
type supervisionOracleRecord struct {
	Name, Category, SourceID, ReviewedSourceID, ConfigID, ScenarioID string
	Config                                                           config
	Seed, Suffix                                                     []event
	ForbiddenWitness                                                 string
	ObservedWitnesses                                                []string
	RuntimeViolation                                                 string
	SeedMonitor, FinalMonitor                                        monitor
}

func supervisionInterruptionOracleRegression(t *testing.T, sourceID string) supervisionOracleRecord {
	t.Helper()
	var spec compositionSpec
	for _, candidate := range supervisionScenarios() {
		if candidate.Name == "H_nested_boundary_lost_ack_interrupt" {
			spec = candidate
			break
		}
	}
	if len(spec.Seed) != 68 || digest(spec.Seed) != "5c388af47e006ff6f98990430419c3096e9fe851627ed8f86ee18d816d225222" || digest(spec.Config) != "d41916267a46918a4492bbc3f47f0c0268b30e7e3f3e21ae43a6b12c27cc64c5" {
		t.Fatal("review regression fixture changed")
	}
	id := intentID("outer", 1, 10)
	suffix := []event{
		{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7},
		{Kind: decideLifecycle, ID: id},
		life(reactivate, "outer", 1),
		{Kind: prepare, A: "inner", G: 2, C: 1},
		{Kind: interrupt},
	}
	n, err := replayCompositionSeed(spec)
	if err != nil {
		t.Fatal(err)
	}
	record := supervisionOracleRecord{Name: "oracle_regression_interruption_execution_and_order", Category: "witness-oracle-regression", SourceID: sourceID, ReviewedSourceID: "0f507d1dc22c6faf403de916e0ccbcb4ca5cff1fb8680140ed863fe760caee1b", ConfigID: digest(spec.Config), Config: spec.Config, Seed: spec.Seed, Suffix: suffix, ForbiddenWitness: "committed-intent-interrupted-reactivated", SeedMonitor: n.Monitor, ObservedWitnesses: []string{}}
	record.ScenarioID = digest(struct {
		Config       config
		Seed, Suffix []event
	}{spec.Config, spec.Seed, suffix})
	found := map[string]bool{}
	for _, e := range suffix {
		next, why := step(n.Machine, e)
		if why != "" {
			t.Fatalf("review suffix rejected %s: %s", e.Kind, why)
		}
		mon, bad := inspect(n.Machine, next, e, n.Monitor)
		if bad != "" {
			t.Fatalf("review suffix runtime safety violation: %s", bad)
		}
		for _, w := range supervisionWitness(n.Machine, next, e, why, &mon) {
			found[w] = true
		}
		n = compositionNode{Machine: next, Monitor: mon}
	}
	record.FinalMonitor = n.Monitor
	for w := range found {
		record.ObservedWitnesses = append(record.ObservedWitnesses, w)
	}
	slices.Sort(record.ObservedWitnesses)
	if found[record.ForbiddenWitness] {
		t.Fatal("oracle falsely joined outer/g1 reactivation with later inner/g2 interruption")
	}
	return record
}

func TestSupervisionInterruptionWitnessCorrelation(t *testing.T) {
	supervisionInterruptionOracleRegression(t, "")
}

func TestSupervisionInterruptionWitnessOrdering(t *testing.T) {
	var spec compositionSpec
	for _, candidate := range supervisionScenarios() {
		if candidate.Name == "H_nested_boundary_lost_ack_interrupt" {
			spec = candidate
			break
		}
	}
	id := intentID("outer", 1, 10)
	for _, ordered := range []bool{false, true} {
		name := "reactivate_before_interrupt"
		if ordered {
			name = "interrupt_before_reactivate"
		}
		t.Run(name, func(t *testing.T) {
			n, err := replayCompositionSeed(spec)
			if err != nil {
				t.Fatal(err)
			}
			suffix := []event{{Kind: authorizeLifecycle, ID: id, Owner: "owner", W: 7}, {Kind: decideLifecycle, ID: id}, {Kind: admitLifecycle, ID: id}}
			proposal := event{Kind: prepare, A: "outer", G: 1, C: 1, Observation: resultID(id, "")}
			if ordered {
				suffix = append(suffix, proposal, event{Kind: interrupt}, life(reactivate, "outer", 1))
			} else {
				suffix = append(suffix, life(reactivate, "outer", 1), proposal, event{Kind: interrupt})
			}
			found := false
			for _, e := range suffix {
				next, why := step(n.Machine, e)
				if why != "" {
					t.Fatal(why)
				}
				mon, bad := inspect(n.Machine, next, e, n.Monitor)
				if bad != "" {
					t.Fatal(bad)
				}
				for _, w := range supervisionWitness(n.Machine, next, e, why, &mon) {
					if w == "committed-intent-interrupted-reactivated" {
						found = true
					}
				}
				n = compositionNode{Machine: next, Monitor: mon}
			}
			if found != ordered {
				t.Fatalf("combined witness=%t, want=%t", found, ordered)
			}
			if n.Machine.Runs[runID("outer", 1)].Projection.Local.Pending != id {
				t.Fatal("pending identity changed")
			}
		})
	}
}

func TestSupervisionHardCapFails(t *testing.T) {
	spec := compositionSpec{Name: "hard_cap_probe", Config: supervisionFixture(false), Alphabet: []event{{Kind: prepare, A: "inner", G: 1, C: 1}}, Depth: 8, StateCap: 1}
	if _, err := exploreSupervision(spec); err == nil {
		t.Fatal("supervision hard state cap exhaustion was silently accepted")
	}
}
