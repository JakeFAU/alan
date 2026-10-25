package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/mattn/go-sqlite3"
)

// TestTask4FaultProcess is an explicitly test-only faulty dispatcher. Neither this
// code nor its environment switch exists in the production CLI. The real boundary
// observer still only reports and pauses after the separate fault has happened.
func TestTask4FaultProcess(t *testing.T) {
	if os.Getenv("ALAN_TEST_FAULT") != "speculative-dispatch" {
		return
	}
	path := os.Getenv("ALAN_TEST_STORE")
	ctx, err := boundary.FromEnvironment(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(ctx, path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.conn.Raw(func(connection any) error {
		return connection.(*sqlite3.SQLiteConn).RegisterFunc("fault_dispatch", func(raw string, epoch int64) int {
			var intent Intent
			if err := json.Unmarshal([]byte(raw), &intent); err != nil {
				panic(err)
			}
			request, err := externalRequest(intent)
			if err != nil {
				panic(err)
			}
			request.Epoch = epoch
			request.Authorization = identity("authorization", intent.ID)
			encoded, _ := json.Marshal(request)
			response, err := http.Post(intent.Target, "application/json", bytes.NewReader(encoded))
			if err != nil {
				panic(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				panic(fmt.Sprintf("fault endpoint status %d", response.StatusCode))
			}
			return 0
		}, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.conn.ExecContext(ctx, `CREATE TEMP TRIGGER inject_speculative_dispatch BEFORE UPDATE OF projection ON runs WHEN EXISTS(SELECT 1 FROM intents WHERE subject=NEW.subject AND generation=NEW.generation AND sequence=NEW.sequence) BEGIN SELECT fault_dispatch((SELECT CAST(descriptor AS TEXT) FROM intents WHERE subject=NEW.subject AND generation=NEW.generation AND sequence=NEW.sequence LIMIT 1),NEW.epoch); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Run(ctx, RunOptions{100, 100}); err != nil {
		t.Fatal(err)
	}
}
