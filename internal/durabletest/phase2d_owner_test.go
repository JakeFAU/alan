//go:build integration

package durabletest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	rq "github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

// TestPhase2DOwner runs only when started as a controlled executor process.
// Its cut selector observes committed local state; it supplies no authored data.
func TestPhase2DOwner(t *testing.T) {
	path := os.Getenv("ALAN_2D_OWNER_DB")
	if path == "" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	audit, err := sql.Open("sqlite3", "file:"+path+"?mode=ro&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	defer audit.Close()
	cut, subject := os.Getenv("ALAN_2D_CUT"), os.Getenv("ALAN_2D_SUBJECT")
	manualCut := errors.New("manual service turn")
	observer := func(ctx context.Context, e boundary.Event) error {
		if cut == "" || (subject != "" && e.Subject != subject) {
			return nil
		}
		matches := e.Boundary == cut
		if cut == "issuer-before-end" && e.Boundary == "execution.staged" && e.Phase == "terminal" {
			var issued int
			err := audit.QueryRowContext(ctx, "SELECT count(*) FROM runs WHERE subject=? AND json_type(projection,'$.Data.original')='object'", e.Subject).Scan(&issued)
			if err != nil {
				return err
			}
			matches = issued == 1
		}

		if cut == "first-final-ack" || cut == "last-final-ack" || cut == "both-final-acks" {
			if e.Boundary != "execution.committed" || e.Intent == "" {
				return nil
			}
			var desc []byte
			if err := audit.QueryRowContext(ctx, "SELECT descriptor FROM intents WHERE id=?", e.Intent).Scan(&desc); err != nil {
				return err
			}
			var intent struct{ Kind, Obligation string }
			if err := json.Unmarshal(desc, &intent); err != nil {
				return err
			}
			ob := "l_report"
			if cut == "last-final-ack" {
				ob = "r_report"
			}
			matches = intent.Kind == "message" && (intent.Obligation == ob || cut == "both-final-acks" && intent.Obligation == "r_report")
		}
		if cut == "left-original-selected" && e.Boundary == "execution.committed" {
			var b []byte
			if err := audit.QueryRowContext(ctx, "SELECT projection FROM runs WHERE subject=? AND generation=?", subject, e.Generation).Scan(&b); err != nil {
				return err
			}
			var projection runtime.Projection
			if err := json.Unmarshal(b, &projection); err != nil {
				return err
			}
			matches = projection.Data["l_source_status"].Text != "" && projection.Data["r_source_status"].Text == ""
		}
		if !matches {
			return nil
		}
		if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"event": e, "cut": cut, "pid": os.Getpid(), "namespace": s.Namespace()}); err != nil {
			return err
		}
		// The controller either sends SIGKILL, or ends the held run for a
		// supported manual-driver cut. All causal reads precede this acknowledgment.
		var release string
		if _, err := fmt.Fscan(os.Stdin, &release); err != nil {
			return err
		}
		if release == "manual" && cut == "last-final-ack" {
			return manualCut
		}
		if release != "continue" {
			return fmt.Errorf("unexpected controller input %q", release)
		}
		return nil
	}
	apply := os.Getenv("ALAN_2D_ENDPOINT")
	read, err := rq.LookupEndpoint(apply)
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := strconv.Atoi(os.Getenv("ALAN_2D_CAPACITY"))
	if err != nil {
		t.Fatal(err)
	}
	// On an adverse manual prefix commit the already-ready parent instructions
	// through End without first dispatching its outstanding final send.
	endManually := func() {
		p := runtime.RunID{Subject: "parent", Generation: 1}
		for steps := 0; ; steps++ {
			r, err := s.Snapshot(ctx, p)
			if err != nil || steps > 8 {
				t.Fatal("unexpected final parent prefix", r, err)
			}
			if r.Projection.Status != "open" {
				break
			}
			if _, err = s.Commit(ctx, r.Authority(), ""); err != nil {
				t.Fatal(err)
			}
		}
	}
	mode := os.Getenv("ALAN_2D_POLICY")
	if mode == "" {
		mode = "success"
	}
	opts := runtime.RunOptions{MaxSteps: 2048, MaxDispatch: 2048, MaxInFlight: capacity, ExternalTimeout: 100 * time.Second,
		Adapters: map[string]effects.Adapter{rq.CounterCapability: rq.Counter{Endpoint: apply, Mode: mode}, rq.ReadCapability: rq.Reader{Endpoint: read}}}
	result, err := s.Run(boundary.WithObserver(ctx, observer), opts)
	if errors.Is(err, manualCut) {
		endManually()
		result, err = s.Run(ctx, opts)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"report": result, "pid": os.Getpid(), "namespace": s.Namespace()}); err != nil {
		t.Fatal(err)
	}
}
