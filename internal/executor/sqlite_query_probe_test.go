package executor

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

type queryReads struct {
	args []driver.NamedValue
	seen map[[3]string]bool
}

type sqliteQueryProbe struct {
	*sqlite3.SQLiteConn
	observe    func(table, column string, args []driver.NamedValue)
	active     *queryReads
	incomplete error
}

func (p *sqliteQueryProbe) authorize(op int, table, column, database string) int {
	if op != sqlite3.SQLITE_READ {
		return sqlite3.SQLITE_OK
	}
	q := p.active
	if q == nil {
		if p.incomplete == nil {
			p.incomplete = fmt.Errorf("unobserved SQLite read of %s.%s", table, column)
		}
		return sqlite3.SQLITE_OK
	}
	key := [3]string{database, table, column}
	if !q.seen[key] {
		q.seen[key] = true
		p.observe(table, column, q.args)
	}
	return sqlite3.SQLITE_OK
}

func (p *sqliteQueryProbe) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	// The measured queries contain a single statement. An internal semicolon
	// may introduce another VM in this call, so report an incomplete count.
	if strings.Contains(strings.TrimSuffix(strings.TrimSpace(query), ";"), ";") && p.incomplete == nil {
		p.incomplete = errors.New("cannot count SQLite queries containing an internal semicolon")
	}
	reads := &queryReads{args: slices.Clone(args), seen: make(map[[3]string]bool)}
	p.active = reads
	defer func() { p.active = nil }()
	rows, err := p.SQLiteConn.QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return &sqliteProbeRows{SQLiteRows: rows.(*sqlite3.SQLiteRows), probe: p, reads: reads}, nil
}

type sqliteProbeRows struct {
	*sqlite3.SQLiteRows
	probe *sqliteQueryProbe
	reads *queryReads
}

func (r *sqliteProbeRows) Next(values []driver.Value) error {
	// SQLite may reprepare at the first step. Associate that authorization
	// with this result even if another query has run on the connection.
	r.probe.active = r.reads
	defer func() { r.probe.active = nil }()
	return r.SQLiteRows.Next(values)
}

func (p *sqliteQueryProbe) Connect(context.Context) (driver.Conn, error) { return p, nil }
func (p *sqliteQueryProbe) Driver() driver.Driver                        { return &sqlite3.SQLiteDriver{} }
func (p *sqliteQueryProbe) Close() error                                 { return nil } // The owner closes the borrowed driver connection.

// observeQueryReads reports each column once per database/sql query, including
// its actual bound arguments. Repeated queries with identical arguments count
// separately. The qualifying tests perform synchronous reads on one owner.
func observeQueryReads(t *testing.T, s *Store, observe func(string, string, []driver.NamedValue), operation func()) {
	t.Helper()
	if err := collectQueryReads(t.Context(), s, observe, operation); err != nil {
		t.Fatal(err)
	}
}

func collectQueryReads(ctx context.Context, s *Store, observe func(string, string, []driver.NamedValue), operation func()) error {
	original := s.conn
	return original.Raw(func(raw any) error {
		probe := &sqliteQueryProbe{SQLiteConn: raw.(*sqlite3.SQLiteConn), observe: observe}
		probe.RegisterAuthorizer(probe.authorize)
		defer probe.RegisterAuthorizer(nil)
		db := sql.OpenDB(probe)
		defer db.Close()
		db.SetMaxOpenConns(1)
		pinned, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		defer pinned.Close()
		s.conn = pinned
		defer func() { s.conn = original }()
		// Preserve the guard's original connection identity relationship.
		// Both handles refer to this same borrowed SQLite connection.
		if cache := s.execution; cache != nil && cache.conn == original {
			cache.conn = pinned
			defer func() { cache.conn = original }()
		}
		operation()
		return probe.incomplete
	})
}

func TestQueryReadProbeDistinguishesRepeatedLookups(t *testing.T) {
	s, _ := messageStore(t, p3WaitingProvision(t, 2, 1))
	var visited []string
	observeQueryReads(t, s, func(table, column string, args []driver.NamedValue) {
		if table == "runs" && column == "projection" {
			visited = append(visited, args[0].Value.(string))
		}
	}, func() {
		for _, subject := range []string{"r00", "r00", "r01"} {
			var first, second []byte
			err := s.conn.QueryRowContext(t.Context(),
				"SELECT r.projection, r.projection FROM runs r JOIN subjects s ON s.id=r.subject WHERE r.subject=? AND r.generation=1", subject).Scan(&first, &second)
			if err != nil || len(first) == 0 || !bytes.Equal(first, second) {
				t.Fatal("projection lookup", subject, err)
			}
		}
	})
	if !slices.Equal(visited, []string{"r00", "r00", "r01"}) {
		t.Fatalf("expected one visit per lookup, including the repeated r00 lookup: %v", visited)
	}
}

// Queries outside the supported observation path must never silently disappear
// from a work count. Execute each shape against a populated disposable Store.
func TestQueryReadProbeRejectsUnmeasuredWork(t *testing.T) {
	const query = "SELECT projection FROM runs LIMIT 1"
	for _, shape := range []string{"prepared", "exec", "batch"} {
		t.Run(shape, func(t *testing.T) {
			s, _ := messageStore(t, p3WaitingProvision(t, 1, 1))
			var callErr error
			err := collectQueryReads(t.Context(), s, func(string, string, []driver.NamedValue) {}, func() {
				var data []byte
				switch shape {
				case "prepared":
					var stmt *sql.Stmt
					stmt, callErr = s.conn.PrepareContext(t.Context(), query)
					if callErr == nil {
						callErr = stmt.QueryRowContext(t.Context()).Scan(&data)
						callErr = errors.Join(callErr, stmt.Close())
					}
				case "exec":
					_, callErr = s.conn.ExecContext(t.Context(), query)
				case "batch":
					callErr = s.conn.QueryRowContext(t.Context(), query+"; "+query).Scan(&data)
				}
				if callErr == nil && shape != "exec" && len(data) == 0 {
					t.Fatal("missing queried projection")
				}
			})
			if callErr != nil || err == nil {
				t.Fatalf("SQL call error %v, expected incomplete observation, got %v", callErr, err)
			}
		})
	}
}
