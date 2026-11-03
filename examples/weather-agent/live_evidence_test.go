package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// The live demonstration is a stopped database. Exercise the real application
// readers offline against its manifest rather than consulting today's weather.
func TestWeatherArchivedLiveObservation(t *testing.T) {
	raw, err := os.ReadFile("testdata/live.json")
	if err != nil {
		t.Fatal(err)
	}
	var witness struct {
		Namespace, Operation, Report, Temperature string
		Database                                  string `json:"sqlite_sha256"`
		Body                                      string `json:"response_sha256"`
		Time                                      string `json:"provider_time_utc"`
		Humidity                                  string `json:"relative_humidity"`
		Observation                               string `json:"selected_observation"`
		Commits                                   int    `json:"replay_reductions"`
	}
	if err := json.Unmarshal(raw, &witness); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("testdata/live.sqlite.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	db, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(db)
	if hex.EncodeToString(digest[:]) != witness.Database {
		t.Fatal("original live database digest differs")
	}
	p := filepath.Join(t.TempDir(), "live.sqlite")
	if err := os.WriteFile(p, db, 0600); err != nil {
		t.Fatal(err)
	}
	before := weatherDBBytes(t, p)
	for _, action := range []string{"inspect", "replay"} {
		s := appCommand(t, action, "--db", p)
		if s.Namespace != witness.Namespace || s.OpportunitiesSpent != 1 || s.Allowance != 1 || s.ProviderCostUSD != nil || len(s.Reads) != 1 {
			t.Fatal("unexpected live account", s)
		}
		read := s.Reads[0]
		if read.State != "reported" || read.HTTPStatus != 200 || read.ProviderContacts != (contactBounds{1, 1}) || read.Disposition != "validated" || read.OperationID != witness.Operation || read.SelectedObservationID != witness.Observation || read.Report != witness.Report || read.ObservedAt != witness.Time || read.Temperature != witness.Temperature || read.Humidity != witness.Humidity {
			t.Fatal("live observation differs", read)
		}
		if action == "inspect" {
			sum := sha256.Sum256([]byte(read.Response))
			if hex.EncodeToString(sum[:]) != witness.Body {
				t.Fatal("provider body digest differs")
			}
		}
		if action == "replay" && (s.ReplayedCommits == nil || *s.ReplayedCommits != witness.Commits) {
			t.Fatal("replayed reductions differ")
		}
	}
	if weatherDBBytes(t, p) != before {
		t.Fatal("offline verification changed the database or WAL")
	}
}
