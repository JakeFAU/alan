//go:build integration

package substitutiontest

import (
	"encoding/json"
	"testing"

	model "github.com/JakeFAU/alan/research/substitution"
	"github.com/JakeFAU/alan/runtime"
)

func consumerPolicyOrderViolation(violations []model.Violation) bool {
	for _, violation := range violations {
		if violation.Code == "consume_order" {
			return true
		}
	}
	return false
}

func TestConsumerPolicyDecoder(t *testing.T) {
	// Authentic executions are valid raw records even when the selected policy
	// rejects their actual consumption order. No events are reordered here.
	for _, context := range []string{"historical-es", "s-first-se"} {
		t.Run(context, func(t *testing.T) {
			config := consumerPolicyConfig()
			var path string
			if context == "historical-es" {
				_, path, config = invocationJoinFixture(t, "a", "answer", "answer", 3)
			} else {
				_, path = consumerPolicyFixture(t, "a", "answer", config)
			}
			run(t, path)
			raw := inspect(t, path)
			historical, err := decodeInvocationJoin(raw, config)
			must(t, err)
			current, err := decodeConsumerPolicy(raw, config)
			must(t, err)
			accepted, rejected := historical, current
			if context == "s-first-se" {
				accepted, rejected = current, historical
			}
			if !accepted.RawValid || !rejected.RawValid || len(accepted.Prefix) != 0 ||
				len(accepted.Component) != 0 || len(accepted.Guarantee) != 0 || len(rejected.Component) != 0 ||
				!consumerPolicyOrderViolation(rejected.Prefix) || !consumerPolicyOrderViolation(rejected.Guarantee) {
				t.Fatal("raw/policy order distinction lost", accepted, rejected)
			}
			retain(t, "result", map[string]any{
				"kind": "policy-negative", "context": context, "inspection": raw, "config": config,
				"historical": historical, "s_first": current,
			})
		})
	}
	t.Run("swapped-selected-outcomes", func(t *testing.T) {
		config := consumerPolicyConfig()
		_, path := consumerPolicyFixture(t, "b", "answer", config)
		run(t, path)
		original := inspect(t, path)
		consumerPolicyOriginal(t, original, config)
		var changed runtime.Inspection
		bytes, err := json.Marshal(original)
		must(t, err)
		must(t, json.Unmarshal(bytes, &changed))
		selected := []map[string]any{}
		for _, row := range changed.Tables["history"] {
			if row["subject"] == "root" && row["selected_observation"] != nil {
				selected = append(selected, row)
			}
		}
		if len(selected) != 2 {
			t.Fatal("expected both original parent selections")
		}
		selected[0]["selected_observation"], selected[1]["selected_observation"] =
			selected[1]["selected_observation"], selected[0]["selected_observation"]
		for _, row := range selected {
			observation := communicationRow(changed, "observations", "id", row["selected_observation"])
			observation["consumed_sequence"] = row["sequence"]
			invocationJoinRehashHistory(t, &changed, row)
		}
		actual, err := decodeConsumerPolicy(changed, config)
		if err == nil || actual.RawValid {
			t.Fatal("coherent detached selection swap accepted", actual)
		}
		retain(t, "rejection", map[string]any{
			"kind": "decoder-negative", "mutation": "swapped-selected-outcomes", "error": err.Error(),
			"source_inspection": original, "mutated_inspection": changed,
		})
	})
	t.Run("delivery-is-not-consumption", func(t *testing.T) {
		config := consumerPolicyConfig()
		_, path := consumerPolicyFixture(t, "b", "answer", config)
		driver := newCommunicationDriver(t, path)
		root := runtime.RunID{Subject: "root", Generation: 1}
		if !driver.step(root) || !driver.step(root) {
			t.Fatal("missing original commitments")
		}
		prefix := consumerPolicyOriginal(t, inspect(t, path), config)
		invocationJoinAdmit(t, driver, prefix.Handles["e"])
		invocationJoinAdmit(t, driver, prefix.Handles["s"])
		invocationJoinComplete(t, driver, prefix.Handles["s"])
		original := inspect(t, path)
		prefix = consumerPolicyOriginal(t, original, config)
		causal := consumerPolicyCausality(t, original, prefix)
		if causal["s"].PublicationOrder == 0 || causal["s"].ConsumptionSequence != 0 ||
			projection(original, "root").PC != 2 || len(prefix.Guarantee) == 0 {
			t.Fatal("original delivered-unconsumed prefix absent")
		}
		var changed runtime.Inspection
		bytes, err := json.Marshal(original)
		must(t, err)
		must(t, json.Unmarshal(bytes, &changed))
		observation := communicationRow(changed, "observations", "id", causal["s"].OutcomeObservation)
		observation["consumed_sequence"] = int64(3)
		actual, err := decodeConsumerPolicy(changed, config)
		if err == nil || actual.RawValid {
			t.Fatal("delivery promoted to actual consumption", actual)
		}
		retain(t, "rejection", map[string]any{
			"kind": "decoder-negative", "mutation": "delivery-is-not-consumption", "error": err.Error(),
			"source_inspection": original, "source_trace": prefix, "mutated_inspection": changed,
		})
	})
}
