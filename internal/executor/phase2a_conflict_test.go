//go:build integration

package executor

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JakeFAU/alan/effects"
)

// Challenge the recorder directly. The public parent has no corresponding
// receipt injection method. Keep both candidates even though only one survives.
func TestPhase2ARecordConflict(t *testing.T) {
	s, path, id := committedAdapter(t)
	work, err := s.prepareAdapterWork(t.Context(), id, map[string]effects.Adapter{"model": callbackAdapter{}}, time.Second)
	if err != nil || work == nil {
		t.Fatal(work, err)
	}
	first := transportAdapter(t.Context(), *work).candidate
	retained, err := s.recordAdapterResult(t.Context(), work.attempt, first)
	if err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"foreign-key", "contradictory-result"} {
		t.Run(control, func(t *testing.T) {
			changed := effects.Copy(first)
			if control == "foreign-key" {
				changed.Key += "-other"
			} else {
				changed.Payload["answer"] = Text("incompatible")
			}
			_, err := s.recordAdapterResult(t.Context(), work.attempt, changed)
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("unexpected recorder disposition %v", err)
			}
			inspection, inspectErr := Inspect(t.Context(), path, DefaultLimits())
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			rows := inspection.Tables["adapter_results"]
			if len(rows) != 1 {
				t.Fatal("lost or duplicated original result")
			}
			var actual AdapterResult
			if e := json.Unmarshal([]byte(rows[0]["descriptor"].(string)), &actual); e != nil || !reflect.DeepEqual(actual, retained) {
				t.Fatal("changed original", e)
			}
			raw, e := json.Marshal(map[string]any{"original": first, "offered": changed, "error": err.Error(), "retained": actual, "attempt": work.attempt})
			if e != nil {
				t.Fatal(e)
			}
			t.Logf("RECORDER %s", raw)
		})
	}
}
