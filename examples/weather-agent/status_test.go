package main

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func humanWeatherCommand(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(t.Context(), args, &out); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	return out.String()
}

func requireWeatherText(t *testing.T, got string, present, absent []string) {
	t.Helper()
	for _, text := range present {
		if !strings.Contains(got, text) {
			t.Errorf("missing %q in:\n%s", text, got)
		}
	}
	for _, text := range absent {
		if strings.Contains(got, text) {
			t.Errorf("unexpected %q in:\n%s", text, got)
		}
	}
}

func TestWeatherStatusGuidesOnlyEligibleReads(t *testing.T) {
	p := newWeatherProvider(t, seriesReplies()...)
	db := filepath.Join(t.TempDir(), "weather '$(printf changed)'; store.sqlite")
	appCommand(t, provisioning(db, p.endpoint(), 3)...)
	status := func() string {
		t.Helper()
		before := weatherDBBytes(t, db)
		contacts := len(p.contacts())
		got := humanWeatherCommand(t, "status", "--db", db)
		if json := humanWeatherCommand(t, "status", "--db", db, "--json"); json != humanWeatherCommand(t, "inspect", "--db", db, "--json") ||
			json != humanWeatherCommand(t, "inspect", "--db", db) {
			t.Fatal("detailed status and inspect differ for the same cut")
		}
		if before != weatherDBBytes(t, db) {
			t.Fatal("status or inspect changed the database/WAL bytes")
		}
		p.requireContacts(t, contacts)
		return got
	}

	requireWeatherText(t, humanWeatherCommand(t, "--help"), []string{"status"}, nil)
	initial := status()
	requireWeatherText(t, initial, []string{"Read 1.", "Original read can run", "Opportunities spent 0 of 3"},
		[]string{"weather-agent refresh", "Optional separate approval"})
	checkWeatherShellCommand(t, initial, "resume", db)

	for _, n := range []string{"2", "3"} {
		err := run(t.Context(), []string{"refresh", "--db", db, "--read", n}, io.Discard)
		cut := weatherSnapshot(t, db)
		target, predecessor := readID(2), readID(1)
		if n == "3" {
			target, predecessor = readID(3), readID(2)
		}
		_, basisErr := cut.ContinuationBasis(target)
		if basisErr == nil || err == nil || !errors.Is(err, basisErr) {
			t.Fatalf("read %s lost the runtime failure: %v (want %v)", n, err, basisErr)
		}
		requireWeatherText(t, err.Error(), []string{"read " + n, target.Subject, "predecessor", predecessor.Subject}, nil)
	}

	var fresh bytes.Buffer
	if err := run(t.Context(), []string{"run", "--db", db}, &fresh); err != nil {
		t.Fatal(err)
	}
	requireWeatherText(t, fresh.String(), []string{"Read 2. Eligible for separate approval", "Read 3. Waiting for read 2"}, nil)
	first := status()
	original := p.contacts()[0].operation
	requireWeatherText(t, first, []string{"Previously retained", original, "Response retained (HTTP 200)", "Observation selected", "Opportunity spent",
		"Optional separate approval for read 2", "Read 3. Waiting for read 2", "Opportunities spent 1 of 3"},
		[]string{"weather-agent resume", "--read 3"})
	checkWeatherShellCommand(t, first, "refresh", db, "--read", "2")

	r := appCommand(t, "refresh", "--db", db, "--read", "2").Reads[1]
	duplicate := appCommand(t, "refresh", "--db", db, "--read", "2").Reads[1]
	if r.ApprovalObservationID == "" || r.ApprovalObservationID != duplicate.ApprovalObservationID {
		t.Fatal("repeated approval changed the retained decision")
	}
	approved := status()
	requireWeatherText(t, approved, []string{"Read 2. Approved", "Resume may contact the provider", "Opportunities spent 1 of 3"},
		[]string{"weather-agent refresh"})
	checkWeatherShellCommand(t, approved, "resume", db)
	appCommand(t, "resume", "--db", db)
	requireWeatherText(t, status(), []string{"Optional separate approval for read 3", "Opportunities spent 2 of 3"}, []string{"--read 2"})
	appCommand(t, "refresh", "--db", db, "--read", "3")
	appCommand(t, "resume", "--db", db)
	requireWeatherText(t, status(), []string{"Opportunities spent 3 of 3", "No remaining read slot"}, []string{"weather-agent refresh", "weather-agent resume"})
	p.requireContacts(t, 3)
}

func TestWeatherStatusDetectsOccupiedLegacyApproval(t *testing.T) {
	p := newWeatherProvider(t, seriesReplies()...)
	source := filepath.Join(t.TempDir(), "legacy.sqlite")
	createM4Weather(t, source, p.endpoint(), 2)
	original := appCommand(t, "resume", "--db", source).Reads[0]
	if original.State != "reported" || original.OperationID == "" {
		t.Fatalf("missing completed predecessor: %+v", original)
	}
	for _, tc := range []struct{ name, target, reason string }{
		{"refused_canonical_route", "weather-2", "schema_invalid"},
		{"refused_other_route", "foreign", "target_missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := copyWeatherApprovalCut(t, source)
			owner, err := runtime.Open(t.Context(), db, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			const transport = "weather-refresh-2"
			refused, err := owner.AdmitInput(t.Context(), runtime.Input{
				TransportID: transport, LogicalID: transport, Target: tc.target, Generation: 1, Obligation: approval,
				Payload: map[string]schema.Value{"previous_operation_id": schema.Text("incomplete")},
			})
			if err := errors.Join(err, owner.Close()); err != nil {
				t.Fatal(err)
			}
			if refused.Accepted || refused.Reason != tc.reason || refused.TransportID != transport {
				t.Fatalf("expected the retained refusal, got %+v", refused)
			}
			before := weatherDBBytes(t, db)
			detailed := humanWeatherCommand(t, "inspect", "--db", db)
			got := humanWeatherCommand(t, "status", "--db", db)
			requireWeatherText(t, got,
				[]string{original.OperationID, original.Report, "Read 2. Approval blocked", transport, "retained input"},
				[]string{"weather-agent refresh", "Eligible for separate approval", "weather-agent resume"})
			if detailed != humanWeatherCommand(t, "status", "--db", db, "--json") || before != weatherDBBytes(t, db) {
				t.Fatal("status changed the detailed report or modified the database/WAL")
			}
			err = run(t.Context(), []string{"refresh", "--db", db, "--read", "2"}, io.Discard)
			if !errors.Is(err, runtime.ErrConflict) {
				t.Fatalf("refresh should preserve its conflict, got %v", err)
			}
			if after := weatherSnapshot(t, db); len(after.Effects) != 1 || weatherEffect(t, after, "weather-1").OperationID != original.OperationID {
				t.Fatal("the failed approval changed original effects")
			}
			p.requireContacts(t, 1)
		})
	}
	// The same fixed key still works when it contains a valid approval.
	appCommand(t, "refresh", "--db", source, "--read", "2")
	requireWeatherText(t, humanWeatherCommand(t, "status", "--db", source),
		[]string{"Read 2. Approved", "weather-agent resume"}, []string{"Approval blocked", "weather-agent refresh"})
	p.requireContacts(t, 1)
}

func checkWeatherShellCommand(t *testing.T, status, action, db string, extra ...string) {
	t.Helper()
	prefix := "weather-agent " + action + " "
	start := strings.Index(status, prefix)
	if start == -1 {
		t.Fatalf("status omitted %s command: %s", action, status)
	}
	command, _, _ := strings.Cut(status[start:], "\n")
	args := append([]string{action, "--db", db}, extra...)
	script := "check_args() { printf '%s\\n' \"$@\"; }\ncheck_args" + strings.TrimPrefix(command, "weather-agent")
	out, err := exec.CommandContext(t.Context(), "sh", "-c", script).CombinedOutput()
	if err != nil || string(out) != strings.Join(args, "\n")+"\n" {
		t.Fatalf("copying status guidance changed its arguments: %v, %q", err, out)
	}
}
