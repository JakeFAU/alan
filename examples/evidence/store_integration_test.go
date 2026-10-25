//go:build integration

package evidence_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Reading Runs from a second transaction while the runner progresses must fail
// this contract even if each transaction separately reads valid committed data.
func TestInspectConcurrentProgressUsesOneCommittedView(t *testing.T) {
	for _, decision := range []string{"clarification", "lookup"} {
		t.Run(decision, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			entered, release := make(chan struct{}), make(chan struct{})
			var first, releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				first.Do(func() { close(entered) })
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				var request struct{ Input []struct{ Content string } }
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				var input map[string]any
				if len(request.Input) != 1 {
					t.Error("missing input")
					return
				}
				if err := json.Unmarshal([]byte(request.Input[0].Content), &input); err != nil {
					t.Error(err)
					return
				}
				payload := map[string]any{"decision": decision, "tool": "lookup", "query": "WAL", "message": "Which guarantee?"}
				if _, ok := input["snippet"]; ok {
					payload = map[string]any{"answer": "WAL records changes before updating the main database.", "document": input["document"], "version": input["version"], "snippet": input["snippet"], "start": input["start"], "end": input["end"]}
				}
				text, _ := json.Marshal(payload)
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "completed", "model": "controlled", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(text)}}}}})
			}))
			defer server.Close()
			defer unblock()
			c := evidence.DefaultConfig()
			c.Model = "controlled"
			c.Endpoint = server.URL
			p, err := evidence.Provision("question", c, nil)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "view.sqlite")
			store, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			initial, err := evidence.Inspect(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			if err = coherentView(initial); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, err := evidence.Run(ctx, path, "controlled-secret"); done <- err }()
			joined := false
			defer func() {
				unblock()
				cancel()
				if !joined {
					<-done
				}
			}()
			select {
			case <-entered:
			case err := <-done:
				joined = true
				t.Fatalf("runner ended before held request: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			held, err := evidence.Inspect(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			if err = coherentView(held); err != nil {
				t.Fatal(err)
			}
			if held.ModelRequests != 1 || held.Attempts != 1 || len(held.Results) != 0 || len(held.Unknown) != 1 {
				t.Fatalf("held request summary=%+v", held)
			}
			unblock()
			snapshots := 0
			for {
				view, err := evidence.Inspect(ctx, path)
				if err != nil {
					t.Fatal(err)
				}
				snapshots++
				if err = coherentView(view); err != nil {
					t.Fatalf("snapshot %d: %v", snapshots, err)
				}
				select {
				case err := <-done:
					joined = true
					if err != nil {
						t.Fatal(err)
					}
					final, err := evidence.Inspect(ctx, path)
					if err != nil {
						t.Fatal(err)
					}
					if err = coherentView(final); err != nil {
						t.Fatal(err)
					}
					wantDisposition := "clarification"
					wantModel, wantLookup := 1, 0
					if decision == "lookup" {
						wantDisposition = "answer"
						wantModel, wantLookup = 2, 1
					}
					if final.Result["disposition"].Text != wantDisposition || final.ModelRequests != wantModel || final.LookupRequests != wantLookup {
						t.Fatalf("final result/counts=%+v", final)
					}
					t.Logf("checked %d progressing views plus initial, held and final snapshots; result=%s model=%d lookup=%d", snapshots, wantDisposition, wantModel, wantLookup)
					return
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				default:
				}
			}
		})
	}
}

// Expected fields are independently decoded from the returned public snapshot,
// including subject revisions. The function never obtains a second database read.
func coherentView(v evidence.View) error {
	rows := v.Snapshot.Tables["runs"]
	if len(rows) != len(v.Runs) {
		return fmt.Errorf("run count: snapshot=%d summary=%d", len(rows), len(v.Runs))
	}
	revisions := map[string]int64{}
	for _, row := range v.Snapshot.Tables["subjects"] {
		revisions[row["id"].(string)] = row["revision"].(int64)
	}
	seen := map[runtime.RunID]bool{}
	var caller map[string]schema.Value
	for _, run := range v.Runs {
		if seen[run.ID] {
			return fmt.Errorf("duplicate run %v", run.ID)
		}
		seen[run.ID] = true
		matched := false
		for _, row := range rows {
			if row["subject"] != run.ID.Subject || row["generation"] != run.ID.Generation {
				continue
			}
			matched = true
			var projection runtime.Projection
			if err := json.Unmarshal([]byte(row["projection"].(string)), &projection); err != nil {
				return err
			}
			if run.ProgramID != row["program_id"] || run.Epoch != row["epoch"] || run.Sequence != row["sequence"] || run.Revision != revisions[run.ID.Subject] || !reflect.DeepEqual(run.Projection, projection) {
				return fmt.Errorf("run %v differs: snapshot sequence=%v projection=%+v; summary sequence=%d projection=%+v", run.ID, row["sequence"], projection, run.Sequence, run.Projection)
			}
			if run.ID.Subject == "caller" {
				caller = projection.Data
			}
		}
		if !matched {
			return fmt.Errorf("run %v absent in snapshot", run.ID)
		}
	}
	if caller == nil {
		return fmt.Errorf("caller absent")
	}
	var wantResult map[string]schema.Value
	if _, ok := caller["disposition"]; ok {
		wantResult = map[string]schema.Value{}
		for _, key := range []string{"disposition", "answer", "document", "version", "snippet", "start", "end"} {
			wantResult[key] = caller[key]
		}
	}
	if !reflect.DeepEqual(v.Result, wantResult) {
		return fmt.Errorf("public result differs: snapshot=%v summary=%v", wantResult, v.Result)
	}
	var config evidence.Config
	if err := json.Unmarshal([]byte(caller["config"].Text), &config); err != nil {
		return err
	}
	if v.Config != config {
		return fmt.Errorf("configuration differs")
	}
	model, lookup := 0, 0
	for _, row := range v.Snapshot.Tables["intents"] {
		var intent struct{ Kind, Target string }
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &intent); err != nil {
			return err
		}
		if intent.Kind == "adapter" {
			switch intent.Target {
			case evidence.ModelCapability:
				model++
			case evidence.LookupCapability:
				lookup++
			}
		}
	}
	if model != v.ModelRequests || lookup != v.LookupRequests {
		return fmt.Errorf("logical counts differ: snapshot=%d/%d summary=%d/%d", model, lookup, v.ModelRequests, v.LookupRequests)
	}
	for _, run := range v.Runs {
		if spent, ok := run.Projection.Data["model_spent"]; ok && spent.Number != int64(model) {
			return fmt.Errorf("model spent %d differs from logical count %d", spent.Number, model)
		}
		if spent, ok := run.Projection.Data["lookup_spent"]; ok && spent.Number != int64(lookup) {
			return fmt.Errorf("lookup spent %d differs from logical count %d", spent.Number, lookup)
		}
	}
	results := map[string]runtime.AdapterResult{}
	for _, row := range v.Snapshot.Tables["adapter_results"] {
		var result runtime.AdapterResult
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &result); err != nil {
			return err
		}
		results[row["intent_id"].(string)] = result
	}
	if len(v.Results) != len(results) {
		return fmt.Errorf("result count differs")
	}
	for _, result := range v.Results {
		want, ok := results[result.Candidate.ID]
		if !ok || !reflect.DeepEqual(want, result) {
			return fmt.Errorf("result %s differs", result.Candidate.ID)
		}
	}
	allowed, denied := 0, 0
	var unknown []string
	for _, row := range v.Snapshot.Tables["adapter_attempts"] {
		var attempt runtime.AdapterAttempt
		if err := json.Unmarshal([]byte(row["descriptor"].(string)), &attempt); err != nil {
			return err
		}
		if attempt.Allowed {
			allowed++
			if _, ok := results[row["intent_id"].(string)]; !ok {
				unknown = append(unknown, row["intent_id"].(string))
			}
		} else {
			denied++
		}
	}
	if allowed != v.Attempts || denied != v.DeniedAttempts {
		return fmt.Errorf("attempt counts differ")
	}
	slices.Sort(unknown)
	gotUnknown := slices.Clone(v.Unknown)
	slices.Sort(gotUnknown)
	if !slices.Equal(unknown, gotUnknown) {
		return fmt.Errorf("unknown identities differ: snapshot=%v summary=%v", unknown, gotUnknown)
	}
	return nil
}
