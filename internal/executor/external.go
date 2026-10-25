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
	return resource.Request{Key: intent.ID, Profile: profile.Text, Delta: delta.Number, Subject: intent.Run.Subject, Generation: intent.Run.Generation, Sequence: intent.Sequence}, nil
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
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return ExternalAttempt{}, false, err
	}
	defer tx.Rollback()
	old, err := loadExternalAttempt(ctx, tx, id)
	if err == nil {
		return old, false, tx.Commit()
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
	raw, err := json.Marshal(a)
	if err != nil {
		return a, false, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO external_attempts VALUES(?,?)", id, raw); err != nil {
		return a, false, err
	}
	return a, true, tx.Commit()
}

// DispatchExternal makes at most one bounded HTTP request. Non-idempotent recovery
// never sends again after a retained attempt, even when the first send is uncertain.
func (s *Store) DispatchExternal(ctx context.Context, id string) (bool, error) {
	a, fresh, err := s.prepareExternal(ctx, id)
	if err != nil {
		return false, err
	}
	source, err := s.Snapshot(ctx, a.Intent.Run)
	if err != nil {
		return false, err
	}
	event := boundary.Event{Boundary: "external.authorized", Phase: "pre-transport", Subject: a.Intent.Run.Subject, Generation: a.Intent.Run.Generation, Epoch: source.Epoch, Sequence: source.Sequence, Intent: id, Attempt: a.AuthorizationID}
	if err = boundary.Reached(ctx, event); err != nil {
		return false, err
	}
	outcome, err := s.externalOutcome(ctx, id)
	if err == nil {
		return true, s.settleExternal(ctx, outcome)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if !a.Allowed {
		return false, ErrCorrupt
	}
	if !fresh && a.Request.Profile == "non-idempotent" {
		return false, nil
	}
	raw, err := json.Marshal(a.Request)
	if err != nil {
		return false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Intent.Target, bytes.NewReader(raw))
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, nil
	}
	var receipt resource.Receipt
	decoder := json.NewDecoder(io.LimitReader(response.Body, int64(s.limits.MaxObservationBytes)+1))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&receipt); err != nil {
		return false, nil
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return false, nil
	}
	if !validReceipt(a, receipt) {
		return false, fmt.Errorf("external receipt does not match retained request")
	}
	event.Boundary = "external.response"
	event.Phase = "pre-record"
	if err = boundary.Reached(ctx, event); err != nil {
		return false, err
	}
	if err = s.recordExternalReceipt(ctx, a, receipt); err != nil {
		return false, err
	}
	return true, s.settleExternal(ctx, ExternalOutcome{Attempt: a, Disposition: "applied", Receipt: &receipt})
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
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
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
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO external_receipts VALUES(?,?)", a.Intent.ID, raw); err != nil {
		return err
	}
	return tx.Commit()
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
	tx, err := s.conn.BeginTx(ctx, nil)
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
		return tx.Commit()
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
	return tx.Commit()
}
func validateExternal(ctx context.Context, q queryer) error {
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
