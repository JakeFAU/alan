// Public differential qualification, deliberately separate from timing samples.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type contact struct {
	Prepared         effects.Prepared `json:"prepared"`
	RequestWire      string           `json:"request_wire"`
	RequestSHA256    string           `json:"request_sha256"`
	ResponseWire     string           `json:"response_wire"`
	ResponseSHA256   string           `json:"response_sha256"`
	ResponseComplete bool             `json:"response_complete"`
	Ordinal          int              `json:"ordinal"`
	OperationID      string           `json:"operation_id"`
	Key              string           `json:"key"`
	PreparedDigest   string           `json:"prepared_digest"`
	Capability       string           `json:"capability"`
	Effect           string           `json:"effect"`
	Protocol         string           `json:"protocol"`
	Endpoint         string           `json:"endpoint"`
	ResponseID       string           `json:"response_id"`
}
type cut struct {
	Label           string                      `json:"label"`
	Position        int                         `json:"position"`
	Snapshot        *aruntime.ExecutionSnapshot `json:"snapshot"`
	History         []aruntime.ExecutionStep    `json:"history"`
	Replay          aruntime.ReplayReport       `json:"replay"`
	View            author.SeriesView           `json:"series_view"`
	Contacts        int                         `json:"contacts"`
	AdapterContacts []contact                   `json:"adapter_contacts"`
	ServerContacts  []contact                   `json:"server_contacts"`
	RunReport       *aruntime.RunReport         `json:"run_report"`
}
type report struct {
	Version        string          `json:"version"`
	Mode           string          `json:"mode"`
	Bound          int             `json:"bound"`
	Scenario       string          `json:"scenario"`
	ReopenEvery    int             `json:"reopen_every"`
	SourceLabel    string          `json:"source_label"`
	Store          string          `json:"store"`
	Cuts           []cut           `json:"cuts"`
	Contacts       []contact       `json:"contacts"`
	ServerContacts []contact       `json:"server_contacts"`
	Transport      transportConfig `json:"transport"`
	OK             bool            `json:"ok"`
	Error          string          `json:"error,omitempty"`
}

func definition() author.Definition {
	return author.Definition{Name: "measurement", Inputs: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Effects: map[string]author.Effect{"read": {Capability: "meter", Input: schema.Record{"station": {Kind: "string", MaxBytes: 64}}, Output: schema.Record{"sample": {Kind: "string", MaxBytes: 64}}}}, Steps: []author.Step{
		author.Request("operation", "read", map[string]author.Expr{"station": author.Var("station")}, nil), author.Await("read", "operation", map[string]string{"sample": "sample"}), author.Success()}}
}
func declaration(n int) (author.Series, error) {
	return author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: n, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
}
func (r *report) capture(ctx context.Context, decl author.Series, m *meter, label string, position int, rr *aruntime.RunReport) error {
	fmt.Fprintf(os.Stderr, "source=%s scenario=%s cut=%s position=%d\n", r.SourceLabel, r.Scenario, label, position)
	s, err := aruntime.InspectExecution(ctx, r.Store, aruntime.DefaultLimits())
	if err != nil {
		return err
	}
	var v author.SeriesView
	v, err = decl.Inspect(s)
	if err != nil {
		return err
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
	if replay.Namespace != s.Namespace || replay.HistoryCount != len(history) || len(replay.Runs) != r.Bound {
		return fmt.Errorf("cut %s replay differs from original retained computation", label)
	}
	m.mu.Lock()
	contacts := len(m.contacts)
	adapterContacts := append([]contact{}, m.contacts...)
	m.mu.Unlock()
	r.Cuts = append(r.Cuts, cut{Label: label, Position: position, Snapshot: s, History: history, Replay: replay, View: v, Contacts: contacts, AdapterContacts: adapterContacts, ServerContacts: m.serverCopy(), RunReport: effects.Copy(rr)})
	return nil
}
func (r *report) execute(ctx context.Context) (err error) {
	decl, err := declaration(r.Bound)
	if err != nil {
		return err
	}
	m := &meter{}
	if r.Mode == "execute-existing" {
		m, err = newMeter(r.Transport)
		if err != nil {
			return err
		}
		r.Transport.Endpoint = m.endpoint
	}
	defer func() {
		err = errors.Join(err, m.finishReport(r))
		if err != nil {
			r.OK = false
		}
	}()
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
		if e = r.capture(ctx, decl, m, "seed-closed", 0, nil); e != nil {
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
	if err = r.capture(ctx, decl, m, "before-open", 0, nil); err != nil {
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
	if err = r.capture(ctx, decl, m, "after-open", 0, nil); err != nil {
		return err
	}
	verifier, err := newCutVerifier(initial.Snapshot.Roots, r.Bound)
	if err != nil {
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
			if err = r.capture(ctx, decl, m, "after-approval", i, nil); err != nil {
				return err
			}
		}
		rr, runErr := store.Run(ctx, opts)
		captureErr := r.capture(ctx, decl, m, "after-run", i, &rr)
		if err = errors.Join(runErr, captureErr); err != nil {
			return err
		}
		if _, err = verifier.verify(rr, i, true); err != nil {
			return err
		}
		last := r.Cuts[len(r.Cuts)-1]
		if last.Contacts != i || last.View.Spent != i || last.View.Selected != i {
			return fmt.Errorf("position %d contact/charge/selection mismatch", i)
		}
		if err = verifyPrefix(last, m, i); err != nil {
			return err
		}
		if r.ReopenEvery > 0 && i%r.ReopenEvery == 0 {
			if err = store.Close(); err != nil {
				return err
			}
			store = nil
			store, err = aruntime.Open(ctx, r.Store, aruntime.DefaultLimits())
			if err != nil {
				return err
			}
			rr, runErr := store.Run(ctx, opts)
			captureErr := r.capture(ctx, decl, m, "after-interval-reopen", i, &rr)
			if err = errors.Join(runErr, captureErr); err != nil {
				return err
			}
			if _, err = verifier.verify(rr, i, false); err != nil {
				return err
			}
			if err = verifyPrefix(r.Cuts[len(r.Cuts)-1], m, i); err != nil {
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
	if err = r.capture(ctx, decl, m, "final-closed", r.Bound, nil); err != nil {
		return err
	}
	final := r.Cuts[len(r.Cuts)-1]
	if final.View.Completed != r.Bound || final.View.Failed != 0 || final.View.Selected != r.Bound || final.View.Spent != r.Bound || final.View.Unresolved != 0 || final.View.Pending != 0 || final.Contacts != r.Bound || len(final.Snapshot.Continuations) != r.Bound-1 {
		return fmt.Errorf("final series accounting mismatch: %+v", final.View)
	}
	if err = verifyPrefix(final, m, r.Bound); err != nil {
		return err
	}
	r.OK = true
	return nil
}
func main() {
	mode := flag.String("mode", "", "create-seed or execute-existing")
	db := flag.String("db", "", "seed/output store path (parent directory must exist)")
	bound := flag.Int("bound", 8, "series bound")
	scenario := flag.String("scenario", "success", "success only; failure coverage reuses existing direct and real crash fixtures")
	reopen := flag.Int("reopen-every", 0, "reopen after each K selected positions, including the final position; 0 disables")
	label := flag.String("source-label", "unspecified", "source provenance label")
	endpoint := flag.String("endpoint", "", "fixed numeric-loopback listen address; no ephemeral port permitted")
	timeout := flag.Duration("timeout", 10*time.Minute, "whole qualification deadline")
	flag.Parse()
	if (*mode != "create-seed" && *mode != "execute-existing") || *db == "" || *bound < 1 || *bound > 256 || *reopen < 0 || *scenario != "success" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "require -mode create-seed|execute-existing -db PATH [-bound 8] [-scenario success] [-reopen-every K]")
		os.Exit(2)
	}
	if *mode == "execute-existing" {
		host, port, e := net.SplitHostPort(*endpoint)
		portNumber, portErr := strconv.Atoi(port)
		if portErr != nil || portNumber < 1 || portNumber > 65535 || e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port == "" || port == "0" {
			fmt.Fprintln(os.Stderr, "require a fixed numeric-loopback endpoint")
			os.Exit(2)
		}
	}
	r := report{Version: "alan/performance-p2-http-differential/v1", Mode: *mode, Bound: *bound, Scenario: *scenario, ReopenEvery: *reopen, SourceLabel: *label, Store: *db, Transport: frozenTransport("http", *endpoint)}
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
