//go:build integration

package adaptivetest

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/adaptive"
	"github.com/JakeFAU/alan/runtime"
)

func TestProcessAdaptiveCuts(t *testing.T) {
	c := component(t, false)
	for _, nested := range []bool{false, true} {
		for _, cut := range []string{"decision", "intents", "partial"} {
			t.Run(fmt.Sprintf("nested=%v/%s", nested, cut), func(t *testing.T) {
				d, path := provision(t, c, nested, 1, nil)
				create(t, d, path, true)
				p := start(t, path, c.ID(), cut, "", false, "")
				ack := p.acknowledgment(t)
				before := inspect(t, path)
				must(t, durableAck(before, ack))
				must(t, cutMonitor(before, c.ID(), cut, ack))
				monitorControls(t, before, ack, p.filter, p.cmd.Process.Pid)
				retain(t, "before", before)
				emit(t, map[string]any{"kind": "crash_cut", "cut": cut, "ack": ack, "adaptive_template": c.ID(), "entry_template": d.Entry.ID(), "padding_items": paddingItems, "operations": itemOperations(before, c.ID()), "charges": before.Roots[0].Charges, "runtime_verdict": "retained source verified", "application_verdict": "cut verified"})
				p.wait(t, true)
				stopped := inspect(t, path)
				if !reflect.DeepEqual(before, stopped) {
					t.Fatal("paused pure store moved after kill")
				}
				fresh := start(t, path, c.ID(), "", "", false, "")
				if fresh.cmd.Process.Pid == p.cmd.Process.Pid {
					t.Fatal("PID reused")
				}
				fresh.wait(t, false)
				after := inspect(t, path)
				must(t, preserve(before, after))
				must(t, applicationOracle(after, c.ID(), frozen(t, false), nested, 0))
				retain(t, "after", after)
				emit(t, map[string]any{"kind": "recovered", "cut": cut, "old_pid": p.cmd.Process.Pid, "new_pid": fresh.cmd.Process.Pid, "operations": itemOperations(after, c.ID()), "charges": after.Roots[0].Charges, "runtime_verdict": "Run completed", "application_verdict": "frozen primary accepted", "resubmitted": false})
			})
		}
	}
}
func TestAdaptiveRootDenial(t *testing.T) {
	c := component(t, false)
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			e := int64(1)
			if nested {
				e = 2
			}
			l := adaptive.DefaultLimits(1)
			l.MaxInvocations = e + 4
			d, path := provision(t, c, nested, 1, &l)
			create(t, d, path, true)
			p := start(t, path, c.ID(), "denial", "", false, "")
			ack := p.acknowledgment(t)
			before := inspect(t, path)
			must(t, durableAck(before, ack))
			must(t, cutMonitor(before, c.ID(), "denial", ack))
			retain(t, "before", before)
			p.wait(t, true)
			fresh := start(t, path, c.ID(), "", "", false, "")
			fresh.wait(t, false)
			after := inspect(t, path)
			must(t, preserve(before, after))
			q := projection(after, "root")
			if q.Data["work_count"].Number != 6 || q.Data["rounds"].Number != 2 || q.Data["stop_reason"].Text != "root_budget_denied" || q.Data["complete"].Flag || len(rows(q, "remaining")) != 0 || !reflect.DeepEqual(ids(q, "results"), []int64{1, 2, 3, 7, 10, 11}) {
				t.Fatal("whole frontier/denial response", q.Data)
			}
			for i, r := range rows(q, "results") {
				status := "succeeded"
				if i >= 4 {
					status = "rejected"
					if r["reason"].Text != "invocation_limit" {
						t.Fatal("denial reason")
					}
				}
				if r["status"].Text != status {
					t.Fatal("ordered status")
				}
			}
			if after.Roots[0].Charges.Invocations != e+4 || after.Roots[0].Charges.CreatedRuns != e+4 || after.Roots[0].Charges.Effects != 0 || after.Roots[0].Outcome == nil || after.Roots[0].Outcome.Status != "failed" {
				t.Fatal("denial charges/root", after.Roots)
			}
			must(t, origins(after))
			retain(t, "after", after)
			emit(t, map[string]any{"kind": "root_denial", "ack": ack, "request": primary(), "persisted_limits": l, "before_charges": before.Roots[0].Charges, "after_charges": after.Roots[0].Charges, "accepted_work": 6, "runtime_accepted_leaves": 4, "ordered_denied_work": []int64{10, 11}, "runtime_verdict": "diagnostic failure", "application_verdict": "literal additive next-frontier denial accepted"})
		})
	}
}
func pendingMonitor(s runtime.Inspection, template string) error {
	_, subject := adaptiveMember(s, template)
	q := projection(s, subject)
	ops := itemOperations(s, template)
	if len(ops) != 3 || len(rows(q, "decision")) != 0 || q.Data["rounds"].Number != 0 || q.Data["work_count"].Number != 3 {
		return fmt.Errorf("unknown bypassed barrier")
	}
	beta, ok := outcome(s, ops[2])
	if !ok || beta.Status != "failed" || beta.ResponsePresent {
		return fmt.Errorf("known failed sibling absent")
	}
	if _, ok := outcome(s, ops[3]); ok {
		return fmt.Errorf("unknown gamma fabricated known outcome")
	}
	if s.Roots[0].Outcome != nil {
		return fmt.Errorf("public root outcome despite required unknown")
	}
	if len(rows(projection(s, "root"), "results")) != 0 {
		return fmt.Errorf("public application response despite unknown")
	}
	return nil
}
func TestAdaptiveUnknownSibling(t *testing.T) {
	c := component(t, true)
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			provider, server := newProvider(t, 2, false, true)
			d, path := provision(t, c, nested, 2, nil)
			create(t, d, path, true)
			p := start(t, path, c.ID(), "unknown", server.URL, false, "")
			select {
			case <-provider.entered:
			case <-time.After(raceDuration(5 * time.Minute)):
				t.Fatal("watchdog: missing held request")
			}
			ack := p.acknowledgment(t)
			before := inspect(t, path)
			must(t, durableAck(before, ack))
			must(t, pendingMonitor(before, c.ID()))
			ledger, active, peak := provider.snapshot()
			if active != 1 || peak > 2 || contacts(ledger) != 3 {
				t.Fatal("controlled contacts", ledger, active, peak)
			}
			must(t, contactMonitor(before, ledger))
			retain(t, "before", before)
			p.wait(t, true)
			select {
			case <-provider.ended:
			case <-time.After(raceDuration(time.Minute)):
				t.Fatal("watchdog: transport did not close")
			}
			fresh := start(t, path, c.ID(), "", server.URL, false, "")
			fresh.wait(t, false)
			after := inspect(t, path)
			must(t, preserve(before, after))
			must(t, pendingMonitor(after, c.ID()))
			last, _, peak := provider.snapshot()
			must(t, contactMonitor(after, last))
			if contacts(last) != 3 || peak > 2 || len(after.Roots[0].UnknownExternal) != 1 || after.Roots[0].Charges.Effects != 3 {
				t.Fatal("unknown reservation/contact changed", after.Roots, last)
			}
			e := int64(4)
			if nested {
				e++
			}
			if after.Roots[0].Charges.Invocations != e || after.Roots[0].Charges.CreatedRuns != e {
				t.Fatal("unknown charges")
			}
			mutant := after
			mutant.Roots = append(after.Roots[:0:0], after.Roots...)
			mutant.Roots[0].Outcome = &runtime.InvocationOutcome{Status: "succeeded"}
			if pendingMonitor(mutant, c.ID()) == nil {
				t.Fatal("premature-success control survived")
			}
			retain(t, "after", after)
			emit(t, map[string]any{"kind": "unknown_failed_sibling", "ack": ack, "original_unknown": after.Roots[0].UnknownExternal, "original_operations": itemOperations(after, c.ID()), "provider_contacts": last, "additional_contacts": 0, "runtime_verdict": "pending spent unknown", "application_verdict": "barrier preserved; no eventual result claimed"})
		})
	}
}
func TestAdaptiveCapacityAndOutOfOrder(t *testing.T) {
	c := component(t, true)
	for _, nested := range []bool{false, true} {
		for _, capacity := range []int{1, 2} {
			for _, held := range []int64{0, 2} {
				if capacity == 1 && held == 2 {
					continue
				}
				t.Run(fmt.Sprintf("nested=%v/capacity=%d/held=%d", nested, capacity, held), func(t *testing.T) {
					provider, server := newProvider(t, held, false, false)
					d, path := provision(t, c, nested, capacity, nil)
					addProgressWitness(t, &d.Provision)
					create(t, d, path, true)
					p := start(t, path, c.ID(), "capacity", server.URL, false, "")
					select {
					case <-provider.entered:
					case <-time.After(raceDuration(5 * time.Minute)):
						t.Fatal("watchdog: missing held request")
					}
					// HTTP entry precedes admission of the witness input. Its retained
					// initial state rules out progress from preexisting frontier admission.
					// Every recorded contact follows its committed allowed attempt. Capture
					// the ledger first so concurrent later contacts cannot outrun inspection.
					contactedLedger, active, contactPeak := provider.snapshot()
					contacted := inspect(t, path)
					retain(t, "contact_before_progress", contacted)
					retain(t, "contact_ledger_before_progress", map[string]any{"ledger": contactedLedger, "active": active, "peak": contactPeak})
					initial := projection(contacted, "progress")
					if initial.Status != "open" || initial.Data["marker"].Number != 0 {
						t.Fatal("witness moved before trigger")
					}
					for _, o := range contacted.Tables["observations"] {
						if o["subject"] == "progress" {
							t.Fatal("witness input preceded contact trigger")
						}
					}
					must(t, contactMonitor(contacted, contactedLedger))
					if active < 1 || active > capacity {
						t.Fatal("held request absent before progress input", contactedLedger)
					}
					_, err := p.progress.Write([]byte{'p'})
					must(t, err)
					ack := p.acknowledgment(t)
					before := inspect(t, path)
					retain(t, "held", before)
					must(t, durableAck(before, ack))
					must(t, cutMonitor(before, c.ID(), "capacity", ack))
					ledger, active, peak := provider.snapshot()
					retain(t, "held_provider", map[string]any{"ledger": ledger, "active": active, "peak": peak})
					must(t, contactMonitor(before, ledger))
					if active != 1 || peak > capacity {
						t.Fatal("physical capacity", ledger, active, peak)
					}
					if capacity == 1 && contacts(ledger) != 1 {
						t.Fatal("second contact while slot held", ledger)
					}
					if capacity == 2 {
						if contacts(ledger) != 3 {
							t.Fatal("independent transport did not progress", ledger)
						}
						ops := itemOperations(before, c.ID())
						finished := 0
						for _, op := range ops {
							if _, ok := outcome(before, op); ok {
								finished++
							}
						}
						if finished != 2 {
							t.Fatal("second transport not durably completed", finished)
						}
					}
					emit(t, map[string]any{"kind": "capacity_held", "ack": ack, "capacity": capacity, "held_value": held, "peak": peak, "ledger": ledger, "logical_admitted": 3, "in_flight_is_advisory": true, "fixed_static_runs": 2, "progress_witness": projection(before, "progress"), "progress_trigger_after_http_entry": true, "pre_trigger_ledger": contactedLedger})
					provider.unblock()
					p.resume(t)
					p.wait(t, false)
					after := inspect(t, path)
					retain(t, "after", after)
					last, _, peak := provider.snapshot()
					retain(t, "after_provider", map[string]any{"ledger": last, "peak": peak})
					must(t, contactMonitor(after, last))
					must(t, applicationOracle(after, c.ID(), frozen(t, false), nested, 7))
					if peak > capacity || contacts(last) != 7 {
						t.Fatal("physical contact conservation")
					}
					emit(t, map[string]any{"kind": "out_of_order_completed", "capacity": capacity, "held_value": held, "peak": peak, "ledger": last, "runtime_verdict": "succeeded", "application_verdict": "same frozen ordered primary accepted"})
				})
			}
		}
	}
}
