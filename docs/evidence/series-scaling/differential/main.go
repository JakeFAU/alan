// Public differential qualification, deliberately separate from timing samples.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type contact struct {
	Ordinal   int               `json:"ordinal"`
	Prepared  effects.Prepared  `json:"prepared"`
	Candidate effects.Candidate `json:"candidate"`
}
type cut struct {
	Label    string                      `json:"label"`
	Position int                         `json:"position"`
	Snapshot *aruntime.ExecutionSnapshot `json:"snapshot"`
	History  []aruntime.ExecutionStep    `json:"history"`
	Replay   aruntime.ReplayReport       `json:"replay"`
	View     author.SeriesView           `json:"series_view"`
	Contacts int                         `json:"contacts"`
}
type report struct {
	Version     string    `json:"version"`
	Mode        string    `json:"mode"`
	Bound       int       `json:"bound"`
	Scenario    string    `json:"scenario"`
	ReopenEvery int       `json:"reopen_every"`
	SourceLabel string    `json:"source_label"`
	Store       string    `json:"store"`
	Cuts        []cut     `json:"cuts"`
	Contacts    []contact `json:"contacts"`
	OK          bool      `json:"ok"`
	Error       string    `json:"error,omitempty"`
}
type meter struct {
	mu       sync.Mutex
	contacts []contact
	refuseID string
}

func (m *meter) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "measurement/v1", Endpoint: "deterministic-local-no-http", Wire: r.Payload["station"].Text, MaxResponseBytes: 256}, nil
}
func (m *meter) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := effects.Candidate{ID: p.Request.ID, Key: p.Request.Key, Disposition: "validated", Payload: map[string]schema.Value{"sample": schema.Text("present")}, ResponseID: "local-response:" + p.Request.ID}
	if p.Request.ID == m.refuseID {
		c.Disposition = "refusal"
		c.Payload = nil
	}
	m.contacts = append(m.contacts, contact{Ordinal: len(m.contacts) + 1, Prepared: p, Candidate: c})
	return c
}
func definition() author.Definition {
	return author.Definition{Name: "measurement", Inputs: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 64}}}}, Steps: []author.Step{
		author.Request("operation", "read", map[string]author.Expr{"station": author.Var("station")}, nil), author.Await("read", "operation", map[string]string{"sample": "sample"}), author.Success()}}
}
func declaration(n int) (author.Series, error) {
	return author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: n, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
}
func (r *report) capture(ctx context.Context, decl author.Series, m *meter, label string, position int) error {
	fmt.Fprintf(os.Stderr, "source=%s scenario=%s cut=%s position=%d\n", r.SourceLabel, r.Scenario, label, position)
	s, err := aruntime.InspectExecution(ctx, r.Store, aruntime.DefaultLimits())
	if err != nil {
		return err
	}
	var v author.SeriesView
	if r.Mode != "generic-reader" {
		v, err = decl.Inspect(s)
		if err != nil {
			return err
		}
	}
	replay, err := aruntime.Replay(ctx, r.Store, aruntime.DefaultLimits())
	if err != nil {
		return err
	}
	var history []aruntime.ExecutionStep
	for offset := 0; ; {
		page, e := s.History(offset, aruntime.MaxHistoryPage)
		if e != nil {
			return e
		}
		history = append(history, page.Steps...)
		if !page.More {
			break
		}
		offset = page.Next
	}
	if replay.Namespace != s.Namespace || replay.HistoryCount != len(history) || (r.Mode != "generic-reader" && len(replay.Runs) != r.Bound) {
		return fmt.Errorf("cut %s replay differs from original retained computation", label)
	}
	m.mu.Lock()
	contacts := len(m.contacts)
	m.mu.Unlock()
	r.Cuts = append(r.Cuts, cut{Label: label, Position: position, Snapshot: s, History: history, Replay: replay, View: v, Contacts: contacts})
	return nil
}
func (r *report) execute(ctx context.Context) (err error) {
	if r.Mode == "generic-reader" {
		// Generic compatibility/renewal only: no author declaration, Run,
		// adapter registry, approval, or source-specific configuration.
		m := &meter{}
		if err = r.capture(ctx, author.Series{}, m, "generic-before-open", 0); err != nil {
			return err
		}
		owner, e := aruntime.Open(ctx, r.Store, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		if err = owner.Close(); err != nil {
			return err
		}
		if err = r.capture(ctx, author.Series{}, m, "generic-after-open-close", 0); err != nil {
			return err
		}
		r.OK = true
		return nil
	}
	decl, err := declaration(r.Bound)
	if err != nil {
		return err
	}
	m := &meter{}
	if r.Scenario == "refusal3" {
		m.refuseID = `["series-3",1,2,0]`
	}
	defer func() { m.mu.Lock(); r.Contacts = append([]contact(nil), m.contacts...); m.mu.Unlock() }()
	if r.Mode == "create-seed" {
		if _, e := os.Stat(r.Store); !errors.Is(e, os.ErrNotExist) {
			return fmt.Errorf("seed path must be absent")
		}
		p, e := decl.Provision()
		if e != nil {
			return e
		}
		s, e := aruntime.Create(ctx, r.Store, p, aruntime.DefaultLimits())
		if e != nil {
			return e
		}
		if e = s.Close(); e != nil {
			return e
		}
		if e = r.capture(ctx, decl, m, "seed-closed", 0); e != nil {
			return e
		}
		last := r.Cuts[len(r.Cuts)-1]
		if len(last.Snapshot.Effects) != 0 || len(last.Snapshot.Continuations) != 0 || len(last.History) != 0 || last.View.Spent != 0 {
			return fmt.Errorf("seed contains executed work")
		}
		if info, e := os.Stat(r.Store + "-wal"); e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		} else if e == nil && info.Size() != 0 {
			return fmt.Errorf("seed WAL must be empty before cloning")
		}
		r.OK = true
		return nil
	}
	if _, e := os.Stat(r.Store); e != nil {
		return e
	}
	if err = r.capture(ctx, decl, m, "before-open", 0); err != nil {
		return err
	}
	initial := r.Cuts[len(r.Cuts)-1]
	if len(initial.Snapshot.Effects) != 0 || len(initial.Snapshot.Continuations) != 0 || len(initial.History) != 0 || initial.View.Spent != 0 {
		return fmt.Errorf("execute-existing requires an untouched provisioned seed clone")
	}
	store, err := aruntime.Open(ctx, r.Store, aruntime.DefaultLimits())
	if err != nil {
		return err
	}
	defer func() {
		if store != nil {
			err = errors.Join(err, store.Close())
		}
	}()
	if err = r.capture(ctx, decl, m, "after-open", 0); err != nil {
		return err
	}
	opts := aruntime.RunOptions{MaxSteps: 16*r.Bound + 32, MaxDispatch: r.Bound + 8, MaxInFlight: 1, Adapters: map[string]effects.Adapter{"meter": m}}
	for i := 1; i <= r.Bound; i++ {
		if i > 1 {
			s, e := aruntime.InspectExecution(ctx, r.Store, aruntime.DefaultLimits())
			if e != nil {
				return e
			}
			basis, e := s.ContinuationBasis(aruntime.RunID{Subject: fmt.Sprintf("series-%d", i), Generation: 1})
			if e != nil {
				return e
			}
			if _, e = store.ApproveContinuation(ctx, basis, "operator"); e != nil {
				return e
			}
			if err = r.capture(ctx, decl, m, "after-approval", i); err != nil {
				return err
			}
		}
		if _, err = store.Run(ctx, opts); err != nil {
			return err
		}
		if err = r.capture(ctx, decl, m, "after-run", i); err != nil {
			return err
		}
		last := r.Cuts[len(r.Cuts)-1]
		if last.Contacts != i || last.View.Spent != i || last.View.Selected != i {
			return fmt.Errorf("position %d contact/charge/selection mismatch", i)
		}
		if r.Scenario == "refusal3" && i >= 3 {
			third := last.View.Positions[2]
			if third.State != author.SeriesFailed || third.Effect == nil || third.Effect.AdapterResult == nil || third.Effect.AdapterResult.Disposition != "refusal" || third.Effect.Consequence != aruntime.ConsequenceUnresolved || third.Observation == nil || third.Observation.SelectedAt == nil || third.Root.Charges.Effects != 1 || last.View.Completed != i-1 || last.View.Failed != 1 {
				return fmt.Errorf("paid third refusal lost original result/selection attribution")
			}
			if i >= 4 {
				fourth := last.View.Positions[3]
				if fourth.Approval == nil || fourth.Approval.Approval.PredecessorID != third.Effect.OperationID || fourth.Approval.Approval.PreparedDigest != third.Effect.AdapterAttempt.Prepared.Digest || fourth.Approval.Approval.ResultDigest != third.Effect.AdapterResult.Digest || !fourth.Approval.Approval.Admitted || fourth.Effect == nil || fourth.Effect.OperationID == third.Effect.OperationID {
					return fmt.Errorf("fourth did not follow its independent approval of paid third evidence")
				}
			}
		}
		if r.ReopenEvery > 0 && i < r.Bound && i%r.ReopenEvery == 0 {
			if err = store.Close(); err != nil {
				return err
			}
			store = nil
			store, err = aruntime.Open(ctx, r.Store, aruntime.DefaultLimits())
			if err != nil {
				return err
			}
			if _, err = store.Run(ctx, opts); err != nil {
				return err
			}
			if err = r.capture(ctx, decl, m, "after-interval-reopen", i); err != nil {
				return err
			}
			if r.Cuts[len(r.Cuts)-1].Contacts != i {
				return fmt.Errorf("reopen duplicated contact")
			}
		}
	}
	if err = store.Close(); err != nil {
		return err
	}
	store = nil
	if err = r.capture(ctx, decl, m, "final-closed", r.Bound); err != nil {
		return err
	}
	final := r.Cuts[len(r.Cuts)-1]
	wantUnresolved := 0
	wantFailed := 0
	wantCompleted := r.Bound
	if r.Scenario == "refusal3" {
		wantCompleted--
		wantFailed = 1
		wantUnresolved = 1
	}
	if final.View.Completed != wantCompleted || final.View.Failed != wantFailed || final.View.Selected != r.Bound || final.View.Spent != r.Bound || final.View.Unresolved != wantUnresolved || final.View.Pending != 0 || final.Contacts != r.Bound || len(final.Snapshot.Continuations) != r.Bound-1 {
		return fmt.Errorf("final series accounting mismatch: %+v", final.View)
	}
	r.OK = true
	return nil
}
func main() {
	mode := flag.String("mode", "", "create-seed, execute-existing, or generic-reader")
	db := flag.String("db", "", "seed/output store path (parent directory must exist)")
	bound := flag.Int("bound", 8, "series bound")
	scenario := flag.String("scenario", "success", "success or refusal3 (third adapter result is retained refusal)")
	reopen := flag.Int("reopen-every", 0, "reopen after each K selected positions before final position; 0 disables")
	label := flag.String("source-label", "unspecified", "source provenance label")
	timeout := flag.Duration("timeout", 10*time.Minute, "whole qualification deadline")
	flag.Parse()
	if (*mode != "create-seed" && *mode != "execute-existing" && *mode != "generic-reader") || *db == "" || *bound < 1 || *bound > 256 || *reopen < 0 || (*scenario != "success" && *scenario != "refusal3") || (*scenario == "refusal3" && *bound < 3) || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "require -mode create-seed|execute-existing|generic-reader -db PATH [-bound 8] [-scenario success|refusal3] [-reopen-every K]")
		os.Exit(2)
	}
	if *mode == "generic-reader" {
		*bound = 0
	}
	r := report{Version: "alan/series-differential/v1", Mode: *mode, Bound: *bound, Scenario: *scenario, ReopenEvery: *reopen, SourceLabel: *label, Store: *db}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	err := r.execute(ctx)
	cancel()
	if err != nil {
		r.Error = err.Error()
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(r); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
