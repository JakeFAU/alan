package executor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"time"

	resource "github.com/JakeFAU/alan/internal/fixture"
)

type ExternalAttempt struct {
	Intent          Intent
	AuthorizationID string
	Request         resource.Request
	Allowed         bool
	Reason          string
}
type ExternalOutcome struct {
	Attempt     ExternalAttempt
	Disposition string
	Receipt     *resource.Receipt
}

// externalRequest validates fixture parameters; the store binds the returned
// request to its durable namespace before retaining or comparing it.
func externalRequest(intent Intent) (resource.Request, error) {
	profile, pok := intent.Payload["profile"]
	delta, dok := intent.Payload["delta"]
	u, err := url.Parse(intent.Target)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() {
		return resource.Request{}, fmt.Errorf("external endpoint must be literal loopback HTTP")
	}
	if intent.Kind != "external" || intent.Generation != 1 || intent.Obligation == "" || !pok || profile.Kind != "string" || (profile.Text != "idempotent" && profile.Text != "non-idempotent") || !dok || delta.Kind != "int" {
		return resource.Request{}, fmt.Errorf("invalid external intent")
	}
	return resource.Request{Profile: profile.Text, Delta: delta.Number, Subject: intent.Run.Subject, Generation: intent.Run.Generation, Sequence: intent.Sequence}, nil
}
func loadExternalAttempt(ctx context.Context, q queryer, id string) (ExternalAttempt, error) {
	var a ExternalAttempt
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM external_attempts WHERE intent_id=?", id).Scan(&raw); err != nil {
		return a, err
	}
	if err := decode(raw, &a); err != nil {
		return a, err
	}
	if a.Intent.ID != id || !sameJSON(raw, a) {
		return a, ErrCorrupt
	}
	return a, nil
}
func (s *Store) prepareExternal(ctx context.Context, id string) (ExternalAttempt, bool, error) {
	auth, err := s.AuthorizeIntent(ctx, id)
	if err != nil {
		return ExternalAttempt{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ExternalAttempt{}, false, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return ExternalAttempt{}, false, err
	}
	defer tx.Rollback()
	old, err := loadExternalAttempt(ctx, tx, id)
	if err == nil {
		return old, false, s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return old, false, err
	}
	retained, err := loadIntent(ctx, tx, id)
	if err != nil {
		return old, false, err
	}
	request, err := externalRequest(retained.Intent)
	if err != nil {
		return old, false, err
	}
	request.Key = ExternalKey(s.namespace, id)
	request.Epoch = auth.Source.Epoch
	request.Authorization = auth.ID
	a := ExternalAttempt{Intent: retained.Intent, AuthorizationID: auth.ID, Request: request, Allowed: auth.Allowed, Reason: auth.Reason}
	if auth.Allowed {
		var count int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind='external' AND source=? AND target=? AND operation='apply' AND revision=?", retained.Run.Subject, retained.Target, retained.revision).Scan(&count); err != nil {
			return a, false, err
		}
		if count == 0 {
			a.Allowed = false
			a.Reason = "grant_denied"
		}
	}
	a.Allowed, a.Reason, err = compositionReservationDecision(ctx, tx, retained.Intent, a.Allowed, a.Reason)
	if err != nil {
		return a, false, err
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return a, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO external_attempts VALUES(?,?)", id, raw); err != nil {
		return a, false, err
	}
	if err = recordCompositionReservation(ctx, tx, retained.Intent, a.Allowed, a.Reason, a); err != nil {
		return a, false, err
	}
	return a, true, s.commitOwnedTx(ctx, tx)
}

// externalWork is copied into a transport worker. Workers never access Store.
type externalWork struct {
	attempt  ExternalAttempt
	event    boundary.Event
	maxBytes int
	timeout  time.Duration
}
type externalCompletion struct {
	adapter *adapterCompletion
	work    externalWork
	receipt *resource.Receipt
	err     error
}

// prepareExternalWork retains authorization before exposing transport work.
func (s *Store) prepareExternalWork(ctx context.Context, id string, timeout time.Duration) (*externalWork, error) {
	a, fresh, err := s.prepareExternal(ctx, id)
	if err != nil {
		return nil, err
	}
	source, err := s.Snapshot(ctx, a.Intent.Run)
	if err != nil {
		return nil, err
	}
	outcome, err := s.externalOutcome(ctx, id)
	if err == nil {
		return nil, s.settleExternal(ctx, outcome)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if !a.Allowed {
		return nil, ErrCorrupt
	}
	if !fresh && a.Request.Profile == "non-idempotent" {
		return nil, nil
	}
	return &externalWork{attempt: a, event: boundary.Event{Boundary: "external.authorized", Phase: "pre-transport", Subject: a.Intent.Run.Subject, Generation: a.Intent.Run.Generation, Epoch: source.Epoch, Sequence: source.Sequence, Intent: id, Attempt: a.AuthorizationID}, maxBytes: s.limits.MaxObservationBytes, timeout: timeout}, nil
}

func transportExternal(ctx context.Context, work externalWork) externalCompletion {
	result := exchangeExternal(ctx, work)
	event := work.event
	event.Boundary = "external.finished"
	event.Phase = "transport-ended"
	result.err = errors.Join(result.err, boundary.Reached(ctx, event))
	return result
}
func exchangeExternal(ctx context.Context, work externalWork) externalCompletion {
	result := externalCompletion{work: work}
	a := work.attempt
	if err := boundary.Reached(ctx, work.event); err != nil {
		result.err = err
		return result
	}
	raw, err := json.Marshal(a.Request)
	if err != nil {
		result.err = err
		return result
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Intent.Target, bytes.NewReader(raw))
	if err != nil {
		result.err = err
		return result
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: work.timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		result.err = ctx.Err()
		return result
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result
	}
	var receipt resource.Receipt
	decoder := json.NewDecoder(io.LimitReader(response.Body, int64(work.maxBytes)+1))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&receipt); err != nil {
		return result
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return result
	}
	if !validReceipt(a, receipt) {
		result.err = fmt.Errorf("external receipt does not match retained request")
		return result
	}
	// Preserve a received outcome even if cancellation interrupts the test boundary.
	result.receipt = &receipt
	event := work.event
	event.Boundary = "external.response"
	event.Phase = "pre-record"
	result.err = boundary.Reached(ctx, event)
	return result
}
func (s *Store) accountExternal(ctx context.Context, result externalCompletion) error {
	if result.adapter != nil {
		return s.accountAdapter(ctx, *result.adapter)
	}
	if result.receipt == nil {
		return result.err
	}
	if err := s.recordExternalReceipt(ctx, result.work.attempt, *result.receipt); err != nil {
		return errors.Join(result.err, err)
	}
	err := s.settleExternal(ctx, ExternalOutcome{Attempt: result.work.attempt, Disposition: "applied", Receipt: result.receipt})
	if err == nil {
		event := result.work.event
		event.Boundary = "external.settled"
		event.Phase = "post-record"
		err = boundary.Reached(ctx, event)
	}
	return errors.Join(result.err, err)
}

// ErrDispatchMode rejects physical transport outside the composition Run owner.
// No authorization, reservation, outcome, or contact is made by this refusal.
var ErrDispatchMode = errors.New("composition transport requires Store.Run")

// DispatchExternal is the legacy synchronous API. Complete composition stores
// require Run for all physical dispatch, including their unscoped executions.
func (s *Store) DispatchExternal(ctx context.Context, id string) (bool, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false, sql.ErrConnDone
	}
	composition := s.invocationEnabled()
	s.mu.Unlock()
	if composition {
		return false, ErrDispatchMode
	}
	work, err := s.prepareExternalWork(ctx, id, DefaultExternalTimeout)
	if err != nil {
		return false, err
	}
	if work == nil {
		_, err = s.externalOutcome(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return err == nil, err
	}
	result := transportExternal(ctx, *work)
	err = s.accountExternal(ctx, result)
	return result.receipt != nil && err == nil, err
}
func validReceipt(a ExternalAttempt, r resource.Receipt) bool {
	return r.Key == a.Request.Key && r.Profile == a.Request.Profile && r.Delta == a.Request.Delta && r.Mutation > 0 && r.Process != "" && r.PID > 0
}
func (s *Store) recordExternalReceipt(ctx context.Context, a ExternalAttempt, r resource.Receipt) error {
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
	// Validate the retained attempt, intent and authorization, not the current epoch:
	// authentic old evidence remains valid after closure/restart.
	retained, err := loadExternalAttempt(ctx, tx, a.Intent.ID)
	if err != nil {
		return err
	}
	intent, err := loadIntent(ctx, tx, a.Intent.ID)
	if err != nil {
		return err
	}
	auth, err := loadAuthorization(ctx, tx, a.Intent.ID)
	if err != nil {
		return err
	}
	request, err := externalRequest(intent.Intent)
	if err != nil {
		return err
	}
	request.Key = ExternalKey(s.namespace, a.Intent.ID)
	request.Epoch = auth.Source.Epoch
	request.Authorization = auth.ID
	if !reflect.DeepEqual(retained, a) || !reflect.DeepEqual(intent.Intent, a.Intent) || !reflect.DeepEqual(request, a.Request) || !auth.Allowed || !a.Allowed || a.AuthorizationID != auth.ID || !validReceipt(a, r) {
		return corrupt("external completion binding")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(raw) > s.limits.MaxObservationBytes {
		return ErrLimit
	}
	var prior []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM external_receipts WHERE intent_id=?", a.Intent.ID).Scan(&prior)
	if err == nil {
		if !bytes.Equal(prior, raw) {
			return ErrConflict
		}
		return s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO external_receipts VALUES(?,?)", a.Intent.ID, raw); err != nil {
		return err
	}
	if err = s.recordCompositionSource(ctx, tx, compositionExternalResultFact, a.Intent.ID); err != nil {
		return err
	}
	return s.commitOwnedTx(ctx, tx)
}
func externalOutcome(ctx context.Context, q queryer, id string) (ExternalOutcome, error) {
	a, err := loadExternalAttempt(ctx, q, id)
	if err != nil {
		return ExternalOutcome{}, err
	}
	o := ExternalOutcome{Attempt: a, Disposition: "source_denied"}
	if !a.Allowed {
		if a.Reason == "grant_denied" {
			o.Disposition = "grant_denied"
		}
		return o, nil
	}
	var raw []byte
	var receipt resource.Receipt
	if err = q.QueryRowContext(ctx, "SELECT descriptor FROM external_receipts WHERE intent_id=?", id).Scan(&raw); err != nil {
		return o, err
	}
	if err = decode(raw, &receipt); err != nil {
		return o, err
	}
	if !sameJSON(raw, receipt) || !validReceipt(a, receipt) {
		return o, ErrCorrupt
	}
	o.Disposition = "applied"
	o.Receipt = &receipt
	return o, nil
}
func (s *Store) externalOutcome(ctx context.Context, id string) (ExternalOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ExternalOutcome{}, sql.ErrConnDone
	}
	return externalOutcome(ctx, s.conn, id)
}
func externalDescriptor(o ExternalOutcome) AdmissionDescriptor {
	intent := o.Attempt.Intent
	total := int64(0)
	mutation := int64(0)
	if o.Receipt != nil {
		total = o.Receipt.Total
		mutation = o.Receipt.Mutation
	}
	return AdmissionDescriptor{TransportID: identity("receipt-delivery", intent.ID), Kind: "receipt", LogicalID: intent.ID, Source: RunID{intent.Target, 1}, Target: intent.Run.Subject, Generation: intent.Run.Generation, Obligation: intent.Obligation, Operation: intent.ID, Payload: map[string]Value{"accepted": {Kind: "bool", Flag: o.Disposition == "applied"}, "disposition": Text(o.Disposition), "total": Int(total), "mutation": Int(mutation)}, IntentID: intent.ID}
}
func (s *Store) settleExternal(ctx context.Context, o ExternalOutcome) error {
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
	d := externalDescriptor(o)
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if len(raw) > s.limits.MaxObservationBytes {
		return ErrLimit
	}
	old, _, err := loadAdmission(ctx, tx, "receipt", d.TransportID)
	if err == nil {
		if !sameJSON(raw, old) {
			return ErrConflict
		}
		return s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	bound := o.Attempt.Intent.Run
	observation := observationFor(d, bound)
	if err = insertObservation(ctx, tx, observation, s.limits); err != nil {
		return err
	}
	result := AdmissionResult{TransportID: d.TransportID, Accepted: true, Reason: "settled", Bound: &bound, ObservationID: observation.ID}
	rr, _ := json.Marshal(result)
	if _, err = tx.ExecContext(ctx, "INSERT INTO admissions VALUES(?,?,?,?,?,?)", admissionKey("receipt", d.TransportID), "receipt", d.TransportID, raw, rr, observation.ID); err != nil {
		return err
	}
	raw, err = json.Marshal(o)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO settlements VALUES(?,?,?,?)", identity("external-outcome", o.Attempt.Intent.ID), o.Attempt.Intent.ID, raw, observation.ID); err != nil {
		return err
	}
	return s.commitOwnedTx(ctx, tx)
}
func validateExternal(ctx context.Context, q queryer) error {
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	rows, err := q.QueryContext(ctx, "SELECT intent_id FROM external_attempts ORDER BY intent_id")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return err
	}
	for _, id := range ids {
		a, err := loadExternalAttempt(ctx, q, id)
		if err != nil {
			return err
		}
		intent, err := loadIntent(ctx, q, id)
		if err != nil {
			return err
		}
		request, err := externalRequest(intent.Intent)
		if err != nil {
			return err
		}
		auth, err := loadAuthorization(ctx, q, id)
		if err != nil {
			return err
		}
		request.Key = ExternalKey(namespace, id)
		request.Epoch = auth.Source.Epoch
		request.Authorization = auth.ID
		allowed, reason := auth.Allowed, auth.Reason
		if allowed {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind='external' AND source=? AND target=? AND operation='apply' AND revision=?", intent.Run.Subject, intent.Target, intent.revision).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				allowed = false
				reason = "grant_denied"
			}
		}
		allowed, reason, err = retainedCompositionReservation(ctx, q, intent.Intent, allowed, reason)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(a, ExternalAttempt{Intent: intent.Intent, AuthorizationID: auth.ID, Request: request, Allowed: allowed, Reason: reason}) {
			return corrupt("external attempt binding")
		}
		var receiptCount int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM external_receipts WHERE intent_id=?", id).Scan(&receiptCount); err != nil {
			return err
		}
		if receiptCount > 0 {
			if !allowed {
				return corrupt("receipt without authorization")
			}
			if _, err = externalOutcome(ctx, q, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// retryEligible skips retained non-idempotent uncertainty without spending a launch.
func (s *Store) retryEligible(ctx context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false, sql.ErrConnDone
	}
	a, err := loadExternalAttempt(ctx, s.conn, id)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !a.Allowed || a.Request.Profile != "non-idempotent" {
		return true, nil
	}
	_, err = externalOutcome(ctx, s.conn, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
