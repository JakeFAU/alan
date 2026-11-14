//go:build integration

package executor

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
)

func progressAdapterOptions(execute func(context.Context, effects.Prepared) effects.Candidate) RunOptions {
	return RunOptions{
		MaxSteps: 20, MaxDispatch: 20, MaxInFlight: 1,
		Adapters: map[string]effects.Adapter{"model": callbackAdapter{execute: execute}},
	}
}

func TestRunProgressStaticAdapterParity(t *testing.T) {
	for _, disposition := range []string{"validated", "provider_refused", "unknown"} {
		t.Run(disposition, func(t *testing.T) {
			seed, path := staticAdapterStore(t, 1, true)
			if err := seed.Close(); err != nil {
				t.Fatal(err)
			}
			full, fullPath := forkProgressStore(t, path)
			progress, progressPath := forkProgressStore(t, path)
			var contacts atomic.Int64
			options := progressAdapterOptions(func(ctx context.Context, p effects.Prepared) effects.Candidate {
				contacts.Add(1)
				if disposition == "validated" {
					return callbackAdapter{}.Execute(ctx, p)
				}
				return effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: disposition}
			})

			want, err := full.Run(t.Context(), options)
			if err != nil || contacts.Load() != 1 {
				t.Fatalf("full=%+v error=%v contacts=%d", want, err, contacts.Load())
			}
			got, err := progress.RunProgress(t.Context(), options)
			if err != nil || got != reportedProgress(want) || !got.Quiescent || got.Dispatches != 1 || contacts.Load() != 2 {
				t.Fatalf("full=%+v progress=%+v error=%v contacts=%d", want, got, err, contacts.Load())
			}
			staticAssertActive(t, full)
			staticAssertActive(t, progress)
			cut := assertProgressEvidenceParity(t, fullPath, progressPath)
			if len(cut.Effects) != 1 || len(cut.Roots) != 1 {
				t.Fatalf("expected one original effect and one root: effects=%d roots=%d", len(cut.Effects), len(cut.Roots))
			}
			effect := cut.Effects[0]
			root := cut.Roots[0]
			if effect.Authorization == nil || !effect.Authorization.Allowed || effect.AdapterAttempt == nil ||
				effect.DispatchAdmission == nil || effect.Dispatch.Maximum == nil || *effect.Dispatch.Maximum != 1 ||
				root.Charges.Effects != 1 {
				t.Fatalf("missing original authority, reservation, gate or charge: effect=%+v root=%+v", effect, root)
			}

			if disposition == "unknown" {
				if got.Steps != 1 || effect.State != EffectReservedUnknown || effect.Consequence != ConsequenceUnresolved ||
					effect.AdapterResult != nil || effect.Dispatch.Minimum != 0 || root.LocalStatus != "open" ||
					len(want.UnknownAdapters) != 1 ||
					want.UnknownAdapters[0] != effect.OperationID {
					t.Fatalf("unknown result lost or resolved: progress=%+v effect=%+v root=%+v", got, effect, root)
				}
				if err := progress.Close(); err != nil {
					t.Fatal(err)
				}
				reopened, err := Open(t.Context(), progressPath, DefaultLimits())
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				again, err := reopened.RunProgress(t.Context(), options)
				if err != nil || again != (RunProgress{Quiescent: true}) || contacts.Load() != 2 {
					t.Fatalf("reopen retried an unknown attempt: progress=%+v error=%v contacts=%d", again, err, contacts.Load())
				}
				return
			}

			if got.Steps != 3 || effect.State != EffectSettled || effect.AdapterResult == nil || effect.Dispatch.Minimum != 1 ||
				effect.ResultDisposition != disposition || effect.Settlement == nil || root.Outcome == nil {
				t.Fatalf("missing settled result or root outcome: progress=%+v effect=%+v root=%+v", got, effect, root)
			}
			observation, ok := cut.Observation(effect.Settlement.ObservationID)
			if !ok || observation.SelectedAt == nil || observation.SelectedAt.Run != effect.Source || observation.SelectedAt.Sequence != 2 {
				t.Fatalf("original result selection=%+v", observation)
			}
		})
	}
}

func TestRunProgressCancellationAccountsReceivedResult(t *testing.T) {
	for _, stopSettlement := range []bool{false, true} {
		name := "settled"
		if stopSettlement {
			name = "retained_before_settlement"
		}
		t.Run(name, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			receivedErr := errors.New("stop after dequeue")
			settlementErr := errors.New("stop before settlement")
			ctx = boundary.WithObserver(ctx, func(accounting context.Context, event boundary.Event) error {
				switch event.Boundary {
				case "runner.received":
					cancel()
					return receivedErr
				case "adapter.result-recorded":
					deadline, bounded := accounting.Deadline()
					if accounting.Err() != nil || !bounded || time.Until(deadline) > runnerCleanupTimeout {
						t.Error("accounting did not use a bounded context detached from cancellation")
					}
					if stopSettlement {
						return settlementErr
					}
				}
				return nil
			})
			var contacts atomic.Int64
			options := progressAdapterOptions(func(ctx context.Context, p effects.Prepared) effects.Candidate {
				contacts.Add(1)
				return callbackAdapter{}.Execute(ctx, p)
			})
			got, err := s.RunProgress(ctx, options)
			if !errors.Is(err, context.Canceled) || !errors.Is(err, receivedErr) ||
				errors.Is(err, settlementErr) != stopSettlement || got != (RunProgress{Dispatches: 1}) {
				t.Fatalf("canceled accounting progress=%+v error=%v", got, err)
			}
			cut, err := InspectExecution(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			effect, ok := cut.Effect(id)
			stage := EffectSettled
			recoveryDispatches := 0
			if stopSettlement {
				stage = EffectResultUnsettled
				recoveryDispatches = 1
			}
			if !ok || effect.AdapterResult == nil || effect.State != stage || effect.Consequence != ConsequenceReceiptRetained ||
				current(t, s, "agent").Projection.Status != "open" {
				t.Fatalf("owned evidence lost or source advanced: %+v", effect)
			}
			got, err = s.RunProgress(t.Context(), options)
			if err != nil || got != (RunProgress{Steps: 2, Quiescent: true, Dispatches: recoveryDispatches}) ||
				contacts.Load() != 1 || current(t, s, "agent").Projection.Status != "succeeded" {
				t.Fatalf("recovery progress=%+v error=%v contacts=%d", got, err, contacts.Load())
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunProgressRejectsReachedMalformedAdapterEvidence(t *testing.T) {
	for _, table := range []string{"adapter_attempts", "adapter_results"} {
		t.Run(table, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			var contacts atomic.Int64
			options := progressAdapterOptions(func(ctx context.Context, p effects.Prepared) effects.Candidate {
				contacts.Add(1)
				return callbackAdapter{}.Execute(ctx, p)
			})
			stop := errors.New("retain result without settlement")
			ctx := boundary.WithObserver(t.Context(), func(_ context.Context, event boundary.Event) error {
				if event.Boundary == "adapter.result-recorded" {
					return stop
				}
				return nil
			})
			if _, err := s.RunProgress(ctx, options); !errors.Is(err, stop) || contacts.Load() != 1 {
				t.Fatalf("fixture failed to retain a genuine result: error=%v contacts=%d", err, contacts.Load())
			}
			result, err := s.conn.ExecContext(t.Context(), "UPDATE "+table+" SET descriptor=? WHERE intent_id=?", []byte("{"), id)
			if err != nil {
				t.Fatal(err)
			}
			if rows, err := result.RowsAffected(); err != nil || rows != 1 {
				t.Fatalf("malformed evidence rows=%d error=%v", rows, err)
			}

			got, err := s.RunProgress(t.Context(), options)
			wantError := "adapter result encoding"
			if table == "adapter_attempts" {
				wantError = "adapter attempt encoding"
			}
			if !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), wantError) ||
				got != (RunProgress{}) || contacts.Load() != 1 || count(t, s, "settlements") != 0 {
				t.Fatalf("reached corrupt evidence progress=%+v error=%v contacts=%d", got, err, contacts.Load())
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("Replay accepted malformed adapter evidence")
			}
		})
	}
}

func TestRunProgressOverlapAndClose(t *testing.T) {
	for _, primary := range []string{"full", "progress"} {
		t.Run(primary, func(t *testing.T) {
			s, path, id := committedAdapter(t)
			ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
			defer stop()
			entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var contacts atomic.Int64
			options := progressAdapterOptions(func(call context.Context, p effects.Prepared) effects.Candidate {
				if contacts.Add(1) != 1 {
					t.Error("overlap caused another adapter contact")
					return effects.Candidate{Disposition: "unknown"}
				}
				close(entered)
				<-call.Done()
				close(canceled)
				select {
				case <-release:
				case <-ctx.Done():
				}
				return callbackAdapter{}.Execute(call, p)
			})
			options.ExternalTimeout = time.Minute
			waiting := make(chan struct{})
			runCtx := boundary.WithObserver(ctx, func(runnerCtx context.Context, event boundary.Event) error {
				if event.Boundary != "runner.waiting" {
					return nil
				}
				// Force the canceled runner to drain this completion in its
				// deferred cleanup while the adapter is still in flight.
				close(waiting)
				<-runnerCtx.Done()
				return runnerCtx.Err()
			})
			run := map[string]func(context.Context, RunOptions) (RunProgress, error){
				"progress": s.RunProgress,
				"full": func(ctx context.Context, options RunOptions) (RunProgress, error) {
					report, err := s.Run(ctx, options)
					return reportedProgress(report), err
				},
			}
			type outcome struct {
				progress RunProgress
				err      error
			}
			finished := make(chan outcome, 1)
			go func() {
				result, err := run[primary](runCtx, options)
				finished <- outcome{result, err}
			}()
			awaitSignal(t, ctx, entered)
			awaitSignal(t, ctx, waiting)
			for _, method := range []string{"full", "progress"} {
				if got, err := run[method](ctx, options); !errors.Is(err, ErrRunActive) || got != (RunProgress{}) {
					t.Fatalf("active %s admitted %s: progress=%+v error=%v", primary, method, got, err)
				}
			}

			closed := make(chan error, 1)
			go func() { closed <- s.Close() }()
			awaitSignal(t, ctx, canceled)
			if owner, err := Open(ctx, path, DefaultLimits()); !errors.Is(err, ErrLocked) {
				if owner != nil {
					owner.Close()
				}
				t.Fatalf("Close released ownership while adapter was held: %v", err)
			}
			select {
			case err := <-closed:
				t.Fatalf("Close returned before adapter release: %v", err)
			default:
			}
			if got, err := s.RunProgress(ctx, options); !errors.Is(err, sql.ErrConnDone) || got != (RunProgress{}) {
				t.Fatalf("closing owner admitted progress: progress=%+v error=%v", got, err)
			}
			close(release)

			select {
			case result := <-finished:
				if !errors.Is(result.err, context.Canceled) || result.progress != (RunProgress{Dispatches: 1}) {
					t.Fatalf("Close cancellation result=%+v", result)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			select {
			case err := <-closed:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			cut, err := InspectExecution(ctx, path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			effect, ok := cut.Effect(id)
			if !ok || effect.State != EffectSettled || effect.AdapterResult == nil || contacts.Load() != 1 {
				t.Fatalf("Close failed to join and account the in-flight result: effect=%+v contacts=%d", effect, contacts.Load())
			}
		})
	}
}
