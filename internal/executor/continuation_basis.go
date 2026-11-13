package executor

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"slices"
)

// ContinuationBasis reads the original paid adapter evidence for a declared
// continuation target at one owner cut. An existing approval returns its
// historical basis. ApproveContinuation independently checks the operator's
// basis in its write transaction. Use InspectExecution for a full history audit.
func (s *Store) ContinuationBasis(ctx context.Context, target RunID) (ContinuationBasis, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ContinuationBasis{}, sql.ErrConnDone
	}
	if err := ctx.Err(); err != nil {
		return ContinuationBasis{}, err
	}
	if s.composition == nil {
		return ContinuationBasis{}, ErrVersion
	}
	slot, ok := s.continuationSlot(target)
	if !ok {
		return ContinuationBasis{}, ErrIneligible
	}
	// Only the transaction lifetime is detached. Rollback releases this read
	// without publishing a delta, waking Run or dropping the pinned connection.
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return ContinuationBasis{}, err
	}
	defer tx.Rollback()

	basis, err := s.readContinuationBasis(ctx, tx, slot)
	if errors.Is(err, sql.ErrNoRows) {
		err = corrupt("missing referenced continuation evidence")
	}
	if err != nil {
		return ContinuationBasis{}, err
	}
	if err := ctx.Err(); err != nil {
		return ContinuationBasis{}, err
	}
	return basis, nil
}

func (s *Store) readContinuationBasis(ctx context.Context, q queryer, slot ContinuationSlot) (ContinuationBasis, error) {
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return ContinuationBasis{}, err
	}
	if namespace != s.namespace {
		return ContinuationBasis{}, corrupt("continuation owner namespace changed")
	}
	approvalID := compositionIdentity("alan/fact/continuation/v1", slot.ApprovalID)
	decision, err := continuationPointFact(ctx, q, approvalID, compositionContinuationFact)
	if err != nil {
		return ContinuationBasis{}, err
	}
	d, r, err := loadAdmission(ctx, q, "input", slot.ApprovalID)
	switch {
	case err == nil:
		return s.approvedContinuationBasis(ctx, q, slot, d, r, decision)
	case !errors.Is(err, sql.ErrNoRows):
		return ContinuationBasis{}, err
	}
	observation, err := continuationPointObservation(ctx, q, identity("observation", "input", slot.ApprovalID))
	if err != nil {
		return ContinuationBasis{}, err
	}
	if decision.Order != 0 || observation != nil {
		return ContinuationBasis{}, corrupt("continuation lacks its original admission")
	}

	if _, err := loadRun(ctx, q, slot.Predecessor); err != nil {
		return ContinuationBasis{}, err
	}
	intents, err := continuationOriginals(ctx, q, slot.Predecessor)
	if err != nil {
		return ContinuationBasis{}, err
	}
	var basis ContinuationBasis
	for _, i := range intents {
		if i.Kind != "adapter" || i.Obligation != slot.Effect {
			continue
		}
		e, err := s.continuationOriginal(ctx, q, slot, i)
		if err != nil {
			return ContinuationBasis{}, err
		}
		if !e.attempt.Allowed {
			continue
		}
		if basis.PredecessorID != "" {
			return ContinuationBasis{}, ErrConflict
		}
		basis = e.basis(s.namespace, slot.Target, 0)
	}
	if basis.PredecessorID == "" {
		return ContinuationBasis{}, ErrIneligible
	}
	return basis, nil
}

// The UNIQUE(subject,generation,sequence,ordinal) index bounds discovery to
// the declared predecessor generation. The join uses history's original
// reduction key. Reduce emits one intent at ordinal zero per emit instruction.
func continuationOriginals(ctx context.Context, q queryer, run RunID) ([]retainedIntent, error) {
	rows, err := q.QueryContext(ctx, `SELECT i.id,i.sequence,i.ordinal,i.source_revision,i.descriptor,h.source_revision,h.intents
		FROM intents i LEFT JOIN history h USING(subject,generation,sequence)
		WHERE i.subject=? AND i.generation=? ORDER BY i.sequence,i.ordinal`, run.Subject, run.Generation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var intents []retainedIntent
	for rows.Next() {
		var i retainedIntent
		var id string
		var sequence, ordinal int64
		var revision sql.NullInt64
		var emitted []byte
		if err := rows.Scan(&id, &sequence, &ordinal, &i.revision, &i.raw, &revision, &emitted); err != nil {
			return nil, err
		}
		if decode(i.raw, &i.Intent) != nil || !sameJSON(i.raw, i.Intent) || sequence < 1 || i.revision < 1 ||
			i.Run != run || i.Sequence != sequence || int64(i.Ordinal) != ordinal || i.ID != id ||
			id != identity(run.Subject, run.Generation, sequence, ordinal) ||
			ordinal != 0 || !revision.Valid || revision.Int64 != i.revision || !sameJSON(emitted, []Intent{i.Intent}) {
			return nil, corrupt("continuation intent row identity")
		}
		intents = append(intents, i)
	}
	return intents, rows.Err()
}

type continuationOriginalEvidence struct {
	attempt        AdapterAttempt
	reservation    int64
	dispatchPrefix int64
	result         AdapterResult
	resultOrder    int64
}

func (e continuationOriginalEvidence) basis(namespace string, target RunID, before int64) ContinuationBasis {
	basis := ContinuationBasis{
		Namespace: namespace, Target: target, PredecessorID: e.attempt.Intent.ID,
		PreparedDigest: e.attempt.Prepared.Digest,
		Admitted:       e.dispatchPrefix > 0 && (before == 0 || e.dispatchPrefix < before),
	}
	if e.resultOrder > 0 && (before == 0 || e.resultOrder < before) {
		basis.ResultDigest = e.result.Digest
	}
	return basis
}

func (s *Store) continuationOriginal(ctx context.Context, q queryer, slot ContinuationSlot, i retainedIntent) (continuationOriginalEvidence, error) {
	var e continuationOriginalEvidence
	id := i.ID
	if i.Run != slot.Predecessor || i.Kind != "adapter" || i.Obligation != slot.Effect ||
		i.ID != identity(i.Run.Subject, i.Run.Generation, i.Sequence, i.Ordinal) {
		return e, corrupt("continuation references a different original operation")
	}
	a, err := loadAdapterAttempt(ctx, q, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e, err
	}
	hasAttempt := err == nil
	reservation, err := continuationProtocolFact(ctx, q, compositionReservationFact, id)
	if err != nil {
		return e, err
	}
	if hasAttempt != (reservation.Order != 0) {
		return e, corrupt("continuation reservation lacks its attempt or fact")
	}
	var chargeRoot string
	var chargeOrder int64
	var chargeRaw []byte
	err = q.QueryRowContext(ctx, "SELECT root_id,decision_order,descriptor FROM composition_charges WHERE operation_id=? AND kind='effect'", id).
		Scan(&chargeRoot, &chargeOrder, &chargeRaw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e, err
	}
	if (err == nil) != a.Allowed {
		return e, corrupt("continuation attempt and effect charge disagree")
	}

	var gate DispatchAdmission
	err = q.QueryRowContext(ctx, "SELECT intent_id,prepared_digest,after_fact FROM adapter_dispatch_admissions WHERE intent_id=?", id).
		Scan(&gate.OperationID, &gate.PreparedDigest, &gate.AfterFact)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e, err
	}
	if err == nil {
		if !a.Allowed || gate.OperationID != id || gate.PreparedDigest != a.Prepared.Digest || gate.AfterFact < reservation.Order {
			return e, corrupt("continuation dispatch admission differs from its reservation")
		}
		var prefix int64
		if err := q.QueryRowContext(ctx, "SELECT decision_order FROM composition_facts WHERE decision_order=?", gate.AfterFact).Scan(&prefix); err != nil {
			return e, err
		}
		e.dispatchPrefix = gate.AfterFact
	}

	result, err := loadAdapterResult(ctx, q, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e, err
	}
	hasResult := err == nil
	resultFact, err := continuationProtocolFact(ctx, q, compositionExternalResultFact, id)
	if err != nil {
		return e, err
	}
	if hasResult != (resultFact.Order != 0) {
		return e, corrupt("continuation result lacks its original record or fact")
	}
	if hasResult {
		if !hasAttempt || resultFact.Order <= reservation.Order || a.Allowed && (e.dispatchPrefix == 0 || e.dispatchPrefix >= resultFact.Order) {
			return e, corrupt("continuation result lacks its earlier reservation or admission")
		}
		// The dispatch-admission profile for continuations excludes controlled
		// receipt reconstruction. Refused ordinary adapter responses remain results.
		if result.Read != nil || result.Reconciled != nil || result.RefusalWitness != "" {
			return e, corrupt("unexpected continuation result protocol")
		}
		if err := validateAdapterResult(a, result, s.limits); err != nil {
			return e, err
		}
		if err := matchContinuationFact(resultFact, compositionExternalResultFact, i, result, reservation.RootIDs); err != nil {
			return e, err
		}
	}
	if err := validateContinuationSettlement(ctx, q, i, a, result, hasResult, resultFact.Order); err != nil {
		return e, err
	}
	if !hasAttempt {
		return e, nil
	}
	if err := s.validateContinuationReservation(ctx, q, i, a, reservation, e.dispatchPrefix, chargeRoot, chargeOrder, chargeRaw); err != nil {
		return e, err
	}
	e.attempt, e.reservation, e.result, e.resultOrder = a, reservation.Order, result, resultFact.Order
	return e, nil
}

// Validate the original authority and one paid reservation, including the
// separate charge and the fact that gives the reservation its historical order.
func (s *Store) validateContinuationReservation(ctx context.Context, q queryer, i retainedIntent, a AdapterAttempt, reservation compositionFact, dispatchPrefix int64, chargeRoot string, chargeOrder int64, chargeRaw []byte) error {
	id := i.ID
	if err := validateAdapterAttempt(ctx, q, a, s.namespace, s.limits); err != nil {
		return err
	}
	auth, err := validatedAuthorization(ctx, q, a.AuthorizationID)
	if err != nil {
		return err
	}
	scope, err := compositionRootFor(ctx, q, i.Run)
	if err != nil {
		return err
	}
	root := scope.ID
	if root != compositionRootID(s.namespace, scope.Run) || !slices.Contains(scope.Members, i.Run) || (a.Allowed && a.Reason != "authorized") {
		return corrupt("continuation reservation lacks scoped authority")
	}
	roots := []string{root}
	wantReservation := compositionReservation{
		OperationID: id, RootID: root, Kind: i.Kind, Run: i.Run, Sequence: i.Sequence,
		Allowed: a.Allowed, Reason: a.Reason, AttemptDigest: compositionIdentity("alan/attempt-row/v1", a),
	}
	if err := matchContinuationFact(reservation, compositionReservationFact, i, wantReservation, roots); err != nil {
		return err
	}
	var storedRoot string
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT root_id,descriptor FROM composition_reservations WHERE operation_id=?", id).Scan(&storedRoot, &raw); err != nil {
		return err
	}
	if storedRoot != root || !sameJSON(raw, wantReservation) {
		return corrupt("continuation reservation differs from its original attempt")
	}
	if a.Allowed && (chargeRoot != root || chargeOrder != reservation.Order || !sameJSON(chargeRaw, compositionCharge{
		OperationID: id, Kind: "effect", RootID: root, DecisionOrder: reservation.Order, Effects: 1,
	})) {
		return corrupt("continuation charge differs from its original reservation")
	}
	authorization, err := continuationProtocolFact(ctx, q, compositionAuthorizationFact, id)
	if err != nil {
		return err
	}
	if err := matchContinuationFact(authorization, compositionAuthorizationFact, i, auth, roots); err != nil {
		return err
	}
	committed, err := continuationExecutionOrder(ctx, q, i.Run, auth.Source.Sequence)
	if err != nil {
		return err
	}
	if auth.ClosureOperation != "" {
		closure, err := continuationProtocolFact(ctx, q, compositionLifecycleFact, auth.ClosureOperation)
		if err != nil {
			return err
		}
		if closure.Order <= committed || closure.Order >= authorization.Order || closure.Run != i.Run {
			return corrupt("continuation authorization cites a different closure")
		}
	}
	run, err := loadRun(ctx, q, i.Run)
	if err != nil {
		return err
	}
	if auth.Source.Sequence < run.Sequence {
		next, err := continuationExecutionOrder(ctx, q, i.Run, auth.Source.Sequence+1)
		if err != nil {
			return err
		}
		if authorization.Order >= next {
			return corrupt("continuation authorization lies outside its source sequence")
		}
	}
	if a.Allowed {
		prefix := max(reservation.Order, dispatchPrefix)
		if err := continuationBeforeTerminal(ctx, q, i.Run, prefix); err != nil {
			return err
		}
		if scope.Run != i.Run {
			if err := continuationBeforeTerminal(ctx, q, scope.Run, prefix); err != nil {
				return err
			}
			// A failed peer can freeze a shared root before the root run
			// itself reaches a terminal projection.
			if err := continuationBeforeRootFreeze(ctx, q, root, prefix); err != nil {
				return err
			}
		}
	}
	if committed >= authorization.Order || authorization.Order >= reservation.Order {
		return corrupt("continuation reservation precedes its source authority")
	}
	var revision int64
	var intents []byte
	if err := q.QueryRowContext(ctx, "SELECT source_revision,intents FROM history WHERE subject=? AND generation=? AND sequence=?",
		i.Run.Subject, i.Run.Generation, i.Sequence).Scan(&revision, &intents); err != nil {
		return err
	}
	if i.Ordinal != 0 || i.revision != revision || !sameJSON(intents, []Intent{i.Intent}) {
		return corrupt("continuation intent differs from its original reduction")
	}
	return nil
}

func continuationPointFact(ctx context.Context, q queryer, id string, kind compositionFactKind) (compositionFact, error) {
	var f compositionFact
	var order int64
	var storedKind compositionFactKind
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT decision_order,kind,descriptor FROM composition_facts WHERE id=?", id).Scan(&order, &storedKind, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if decode(raw, &f) != nil || !sameJSON(raw, f) || f.Version != compositionFactVersion ||
		f.ID != id || f.Kind != kind || storedKind != kind || f.Order != order || order < 1 {
		return compositionFact{}, corrupt("continuation fact row mismatch")
	}
	return f, nil
}

func continuationProtocolFact(ctx context.Context, q queryer, kind compositionFactKind, operation string) (compositionFact, error) {
	return continuationPointFact(ctx, q, compositionIdentity("alan/fact/"+string(kind)+"/v1", operation), kind)
}

func matchContinuationFact(f compositionFact, kind compositionFactKind, i retainedIntent, record any, roots []string) error {
	want := compositionFact{
		Version: compositionFactVersion, Kind: kind, ID: compositionIdentity("alan/fact/"+string(kind)+"/v1", i.ID), Order: f.Order,
		RootIDs: roots, Run: i.Run, Sequence: i.Sequence, OperationID: i.ID,
		RecordDigest: compositionIdentity("alan/"+string(kind)+"-row/v1", record),
	}
	if f.Order == 0 || !reflect.DeepEqual(f, want) {
		return corrupt("continuation %s fact differs from its original row", kind)
	}
	return nil
}

func continuationExecutionOrder(ctx context.Context, q queryer, run RunID, sequence int64) (int64, error) {
	want, err := compositionExecutionEvidence(ctx, q, run, sequence)
	if err != nil {
		return 0, err
	}
	got, err := continuationPointFact(ctx, q, want.ID, compositionExecutionFact)
	if err != nil {
		return 0, err
	}
	want.Order = got.Order
	if got.Order == 0 || !reflect.DeepEqual(got, want) {
		return 0, corrupt("continuation lacks its original execution fact")
	}
	return got.Order, nil
}

// Terminal evidence can be later than an approval or dispatch. Compare the
// original terminal order to that boundary, independent of the final status.
func continuationBeforeTerminal(ctx context.Context, q queryer, run RunID, before int64) error {
	var seq int64
	var operation sql.NullString
	err := q.QueryRowContext(ctx, "SELECT sequence,operation_id FROM terminal_obligations WHERE subject=? AND generation=?",
		run.Subject, run.Generation).Scan(&seq, &operation)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var order int64
	if operation.Valid {
		f, err := continuationProtocolFact(ctx, q, compositionLifecycleFact, operation.String)
		if err != nil {
			return err
		}
		if f.Run != run || f.Sequence != seq {
			return corrupt("continuation terminal differs from its lifecycle fact")
		}
		order = f.Order
	} else {
		order, err = continuationExecutionOrder(ctx, q, run, seq)
		if err != nil {
			return err
		}
	}
	if order <= before {
		return corrupt("continuation evidence follows its source or target terminal")
	}
	return nil
}

func continuationBeforeRootFreeze(ctx context.Context, q queryer, root string, before int64) error {
	scope := "root:" + root
	f, err := continuationProtocolFact(ctx, q, compositionFreezeFact, scope)
	if err != nil {
		return err
	}
	var raw []byte
	err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_freezes WHERE scope_id=?", scope).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) && f.Order == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	var freeze compositionFreeze
	if decode(raw, &freeze) != nil || !sameJSON(raw, freeze) || freeze.InvocationID != scope ||
		freeze.DecisionOrder != f.Order || f.Order == 0 {
		return corrupt("continuation predecessor has invalid root freeze")
	}
	want, err := compositionProtocolEvidence(ctx, q, compositionFreezeFact, scope)
	if err != nil {
		return err
	}
	want.Order = f.Order
	if !reflect.DeepEqual(want, f) || (freeze.Status != "succeeded" && f.Order <= before) {
		return corrupt("continuation predecessor reservation or dispatch differs from root freeze")
	}
	return nil
}

type continuationObservation struct {
	run      RunID
	raw      []byte
	selected sql.NullInt64
}

func continuationPointObservation(ctx context.Context, q queryer, id string) (*continuationObservation, error) {
	var o continuationObservation
	err := q.QueryRowContext(ctx, "SELECT subject,generation,descriptor,consumed_sequence FROM observations WHERE id=?", id).
		Scan(&o.run.Subject, &o.run.Generation, &o.raw, &o.selected)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func validateContinuationAdmission(ctx context.Context, q queryer, d AdmissionDescriptor, r AdmissionResult, o *continuationObservation, run RunID, reason string, before int64) error {
	var id sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT observation_id FROM admissions WHERE id=?", admissionKey(d.Kind, d.TransportID)).Scan(&id); err != nil {
		return err
	}
	want := observationFor(d, run)
	if !r.Accepted || r.Reason != reason || r.Bound == nil || *r.Bound != run || r.ObservationID != want.ID ||
		!id.Valid || id.String != want.ID || o == nil || o.run != run || !sameJSON(o.raw, want) {
		return corrupt("continuation admission and observation disagree")
	}
	if !o.selected.Valid {
		return nil
	}
	var selected sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT selected_observation FROM history WHERE subject=? AND generation=? AND sequence=?",
		run.Subject, run.Generation, o.selected.Int64).Scan(&selected); err != nil {
		return err
	}
	order, err := continuationExecutionOrder(ctx, q, run, o.selected.Int64)
	if err != nil {
		return err
	}
	if !selected.Valid || selected.String != want.ID || order <= before {
		return corrupt("continuation observation selected outside its retained admission")
	}
	return nil
}

func validateContinuationSettlement(ctx context.Context, q queryer, i retainedIntent, a AdapterAttempt, result AdapterResult, hasResult bool, resultOrder int64) error {
	transport := identity("adapter-delivery", i.ID)
	d, r, err := loadAdmission(ctx, q, "adapter", transport)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	hasAdmission := err == nil
	var intentID string
	var observationID sql.NullString
	var raw []byte
	err = q.QueryRowContext(ctx, "SELECT intent_id,descriptor,observation_id FROM settlements WHERE id=?", identity("adapter-outcome", i.ID)).
		Scan(&intentID, &raw, &observationID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	hasSettlement := err == nil
	o, err := continuationPointObservation(ctx, q, identity("observation", "adapter", i.ID))
	if err != nil {
		return err
	}
	if !hasAdmission && !hasSettlement && o == nil {
		return nil
	}
	if !hasResult || !hasAdmission || !hasSettlement || intentID != i.ID || !sameJSON(raw, result) ||
		!observationID.Valid || observationID.String != r.ObservationID || !reflect.DeepEqual(d, adapterDescriptor(a, result)) {
		return corrupt("continuation settlement differs from its original result")
	}
	return validateContinuationAdmission(ctx, q, d, r, o, i.Run, "settled", resultOrder)
}

func (s *Store) approvedContinuationBasis(ctx context.Context, q queryer, slot ContinuationSlot, d AdmissionDescriptor, r AdmissionResult, decision compositionFact) (ContinuationBasis, error) {
	w := continuationWitness{slot: slot}
	index := continuationIndex{bySubject: map[string]*continuationWitness{slot.Target.Subject: &w}, byID: map[string]*continuationWitness{slot.ApprovalID: &w}}
	if err := index.admission(d, r); err != nil {
		return ContinuationBasis{}, err
	}
	if w.view == nil || decision.Order == 0 || d.TerminalSequence != 0 || !validAdmissionHeader(d) ||
		!validAdmissionPayload(d.Payload) || checkSize(d, s.limits.MaxObservationBytes) != nil {
		return ContinuationBasis{}, corrupt("invalid retained continuation approval")
	}
	target, err := loadRun(ctx, q, slot.Target)
	if err != nil {
		return ContinuationBasis{}, err
	}
	program, err := loadProgram(ctx, q, target.ProgramID, s.limits)
	if err != nil {
		return ContinuationBasis{}, err
	}
	permitted, err := admissionPermitted(ctx, q, d)
	if err != nil {
		return ContinuationBasis{}, err
	}
	if !permitted || !authorAdmissionValid(program, d) {
		return ContinuationBasis{}, corrupt("continuation approval lacks its declared input route")
	}
	scope, err := compositionRootFor(ctx, q, slot.Target)
	if err != nil {
		return ContinuationBasis{}, err
	}
	root := scope.ID
	if scope.Run != slot.Target || root != compositionRootID(s.namespace, slot.Target) || !slices.Equal(scope.Members, []RunID{slot.Target}) {
		return ContinuationBasis{}, corrupt("continuation approval lacks original target membership")
	}
	if err := continuationBeforeTerminal(ctx, q, slot.Target, decision.Order); err != nil {
		return ContinuationBasis{}, err
	}
	o, err := continuationPointObservation(ctx, q, r.ObservationID)
	if err != nil {
		return ContinuationBasis{}, err
	}
	if err := validateContinuationAdmission(ctx, q, d, r, o, slot.Target, "accepted", decision.Order); err != nil {
		return ContinuationBasis{}, err
	}
	i, err := loadIntent(ctx, q, w.view.Approval.PredecessorID)
	if err != nil {
		return ContinuationBasis{}, err
	}
	e, err := s.continuationOriginal(ctx, q, slot, i)
	if err != nil {
		return ContinuationBasis{}, err
	}
	if e.reservation == 0 || e.reservation >= decision.Order {
		return ContinuationBasis{}, corrupt("continuation decision precedes its original reservation")
	}
	basis := e.basis(s.namespace, slot.Target, decision.Order)
	if err := w.decision(decision, s.namespace, root, target.Revision, &e.attempt, basis.Admitted, basis.ResultDigest); err != nil {
		return ContinuationBasis{}, err
	}
	return w.view.Approval.ContinuationBasis, nil
}
