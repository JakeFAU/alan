package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

var ErrConflict = errors.New("identity conflict")
var ErrStale = errors.New("stale authority")
var ErrLocked = errors.New("store already active")

type Artifact struct {
	ID, Semantics string
	Bytes         []byte
}
type Subject struct {
	ID                      string
	Generation              int64
	Owner                   string
	Revision                int64
	Retired, RetryCancelled bool
}
type InitialRun struct {
	ID         RunID
	ProgramID  string
	Projection Projection
}
type Permission struct {
	Kind, Source, Target, Operation string
	Revision                        int64
	Obligation                      string
}
type Provision struct {
	Programs    []Artifact
	Subjects    []Subject
	Runs        []InitialRun
	Permissions []Permission
}
type Authority struct {
	Run                       RunID
	Epoch, Sequence, Revision int64
}
type RunState struct {
	ID                        RunID
	ProgramID                 string
	Epoch, Sequence, Revision int64
	Projection                Projection
}

func (r RunState) Authority() Authority { return Authority{r.ID, r.Epoch, r.Sequence, r.Revision} }

const SchemaVersion = 2
const ExternalKeyVersion = 2
const busyTimeoutMS = 5000

// Store owns one pinned SQLite connection and an OS lock. It is safe for concurrent
// callers; serialized transactions preserve the single-owner execution model.
type Store struct {
	ownerMu   sync.Mutex
	runDone   chan struct{}
	runWake   chan struct{}
	runCancel context.CancelFunc
	closing   bool
	mu        sync.Mutex
	db        *sql.DB
	conn      *sql.Conn
	lock      *os.File
	limits    Limits
	closed    bool
	namespace string
}

// Every protocol ledger is retained. Later protocol slices supply their operations;
// these tables do not grant authority merely by containing an opaque descriptor.
const schema = `
CREATE TABLE metadata (singleton INTEGER PRIMARY KEY CHECK(singleton=1), schema_version INTEGER NOT NULL, semantics TEXT NOT NULL, key_version INTEGER NOT NULL, namespace TEXT NOT NULL);
CREATE TABLE programs (id TEXT PRIMARY KEY, semantics TEXT NOT NULL, bytes BLOB NOT NULL);
CREATE TABLE subjects (id TEXT PRIMARY KEY, generation INTEGER NOT NULL CHECK(generation>0), owner TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision>0), retired INTEGER NOT NULL CHECK(retired IN (0,1)), retry_cancelled INTEGER NOT NULL CHECK(retry_cancelled IN (0,1)));
CREATE TABLE permissions (kind TEXT NOT NULL, source TEXT NOT NULL, target TEXT NOT NULL, operation TEXT NOT NULL, revision INTEGER NOT NULL, obligation TEXT NOT NULL, PRIMARY KEY(kind,source,target,operation,revision,obligation));
CREATE TABLE runs (subject TEXT NOT NULL REFERENCES subjects(id), generation INTEGER NOT NULL CHECK(generation>0), program_id TEXT NOT NULL REFERENCES programs(id), initial BLOB NOT NULL, epoch INTEGER NOT NULL CHECK(epoch>0), sequence INTEGER NOT NULL CHECK(sequence>=0), projection BLOB NOT NULL, PRIMARY KEY(subject,generation));
CREATE TABLE history (subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL CHECK(sequence>0), epoch INTEGER NOT NULL CHECK(epoch>0), source_revision INTEGER NOT NULL CHECK(source_revision>0), selected_observation TEXT REFERENCES observations(id), successor BLOB NOT NULL, intents BLOB NOT NULL, PRIMARY KEY(subject,generation,sequence), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE observations (id TEXT PRIMARY KEY, subject TEXT NOT NULL, generation INTEGER NOT NULL, descriptor BLOB NOT NULL, consumed_sequence INTEGER, FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation), FOREIGN KEY(subject,generation,consumed_sequence) REFERENCES history(subject,generation,sequence) DEFERRABLE INITIALLY DEFERRED);
CREATE TABLE intents (id TEXT PRIMARY KEY, subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL, ordinal INTEGER NOT NULL CHECK(ordinal>=0), source_revision INTEGER NOT NULL CHECK(source_revision>0), descriptor BLOB NOT NULL, UNIQUE(subject,generation,sequence,ordinal), FOREIGN KEY(subject,generation,sequence) REFERENCES history(subject,generation,sequence));
CREATE TABLE terminal_obligations (subject TEXT NOT NULL, generation INTEGER NOT NULL, sequence INTEGER NOT NULL, outcome BLOB NOT NULL, destinations BLOB NOT NULL, delivery_status BLOB NOT NULL, operation_id TEXT REFERENCES lifecycle_results(id) DEFERRABLE INITIALLY DEFERRED, source_revision INTEGER NOT NULL CHECK(source_revision>0), PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE admissions (id TEXT PRIMARY KEY, domain TEXT NOT NULL CHECK(domain IN ('input','message','terminal','lifecycle','receipt')), transport_id TEXT NOT NULL, descriptor BLOB NOT NULL, result BLOB NOT NULL, observation_id TEXT REFERENCES observations(id), UNIQUE(domain,transport_id));
CREATE TABLE message_acknowledgments (intent_id TEXT PRIMARY KEY REFERENCES intents(id), admission_id TEXT NOT NULL REFERENCES admissions(id));
CREATE TABLE attempts (id TEXT PRIMARY KEY, intent_id TEXT NOT NULL REFERENCES intents(id), descriptor BLOB NOT NULL, result BLOB NOT NULL);
CREATE TABLE lifecycle_results (id TEXT PRIMARY KEY, decision_sequence INTEGER NOT NULL UNIQUE CHECK(decision_sequence>0), descriptor BLOB NOT NULL, result BLOB NOT NULL);
CREATE TABLE run_origins (subject TEXT NOT NULL, generation INTEGER NOT NULL, kind TEXT NOT NULL CHECK(kind IN ('root','retry')), operation_id TEXT REFERENCES lifecycle_results(id) DEFERRABLE INITIALLY DEFERRED, predecessor_generation INTEGER, descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE UNIQUE INDEX one_static_root ON run_origins(subject) WHERE kind='root';
CREATE TABLE external_attempts (intent_id TEXT PRIMARY KEY REFERENCES intents(id), descriptor BLOB NOT NULL);
CREATE TABLE external_receipts (intent_id TEXT PRIMARY KEY REFERENCES external_attempts(intent_id), descriptor BLOB NOT NULL);
CREATE TABLE settlements (id TEXT PRIMARY KEY, intent_id TEXT NOT NULL REFERENCES intents(id), descriptor BLOB NOT NULL, observation_id TEXT REFERENCES observations(id));
`

func canonicalPath(path string, creating bool) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !creating || !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		// A dangling symlink must not silently select a different lock identity.
		if _, e := os.Lstat(abs); e == nil {
			return "", fmt.Errorf("dangling database alias")
		}
		parent, e := filepath.EvalSymlinks(filepath.Dir(abs))
		if e != nil {
			return "", e
		}
		real = filepath.Join(parent, filepath.Base(abs))
	}
	if info, e := os.Stat(real); e == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("database is not a regular file")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Nlink != 1 {
			return "", fmt.Errorf("unsupported multiply linked database")
		}
	} else if !creating || !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	return real, nil
}

func acquire(path string) (*os.File, error) {
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || stat.Nlink != 1 {
		return nil, errors.Join(fmt.Errorf("unsafe lock file"), f.Close())
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeErr := f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.Join(ErrLocked, closeErr)
		}
		return nil, errors.Join(err, closeErr)
	}
	return f, nil
}

func connect(ctx context.Context, path string, readonly bool) (*Store, error) {
	q := url.Values{"_foreign_keys": {"on"}, "_synchronous": {"FULL"}, "_busy_timeout": {"5000"}}
	if readonly {
		q.Set("mode", "ro")
		q.Set("_query_only", "1")
	} else {
		q.Set("mode", "rw")
		q.Set("_journal_mode", "WAL")
	}
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: q.Encode()}).String()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	s := &Store{db: db, conn: conn}
	if _, _, err = verifyConnection(ctx, conn, readonly); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}

// The store owns rollback while holding mu. database/sql's asynchronous rollback
// on context cancellation discards this driver's pinned connection. Detach only
// the transaction lifetime; each statement and the commit gate retain ctx.
func beginOwnedTx(ctx context.Context, conn *sql.Conn) (*sql.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return conn.BeginTx(context.WithoutCancel(ctx), nil)
}
func (s *Store) commitOwnedTx(ctx context.Context, tx *sql.Tx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.ownerMu.Lock()
	if s.runWake != nil {
		select {
		case s.runWake <- struct{}{}:
		default:
		}
	}
	s.ownerMu.Unlock()
	return nil
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func verifyConnection(ctx context.Context, q queryer, readonly bool) (string, string, error) {
	var mode, version, source string
	var syncMode, fk, busy, queryOnly int
	if err := q.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		return "", "", err
	}
	for _, v := range []struct {
		sql string
		dst *int
	}{{"PRAGMA synchronous", &syncMode}, {"PRAGMA foreign_keys", &fk}, {"PRAGMA busy_timeout", &busy}, {"PRAGMA query_only", &queryOnly}} {
		if err := q.QueryRowContext(ctx, v.sql).Scan(v.dst); err != nil {
			return "", "", err
		}
	}
	expectedQueryOnly := 0
	if readonly {
		expectedQueryOnly = 1
	}
	if strings.ToLower(mode) != "wal" || syncMode != 2 || fk != 1 || busy != busyTimeoutMS || queryOnly != expectedQueryOnly {
		return "", "", fmt.Errorf("unsafe SQLite settings: journal=%s synchronous=%d foreign_keys=%d busy_timeout=%d query_only=%d", mode, syncMode, fk, busy, queryOnly)
	}
	if err := q.QueryRowContext(ctx, "SELECT sqlite_version(),sqlite_source_id()").Scan(&version, &source); err != nil {
		return "", "", err
	}
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return "", "", err
	}
	// The supported bundled engine must include the WAL-reset fix (3.51.3+).
	if major < 3 || major == 3 && (minor < 51 || minor == 51 && patch < 3) {
		return "", "", fmt.Errorf("unsupported SQLite engine %s", version)
	}
	return version, source, nil
}

// Create provisions a new store in one transaction; it never re-provisions an
// existing file. Open is the only recovery path.
func Create(ctx context.Context, path string, p Provision, limits Limits) (store *Store, err error) {
	real, err := canonicalPath(path, true)
	if err != nil {
		return nil, err
	}
	lock, err := acquire(real)
	if err != nil {
		return nil, err
	}
	defer func() {
		if store == nil {
			err = errors.Join(err, lock.Close())
		}
	}()
	file, err := os.OpenFile(real, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	s, err := connect(ctx, real, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		if store == nil {
			err = errors.Join(err, s.Close())
		}
	}()
	s.limits = limits
	var namespaceBytes [32]byte
	if _, err = rand.Read(namespaceBytes[:]); err != nil {
		return nil, err
	}
	s.namespace = hex.EncodeToString(namespaceBytes[:])
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO metadata VALUES(1,?,?,?,?)", SchemaVersion, SemanticVersion, ExternalKeyVersion, s.namespace); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "PRAGMA user_version=2"); err != nil {
		return nil, err
	}
	for _, a := range p.Programs {
		if err = insertArtifact(ctx, tx, a, limits); err != nil {
			return nil, err
		}
	}
	for _, v := range p.Subjects {
		if v.ID == "" || v.Owner == "" || v.Generation <= 0 || v.Revision <= 0 || v.Retired {
			return nil, fmt.Errorf("invalid initial subject")
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO subjects VALUES(?,?,?,?,?,?)", v.ID, v.Generation, v.Owner, v.Revision, v.Retired, v.RetryCancelled); err != nil {
			return nil, err
		}
	}
	for _, v := range p.Permissions {
		if v.Kind != "lifecycle" && v.Kind != "observation" && v.Kind != "message" && v.Kind != "input" && v.Kind != "external" && v.Kind != "adapter" {
			return nil, fmt.Errorf("invalid permission kind")
		}
		if v.Kind == "input" {
			if v.Source != "local" || v.Operation != "input" || v.Revision != 0 {
				return nil, fmt.Errorf("invalid local input permission")
			}
		} else {
			var exists int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM subjects WHERE id=?", v.Source).Scan(&exists); err != nil {
				return nil, err
			}
			if exists != 1 || v.Revision <= 0 {
				return nil, fmt.Errorf("invalid permission source")
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO permissions VALUES(?,?,?,?,?,?)", v.Kind, v.Source, v.Target, v.Operation, v.Revision, v.Obligation); err != nil {
			return nil, err
		}
	}
	for _, v := range p.Runs {
		program, e := loadProgram(ctx, tx, v.ProgramID, limits)
		if e != nil {
			return nil, e
		}
		if err = validateAuthorInitial(program, v.Projection); err != nil {
			return nil, err
		}
		if err = validateProjection(v.Projection, program, limits); err != nil {
			return nil, err
		}
		if v.Projection.Status != "open" {
			return nil, fmt.Errorf("initial run must be open")
		}
		b, e := json.Marshal(v.Projection)
		if e != nil {
			return nil, e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO runs VALUES(?,?,?,?,1,0,?)", v.ID.Subject, v.ID.Generation, v.ProgramID, b, b); err != nil {
			return nil, err
		}
		subject, e := loadSubject(ctx, tx, v.ID.Subject)
		if e != nil {
			return nil, e
		}
		if subject.Generation != v.ID.Generation {
			return nil, fmt.Errorf("static run must match initial subject generation")
		}
		origin, e := json.Marshal(rootOrigin{Subject: subject, ProgramID: v.ProgramID, Initial: v.Projection})
		if e != nil {
			return nil, e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO run_origins VALUES(?,?,'root',NULL,NULL,?)", v.ID.Subject, v.ID.Generation, origin); err != nil {
			return nil, err
		}
	}
	if _, err = validateLedger(ctx, tx, limits); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.lock = lock
	return s, nil
}

func insertArtifact(ctx context.Context, tx *sql.Tx, a Artifact, l Limits) error {
	if !supportedSemantics(a.Semantics) {
		return ErrVersion
	}
	if a.ID != programHash(a.Bytes) {
		return fmt.Errorf("artifact hash: %w", ErrCorrupt)
	}
	if len(a.Bytes) > l.MaxProgramBytes || l.MaxProgramBytes <= 0 {
		return ErrLimit
	}
	var p Program
	if err := decode(a.Bytes, &p); err != nil {
		return err
	}
	if p.Semantics != a.Semantics {
		return ErrVersion
	}
	if err := validateProgram(p); err != nil {
		return err
	}
	var sem string
	var existing []byte
	err := tx.QueryRowContext(ctx, "SELECT semantics,bytes FROM programs WHERE id=?", a.ID).Scan(&sem, &existing)
	if err == nil {
		if sem != a.Semantics || !bytes.Equal(existing, a.Bytes) {
			return ErrConflict
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO programs VALUES(?,?,?)", a.ID, a.Semantics, a.Bytes)
	return err
}
func loadProgram(ctx context.Context, q queryer, id string, l Limits) (Program, error) {
	var a Artifact
	a.ID = id
	if err := q.QueryRowContext(ctx, "SELECT semantics,bytes FROM programs WHERE id=?", id).Scan(&a.Semantics, &a.Bytes); err != nil {
		return Program{}, err
	}
	if !supportedSemantics(a.Semantics) {
		return Program{}, ErrVersion
	}
	if a.ID != programHash(a.Bytes) {
		return Program{}, fmt.Errorf("artifact hash: %w", ErrCorrupt)
	}
	if len(a.Bytes) > l.MaxProgramBytes || l.MaxProgramBytes <= 0 {
		return Program{}, ErrLimit
	}
	var p Program
	if err := decode(a.Bytes, &p); err != nil {
		return Program{}, err
	}
	if p.Semantics != a.Semantics {
		return Program{}, ErrVersion
	}
	return p, validateProgram(p)
}

// Open validates the full retained ledger before advancing epochs. Epoch changes
// preserve every logical projection, pending identity and observation consumption.
func Open(ctx context.Context, path string, limits Limits) (store *Store, err error) {
	real, err := canonicalPath(path, false)
	if err != nil {
		return nil, err
	}
	lock, err := acquire(real)
	if err != nil {
		return nil, err
	}
	defer func() {
		if store == nil {
			err = errors.Join(err, lock.Close())
		}
	}()
	s, err := connect(ctx, real, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		if store == nil {
			err = errors.Join(err, s.Close())
		}
	}()
	s.limits = limits
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	report, err := validateLedger(ctx, tx, limits)
	if err != nil {
		return nil, err
	}
	s.namespace = report.Namespace
	for _, r := range report.Runs {
		if r.Projection.Status != "open" {
			continue
		}
		if r.Epoch == math.MaxInt64 {
			return nil, ErrLimit
		}
		if _, err = tx.ExecContext(ctx, "UPDATE runs SET epoch=epoch+1 WHERE subject=? AND generation=?", r.ID.Subject, r.ID.Generation); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.lock = lock
	return s, nil
}
func (s *Store) Close() error {
	s.ownerMu.Lock()
	s.closing = true
	done := s.runDone
	if s.runCancel != nil {
		s.runCancel()
	}
	s.ownerMu.Unlock()
	if done != nil {
		<-done
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	var errs []error
	if s.conn != nil {
		errs = append(errs, s.conn.Close())
	}
	if s.db != nil {
		errs = append(errs, s.db.Close())
	}
	// Never unlink the lock file: waiters must continue to refer to one inode.
	if s.lock != nil {
		errs = append(errs, s.lock.Close())
	}
	return errors.Join(errs...)
}
func loadRun(ctx context.Context, q queryer, id RunID) (RunState, error) {
	r := RunState{ID: id}
	var data []byte
	err := q.QueryRowContext(ctx, `SELECT r.program_id,r.epoch,r.sequence,s.revision,r.projection FROM runs r JOIN subjects s ON s.id=r.subject WHERE r.subject=? AND r.generation=?`, id.Subject, id.Generation).Scan(&r.ProgramID, &r.Epoch, &r.Sequence, &r.Revision, &data)
	if err != nil {
		return RunState{}, err
	}
	if err = decode(data, &r.Projection); err != nil {
		return RunState{}, err
	}
	return r, nil
}
func (s *Store) Snapshot(ctx context.Context, id RunID) (RunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return RunState{}, sql.ErrConnDone
	}
	return loadRun(ctx, s.conn, id)
}

// recordObservation is a trusted runtime admission primitive, not a transport API.
// Its caller must establish provenance and original binding. Later message and
// settlement protocols perform that authentication before inserting observations.
func (s *Store) recordObservation(ctx context.Context, o Observation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = insertObservation(ctx, tx, o, s.limits); err != nil {
		return err
	}
	return s.commitOwnedTx(ctx, tx)
}

func insertObservation(ctx context.Context, tx *sql.Tx, o Observation, limits Limits) error {
	if o.ID == "" || o.Run.Subject == "" || o.Run.Generation <= 0 || o.Kind == "" || o.Provenance == "" || o.Source == "" || o.SourceGeneration <= 0 || o.Payload == nil {
		return fmt.Errorf("invalid observation")
	}
	for k, v := range o.Payload {
		if k == "" || !validValue(v) {
			return fmt.Errorf("invalid observation payload")
		}
	}
	if err := checkSize(o, limits.MaxObservationBytes); err != nil {
		return err
	}
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}
	var old []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM observations WHERE id=?", o.ID).Scan(&old)
	if err == nil {
		if !bytes.Equal(old, b) {
			return ErrConflict
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO observations(id,subject,generation,descriptor) VALUES(?,?,?,?)", o.ID, o.Run.Subject, o.Run.Generation, b); err != nil {
		return err
	}
	return nil
}

// Commit selects at most one retained observation and evaluates inside the same
// transaction as consumption, history, successor, intents and terminal retention.
// No speculative result can dispatch through this API.
func (s *Store) Commit(ctx context.Context, a Authority, selected string) (RunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return RunState{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return RunState{}, err
	}
	defer tx.Rollback()
	r, err := loadRun(ctx, tx, a.Run)
	if errors.Is(err, sql.ErrNoRows) {
		return RunState{}, ErrStale
	}
	if err != nil {
		return RunState{}, err
	}
	var generation int64
	var retired bool
	if err = tx.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", a.Run.Subject).Scan(&generation, &retired); err != nil {
		return RunState{}, err
	}
	if a != r.Authority() || generation != a.Run.Generation || retired {
		return RunState{}, ErrStale
	}
	if r.Projection.Status != "open" {
		return RunState{}, ErrClosed
	}
	if r.Sequence == math.MaxInt64 {
		return RunState{}, ErrLimit
	}
	p, err := loadProgram(ctx, tx, r.ProgramID, s.limits)
	if err != nil {
		return RunState{}, err
	}
	var observation *Observation
	if selected != "" {
		var b []byte
		var consumed sql.NullInt64
		var subject string
		var gen int64
		if err = tx.QueryRowContext(ctx, "SELECT subject,generation,descriptor,consumed_sequence FROM observations WHERE id=?", selected).Scan(&subject, &gen, &b, &consumed); errors.Is(err, sql.ErrNoRows) {
			return RunState{}, ErrIneligible
		}
		if err != nil {
			return RunState{}, err
		}
		if consumed.Valid || subject != a.Run.Subject || gen != a.Run.Generation {
			return RunState{}, ErrIneligible
		}
		observation = &Observation{}
		if err = decode(b, observation); err != nil {
			return RunState{}, err
		}
		if observation.ID != selected || observation.Run != a.Run {
			return RunState{}, ErrCorrupt
		}
	}
	reduction, err := Reduce(p, r.Projection, observation, r.ID, r.Sequence+1, s.limits)
	if err != nil {
		return RunState{}, err
	}
	r.Sequence++
	r.Projection = reduction.Projection
	successor, err := json.Marshal(r.Projection)
	if err != nil {
		return RunState{}, err
	}
	intents, err := json.Marshal(reduction.Intents)
	if err != nil {
		return RunState{}, err
	}
	var selectedArg any
	if selected != "" {
		selectedArg = selected
	}
	if selected != "" {
		if _, err = tx.ExecContext(ctx, "UPDATE observations SET consumed_sequence=? WHERE id=?", r.Sequence, selected); err != nil {
			return RunState{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO history VALUES(?,?,?,?,?,?,?,?)", r.ID.Subject, r.ID.Generation, r.Sequence, r.Epoch, r.Revision, selectedArg, successor, intents); err != nil {
		return RunState{}, err
	}
	for _, intent := range reduction.Intents {
		b, e := json.Marshal(intent)
		if e != nil {
			return RunState{}, e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO intents VALUES(?,?,?,?,?,?,?)", intent.ID, r.ID.Subject, r.ID.Generation, r.Sequence, intent.Ordinal, r.Revision, b); err != nil {
			return RunState{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE runs SET sequence=?,projection=? WHERE subject=? AND generation=?", r.Sequence, successor, r.ID.Subject, r.ID.Generation); err != nil {
		return RunState{}, err
	}
	if r.Projection.Status != "open" {
		destinations, e := terminalDestinations(ctx, tx, r.ID.Subject, r.Revision)
		if e != nil {
			return RunState{}, e
		}
		b, e := json.Marshal(destinations)
		if e != nil {
			return RunState{}, e
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO terminal_obligations VALUES(?,?,?,?,?,?,NULL,?)", r.ID.Subject, r.ID.Generation, r.Sequence, successor, b, []byte("{}"), r.Revision); err != nil {
			return RunState{}, err
		}
	}
	event := boundary.Event{Boundary: "execution.staged", Phase: "pre-commit", Subject: r.ID.Subject, Generation: r.ID.Generation, Epoch: r.Epoch, Sequence: r.Sequence}
	if len(reduction.Intents) > 0 {
		event.Intent = reduction.Intents[0].ID
	}
	if r.Projection.Status != "open" {
		event.Phase = "terminal"
	}
	if err = boundary.Reached(ctx, event); err != nil {
		return RunState{}, err
	}
	if err = s.commitOwnedTx(ctx, tx); err != nil {
		return RunState{}, err
	}
	event.Boundary = "execution.committed"
	if event.Phase != "terminal" {
		event.Phase = "post-commit"
	}
	if err = boundary.Reached(ctx, event); err != nil {
		// A post-commit notification failure does not undo the durable transition.
		return r, err
	}
	return r, nil
}
func terminalDestinations(ctx context.Context, q queryer, subject string, revision int64) ([]TerminalDestination, error) {
	rows, err := q.QueryContext(ctx, "SELECT DISTINCT target,obligation FROM permissions WHERE kind='observation' AND operation='terminal' AND source=? AND revision=? ORDER BY target,obligation", subject, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	destinations := []TerminalDestination{}
	for rows.Next() {
		var destination TerminalDestination
		if err = rows.Scan(&destination.Target, &destination.Obligation); err != nil {
			return nil, err
		}
		destinations = append(destinations, destination)
	}
	return destinations, rows.Err()
}

// Namespace returns the immutable namespace assigned during provisioning.
// Renames and database copies retain it; independent concurrent use of copies is unsupported.
func (s *Store) Namespace() string { return s.namespace }

// ExternalKey canonically binds an external operation to a provisioned namespace
// and local intent identity. Physical epochs and workers do not change this key.
func ExternalKey(namespace, intentID string) string {
	raw, _ := json.Marshal([]any{ExternalKeyVersion, namespace, intentID})
	return string(raw)
}

// Read version fields first so version-1 metadata is rejected without querying
// columns that did not exist in that schema. No migration or historical rekey occurs.
func readNamespace(ctx context.Context, q queryer) (string, error) {
	var schemaVersion, userVersion int
	var semantics string
	if err := q.QueryRowContext(ctx, "SELECT schema_version,semantics FROM metadata WHERE singleton=1").Scan(&schemaVersion, &semantics); err != nil {
		return "", err
	}
	if err := q.QueryRowContext(ctx, "PRAGMA user_version").Scan(&userVersion); err != nil {
		return "", err
	}
	if schemaVersion != SchemaVersion || userVersion != SchemaVersion || semantics != SemanticVersion {
		return "", ErrVersion
	}
	var namespace string
	var keyVersion int
	if err := q.QueryRowContext(ctx, "SELECT key_version,namespace FROM metadata WHERE singleton=1").Scan(&keyVersion, &namespace); err != nil {
		return "", err
	}
	if keyVersion != ExternalKeyVersion {
		return "", ErrVersion
	}
	decoded, err := hex.DecodeString(namespace)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != namespace {
		return "", corrupt("invalid store namespace")
	}
	return namespace, nil
}
