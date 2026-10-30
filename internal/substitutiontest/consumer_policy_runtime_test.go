//go:build integration

package substitutiontest

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	policyapp "github.com/JakeFAU/alan/examples/consumingpolicy"
	app "github.com/JakeFAU/alan/examples/substitution"
	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

// Reuse authenticated original records; only the context verdicts differ.
func decodeConsumerPolicy(s runtime.Inspection, c model.JoinConfig) (invocationJoinTrace, error) {
	out, err := decodeInvocationJoinRaw(s, c)
	if err != nil {
		return out, err
	}
	out.Prefix = model.CheckConsumerPolicyPrefix(c, out.Events)
	out.Component = model.CheckJoinComponent(c, out.Events)
	out.Guarantee = model.CheckConsumerPolicyGuarantee(c, out.Events)
	return out, nil
}

func consumerPolicyFixture(t *testing.T, name, sMode string, c model.JoinConfig) (policyapp.Deployment, string) {
	t.Helper()
	config := app.Config{
		Request: app.Request{ID: int64(c.Request.ID), Value: int64(c.Request.Value), Mode: c.Request.Mode},
		Limits:  limits(c.I, c.R, c.Depth),
	}
	d, err := policyapp.NewSFirst(candidate(t, name), config, policyapp.Options{SMode: sMode})
	must(t, err)
	path := filepath.Join(t.TempDir(), "retained.db")
	store, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	must(t, err)
	must(t, store.Close())
	return d, path
}

func consumerPolicyConfig() model.JoinConfig {
	return model.JoinConfig{Request: model.Request{ID: 1, Value: 0, Mode: "answer"}, I: 3, R: 3, Depth: 2}
}

func consumerPolicyOriginal(t *testing.T, s runtime.Inspection, c model.JoinConfig) invocationJoinTrace {
	t.Helper()
	trace, err := decodeConsumerPolicy(s, c)
	must(t, err)
	if !trace.RawValid || len(trace.Prefix) != 0 {
		retain(t, "rejected-original", map[string]any{"inspection": s, "config": c, "trace": trace})
		t.Fatal("authentic C_se prefix rejected", trace)
	}
	return trace
}

// Preserve separate member termination, publication, delivery and consumption
// records. Zero order/sequence denotes an event that has not occurred.
type consumerPolicyOperation struct {
	Operation, InvocationID, OutcomeObservation, RequestObservation string
	Member                                                          runtime.RunID
	MemberStatus                                                    string
	MemberSequence, MemberTerminalOrder, PublicationOrder           int64
	ConsumptionSequence, ConsumptionOrder, RequestConsumedSequence  int64
}

func consumerPolicyCausality(t *testing.T, s runtime.Inspection, trace invocationJoinTrace) map[string]consumerPolicyOperation {
	t.Helper()
	result := map[string]consumerPolicyOperation{}
	for role, op := range trace.Handles {
		row := communicationRow(s, "composition_admission_decisions", "operation_id", op)
		if row == nil {
			continue
		}
		admission := decode[runtime.InvocationAdmission](row, "descriptor")
		item := consumerPolicyOperation{Operation: op, InvocationID: admission.Invocation.ID}
		if admission.Accepted {
			if len(admission.Members) != 1 {
				t.Fatal("expected original singleton member")
			}
			item.Member = admission.Members[0]
			for _, run := range s.Tables["runs"] {
				id, err := evidenceID(run)
				must(t, err)
				if id == item.Member {
					item.MemberStatus = decode[runtime.Projection](run, "projection").Status
					item.MemberSequence, err = evidenceNumber(run["sequence"])
					must(t, err)
				}
			}
			request := communicationRow(s, "composition_requests", "invocation_id", item.InvocationID)
			item.RequestObservation = request["observation_id"].(string)
			observation := communicationRow(s, "observations", "id", item.RequestObservation)
			if observation["consumed_sequence"] != nil {
				var err error
				item.RequestConsumedSequence, err = evidenceNumber(observation["consumed_sequence"])
				must(t, err)
			}
		}
		delivery := communicationRow(s, "composition_outcome_deliveries", "operation_id", op)
		if delivery != nil {
			item.OutcomeObservation = delivery["observation_id"].(string)
			observation := communicationRow(s, "observations", "id", item.OutcomeObservation)
			if observation["consumed_sequence"] != nil {
				var err error
				item.ConsumptionSequence, err = evidenceNumber(observation["consumed_sequence"])
				must(t, err)
			}
		}
		for _, row := range s.Tables["composition_facts"] {
			fact := decode[invocationJoinFact](row, "descriptor")
			if fact.Kind == "freeze" && fact.Run == item.Member {
				item.MemberTerminalOrder = fact.Order
			}
			if fact.Kind == "outcome" && fact.OperationID == op {
				item.PublicationOrder = fact.Order
			}
			if fact.Kind == "execution" && fact.Run == admission.Invocation.Caller &&
				item.ConsumptionSequence != 0 && fact.Sequence == item.ConsumptionSequence {
				item.ConsumptionOrder = fact.Order
			}
		}
		result[role] = item
	}
	return result
}

func consumerPolicyComplete(t *testing.T, s runtime.Inspection, c model.JoinConfig, spending int64) invocationJoinTrace {
	t.Helper()
	trace := consumerPolicyOriginal(t, s, c)
	if len(trace.Component) != 0 || len(trace.Guarantee) != 0 {
		t.Fatal("C_se positive rejected", trace)
	}
	charges := s.Roots[0].Charges
	if charges.Invocations != spending || charges.CreatedRuns != spending || charges.Effects != 0 {
		t.Fatal("unexpected actual spending", charges)
	}
	causal := consumerPolicyCausality(t, s, trace)
	if causal["s"].ConsumptionSequence != 3 || causal["e"].ConsumptionSequence != 4 {
		t.Fatal("original owning parent did not consume S then e", causal)
	}
	return trace
}

func TestConsumerPolicyRuntime(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		for _, config := range model.ConsumerPolicyConfigurations() {
			t.Run(fmt.Sprintf("%s/id%d-value%d", name, config.Request.ID, config.Request.Value), func(t *testing.T) {
				deployment, path := consumerPolicyFixture(t, name, "answer", config)
				report := run(t, path)
				raw := inspect(t, path)
				spending := int64(2)
				if name == "b" {
					spending = 3
				}
				trace := consumerPolicyComplete(t, raw, config, spending)
				if deployment.FixedRuns != 1 || len(raw.Tables["runs"]) != int(spending)+1 {
					t.Fatal("fixed/dynamic run accounting")
				}
				retain(t, "result", map[string]any{
					"kind": "automatic", "candidate_name": name, "config": config,
					"inspection": raw, "trace": trace, "report": report,
					"causality":  consumerPolicyCausality(t, raw, trace),
					"fixed_runs": deployment.FixedRuns, "candidate": deployment.Candidate.ID(),
					"successor": deployment.Successor.ID(), "parent": deployment.Parent.ID(),
				})
				replayUnchanged(t, path)
			})
		}
	}
}

func TestConsumerPolicyControls(t *testing.T) {
	for _, name := range []string{"candidate-pending", "s-pending", "wrong-id", "budget2"} {
		t.Run(name, func(t *testing.T) {
			config := consumerPolicyConfig()
			candidateName, sMode := "b", "answer"
			switch name {
			case "candidate-pending":
				config.Request.Mode = "pending"
			case "s-pending":
				sMode = "pending"
			case "wrong-id":
				candidateName = "wrong-id"
			case "budget2":
				config.I, config.R = 2, 2
			}
			_, path := consumerPolicyFixture(t, candidateName, sMode, config)
			report := run(t, path)
			raw := inspect(t, path)
			trace := consumerPolicyOriginal(t, raw, config)
			causal := consumerPolicyCausality(t, raw, trace)
			root := projection(raw, "root")
			if len(trace.Guarantee) == 0 {
				t.Fatal("control incorrectly certified G_answer")
			}
			if name == "wrong-id" {
				if len(trace.Component) == 0 {
					t.Fatal("actual wrong reply conformance violation missed")
				}
				found := false
				for _, event := range trace.Events {
					fact := event.Fact
					if fact.Kind == "outcome_consume" && fact.Operation == "e" {
						found = fact.Status == "succeeded" && fact.Present && fact.ID == 2 && fact.Value == 1
					}
				}
				if !found {
					t.Fatal("actual mutant consumed envelope not retained")
				}
			} else if len(trace.Component) != 0 {
				t.Fatal("component control rejected", trace.Component)
			}
			switch name {
			case "candidate-pending":
				if root.Status != "open" || root.PC != 3 || raw.Roots[0].Outcome != nil ||
					causal["s"].ConsumptionSequence != 3 || causal["e"].PublicationOrder != 0 ||
					causal["e"].ConsumptionSequence != 0 || !pendingReceive(raw, causal["e"].Member.Subject) {
					t.Fatal("candidate pending does not retain required original wait", causal, root)
				}
			case "s-pending":
				if root.Status != "open" || root.PC != 2 || raw.Roots[0].Outcome != nil ||
					causal["e"].PublicationOrder == 0 || causal["s"].PublicationOrder != 0 ||
					causal["e"].ConsumptionSequence != 0 || causal["s"].ConsumptionSequence != 0 ||
					!pendingReceive(raw, causal["s"].Member.Subject) {
					t.Fatal("S pending does not retain first required await", causal, root)
				}
			default:
				if root.Status != "failed" || raw.Roots[0].Outcome == nil || raw.Roots[0].Outcome.Status != "failed" ||
					causal["s"].ConsumptionSequence != 3 || causal["e"].ConsumptionSequence != 4 {
					t.Fatal("defensive failure skipped original joins", causal, root)
				}
			}
			refused := []model.Observation{}
			for _, event := range trace.Events {
				if event.Fact.Kind == "admit" && event.Fact.Status == "rejected" {
					refused = append(refused, event.Fact)
				}
			}
			if name == "budget2" {
				charges := raw.Roots[0].Charges
				if len(refused) != 1 || refused[0].Reason != "invocation_limit" ||
					refused[0].I != 0 || refused[0].R != 0 || charges.Invocations != 2 || charges.CreatedRuns != 2 {
					t.Fatal("scarcity evidence", refused, charges)
				}
			}
			retain(t, "result", map[string]any{
				"kind": "control", "control": name, "config": config, "inspection": raw,
				"trace": trace, "causality": causal, "report": report, "actual_refused": refused,
			})
			replayUnchanged(t, path)
		})
	}
}

func TestConsumerPolicyOrders(t *testing.T) {
	for _, order := range []string{"s-first", "candidate-first"} {
		t.Run(order, func(t *testing.T) {
			config := consumerPolicyConfig()
			_, path := consumerPolicyFixture(t, "b", "answer", config)
			driver := newCommunicationDriver(t, path)
			root := runtime.RunID{Subject: "root", Generation: 1}
			if !driver.step(root) || !driver.step(root) {
				t.Fatal("original invocation commitments absent")
			}
			trace := consumerPolicyOriginal(t, inspect(t, path), config)
			invocationJoinAdmit(t, driver, trace.Handles["e"])
			invocationJoinAdmit(t, driver, trace.Handles["s"])
			state, spending := "unproduced", int64(2)
			if order == "candidate-first" {
				invocationJoinComplete(t, driver, trace.Handles["e"])
				state, spending = "retained-unconsumed", 3
			}
			invocationJoinComplete(t, driver, trace.Handles["s"])
			if !driver.step(root) {
				t.Fatal("original S outcome not selected")
			}
			before := inspect(t, path)
			prefix := consumerPolicyOriginal(t, before, config)
			causal := consumerPolicyCausality(t, before, prefix)
			parent := projection(before, "root")
			charges := before.Roots[0].Charges
			if parent.Status != "open" || parent.PC != 3 || before.Roots[0].Outcome != nil ||
				causal["s"].ConsumptionSequence != 3 || causal["e"].ConsumptionSequence != 0 ||
				len(prefix.Component) != 0 || len(prefix.Guarantee) == 0 ||
				charges.Invocations != spending || charges.CreatedRuns != spending || charges.Effects != 0 {
				t.Fatal("S-consumed/e-required-open cut absent", causal, parent, charges)
			}
			if order == "s-first" {
				if causal["e"].PublicationOrder != 0 || causal["e"].OutcomeObservation != "" ||
					causal["e"].MemberSequence != 0 || causal["e"].MemberStatus != "open" ||
					causal["e"].RequestConsumedSequence != 0 || prefix.Handles["h"] != "" {
					t.Fatal("unproduced original e cut absent", causal)
				}
			} else if causal["e"].PublicationOrder == 0 ||
				causal["e"].PublicationOrder >= causal["s"].PublicationOrder || causal["e"].OutcomeObservation == "" {
				t.Fatal("retained unconsumed original e cut absent", causal)
			}
			acknowledgment := map[string]any{
				"predicate":       "S consumed once; original e required and unconsumed; parent open at e await",
				"candidate_state": state, "root": root, "pc": parent.PC, "sequence": int64(3),
				"handles": prefix.Handles, "charges": charges, "inspection_hash": hash(before),
			}
			driver.record(map[string]any{"kind": "consumer_policy_checked_cut", "ack": acknowledgment, "causality": causal})
			retain(t, "cut", map[string]any{
				"inspection": before, "trace": prefix, "config": config,
				"ack": acknowledgment, "causality": causal, "order": order,
			})
			report, err := driver.store.Run(t.Context(), runtime.RunOptions{MaxSteps: 200, MaxDispatch: 200, MaxInFlight: 1})
			must(t, err)
			if !report.Quiescent {
				t.Fatal("remaining original work exhausted bound", report)
			}
			after := inspect(t, path)
			final := consumerPolicyComplete(t, after, config, 3)
			completed := consumerPolicyCausality(t, after, final)
			sFirst := completed["s"].PublicationOrder < completed["e"].PublicationOrder
			if sFirst != (order == "s-first") {
				t.Fatal("wrong relative original invocation publication order", completed)
			}
			stablePrefix(t, before, after)
			for _, old := range before.Tables["observations"] {
				if old["consumed_sequence"] != nil &&
					!reflect.DeepEqual(old, communicationRow(after, "observations", "id", old["id"])) {
					t.Fatal("original consumed sequence changed", old["id"])
				}
			}
			retain(t, "result", map[string]any{
				"kind": "public-schedule", "order": order, "config": config,
				"inspection": after, "trace": final, "causality": completed, "report": report,
			})
			replayUnchanged(t, path)
		})
	}
}

// Check the old wrapper against the archived exact observations before and
// after extraction. This reads the historical archive; it never rewrites it.
func TestConsumerPolicyHistoricalWrapper(t *testing.T) {
	path := "../../docs/foundational/phase1l/evidence/runtime/records.json.gz"
	file, err := os.Open(path)
	must(t, err)
	defer file.Close()
	archive, err := gzip.NewReader(file)
	must(t, err)
	defer archive.Close()
	var bundle struct {
		Profiles map[string]map[string]json.RawMessage `json:"profiles"`
	}
	must(t, json.NewDecoder(archive).Decode(&bundle))
	hashes := map[string]string{}
	for profile, records := range bundle.Profiles {
		for key, value := range records {
			var record struct {
				Inspection *runtime.Inspection  `json:"inspection"`
				Trace      *invocationJoinTrace `json:"trace"`
			}
			if json.Unmarshal(value, &record) != nil || record.Inspection == nil || record.Trace == nil {
				continue
			}
			raw := *record.Inspection
			initial := decode[runtime.Projection](communicationRow(raw, "runs", "subject", "root"), "initial")
			bounds := raw.Roots[0].Limits
			config := model.JoinConfig{
				Request: model.Request{
					ID: int(initial.Data["request_id"].Number), Value: int(initial.Data["request_value"].Number),
					Mode: initial.Data["request_mode"].Text,
				},
				I: int(bounds.MaxInvocations), R: int(bounds.MaxCreatedRuns), Depth: int(bounds.MaxDepth),
			}
			actual, err := decodeInvocationJoin(raw, config)
			must(t, err)
			if !reflect.DeepEqual(actual, *record.Trace) {
				t.Fatal("historical wrapper changed archived verdict/events/handles", profile, key, actual, *record.Trace)
			}
			hashes[profile+"/"+key] = hash(actual)
		}
	}
	if len(hashes) == 0 {
		t.Fatal("no historical inspections checked")
	}
	keys := make([]string, 0, len(hashes))
	for key := range hashes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	t.Logf("historical wrapper exact archived equality: %d inspections", len(hashes))
	retain(t, "equivalence", map[string]any{"archive": path, "count": len(hashes), "trace_hashes": hashes, "keys": keys})
}
