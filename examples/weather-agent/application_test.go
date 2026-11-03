package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func appCommand(t *testing.T, args ...string) summary {
	t.Helper()
	var out bytes.Buffer
	if err := run(t.Context(), append(args, "--json"), &out); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var got summary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func provisioning(db, endpoint string, allowance int) []string {
	return []string{"start", "--db", db, "--lat", "26.93", "--lon", "-80.08", "--units", "fahrenheit", "--endpoint", endpoint, "--allowance", fmt.Sprint(allowance)}
}

func TestWeatherApplicationApprovesEachRead(t *testing.T) {
	var contacts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := contacts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"latitude":26.94,"longitude":-80.09,"utc_offset_seconds":0,"timezone":"GMT","current_units":{"time":"iso8601","interval":"seconds","temperature_2m":"°F","relative_humidity_2m":"%%"},"current":{"time":"2026-10-01T15:%02d","interval":900,"temperature_2m":%d,"relative_humidity_2m":73}}`, 15*n, 70+n)
	}))
	defer server.Close()
	db := filepath.Join(t.TempDir(), "weather.sqlite")
	start := provisioning(db, server.URL+"/v1/forecast", 2)
	first := appCommand(t, start...)
	if contacts.Load() != 0 || len(first.Reads) != 2 || first.OpportunitiesSpent != 0 {
		t.Fatalf("start: %+v", first)
	}
	if got := appCommand(t, start...); got.Namespace != first.Namespace {
		t.Fatal("identical provisioning replaced original computation")
	}
	if err := run(t.Context(), append([]string{"resume", "--db", db}, "--endpoint", server.URL), io.Discard); err == nil {
		t.Fatal("existing endpoint was replaceable")
	}
	refresh := []string{"refresh", "--db", db, "--read", "2"}
	if err := run(t.Context(), refresh, io.Discard); err == nil {
		t.Fatal("approved a refresh of a nonexistent operation")
	}
	one := appCommand(t, "run", "--db", db)
	a := one.Reads[0]
	if contacts.Load() != 1 || one.OpportunitiesSpent != 1 || a.DispatchAdmission == nil || a.OperationID == "" || a.SelectedObservationID == "" || a.Report == "" || a.Temperature != "71" || a.ResultOrigin != "read_during_this_run" || a.ObservedAt != "2026-10-01T15:15:00Z" || strings.Contains(one.Attribution, "Weather data by Open-Meteo") {
		t.Fatalf("first observation: %+v", one)
	}
	pending := appCommand(t, refresh...).Reads[1]
	if pending.State != "refresh_approved" || pending.ApprovalSelectedAt != nil ||
		pending.ApprovalPreparedDigest != a.DispatchAdmission.PreparedDigest || pending.ApprovalResultDigest != a.ResultDigest {
		t.Fatalf("pending approval did not identify its original reservation and result: %+v", pending)
	}
	_ = appCommand(t, refresh...)
	if contacts.Load() != 1 {
		t.Fatal("admitting approval contacted the endpoint")
	}
	two := appCommand(t, "resume", "--db", db)
	b := two.Reads[1]
	if contacts.Load() != 2 || two.OpportunitiesSpent != 2 || b.OperationID == a.OperationID || b.PreviousOperationID != a.OperationID || b.ApprovalReason != afterReport || b.ApprovalSelectedAt == nil || b.Temperature != "72" || b.ObservedAt != "2026-10-01T15:30:00Z" || b.ResultOrigin != "read_during_this_run" || two.Reads[0].ResultOrigin != "previously_retained" || two.Reads[0].Report != a.Report || b.SelectedObservationID == a.SelectedObservationID {
		t.Fatalf("second observation: %+v", two)
	}
	_ = appCommand(t, refresh...)
	if err := run(t.Context(), []string{"refresh", "--db", db, "--read", "3"}, io.Discard); err == nil || !strings.Contains(err.Error(), "allowance") {
		t.Fatalf("outside allowance: %v", err)
	}
	for _, command := range []string{"inspect", "replay", "resume"} {
		got := appCommand(t, command, "--db", db)
		if got.Reads[0].Report != a.Report || got.Reads[1].Report != b.Report || got.Reads[1].ResultOrigin != "previously_retained" || contacts.Load() != 2 {
			t.Fatalf("%s changed answers or contacted endpoint", command)
		}
	}
}

type alteredMeasurement struct{ *openmeteo.Adapter }

func (a alteredMeasurement) Execute(ctx context.Context, p effects.Prepared) effects.Candidate {
	c := a.Adapter.Execute(ctx, p)
	if c.Disposition == "validated" {
		c.Payload[openmeteo.Temperature] = schema.Text("99")
	}
	return c
}

// Even trusted runtime users can supply a different adapter. The example must
// establish its own endpoint and native-body evidence before displaying weather
// or giving the next owner authority to contact the configured service.
func TestWeatherApplicationRejectsSubstitutedEvidence(t *testing.T) {
	var contacts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"latitude":26.93,"longitude":-80.08,"utc_offset_seconds":0,"timezone":"GMT","current_units":{"time":"iso8601","temperature_2m":"°F","relative_humidity_2m":"%"},"current":{"time":"2026-10-01T15:15","temperature_2m":71,"relative_humidity_2m":73}}`)
	}))
	defer server.Close()
	actual := server.URL + "/v1/forecast"
	adapter, err := openmeteo.New(actual)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, configured string
		executed         effects.Adapter
	}{
		{"different_endpoint", "http://127.0.0.1:1/v1/forecast", adapter},
		{"different_measurement", actual, alteredMeasurement{adapter}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := filepath.Join(t.TempDir(), "weather.sqlite")
			appCommand(t, provisioning(db, tc.configured, 1)...)
			s, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			before := contacts.Load()
			_, err = s.Run(t.Context(), runtime.RunOptions{
				MaxSteps: 128, MaxDispatch: 32, Adapters: map[string]effects.Adapter{capability: tc.executed},
			})
			if closeErr := s.Close(); err != nil || closeErr != nil || contacts.Load() != before+1 {
				t.Fatalf("setup run: %v, close: %v, contacts: %d", err, closeErr, contacts.Load()-before)
			}
			cut, err := runtime.InspectExecution(t.Context(), db, runtime.DefaultLimits())
			if err != nil || cut.Effects[0].ResultDisposition != "validated" {
				t.Fatalf("expected a valid ALAN result from substituted adapter: %v", err)
			}
			for _, action := range []string{"inspect", "replay", "resume"} {
				var output bytes.Buffer
				err := run(t.Context(), []string{action, "--db", db}, &output)
				if err == nil || output.Len() != 0 || contacts.Load() != before+1 {
					t.Fatalf("%s accepted substituted evidence: %s (%v)", action, output.String(), err)
				}
			}
		})
	}
}

func TestWeatherApplicationZeroAllowanceAndProvisioningConflicts(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "weather.sqlite")
	endpoint := "http://127.0.0.1:1/v1/forecast"
	start := provisioning(db, endpoint, 0)
	created := appCommand(t, start...)
	blocked := appCommand(t, "run", "--db", db)
	if blocked.OpportunitiesSpent != 0 || blocked.Reads[0].Report != "" || blocked.Reads[0].State != "unavailable" {
		t.Fatalf("zero allowance: %+v", blocked)
	}
	if err := run(t.Context(), provisioning(db, endpoint, 1), io.Discard); err == nil {
		t.Fatal("different allowance replaced the deployment")
	}
	if got := appCommand(t, "inspect", "--db", db); got.Namespace != created.Namespace || got.Allowance != 0 {
		t.Fatal("original deployment changed")
	}
	foreign := filepath.Join(dir, "foreign")
	if err := os.WriteFile(foreign, []byte("original contents"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), provisioning(foreign, endpoint, 1), io.Discard); err == nil {
		t.Fatal("accepted unrelated file")
	}
	b, err := os.ReadFile(foreign)
	if err != nil || string(b) != "original contents" {
		t.Fatalf("existing file changed: %q %v", b, err)
	}
	for _, bad := range []string{"nan", "91", ""} {
		invalid := provisioning(filepath.Join(dir, "invalid"), endpoint, 1)
		invalid[4] = bad
		if err := run(t.Context(), invalid, io.Discard); err == nil {
			t.Fatalf("accepted latitude %q", bad)
		}
	}
}

func TestWeatherApplicationRejectsUnrelatedPeerHandle(t *testing.T) {
	var contacts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		contacts.Add(1)
		http.Error(w, "limited", http.StatusTooManyRequests)
	}))
	defer server.Close()
	db := filepath.Join(t.TempDir(), "weather.sqlite")
	_ = appCommand(t, provisioning(db, server.URL+"/v1/forecast", 2)...)
	s, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	refused, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "incomplete", LogicalID: "incomplete", Target: "weather-2",
		Generation: 1, Obligation: approval, Payload: map[string]schema.Value{"previous_operation_id": schema.Text("unrelated")}})
	if err != nil || refused.Accepted || refused.Reason != "schema_invalid" {
		t.Fatalf("malformed refresh input: %+v, error: %v", refused, err)
	}
	a, err := s.AdmitInput(t.Context(), runtime.Input{TransportID: "other-approval", LogicalID: "other-approval", Target: "weather-2", Generation: 1, Obligation: "approve-refresh",
		Payload: map[string]schema.Value{"previous_operation_id": schema.Text("unrelated"), "purpose": schema.Text(afterUnavailable),
			"prepared_digest": schema.Text("unrelated"), "result_digest": schema.Text("")}})
	if err != nil || !a.Accepted {
		t.Fatalf("admission: %+v, error: %v", a, err)
	}
	// Bypass the application's input check and drive the authored computation
	// directly. Its independent peer-handle comparison also rejects this input.
	adapter, err := openmeteo.New(server.URL + "/v1/forecast")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Run(t.Context(), runtime.RunOptions{MaxSteps: 128, MaxDispatch: 32, Adapters: map[string]effects.Adapter{capability: adapter}})
	if closeErr := s.Close(); err != nil || closeErr != nil {
		t.Fatalf("run: %v, close: %v", err, closeErr)
	}
	got := appCommand(t, "inspect", "--db", db)
	if len(got.RefusedInputs) != 1 || got.RefusedInputs[0].TransportID != refused.TransportID {
		t.Fatalf("missing declined local input: %+v", got.RefusedInputs)
	}
	if got.Reads[1].State != "refresh_input_rejected" || got.Reads[1].LocalStatus != "failed" || got.Reads[1].OperationID != "" || got.OpportunitiesSpent != 1 || contacts.Load() != 1 {
		t.Fatalf("unrelated input unlocked new operation: %+v", got)
	}
}
