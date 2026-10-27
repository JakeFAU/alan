package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"slices"
	"strings"
)

func compositionProtocolEvidence(ctx context.Context, q queryer, kind compositionFactKind, operation string) (compositionFact, error) {
	var version string
	if err := q.QueryRowContext(ctx, "SELECT version FROM composition_profile WHERE singleton=1").Scan(&version); err != nil {
		return compositionFact{}, err
	}
	if version != compositionProtocolVersion {
		return compositionFact{}, ErrVersion
	}
	if kind == compositionAuthorizationFact || kind == compositionExternalResultFact {
		return compositionSourceEvidence(ctx, q, kind, operation)
	}
	var raw []byte
	var root string
	var run RunID
	var sequence int64
	var table, column string
	switch kind {
	case compositionReservationFact:
		table, column = "composition_reservations", "operation_id"
	case compositionAdmissionFact:
		table, column = "composition_admission_decisions", "operation_id"
	case compositionReplyFact:
		table, column = "composition_replies", "operation_id"
	case compositionFreezeFact:
		table, column = "composition_freezes", "scope_id"
	case compositionOutcomeFact:
		table, column = "composition_publications", "operation_id"
		if strings.HasPrefix(operation, "root:") {
			table, column = "composition_root_publications", "scope_id"
		}
	default:
		return compositionFact{}, ErrVersion
	}
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM "+table+" WHERE "+column+"=?", operation).Scan(&raw); err != nil {
		return compositionFact{}, err
	}
	switch kind {
	case compositionReservationFact:
		var r compositionReservation
		if err := decode(raw, &r); err != nil {
			return compositionFact{}, err
		}
		if !sameJSON(raw, r) || r.OperationID != operation {
			return compositionFact{}, ErrCorrupt
		}
		root, run, sequence = r.RootID, r.Run, r.Sequence
	case compositionAdmissionFact:
		var a compositionAdmission
		if err := decode(raw, &a); err != nil {
			return compositionFact{}, err
		}
		if !sameJSON(raw, a) || a.Invocation.OperationID != operation {
			return compositionFact{}, ErrCorrupt
		}
		root, run, sequence = a.Invocation.RootID, a.Invocation.Caller, a.Invocation.SourceSequence
	case compositionReplyFact:
		var r compositionReply
		if err := decode(raw, &r); err != nil {
			return compositionFact{}, err
		}
		if !sameJSON(raw, r) || r.Intent.ID != operation {
			return compositionFact{}, ErrCorrupt
		}
		root, run, sequence = r.RootID, r.Intent.Run, r.Intent.Sequence
	case compositionFreezeFact:
		var f compositionFreeze
		if err := decode(raw, &f); err != nil {
			return compositionFact{}, err
		}
		if !sameJSON(raw, f) || f.InvocationID != operation {
			return compositionFact{}, ErrCorrupt
		}
		if err := q.QueryRowContext(ctx, "SELECT root_id FROM composition_freezes WHERE scope_id=?", operation).Scan(&root); err != nil {
			return compositionFact{}, err
		}
		run, sequence = f.Run, f.Sequence
	case compositionOutcomeFact:
		var o compositionOutcome
		if err := decode(raw, &o); err != nil {
			return compositionFact{}, err
		}
		if !sameJSON(raw, o) || o.OperationID != operation {
			return compositionFact{}, ErrCorrupt
		}
		if strings.HasPrefix(operation, "root:") {
			root, run = strings.TrimPrefix(operation, "root:"), o.Caller
		} else {
			a, err := loadCompositionAdmission(ctx, q, operation)
			if err != nil {
				return compositionFact{}, err
			}
			root, run, sequence = a.Invocation.RootID, o.Caller, a.Invocation.SourceSequence
		}
	}
	return compositionFact{Version: compositionFactVersion,
			ID: compositionIdentity("alan/fact/"+string(kind)+"/v1",
				operation),
			Kind:        kind,
			RootIDs:     []string{root},
			Run:         run,
			Sequence:    sequence,
			OperationID: operation,
			RecordDigest: compositionIdentity("alan/"+string(kind)+"-row/v1",
				json.RawMessage(raw))},
		nil
}

func recordCompositionReply(ctx context.Context,
	tx *sql.Tx,
	b compositionBinding,
	intent retainedIntent,
	auth Authorization,
	transport string,
	l Limits) (AdmissionResult,
	error) {
	if transport == "" {
		return AdmissionResult{}, fmt.Errorf("missing transport")
	}
	d := messageDescriptor(intent.Intent, transport)
	if err := checkSize(d, l.MaxObservationBytes); err != nil {
		return AdmissionResult{}, err
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		return AdmissionResult{}, err
	}
	oldD, old, err := loadAdmission(ctx, tx, "message", transport)
	if err == nil {
		if !sameJSON(encoded, oldD) {
			return old, ErrConflict
		}
		return old, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return old, err
	}
	var reply compositionReply
	var raw []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM composition_replies WHERE operation_id=?", intent.ID).Scan(&raw)
	if err == nil {
		if err = decode(raw, &reply); err != nil {
			return old, err
		}
		if !sameJSON(raw, reply) || !sameJSON(intent.raw, reply.Intent) {
			return old, ErrCorrupt
		}
	} else {
		if !errors.Is(err, sql.ErrNoRows) {
			return old, err
		}
		t, e := loadCompositionTemplate(ctx, tx, b.TemplateID)
		if e != nil {
			return old, e
		}
		p, e := loadProgram(ctx, tx, t.Members[t.EntryMember].ProgramID, l)
		if e != nil {
			return old, e
		}
		reply = compositionReply{InvocationID: b.InvocationID,
			RootID:        b.RootID,
			Intent:        intent.Intent,
			Authorization: auth,
			Accepted:      true,
			Reason:        "recorded_reply"}
		if !auth.Allowed {
			reply.Accepted = false
			reply.Reason = "source_denied:" + auth.Reason
		} else if p.Author.Ports[t.EntryPort].Send.Record.Validate(intent.Payload) != nil {
			reply.Accepted = false
			reply.Reason = "schema_invalid"
		}
		raw, err = json.Marshal(reply)
		if err != nil {
			return old, err
		}
		if err = checkSize(reply, l.MaxObservationBytes); err != nil {
			return old, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_replies VALUES(?,?,?)", intent.ID, b.InvocationID, raw); err != nil {
			return old, err
		}
		fact, e := compositionProtocolEvidence(ctx, tx, compositionReplyFact, intent.ID)
		if e != nil {
			return old, e
		}
		if _, err = appendCompositionFact(ctx, tx, fact); err != nil {
			return old, err
		}
	}
	result := AdmissionResult{TransportID: transport, Accepted: reply.Accepted, Reason: reply.Reason, Bound: &b.Caller}
	resultRaw, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	if _, err = tx.ExecContext(ctx,
		"INSERT INTO admissions VALUES(?,'message',?,?,?,NULL)",
		admissionKey("message",
			transport),
		transport,
		encoded,
		resultRaw); err != nil {
		return result, err
	}
	return result, nil
}

// freezeCompositionScope runs within the execution/lifecycle owner transaction.
// Its exact prefix sets cannot depend on subsequent callback delivery order.
func freezeCompositionScope(ctx context.Context, tx *sql.Tx, run RunID, trigger string, l Limits) error {
	scope, err := compositionScopeFor(ctx, tx, run)
	if err != nil {
		return err
	}
	if _, err = loadCompositionClosure(ctx, tx, scope.ID); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var exists int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM composition_freezes WHERE scope_id=?", scope.ID).Scan(&exists); err != nil {
		return err
	}
	if exists != 0 {
		return nil
	}
	current, err := loadRun(ctx, tx, run)
	if err != nil {
		return err
	}
	status := current.Projection.Status
	if status == "open" {
		return nil
	}
	if status == "succeeded" {
		for _, m := range scope.Members {
			state, e := loadRun(ctx, tx, m)
			if e != nil {
				return e
			}
			if state.Projection.Status != "succeeded" {
				return nil
			}
		}
	}
	freeze := compositionFreeze{InvocationID: scope.ID,
		Trigger:           trigger,
		Run:               run,
		Sequence:          current.Sequence,
		Status:            status,
		ReplyIntents:      []string{},
		InvocationIntents: []string{},
		ExternalIntents:   []string{}}
	if err = captureCompositionIntents(ctx, tx, &freeze, scope.Members); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx,
		"SELECT coalesce(max(decision_order),0)+1 FROM composition_facts").Scan(&freeze.DecisionOrder); err != nil {
		return err
	}
	raw, err := json.Marshal(freeze)
	if err != nil {
		return err
	}
	if err = checkSize(freeze, l.MaxDataBytes); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_freezes VALUES(?,?,?)", scope.ID, scope.RootID, raw); err != nil {
		return err
	}
	fact, err := compositionProtocolEvidence(ctx, tx, compositionFreezeFact, scope.ID)
	if err != nil {
		return err
	}
	_, err = appendCompositionFact(ctx, tx, fact)
	return err
}

func prepareInvocation(ctx context.Context, q queryer, operation string) (InvocationPreparation, error) {
	a, err := loadCompositionAdmission(ctx, q, operation)
	if err != nil {
		return InvocationPreparation{}, err
	}
	p := InvocationPreparation{Outcome: compositionOutcome{InvocationID: a.Invocation.ID,
		OperationID: operation,
		TemplateID:  a.Invocation.TemplateID,
		Caller:      a.Invocation.Caller}}
	var published []byte
	publicationErr := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_publications WHERE operation_id=?", operation).Scan(&published)
	if publicationErr == nil {
		if err = decode(published, &p.Outcome); err != nil {
			return p, err
		}
		if !sameJSON(published, p.Outcome) {
			return p, ErrCorrupt
		}
		p.Ready = true
		return p, nil
	}
	if !errors.Is(publicationErr, sql.ErrNoRows) {
		return p, publicationErr
	}
	if !a.Accepted {
		p.Ready = true
		p.Outcome.Status = "rejected"
		p.Outcome.Reason = a.Reason
		return p, nil
	}
	return prepareCompositionScope(ctx, q, p, a.Invocation.ID, true)
}

func prepareCompositionScope(ctx context.Context, q queryer, p InvocationPreparation, scope string, requireReply bool) (InvocationPreparation, error) {
	var err error
	var raw []byte
	err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_freezes WHERE scope_id=?", scope).Scan(&raw)
	hasFreeze := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	if hasFreeze {
		if err = decode(raw, &p.Freeze); err != nil {
			return p, err
		}
		if !sameJSON(raw, p.Freeze) {
			return p, ErrCorrupt
		}
	}
	closure, closureErr := loadCompositionClosure(ctx, q, scope)
	if closureErr != nil && !errors.Is(closureErr, sql.ErrNoRows) {
		return p, closureErr
	}
	closureReason := ""
	if closureErr == nil && (!hasFreeze || p.Freeze.Status == "succeeded" || closure.Freeze.DecisionOrder < p.Freeze.DecisionOrder) {
		p.Freeze = closure.Freeze
		closureReason = closure.Reason
		hasFreeze = true
	}
	if !hasFreeze {
		p.Pending = "members"
		return p, nil
	}
	responses := []map[string]Value{}
	for _, id := range p.Freeze.ReplyIntents {
		var r compositionReply
		if err = q.QueryRowContext(ctx,
			"SELECT descriptor FROM composition_replies WHERE operation_id=?",
			id).Scan(&raw); errors.Is(err,
			sql.ErrNoRows) {
			p.Pending = "reply_admission"
			return p, nil
		}
		if err != nil {
			return p, err
		}
		if err = decode(raw, &r); err != nil {
			return p, err
		}
		if r.Accepted {
			responses = append(responses, r.Intent.Payload)
		}
	}
	p.Outcome.Status = p.Freeze.Status
	p.Outcome.Reason = "member_" + p.Freeze.Status
	if closureReason != "" {
		p.Outcome.Reason = closureReason
	}
	if p.Freeze.Status == "succeeded" {
		for _, id := range p.Freeze.InvocationIntents {
			child, e := loadCompositionAdmission(ctx, q, id)
			if errors.Is(e, sql.ErrNoRows) {
				p.Pending = "invocation_admission"
				return p, nil
			}
			if e != nil {
				return p, e
			}
			if child.Accepted {
				pending, e := compositionDescendantsPending(ctx, q, child, map[string]bool{})
				if e != nil {
					return p, e
				}
				if pending != "" {
					p.Pending = pending
					return p, nil
				}
				var n int
				if e = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_publications WHERE operation_id=?", id).Scan(&n); e != nil {
					return p, e
				}
				if n == 0 {
					p.Pending = "descendant_outcome"
					return p, nil
				}
			}
		}
		for _, id := range p.Freeze.ExternalIntents {
			intent, e := loadIntent(ctx, q, id)
			if e != nil {
				return p, e
			}
			if intent.Kind == "adapter" {
				_, e = loadAdapterResult(ctx, q, id)
			} else {
				_, e = externalOutcome(ctx, q, id)
			}
			if errors.Is(e, sql.ErrNoRows) {
				p.Pending = "external_outcome"
				return p, nil
			}
			if e != nil {
				return p, e
			}
		}
		if requireReply && len(responses) == 0 {
			p.Outcome.Status = "failed"
			p.Outcome.Reason = "missing_reply"
		}
	}
	if len(p.Freeze.ReplyIntents) > 1 {
		// Conflicts remain retained evidence. They cannot replace the first
		// retirement closure or select one candidate as its public response.
		if closureReason == "" {
			p.Outcome.Status = "failed"
			p.Outcome.Reason = "conflicting_replies"
		}
	} else if len(responses) == 1 {
		p.Outcome.ResponsePresent = true
		p.Outcome.Response = responses[0]
	}
	p.Ready = true
	return p, nil
}
func (s *Store) PrepareInvocation(ctx context.Context, operation string) (InvocationPreparation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return InvocationPreparation{}, sql.ErrConnDone
	}
	if !s.invocationEnabled() {
		return InvocationPreparation{}, ErrVersion
	}
	return prepareInvocation(ctx, s.conn, operation)
}

// SettleInvocation retains a known outcome before making its authenticated
// observation available. It never publishes a success with unfinished joins.
func (s *Store) SettleInvocation(ctx context.Context, operation string) (AdmissionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	if !s.invocationEnabled() {
		return AdmissionResult{}, ErrVersion
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	a, err := loadCompositionAdmission(ctx, tx, operation)
	if err != nil {
		return AdmissionResult{}, err
	}
	var outcome compositionOutcome
	var raw []byte
	err = tx.QueryRowContext(ctx, "SELECT descriptor FROM composition_publications WHERE operation_id=?", operation).Scan(&raw)
	if err == nil {
		if err = decode(raw, &outcome); err != nil {
			return AdmissionResult{}, err
		}
		if !sameJSON(raw, outcome) {
			return AdmissionResult{}, ErrCorrupt
		}
	} else {
		if !errors.Is(err, sql.ErrNoRows) {
			return AdmissionResult{}, err
		}
		prepared, e := prepareInvocation(ctx, tx, operation)
		if e != nil {
			return AdmissionResult{}, e
		}
		if !prepared.Ready {
			return AdmissionResult{Reason: prepared.Pending}, nil
		}
		outcome = prepared.Outcome
		outcome.ID = compositionIdentity("alan/invocation-outcome/v1", a.Invocation.ID)
		if err = tx.QueryRowContext(ctx,
			"SELECT coalesce(max(decision_order),0)+1 FROM composition_facts").Scan(&outcome.DecisionOrder); err != nil {
			return AdmissionResult{}, err
		}
		raw, err = json.Marshal(outcome)
		if err != nil {
			return AdmissionResult{}, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_publications VALUES(?,?)", operation, raw); err != nil {
			return AdmissionResult{}, err
		}
		fact, e := compositionProtocolEvidence(ctx, tx, compositionOutcomeFact, operation)
		if e != nil {
			return AdmissionResult{}, e
		}
		if _, err = appendCompositionFact(ctx, tx, fact); err != nil {
			return AdmissionResult{}, err
		}
		if a.Accepted {
			if _, err = tx.ExecContext(ctx,
				"INSERT INTO composition_results VALUES(?,?,?)",
				a.Invocation.ID,
				outcome.DecisionOrder,
				raw); err != nil {
				return AdmissionResult{}, err
			}
		}
	}
	intent, err := loadIntent(ctx, tx, operation)
	if err != nil {
		return AdmissionResult{}, err
	}
	payload := map[string]Value{"status": Text(outcome.Status),
		"reason": Text(outcome.Reason),
		"response_present": {Kind: "bool",
			Flag: outcome.ResponsePresent}}
	if outcome.ResponsePresent {
		for field, value := range outcome.Response {
			payload["response."+field] = value.Clone()
		}
	}
	observation := Observation{ID: compositionIdentity("alan/invocation-result-observation/v1",
		outcome.ID),
		Run:              outcome.Caller,
		Kind:             "invocation",
		Provenance:       "runtime-invocation",
		Source:           outcome.TemplateID,
		SourceGeneration: 1,
		Obligation:       intent.Obligation,
		Operation:        operation,
		Payload:          payload}
	if err = insertObservation(ctx, tx, observation, s.limits); err != nil {
		return AdmissionResult{}, err
	}
	if _, err = tx.ExecContext(ctx,
		"INSERT OR IGNORE INTO composition_outcome_deliveries VALUES(?,?)",
		operation,
		observation.ID); err != nil {
		return AdmissionResult{}, err
	}
	result := AdmissionResult{TransportID: outcome.ID,
		Accepted:      true,
		Reason:        "recorded_outcome",
		Bound:         &outcome.Caller,
		ObservationID: observation.ID}
	if err = s.commitOwnedTx(ctx, tx); err != nil {
		return result, err
	}
	return result, boundary.Reached(ctx, boundary.Event{Boundary: "invocation.result-recorded", Phase: "pre-consumption", Subject: outcome.Caller.Subject, Generation: outcome.Caller.Generation, Sequence: a.Invocation.SourceSequence, Intent: operation, Attempt: outcome.ID, Root: a.Invocation.RootID})
}

func captureCompositionIntents(ctx context.Context, tx *sql.Tx, freeze *compositionFreeze, members []RunID) error {
	// All member intent rows at this transaction are the committed prefix. No
	// future member commit can become part of these retained lists.
	for _, m := range members {
		rows, e := tx.QueryContext(ctx,
			"SELECT descriptor FROM intents WHERE subject=? AND generation=? ORDER BY sequence,ordinal",
			m.Subject,
			m.Generation)
		if e != nil {
			return e
		}
		intents := []Intent{}
		for rows.Next() {
			var raw []byte
			var intent Intent
			if e = rows.Scan(&raw); e != nil {
				rows.Close()
				return e
			}
			if e = decode(raw, &intent); e != nil {
				rows.Close()
				return e
			}
			intents = append(intents, intent)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, intent := range intents {
			switch intent.Kind {
			case "invocation":
				freeze.InvocationIntents = append(freeze.InvocationIntents, intent.ID)
			case "adapter", "external":
				freeze.ExternalIntents = append(freeze.ExternalIntents, intent.ID)
			case "message":
				_, reply, e := compositionReplyBinding(ctx, tx, intent)
				if e != nil {
					return e
				}
				if reply {
					freeze.ReplyIntents = append(freeze.ReplyIntents, intent.ID)
				}
			}
		}
	}
	slices.Sort(freeze.ReplyIntents)
	slices.Sort(freeze.InvocationIntents)
	slices.Sort(freeze.ExternalIntents)
	return nil
}

// A known failed child may still own unresolved descendants or external work.
// Its early non-success publication does not erase those obligations for an
// ancestor that is attempting to publish success.
func compositionDescendantsPending(ctx context.Context, q queryer, a compositionAdmission, seen map[string]bool) (string, error) {
	if seen[a.Invocation.ID] {
		return "", ErrCorrupt
	}
	seen[a.Invocation.ID] = true
	var n int
	if err := q.QueryRowContext(ctx, "SELECT count(*) FROM composition_publications WHERE operation_id=?", a.Invocation.OperationID).Scan(&n); err != nil {
		return "", err
	}
	if n == 0 {
		return "descendant_outcome", nil
	}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return "", err
	}
	for _, id := range sortedKeys(intents) {
		i := intents[id]
		if !slices.Contains(a.Members, i.Run) {
			continue
		}
		switch i.Kind {
		case "invocation":
			child, e := loadCompositionAdmission(ctx, q, id)
			if errors.Is(e, sql.ErrNoRows) {
				return "invocation_admission", nil
			}
			if e != nil {
				return "", e
			}
			if child.Accepted {
				pending, e := compositionDescendantsPending(ctx, q, child, seen)
				if e != nil || pending != "" {
					return pending, e
				}
			}
		case "adapter", "external":
			known, e := compositionExternalKnown(ctx, q, id)
			if e != nil {
				return "", e
			}
			if !known {
				return "external_outcome", nil
			}
		}
	}
	return "", nil
}
