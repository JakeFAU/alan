package executor

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
	publicschema "github.com/JakeFAU/alan/schema"
)

type AdapterAttempt struct {
	Intent          Intent
	AuthorizationID string
	Prepared        effects.Prepared
	Allowed         bool
	Reason          string
}
type AdapterResult struct {
	Digest      string
	Candidate   effects.Candidate
	Values      map[string]Value
	Disposition string
}

type adapterWork struct {
	attempt AdapterAttempt
	adapter effects.Adapter
	timeout time.Duration
	event   boundary.Event
}
type adapterCompletion struct {
	attempt   AdapterAttempt
	candidate effects.Candidate
	event     boundary.Event
	err       error
}

func readAssets(ctx context.Context, q queryer) ([]effects.Asset, error) {
	ids, e := stringList(ctx, q, "SELECT id FROM corpus_assets ORDER BY id")
	if e != nil {
		return nil, e
	}
	assets := []effects.Asset{}
	for _, id := range ids {
		var b []byte
		if e = q.QueryRowContext(ctx, "SELECT descriptor FROM corpus_assets WHERE id=?", id).Scan(&b); e != nil {
			return nil, e
		}
		var a effects.Asset
		if decode(b, &a) != nil || !sameJSON(b, a) || a.ID != id || a.Check() != nil {
			return nil, corrupt("corpus asset")
		}
		assets = append(assets, a)
	}
	if e = effects.CheckAssets(assets); e != nil {
		return nil, corrupt("corpus bounds")
	}
	return assets, nil
}
func adapterRequest(ctx context.Context, q queryer, id, namespace string, l Limits) (effects.Request, retainedIntent, error) {
	i, e := loadIntent(ctx, q, id)
	if e != nil {
		return effects.Request{}, i, e
	}
	if i.Kind != "adapter" || i.Generation != 1 {
		return effects.Request{}, i, fmt.Errorf("invalid adapter intent")
	}
	run, e := loadRun(ctx, q, i.Run)
	if e != nil {
		return effects.Request{}, i, e
	}
	p, e := loadProgram(ctx, q, run.ProgramID, l)
	if e != nil {
		return effects.Request{}, i, e
	}
	if p.Author == nil {
		return effects.Request{}, i, fmt.Errorf("adapter requires retained author contract")
	}
	effect, ok := p.Author.Effects[i.Obligation]
	if !ok || effect.Capability != i.Target {
		return effects.Request{}, i, fmt.Errorf("undeclared adapter capability")
	}
	if e = effect.Input.Record.Validate(i.Payload); e != nil {
		return effects.Request{}, i, fmt.Errorf("adapter input schema: %w", e)
	}
	assets, e := readAssets(ctx, q)
	return effects.Request{ID: id, Key: ExternalKey(namespace, id), Capability: i.Target, Effect: i.Obligation, Input: effect.Input, Output: effect.Output, Payload: i.Payload, Assets: assets}, i, e
}
func loadAdapterAttempt(ctx context.Context, q queryer, id string) (AdapterAttempt, error) {
	var a AdapterAttempt
	var b []byte
	e := q.QueryRowContext(ctx, "SELECT descriptor FROM adapter_attempts WHERE intent_id=?", id).Scan(&b)
	if e != nil {
		return a, e
	}
	if decode(b, &a) != nil || !sameJSON(b, a) || a.Intent.ID != id {
		return a, corrupt("adapter attempt encoding")
	}
	return a, nil
}
func loadAdapterResult(ctx context.Context, q queryer, id string) (AdapterResult, error) {
	var r AdapterResult
	var b []byte
	e := q.QueryRowContext(ctx, "SELECT descriptor FROM adapter_results WHERE intent_id=?", id).Scan(&b)
	if e != nil {
		return r, e
	}
	if decode(b, &r) != nil || !sameJSON(b, r) || r.Candidate.ID != id {
		return r, corrupt("adapter result encoding")
	}
	return r, nil
}
func adapterGrant(ctx context.Context, q queryer, i retainedIntent) (bool, error) {
	var n int
	e := q.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind='adapter' AND source=? AND target=? AND operation=? AND revision=? AND obligation=?", i.Run.Subject, i.Target, i.Obligation, i.revision, i.Obligation).Scan(&n)
	return n == 1, e
}

// prepareAdapterWork runs trusted pure serialization outside the Store lock.
// It checks a retained attempt first: recovery never needs an adapter for results
// already known, nor can registration cause uncertain attempts to be repeated.
func (s *Store) prepareAdapterWork(ctx context.Context, id string, registry map[string]effects.Adapter, timeout time.Duration) (*adapterWork, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, sql.ErrConnDone
	}
	old, e := loadAdapterAttempt(ctx, s.conn, id)
	if e == nil {
		r, re := loadAdapterResult(ctx, s.conn, id)
		s.mu.Unlock()
		if errors.Is(re, sql.ErrNoRows) {
			if old.Allowed {
				return nil, nil
			}
			disposition := "source_denied"
			if old.Reason == "grant_denied" {
				disposition = "grant_denied"
			}
			result, err := s.recordAdapterResult(ctx, old, effects.Candidate{ID: id, Key: old.Prepared.Request.Key, Disposition: disposition})
			if err != nil {
				return nil, err
			}
			return nil, s.settleAdapter(ctx, old, result)
		}
		if re != nil {
			return nil, re
		}
		return nil, s.settleAdapter(ctx, old, r)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		s.mu.Unlock()
		return nil, e
	}
	request, intent, e := adapterRequest(ctx, s.conn, id, s.namespace, s.limits)
	s.mu.Unlock()
	if e != nil {
		return nil, e
	}
	adapter := registry[intent.Target]
	if adapter == nil {
		return nil, fmt.Errorf("adapter capability %q unavailable", intent.Target)
	}
	// No locks, I/O or durable authority are exposed to Prepare. Implementations are
	// trusted to follow the documented pure/bounded contract.
	p, e := adapter.Prepare(effects.Copy(request))
	if e != nil {
		return nil, fmt.Errorf("adapter configuration/preparation rejected")
	}
	p = effects.Freeze(p)
	if !reflect.DeepEqual(p.Request, request) || effects.CheckPrepared(p) != nil || p.MaxResponseBytes > s.limits.MaxObservationBytes/8 {
		return nil, fmt.Errorf("prepared adapter request or bounds invalid")
	}
	auth, e := s.AuthorizeIntent(ctx, id)
	if e != nil {
		return nil, e
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, sql.ErrConnDone
	}
	tx, e := beginOwnedTx(ctx, s.conn)
	if e != nil {
		s.mu.Unlock()
		return nil, e
	}
	defer tx.Rollback()
	current, _, e := adapterRequest(ctx, tx, id, s.namespace, s.limits)
	if e != nil {
		s.mu.Unlock()
		return nil, e
	}
	if !reflect.DeepEqual(current, request) {
		s.mu.Unlock()
		return nil, ErrConflict
	}
	grant, e := adapterGrant(ctx, tx, intent)
	if e != nil {
		s.mu.Unlock()
		return nil, e
	}
	a := AdapterAttempt{Intent: intent.Intent, AuthorizationID: auth.ID, Prepared: p, Allowed: auth.Allowed && grant, Reason: auth.Reason}
	if auth.Allowed && !grant {
		a.Reason = "grant_denied"
	}
	a.Allowed, a.Reason, e = compositionReservationDecision(ctx, tx, intent.Intent, a.Allowed, a.Reason)
	if e != nil {
		s.mu.Unlock()
		return nil, e
	}
	b, e := json.Marshal(a)
	if e == nil {
		_, e = tx.ExecContext(ctx, "INSERT INTO adapter_attempts VALUES(?,?)", id, b)
	}
	if e == nil {
		e = recordCompositionReservation(ctx, tx, intent.Intent, a.Allowed, a.Reason, a)
	}
	if e == nil {
		e = s.commitOwnedTx(ctx, tx)
	}
	s.mu.Unlock()
	if e != nil {
		return nil, e
	}
	event := boundary.Event{Boundary: "adapter.authorized", Phase: "pre-transport", Subject: intent.Run.Subject, Generation: intent.Run.Generation, Epoch: auth.Source.Epoch, Sequence: intent.Sequence, Intent: id, Attempt: auth.ID}
	if !a.Allowed {
		d := "source_denied"
		if a.Reason == "grant_denied" {
			d = "grant_denied"
		}
		c := effects.Candidate{ID: id, Key: request.Key, Disposition: d}
		r, e := s.recordAdapterResult(ctx, a, c)
		if e != nil {
			return nil, e
		}
		return nil, s.settleAdapter(ctx, a, r)
	}
	return &adapterWork{attempt: effects.Copy(a), adapter: adapter, timeout: timeout, event: event}, nil
}
func transportAdapter(ctx context.Context, w adapterWork) adapterCompletion {
	r := adapterCompletion{attempt: w.attempt, event: w.event}
	if e := boundary.Reached(ctx, w.event); e != nil {
		r.err = e
		return r
	}
	limited, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()
	r.candidate = w.adapter.Execute(limited, effects.Copy(w.attempt.Prepared))
	r.event.Boundary = "adapter.response"
	r.event.Phase = "pre-record"
	r.err = boundary.Reached(ctx, r.event)
	finished := w.event
	finished.Boundary = "adapter.finished"
	finished.Phase = "transport-ended"
	r.err = errors.Join(r.err, boundary.Reached(ctx, finished))
	return r
}
func (s *Store) accountAdapter(ctx context.Context, c adapterCompletion) error {
	if c.candidate.Disposition == "unknown" || c.candidate.Disposition == "" {
		e := c.event
		e.Boundary = "adapter.unknown"
		e.Phase = "unresolved"
		return errors.Join(c.err, boundary.Reached(ctx, e))
	}
	r, e := s.recordAdapterResult(ctx, c.attempt, c.candidate)
	if e != nil {
		return errors.Join(c.err, e)
	}
	event := c.event
	event.Boundary = "adapter.result-recorded"
	event.Phase = "pre-settlement"
	if e = boundary.Reached(ctx, event); e != nil {
		return errors.Join(c.err, e)
	}
	if e = s.settleAdapter(ctx, c.attempt, r); e != nil {
		return errors.Join(c.err, e)
	}
	event.Boundary = "adapter.settled"
	event.Phase = "pre-consumption"
	return errors.Join(c.err, boundary.Reached(ctx, event))
}
func (s *Store) recordAdapterResult(ctx context.Context, a AdapterAttempt, c effects.Candidate) (AdapterResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := AdapterResult{}
	if s.closed {
		return r, sql.ErrConnDone
	}
	tx, e := beginOwnedTx(ctx, s.conn)
	if e != nil {
		return r, e
	}
	defer tx.Rollback()
	retained, e := loadAdapterAttempt(ctx, tx, a.Intent.ID)
	if e != nil {
		return r, e
	}
	if !reflect.DeepEqual(retained, a) || c.ID != a.Intent.ID || c.Key != a.Prepared.Request.Key {
		return r, ErrConflict
	}
	if e = validateAdapterAttempt(ctx, tx, a, s.namespace, s.limits); e != nil {
		return r, e
	}
	if c.Disposition == "unknown" || c.Disposition == "" {
		return r, fmt.Errorf("unknown outcome has no result")
	}
	if !a.Allowed && c.Disposition != "grant_denied" && c.Disposition != "source_denied" {
		return r, ErrConflict
	}
	values, d := effects.Result(a.Prepared.Request.Output.Record, c)
	r = AdapterResult{Candidate: effects.Copy(c), Values: values, Disposition: d}
	r.Digest = adapterResultDigest(r)
	if e = validateAdapterResult(a, r, s.limits); e != nil {
		return r, e
	}
	b, _ := json.Marshal(r)
	var prior []byte
	e = tx.QueryRowContext(ctx, "SELECT descriptor FROM adapter_results WHERE intent_id=?", a.Intent.ID).Scan(&prior)
	if e == nil {
		if !sameJSON(prior, r) {
			return r, ErrConflict
		}
		return r, s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return r, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO adapter_results VALUES(?,?)", a.Intent.ID, b); e != nil {
		return r, e
	}
	if e = s.recordCompositionSource(ctx, tx, compositionExternalResultFact, a.Intent.ID); e != nil {
		return r, e
	}
	return r, s.commitOwnedTx(ctx, tx)
}
func adapterDescriptor(a AdapterAttempt, r AdapterResult) AdmissionDescriptor {
	i := a.Intent
	return AdmissionDescriptor{TransportID: identity("adapter-delivery", i.ID), Kind: "adapter", LogicalID: i.ID, Source: RunID{i.Target, 1}, Target: i.Run.Subject, Generation: i.Run.Generation, Obligation: i.Obligation, Operation: i.ID, Payload: r.Values, IntentID: i.ID}
}
func (s *Store) settleAdapter(ctx context.Context, a AdapterAttempt, r AdapterResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return sql.ErrConnDone
	}
	tx, e := beginOwnedTx(ctx, s.conn)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	retained, e := loadAdapterResult(ctx, tx, a.Intent.ID)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(retained, r) {
		return ErrConflict
	}
	old, e := loadAdapterAttempt(ctx, tx, a.Intent.ID)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(old, a) {
		return ErrConflict
	}
	d := adapterDescriptor(a, r)
	prior, _, e := loadAdmission(ctx, tx, "adapter", d.TransportID)
	if e == nil {
		if !reflect.DeepEqual(prior, d) {
			return ErrConflict
		}
		return s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	bound := a.Intent.Run
	o := observationFor(d, bound)
	if e = insertObservation(ctx, tx, o, s.limits); e != nil {
		return e
	}
	result := AdmissionResult{TransportID: d.TransportID, Accepted: true, Reason: "settled", Bound: &bound, ObservationID: o.ID}
	raw, _ := json.Marshal(d)
	rr, _ := json.Marshal(result)
	if _, e = tx.ExecContext(ctx, "INSERT INTO admissions VALUES(?,?,?,?,?,?)", admissionKey("adapter", d.TransportID), "adapter", d.TransportID, raw, rr, o.ID); e != nil {
		return e
	}
	raw, _ = json.Marshal(r)
	if _, e = tx.ExecContext(ctx, "INSERT INTO settlements VALUES(?,?,?,?)", identity("adapter-outcome", a.Intent.ID), a.Intent.ID, raw, o.ID); e != nil {
		return e
	}
	return s.commitOwnedTx(ctx, tx)
}
func validateAdapterAttempt(ctx context.Context, q queryer, a AdapterAttempt, namespace string, l Limits) error {
	r, i, e := adapterRequest(ctx, q, a.Intent.ID, namespace, l)
	if e != nil {
		return e
	}
	auth, e := loadAuthorization(ctx, q, a.Intent.ID)
	if e != nil {
		return e
	}
	grant, e := adapterGrant(ctx, q, i)
	if e != nil {
		return e
	}
	reason := auth.Reason
	if auth.Allowed && !grant {
		reason = "grant_denied"
	}
	allowed, reason, e := retainedCompositionReservation(ctx, q, i.Intent, auth.Allowed && grant, reason)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(r, a.Prepared.Request) || !reflect.DeepEqual(i.Intent, a.Intent) || a.AuthorizationID != auth.ID || a.Allowed != allowed || a.Reason != reason || a.Prepared.Digest == "" || effects.CheckPrepared(a.Prepared) != nil || a.Prepared.MaxResponseBytes > l.MaxObservationBytes/8 {
		return corrupt("adapter attempt binding")
	}
	return nil
}
func validateAdapterResult(a AdapterAttempt, r AdapterResult, l Limits) error {
	if r.Digest == "" || r.Digest != adapterResultDigest(r) {
		return corrupt("adapter result digest")
	}
	c := r.Candidate
	if c.ID != a.Intent.ID || c.Key != a.Prepared.Request.Key || c.Disposition == "" || c.Disposition == "unknown" {
		return corrupt("adapter result correlation")
	}
	if len(c.Response) > a.Prepared.MaxResponseBytes || len(c.RequestID) > 512 || len(c.ResponseID) > 512 || len(c.ReturnedModel) > 512 {
		return corrupt("adapter response bounds")
	}
	if !a.Allowed {
		expected := "source_denied"
		if a.Reason == "grant_denied" {
			expected = "grant_denied"
		}
		if c.Disposition != expected || len(c.Payload) > 0 || c.Response != "" {
			return corrupt("denied adapter result")
		}
	}
	values, d := effects.Result(a.Prepared.Request.Output.Record, c)
	if d != r.Disposition || !reflect.DeepEqual(values, r.Values) || publicschema.ValidateResult(a.Prepared.Request.Output.Record, r.Values) != nil {
		return corrupt("adapter result validation")
	}
	return checkSize(r, l.MaxObservationBytes)
}
func validateAdapters(ctx context.Context, q queryer, l Limits) error {
	if _, e := readAssets(ctx, q); e != nil {
		return e
	}
	namespace, e := readNamespace(ctx, q)
	if e != nil {
		return e
	}
	ids, e := stringList(ctx, q, "SELECT intent_id FROM adapter_attempts ORDER BY intent_id")
	if e != nil {
		return e
	}
	for _, id := range ids {
		a, e := loadAdapterAttempt(ctx, q, id)
		if e != nil {
			return e
		}
		if e = validateAdapterAttempt(ctx, q, a, namespace, l); e != nil {
			return e
		}
		r, e := loadAdapterResult(ctx, q, id)
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			return e
		}
		if e = validateAdapterResult(a, r, l); e != nil {
			return e
		}
	}
	return validateAdapterEdges(ctx, q, l)
}

func validateAdapterEdges(ctx context.Context, q queryer, l Limits) error {
	verifiedObservations := map[string]int{}
	for _, query := range []string{
		"SELECT count(*) FROM adapter_results r LEFT JOIN adapter_attempts a ON a.intent_id=r.intent_id WHERE a.intent_id IS NULL",
		"SELECT count(*) FROM adapter_attempts a LEFT JOIN intents i ON i.id=a.intent_id WHERE i.id IS NULL",
	} {
		var n int
		if e := q.QueryRowContext(ctx, query).Scan(&n); e != nil {
			return e
		}
		if n != 0 {
			return corrupt("orphan adapter record")
		}
	}
	ids, e := stringList(ctx, q, "SELECT id FROM admissions WHERE domain='adapter' ORDER BY id")
	if e != nil {
		return e
	}
	for _, id := range ids {
		var raw, resultRaw []byte
		var transport, observation string
		if e = q.QueryRowContext(ctx, "SELECT transport_id,descriptor,result,observation_id FROM admissions WHERE id=?", id).Scan(&transport, &raw, &resultRaw, &observation); e != nil {
			return e
		}
		var d AdmissionDescriptor
		var admitted AdmissionResult
		if decode(raw, &d) != nil || decode(resultRaw, &admitted) != nil {
			return corrupt("adapter admission encoding")
		}
		a, e := loadAdapterAttempt(ctx, q, d.IntentID)
		if e != nil {
			return e
		}
		r, e := loadAdapterResult(ctx, q, d.IntentID)
		if e != nil {
			return e
		}
		expected := adapterDescriptor(a, r)
		bound := a.Intent.Run
		o := observationFor(expected, bound)
		expectedAdmission := AdmissionResult{TransportID: expected.TransportID, Accepted: true, Reason: "settled", Bound: &bound, ObservationID: o.ID}
		if id != admissionKey("adapter", expected.TransportID) || transport != expected.TransportID || observation != o.ID || !sameJSON(raw, expected) || !sameJSON(resultRaw, expectedAdmission) {
			return corrupt("adapter admission identity")
		}
		var retained []byte
		if e = q.QueryRowContext(ctx, "SELECT descriptor FROM observations WHERE id=?", o.ID).Scan(&retained); e != nil {
			return e
		}
		if !sameJSON(retained, o) {
			return corrupt("adapter observation binding")
		}
		var settled []byte
		var linked string
		if e = q.QueryRowContext(ctx, "SELECT descriptor,observation_id FROM settlements WHERE id=? AND intent_id=?", identity("adapter-outcome", d.IntentID), d.IntentID).Scan(&settled, &linked); e != nil {
			return e
		}
		if linked != o.ID || !sameJSON(settled, r) {
			return corrupt("adapter settlement binding")
		}
		verifiedObservations[o.ID]++
	}
	ids, e = stringList(ctx, q, "SELECT id FROM settlements ORDER BY id")
	if e != nil {
		return e
	}
	for _, id := range ids {
		var intent, observation string
		if e = q.QueryRowContext(ctx, "SELECT intent_id,observation_id FROM settlements WHERE id=?", id).Scan(&intent, &observation); e != nil {
			return e
		}
		i, e := loadIntent(ctx, q, intent)
		if e != nil {
			return e
		}
		if i.Kind != "adapter" {
			continue
		}
		if id != identity("adapter-outcome", intent) {
			return corrupt("adapter settlement identity")
		}
		var n int
		if e = q.QueryRowContext(ctx, "SELECT count(*) FROM admissions WHERE domain='adapter' AND observation_id=?", observation).Scan(&n); e != nil {
			return e
		}
		if n != 1 {
			return corrupt("adapter settlement without admission")
		}
	}
	// Forward links alone cannot explain an observation after both ledger edges
	// have been deleted. Reading descriptors is reducer-free and preserves the
	// valid known-result/no-observation state before settlement.
	observations, e := readObservations(ctx, q, l)
	if e != nil {
		return e
	}
	for id, observation := range observations {
		if observation.Provenance == "runtime-adapter" && verifiedObservations[id] != 1 {
			return corrupt("unexplained authenticated adapter observation")
		}
	}
	return nil
}

func adapterResultDigest(r AdapterResult) string {
	r.Digest = ""
	b, _ := json.Marshal(r)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
