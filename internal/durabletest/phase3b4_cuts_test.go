//go:build integration

package durabletest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/examples/delegation"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestPhase3B4Owner(t *testing.T) {
	if os.Getenv("ALAN_PHASE3B4_CHILD") != "1" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	path := os.Getenv("ALAN_PHASE2_DB")
	out := os.Getenv("ALAN_PHASE3B4_OUTPUT")
	hooks, err := boundary.FromEnvironment(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		baseline3b3Write(t, out, "open.json", map[string]string{"Error": err.Error()})
		t.Fatal(err)
	}
	defer s.Close()
	states := map[string]runtime.RunState{}
	for _, id := range []string{"a", "b"} {
		states[id], err = s.Snapshot(ctx, runtime.RunID{Subject: id, Generation: 1})
		if err != nil {
			t.Fatal(err)
		}
	}
	baseline3b3Write(t, out, "open.json", map[string]any{"PID": os.Getpid(), "Namespace": s.Namespace(), "Runs": states, "Error": "<nil>"})
	backup := filepath.Join(t.TempDir(), "post-open.sqlite")
	reciprocalBackup(t, path, backup)
	bindingCapture(t, backup, filepath.Join(out, "post-open"))
	f, err := os.OpenFile(filepath.Join(out, "boundaries.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var mu sync.Mutex
	revoked := false
	ctx = boundary.WithObserver(ctx, func(ctx context.Context, e boundary.Event) error {
		if os.Getenv("ALAN_PHASE3B4_REVOKE") == "1" && !revoked && e.Boundary == "dispatch.consider" && e.Phase == "pre-admission" && e.Intent == `["a",1,4,0]` {
			revoked = true
			raw, err := os.ReadFile(os.Getenv("ALAN_PHASE3B4_MANIFEST"))
			if err != nil {
				t.Fatal(err)
			}
			var m delegation.RequestBindingManifest
			if err = json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			reciprocalRevoke(t, s, m.Reciprocal, out)
		}

		mu.Lock()
		err := json.NewEncoder(f).Encode(e)
		if err == nil {
			err = f.Sync()
		}
		mu.Unlock()
		if err != nil {
			return err
		}
		if os.Getenv("ALAN_BARRIER") == "" {
			return nil
		}
		return boundary.Reached(boundary.Inherit(ctx, hooks), e)
	})
	var config []reciprocalAdapter
	if raw := os.Getenv("ALAN_PHASE3B4_ADAPTERS"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			t.Fatal(err)
		}
	}
	report, runErr := s.Run(ctx, runtime.RunOptions{Adapters: reciprocalAdapters(config), MaxSteps: 256, MaxDispatch: 256, MaxInFlight: 2, ExternalTimeout: 5 * time.Second})
	baseline3b3Write(t, out, "run.json", map[string]any{"Report": report, "Error": fmt.Sprint(runErr)})
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func bindingOwner(t *testing.T, db, dir string, cut *barrier, manifest string, revoke bool) *child {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return phaseTestProcess(t, db, "", "", cut, "TestPhase3B4Owner", func(cmd *exec.Cmd) {
		cmd.Env = append(cmd.Env, "ALAN_PHASE3B4_CHILD=1", "ALAN_PHASE3B4_OUTPUT="+dir, "ALAN_PHASE3B4_MANIFEST="+manifest)
		if revoke {
			cmd.Env = append(cmd.Env, "ALAN_PHASE3B4_REVOKE=1")
		}
	})
}

func bindingReadCapture(t *testing.T, path string) *runtime.ExecutionSnapshot {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "read.sqlite")
	reciprocalCopyDB(t, filepath.Join(path, "runtime.sqlite"), probe)
	s, err := runtime.InspectExecution(t.Context(), probe, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func bindingRenewal(t *testing.T, before, after *runtime.ExecutionSnapshot) {
	t.Helper()
	if before.Namespace != after.Namespace || !reflect.DeepEqual(before.Operations, after.Operations) || !reflect.DeepEqual(before.Observations, after.Observations) || !reflect.DeepEqual(before.Effects, after.Effects) || !reflect.DeepEqual(before.Composition, after.Composition) || !reflect.DeepEqual(before.Deployment, after.Deployment) || !reflect.DeepEqual(before.LifecycleDecisions, after.LifecycleDecisions) || !reflect.DeepEqual(before.Pending, after.Pending) {
		t.Fatal("renewal changed retained logical evidence")
	}
	for _, old := range before.Runs {
		now, ok := after.Run(old.ID)
		wantEpoch := old.Epoch
		if old.Projection.Status == "open" {
			wantEpoch++
		}
		if !ok || now.Epoch != wantEpoch || now.Sequence != old.Sequence || !reflect.DeepEqual(now.Projection, old.Projection) || now.ProgramID != old.ProgramID {
			t.Fatal("renewal identity", old, now)
		}
	}
}
func TestPhase3B4CampaignCuts(t *testing.T) {
	for _, name := range []string{"C01", "C02", "C03", "C04", "C05", "C06", "C07", "C08"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", nil)
			n := 0
			if name == "C03" || name == "C07" {
				n = 3
			}
			p, m := bindingFixture(t, ua, ub, n, nil, false)
			if name == "C08" {
				reciprocalRevokeProgram(t, &p, &m.Reciprocal)
				bindingVersion(t, &p, runtime.RequestAuthorityBindingVersion)
			}
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			initial := bindingCapture(t, db, filepath.Join(dir, "initial"))
			c := bindingCase{Name: name, Schedule: "acknowledged-SIGKILL-controlled-boundary", Classification: "recovered-complete", Manifest: m, Captures: []bindingCaptureRef{{Ordinal: 1, Label: "initial", Directory: "initial"}}}
			add := func(label, path string, pid int, killed bool) {
				prev := c.Captures[len(c.Captures)-1].Directory
				c.Captures = append(c.Captures, bindingCaptureRef{Ordinal: len(c.Captures) + 1, Predecessor: prev, Label: label, Directory: path, OwnerPID: pid, AcknowledgedSIGKILL: killed})
			}
			filter := &barrier{Subject: "a", Boundary: "execution.committed", Phase: "post-commit"}
			switch name {
			case "C01":
				filter.Boundary = "execution.staged"
				filter.Phase = "pre-commit"
				filter.Sequence = 2
			case "C02", "C07":
				filter.Sequence = 2
			case "C03":
				filter.Sequence = 6
			case "C04":
				filter.Sequence = 4
			case "C05", "C08":
				filter.Boundary = "dispatch.consider"
				filter.Phase = "pre-admission"
				filter.Intent = m.Reciprocal.BAService
			case "C06":
				filter.Boundary = "external.settled"
				filter.Phase = "post-record"
				filter.Intent = m.Reciprocal.BAService
			}
			previous := initial
			losses := 1
			if name == "C07" {
				losses = 2
			}
			var stopped *runtime.ExecutionSnapshot
			var priorPID int
			for loss := 1; loss <= losses; loss++ {
				ownerName := fmt.Sprintf("owner-%02d", loss)
				owner := bindingOwner(t, db, filepath.Join(dir, ownerName), filter, filepath.Join(dir, "manifest.json"), name == "C08")
				ack := reciprocalAwait(t, owner)
				owner.kill(t)
				baseline3b3Write(t, dir, fmt.Sprintf("loss-%02d.json", loss), map[string]any{"Acknowledgment": ack, "PID": owner.cmd.Process.Pid, "Wait": owner.cmd.ProcessState.String(), "Command": owner.cmd.Args})
				opened := bindingReadCapture(t, filepath.Join(dir, ownerName, "post-open"))
				bindingRenewal(t, previous, opened)
				add("post-open", ownerName+"/post-open", owner.cmd.Process.Pid, false)
				stopName := fmt.Sprintf("stopped-%02d", loss)
				stopped = bindingCapture(t, db, filepath.Join(dir, stopName))
				add("acknowledged-SIGKILL", stopName, owner.cmd.Process.Pid, true)
				q := bindingSelected(t, stopped, m.Reciprocal.A, m.Reciprocal.BARequest)
				if name == "C01" {
					if q.SelectedAt != nil {
						t.Fatal("admitted request already selected", q)
					}
				} else if q.SelectedAt == nil || q.SelectedAt.Sequence != 2 {
					t.Fatal("selected original lost", q)
				}
				a, _ := stopped.Run(m.Reciprocal.A)
				e, _ := stopped.Effect(m.Reciprocal.BAService)
				switch name {
				case "C02":
					if a.Sequence != 2 {
						t.Fatal(a)
					}
				case "C03":
					if a.Sequence != 6 || e != nil {
						t.Fatal("future service fabricated", a, e)
					}
				case "C04":
					if e == nil || e.ExternalAttempt != nil {
						t.Fatal(e)
					}
				case "C05", "C08":
					if e == nil || e.ExternalAttempt == nil || !e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 0 {
						t.Fatal(e)
					}
				case "C06":
					if e == nil || e.ExternalReceipt == nil || e.Settlement == nil {
						t.Fatal(e)
					}
					o, _ := stopped.Observation(e.Settlement.ObservationID)
					if o == nil || o.SelectedAt != nil {
						t.Fatal(o)
					}
				case "C07":
					want := int64(2)
					if loss == 2 {
						want = 6
					}
					if a.Sequence != want || e != nil {
						t.Fatal(a, e)
					}
				}
				baseline3b3Write(t, dir, fmt.Sprintf("endpoints-loss-%02d.json", loss), map[string]any{"a": phaseSnapshot(t, pa), "b": phaseSnapshot(t, pb)})
				previous = stopped
				priorPID = owner.cmd.Process.Pid
				filter = &barrier{Subject: "a", Boundary: "execution.committed", Phase: "post-commit", Sequence: 6}
			}
			finalOwner := bindingOwner(t, db, filepath.Join(dir, "final-owner"), nil, filepath.Join(dir, "manifest.json"), false)
			phaseFinish(t, finalOwner)
			if priorPID == finalOwner.cmd.Process.Pid {
				t.Fatal("not a fresh process")
			}
			opened := bindingReadCapture(t, filepath.Join(dir, "final-owner", "post-open"))
			bindingRenewal(t, stopped, opened)
			add("post-open", "final-owner/post-open", finalOwner.cmd.Process.Pid, false)
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			final := bindingCapture(t, db, filepath.Join(dir, "final"))
			add("final", "final", finalOwner.cmd.Process.Pid, false)
			if name == "C08" {
				c.Classification = "historical-allow-current-gate-refused"
				e, _ := final.Effect(m.Reciprocal.BAService)
				if e == nil || !e.ExternalAttempt.Allowed || len(e.AuthorityAdmissions) != 1 || e.AuthorityAdmissions[0].Allowed || e.ExternalReceipt != nil || endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != 0 {
					t.Fatal(e, endpoints)
				}
			} else {
				bindingComplete(t, final, m.Reciprocal, endpoints)
				var finalRun struct {
					Report runtime.RunReport
					Error  string
				}
				raw, err := os.ReadFile(filepath.Join(dir, "final-owner", "run.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(raw, &finalRun); err != nil {
					t.Fatal(err)
				}
				if finalRun.Error != "<nil>" || !finalRun.Report.Quiescent || len(finalRun.Report.OpenRuns) != 0 || len(finalRun.Report.PendingDeliveries) != 0 || len(finalRun.Report.UnresolvedIntents) != 0 {
					t.Fatal("recovery not globally quiescent", finalRun)
				}

				if name == "C06" {
					old, _ := stopped.Effect(m.Reciprocal.BAService)
					now, _ := final.Effect(m.Reciprocal.BAService)
					if !reflect.DeepEqual(old.ExternalAttempt, now.ExternalAttempt) || !reflect.DeepEqual(old.AuthorityAdmissions, now.AuthorityAdmissions) || endpoints["b"].Contacts != 1 {
						t.Fatal("known result recontacted", old, now, endpoints)
					}
				}
			}
			bindingIndex(t, dir, c, p)
		})
	}
}
