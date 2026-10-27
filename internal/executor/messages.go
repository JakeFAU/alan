package executor

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"reflect"
)

// Input is the local operator ingress. Its origin/type are fixed by this API;
// arbitrary payload keys cannot become authenticated runtime metadata.
type Input struct {
	TransportID, LogicalID, Target string
	Generation                     int64 // Zero selects and retains the target's first current-run binding.
	Obligation, Operation          string
	Payload                        map[string]Value
}
type AdmissionResult struct {
	TransportID   string
	Accepted      bool
	Reason        string
	Bound         *RunID
	ObservationID string
}
type TerminalDestination struct{ Target, Obligation string }

func (d TerminalDestination) key() string { return identity("route", d.Target, d.Obligation) }
func identity(parts ...any) string        { b, _ := json.Marshal(parts); return string(b) }

// AdmissionDescriptor is retained evidence, never an unauthenticated ingress API.
type AdmissionDescriptor struct {
	TransportID, Kind, LogicalID string
	Source                       RunID
	Target                       string
	Generation                   int64
	Obligation, Operation        string
	Payload                      map[string]Value
	IntentID                     string
	TerminalSequence             int64
}
type Authorization struct {
	ID, IntentID     string
	Allowed          bool
	Reason           string
	Source           Authority
	ClosureOperation string
}
type authorizationRequest struct {
	Intent   Intent
	Revision int64
}

func (s *Store) AdmitInput(ctx context.Context, input Input) (AdmissionResult, error) {
	d := AdmissionDescriptor{TransportID: input.TransportID, Kind: "input", LogicalID: input.LogicalID, Source: RunID{"local", 1}, Target: input.Target, Generation: input.Generation, Obligation: input.Obligation, Operation: input.Operation, Payload: input.Payload}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	result, err := admit(ctx, tx, d, s.limits, "")
	if err != nil {
		return result, err
	}
	return result, s.commitOwnedTx(ctx, tx)
}

func loadIntent(ctx context.Context, q queryer, id string) (retainedIntent, error) {
	var r retainedIntent
	if err := q.QueryRowContext(ctx, "SELECT source_revision,descriptor FROM intents WHERE id=?", id).Scan(&r.revision, &r.raw); err != nil {
		return r, err
	}
	if err := decode(r.raw, &r.Intent); err != nil {
		return r, err
	}
	if r.ID != id || !sameJSON(r.raw, r.Intent) {
		return r, ErrCorrupt
	}
	return r, nil
}

// AuthorizeIntent retains the first source disposition separately from admission.
// Prior authorization survives source closure; uncommitted identities are errors.
func (s *Store) AuthorizeIntent(ctx context.Context, id string) (Authorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Authorization{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return Authorization{}, err
	}
	defer tx.Rollback()
	intent, err := loadIntent(ctx, tx, id)
	if err != nil {
		return Authorization{}, err
	}
	request := authorizationRequest{intent.Intent, intent.revision}
	descriptor, err := json.Marshal(request)
	if err != nil {
		return Authorization{}, err
	}
	authID := identity("authorization", id)
	var old, raw []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor,result FROM attempts WHERE id=?", authID).Scan(&old, &raw)
	if err == nil {
		if !bytes.Equal(old, descriptor) {
			return Authorization{}, ErrConflict
		}
		var result Authorization
		if err = decode(raw, &result); err != nil {
			return result, err
		}
		return result, s.authorizationBoundary(ctx, tx, result)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Authorization{}, err
	}
	source, err := loadRun(ctx, tx, intent.Run)
	if err != nil {
		return Authorization{}, err
	}
	result := Authorization{ID: authID, IntentID: id, Source: source.Authority()}
	var closure sql.NullString
	closureErr := tx.QueryRowContext(ctx, "SELECT operation_id FROM terminal_obligations WHERE subject=? AND generation=?", source.ID.Subject, source.ID.Generation).Scan(&closure)
	if closureErr != nil && !errors.Is(closureErr, sql.ErrNoRows) {
		return result, closureErr
	}
	if closure.Valid {
		result.ClosureOperation = closure.String
	}

	var generation int64
	var retired bool
	if err = tx.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", intent.Run.Subject).Scan(&generation, &retired); err != nil {
		return result, err
	}
	reply := false
	if s.invocationEnabled() {
		_, reply, err = compositionReplyBinding(ctx, tx, intent.Intent)
		if err != nil {
			return result, err
		}
	}
	switch {
	case source.Revision != intent.revision:
		result.Reason = "source_revision_changed"
	case generation != intent.Run.Generation:
		result.Reason = "source_generation_changed"
	case retired && !reply:
		result.Reason = "source_retired"
	case source.Projection.Status != "open" && !reply:
		result.Reason = "source_closed"
	default:
		result.Allowed = true
		result.Reason = "authorized"
	}
	raw, err = json.Marshal(result)
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO attempts VALUES(?,?,?,?)", authID, id, descriptor, raw); err != nil {
		return result, err
	}
	if err = s.recordCompositionSource(ctx, tx, compositionAuthorizationFact, id); err != nil {
		return result, err
	}
	return result, s.authorizationBoundary(ctx, tx, result)
}

func (s *Store) DeliverMessage(ctx context.Context, intentID, transportID string) (AdmissionResult, error) {
	// This authorization is its own committed transaction, not an in-memory permit.
	auth, err := s.AuthorizeIntent(ctx, intentID)
	if err != nil {
		return AdmissionResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	intent, err := loadIntent(ctx, tx, intentID)
	if err != nil {
		return AdmissionResult{}, err
	}
	if intent.Kind != "message" {
		return AdmissionResult{}, fmt.Errorf("intent is not an ordinary message")
	}
	if s.invocationEnabled() {
		b, reply, e := compositionReplyBinding(ctx, tx, intent.Intent)
		if e != nil {
			return AdmissionResult{}, e
		}
		if reply {
			result, e := recordCompositionReply(ctx, tx, b, intent, auth, transportID, s.limits)
			if e != nil {
				return result, e
			}
			return result, s.commitOwnedTx(ctx, tx)
		}
	}
	descriptor := messageDescriptor(intent.Intent, transportID)
	denial := ""
	if !auth.Allowed {
		denial = "source_denied:" + auth.Reason
	}
	result, err := admit(ctx, tx, descriptor, s.limits, denial)
	if err != nil {
		return result, err
	}
	return result, s.commitOwnedTx(ctx, tx)
}
func messageDescriptor(intent Intent, transport string) AdmissionDescriptor {
	return AdmissionDescriptor{TransportID: transport, Kind: "message", LogicalID: intent.ID, Source: intent.Run, Target: intent.Target, Generation: intent.Generation, Obligation: intent.Obligation, Payload: intent.Payload, IntentID: intent.ID}
}

func (s *Store) AcknowledgeMessage(ctx context.Context, intentID, transportID string) error {
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
	descriptor, _, err := loadAdmission(ctx, tx, "message", transportID)
	if err != nil {
		return err
	}
	if descriptor.Kind != "message" || descriptor.IntentID != intentID {
		return ErrConflict
	}
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT admission_id FROM message_acknowledgments WHERE intent_id=?", intentID).Scan(&existing)
	if err == nil {
		return s.commitOwnedTx(ctx, tx)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO message_acknowledgments VALUES(?,?)", intentID, admissionKey("message", transportID)); err != nil {
		return err
	}
	return s.commitOwnedTx(ctx, tx)
}

func terminalDescriptor(ctx context.Context, q queryer, source RunID, destination TerminalDestination, transport string) (AdmissionDescriptor, error) {
	var sequence int64
	var raw, routes []byte
	if err := q.QueryRowContext(ctx, "SELECT sequence,outcome,destinations FROM terminal_obligations WHERE subject=? AND generation=?", source.Subject, source.Generation).Scan(&sequence, &raw, &routes); err != nil {
		return AdmissionDescriptor{}, err
	}
	var projection Projection
	var destinations []TerminalDestination
	if err := decode(raw, &projection); err != nil {
		return AdmissionDescriptor{}, err
	}
	if err := decode(routes, &destinations); err != nil {
		return AdmissionDescriptor{}, err
	}
	allowed := false
	for _, d := range destinations {
		if d == destination {
			allowed = true
		}
	}
	if !allowed {
		return AdmissionDescriptor{}, fmt.Errorf("terminal destination not retained")
	}
	return AdmissionDescriptor{TransportID: transport, Kind: "terminal", LogicalID: identity("terminal", source.Subject, source.Generation, sequence, destination.Target, destination.Obligation), Source: source, Target: destination.Target, Obligation: destination.Obligation, Payload: map[string]Value{"outcome": Text(projection.Status)}, TerminalSequence: sequence}, nil
}
func (s *Store) DeliverTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) (AdmissionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	descriptor, err := terminalDescriptor(ctx, tx, source, destination, transportID)
	if err != nil {
		return AdmissionResult{}, err
	}
	result, err := admit(ctx, tx, descriptor, s.limits, "")
	if err != nil {
		return result, err
	}
	return result, s.commitOwnedTx(ctx, tx)
}
func (s *Store) AcknowledgeTerminal(ctx context.Context, source RunID, destination TerminalDestination, transportID string) error {
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
	expected, err := terminalDescriptor(ctx, tx, source, destination, transportID)
	if err != nil {
		return err
	}
	descriptor, _, err := loadAdmission(ctx, tx, "terminal", transportID)
	if err != nil {
		return err
	}
	b, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}
	if !sameJSON(b, expected) {
		return ErrConflict
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, "SELECT delivery_status FROM terminal_obligations WHERE subject=? AND generation=?", source.Subject, source.Generation).Scan(&raw); err != nil {
		return err
	}
	delivered := map[string]bool{}
	if err = decode(raw, &delivered); err != nil {
		return err
	}
	delivered[destination.key()] = true
	raw, err = json.Marshal(delivered)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE terminal_obligations SET delivery_status=? WHERE subject=? AND generation=?", raw, source.Subject, source.Generation); err != nil {
		return err
	}
	return s.commitOwnedTx(ctx, tx)
}

// admissionKey includes a domain chosen by the trusted API entry point. A raw
// transport ID is never parsed as, or allowed to supply, that discriminator.
func admissionKey(domain, transportID string) string {
	return identity("admission", domain, transportID)
}
func loadAdmission(ctx context.Context, q queryer, domain, transportID string) (AdmissionDescriptor, AdmissionResult, error) {
	return loadAdmissionKey(ctx, q, admissionKey(domain, transportID))
}
func loadAdmissionKey(ctx context.Context, q queryer, id string) (AdmissionDescriptor, AdmissionResult, error) {
	var d AdmissionDescriptor
	var r AdmissionResult
	var domain, transportID string
	var descriptor, result []byte
	if err := q.QueryRowContext(ctx, "SELECT domain,transport_id,descriptor,result FROM admissions WHERE id=?", id).Scan(&domain, &transportID, &descriptor, &result); err != nil {
		return d, r, err
	}
	if err := decode(descriptor, &d); err != nil {
		return d, r, err
	}
	if err := decode(result, &r); err != nil {
		return d, r, err
	}
	if id != admissionKey(domain, transportID) || domain != d.Kind || d.TransportID != transportID || r.TransportID != transportID || !sameJSON(descriptor, d) || !sameJSON(result, r) {
		return d, r, ErrCorrupt
	}
	return d, r, nil
}
func observationFor(d AdmissionDescriptor, bound RunID) Observation {
	provenance := map[string]string{"input": "local-input", "message": "ordinary-message", "terminal": "runtime-terminal", "lifecycle": "runtime-lifecycle", "receipt": "resource-receipt", "adapter": "runtime-adapter"}[d.Kind]
	return Observation{ID: identity("observation", d.Kind, d.LogicalID), Run: bound, Kind: d.Kind, Provenance: provenance, Source: d.Source.Subject, SourceGeneration: d.Source.Generation, Obligation: d.Obligation, Operation: d.Operation, Payload: d.Payload}
}
func admit(ctx context.Context, tx *sql.Tx, d AdmissionDescriptor, limits Limits, sourceDenial string) (AdmissionResult, error) {
	result := AdmissionResult{TransportID: d.TransportID}
	switch d.Kind {
	case "input", "message", "terminal":
	default:
		return result, fmt.Errorf("invalid trusted admission domain")
	}
	key := admissionKey(d.Kind, d.TransportID)
	if d.TransportID == "" || d.LogicalID == "" || d.Target == "" || d.Generation < 0 || d.Payload == nil {
		return result, fmt.Errorf("invalid admission descriptor")
	}
	if err := checkSize(d, limits.MaxObservationBytes); err != nil {
		return result, err
	}
	for key, value := range d.Payload {
		if key == "" || !validValue(value) {
			return result, fmt.Errorf("invalid admission payload")
		}
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		return result, err
	}
	var existing []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM admissions WHERE id=?", key).Scan(&existing)
	if err == nil {
		if !bytes.Equal(existing, encoded) {
			return result, ErrConflict
		}
		_, result, err = loadAdmission(ctx, tx, d.Kind, d.TransportID)
		return result, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	var generation int64
	var retired bool
	err = tx.QueryRowContext(ctx, "SELECT generation,retired FROM subjects WHERE id=?", d.Target).Scan(&generation, &retired)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		result.Reason = "target_missing"
	case err != nil:
		return result, err
	default:
		result.Bound = &RunID{d.Target, generation}
		switch {
		case sourceDenial != "":
			result.Reason = sourceDenial
		case retired:
			result.Reason = "target_retired"
		case d.Generation != 0 && d.Generation != generation:
			result.Reason = "target_generation_changed"
		default:
			target, e := loadRun(ctx, tx, *result.Bound)
			if e != nil {
				return result, e
			}
			if target.Projection.Status != "open" {
				result.Reason = "target_closed"
			} else {
				permitted, e := admissionPermitted(ctx, tx, d)
				if e != nil {
					return result, e
				}
				if !permitted {
					result.Reason = "route_denied"
				} else {
					program, e := loadProgram(ctx, tx, target.ProgramID, limits)
					if e != nil {
						return result, e
					}
					program, e = boundCompositionProgram(ctx, tx, program, target.ID, limits)
					if e != nil {
						return result, e
					}
					if !authorAdmissionValid(program, d) {
						result.Reason = "schema_invalid"
					} else {
						result.Accepted = true
						result.Reason = "accepted"
					}
				}
			}
		}
	}
	if sourceDenial != "" {
		result.Accepted = false
		result.Reason = sourceDenial
	}
	if result.Accepted {
		observation := observationFor(d, *result.Bound)
		if err = insertObservation(ctx, tx, observation, limits); err != nil {
			return result, err
		}
		result.ObservationID = observation.ID
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	var observationID any
	if result.ObservationID != "" {
		observationID = result.ObservationID
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO admissions VALUES(?,?,?,?,?,?)", key, d.Kind, d.TransportID, encoded, raw, observationID); err != nil {
		return result, err
	}
	return result, nil
}
func admissionPermitted(ctx context.Context, q queryer, d AdmissionDescriptor) (bool, error) {
	if d.Kind == "terminal" || d.Kind == "lifecycle" || d.Kind == "receipt" || d.Kind == "adapter" {
		return true, nil
	} // terminalDescriptor validated the frozen route.
	revision := int64(0)
	if d.Kind == "message" {
		intent, err := loadIntent(ctx, q, d.IntentID)
		if err != nil {
			return false, err
		}
		revision = intent.revision
	}
	var count int
	err := q.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind=? AND source=? AND target=? AND operation=? AND revision=? AND obligation=?", d.Kind, d.Source.Subject, d.Target, d.Kind, revision, d.Obligation).Scan(&count)
	return count > 0, err
}

// validateMessaging checks protocol evidence without choosing new work or
// reevaluating retained admissions against changed target circumstances.
func validateMessaging(ctx context.Context, q queryer, limits Limits) error {
	observations, err := readObservations(ctx, q, limits)
	if err != nil {
		return err
	}
	authIDs, err := stringList(ctx, q, "SELECT id FROM attempts ORDER BY id")
	if err != nil {
		return err
	}
	authorizations := map[string]Authorization{}
	for _, id := range authIDs {
		var intentID string
		var descriptor, result []byte
		if err = q.QueryRowContext(ctx, "SELECT intent_id,descriptor,result FROM attempts WHERE id=?", id).Scan(&intentID, &descriptor, &result); err != nil {
			return err
		}
		intent, e := loadIntent(ctx, q, intentID)
		if e != nil {
			return e
		}
		if !sameJSON(descriptor, authorizationRequest{intent.Intent, intent.revision}) {
			return corrupt("authorization descriptor %s", id)
		}
		var auth Authorization
		if err = decode(result, &auth); err != nil {
			return err
		}
		if !sameJSON(result, auth) || id != identity("authorization", intentID) || auth.ID != id || auth.IntentID != intentID || auth.Source.Run != intent.Run || auth.Source.Sequence < intent.Sequence || auth.Source.Epoch <= 0 || auth.Source.Revision <= 0 {
			return corrupt("authorization identity %s", id)
		}
		source, e := loadRun(ctx, q, intent.Run)
		if e != nil {
			return e
		}
		if auth.Source.Sequence > source.Sequence || auth.Source.Epoch > source.Epoch {
			return corrupt("authorization source boundary %s", id)
		}
		projection, e := authorizationProjection(ctx, q, auth)
		if e != nil {
			return e
		}
		reply := false
		enabled, e := compositionProtocol(ctx, q)
		if e != nil {
			return e
		}
		if enabled {
			_, reply, e = compositionReplyBinding(ctx, q, intent.Intent)
			if e != nil {
				return e
			}
		}
		if auth.Allowed {
			if auth.Reason != "authorized" || auth.Source.Revision != intent.revision || (projection.Status != "open" && !reply) {
				return corrupt("unsupported authorization %s", id)
			}
		} else {
			switch auth.Reason {
			case "source_closed":
				if projection.Status == "open" {
					return corrupt("unsupported closure denial")
				}
			case "source_revision_changed":
				if auth.Source.Revision == intent.revision {
					return corrupt("unsupported revision denial")
				}
			case "source_generation_changed", "source_retired":
			default:
				return corrupt("unknown authorization denial")
			}
		}
		authorizations[intentID] = auth
	}
	ids, err := stringList(ctx, q, "SELECT id FROM admissions ORDER BY id")
	if err != nil {
		return err
	}
	explained := map[string]bool{}
	descriptors := map[string]AdmissionDescriptor{}
	for _, id := range ids {
		d, r, e := loadAdmissionKey(ctx, q, id)
		if e != nil {
			return e
		}
		if d.LogicalID == "" || d.Target == "" || d.Generation < 0 || d.Payload == nil {
			return corrupt("invalid admission descriptor")
		}
		if err = checkSize(d, limits.MaxObservationBytes); err != nil {
			return err
		}
		switch d.Kind {
		case "input":
			if d.Source != (RunID{"local", 1}) || d.IntentID != "" || d.TerminalSequence != 0 {
				return corrupt("input origin")
			}
		case "message":
			intent, e := loadIntent(ctx, q, d.IntentID)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			if !sameJSON(raw, messageDescriptor(intent.Intent, d.TransportID)) || intent.Kind != "message" {
				return corrupt("message origin")
			}
			auth, ok := authorizations[d.IntentID]
			if !ok {
				return corrupt("message lacks retained source disposition")
			}
			if !auth.Allowed && (r.Accepted || r.Reason != "source_denied:"+auth.Reason) {
				return corrupt("message bypassed source denial")
			}
		case "terminal":
			expected, e := terminalDescriptor(ctx, q, d.Source, TerminalDestination{d.Target, d.Obligation}, d.TransportID)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			if !sameJSON(raw, expected) {
				return corrupt("terminal origin")
			}
		case "adapter":
			a, e := loadAdapterAttempt(ctx, q, d.IntentID)
			if e != nil {
				return e
			}
			result, e := loadAdapterResult(ctx, q, d.IntentID)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(d, adapterDescriptor(a, result)) || r.Bound == nil || *r.Bound != a.Intent.Run {
				return corrupt("adapter admission binding")
			}
			var count int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=? AND observation_id=?", identity("adapter-outcome", d.IntentID), r.ObservationID).Scan(&count); e != nil {
				return e
			}
			if count != 1 {
				return corrupt("adapter admission lacks settlement")
			}
		case "receipt":
			outcome, e := externalOutcome(ctx, q, d.IntentID)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			if !sameJSON(raw, externalDescriptor(outcome)) || r.Bound == nil || *r.Bound != outcome.Attempt.Intent.Run {
				return corrupt("external receipt admission")
			}
			var count int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=? AND observation_id=?", identity("external-outcome", d.IntentID), r.ObservationID).Scan(&count); e != nil {
				return e
			}
			if count != 1 {
				return corrupt("receipt admission lacks settlement")
			}
		case "lifecycle":
			outcome, e := lifecycleOutcome(ctx, q, d.IntentID)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			if !sameJSON(raw, lifecycleDescriptor(outcome, d.TransportID)) || r.Bound == nil || *r.Bound != outcome.Request.Intent.Run {
				return corrupt("lifecycle outcome admission")
			}
			var count int
			if e = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=? AND observation_id=?", identity("lifecycle-outcome", d.IntentID), r.ObservationID).Scan(&count); e != nil {
				return e
			}
			if count != 1 {
				return corrupt("lifecycle admission lacks settlement")
			}
		default:
			return corrupt("unknown admission kind")
		}
		var observationID sql.NullString
		if err = q.QueryRowContext(ctx, "SELECT observation_id FROM admissions WHERE id=?", id).Scan(&observationID); err != nil {
			return err
		}
		if r.Bound != nil && (r.Bound.Subject != d.Target || r.Bound.Generation <= 0) {
			return corrupt("admission binding")
		}
		if d.Kind == "message" {
			enabled, e := compositionProtocol(ctx, q)
			if e != nil {
				return e
			}
			if enabled {
				i, e := loadIntent(ctx, q, d.IntentID)
				if e != nil {
					return e
				}
				b, reply, e := compositionReplyBinding(ctx, q, i.Intent)
				if e != nil {
					return e
				}
				if reply {
					var raw []byte
					var candidate compositionReply
					if e = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_replies WHERE operation_id=?", i.ID).Scan(&raw); e != nil {
						return e
					}
					if e = decode(raw, &candidate); e != nil {
						return e
					}
					want := AdmissionResult{TransportID: d.TransportID, Accepted: candidate.Accepted, Reason: candidate.Reason, Bound: &b.Caller}
					if !reflect.DeepEqual(r, want) || observationID.Valid {
						return corrupt("reply transport admission")
					}
					descriptors[id] = d
					continue
				}
			}
		}
		if r.Accepted {
			if (r.Reason != "accepted" && !((d.Kind == "lifecycle" || d.Kind == "receipt" || d.Kind == "adapter") && r.Reason == "settled")) || r.Bound == nil || r.ObservationID == "" || !observationID.Valid || observationID.String != r.ObservationID {
				return corrupt("accepted admission result")
			}
			if d.Generation != 0 && d.Generation != r.Bound.Generation {
				return corrupt("accepted admission generation")
			}
			expected := observationFor(d, *r.Bound)
			o, ok := observations[r.ObservationID]
			if !ok {
				return corrupt("admission observation missing")
			}
			raw, e := json.Marshal(o.Observation)
			if e != nil {
				return e
			}
			if !sameJSON(raw, expected) {
				return corrupt("admission observation mismatch")
			}
			permitted, e := admissionPermitted(ctx, q, d)
			if e != nil {
				return e
			}
			if !permitted {
				return corrupt("accepted admission lacks static route")
			}
			target, e := loadRun(ctx, q, *r.Bound)
			if e != nil {
				return e
			}
			program, e := loadProgram(ctx, q, target.ProgramID, limits)
			if e != nil {
				return e
			}
			program, e = boundCompositionProgram(ctx, q, program, target.ID, limits)
			if e != nil {
				return e
			}
			if !authorAdmissionValid(program, d) {
				return corrupt("accepted admission violates author schema")
			}
			explained[r.ObservationID] = true
		} else {
			if r.ObservationID != "" || observationID.Valid {
				return corrupt("rejected admission has observation")
			}
			switch r.Reason {
			case "schema_invalid", "target_missing", "target_retired", "target_generation_changed", "target_closed", "route_denied", "source_denied:source_closed", "source_denied:source_retired", "source_denied:source_revision_changed", "source_denied:source_generation_changed":
			default:
				return corrupt("unknown admission rejection")
			}
		}
		descriptors[id] = d
	}
	for id, o := range observations {
		switch o.Provenance {
		case "local-input", "ordinary-message", "runtime-terminal", "runtime-lifecycle", "resource-receipt", "runtime-adapter":
			if !explained[id] {
				return corrupt("unexplained authenticated observation %s", id)
			}
		}
	}
	rows, err := q.QueryContext(ctx, "SELECT intent_id,admission_id FROM message_acknowledgments")
	if err != nil {
		return err
	}
	for rows.Next() {
		var intent, admissionID string
		if err = rows.Scan(&intent, &admissionID); err != nil {
			rows.Close()
			return err
		}
		d, ok := descriptors[admissionID]
		if !ok || d.Kind != "message" || d.IntentID != intent {
			rows.Close()
			return corrupt("message acknowledgment binding")
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	terminalRows, err := q.QueryContext(ctx, "SELECT subject,generation,delivery_status FROM terminal_obligations")
	if err != nil {
		return err
	}
	defer terminalRows.Close()
	for terminalRows.Next() {
		var source RunID
		var raw []byte
		if err = terminalRows.Scan(&source.Subject, &source.Generation, &raw); err != nil {
			return err
		}
		var status map[string]bool
		if err = decode(raw, &status); err != nil {
			return err
		}
		for route, delivered := range status {
			if !delivered {
				continue
			}
			found := false
			for _, d := range descriptors {
				if d.Kind == "terminal" && d.Source == source && (TerminalDestination{d.Target, d.Obligation}).key() == route {
					found = true
					break
				}
			}
			if !found {
				return corrupt("terminal acknowledgment lacks admission")
			}
		}
	}
	return terminalRows.Err()
}

func (s *Store) authorizationBoundary(ctx context.Context, tx *sql.Tx, auth Authorization) error {
	source, err := loadRun(ctx, tx, auth.Source.Run)
	if err != nil {
		return err
	}
	if err = s.commitOwnedTx(ctx, tx); err != nil {
		return err
	}
	return boundary.Reached(ctx, boundary.Event{Boundary: "intent.authorized", Phase: auth.Reason, Subject: source.ID.Subject, Generation: source.ID.Generation, Epoch: source.Epoch, Sequence: source.Sequence, Intent: auth.IntentID, Attempt: auth.ID})
}
