package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JakeFAU/alan/effects"
)

// ReductionRef identifies one committed reduction within a logical run.
type ReductionRef struct {
	Run      RunID
	Sequence int64
}

// ExecutionStep records a causal transition. Epoch is the owner activation at
// commit, which can differ from the run's current activation after recovery.
type ExecutionStep struct {
	ReductionRef
	Epoch, SourceRevision                      int64
	BeforePC, AfterPC                          int
	Instruction, Status, SelectedObservationID string
	OperationIDs                               []string
}

// HistoryPage is a window in this snapshot's append order. Next is an offset
// for the same snapshot. Slicing diagnostics never limits the causal indexes.
type HistoryPage struct {
	Steps       []ExecutionStep
	Next, Total int
	More        bool
}

const MaxHistoryPage = 1000

// ReceiveWait identifies the exact receive contract at the current instruction.
// Resolved is false when a dynamic operand cannot be read from the retained data.
type ReceiveWait struct {
	Kind, Provenance, Source string
	SourceGeneration         int64
	Obligation, OperationID  string
	Resolved                 bool
	MatchingObservationIDs   []string
}

// InspectionFeatures identifies optional protocols actually present in this
// storage format. A missing older protocol differs from an empty newer journal.
type InspectionFeatures struct{ Composition, ReceiptReads, ConflictJournal bool }

// ExecutionRun describes a logical generation, its current subject authority,
// and its original parentage. InvocationIDs are original invocation operations.
type ExecutionRun struct {
	RunState
	LocalState                                                       RunLocalState
	UnresolvedConsequences                                           []string
	Initial                                                          Projection
	Subject                                                          Subject
	Semantics, Origin, RootID, OwningInvocationID, OriginOperationID string
	Predecessor                                                      *RunID
	Instruction                                                      string
	Wait                                                             *ReceiveWait
	OperationIDs, InvocationIDs, EffectIDs                           []string
}

// PendingObligation is a retained dependency or outstanding local delivery.
// DependsOn is an original operation or observation identity, when applicable.
type PendingObligation struct {
	Kind                                  string
	Run                                   RunID
	OperationID, ObservationID, DependsOn string
	DependsOnRun                          *RunID
	AdmissionIDs                          []string
}

// ExecutionDeployment is the original static root work specification of a
// composed store. Its permissions precede any generated member permissions.
type ExecutionDeployment struct {
	Subjects    []Subject
	Runs        []InitialRun
	Permissions []Permission
}

// ExecutionSnapshot is detached from its database. Treat its caller-owned
// slices and pointed-to values as immutable when sharing it between readers.
type ExecutionSnapshot struct {
	Features           InspectionFeatures
	Namespace          string
	StorageVersion     int
	Runs               []*ExecutionRun
	Invocations        []*InvocationView
	Effects            []*ExecutionEffect
	Observations       []*ObservationView
	Admissions         []*AdmissionView
	LifecycleDecisions []LifecycleResult
	Operations         []Intent
	Roots              []RootCompletion
	Assets             []effects.Asset
	Composition        *CompositionProvision
	Deployment         *ExecutionDeployment
	Permissions        []Permission
	Reconciliation     *ReconciliationProfile
	Pending            []PendingObligation
	byInvocationID     map[string]*InvocationView
	operationIndex     map[string]*Intent
	admissionIndex     map[string]*AdmissionView
	reductionIndex     map[ReductionRef]*ExecutionStep
	rootIndex          map[string]*RootCompletion
	lifecycleIndex     map[string]LifecycleResult
	runIndex           map[RunID]*ExecutionRun
	invocationIndex    map[string]*InvocationView
	effectIndex        map[string]*ExecutionEffect
	observationIndex   map[string]*ObservationView
	assetIndex         map[string]*effects.Asset
	history            []ExecutionStep
}

// Run looks up one original logical generation.
func (s *ExecutionSnapshot) Run(id RunID) (*ExecutionRun, bool) {
	v, ok := s.runIndex[id]
	return v, ok
}

// Invocation looks up the original operation handle returned to the caller.
func (s *ExecutionSnapshot) Invocation(operationID string) (*InvocationView, bool) {
	v, ok := s.invocationIndex[operationID]
	return v, ok
}

// Effect follows the original adapter or external operation handle.
func (s *ExecutionSnapshot) Effect(operationID string) (*ExecutionEffect, bool) {
	v, ok := s.effectIndex[operationID]
	return v, ok
}

// Observation looks up an admitted logical envelope.
func (s *ExecutionSnapshot) Observation(id string) (*ObservationView, bool) {
	v, ok := s.observationIndex[id]
	return v, ok
}

// Asset returns a retained versioned source document.
func (s *ExecutionSnapshot) Asset(id string) (*effects.Asset, bool) {
	v, ok := s.assetIndex[id]
	return v, ok
}

// InvocationByID takes the namespace-bound invocation identity recorded at admission.
func (s *ExecutionSnapshot) InvocationByID(id string) (*InvocationView, bool) {
	v, ok := s.byInvocationID[id]
	return v, ok
}

// Reduction follows an observation's actual selection to a retained commit.
func (s *ExecutionSnapshot) Reduction(ref ReductionRef) (*ExecutionStep, bool) {
	v, ok := s.reductionIndex[ref]
	return v, ok
}

// Operation returns an original commit output.
func (s *ExecutionSnapshot) Operation(id string) (*Intent, bool) {
	v, ok := s.operationIndex[id]
	return v, ok
}

// Admission looks up a single delivery attempt by its domain-bound ID.
func (s *ExecutionSnapshot) Admission(id string) (*AdmissionView, bool) {
	v, ok := s.admissionIndex[id]
	return v, ok
}

// Root returns the structured publication and resource account by resource-root ID.
func (s *ExecutionSnapshot) Root(id string) (*RootCompletion, bool) {
	v, ok := s.rootIndex[id]
	return v, ok
}

// LifecycleDecision returns the original target decision by its requesting
// operation. A missing-target decision has no Before witness even if a later
// invocation created that subject.
func (s *ExecutionSnapshot) LifecycleDecision(operationID string) (LifecycleResult, bool) {
	r, ok := s.lifecycleIndex[operationID]
	return r, ok
}

// History returns up to limit committed steps starting at offset.
func (s *ExecutionSnapshot) History(offset, limit int) (HistoryPage, error) {
	if offset < 0 || offset > len(s.history) || limit < 1 || limit > MaxHistoryPage {
		return HistoryPage{}, fmt.Errorf("history offset or page size: %w", ErrLimit)
	}
	end := offset + min(limit, len(s.history)-offset)
	return HistoryPage{Steps: s.history[offset:end:end], Next: end, Total: len(s.history), More: end < len(s.history)}, nil
}

type inspectionBuilder struct {
	limits         Limits
	tables         map[string][]map[string]any
	snap           *ExecutionSnapshot
	runs           map[RunID]*ExecutionRun
	programs       map[string]Program
	resolved       map[RunID]Program
	intents        map[string]retainedIntent
	steps          map[ReductionRef]*ExecutionStep
	before, after  map[ReductionRef]Projection
	observations   map[string]*ObservationView
	invocations    map[string]*InvocationView
	effects        map[string]*ExecutionEffect
	membership     map[RunID]string
	facts          map[string]compositionFact
	authorizations map[string]Authorization
	lifecycle      map[string]LifecycleResult
}

// InspectExecution reads one coherent SQLite cut without acquiring an owner,
// interpreting reductions, servicing effects, or replaying the computation.
func InspectExecution(ctx context.Context, path string, limits Limits) (snapshot *ExecutionSnapshot, err error) {
	real, err := canonicalPath(path, false)
	if err != nil {
		return nil, err
	}
	store, err := connect(ctx, real, true)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, store.Close())
		if err != nil {
			snapshot = nil
		}
	}()
	tx, err := store.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	s := &ExecutionSnapshot{}
	if s.Namespace, err = readNamespace(ctx, tx); err != nil {
		return nil, err
	}
	if s.StorageVersion, err = storedSchemaNumber(ctx, tx); err != nil {
		return nil, err
	}
	base := protocolSchemaNumber(s.StorageVersion)

	s.Features = InspectionFeatures{
		Composition:     isCompositionSchema(base),
		ReceiptReads:    base == reconciliationSchemaVersion,
		ConflictJournal: hasReceiptConflictSchema(s.StorageVersion),
	}
	tables := protocolTables(s.StorageVersion)
	if err := checkProtocolTables(ctx, tx, tables); err != nil {
		return nil, err
	}
	if err := checkForeignKeys(ctx, tx); err != nil {
		return nil, err
	}
	b := inspectionBuilder{
		limits:       limits,
		snap:         s,
		tables:       make(map[string][]map[string]any),
		runs:         make(map[RunID]*ExecutionRun),
		programs:     make(map[string]Program),
		resolved:     make(map[RunID]Program),
		intents:      make(map[string]retainedIntent),
		steps:        make(map[ReductionRef]*ExecutionStep),
		before:       make(map[ReductionRef]Projection),
		after:        make(map[ReductionRef]Projection),
		observations: make(map[string]*ObservationView),
		invocations:  make(map[string]*InvocationView),
		effects:      make(map[string]*ExecutionEffect),
		membership:   make(map[RunID]string),
		facts:        make(map[string]compositionFact),
	}
	for _, name := range tables {
		if b.tables[name], err = scanInspectionTable(ctx, tx, name, true); err != nil {
			return nil, err
		}
	}
	// Everything used below came from this transaction. Release the read mark
	// before constructing the in-memory indexes so a writer can checkpoint.
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	for _, build := range []func() error{b.buildRuns, b.buildDeployment, b.buildHistory, b.buildCausality, b.buildEffects, b.checkCreationChronology, b.finishInspection} {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err = build(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func inspectionDecode[T any](row map[string]any, column string) (T, error) {
	var v T
	raw, ok := row[column].(string)
	if !ok || decode([]byte(raw), &v) != nil || !sameJSON([]byte(raw), v) {
		return v, corrupt("invalid canonical %s", column)
	}
	return v, nil
}

func inspectionRun(row map[string]any) RunID {
	s, _ := row["subject"].(string)
	g, _ := row["generation"].(int64)
	return RunID{s, g}
}

func inspectInt(row map[string]any, key string) int64   { v, _ := row[key].(int64); return v }
func inspectText(row map[string]any, key string) string { v, _ := row[key].(string); return v }

func runLess(a, b RunID) int {
	if a.Subject < b.Subject {
		return -1
	}
	if a.Subject > b.Subject {
		return 1
	}
	if a.Generation < b.Generation {
		return -1
	}
	if a.Generation > b.Generation {
		return 1
	}
	return 0
}

func (b *inspectionBuilder) buildRuns() error {
	subjects := make(map[string]Subject)
	for _, r := range b.tables["subjects"] {
		s := Subject{ID: inspectText(r, "id"), Generation: inspectInt(r, "generation"), Owner: inspectText(r, "owner"), Revision: inspectInt(r, "revision"), Retired: inspectInt(r, "retired") == 1, RetryCancelled: inspectInt(r, "retry_cancelled") == 1}
		if s.ID == "" || s.Owner == "" || s.Generation < 1 || s.Revision < 1 {
			return corrupt("invalid subject authority")
		}
		subjects[s.ID] = s
	}
	for _, row := range b.tables["permissions"] {
		g := Permission{Kind: inspectText(row, "kind"), Source: inspectText(row, "source"), Target: inspectText(row, "target"), Operation: inspectText(row, "operation"), Revision: inspectInt(row, "revision"), Obligation: inspectText(row, "obligation")}
		b.snap.Permissions = append(b.snap.Permissions, g)
	}
	for _, r := range b.tables["programs"] {
		raw, id := inspectText(r, "bytes"), inspectText(r, "id")
		if len(raw) > b.limits.MaxProgramBytes {
			return ErrLimit
		}
		var p Program
		if programHash([]byte(raw)) != id || decode([]byte(raw), &p) != nil || p.Semantics != r["semantics"] {
			return corrupt("invalid retained program %s", id)
		}
		if err := validateProgram(p); err != nil {
			return err
		}
		if !b.snap.Features.Composition && programHasInvocations(p) {
			return ErrVersion
		}
		b.programs[id] = p
	}
	for _, r := range b.tables["runs"] {
		id := inspectionRun(r)
		subject, ok := subjects[id.Subject]
		pid := inspectText(r, "program_id")
		p, found := b.programs[pid]
		if !ok || !found || id.Generation < 1 || id.Generation > subject.Generation {
			return corrupt("invalid run origin %+v", id)
		}
		initial, err := inspectionDecode[Projection](r, "initial")
		if err != nil {
			return err
		}
		current, err := inspectionDecode[Projection](r, "projection")
		if err != nil {
			return err
		}
		if err = validateProjection(initial, p, b.limits); err != nil {
			return fmt.Errorf("initial projection: %w", errors.Join(ErrCorrupt, err))
		}
		if err = validateAuthorInitial(p, initial); err != nil {
			return corrupt("invalid authored initial data for %+v: %v", id, err)
		}
		if err = validateProjection(current, p, b.limits); err != nil {
			return fmt.Errorf("current projection: %w", errors.Join(ErrCorrupt, err))
		}
		run := &ExecutionRun{RunState: RunState{ID: id, ProgramID: pid, Epoch: inspectInt(r, "epoch"), Sequence: inspectInt(r, "sequence"), Revision: subject.Revision, Projection: current}, Initial: initial, Subject: subject, Semantics: p.Semantics}
		if run.Epoch < 1 || run.Sequence < 0 || initial.Status != "open" {
			return corrupt("invalid run activation %+v", id)
		}
		b.runs[id] = run
		b.resolved[id] = p
		b.snap.Runs = append(b.snap.Runs, run)
	}
	for _, subject := range subjects {
		if b.runs[RunID{subject.ID, subject.Generation}] == nil {
			return corrupt("subject %s has no current generation", subject.ID)
		}
	}
	slices.SortFunc(b.snap.Runs, func(a, c *ExecutionRun) int { return runLess(a.ID, c.ID) })
	for _, r := range b.tables["corpus_assets"] {
		a, err := inspectionDecode[effects.Asset](r, "descriptor")
		if err != nil {
			return err
		}
		if a.ID != r["id"] {
			return corrupt("asset identity")
		}
		b.snap.Assets = append(b.snap.Assets, a)
	}
	if err := effects.CheckAssets(b.snap.Assets); err != nil {
		return corrupt("invalid retained assets: %v", err)
	}
	slices.SortFunc(b.snap.Assets, func(a, c effects.Asset) int { return strings.Compare(a.ID, c.ID) })
	b.snap.assetIndex = map[string]*effects.Asset{}
	for i := range b.snap.Assets {
		a := &b.snap.Assets[i]
		b.snap.assetIndex[a.ID] = a
	}
	return nil
}

func protocolTables(stored int) []string {
	tables := slices.Clone(inspectionTables)
	base := protocolSchemaNumber(stored)
	if isCompositionSchema(base) {
		tables = append(tables, compositionInspectionTables...)
		if base == reconciliationSchemaVersion {
			tables = append(tables, "reconciliation_profile", "receipt_read_ordinals")
		}
	}
	if hasReceiptConflictSchema(stored) {
		tables = append(tables, "receipt_conflicts")
	}
	return tables
}

// Check the physical format before selecting validators. Editing both version
// numbers cannot hide retained membership, expenditure or receipt evidence.
func checkProtocolTables(ctx context.Context, q queryer, tables []string) error {
	rows, err := q.QueryContext(ctx, "SELECT lower(name), type FROM sqlite_schema WHERE type IN ('table','view')")
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			rows.Close()
			return err
		}
		have[name] = kind == "table"
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	required := map[string]bool{}
	for _, name := range tables {
		required[name] = true
		if !have[name] {
			return ErrVersion
		}
	}
	for _, group := range [][]string{compositionInspectionTables, {"reconciliation_profile", "receipt_read_ordinals", "receipt_conflicts"}} {
		for _, name := range group {
			if _, exists := have[name]; exists && !required[name] {
				return ErrVersion
			}
		}
	}
	return nil
}

func checkForeignKeys(ctx context.Context, q queryer) error {
	rows, err := q.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	broken := rows.Next()
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	if broken {
		return corrupt("broken retained foreign key")
	}
	return nil
}
