package main

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
)

func seriesReplies() []weatherReply {
	replies := make([]weatherReply, 8)
	for i := range replies {
		replies[i] = weatherReply{status: 200, body: fmt.Sprintf(`{"latitude":26.94,"longitude":-80.09,"utc_offset_seconds":0,"timezone":"GMT","current_units":{"time":"iso8601","interval":"seconds","temperature_2m":"°F","relative_humidity_2m":"%%"},"current":{"time":"2026-10-01T%02d:15","interval":900,"temperature_2m":%d,"relative_humidity_2m":73}}`, 10+i, 71+i)}
	}
	return replies
}

func TestWeatherSeriesEight(t *testing.T) {
	binary := publicWeatherBinary(t)
	p := newWeatherProvider(t, seriesReplies()...)
	db := startWeather(t, binary, p.endpoint(), 8)
	c := configuration{Latitude: "26.93", Longitude: "-80.08", Units: "fahrenheit", Endpoint: p.endpoint(), Allowance: 8}
	series, err := c.series()
	if err != nil {
		t.Fatal(err)
	}
	initial := weatherSnapshot(t, db)
	view, err := series.Inspect(initial)
	if err != nil {
		t.Fatal(err)
	}
	if view.Bound != 8 || view.Spent != 0 || view.Pending != 8 {
		t.Fatalf("initial: %+v", view)
	}
	p.requireContacts(t, 0)
	var previous []string
	for n := 1; n <= 8; n++ {
		if n == 2 {
			if err := run(t.Context(), []string{"refresh", "--db", db, "--read", "3"}, io.Discard); err == nil {
				t.Fatal("out-of-order third approval accepted")
			}
		}
		if n > 1 {
			for range 2 {
				weatherCommand(t, binary, "refresh", "--db", db, "--read", fmt.Sprint(n))
			}
		}
		p.requireContacts(t, n-1)
		weatherCommand(t, binary, "resume", "--db", db)
		s, report := inspectAndReplayWeather(t, binary, db, p, 8, n)
		v, err := series.Inspect(s)
		if err != nil {
			t.Fatal(err)
		}
		if v.Completed != n || v.Selected != n || v.Spent != n || v.Pending != 8-n || v.Unresolved != 0 || v.BoundExhausted != (n == 8) {
			t.Fatalf("position %d: %+v", n, v)
		}
		ids, keys, times, observations := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
		for i, pos := range v.Positions {
			if i >= n {
				if pos.Observation != nil || pos.Effect != nil {
					t.Fatal("fabricated future observation")
				}
				continue
			}
			e := pos.Effect
			if pos.State != author.SeriesCompleted || pos.Observation == nil || e == nil || e.Authorization == nil || pos.Root.Charges.Effects != 1 {
				t.Fatal("missing selected original evidence")
			}
			for _, identity := range []struct {
				value string
				seen  map[string]bool
			}{{e.OperationID, ids}, {e.AdapterAttempt.Prepared.Request.ID, keys}, {e.AdapterResult.Candidate.Payload["observed_at"].Text, times}, {pos.Observation.ID, observations}} {
				value, set := identity.value, identity.seen
				if value == "" || set[value] {
					t.Fatal("duplicate original identity or timestamp")
				}
				set[value] = true
			}
			if i < len(previous) && previous[i] != e.AdapterResult.Candidate.Response {
				t.Fatal("earlier response changed")
			}
			r := report.read(t, i+1)
			if r.Operation != e.OperationID || r.Report == "" || r.ObservedAt != fmt.Sprintf("2026-10-01T%02d:15:00Z", 10+i) || r.Temperature != fmt.Sprint(71+i) || p.contacts()[i].operation != e.OperationID {
				t.Fatal("report differs from independently scripted observation")
			}
		}
		visible := appCommand(t, "inspect", "--db", db)
		if visible.Series == nil || visible.Series.ID != "weather" || visible.Series.Bound != 8 || visible.Series.Selected != n || visible.Series.Completed != n || visible.Series.Pending != 8-n || visible.Series.BoundExhausted != (n == 8) {
			t.Fatal("visible aggregate differs from typed series")
		}
		for i := n + 1; i < 8; i++ {
			if visible.Reads[i].SeriesState != author.SeriesNotEligible {
				t.Fatal("future read is not visibly ineligible")
			}
		}
		previous = nil
		for _, pos := range v.Positions[:n] {
			previous = append(previous, pos.Effect.AdapterResult.Candidate.Response)
		}
		p.requireContacts(t, n)
	}
	deniedWeatherCommand(t, binary, "refresh", "--db", db, "--read", "9")
	for range 2 {
		weatherCommand(t, binary, "resume", "--db", db)
	}
	p.requireContacts(t, 8)
}

func TestWeatherSeriesOriginalAndLegacyPrograms(t *testing.T) {
	p := newWeatherProvider(t, seriesReplies()...)
	c := configuration{Latitude: "26.93", Longitude: "-80.08", Units: "fahrenheit", Endpoint: p.endpoint(), Allowance: 1}
	current, err := c.provision(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []bool{false, true} {
		old, err := c.legacyProvision(legacy)
		if err != nil {
			t.Fatal(err)
		}
		if current.Runs[0].ProgramID != old.Runs[0].ProgramID {
			t.Fatal("first original artifact changed")
		}
	}
	binary := publicWeatherBinary(t)
	db := startWeather(t, binary, p.endpoint(), 1)
	weatherCommand(t, binary, "resume", "--db", db)
	cut, view := inspectAndReplayWeather(t, binary, db, p, 1, 1)
	first, _ := cut.Run(readID(1))
	if first.Sequence != 15 || view.read(t, 1).Report == "" {
		t.Fatal("single original program did not finish in 15 reductions")
	}
	c.Allowance = 2
	old, err := c.legacyProvision(false)
	if err != nil {
		t.Fatal(err)
	}
	if old.Composition.Continuations[0].ApprovalID != "weather-refresh-2" || old.Composition.Continuations[0].Obligation != "approve-refresh" {
		t.Fatal("M5 source contract changed")
	}
	path := filepath.Join(t.TempDir(), "legacy-m5.sqlite")
	owner, err := runtime.Create(t.Context(), path, old, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	appCommand(t, "resume", "--db", path)
	requireWeatherText(t, humanWeatherCommand(t, "status", "--db", path),
		[]string{"Optional separate approval for read 2", "Observation selected"}, nil)
	appCommand(t, "refresh", "--db", path, "--read", "2")
	result := appCommand(t, "resume", "--db", path)
	if result.Series != nil || result.Reads[1].State != "reported" || result.OpportunitiesSpent != 2 || result.Reads[1].PreviousOperationID != result.Reads[0].OperationID {
		t.Fatal("M5 lost exact compatibility")
	}
	p.requireContacts(t, 3)
}
