//go:build integration

package durabletest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

// This sidecar records exact serialized representations against each separate
// byte limit; it is neither a total-history limit nor an allocation estimate.
func bindingSidecar(t *testing.T, dir string, raw runtime.Inspection, s *runtime.ExecutionSnapshot) {
	t.Helper()
	max := func(table, key string) int {
		n := 0
		for _, row := range raw.Tables[table] {
			if value, ok := row[key].(string); ok && len(value) > n {
				n = len(value)
			}
		}
		return n
	}
	selected := 0
	for _, o := range s.Observations {
		if o.SelectedAt != nil {
			selected++
		}
	}
	baseline3b3Write(t, dir, "cost-sidecar.json", map[string]any{
		"Limits": runtime.DefaultLimits(), "MaxProgramBytes": max("programs", "bytes"), "MaxProjectionBytes": max("history", "successor"), "MaxObservationDescriptorBytes": max("observations", "descriptor"), "MaxAdmissionDescriptorBytes": max("admissions", "descriptor"), "MaxOutputBytes": max("history", "intents"),
		"CommittedReductions": len(raw.Tables["history"]), "SelectedObservations": selected, "Admissions": len(raw.Tables["admissions"]), "Charges": raw.Tables["composition_charges"], "Reservations": raw.Tables["composition_reservations"], "Pending": s.Pending, "Roots": s.Roots, "Allocations": "not measured",
	})
}
func TestPhase3B4CampaignCosts(t *testing.T) {
	for _, n := range []int{0, 1, 8, 32} {
		for observation := 1; observation <= 5; observation++ {
			t.Run(fmt.Sprintf("n%02d-observation%02d", n, observation), func(t *testing.T) {
				name := fmt.Sprintf("cost-n%02d-observation%02d", n, observation)
				dir := bindingDirectory(t, name)
				ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
				cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
				start := time.Now()
				p, m := bindingFixture(t, ua, ub, n, nil, false)
				construction := time.Since(start)
				baseline3b3Write(t, dir, "provision.json", p)
				baseline3b3Write(t, dir, "manifest.json", m)
				baseline3b3Write(t, dir, "identity-bijection.json", map[string]any{"Domain": "designated roles across fresh provisions, not physical operation identity across namespaces", "ABRequest": [2]string{`["a",1,1,0]`, m.Reciprocal.ABRequest}, "BARequest": [2]string{`["b",1,1,0]`, m.Reciprocal.BARequest}, "Grant": [2]string{`["b",1,2,0]`, m.Reciprocal.Grant}, "Ready": [2]string{`["b",1,5,0]`, m.Reciprocal.Ready}, "BAService": [2]string{`["a",1,4,0]`, m.Reciprocal.BAService}, "BAReply": [2]string{`["a",1,7,0]`, m.Reciprocal.BAReply}, "ABService": [2]string{`["b",1,9,0]`, m.Reciprocal.ABService}, "ABReply": [2]string{`["b",1,12,0]`, m.Reciprocal.ABReply}})
				db := filepath.Join(t.TempDir(), "runtime.sqlite")
				s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				// Prepare genuine selection, grant, padding and exact committed service
				// outside the measured first-decision and gate intervals.
				_, q := bindingInitial(t, s, m.Reciprocal)
				bindingGrant(t, s, m.Reciprocal)
				reciprocalCommit(t, s, m.Reciprocal.A, q)
				ready := bindingDeliver(t, s, m.Reciprocal.Ready)
				reciprocalCommit(t, s, m.Reciprocal.A, ready)
				for i := 0; i < n+1; i++ {
					reciprocalCommit(t, s, m.Reciprocal.A, "")
				}
				var decisionStart, gateStart time.Time
				var decision, gate time.Duration
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer cancel()
				ctx = boundary.WithObserver(ctx, func(_ context.Context, e boundary.Event) error {
					if e.Intent != m.Reciprocal.BAService {
						return nil
					}
					switch e.Boundary {
					case "intent.authorized":
						decisionStart = time.Now()
					case "dispatch.consider":
						decision = time.Since(decisionStart)
						gateStart = time.Now()
					case "authority.admission":
						gate = time.Since(gateStart)
					}
					return nil
				})
				report, err := s.Run(ctx, runtime.RunOptions{MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
				s.Close()
				if err != nil {
					t.Fatal(err)
				}
				if decisionStart.IsZero() || gateStart.IsZero() || decision <= 0 || gate <= 0 {
					t.Fatal("missing measured boundaries")
				}
				// Copies and fixture setup are outside Replay/Open measurements.
				replayDB := filepath.Join(t.TempDir(), "replay.sqlite")
				openDB := filepath.Join(t.TempDir(), "open.sqlite")
				reciprocalCopyDB(t, db, replayDB)
				reciprocalCopyDB(t, db, openDB)
				start = time.Now()
				_, err = runtime.Replay(t.Context(), replayDB, runtime.DefaultLimits())
				replayDuration := time.Since(start)
				if err != nil {
					t.Fatal(err)
				}
				start = time.Now()
				opened, err := runtime.Open(t.Context(), openDB, runtime.DefaultLimits())
				openDuration := time.Since(start)
				if err != nil {
					t.Fatal(err)
				}
				opened.Close()
				endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
				snap := bindingCapture(t, db, filepath.Join(dir, "final"))
				bindingComplete(t, snap, m.Reciprocal, endpoints)
				baseline3b3Write(t, dir, "measurements.json", map[string]any{"N": n, "Observation": observation, "Nanoseconds": map[string]int64{"construction": construction.Nanoseconds(), "first_decision_window": decision.Nanoseconds(), "fresh_gate_window": gate.Nanoseconds(), "Replay": replayDuration.Nanoseconds(), "Open": openDuration.Nanoseconds()}, "TimingDefinitions": map[string]string{"construction": "public RequestBinding constructor only", "first_decision_window": "intent.authorized callback to dispatch.consider callback; includes authority/reservation and runner setup", "fresh_gate_window": "dispatch.consider callback to authority.admission callback; includes current-gate writer and boundary overhead"}, "Run": report, "Allocations": "not measured", "TotalConstantTimeClaim": false})
				queryDB, err := sql.Open("sqlite3", replayDB)
				if err != nil {
					t.Fatal(err)
				}
				defer queryDB.Close()
				rows, err := queryDB.QueryContext(t.Context(), "EXPLAIN QUERY PLAN SELECT id,descriptor,result FROM admissions WHERE observation_id=?", q)
				if err != nil {
					t.Fatal(err)
				}
				var plans []string
				for rows.Next() {
					var id, parent, unused int
					var detail string
					if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
						t.Fatal(err)
					}
					plans = append(plans, detail)
				}
				if err = rows.Close(); err != nil {
					t.Fatal(err)
				}
				if len(plans) != 1 || !strings.Contains(plans[0], "SEARCH admissions USING INDEX admissions_observation_id (observation_id=?)") {
					t.Fatal("lookup scan", plans)
				}
				baseline3b3Write(t, dir, "query-plan.json", plans)
				bindingIndex(t, dir, bindingCase{Name: name, Schedule: "controlled-public-prefix-then-run-cost-only", Classification: "measured-no-threshold", Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}}}, p)
			})
		}
	}
}
