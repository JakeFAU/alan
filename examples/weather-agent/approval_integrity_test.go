package main

import (
	"bytes"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Copy a closed store before a legitimate approval. Independent receivers can
// then vary one field without guessing any namespace-bound application data.
func copyWeatherApprovalCut(t *testing.T, db string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "weather.sqlite")
	for _, suffix := range []string{"", "-wal"} {
		b, err := os.ReadFile(db + suffix)
		if suffix == "-wal" && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst+suffix, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// Recover even the transport identity from an actual successful CLI approval.
// A different controller is authorized to call the public local-input API.
func retainedWeatherApproval(t *testing.T, db string) runtime.Input {
	t.Helper()
	cut := weatherSnapshot(t, db)
	for _, a := range cut.Admissions {
		if a.Domain == "input" && a.Accepted && a.Target == "weather-2" && a.Obligation == "approve-refresh" {
			o, ok := cut.Observation(a.ObservationID)
			if !ok {
				t.Fatal("accepted approval has no observation")
			}
			return runtime.Input{TransportID: a.TransportID, LogicalID: a.LogicalID, Target: a.Target,
				Generation: a.RequestedGeneration, Obligation: a.Obligation, Operation: a.Correlation, Payload: maps.Clone(o.Payload)}
		}
	}
	t.Fatal("CLI failed to retain its approval")
	return runtime.Input{}
}

func TestWeatherApprovalIntegrity(t *testing.T) {
	reply := weatherReply{status: 200, body: morningWeather}
	p := newWeatherProvider(t, reply, reply, reply, reply, reply)
	source := filepath.Join(t.TempDir(), "weather.sqlite")
	createM4Weather(t, source, p.endpoint(), 2)
	r := appCommand(t, "run", "--db", source)
	if r.Reads[0].State != "reported" || r.OpportunitiesSpent != 1 {
		t.Fatal("expected a completed first observation")
	}
	wrongReason := copyWeatherApprovalCut(t, source)
	wrongHandle := copyWeatherApprovalCut(t, source)
	requireWeatherText(t, humanWeatherCommand(t, "status", "--db", source),
		[]string{"Optional separate approval for read 2"}, nil)
	appCommand(t, "refresh", "--db", source, "--read", "2")
	requireWeatherText(t, humanWeatherCommand(t, "status", "--db", source),
		[]string{"Read 2. Approved", "weather-agent resume"}, []string{"weather-agent refresh"})
	control := retainedWeatherApproval(t, source)
	foreign := filepath.Join(t.TempDir(), "weather.sqlite")
	if got := createM4Weather(t, foreign, p.endpoint(), 2); got.Namespace == r.Namespace || got.OpportunitiesSpent != 0 {
		t.Fatal("expected an unrelated, unstarted computation")
	}

	for _, tc := range []struct {
		name, db, field, replacement string
		allowedOriginalContacts      int
	}{
		{"completed_original_described_as_unknown", wrongReason, "purpose", "new_read_after_unknown_response", 0},
		{"different_predecessor", wrongHandle, "previous_operation_id", "unrelated-operation", 0},
		{"genuine_approval_from_another_store_before_any_request", foreign, "", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := control
			input.Payload = maps.Clone(control.Payload)
			if tc.field != "" {
				input.Payload[tc.field] = schema.Text(tc.replacement)
			}
			owner, err := runtime.Open(t.Context(), tc.db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			_, err = owner.AdmitInput(t.Context(), input)
			if err := errors.Join(err, owner.Close()); err != nil {
				t.Fatal(err)
			}
			priorContacts := len(p.contacts())
			for _, action := range []string{"inspect", "status", "replay"} {
				var out bytes.Buffer
				if err := run(t.Context(), []string{action, "--db", tc.db, "--json"}, &out); err == nil {
					if got := decodeWeather(t, out.String()).read(t, 2); got.State == "refresh_approved" {
						t.Errorf("%s reported an approval with a false predecessor or recovery reason as ready", action)
					}
				}
			}
			// Refusing this deployment or continuing only its original work are
			// both safe. The application must not spend read 2 on this approval.
			_ = run(t.Context(), []string{"resume", "--db", tc.db}, io.Discard)
			if n := len(p.contacts()) - priorContacts; n > tc.allowedOriginalContacts {
				t.Errorf("endpoint accepted %d requests, at most %d original requests were eligible", n, tc.allowedOriginalContacts)
			}
			for _, e := range weatherSnapshot(t, tc.db).Effects {
				if e.Source.Subject == "weather-2" && (e.DispatchAdmission != nil || e.AdapterAttempt != nil && e.AdapterAttempt.Allowed) {
					t.Errorf("read 2 spent physical authority on inapplicable approval %s", input.LogicalID)
				}
			}
		})
	}
}
