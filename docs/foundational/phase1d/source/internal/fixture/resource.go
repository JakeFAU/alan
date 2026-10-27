// Package fixture implements a deliberately small independent external resource.
// It knows no executor projections, lifecycle rules, or retry budget.
package fixture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

var ErrConflict = errors.New("resource parameter conflict")

type Request struct {
	Key, Profile                string
	Subject, Authorization      string
	Generation, Epoch, Sequence int64
	Delta                       int64
}
type Receipt struct {
	Key, Profile           string
	Delta, Mutation, Total int64
	Process                string
	PID                    int
}
type Snapshot struct {
	ForeignKeys, BusyTimeoutMS                 int
	SQLiteVersion, SQLiteSourceID, JournalMode string
	Synchronous, QueryOnly                     int
	Requests                                   []Request
	Applications                               []Receipt
	Total, Contacts                            int64
}
type Resource struct {
	mu      sync.Mutex
	db      *sql.DB
	conn    *sql.Conn
	lock    *os.File
	process string
}

func Open(ctx context.Context, path, process string) (*Resource, error) {
	if process == "" {
		return nil, fmt.Errorf("process identity required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return nil, err
	}
	abs = filepath.Join(parent, filepath.Base(abs))
	if info, e := os.Lstat(abs); e == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.Sys().(*syscall.Stat_t).Nlink != 1 {
			return nil, fmt.Errorf("resource alias prohibited")
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	lock, err := os.OpenFile(abs+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, err
	}
	r := &Resource{lock: lock, process: process}
	r.db, err = sql.Open("sqlite3", (&url.URL{Scheme: "file", Path: abs}).String()+"?_journal_mode=WAL&_synchronous=FULL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		r.Close()
		return nil, err
	}
	r.db.SetMaxOpenConns(1)
	r.conn, err = r.db.Conn(ctx)
	if err != nil {
		r.Close()
		return nil, err
	}
	if _, err = verifySettings(ctx, r.conn, false); err != nil {
		r.Close()
		return nil, err
	}

	_, err = r.conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metadata(version INTEGER NOT NULL); INSERT INTO metadata SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM metadata); CREATE TABLE IF NOT EXISTS keys(key TEXT PRIMARY KEY,profile TEXT NOT NULL,delta INTEGER NOT NULL,descriptor BLOB NOT NULL); CREATE TABLE IF NOT EXISTS applications(number INTEGER PRIMARY KEY,receipt BLOB NOT NULL); CREATE TABLE IF NOT EXISTS contacts(number INTEGER PRIMARY KEY,request BLOB NOT NULL);`)
	if err == nil {
		var version int
		err = r.conn.QueryRowContext(ctx, "SELECT version FROM metadata").Scan(&version)
		if version != 1 {
			err = fmt.Errorf("unsupported resource version")
		}
	}
	if err != nil {
		r.Close()
		return nil, err
	}
	return r, nil
}
func (r *Resource) Close() error {
	var err error
	if r.conn != nil {
		err = errors.Join(err, r.conn.Close())
	}
	if r.db != nil {
		err = errors.Join(err, r.db.Close())
	}
	if r.lock != nil {
		err = errors.Join(err, r.lock.Close())
	}
	return err
}
func (r *Resource) Apply(ctx context.Context, request Request) (Receipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if request.Key == "" || len(request.Key) > 1<<20 || (request.Profile != "idempotent" && request.Profile != "non-idempotent") {
		return Receipt{}, fmt.Errorf("invalid request")
	}
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback()
	var profile string
	var descriptor []byte
	var delta int64
	err = tx.QueryRowContext(ctx, "SELECT profile,delta,descriptor FROM keys WHERE key=?", request.Key).Scan(&profile, &delta, &descriptor)
	if err == nil {
		expected, _ := json.Marshal(request)
		if profile != request.Profile || delta != request.Delta || string(descriptor) != string(expected) {
			return Receipt{}, ErrConflict
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err = tx.ExecContext(ctx, "INSERT INTO keys VALUES(?,?,?,?)", request.Key, request.Profile, request.Delta, mustJSON(request)); err != nil {
			return Receipt{}, err
		}
	} else {
		return Receipt{}, err
	}
	raw, _ := json.Marshal(request)
	if _, err = tx.ExecContext(ctx, "INSERT INTO contacts(request) VALUES(?)", raw); err != nil {
		return Receipt{}, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT receipt FROM applications ORDER BY number")
	if err != nil {
		return Receipt{}, err
	}
	var prior *Receipt
	var last Receipt
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			break
		}
		if err = json.Unmarshal(b, &last); err != nil {
			break
		}
		if last.Key == request.Key && prior == nil {
			copy := last
			prior = &copy
		}
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return Receipt{}, err
	}
	if request.Profile == "idempotent" && prior != nil {
		return *prior, tx.Commit()
	}
	if (request.Delta > 0 && last.Total > math.MaxInt64-request.Delta) || (request.Delta < 0 && last.Total < math.MinInt64-request.Delta) || last.Mutation == math.MaxInt64 {
		return Receipt{}, fmt.Errorf("resource capacity")
	}
	result := Receipt{Key: request.Key, Profile: request.Profile, Delta: request.Delta, Mutation: last.Mutation + 1, Total: last.Total + request.Delta, Process: r.process, PID: os.Getpid()}
	raw, _ = json.Marshal(result)
	if _, err = tx.ExecContext(ctx, "INSERT INTO applications VALUES(?,?)", result.Mutation, raw); err != nil {
		return Receipt{}, err
	}
	return result, tx.Commit()
}
func Inspect(ctx context.Context, path string) (Snapshot, error) {
	snapshot := Snapshot{Applications: []Receipt{}}
	abs, err := filepath.Abs(path)
	if err != nil {
		return snapshot, err
	}
	db, err := sql.Open("sqlite3", (&url.URL{Scheme: "file", Path: abs}).String()+"?mode=ro&_query_only=on&_foreign_keys=on&_synchronous=FULL&_busy_timeout=5000")
	if err != nil {
		return snapshot, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return snapshot, err
	}
	defer tx.Rollback()
	var version, count int
	if err = tx.QueryRowContext(ctx, "SELECT version,(SELECT count(*) FROM metadata) FROM metadata").Scan(&version, &count); err != nil {
		return snapshot, err
	}
	if version != 1 || count != 1 {
		return snapshot, fmt.Errorf("unsupported resource version")
	}
	settings, err := verifySettings(ctx, tx, true)
	if err != nil {
		return snapshot, err
	}
	snapshot = settings

	var integrity string
	if err = tx.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return snapshot, err
	}
	if integrity != "ok" {
		return snapshot, fmt.Errorf("corrupt resource: %s", integrity)
	}
	rows, err := tx.QueryContext(ctx, "SELECT receipt FROM applications ORDER BY number")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var b []byte
		var receipt Receipt
		if err = rows.Scan(&b); err == nil {
			err = json.Unmarshal(b, &receipt)
		}
		if err != nil {
			rows.Close()
			return snapshot, err
		}
		snapshot.Applications = append(snapshot.Applications, receipt)
		snapshot.Total = receipt.Total
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return snapshot, err
	}
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM contacts").Scan(&snapshot.Contacts); err != nil {
		return snapshot, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT request FROM contacts ORDER BY number")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var raw []byte
		var request Request
		if err = rows.Scan(&raw); err == nil {
			err = json.Unmarshal(raw, &request)
		}
		if err != nil {
			rows.Close()
			return snapshot, err
		}
		snapshot.Requests = append(snapshot.Requests, request)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return snapshot, err
	}
	return snapshot, tx.Commit()
}

func mustJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }

type settingsQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func verifySettings(ctx context.Context, q settingsQuery, readonly bool) (Snapshot, error) {
	var s Snapshot
	for _, v := range []struct {
		sql string
		dst any
	}{{"PRAGMA journal_mode", &s.JournalMode}, {"PRAGMA synchronous", &s.Synchronous}, {"PRAGMA foreign_keys", &s.ForeignKeys}, {"PRAGMA busy_timeout", &s.BusyTimeoutMS}, {"PRAGMA query_only", &s.QueryOnly}} {
		if err := q.QueryRowContext(ctx, v.sql).Scan(v.dst); err != nil {
			return s, err
		}
	}
	expected := 0
	if readonly {
		expected = 1
	}
	if s.JournalMode != "wal" || s.Synchronous != 2 || s.ForeignKeys != 1 || s.BusyTimeoutMS != 5000 || s.QueryOnly != expected {
		return s, fmt.Errorf("unsafe resource SQLite settings: %+v", s)
	}
	if err := q.QueryRowContext(ctx, "SELECT sqlite_version(),sqlite_source_id()").Scan(&s.SQLiteVersion, &s.SQLiteSourceID); err != nil {
		return s, err
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(s.SQLiteVersion, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return s, err
	}
	if major < 3 || major == 3 && (minor < 51 || minor == 51 && patch < 3) {
		return s, fmt.Errorf("unsupported resource engine %s", s.SQLiteVersion)
	}
	s.Applications = []Receipt{}
	return s, nil
}
