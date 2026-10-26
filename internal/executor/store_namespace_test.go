package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	resource "github.com/JakeFAU/alan/internal/fixture"
)

// Removing store isolation must produce one mutation for two independent programs.
func TestStoreNamespacesSeparateSharedEndpointAndSurviveReopen(t *testing.T) {
	dir := t.TempDir()
	if retained := os.Getenv("ALAN_NAMESPACE_EVIDENCE"); retained != "" {
		dir = retained
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	endpointPath := filepath.Join(dir, "endpoint.sqlite")
	endpoint, err := resource.Open(t.Context(), endpointPath, "namespace-endpoint")
	if err != nil {
		t.Fatal(err)
	}
	defer endpoint.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resource.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		receipt, err := endpoint.Apply(r.Context(), req)
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		// Retain unknown local outcomes so reopen must retransmit the original request.
		_ = receipt
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	p := externalProvision(t, server.URL, "idempotent")
	raw, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "provision.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	var keys [2]string
	for i := range 2 {
		path := filepath.Join(dir, fmt.Sprintf("store-%d.sqlite", i))
		s, err := Create(t.Context(), path, p, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100})
		if err != nil {
			s.Close()
			t.Fatal(err)
		}
		if len(report.UnknownExternal) != 1 || report.UnknownExternal[0] != `["client",1,1,0]` {
			t.Fatalf("local identity changed: %+v", report)
		}
		attempt, err := loadExternalAttempt(t.Context(), s.conn, report.UnknownExternal[0])
		if err != nil {
			t.Fatal(err)
		}
		keys[i] = attempt.Request.Key
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		s, err = Open(t.Context(), path, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100}); err != nil {
			t.Fatal(err)
		}
		again, err := loadExternalAttempt(t.Context(), s.conn, report.UnknownExternal[0])
		if err != nil {
			t.Fatal(err)
		}
		if again.Request != attempt.Request {
			t.Fatal("reopen changed immutable request")
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := resource.Inspect(t.Context(), endpointPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.MarshalIndent(snapshot, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "endpoint-inspection.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if snapshot.Contacts != 4 || len(snapshot.Applications) != 2 || snapshot.Total != 10 || keys[0] == keys[1] {
		t.Fatalf("independent stores collided: keys=%q contacts=%d mutations=%d total=%d", keys, snapshot.Contacts, len(snapshot.Applications), snapshot.Total)
	}
	for i, key := range keys {
		if snapshot.Requests[2*i].Key != key || snapshot.Requests[2*i+1].Key != key {
			t.Fatal("endpoint did not observe preserved reopen key")
		}
	}
}

func TestStoreNamespaceSurvivesRenameAndCopy(t *testing.T) {
	s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
	state, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	id := state.Projection.Data["pending"].Text
	original, _, err := s.prepareExternal(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	before, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	namespace, ok := before.Tables["metadata"][0]["namespace"].(string)
	if !ok || len(namespace) != 64 {
		t.Fatalf("missing 256-bit namespace: %+v", before.Tables["metadata"])
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	renamed := path + ".renamed"
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(renamed)
	if err != nil {
		t.Fatal(err)
	}
	copied := path + ".copy"
	if err := os.WriteFile(copied, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{renamed, copied} {
		reopened, err := Open(t.Context(), p, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := Inspect(t.Context(), p, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		attempt, _, err := reopened.prepareExternal(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if attempt.Request != original.Request || attempt.Request.Key != ExternalKey(reopened.Namespace(), id) {
			t.Fatal("rename/copy changed operation identity or immutable request")
		}
		if snapshot.Namespace != namespace || snapshot.KeyVersion != 2 || reopened.Namespace() != namespace {
			t.Fatal("namespace inspection/accessor mismatch")
		}
		if snapshot.Tables["metadata"][0]["namespace"] != namespace {
			t.Fatal("rename/copy allocated new identity")
		}
		if err := reopened.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStoreRejectsVersionOneBeforeActivation(t *testing.T) {
	s, path := newStore(t, false)
	epoch := state(t, s).Epoch
	// Reconstruct the actual version-1 metadata shape, with no namespace columns.
	_, err := s.conn.ExecContext(t.Context(), `DROP TABLE metadata; CREATE TABLE metadata(singleton INTEGER PRIMARY KEY,schema_version INTEGER NOT NULL,semantics TEXT NOT NULL); INSERT INTO metadata VALUES(1,1,?); PRAGMA user_version=1`, SemanticVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if reopened != nil {
		reopened.Close()
	}
	if !errors.Is(err, ErrVersion) {
		t.Errorf("old store live activation: %v", err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatalf("old replay: %v", err)
	}
	if _, err := Inspect(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrVersion) {
		t.Fatalf("old inspect: %v", err)
	}
	rawStore, err := connect(t.Context(), path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer rawStore.Close()
	var after int64
	if err := rawStore.conn.QueryRowContext(t.Context(), "SELECT epoch FROM runs").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != epoch {
		t.Fatal("unsupported store activation advanced epoch")
	}
}

func TestStoreRejectsMalformedNamespaceAndKeyVersion(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		want      error
	}{
		{"missing", "UPDATE metadata SET namespace=''", ErrCorrupt},
		{"short", "UPDATE metadata SET namespace='ab'", ErrCorrupt},
		{"nonhex", "UPDATE metadata SET namespace='" + strings.Repeat("z", 64) + "'", ErrCorrupt},
		{"uppercase", "UPDATE metadata SET namespace='" + strings.Repeat("A", 64) + "'", ErrCorrupt},
		{"key-version", "UPDATE metadata SET key_version=1", ErrVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path := newStore(t, false)
			if _, err := s.conn.ExecContext(t.Context(), tc.sql); err != nil {
				t.Fatal(err)
			}
			s.Close()
			reopened, err := Open(t.Context(), path, DefaultLimits())
			if reopened != nil {
				reopened.Close()
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("Open: %v", err)
			}
			if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, tc.want) {
				t.Fatalf("Replay: %v", err)
			}
			if _, err := Inspect(t.Context(), path, DefaultLimits()); !errors.Is(err, tc.want) {
				t.Fatalf("Inspect: %v", err)
			}
		})
	}
}

func TestStoreNamespaceBindsRetainedExternalAttempt(t *testing.T) {
	s, path := messageStore(t, externalProvision(t, "http://127.0.0.1:1", "idempotent"))
	state, err := s.Commit(t.Context(), current(t, s, "client").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.prepareExternal(t.Context(), state.Projection.Data["pending"].Text); err != nil {
		t.Fatal(err)
	}
	changed := strings.Repeat("a", 64)
	if s.Namespace() == changed {
		changed = strings.Repeat("b", 64)
	}
	if _, err := s.conn.ExecContext(t.Context(), "UPDATE metadata SET namespace=?", changed); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err := Replay(t.Context(), path, DefaultLimits()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("unbound retained request: %v", err)
	}
	reopened, err := Open(t.Context(), path, DefaultLimits())
	if reopened != nil {
		reopened.Close()
	}
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("activated changed identity: %v", err)
	}
}

func TestExternalKeyCanonicalEncoding(t *testing.T) {
	const namespace = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	const intent = `["client",1,1,0]`
	const want = `[2,"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","[\"client\",1,1,0]"]`
	if got := ExternalKey(namespace, intent); got != want {
		t.Fatalf("ambiguous or unversioned encoding: %s", got)
	}
}
