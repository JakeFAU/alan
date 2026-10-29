//go:build integration

package substitutiontest

import (
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"testing"
)

func runLiveEffects(t *testing.T) {
	for _, profile := range []string{"reserved-then-closure", "late-authentic-receipt"} {
		t.Run(profile, func(t *testing.T) {
			d, path := effectDeployment(t, true, true)
			create(t, d, path)
			ep, server := newEndpoint(t, true)
			cut := ""
			if profile == "reserved-then-closure" {
				cut = "effect-reserved"
			}
			p := start(t, path, d.Candidate.ID(), cut, server.URL, false, true)
			if cut != "" {
				p.acknowledgment(t)
				if contacts(ep) != 0 {
					t.Fatal("transport began before reserved pause")
				}
			} else {
				await(t, ep.entered, "original applied request")
			}
			before := inspect(t, path)
			if before.Roots[0].Charges.Effects != 1 || len(before.Tables["adapter_results"]) != 0 {
				t.Fatal("expected one reserved missing receipt")
			}
			retain(t, "before", map[string]any{"inspection": before, "endpoint": ep.snapshot()})
			p.cancelLive(t)
			closed := inspect(t, path)
			if projection(closed, "root").Status != "cancelled" || len(closed.Tables["lifecycle_results"]) != 1 {
				t.Fatal("root not closed")
			}
			decision := decode[executor.LifecycleResult](closed.Tables["lifecycle_results"][0], "result")
			if !decision.Accepted || decision.Reason != "closed" {
				t.Fatal(decision)
			}
			retain(t, "closure-before-response", map[string]any{"inspection": closed, "endpoint": ep.snapshot()})
			if cut != "" {
				p.resume(t)
				await(t, ep.entered, "allowed reserved transport after closure")
			}
			ep.unblock()
			p.wait(t, false)
			after := inspect(t, path)
			retain(t, "after-unchecked", map[string]any{"inspection": after, "endpoint": ep.snapshot()})
			stablePrefix(t, before, after)
			contactMonitor(t, after, ep)
			if contacts(ep) != 1 || len(after.Tables["adapter_results"]) != 2 || after.Roots[0].Charges.Effects != 1 || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "cancelled" {
				t.Fatal("late authentic evidence invariant", after.Roots, ep.snapshot())
			}
			member := ""
			for _, row := range after.Tables["composition_admission_decisions"] {
				a := decode[runtime.InvocationAdmission](row, "descriptor")
				if a.Invocation.TemplateID == d.Candidate.ID() {
					member = a.Members[0].Subject
				}
			}
			q := projection(after, member)
			if !q.Data["effect_valid"].Flag || q.Data["effect_value"].Number != 1 || q.Data["second_valid"].Flag {
				t.Fatal("late receipt not consumed correctly", q)
			}
			finalDenied := false
			for _, row := range after.Tables["composition_reservations"] {
				r := decode[struct {
					Allowed bool
					Reason  string
				}](row, "descriptor")
				if !r.Allowed && r.Reason == "scope_closed" {
					finalDenied = true
				}
			}
			if !finalDenied {
				t.Fatal("second scope reservation not denied")
			}
			first := q.Data["effect_operation"].Text
			second := q.Data["second_operation"].Text
			known, denied, sourceAllowed := false, false, false
			for _, row := range after.Tables["adapter_results"] {
				r := decode[runtime.AdapterResult](row, "descriptor")
				if r.Candidate.ID == first && r.Disposition == "validated" && r.Candidate.HTTPStatus == 200 && r.Values["valid"].Flag {
					known = true
				}
				if r.Candidate.ID == second && r.Disposition == "source_denied" && !r.Values["valid"].Flag {
					denied = true
				}
			}
			for _, row := range after.Tables["attempts"] {
				a := decode[struct {
					IntentID string
					Allowed  bool
				}](row, "result")
				if a.IntentID == second && a.Allowed {
					sourceAllowed = true
				}
			}
			if !known || !denied || !sourceAllowed {
				t.Fatal("original known receipt or distinct final denial/source authorization absent")
			}

			retain(t, "after", map[string]any{"inspection": after, "endpoint": ep.snapshot(), "late_receipt_on_original_request": true, "source_authorization_distinct_from_final_reservation": true, "fixed_runs": 2})
		})
	}
}
